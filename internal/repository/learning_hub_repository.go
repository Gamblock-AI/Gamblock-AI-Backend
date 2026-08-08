package repository

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/academicprogram"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/dailymission"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/experiencegrant"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningcluster"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningitem"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningprogress"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningrevision"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

var (
	ErrLearningItemNotFound      = errors.New("learning item not found")
	ErrLearningProgressNotFound  = errors.New("learning progress not found")
	ErrLearningStateInvalid      = errors.New("learning state is invalid")
	ErrLearningCheckpointInvalid = errors.New("learning checkpoint is invalid")
)

const dailyExperienceCap = 50

// learningRevisionSnapshotKey deliberately namespaces the editorial envelope
// away from the public learning document. A revision must preserve the public
// title, summary, slug, and kind alongside its metadata; otherwise a later
// draft edit could leak those fields before the next publish action.
const learningRevisionSnapshotKey = "_learning_item_snapshot"

func learningRevisionDocument(document map[string]any, slug, kind, titleID, titleEN, summaryID, summaryEN string) map[string]any {
	return map[string]any{
		learningRevisionSnapshotKey: map[string]any{
			"document":   cloneLearningDocument(document),
			"slug":       slug,
			"kind":       kind,
			"title_id":   titleID,
			"title_en":   titleEN,
			"summary_id": summaryID,
			"summary_en": summaryEN,
		},
	}
}

func unpackLearningRevisionDocument(document map[string]any) (map[string]any, map[string]any) {
	envelope, ok := document[learningRevisionSnapshotKey].(map[string]any)
	if !ok {
		return document, nil
	}
	content, ok := envelope["document"].(map[string]any)
	if !ok {
		return document, nil
	}
	return content, envelope
}

func snapshotString(snapshot map[string]any, key, fallback string) string {
	if value, ok := snapshot[key].(string); ok {
		return value
	}
	return fallback
}

func learningItemFromEntDocument(row *ent.LearningItem, document map[string]any, locale string) model.LearningItem {
	document, snapshot := unpackLearningRevisionDocument(document)
	slug, kind := row.Slug, row.Kind.String()
	titleID, titleEN := row.TitleID, row.TitleEn
	summaryID, summaryEN := row.SummaryID, row.SummaryEn
	if snapshot != nil {
		slug = snapshotString(snapshot, "slug", slug)
		kind = snapshotString(snapshot, "kind", kind)
		titleID = snapshotString(snapshot, "title_id", titleID)
		titleEN = snapshotString(snapshot, "title_en", titleEN)
		summaryID = snapshotString(snapshot, "summary_id", summaryID)
		summaryEN = snapshotString(snapshot, "summary_en", summaryEN)
	}
	title, summary := titleID, summaryID
	if locale == "en" {
		title, summary = titleEN, summaryEN
	}
	item := model.LearningItem{ID: row.ID, Slug: slug, Kind: kind, Title: title, Summary: summary}
	var logoMediaID, thumbnailMediaID, logoURL, thumbnailURL string
	for key, value := range document {
		switch key {
		case "provider":
			item.Provider, _ = value.(string)
		case "url":
			item.URL = safeLearningURL(value)
		case "provider_logo_media_id":
			logoMediaID, _ = value.(string)
		case "provider_logo_url":
			logoURL, _ = value.(string)
		case "thumbnail_media_id":
			thumbnailMediaID, _ = value.(string)
		case "thumbnail_url":
			thumbnailURL, _ = value.(string)
		case "cost":
			item.Cost, _ = value.(string)
		case "certificate":
			item.Certificate, _ = value.(string)
		case "difficulty":
			item.Difficulty, _ = value.(string)
		case "prerequisites":
			item.Prerequisites, _ = value.(string)
		case "career_snapshot":
			item.CareerSnapshot, _ = value.(string)
		case "reviewed_at":
			item.ReviewedAt, _ = value.(string)
		case "duration_minutes":
			if number, ok := value.(float64); ok {
				item.DurationMinutes = int(number)
			} else if number, ok := value.(int); ok {
				item.DurationMinutes = number
			}
		case "language":
			item.Language = stringSlice(value)
		case "outcomes":
			item.Outcomes = stringSlice(value)
		case "outcomes_id":
			if locale != "en" {
				item.Outcomes = stringSlice(value)
			}
		case "outcomes_en":
			if locale == "en" {
				item.Outcomes = stringSlice(value)
			}
		case "clusters":
			item.Clusters = stringSlice(value)
		case "programs":
			item.Programs = stringSlice(value)
		case "steps":
			item.Steps = stringSlice(value)
		case "projects":
			item.Projects = stringSlice(value)
		}
	}
	// Self-hosted media (uploaded to the education media pipeline) wins over an
	// external fallback URL, and both are resolved to public URLs the website
	// renders directly.
	item.ProviderLogoURL = learningMediaURL(logoMediaID, logoURL)
	item.ThumbnailURL = learningMediaURL(thumbnailMediaID, thumbnailURL)
	return item
}

