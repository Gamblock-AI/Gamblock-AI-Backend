package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/auditlog"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationrevision"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/operatorinvitation"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/sitesociallink"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

var operatorRoles = []entuser.Role{
	entuser.RoleContentAdmin,
	entuser.RoleModelReleaseOperator,
	entuser.RoleSupportOperator,
	entuser.RolePlatformAdmin,
}

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

func (r *Repository) ListOperatorAccounts(ctx context.Context) ([]model.OperatorAccount, error) {
	if r.db == nil {
		items := make([]model.OperatorAccount, 0)
		for _, user := range r.store.Snapshot().Users {
			if !isOperatorRole(user.Role) {
				continue
			}
			items = append(items, operatorAccountFromModel(user))
		}
		return items, nil
	}
	rows, err := r.db.User.Query().Where(entuser.RoleIn(operatorRoles...)).Order(ent.Asc(entuser.FieldEmail)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.OperatorAccount, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.OperatorAccount{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName,
			Role: row.Role.String(), EmailVerifiedAt: row.EmailVerifiedAt, DisabledAt: row.DisabledAt, CreatedAt: row.CreatedAt})
	}
	return items, nil
}

func operatorAccountFromModel(user model.User) model.OperatorAccount {
	return model.OperatorAccount{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: user.Role,
		EmailVerifiedAt: user.EmailVerifiedAt, DisabledAt: user.DisabledAt, CreatedAt: user.CreatedAt}
}

func isOperatorRole(role string) bool {
	return role == "content_admin" || role == "model_release_operator" || role == "support_operator" || role == "platform_admin"
}

