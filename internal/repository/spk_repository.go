package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entintervention "github.com/gamblock-ai/gamblock-ai-backend/ent/interventionrecord"
	entspkpref "github.com/gamblock-ai/gamblock-ai-backend/ent/spkpreference"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/google/uuid"
)

// SpkInputData is the privacy-safe data the SPK engine consumes. It is derived
// only from aggregate protection events, account activity, and the user's own
// reported intention; it never contains URLs, domains, or browsing content.
type SpkInputData struct {
	BlockedAttemptsToday   int
	BlockedActiveDays7d    int
	RecoveryStreakDays     int
	HasActivity            bool
	DailyMissionsCompleted int
	DailyMissionsTotal     int
	HasAnyMission          bool
	LearningActivities7d   int
	HasAnyLearning         bool
	AccountabilityEnabled  bool
	BlockedEventTimes      []time.Time
	HasIntention           bool
	SchoolImpact           string
	MoneySpent             string
	ScreenTime             string
	QuitAttempts           string
	QuitMotivation         string
	PersonalIntention      string
}

// SpkInputData assembles every SPK input from the in-memory cache (refreshed
// from ent when a database is configured) plus the latest active intention.
func (r *Repository) SpkInputData(ctx context.Context, userID string, now time.Time) (SpkInputData, error) {
	if r.db != nil {
		r.RefreshStore(ctx)
	}
	snapshot := r.store.Snapshot()
	start := startOfDay(now.UTC()).AddDate(0, 0, -6)

	var data SpkInputData
	today := startOfDay(now.UTC())
	activeDays := map[string]struct{}{}
	blockedDays := map[string]struct{}{}
	for _, event := range snapshot.AggregateEvents {
		if event.UserID != userID || event.EventType != "block_count_sync" {
			continue
		}
		date := startOfDay(event.EventDate)
		if !date.Before(today) {
			data.BlockedAttemptsToday += event.Count
		}
		if date.Before(start) {
			continue
		}
		if event.Count > 0 {
			blockedDays[event.EventDate.UTC().Format("2006-01-02")] = struct{}{}
		}
	}
	data.BlockedActiveDays7d = len(blockedDays)

	for _, checkIn := range snapshot.CheckIns {
		if checkIn.UserID == userID && !checkIn.CreatedAt.Before(start) {
			activeDays[checkIn.CreatedAt.UTC().Format("2006-01-02")] = struct{}{}
		}
	}
	for _, entry := range snapshot.JournalEntries {
		if entry.UserID == userID && !entry.CreatedAt.Before(start) {
			activeDays[entry.CreatedAt.UTC().Format("2006-01-02")] = struct{}{}
		}
	}
	for _, mission := range snapshot.Missions {
		if mission.UserID == userID && mission.Date >= start.Format("2006-01-02") && mission.CompletedTaskCount() > 0 {
			activeDays[mission.Date] = struct{}{}
		}
	}
	for _, progress := range snapshot.EducationProgress {
		if progress.UserID == userID && !progress.UpdatedAt.Before(start) {
			activeDays[progress.UpdatedAt.UTC().Format("2006-01-02")] = struct{}{}
		}
	}
	for _, progress := range snapshot.LearningProgress {
		timestamp := progress.UpdatedAt
		if timestamp.IsZero() {
			timestamp = progress.CreatedAt
		}
		if progress.UserID == userID && !timestamp.IsZero() && !timestamp.Before(start) {
			activeDays[timestamp.UTC().Format("2006-01-02")] = struct{}{}
		}
	}
	for _, record := range snapshot.RecoveryRecords {
		if record.UserID == userID && record.Kind == "weekly_review" && record.RecordDate >= start.Format("2006-01-02") {
			activeDays[record.RecordDate] = struct{}{}
		}
	}
	data.RecoveryStreakDays = contiguousDays(activeDays, today)
	data.HasActivity = len(activeDays) > 0

	date, missionStart, missionEnd := spkJakartaDay(now)
	mission, _, err := r.GetMissionByDate(ctx, userID, date, missionStart, missionEnd)
	if err != nil {
		return SpkInputData{}, err
	}
	data.DailyMissionsCompleted = mission.CompletedTaskCount()
	data.DailyMissionsTotal = mission.TotalCount
	for _, item := range snapshot.Missions {
		if item.UserID == userID {
			data.HasAnyMission = true
			break
		}
	}

	for _, progress := range snapshot.LearningProgress {
		if progress.UserID != userID {
			continue
		}
		data.HasAnyLearning = true
		timestamp := progress.UpdatedAt
		if timestamp.IsZero() {
			timestamp = progress.CreatedAt
		}
		if !timestamp.IsZero() && !timestamp.Before(start) {
			data.LearningActivities7d++
		}
	}
	for _, progress := range snapshot.EducationProgress {
		if progress.UserID == userID {
			data.HasAnyLearning = true
			if !progress.UpdatedAt.Before(start) && len(progress.CompletedSectionIDs) > 0 {
				data.LearningActivities7d++
			}
		}
	}

	for _, membership := range snapshot.AccountabilityMemberships {
		if membership.StudentID == userID && membership.Status == "active" {
			data.AccountabilityEnabled = true
			break
		}
	}

	windowStart := now.AddDate(0, 0, -14)
	for _, event := range snapshot.BlockedEvents {
		if event.UserID == userID && !event.OccurredAt.Before(windowStart) {
			data.BlockedEventTimes = append(data.BlockedEventTimes, event.OccurredAt)
		}
	}

	if intention, ok := r.GetIntention(ctx, userID); ok && intention.Text != "" {
		data.HasIntention = true
		data.SchoolImpact = intention.SchoolImpact
		data.MoneySpent = intention.MoneySpent
		data.ScreenTime = intention.ScreenTime
		data.QuitAttempts = intention.QuitAttempts
		data.QuitMotivation = intention.QuitMotivation
		data.PersonalIntention = intention.Text
	}
	return data, nil
}

