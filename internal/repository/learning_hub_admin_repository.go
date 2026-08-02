package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/academicprogram"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/institution"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningcluster"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningitem"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningrevision"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

var (
	ErrLearningAdminNotFound      = errors.New("learning hub admin resource not found")
	ErrLearningAdminConflict      = errors.New("learning hub draft was updated by another editor")
	ErrLearningTaxonomyConflict   = errors.New("learning hub taxonomy resource is in use")
	ErrLearningAdminInvalidStatus = errors.New("learning hub status transition is invalid")
)

func learningRevisionFromEnt(row *ent.LearningRevision) model.LearningRevision {
	document, _ := unpackLearningRevisionDocument(row.DocumentJSON)
	return model.LearningRevision{ID: row.ID, ItemID: row.ItemID, Revision: row.Revision, Document: document, Kind: row.Kind.String(), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}
}

func adminLearningClusterFromEnt(row *ent.LearningCluster) model.AdminLearningCluster {
	return model.AdminLearningCluster{LearningCluster: model.LearningCluster{ID: row.ID, Slug: row.Slug, Title: row.TitleID, Description: row.DescriptionID, SortOrder: row.SortOrder}, TitleID: row.TitleID, TitleEN: row.TitleEn, DescriptionID: row.DescriptionID, DescriptionEN: row.DescriptionEn, Active: row.Active}
}

func (r *Repository) publishedLearningDocument(ctx context.Context, itemID string) (map[string]any, error) {
	if r.db == nil {
		var latest *model.LearningRevision
		revisions := r.store.Snapshot().LearningRevisions
		for index := range revisions {
			row := revisions[index]
			if row.ItemID == itemID && row.Kind == "published" && (latest == nil || row.Revision > latest.Revision) {
				copy := row
				latest = &copy
			}
		}
		if latest == nil {
			return nil, nil
		}
		return latest.Document, nil
	}
	row, err := r.db.LearningRevision.Query().Where(learningrevision.ItemIDEQ(itemID), learningrevision.KindEQ(learningrevision.KindPublished)).Order(ent.Desc(learningrevision.FieldRevision)).First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.DocumentJSON, nil
}

func adminLearningItemFromEnt(row *ent.LearningItem, published map[string]any) model.AdminLearningItem {
	publishedDocument, _ := unpackLearningRevisionDocument(published)
	item := model.AdminLearningItem{
		LearningItem: learningItemFromEntDocument(row, row.DocumentJSON, "id"),
		TitleID:      row.TitleID, TitleEN: row.TitleEn, SummaryID: row.SummaryID, SummaryEN: row.SummaryEn,
		Status: row.Status.String(), DraftRevision: row.DraftRevision, PublishedRevision: row.PublishedRevision,
		DraftDocument: row.DocumentJSON, PublishedDocument: publishedDocument, PublishedAt: row.PublishedAt, ArchivedAt: row.ArchivedAt,
		CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	item.LearningItem.Title = row.TitleID
	item.LearningItem.Summary = row.SummaryID
	return item
}

func adminLearningItemFromMemory(row model.AdminLearningItem) model.AdminLearningItem {
	return row
}

func (r *Repository) ListAdminLearningItems(ctx context.Context, status string) ([]model.AdminLearningItem, error) {
	status = strings.TrimSpace(status)
	if r.db == nil {
		rows := append([]model.AdminLearningItem(nil), r.store.Snapshot().AdminLearningItems...)
		if len(rows) == 0 {
			for _, item := range r.store.Snapshot().LearningItems {
				rows = append(rows, model.AdminLearningItem{LearningItem: item, Status: "published", DraftRevision: 1, PublishedRevision: 1, DraftDocument: map[string]any{}})
			}
		}
		if status != "" {
			rows = slicesDeleteAdminItems(rows, status)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].UpdatedAt.After(rows[j].UpdatedAt) })
		return rows, nil
	}
	query := r.db.LearningItem.Query()
	if status != "" {
		query.Where(learningitem.StatusEQ(learningitem.Status(status)))
	}
	rows, err := query.Order(ent.Desc(learningitem.FieldUpdatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.AdminLearningItem, 0, len(rows))
	for _, row := range rows {
		published, err := r.publishedLearningDocument(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, adminLearningItemFromEnt(row, published))
	}
	return out, nil
}

func slicesDeleteAdminItems(items []model.AdminLearningItem, status string) []model.AdminLearningItem {
	out := items[:0]
	for _, item := range items {
		if item.Status == status {
			out = append(out, item)
		}
	}
	return out
}