// learningMediaURL returns the public route for a self-hosted education media
// id, falling back to a pre-resolved external URL when no media was uploaded.
func learningMediaURL(mediaID, externalURL string) string {
	if mediaID != "" {
		return "/v1/education/media/" + mediaID
	}
	if externalURL == "" {
		return ""
	}
	parsed, err := url.Parse(externalURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Hostname() == "" {
		return ""
	}
	return externalURL
}

func learningItemFromEnt(row *ent.LearningItem, locale string) model.LearningItem {
	return learningItemFromEntDocument(row, row.DocumentJSON, locale)
}

func safeLearningURL(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
		return ""
	}
	return raw
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if ok {
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	}
	stringItems, ok := value.([]string)
	if ok {
		return append([]string(nil), stringItems...)
	}
	return nil
}

func cloneLearningDocument(document map[string]any) map[string]any {
	if document == nil {
		return nil
	}
	clone := make(map[string]any, len(document))
	for key, value := range document {
		clone[key] = value
	}
	return clone
}

func learningProgressFromEnt(row *ent.LearningProgress) model.LearningProgress {
	return model.LearningProgress{UserID: row.UserID, ItemID: row.ItemID, State: row.State.String(), CompletedAt: row.CompletedAt, ReflectionEncrypted: optionalString(row.ReflectionEncrypted), OutcomeEncrypted: optionalString(row.OutcomeEncrypted)}
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (r *Repository) GetLearningCatalog(ctx context.Context, userID, locale string) (model.LearningCatalog, error) {
	if r.db == nil {
		return r.getLearningCatalogMemory(userID, locale), nil
	}
	clusters, err := r.db.LearningCluster.Query().Where(learningcluster.ActiveEQ(true)).All(ctx)
	if err != nil {
		return model.LearningCatalog{}, err
	}
	programs, err := r.db.AcademicProgram.Query().Where(academicprogram.ActiveEQ(true)).All(ctx)
	if err != nil {
		return model.LearningCatalog{}, err
	}
	items, err := r.db.LearningItem.Query().Where(learningitem.StatusNEQ(learningitem.StatusArchived), learningitem.PublishedRevisionGT(0)).All(ctx)
	if err != nil {
		return model.LearningCatalog{}, err
	}
	progressRows, err := r.db.LearningProgress.Query().Where(learningprogress.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return model.LearningCatalog{}, err
	}
	progress := make([]model.LearningProgress, 0, len(progressRows))
	byItem := make(map[string]model.LearningProgress, len(progressRows))
	for _, row := range progressRows {
		value := learningProgressFromEnt(row)
		progress = append(progress, value)
		byItem[value.ItemID] = value
	}
	result := model.LearningCatalog{Clusters: make([]model.LearningCluster, 0, len(clusters)), Programs: make([]model.AcademicProgram, 0, len(programs)), Items: make([]model.LearningItem, 0, len(items)), Progress: progress}
	publishedDocuments := make(map[string]map[string]any, len(items))
	publishedRevisions := make(map[string]int, len(items))
	if len(items) > 0 {
		revisionRows, revisionErr := r.db.LearningRevision.Query().Where(learningrevision.KindEQ(learningrevision.KindPublished)).All(ctx)
		if revisionErr != nil {
			return model.LearningCatalog{}, revisionErr
		}
		for _, revision := range revisionRows {
			if currentRevision, ok := publishedRevisions[revision.ItemID]; !ok || revision.Revision >= currentRevision {
				document := cloneLearningDocument(revision.DocumentJSON)
				publishedDocuments[revision.ItemID] = document
				publishedRevisions[revision.ItemID] = revision.Revision
			}
		}
	}
	for _, cluster := range clusters {
		title, description := cluster.TitleID, cluster.DescriptionID
		if locale == "en" {
			title, description = cluster.TitleEn, cluster.DescriptionEn
		}
		result.Clusters = append(result.Clusters, model.LearningCluster{ID: cluster.ID, Slug: cluster.Slug, Title: title, Description: description, SortOrder: cluster.SortOrder})
	}
	for _, program := range programs {
		result.Programs = append(result.Programs, model.AcademicProgram{ID: program.ID, InstitutionID: program.InstitutionID, Slug: program.Slug, Name: program.Name, Degree: program.Degree, PrimaryClusterSlug: program.PrimaryClusterSlug, SortOrder: program.SortOrder})
	}
	for _, row := range items {
		document := publishedDocuments[row.ID]
		if document == nil {
			// A published revision is the only public source of truth. Do not
			// fall back to the mutable draft document when it is missing.
			continue
		}
		item := learningItemFromEntDocument(row, document, locale)
		if saved, ok := byItem[item.ID]; ok {
			item.Progress = &saved
		}
		result.Items = append(result.Items, item)
	}
	sort.Slice(result.Clusters, func(i, j int) bool { return result.Clusters[i].SortOrder < result.Clusters[j].SortOrder })
	sort.Slice(result.Programs, func(i, j int) bool { return result.Programs[i].SortOrder < result.Programs[j].SortOrder })
	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].Slug < result.Items[j].Slug })
	result.Experience, err = r.learningExperience(ctx, userID)
	return result, err
}