// SpkPreference returns the stored opt-in preference or a default-off shape.
func (r *Repository) SpkPreference(ctx context.Context, userID string) (model.SpkPreference, error) {
	if r.db != nil {
		row, err := r.db.SpkPreference.Query().Where(entspkpref.UserIDEQ(userID)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return defaultSpkPreference(userID), nil
			}
			return model.SpkPreference{}, err
		}
		return spkPreferenceFromEnt(row), nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	for _, pref := range r.store.SpkPreferences {
		if pref.UserID == userID {
			return pref, nil
		}
	}
	return defaultSpkPreference(userID), nil
}

// UpsertSpkPreference stores the full SPK privacy set.
func (r *Repository) UpsertSpkPreference(ctx context.Context, userID string, pref model.SpkPreference) (model.SpkPreference, error) {
	if r.db != nil {
		existing, err := r.db.SpkPreference.Query().Where(entspkpref.UserIDEQ(userID)).Only(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return model.SpkPreference{}, err
		}
		var row *ent.SpkPreference
		if err == nil {
			row, err = r.db.SpkPreference.UpdateOneID(existing.ID).
				SetSpkRecommendationEnabled(pref.SpkRecommendationEnabled).
				SetSpkUseProtection(pref.SpkUseProtection).
				SetSpkUseRecovery(pref.SpkUseRecovery).
				SetSpkUsePersonal(pref.SpkUsePersonal).
				SetLlmPersonalizationEnabled(pref.LLMPersonalizationEnabled).
				Save(ctx)
		} else {
			row, err = r.db.SpkPreference.Create().
				SetID("spk_" + uuid.NewString()).
				SetUserID(userID).
				SetSpkRecommendationEnabled(pref.SpkRecommendationEnabled).
				SetSpkUseProtection(pref.SpkUseProtection).
				SetSpkUseRecovery(pref.SpkUseRecovery).
				SetSpkUsePersonal(pref.SpkUsePersonal).
				SetLlmPersonalizationEnabled(pref.LLMPersonalizationEnabled).
				Save(ctx)
		}
		if err != nil {
			return model.SpkPreference{}, err
		}
		r.RefreshStore(ctx)
		return spkPreferenceFromEnt(row), nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	for index := range r.store.SpkPreferences {
		if r.store.SpkPreferences[index].UserID == userID {
			r.store.SpkPreferences[index].SpkRecommendationEnabled = pref.SpkRecommendationEnabled
			r.store.SpkPreferences[index].SpkUseProtection = pref.SpkUseProtection
			r.store.SpkPreferences[index].SpkUseRecovery = pref.SpkUseRecovery
			r.store.SpkPreferences[index].SpkUsePersonal = pref.SpkUsePersonal
			r.store.SpkPreferences[index].LLMPersonalizationEnabled = pref.LLMPersonalizationEnabled
			r.store.SpkPreferences[index].UpdatedAt = time.Now().UTC()
			return r.store.SpkPreferences[index], nil
		}
	}
	stored := defaultSpkPreference(userID)
	stored.SpkRecommendationEnabled = pref.SpkRecommendationEnabled
	stored.SpkUseProtection = pref.SpkUseProtection
	stored.SpkUseRecovery = pref.SpkUseRecovery
	stored.SpkUsePersonal = pref.SpkUsePersonal
	stored.LLMPersonalizationEnabled = pref.LLMPersonalizationEnabled
	stored.CreatedAt = time.Now().UTC()
	stored.UpdatedAt = stored.CreatedAt
	r.store.SpkPreferences = append(r.store.SpkPreferences, stored)
	return stored, nil
}

// InterventionRecords returns every persisted SPK recommendation for a user.
func (r *Repository) InterventionRecords(ctx context.Context, userID string) ([]model.InterventionRecord, error) {
	if r.db != nil {
		rows, err := r.db.InterventionRecord.Query().Where(entintervention.UserIDEQ(userID)).Order(ent.Desc(entintervention.FieldRecommendedAt)).All(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]model.InterventionRecord, 0, len(rows))
		for _, row := range rows {
			out = append(out, interventionRecordFromEnt(row))
		}
		return out, nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	var out []model.InterventionRecord
	for _, record := range r.store.InterventionRecords {
		if record.UserID == userID {
			out = append(out, record)
		}
	}
	return out, nil
}

// TodayInterventionRecord returns the daily recommendation record for a user,
// scoped to the given Jakarta day, or nil when none exists yet.
func (r *Repository) TodayInterventionRecord(ctx context.Context, userID string, dayStart, dayEnd time.Time) (*model.InterventionRecord, error) {
	if r.db != nil {
		row, err := r.db.InterventionRecord.Query().Where(
			entintervention.UserIDEQ(userID),
			entintervention.RecommendedAtGTE(dayStart),
			entintervention.RecommendedAtLT(dayEnd),
		).Order(ent.Desc(entintervention.FieldRecommendedAt)).First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		record := interventionRecordFromEnt(row)
		return &record, nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	for i := len(r.store.InterventionRecords) - 1; i >= 0; i-- {
		record := r.store.InterventionRecords[i]
		if record.UserID == userID && !record.RecommendedAt.Before(dayStart) && record.RecommendedAt.Before(dayEnd) {
			cp := record
			return &cp, nil
		}
	}
	return nil, nil
}

// UpsertInterventionRecord creates or updates a persisted SPK recommendation.
func (r *Repository) UpsertInterventionRecord(ctx context.Context, record model.InterventionRecord) (model.InterventionRecord, error) {
	if r.db != nil {
		row, err := r.db.InterventionRecord.Create().
			SetID(record.ID).
			SetUserID(record.UserID).
			SetInterventionKey(record.InterventionKey).
			SetResponseType(record.ResponseType).
			SetSupportLevel(entintervention.SupportLevel(record.SupportLevel)).
			SetEngagementLevel(entintervention.EngagementLevel(record.EngagementLevel)).
			SetReadinessLevel(record.ReadinessLevel).
			SetStatus(entintervention.Status(record.Status)).
			SetRecommendedAt(record.RecommendedAt).
			SetNillableCompletedAt(record.CompletedAt).
			SetEffectivenessStatus(entintervention.EffectivenessStatus(record.EffectivenessStatus)).
			SetNillablePersonalizedMessage(optionalText(record.PersonalizedMessage)).
			SetNillablePersonalizedExplanation(optionalText(record.PersonalizedExplanation)).
			SetLlmUsed(record.LLMUsed).
			Save(ctx)
		if err != nil {
			return model.InterventionRecord{}, err
		}
		r.RefreshStore(ctx)
		return interventionRecordFromEnt(row), nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	record.CreatedAt = time.Now().UTC()
	record.UpdatedAt = record.CreatedAt
	r.store.InterventionRecords = append(r.store.InterventionRecords, record)
	return record, nil
}

// CompleteIntervention marks a daily recommendation as completed by its owner.
func (r *Repository) CompleteIntervention(ctx context.Context, userID, recordID string) (*model.InterventionRecord, error) {
	if r.db != nil {
		row, err := r.db.InterventionRecord.Query().Where(
			entintervention.ID(recordID),
			entintervention.UserIDEQ(userID),
		).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("intervention record not found")
			}
			return nil, err
		}
		row, err = row.Update().SetStatus(entintervention.StatusCompleted).SetCompletedAt(time.Now().UTC()).Save(ctx)
		if err != nil {
			return nil, err
		}
		r.RefreshStore(ctx)
		record := interventionRecordFromEnt(row)
		return &record, nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	for index := range r.store.InterventionRecords {
		if r.store.InterventionRecords[index].UserID == userID && r.store.InterventionRecords[index].ID == recordID {
			now := time.Now().UTC()
			r.store.InterventionRecords[index].Status = "completed"
			r.store.InterventionRecords[index].CompletedAt = &now
			r.store.InterventionRecords[index].UpdatedAt = now
			cp := r.store.InterventionRecords[index]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("intervention record not found")
}

// SaveBlockedEvents persists a bounded batch of system-generated blocked-event
// timestamps, skipping exact duplicates per user.
func (r *Repository) SaveBlockedEvents(ctx context.Context, userID, deviceID string, occurredAt []time.Time) (int, error) {
	if len(occurredAt) == 0 {
		return 0, nil
	}
	if r.db != nil {
		created := 0
		for _, ts := range occurredAt {
			_, err := r.db.BlockedEvent.Create().
				SetID("blk_" + uuid.NewString()).
				SetUserID(userID).
				SetNillableDeviceID(optional(deviceID)).
				SetOccurredAt(ts.UTC()).
				Save(ctx)
			if err != nil {
				if ent.IsConstraintError(err) {
					continue
				}
				return created, err
			}
			created++
		}
		r.RefreshStore(ctx)
		return created, nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	seen := make(map[time.Time]bool, len(r.store.BlockedEvents))
	for _, event := range r.store.BlockedEvents {
		if event.UserID == userID {
			seen[event.OccurredAt.UTC()] = true
		}
	}
	created := 0
	for _, ts := range occurredAt {
		key := ts.UTC()
		if seen[key] {
			continue
		}
		seen[key] = true
		r.store.BlockedEvents = append(r.store.BlockedEvents, model.BlockedEvent{
			ID:         "blk_" + uuid.NewString(),
			UserID:     userID,
			DeviceID:   deviceID,
			OccurredAt: key,
			CreatedAt:  time.Now().UTC(),
		})
		created++
	}
	return created, nil
}

// UpdateInterventionRecord refreshes the decision and LLM fields of an
// existing daily recommendation without resetting its completion state.
func (r *Repository) UpdateInterventionRecord(ctx context.Context, userID string, record model.InterventionRecord) (model.InterventionRecord, error) {
	if r.db != nil {
		row, err := r.db.InterventionRecord.Query().Where(
			entintervention.ID(record.ID),
			entintervention.UserIDEQ(userID),
		).Only(ctx)
		if err != nil {
			return model.InterventionRecord{}, err
		}
		row, err = row.Update().
			SetInterventionKey(record.InterventionKey).
			SetResponseType(record.ResponseType).
			SetSupportLevel(entintervention.SupportLevel(record.SupportLevel)).
			SetEngagementLevel(entintervention.EngagementLevel(record.EngagementLevel)).
			SetReadinessLevel(record.ReadinessLevel).
			SetRecommendedAt(record.RecommendedAt).
			SetNillablePersonalizedMessage(optionalText(record.PersonalizedMessage)).
			SetNillablePersonalizedExplanation(optionalText(record.PersonalizedExplanation)).
			SetLlmUsed(record.LLMUsed).
			Save(ctx)
		if err != nil {
			return model.InterventionRecord{}, err
		}
		r.RefreshStore(ctx)
		return interventionRecordFromEnt(row), nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	for index := range r.store.InterventionRecords {
		item := &r.store.InterventionRecords[index]
		if item.ID == record.ID && item.UserID == userID {
			item.InterventionKey = record.InterventionKey
			item.ResponseType = record.ResponseType
			item.SupportLevel = record.SupportLevel
			item.EngagementLevel = record.EngagementLevel
			item.ReadinessLevel = record.ReadinessLevel
			item.RecommendedAt = record.RecommendedAt
			item.PersonalizedMessage = record.PersonalizedMessage
			item.PersonalizedExplanation = record.PersonalizedExplanation
			item.LLMUsed = record.LLMUsed
			item.UpdatedAt = time.Now().UTC()
			return *item, nil
		}
	}
	return model.InterventionRecord{}, fmt.Errorf("intervention record not found")
}

// UpdateInterventionEffectiveness stores a lazily computed effectiveness result.
func (r *Repository) UpdateInterventionEffectiveness(ctx context.Context, userID, recordID, status string) error {
	if r.db != nil {
		row, err := r.db.InterventionRecord.Query().Where(
			entintervention.ID(recordID),
			entintervention.UserIDEQ(userID),
		).Only(ctx)
		if err != nil {
			return err
		}
		_, err = row.Update().SetEffectivenessStatus(entintervention.EffectivenessStatus(status)).Save(ctx)
		if err != nil {
			return err
		}
		r.RefreshStore(ctx)
		return nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	for index := range r.store.InterventionRecords {
		if r.store.InterventionRecords[index].ID == recordID && r.store.InterventionRecords[index].UserID == userID {
			r.store.InterventionRecords[index].EffectivenessStatus = status
			r.store.InterventionRecords[index].UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return fmt.Errorf("intervention record not found")
}

// BlockCountsByDate returns block_count_sync totals keyed by UTC date within
// [start, end), used by the effectiveness feedback loop.
func (r *Repository) BlockCountsByDate(ctx context.Context, userID string, start, end time.Time) (map[string]int, error) {
	if r.db != nil {
		r.RefreshStore(ctx)
	}
	snapshot := r.store.Snapshot()
	counts := map[string]int{}
	for _, event := range snapshot.AggregateEvents {
		if event.UserID != userID || event.EventType != "block_count_sync" {
			continue
		}
		if event.EventDate.Before(start) || !event.EventDate.Before(end) {
			continue
		}
		counts[event.EventDate.UTC().Format("2006-01-02")] += event.Count
	}
	return counts, nil
}

func defaultSpkPreference(userID string) model.SpkPreference {
	return model.SpkPreference{
		ID:                        "spk_" + uuid.NewString(),
		UserID:                    userID,
		SpkRecommendationEnabled:  true,
		SpkUseProtection:          true,
		SpkUseRecovery:            true,
		SpkUsePersonal:            true,
		LLMPersonalizationEnabled: true,
	}
}

func spkPreferenceFromEnt(row *ent.SpkPreference) model.SpkPreference {
	return model.SpkPreference{
		ID:                        row.ID,
		UserID:                    row.UserID,
		SpkRecommendationEnabled:  row.SpkRecommendationEnabled,
		SpkUseProtection:          row.SpkUseProtection,
		SpkUseRecovery:            row.SpkUseRecovery,
		SpkUsePersonal:            row.SpkUsePersonal,
		LLMPersonalizationEnabled: row.LlmPersonalizationEnabled,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}
}

func interventionRecordFromEnt(row *ent.InterventionRecord) model.InterventionRecord {
	return model.InterventionRecord{
		ID:                       row.ID,
		UserID:                   row.UserID,
		InterventionKey:          row.InterventionKey,
		ResponseType:             row.ResponseType,
		SupportLevel:             row.SupportLevel.String(),
		EngagementLevel:          row.EngagementLevel.String(),
		ReadinessLevel:           row.ReadinessLevel,
		Status:                   row.Status.String(),
		RecommendedAt:            row.RecommendedAt,
		CompletedAt:              row.CompletedAt,
		EffectivenessStatus:      row.EffectivenessStatus.String(),
		PersonalizedMessage:      value(row.PersonalizedMessage),
		PersonalizedExplanation:  value(row.PersonalizedExplanation),
		LLMUsed:                  row.LlmUsed,
		CreatedAt:                row.CreatedAt,
		UpdatedAt:                row.UpdatedAt,
	}
}

func optionalText(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}

// spkJakartaDay returns the Asia/Jakarta calendar date plus its UTC window.
func spkJakartaDay(now time.Time) (string, time.Time, time.Time) {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	local := now.In(jakarta)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, jakarta)
	return start.Format("2006-01-02"), start.UTC(), start.AddDate(0, 0, 1).UTC()
}