func (r *Repository) ListOperatorInvitations(ctx context.Context) ([]model.OperatorInvitation, error) {
	if r.db == nil {
		return append([]model.OperatorInvitation(nil), r.store.Snapshot().OperatorInvitations...), nil
	}
	rows, err := r.db.OperatorInvitation.Query().Order(ent.Desc(operatorinvitation.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.OperatorInvitation, 0, len(rows))
	for _, row := range rows {
		items = append(items, operatorInvitationFromEnt(row))
	}
	return items, nil
}

func (r *Repository) SaveOperatorInvitation(ctx context.Context, item model.OperatorInvitation) error {
	if r.db == nil {
		r.store.Lock()
		for index := range r.store.OperatorInvitations {
			if r.store.OperatorInvitations[index].Email == item.Email && r.store.OperatorInvitations[index].Status == "pending" {
				r.store.OperatorInvitations[index].Status = "revoked"
			}
		}
		r.store.OperatorInvitations = append(r.store.OperatorInvitations, item)
		r.store.Unlock()
		return nil
	}
	_, err := r.db.OperatorInvitation.Update().Where(operatorinvitation.EmailEQ(item.Email), operatorinvitation.StatusEQ(operatorinvitation.StatusPending)).SetStatus(operatorinvitation.StatusRevoked).Save(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.OperatorInvitation.Create().SetID(item.ID).SetEmail(item.Email).
		SetRole(operatorinvitation.Role(item.Role)).SetTokenHash(item.TokenHash).SetInvitedBy(item.InvitedBy).
		SetExpiresAt(item.ExpiresAt).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) OperatorInvitationByToken(ctx context.Context, tokenHash string) (model.OperatorInvitation, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().OperatorInvitations {
			if item.TokenHash == tokenHash {
				return item, nil
			}
		}
		return model.OperatorInvitation{}, fmt.Errorf("operator invitation not found")
	}
	row, err := r.db.OperatorInvitation.Query().Where(operatorinvitation.TokenHashEQ(tokenHash)).Only(ctx)
	if err != nil {
		return model.OperatorInvitation{}, fmt.Errorf("operator invitation not found")
	}
	return operatorInvitationFromEnt(row), nil
}

func (r *Repository) AcceptOperatorInvitation(ctx context.Context, tokenHash, displayName, passwordHash string, now time.Time) (model.User, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		var invite *store.OperatorInvitation
		for index := range r.store.OperatorInvitations {
			candidate := &r.store.OperatorInvitations[index]
			if candidate.TokenHash == tokenHash {
				invite = candidate
				break
			}
		}
		if invite == nil || invite.Status != "pending" || !now.Before(invite.ExpiresAt) {
			return model.User{}, fmt.Errorf("operator invitation is invalid or expired")
		}
		for _, existing := range r.store.Users {
			if existing.Email == invite.Email {
				return model.User{}, fmt.Errorf("operator email is already registered")
			}
		}
		user := model.User{ID: "usr_" + invite.ID, Email: invite.Email, DisplayName: displayName, Role: invite.Role,
			PasswordHash: passwordHash, EmailVerifiedAt: &now, CreatedAt: now, UpdatedAt: now}
		r.store.Users = append(r.store.Users, user)
		invite.Status, invite.AcceptedAt, invite.UpdatedAt = "accepted", &now, now
		return user, nil
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.User{}, err
	}
	invite, err := tx.OperatorInvitation.Query().Where(operatorinvitation.TokenHashEQ(tokenHash), operatorinvitation.StatusEQ(operatorinvitation.StatusPending), operatorinvitation.ExpiresAtGT(now)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return model.User{}, fmt.Errorf("operator invitation is invalid or expired")
	}
	exists, err := tx.User.Query().Where(entuser.EmailEQ(invite.Email)).Exist(ctx)
	if err != nil || exists {
		_ = tx.Rollback()
		return model.User{}, fmt.Errorf("operator email is already registered")
	}
	row, err := tx.User.Create().SetID("usr_" + invite.ID).SetEmail(invite.Email).SetDisplayName(displayName).
		SetPasswordHash(passwordHash).SetRole(entuser.Role(invite.Role.String())).SetEmailVerifiedAt(now).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return model.User{}, err
	}
	if _, err = tx.OperatorInvitation.UpdateOneID(invite.ID).SetStatus(operatorinvitation.StatusAccepted).SetAcceptedAt(now).Save(ctx); err != nil {
		_ = tx.Rollback()
		return model.User{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}

func (r *Repository) RevokeOperatorInvitation(ctx context.Context, id string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.OperatorInvitations {
			if r.store.OperatorInvitations[index].ID == id && r.store.OperatorInvitations[index].Status == "pending" {
				r.store.OperatorInvitations[index].Status = "revoked"
				return nil
			}
		}
		return fmt.Errorf("operator invitation not found")
	}
	count, err := r.db.OperatorInvitation.Update().Where(operatorinvitation.IDEQ(id), operatorinvitation.StatusEQ(operatorinvitation.StatusPending)).SetStatus(operatorinvitation.StatusRevoked).Save(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("operator invitation not found")
	}
	r.RefreshStore(ctx)
	return nil
}

func operatorInvitationFromEnt(row *ent.OperatorInvitation) model.OperatorInvitation {
	return model.OperatorInvitation{ID: row.ID, Email: row.Email, Role: row.Role.String(), TokenHash: row.TokenHash,
		Status: row.Status.String(), InvitedBy: row.InvitedBy, ExpiresAt: row.ExpiresAt, AcceptedAt: row.AcceptedAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func (r *Repository) UpdateOperatorAccount(ctx context.Context, id, role string, disabled bool, now time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			item := &r.store.Users[index]
			if item.ID != id {
				continue
			}
			item.Role = role
			if disabled {
				item.DisabledAt = &now
			} else {
				item.DisabledAt = nil
			}
			item.UpdatedAt = now
			return nil
		}
		return fmt.Errorf("operator not found")
	}
	update := r.db.User.UpdateOneID(id).SetRole(entuser.Role(role))
	if disabled {
		update.SetDisabledAt(now)
	} else {
		update.ClearDisabledAt()
	}
	if _, err := update.Save(ctx); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
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