func (r *Repository) GetLearningItemBySlug(ctx context.Context, userID, slug, locale string) (model.LearningItem, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		for _, item := range r.store.LearningItems {
			if item.Slug == slug {
				for _, progress := range r.store.LearningProgress {
					if progress.UserID == userID && progress.ItemID == item.ID {
						item.Progress = &progress
						break
					}
				}
				return item, nil
			}
		}
		return model.LearningItem{}, ErrLearningItemNotFound
	}
	row, err := r.db.LearningItem.Query().Where(learningitem.SlugEQ(slug), learningitem.StatusNEQ(learningitem.StatusArchived), learningitem.PublishedRevisionGT(0)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.LearningItem{}, ErrLearningItemNotFound
	}
	if err != nil {
		return model.LearningItem{}, err
	}
	document, documentErr := r.publishedLearningDocument(ctx, row.ID)
	if documentErr != nil {
		return model.LearningItem{}, documentErr
	}
	if document == nil {
		return model.LearningItem{}, ErrLearningItemNotFound
	}
	item := learningItemFromEntDocument(row, document, locale)
	progress, progressErr := r.db.LearningProgress.Query().Where(learningprogress.UserIDEQ(userID), learningprogress.ItemIDEQ(row.ID)).Only(ctx)
	if progressErr == nil {
		value := learningProgressFromEnt(progress)
		item.Progress = &value
	} else if !ent.IsNotFound(progressErr) {
		return model.LearningItem{}, progressErr
	}
	return item, nil
}