func (r *Repository) GetAdminLearningItem(ctx context.Context, id string) (model.AdminLearningItem, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().AdminLearningItems {
			if item.ID == id {
				return adminLearningItemFromMemory(item), nil
			}
		}
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	row, err := r.db.LearningItem.Get(ctx, id)
	if ent.IsNotFound(err) {
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	published, err := r.publishedLearningDocument(ctx, id)
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	return adminLearningItemFromEnt(row, published), nil
}

func (r *Repository) CreateAdminLearningItem(ctx context.Context, actor string, draft model.LearningItemDraft) (model.AdminLearningItem, error) {
	now := time.Now().UTC()
	id := "item_" + uuid.NewString()
	if r.db == nil {
		row := model.AdminLearningItem{LearningItem: model.LearningItem{ID: id, Slug: draft.Slug, Kind: draft.Kind, Title: draft.TitleID, Summary: draft.SummaryID}, TitleID: draft.TitleID, TitleEN: draft.TitleEN, SummaryID: draft.SummaryID, SummaryEN: draft.SummaryEN, Status: "draft", DraftRevision: 1, DraftDocument: draft.Document, CreatedBy: actor, UpdatedBy: actor, CreatedAt: now, UpdatedAt: now}
		row.LearningItem.Title = draft.TitleID
		row.LearningItem.Summary = draft.SummaryID
		r.store.Lock()
		r.store.AdminLearningItems = append(r.store.AdminLearningItems, row)
		r.store.LearningRevisions = append(r.store.LearningRevisions, model.LearningRevision{ID: "learnrev_" + uuid.NewString(), ItemID: id, Revision: 1, Document: learningRevisionDocument(draft.Document, draft.Slug, draft.Kind, draft.TitleID, draft.TitleEN, draft.SummaryID, draft.SummaryEN), Kind: "draft", CreatedBy: actor, CreatedAt: now})
		r.store.Unlock()
		return row, nil
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	_, err = tx.LearningItem.Create().SetID(id).SetSlug(draft.Slug).SetKind(learningitem.Kind(draft.Kind)).SetTitleID(draft.TitleID).SetTitleEn(draft.TitleEN).SetSummaryID(draft.SummaryID).SetSummaryEn(draft.SummaryEN).SetDocumentJSON(draft.Document).SetStatus(learningitem.StatusDraft).SetDraftRevision(1).SetCreatedBy(actor).SetUpdatedBy(actor).Save(ctx)
	if err == nil {
		_, err = tx.LearningRevision.Create().SetID("learnrev_" + uuid.NewString()).SetItemID(id).SetRevision(1).SetDocumentJSON(learningRevisionDocument(draft.Document, draft.Slug, draft.Kind, draft.TitleID, draft.TitleEN, draft.SummaryID, draft.SummaryEN)).SetKind(learningrevision.KindDraft).SetCreatedBy(actor).Save(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return model.AdminLearningItem{}, ErrLearningAdminConflict
		}
		return model.AdminLearningItem{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.AdminLearningItem{}, err
	}
	r.RefreshStore(ctx)
	return r.GetAdminLearningItem(ctx, id)
}

func (r *Repository) UpdateAdminLearningItem(ctx context.Context, actor, id string, expectedRevision int, draft model.LearningItemDraft) (model.AdminLearningItem, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.AdminLearningItems {
			row := &r.store.AdminLearningItems[index]
			if row.ID != id {
				continue
			}
			if row.DraftRevision != expectedRevision {
				return model.AdminLearningItem{}, ErrLearningAdminConflict
			}
			row.Slug, row.Kind, row.Title, row.Summary = draft.Slug, draft.Kind, draft.TitleID, draft.SummaryID
			row.TitleID, row.TitleEN, row.SummaryID, row.SummaryEN = draft.TitleID, draft.TitleEN, draft.SummaryID, draft.SummaryEN
			row.DraftDocument, row.UpdatedBy, row.UpdatedAt = draft.Document, actor, time.Now().UTC()
			row.DraftRevision++
			if row.Status == "published" || row.Status == "archived" {
				row.Status, row.ArchivedAt = "draft", nil
			}
			r.store.LearningRevisions = append(r.store.LearningRevisions, model.LearningRevision{ID: "learnrev_" + uuid.NewString(), ItemID: id, Revision: row.DraftRevision, Document: learningRevisionDocument(draft.Document, draft.Slug, draft.Kind, draft.TitleID, draft.TitleEN, draft.SummaryID, draft.SummaryEN), Kind: "draft", CreatedBy: actor, CreatedAt: row.UpdatedAt})
			return *row, nil
		}
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	row, err := r.db.LearningItem.Query().Where(learningitem.IDEQ(id)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	nextRevision := expectedRevision + 1
	if row.DraftRevision != expectedRevision {
		return model.AdminLearningItem{}, ErrLearningAdminConflict
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	// Older seeded rows stored only the metadata document in their published
	// revision. Upgrade that revision in-place before changing top-level draft
	// fields so the historical public snapshot remains complete.
	if row.PublishedRevision > 0 {
		published, publishedErr := tx.LearningRevision.Query().Where(learningrevision.ItemIDEQ(id), learningrevision.KindEQ(learningrevision.KindPublished)).Order(ent.Desc(learningrevision.FieldRevision)).First(ctx)
		if publishedErr != nil && !ent.IsNotFound(publishedErr) {
			_ = tx.Rollback()
			return model.AdminLearningItem{}, publishedErr
		}
		if published != nil {
			_, snapshot := unpackLearningRevisionDocument(published.DocumentJSON)
			if snapshot == nil {
				if _, publishedErr = tx.LearningRevision.UpdateOne(published).SetDocumentJSON(learningRevisionDocument(published.DocumentJSON, row.Slug, row.Kind.String(), row.TitleID, row.TitleEn, row.SummaryID, row.SummaryEn)).Save(ctx); publishedErr != nil {
					_ = tx.Rollback()
					return model.AdminLearningItem{}, publishedErr
				}
			}
		}
	}
	update := tx.LearningItem.UpdateOneID(id).SetSlug(draft.Slug).SetKind(learningitem.Kind(draft.Kind)).SetTitleID(draft.TitleID).SetTitleEn(draft.TitleEN).SetSummaryID(draft.SummaryID).SetSummaryEn(draft.SummaryEN).SetDocumentJSON(draft.Document).SetDraftRevision(nextRevision).SetUpdatedBy(actor).SetStatus(learningitem.StatusDraft).ClearArchivedAt()
	if _, err = update.Save(ctx); err == nil {
		_, err = tx.LearningRevision.Create().SetID("learnrev_" + uuid.NewString()).SetItemID(id).SetRevision(nextRevision).SetDocumentJSON(learningRevisionDocument(draft.Document, draft.Slug, draft.Kind, draft.TitleID, draft.TitleEN, draft.SummaryID, draft.SummaryEN)).SetKind(learningrevision.KindDraft).SetCreatedBy(actor).Save(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return model.AdminLearningItem{}, ErrLearningAdminConflict
		}
		return model.AdminLearningItem{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.AdminLearningItem{}, err
	}
	r.RefreshStore(ctx)
	return r.GetAdminLearningItem(ctx, id)
}

func (r *Repository) SetAdminLearningStatus(ctx context.Context, actor, id, status string) (model.AdminLearningItem, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.AdminLearningItems {
			row := &r.store.AdminLearningItems[index]
			if row.ID != id {
				continue
			}
			now := time.Now().UTC()
			row.Status, row.UpdatedBy, row.UpdatedAt = status, actor, now
			if status == "published" {
				row.PublishedRevision, row.PublishedAt, row.ArchivedAt = row.DraftRevision, &now, nil
				row.PublishedDocument = cloneLearningDocument(row.DraftDocument)
				r.store.LearningItems = syncPublishedLearningMemory(r.store.LearningItems, row)
				r.store.LearningRevisions = append(r.store.LearningRevisions, model.LearningRevision{ID: "learnrev_" + uuid.NewString(), ItemID: id, Revision: row.DraftRevision, Document: learningRevisionDocument(row.DraftDocument, row.Slug, row.Kind, row.TitleID, row.TitleEN, row.SummaryID, row.SummaryEN), Kind: "published", CreatedBy: actor, CreatedAt: now})
			}
			if status == "archived" {
				row.ArchivedAt = &now
				r.store.LearningItems = removePublishedLearningMemory(r.store.LearningItems, row.ID)
			}
			return *row, nil
		}
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	row, err := r.db.LearningItem.Query().Where(learningitem.IDEQ(id)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	now := time.Now().UTC()
	if status == "published" {
		tx, txErr := r.db.Tx(ctx)
		if txErr != nil {
			return model.AdminLearningItem{}, txErr
		}
		_, err = tx.LearningItem.UpdateOneID(id).SetStatus(learningitem.StatusPublished).SetUpdatedBy(actor).SetPublishedRevision(row.DraftRevision).SetPublishedAt(now).ClearArchivedAt().Save(ctx)
		if err == nil {
			_, err = tx.LearningRevision.Create().SetID("learnrev_" + uuid.NewString()).SetItemID(id).SetRevision(row.DraftRevision).SetDocumentJSON(learningRevisionDocument(row.DocumentJSON, row.Slug, row.Kind.String(), row.TitleID, row.TitleEn, row.SummaryID, row.SummaryEn)).SetKind(learningrevision.KindPublished).SetCreatedBy(actor).Save(ctx)
		}
		if err != nil {
			_ = tx.Rollback()
			return model.AdminLearningItem{}, err
		}
		if err = tx.Commit(); err != nil {
			return model.AdminLearningItem{}, err
		}
	} else {
		update := r.db.LearningItem.UpdateOne(row).SetStatus(learningitem.Status(status)).SetUpdatedBy(actor)
		if status == "archived" {
			update.SetArchivedAt(now)
		}
		if _, err = update.Save(ctx); ent.IsNotFound(err) {
			return model.AdminLearningItem{}, ErrLearningAdminNotFound
		}
		if err != nil {
			return model.AdminLearningItem{}, err
		}
	}
	r.RefreshStore(ctx)
	return r.GetAdminLearningItem(ctx, id)
}

func syncPublishedLearningMemory(items []model.LearningItem, row *model.AdminLearningItem) []model.LearningItem {
	for index := range items {
		if items[index].ID == row.ID {
			items[index] = row.LearningItem
			return items
		}
	}
	return append(items, row.LearningItem)
}

func removePublishedLearningMemory(items []model.LearningItem, itemID string) []model.LearningItem {
	out := items[:0]
	for _, item := range items {
		if item.ID != itemID {
			out = append(out, item)
		}
	}
	return out
}

func (r *Repository) ListLearningRevisions(ctx context.Context, itemID string) ([]model.LearningRevision, error) {
	if r.db == nil {
		rows := make([]model.LearningRevision, 0)
		for _, row := range r.store.Snapshot().LearningRevisions {
			if row.ItemID == itemID {
				row.Document, _ = unpackLearningRevisionDocument(row.Document)
				rows = append(rows, row)
			}
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
		return rows, nil
	}
	if _, err := r.db.LearningItem.Get(ctx, itemID); ent.IsNotFound(err) {
		return nil, ErrLearningAdminNotFound
	} else if err != nil {
		return nil, err
	}
	rows, err := r.db.LearningRevision.Query().Where(learningrevision.ItemIDEQ(itemID)).Order(ent.Desc(learningrevision.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.LearningRevision, 0, len(rows))
	for _, row := range rows {
		out = append(out, learningRevisionFromEnt(row))
	}
	return out, nil
}

func (r *Repository) RollbackLearningItem(ctx context.Context, actor, itemID, revisionID string) (model.AdminLearningItem, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		var source *model.LearningRevision
		for index := range r.store.LearningRevisions {
			row := &r.store.LearningRevisions[index]
			if row.ID == revisionID && row.ItemID == itemID && (row.Kind == "published" || row.Kind == "rollback" || row.Kind == "draft") {
				source = row
				break
			}
		}
		if source == nil {
			return model.AdminLearningItem{}, ErrLearningAdminNotFound
		}
		for index := range r.store.AdminLearningItems {
			row := &r.store.AdminLearningItems[index]
			if row.ID != itemID {
				continue
			}
			document, snapshot := unpackLearningRevisionDocument(source.Document)
			row.DraftRevision++
			row.Slug = snapshotString(snapshot, "slug", row.Slug)
			row.Kind = snapshotString(snapshot, "kind", row.Kind)
			row.TitleID = snapshotString(snapshot, "title_id", row.TitleID)
			row.TitleEN = snapshotString(snapshot, "title_en", row.TitleEN)
			row.SummaryID = snapshotString(snapshot, "summary_id", row.SummaryID)
			row.SummaryEN = snapshotString(snapshot, "summary_en", row.SummaryEN)
			row.Title, row.Summary = row.TitleID, row.SummaryID
			row.DraftDocument, row.Status, row.UpdatedBy, row.UpdatedAt = cloneLearningDocument(document), "draft", actor, time.Now().UTC()
			r.store.LearningRevisions = append(r.store.LearningRevisions, model.LearningRevision{ID: "learnrev_" + uuid.NewString(), ItemID: itemID, Revision: row.DraftRevision, Document: learningRevisionDocument(row.DraftDocument, row.Slug, row.Kind, row.TitleID, row.TitleEN, row.SummaryID, row.SummaryEN), Kind: "rollback", CreatedBy: actor, CreatedAt: row.UpdatedAt})
			return *row, nil
		}
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	if _, err := r.db.LearningItem.Get(ctx, itemID); ent.IsNotFound(err) {
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	} else if err != nil {
		return model.AdminLearningItem{}, err
	}
	source, err := r.db.LearningRevision.Query().Where(learningrevision.IDEQ(revisionID), learningrevision.ItemIDEQ(itemID)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.AdminLearningItem{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	current, err := r.db.LearningItem.Get(ctx, itemID)
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	document, snapshot := unpackLearningRevisionDocument(source.DocumentJSON)
	nextRevision := current.DraftRevision + 1
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.AdminLearningItem{}, err
	}
	slug := snapshotString(snapshot, "slug", current.Slug)
	kind := snapshotString(snapshot, "kind", current.Kind.String())
	titleID := snapshotString(snapshot, "title_id", current.TitleID)
	titleEN := snapshotString(snapshot, "title_en", current.TitleEn)
	summaryID := snapshotString(snapshot, "summary_id", current.SummaryID)
	summaryEN := snapshotString(snapshot, "summary_en", current.SummaryEn)
	if _, err = tx.LearningItem.UpdateOneID(itemID).SetSlug(slug).SetKind(learningitem.Kind(kind)).SetTitleID(titleID).SetTitleEn(titleEN).SetSummaryID(summaryID).SetSummaryEn(summaryEN).SetDocumentJSON(document).SetDraftRevision(nextRevision).SetStatus(learningitem.StatusDraft).SetUpdatedBy(actor).ClearArchivedAt().Save(ctx); err == nil {
		_, err = tx.LearningRevision.Create().SetID("learnrev_" + uuid.NewString()).SetItemID(itemID).SetRevision(nextRevision).SetDocumentJSON(learningRevisionDocument(document, slug, kind, titleID, titleEN, summaryID, summaryEN)).SetKind(learningrevision.KindRollback).SetCreatedBy(actor).Save(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		return model.AdminLearningItem{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.AdminLearningItem{}, err
	}
	r.RefreshStore(ctx)
	return r.GetAdminLearningItem(ctx, itemID)
}

func (r *Repository) GetLearningHubTaxonomy(ctx context.Context) (model.LearningHubTaxonomy, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		clusters := snapshot.AdminLearningClusters
		programs := snapshot.AdminAcademicPrograms
		if len(clusters) == 0 {
			for _, cluster := range snapshot.LearningClusters {
				clusters = append(clusters, model.AdminLearningCluster{LearningCluster: cluster, TitleID: cluster.Title, DescriptionID: cluster.Description, Active: true})
			}
		}
		if len(programs) == 0 {
			for _, program := range snapshot.AcademicPrograms {
				programs = append(programs, model.AdminAcademicProgram{AcademicProgram: program, Active: true})
			}
		}
		result := model.LearningHubTaxonomy{Clusters: make([]model.AdminLearningCluster, 0, len(clusters)), Programs: make([]model.AdminAcademicProgram, 0, len(programs))}
		if len(snapshot.Institutions) > 0 {
			result.Institution = snapshot.Institutions[0]
		}
		result.Clusters = append(result.Clusters, clusters...)
		result.Programs = append(result.Programs, programs...)
		return result, nil
	}
	inst, err := r.db.Institution.Query().Where(institution.SlugEQ("uty")).Only(ctx)
	if ent.IsNotFound(err) {
		return model.LearningHubTaxonomy{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.LearningHubTaxonomy{}, err
	}
	clusters, err := r.db.LearningCluster.Query().Order(ent.Asc(learningcluster.FieldSortOrder)).All(ctx)
	if err != nil {
		return model.LearningHubTaxonomy{}, err
	}
	programs, err := r.db.AcademicProgram.Query().Order(ent.Asc(academicprogram.FieldSortOrder)).All(ctx)
	if err != nil {
		return model.LearningHubTaxonomy{}, err
	}
	result := model.LearningHubTaxonomy{Institution: model.Institution{ID: inst.ID, Slug: inst.Slug, Name: inst.Name, Status: inst.Status.String(), CreatedAt: inst.CreatedAt, UpdatedAt: inst.UpdatedAt}, Clusters: make([]model.AdminLearningCluster, 0, len(clusters)), Programs: make([]model.AdminAcademicProgram, 0, len(programs))}
	for _, row := range clusters {
		result.Clusters = append(result.Clusters, adminLearningClusterFromEnt(row))
	}
	for _, row := range programs {
		result.Programs = append(result.Programs, model.AdminAcademicProgram{AcademicProgram: model.AcademicProgram{ID: row.ID, InstitutionID: row.InstitutionID, Slug: row.Slug, Name: row.Name, Degree: row.Degree, PrimaryClusterSlug: row.PrimaryClusterSlug, SortOrder: row.SortOrder}, Active: row.Active})
	}
	return result, nil
}

func (r *Repository) CreateLearningCluster(ctx context.Context, input model.LearningClusterInput) (model.AdminLearningCluster, error) {
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		ensureMemoryTaxonomy(r.store)
		for _, row := range r.store.AdminLearningClusters {
			if row.Slug == input.Slug {
				return model.AdminLearningCluster{}, ErrLearningAdminConflict
			}
		}
		activeValue := true
		if input.Active != nil {
			activeValue = *input.Active
		}
		row := model.AdminLearningCluster{LearningCluster: model.LearningCluster{ID: "cluster_" + uuid.NewString(), Slug: input.Slug, Title: input.TitleID, Description: input.DescriptionID, SortOrder: input.SortOrder}, TitleID: input.TitleID, TitleEN: input.TitleEN, DescriptionID: input.DescriptionID, DescriptionEN: input.DescriptionEN, Active: activeValue}
		r.store.AdminLearningClusters = append(r.store.AdminLearningClusters, row)
		if activeValue {
			r.store.LearningClusters = append(r.store.LearningClusters, row.LearningCluster)
		}
		return row, nil
	}
	row, err := r.db.LearningCluster.Create().SetID("cluster_" + uuid.NewString()).SetSlug(input.Slug).SetTitleID(input.TitleID).SetTitleEn(input.TitleEN).SetDescriptionID(input.DescriptionID).SetDescriptionEn(input.DescriptionEN).SetSortOrder(input.SortOrder).SetActive(active).Save(ctx)
	if ent.IsConstraintError(err) {
		return model.AdminLearningCluster{}, ErrLearningAdminConflict
	}
	if err != nil {
		return model.AdminLearningCluster{}, err
	}
	r.RefreshStore(ctx)
	return adminLearningClusterFromEnt(row), nil
}

func (r *Repository) UpdateLearningCluster(ctx context.Context, id string, input model.LearningClusterInput) (model.AdminLearningCluster, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		ensureMemoryTaxonomy(r.store)
		index := -1
		for rowIndex := range r.store.AdminLearningClusters {
			if r.store.AdminLearningClusters[rowIndex].ID == id {
				index = rowIndex
				break
			}
		}
		if index < 0 {
			return model.AdminLearningCluster{}, ErrLearningAdminNotFound
		}
		current := r.store.AdminLearningClusters[index]
		if current.Slug != input.Slug {
			for _, row := range r.store.AdminLearningClusters {
				if row.ID != id && row.Slug == input.Slug {
					return model.AdminLearningCluster{}, ErrLearningAdminConflict
				}
			}
			if inUse, useErr := r.learningClusterInUseMemory(current.Slug); useErr != nil || inUse {
				if useErr != nil {
					return model.AdminLearningCluster{}, useErr
				}
				return model.AdminLearningCluster{}, ErrLearningTaxonomyConflict
			}
		}
		row := &r.store.AdminLearningClusters[index]
		row.Slug, row.TitleID, row.TitleEN = input.Slug, input.TitleID, input.TitleEN
		row.DescriptionID, row.DescriptionEN, row.SortOrder = input.DescriptionID, input.DescriptionEN, input.SortOrder
		row.Title, row.Description = input.TitleID, input.DescriptionID
		if input.Active != nil {
			row.Active = *input.Active
		}
		r.syncLearningClusterMemory(*row)
		return *row, nil
	}
	current, err := r.db.LearningCluster.Get(ctx, id)
	if ent.IsNotFound(err) {
		return model.AdminLearningCluster{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningCluster{}, err
	}
	if current.Slug != input.Slug {
		inUse, useErr := r.LearningClusterInUse(ctx, current.Slug)
		if useErr != nil {
			return model.AdminLearningCluster{}, useErr
		}
		if inUse {
			return model.AdminLearningCluster{}, ErrLearningTaxonomyConflict
		}
	}
	update := r.db.LearningCluster.UpdateOneID(id).SetSlug(input.Slug).SetTitleID(input.TitleID).SetTitleEn(input.TitleEN).SetDescriptionID(input.DescriptionID).SetDescriptionEn(input.DescriptionEN).SetSortOrder(input.SortOrder)
	if input.Active != nil {
		update.SetActive(*input.Active)
	}
	row, err := update.Save(ctx)
	if ent.IsConstraintError(err) {
		return model.AdminLearningCluster{}, ErrLearningAdminConflict
	}
	if ent.IsNotFound(err) {
		return model.AdminLearningCluster{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningCluster{}, err
	}
	r.RefreshStore(ctx)
	return adminLearningClusterFromEnt(row), nil
}

func (r *Repository) DeactivateLearningCluster(ctx context.Context, id string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		ensureMemoryTaxonomy(r.store)
		for index := range r.store.AdminLearningClusters {
			if r.store.AdminLearningClusters[index].ID != id {
				continue
			}
			if inUse, useErr := r.learningClusterInUseMemory(r.store.AdminLearningClusters[index].Slug); useErr != nil {
				return useErr
			} else if inUse {
				return ErrLearningTaxonomyConflict
			}
			r.store.AdminLearningClusters[index].Active = false
			r.syncLearningClusterMemory(r.store.AdminLearningClusters[index])
			return nil
		}
		return ErrLearningAdminNotFound
	}
	current, err := r.db.LearningCluster.Get(ctx, id)
	if ent.IsNotFound(err) {
		return ErrLearningAdminNotFound
	}
	if err != nil {
		return err
	}
	inUse, err := r.LearningClusterInUse(ctx, current.Slug)
	if err != nil {
		return err
	}
	if inUse {
		return ErrLearningTaxonomyConflict
	}
	if _, err := r.db.LearningCluster.UpdateOneID(id).SetActive(false).Save(ctx); ent.IsNotFound(err) {
		return ErrLearningAdminNotFound
	} else if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) CreateLearningProgram(ctx context.Context, input model.AcademicProgramInput) (model.AdminAcademicProgram, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		ensureMemoryTaxonomy(r.store)
		for _, row := range r.store.AdminAcademicPrograms {
			if row.Slug == input.Slug {
				return model.AdminAcademicProgram{}, ErrLearningAdminConflict
			}
		}
		institutionID := ""
		for _, institution := range r.store.Institutions {
			if institution.Slug == "uty" {
				institutionID = institution.ID
				break
			}
		}
		if institutionID == "" {
			return model.AdminAcademicProgram{}, ErrLearningAdminNotFound
		}
		activeValue := true
		if input.Active != nil {
			activeValue = *input.Active
		}
		row := model.AdminAcademicProgram{AcademicProgram: model.AcademicProgram{ID: "program_" + uuid.NewString(), InstitutionID: institutionID, Slug: input.Slug, Name: input.Name, Degree: input.Degree, PrimaryClusterSlug: input.PrimaryClusterSlug, SortOrder: input.SortOrder}, Active: activeValue}
		r.store.AdminAcademicPrograms = append(r.store.AdminAcademicPrograms, row)
		if activeValue {
			r.store.AcademicPrograms = append(r.store.AcademicPrograms, row.AcademicProgram)
		}
		return row, nil
	}
	inst, err := r.db.Institution.Query().Where(institution.SlugEQ("uty")).Only(ctx)
	if ent.IsNotFound(err) {
		return model.AdminAcademicProgram{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminAcademicProgram{}, err
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	row, err := r.db.AcademicProgram.Create().SetID("program_" + uuid.NewString()).SetInstitutionID(inst.ID).SetSlug(input.Slug).SetName(input.Name).SetDegree(input.Degree).SetPrimaryClusterSlug(input.PrimaryClusterSlug).SetSortOrder(input.SortOrder).SetActive(active).Save(ctx)
	if ent.IsConstraintError(err) {
		return model.AdminAcademicProgram{}, ErrLearningAdminConflict
	}
	if err != nil {
		return model.AdminAcademicProgram{}, err
	}
	r.RefreshStore(ctx)
	return model.AdminAcademicProgram{AcademicProgram: model.AcademicProgram{ID: row.ID, InstitutionID: row.InstitutionID, Slug: row.Slug, Name: row.Name, Degree: row.Degree, PrimaryClusterSlug: row.PrimaryClusterSlug, SortOrder: row.SortOrder}, Active: row.Active}, nil
}

func (r *Repository) UpdateLearningProgram(ctx context.Context, id string, input model.AcademicProgramInput) (model.AdminAcademicProgram, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		ensureMemoryTaxonomy(r.store)
		index := -1
		for rowIndex := range r.store.AdminAcademicPrograms {
			if r.store.AdminAcademicPrograms[rowIndex].ID == id {
				index = rowIndex
				break
			}
		}
		if index < 0 {
			return model.AdminAcademicProgram{}, ErrLearningAdminNotFound
		}
		for _, row := range r.store.AdminAcademicPrograms {
			if row.ID != id && row.Slug == input.Slug {
				return model.AdminAcademicProgram{}, ErrLearningAdminConflict
			}
		}
		row := &r.store.AdminAcademicPrograms[index]
		row.Slug, row.Name, row.Degree = input.Slug, input.Name, input.Degree
		row.PrimaryClusterSlug, row.SortOrder = input.PrimaryClusterSlug, input.SortOrder
		if input.Active != nil {
			row.Active = *input.Active
		}
		r.syncLearningProgramMemory(*row)
		return *row, nil
	}
	update := r.db.AcademicProgram.UpdateOneID(id).SetSlug(input.Slug).SetName(input.Name).SetDegree(input.Degree).SetPrimaryClusterSlug(input.PrimaryClusterSlug).SetSortOrder(input.SortOrder)
	if input.Active != nil {
		update.SetActive(*input.Active)
	}
	row, err := update.Save(ctx)
	if ent.IsConstraintError(err) {
		return model.AdminAcademicProgram{}, ErrLearningAdminConflict
	}
	if ent.IsNotFound(err) {
		return model.AdminAcademicProgram{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminAcademicProgram{}, err
	}
	r.RefreshStore(ctx)
	return model.AdminAcademicProgram{AcademicProgram: model.AcademicProgram{ID: row.ID, InstitutionID: row.InstitutionID, Slug: row.Slug, Name: row.Name, Degree: row.Degree, PrimaryClusterSlug: row.PrimaryClusterSlug, SortOrder: row.SortOrder}, Active: row.Active}, nil
}

func (r *Repository) DeactivateLearningProgram(ctx context.Context, id string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		ensureMemoryTaxonomy(r.store)
		for index := range r.store.AdminAcademicPrograms {
			if r.store.AdminAcademicPrograms[index].ID != id {
				continue
			}
			r.store.AdminAcademicPrograms[index].Active = false
			r.syncLearningProgramMemory(r.store.AdminAcademicPrograms[index])
			return nil
		}
		return ErrLearningAdminNotFound
	}
	if _, err := r.db.AcademicProgram.UpdateOneID(id).SetActive(false).Save(ctx); ent.IsNotFound(err) {
		return ErrLearningAdminNotFound
	} else if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) LearningClusterInUse(ctx context.Context, slug string) (bool, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		return r.learningClusterInUseMemory(slug)
	}
	programs, err := r.db.AcademicProgram.Query().Where(academicprogram.PrimaryClusterSlugEQ(slug), academicprogram.ActiveEQ(true)).Count(ctx)
	if err != nil {
		return false, err
	}
	if programs > 0 {
		return true, nil
	}
	return false, nil
}

func ensureMemoryTaxonomy(st *store.Store) {
	if len(st.AdminLearningClusters) == 0 {
		for _, cluster := range st.LearningClusters {
			st.AdminLearningClusters = append(st.AdminLearningClusters, model.AdminLearningCluster{LearningCluster: cluster, TitleID: cluster.Title, DescriptionID: cluster.Description, Active: true})
		}
	}
	if len(st.AdminAcademicPrograms) == 0 {
		for _, program := range st.AcademicPrograms {
			st.AdminAcademicPrograms = append(st.AdminAcademicPrograms, model.AdminAcademicProgram{AcademicProgram: program, Active: true})
		}
	}
}

// The helpers below assume the caller already holds the store lock.
func (r *Repository) learningClusterInUseMemory(slug string) (bool, error) {
	if len(r.store.AdminAcademicPrograms) > 0 {
		for _, program := range r.store.AdminAcademicPrograms {
			if program.Active && program.PrimaryClusterSlug == slug {
				return true, nil
			}
		}
		return false, nil
	}
	for _, program := range r.store.AcademicPrograms {
		if program.PrimaryClusterSlug == slug {
			return true, nil
		}
	}
	return false, nil
}

func (r *Repository) syncLearningClusterMemory(row model.AdminLearningCluster) {
	clusters := r.store.LearningClusters[:0]
	for _, cluster := range r.store.LearningClusters {
		if cluster.ID != row.ID {
			clusters = append(clusters, cluster)
		}
	}
	if row.Active {
		clusters = append(clusters, row.LearningCluster)
	}
	r.store.LearningClusters = clusters
}

func (r *Repository) syncLearningProgramMemory(row model.AdminAcademicProgram) {
	programs := r.store.AcademicPrograms[:0]
	for _, program := range r.store.AcademicPrograms {
		if program.ID != row.ID {
			programs = append(programs, program)
		}
	}
	if row.Active {
		programs = append(programs, row.AcademicProgram)
	}
	r.store.AcademicPrograms = programs
}

func (r *Repository) LearningClusterBySlug(ctx context.Context, slug string) (model.AdminLearningCluster, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		if len(r.store.AdminLearningClusters) > 0 {
			for _, row := range r.store.AdminLearningClusters {
				if row.Slug == slug {
					return row, nil
				}
			}
			return model.AdminLearningCluster{}, ErrLearningAdminNotFound
		}
		for _, row := range r.store.Snapshot().LearningClusters {
			if row.Slug == slug {
				return model.AdminLearningCluster{LearningCluster: row, Active: true}, nil
			}
		}
		return model.AdminLearningCluster{}, ErrLearningAdminNotFound
	}
	row, err := r.db.LearningCluster.Query().Where(learningcluster.SlugEQ(slug)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.AdminLearningCluster{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AdminLearningCluster{}, err
	}
	return adminLearningClusterFromEnt(row), nil
}

func (r *Repository) LearningProgramBySlug(ctx context.Context, slug string) (model.AcademicProgram, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		if len(r.store.AdminAcademicPrograms) > 0 {
			for _, row := range r.store.AdminAcademicPrograms {
				if row.Slug == slug {
					return row.AcademicProgram, nil
				}
			}
			return model.AcademicProgram{}, ErrLearningAdminNotFound
		}
		for _, row := range r.store.Snapshot().AcademicPrograms {
			if row.Slug == slug {
				return row, nil
			}
		}
		return model.AcademicProgram{}, ErrLearningAdminNotFound
	}
	row, err := r.db.AcademicProgram.Query().Where(academicprogram.SlugEQ(slug)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.AcademicProgram{}, ErrLearningAdminNotFound
	}
	if err != nil {
		return model.AcademicProgram{}, err
	}
	return model.AcademicProgram{ID: row.ID, InstitutionID: row.InstitutionID, Slug: row.Slug, Name: row.Name, Degree: row.Degree, PrimaryClusterSlug: row.PrimaryClusterSlug, SortOrder: row.SortOrder}, nil
}
