package repository

import (
	"context"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/auditlog"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationrevision"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/sitesociallink"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) ListSiteSocialLinks(ctx context.Context, publicOnly bool) ([]model.SiteSocialLink, error) {
	if r.db == nil {
		items := r.store.Snapshot().SiteSocialLinks
		filtered := make([]model.SiteSocialLink, 0, len(items))
		for _, item := range items {
			if publicOnly && (!item.Enabled || item.URL == nil || *item.URL == "") {
				continue
			}
			filtered = append(filtered, item)
		}
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].SortOrder < filtered[j].SortOrder })
		return filtered, nil
	}
	query := r.db.SiteSocialLink.Query()
	if publicOnly {
		query = query.Where(sitesociallink.EnabledEQ(true), sitesociallink.URLNotNil())
	}
	rows, err := query.Order(ent.Asc(sitesociallink.FieldSortOrder)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.SiteSocialLink, 0, len(rows))
	for _, row := range rows {
		items = append(items, socialLinkFromEnt(row))
	}
	return items, nil
}

func (r *Repository) ReplaceSiteSocialLinks(ctx context.Context, actor string, items []model.SiteSocialLink) error {
	now := time.Now().UTC()
	if r.db == nil {
		for index := range items {
			items[index].ID = "social_" + items[index].Platform
			items[index].UpdatedBy = actor
			items[index].CreatedAt = now
			items[index].UpdatedAt = now
		}
		r.store.Lock()
		r.store.SiteSocialLinks = append([]store.SiteSocialLink(nil), items...)
		r.store.Unlock()
		return nil
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err = tx.SiteSocialLink.Delete().Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, item := range items {
		creator := tx.SiteSocialLink.Create().SetID("social_" + item.Platform).
			SetPlatform(sitesociallink.Platform(item.Platform)).SetLabel(item.Label).
			SetEnabled(item.Enabled).SetSortOrder(item.SortOrder).SetUpdatedBy(actor)
		if item.URL != nil {
			creator.SetURL(*item.URL)
		}
		if _, err = creator.Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func socialLinkFromEnt(row *ent.SiteSocialLink) model.SiteSocialLink {
	return model.SiteSocialLink{ID: row.ID, Platform: row.Platform.String(), Label: row.Label, URL: row.URL,
		Enabled: row.Enabled, SortOrder: row.SortOrder, UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func (r *Repository) SaveAuditEvent(ctx context.Context, event model.AuditEvent) error {
	if r.db == nil {
		r.store.Lock()
		r.store.AuditEvents = append(r.store.AuditEvents, event)
		r.store.Unlock()
		return nil
	}
	_, err := r.db.AuditLog.Create().SetID(event.ID).SetActorID(event.ActorID).SetActorEmail(event.Actor).
		SetAction(event.Action).SetTargetType(event.TargetType).SetTargetID(event.Target).
		SetReason(event.Reason).SetMetadataJSON(event.Metadata).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) ListAuditEvents(ctx context.Context, limit int) ([]model.AuditEvent, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	if r.db == nil {
		items := append([]model.AuditEvent(nil), r.store.Snapshot().AuditEvents...)
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
		if len(items) > limit {
			items = items[:limit]
		}
		return items, nil
	}
	rows, err := r.db.AuditLog.Query().Order(ent.Desc(auditlog.FieldCreatedAt)).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.AuditEvent, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.AuditEvent{ID: row.ID, ActorID: row.ActorID, Actor: row.ActorEmail,
			Action: row.Action, TargetType: row.TargetType, Target: row.TargetID, Reason: row.Reason,
			Metadata: row.MetadataJSON, CreatedAt: row.CreatedAt})
	}
	return items, nil
}

func (r *Repository) PurgeAuditEventsBefore(ctx context.Context, cutoff time.Time) error {
	if r.db == nil {
		r.store.Lock()
		kept := r.store.AuditEvents[:0]
		for _, item := range r.store.AuditEvents {
			if !item.CreatedAt.Before(cutoff) {
				kept = append(kept, item)
			}
		}
		r.store.AuditEvents = kept
		r.store.Unlock()
		return nil
	}
	_, err := r.db.AuditLog.Delete().Where(auditlog.CreatedAtLT(cutoff)).Exec(ctx)
	return err
}

func (r *Repository) SaveEducationRevision(ctx context.Context, item model.EducationRevision) error {
	if r.db == nil {
		r.store.Lock()
		for _, existing := range r.store.EducationRevisions {
			if existing.ModuleID == item.ModuleID && existing.Revision == item.Revision && existing.Kind == item.Kind {
				r.store.Unlock()
				return nil
			}
		}
		r.store.EducationRevisions = append(r.store.EducationRevisions, item)
		r.store.Unlock()
		return nil
	}
	_, err := r.db.EducationRevision.Create().SetID(item.ID).SetModuleID(item.ModuleID).SetRevision(item.Revision).
		SetDocumentJSON(item.Document).SetSlug(item.Slug).SetKind(educationrevision.Kind(item.Kind)).SetCreatedBy(item.CreatedBy).Save(ctx)
	if err != nil && !ent.IsConstraintError(err) {
		return err
	}
	return nil
}

func (r *Repository) ListEducationRevisions(ctx context.Context, moduleID string) ([]model.EducationRevision, error) {
	if r.db == nil {
		items := make([]model.EducationRevision, 0)
		for _, item := range r.store.Snapshot().EducationRevisions {
			if item.ModuleID == moduleID {
				items = append(items, item)
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
		return items, nil
	}
	rows, err := r.db.EducationRevision.Query().Where(educationrevision.ModuleIDEQ(moduleID)).Order(ent.Desc(educationrevision.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.EducationRevision, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.EducationRevision{ID: row.ID, ModuleID: row.ModuleID, Revision: row.Revision,
			Document: row.DocumentJSON, Slug: row.Slug, Kind: row.Kind.String(), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt})
	}
	return items, nil
}

func (r *Repository) EducationRevisionByID(ctx context.Context, moduleID, revisionID string) (model.EducationRevision, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().EducationRevisions {
			if item.ModuleID == moduleID && item.ID == revisionID {
				return item, nil
			}
		}
		return model.EducationRevision{}, ErrEducationNotFound
	}
	row, err := r.db.EducationRevision.Query().Where(educationrevision.IDEQ(revisionID), educationrevision.ModuleIDEQ(moduleID)).Only(ctx)
	if err != nil {
		return model.EducationRevision{}, ErrEducationNotFound
	}
	return model.EducationRevision{ID: row.ID, ModuleID: row.ModuleID, Revision: row.Revision,
		Document: row.DocumentJSON, Slug: row.Slug, Kind: row.Kind.String(), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}, nil
}