func (r *Repository) GetLearningProgress(ctx context.Context, userID string) ([]model.LearningProgress, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		out := make([]model.LearningProgress, 0)
		for _, progress := range r.store.LearningProgress {
			if progress.UserID == userID && progress.ItemID != "" && progress.State != "" {
				out = append(out, progress)
			}
		}
		return out, nil
	}
	rows, err := r.db.LearningProgress.Query().Where(learningprogress.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.LearningProgress, 0, len(rows))
	for _, row := range rows {
		out = append(out, learningProgressFromEnt(row))
	}
	return out, nil
}

func (r *Repository) UpsertLearningState(ctx context.Context, userID, itemID, state string) (model.LearningProgress, error) {
	if state != "saved" && state != "started" {
		return model.LearningProgress{}, ErrLearningStateInvalid
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.LearningProgress {
			row := &r.store.LearningProgress[index]
			if row.UserID == userID && row.ItemID == itemID && row.State != "" {
				if row.State != "completed" {
					row.State = state
				}
				return *row, nil
			}
		}
		row := model.LearningProgress{UserID: userID, ItemID: itemID, State: state}
		r.store.LearningProgress = append(r.store.LearningProgress, row)
		return row, nil
	}
	if _, err := r.db.LearningItem.Query().Where(learningitem.IDEQ(itemID), learningitem.StatusNEQ(learningitem.StatusArchived), learningitem.PublishedRevisionGT(0)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return model.LearningProgress{}, ErrLearningItemNotFound
		}
		return model.LearningProgress{}, err
	}
	row, err := r.db.LearningProgress.Query().Where(learningprogress.UserIDEQ(userID), learningprogress.ItemIDEQ(itemID)).Only(ctx)
	if ent.IsNotFound(err) {
		created, createErr := r.db.LearningProgress.Create().SetID("lp_" + uuid.NewString()).SetUserID(userID).SetItemID(itemID).SetState(learningprogress.State(state)).Save(ctx)
		if createErr != nil {
			return model.LearningProgress{}, createErr
		}
		return learningProgressFromEnt(created), nil
	}
	if err != nil {
		return model.LearningProgress{}, err
	}
	if row.State == learningprogress.StateCompleted {
		return learningProgressFromEnt(row), nil
	}
	updated, err := r.db.LearningProgress.UpdateOne(row).SetState(learningprogress.State(state)).Save(ctx)
	if err != nil {
		return model.LearningProgress{}, err
	}
	return learningProgressFromEnt(updated), nil
}

func (r *Repository) CompleteLearningProgress(ctx context.Context, userID, itemID, reflection, outcome string) (model.LearningProgress, bool, bool, error) {
	if r.db == nil {
		if reflection == "" && outcome == "" {
			return model.LearningProgress{}, false, false, ErrLearningCheckpointInvalid
		}
		r.store.RLock()
		itemExists := false
		for _, item := range r.store.LearningItems {
			if item.ID == itemID {
				itemExists = true
				break
			}
		}
		r.store.RUnlock()
		if !itemExists {
			return model.LearningProgress{}, false, false, ErrLearningItemNotFound
		}
		r.store.RLock()
		for _, progress := range r.store.LearningProgress {
			if progress.UserID == userID && progress.ItemID == itemID && progress.State == "completed" {
				r.store.RUnlock()
				return progress, false, false, nil
			}
		}
		r.store.RUnlock()
		progress := r.completeLearningMemory(userID, itemID, reflection, outcome)
		granted, capReached := r.grantExperienceMemory(userID, "learning_checkpoint", itemID, 10)
		return progress, granted, capReached, nil
	}
	if reflection == "" && outcome == "" {
		return model.LearningProgress{}, false, false, ErrLearningCheckpointInvalid
	}
	if _, err := r.db.LearningItem.Query().Where(learningitem.IDEQ(itemID), learningitem.StatusNEQ(learningitem.StatusArchived), learningitem.PublishedRevisionGT(0)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return model.LearningProgress{}, false, false, ErrLearningItemNotFound
		}
		return model.LearningProgress{}, false, false, err
	}
	row, err := r.db.LearningProgress.Query().Where(learningprogress.UserIDEQ(userID), learningprogress.ItemIDEQ(itemID)).Only(ctx)
	if ent.IsNotFound(err) {
		row, err = r.db.LearningProgress.Create().SetID("lp_" + uuid.NewString()).SetUserID(userID).SetItemID(itemID).SetState(learningprogress.StateCompleted).SetReflectionEncrypted(reflection).SetOutcomeEncrypted(outcome).SetCompletedAt(time.Now().UTC()).Save(ctx)
	} else if err == nil {
		if row.State == learningprogress.StateCompleted {
			return learningProgressFromEnt(row), false, false, nil
		}
		row, err = r.db.LearningProgress.UpdateOne(row).SetState(learningprogress.StateCompleted).SetReflectionEncrypted(reflection).SetOutcomeEncrypted(outcome).SetCompletedAt(time.Now().UTC()).Save(ctx)
	}
	if err != nil {
		return model.LearningProgress{}, false, false, err
	}
	granted, capReached, err := r.grantExperience(ctx, userID, "learning_checkpoint", itemID, 10)
	return learningProgressFromEnt(row), granted, capReached, err
}

func (r *Repository) grantExperience(ctx context.Context, userID, sourceKind, sourceID string, amount int) (bool, bool, error) {
	today := time.Now().In(time.FixedZone("Asia/Jakarta", 7*60*60)).Format("2006-01-02")
	if r.db == nil {
		return false, false, nil
	}
	if _, err := r.db.ExperienceGrant.Query().Where(experiencegrant.UserIDEQ(userID), experiencegrant.SourceKindEQ(sourceKind), experiencegrant.SourceIDEQ(sourceID)).Only(ctx); err == nil {
		return false, false, nil
	} else if !ent.IsNotFound(err) {
		return false, false, err
	}
	available, err := availableDailyExperience(ctx, r.db, userID, today)
	if err != nil {
		return false, false, err
	}
	if available < amount {
		return false, true, nil
	}
	_, err = r.db.ExperienceGrant.Create().SetID("exp_" + uuid.NewString()).SetUserID(userID).SetSourceKind(sourceKind).SetSourceID(sourceID).SetGrantDate(today).SetAmount(amount).SetIdempotencyKey(userID + ":" + sourceKind + ":" + sourceID).Save(ctx)
	if ent.IsConstraintError(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if _, err = r.db.User.Update().Where(entuser.IDEQ(userID)).AddExperiencePoints(amount).Save(ctx); err != nil {
		return false, false, err
	}
	return true, false, nil
}

func (r *Repository) grantExperienceMemory(userID, sourceKind, sourceID string, amount int) (bool, bool) {
	today := time.Now().In(time.FixedZone("Asia/Jakarta", 7*60*60)).Format("2006-01-02")
	r.store.Lock()
	defer r.store.Unlock()
	for _, grant := range r.store.ExperienceGrants {
		if grant.UserID == userID && grant.SourceKind == sourceKind && grant.SourceID == sourceID {
			return false, false
		}
	}
	used := 0
	for _, mission := range r.store.Missions {
		if mission.UserID == userID && mission.Date == today {
			for _, task := range mission.Tasks {
				if task.Status == "completed" {
					used += task.EXPReward
				}
			}
		}
	}
	for _, grant := range r.store.ExperienceGrants {
		if grant.UserID == userID && grant.GrantDate == today {
			used += grant.Amount
		}
	}
	if dailyExperienceCap-used < amount {
		return false, true
	}
	r.store.ExperienceGrants = append(r.store.ExperienceGrants, model.ExperienceGrant{ID: "exp_" + uuid.NewString(), UserID: userID, SourceKind: sourceKind, SourceID: sourceID, GrantDate: today, Amount: amount, CreatedAt: time.Now().UTC()})
	for index := range r.store.Users {
		if r.store.Users[index].ID == userID {
			r.store.Users[index].ExperiencePoints += amount
			break
		}
	}
	return true, false
}

func availableDailyExperience(ctx context.Context, client *ent.Client, userID, date string) (int, error) {
	missionRows, err := client.DailyMission.Query().Where(dailymission.UserID(userID), dailymission.MissionDateEQ(date), dailymission.StatusEQ(dailymission.StatusCompleted)).All(ctx)
	if err != nil {
		return 0, err
	}
	grantRows, err := client.ExperienceGrant.Query().Where(experiencegrant.UserIDEQ(userID), experiencegrant.GrantDateEQ(date)).All(ctx)
	if err != nil {
		return 0, err
	}
	used := 0
	for _, row := range missionRows {
		used += row.ExpReward
	}
	for _, row := range grantRows {
		used += row.Amount
	}
	return max(0, dailyExperienceCap-used), nil
}

func (r *Repository) learningExperience(ctx context.Context, userID string) (model.ExperienceProgress, error) {
	points, err := r.userExperience(ctx, userID)
	if err != nil {
		return model.ExperienceProgress{}, err
	}
	return model.ExperienceProgress{TotalEXP: points, Level: points/100 + 1, LevelProgress: points % 100, LevelTarget: 100}, nil
}

func (r *Repository) GetLearningExperience(ctx context.Context, userID string) (model.ExperienceProgress, error) {
	if r.db == nil {
		points := 0
		r.store.RLock()
		defer r.store.RUnlock()
		for _, user := range r.store.Users {
			if user.ID == userID {
				points = user.ExperiencePoints
				break
			}
		}
		return model.ExperienceProgress{TotalEXP: points, Level: points/100 + 1, LevelProgress: points % 100, LevelTarget: 100}, nil
	}
	return r.learningExperience(ctx, userID)
}

func (r *Repository) getLearningCatalogMemory(userID, locale string) model.LearningCatalog {
	r.store.RLock()
	defer r.store.RUnlock()
	result := model.LearningCatalog{}
	result.Clusters = append(result.Clusters, r.store.LearningClusters...)
	result.Programs = append(result.Programs, r.store.AcademicPrograms...)
	progressByItem := make(map[string]model.LearningProgress)
	for _, progress := range r.store.LearningProgress {
		if progress.UserID == userID {
			result.Progress = append(result.Progress, progress)
			progressByItem[progress.ItemID] = progress
		}
	}
	for _, item := range r.store.LearningItems {
		if progress, ok := progressByItem[item.ID]; ok {
			item.Progress = &progress
		}
		result.Items = append(result.Items, item)
	}
	points := 0
	for _, user := range r.store.Users {
		if user.ID == userID {
			points = user.ExperiencePoints
			break
		}
	}
	result.Experience = model.ExperienceProgress{TotalEXP: points, Level: points/100 + 1, LevelProgress: points % 100, LevelTarget: 100}
	return result
}

func (r *Repository) completeLearningMemory(userID, itemID, reflection, outcome string) model.LearningProgress {
	r.store.Lock()
	defer r.store.Unlock()
	for index := range r.store.LearningProgress {
		if r.store.LearningProgress[index].UserID == userID && r.store.LearningProgress[index].ItemID == itemID {
			r.store.LearningProgress[index].State = "completed"
			r.store.LearningProgress[index].ReflectionEncrypted = reflection
			r.store.LearningProgress[index].OutcomeEncrypted = outcome
			now := time.Now().UTC()
			r.store.LearningProgress[index].CompletedAt = &now
			return r.store.LearningProgress[index]
		}
	}
	now := time.Now().UTC()
	row := model.LearningProgress{UserID: userID, ItemID: itemID, State: "completed", CompletedAt: &now, ReflectionEncrypted: reflection, OutcomeEncrypted: outcome}
	r.store.LearningProgress = append(r.store.LearningProgress, row)
	return row
}
