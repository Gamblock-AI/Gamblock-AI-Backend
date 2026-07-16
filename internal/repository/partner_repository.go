package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetPartners(ctx context.Context, userID string) (activePartner *model.Partner, items []model.Partner, err error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var active *model.Partner
		var list []model.Partner
		for _, p := range snapshot.Partners {
			if p.UserID != userID && p.PartnerUserID != userID {
				continue
			}
			role := "partner"
			if p.UserID == userID {
				role = "owner"
			}
			list = append(list, model.Partner{ID: p.ID, RelationshipRole: role, Name: p.Name, Contact: p.Contact, Status: p.Status, PartnerEmail: p.PartnerEmail, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt})
			if p.Status == "active" {
				active = &model.Partner{ID: p.ID, RelationshipRole: role, Name: p.Name, Contact: p.Contact, Status: p.Status, PartnerEmail: p.PartnerEmail, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
			}
		}
		return active, list, nil
	}

	rows, err := r.db.PartnerLink.Query().Where(
		partnerlink.Or(partnerlink.UserID(userID), partnerlink.PartnerUserID(userID)),
	).All(ctx)
	if err != nil {
		return nil, nil, err
	}

	for _, item := range rows {
		p := model.Partner{
			ID:               item.ID,
			RelationshipRole: "partner",
			Name:             item.PartnerEmail,
			Contact:          value(item.PartnerPhone),
			Status:           item.Status.String(),
			PartnerEmail:     item.PartnerEmail,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		}
		if item.UserID == userID {
			p.RelationshipRole = "owner"
		}
		items = append(items, p)
		if item.Status == partnerlink.StatusActive {
			activePartner = &p
		}
	}
	return activePartner, items, nil
}

func (r *Repository) CreatePartnerInvitation(ctx context.Context, plID, userID, email string, phone *string, inviteTokenHash string) (model.Partner, error) {
	if r.db == nil {
		now := time.Now().UTC()
		item := model.Partner{ID: plID, UserID: userID, Status: "invited", PartnerEmail: email, InviteTokenHash: inviteTokenHash, CreatedAt: now, UpdatedAt: now}
		r.store.Lock()
		r.store.Partners = append(r.store.Partners, item)
		r.store.Unlock()
		return item, nil
	}
	item, err := r.db.PartnerLink.Create().
		SetID(plID).
		SetUserID(userID).
		SetPartnerEmail(email).
		SetNillablePartnerPhone(phone).
		SetStatus(partnerlink.StatusInvited).
		SetInviteTokenHash(inviteTokenHash).
		Save(ctx)
	if err != nil {
		return model.Partner{}, err
	}
	r.RefreshStore(ctx)
	return model.Partner{
		ID:           item.ID,
		PartnerEmail: item.PartnerEmail,
		Contact:      value(item.PartnerPhone),
		Status:       item.Status.String(),
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}, nil
}

func (r *Repository) GetPartnerLinkByToken(ctx context.Context, inviteTokenHash string) (entID string, err error) {
	oldestValid := time.Now().UTC().Add(-7 * 24 * time.Hour)
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, item := range snapshot.Partners {
			if item.InviteTokenHash == inviteTokenHash && item.Status == "invited" && item.CreatedAt.After(oldestValid) {
				return item.ID, nil
			}
		}
		return "", fmt.Errorf("invitation not found")
	}
	row, err := r.db.PartnerLink.Query().
		Where(partnerlink.InviteTokenHashEQ(inviteTokenHash), partnerlink.StatusEQ(partnerlink.StatusInvited), partnerlink.CreatedAtGT(oldestValid)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (r *Repository) AcceptPartnerInvitation(ctx context.Context, linkID string, partnerUserID string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		userEmail := ""
		for _, user := range r.store.Users {
			if user.ID == partnerUserID {
				userEmail = user.Email
				break
			}
		}
		for index := range r.store.Partners {
			if r.store.Partners[index].ID == linkID && r.store.Partners[index].Status == "invited" &&
				r.store.Partners[index].UserID != partnerUserID && strings.EqualFold(r.store.Partners[index].PartnerEmail, userEmail) {
				r.store.Partners[index].PartnerUserID = partnerUserID
				r.store.Partners[index].Status = "active"
				r.store.Partners[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("invitation not found")
	}
	link, err := r.db.PartnerLink.Get(ctx, linkID)
	if err != nil || link.UserID == partnerUserID || link.Status != partnerlink.StatusInvited {
		return fmt.Errorf("invitation is not valid for this account")
	}
	user, err := r.db.User.Query().Where(entuser.IDEQ(partnerUserID)).Only(ctx)
	if err != nil || !strings.EqualFold(user.Email, link.PartnerEmail) {
		return fmt.Errorf("invitation email does not match this account")
	}
	_, err = link.Update().
		SetPartnerUserID(partnerUserID).
		SetStatus(partnerlink.StatusActive).
		SetAcceptedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) RevokePartner(ctx context.Context, partnerLinkID, userID string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Partners {
			if r.store.Partners[index].ID == partnerLinkID && (r.store.Partners[index].UserID == userID || r.store.Partners[index].PartnerUserID == userID) {
				r.store.Partners[index].Status = "revoked"
				r.store.Partners[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("partner link not found")
	}
	item, err := r.db.PartnerLink.Query().Where(partnerlink.IDEQ(partnerLinkID), partnerlink.Or(partnerlink.UserID(userID), partnerlink.PartnerUserID(userID))).Only(ctx)
	if err != nil {
		return err
	}
	_, err = item.Update().
		SetStatus(partnerlink.StatusRevoked).
		SetRevokedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) IsActivePartnerLinkOwnedBy(ctx context.Context, partnerLinkID, userID string) bool {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, item := range snapshot.Partners {
			if item.ID == partnerLinkID && item.UserID == userID && item.Status == "active" {
				return true
			}
		}
		return false
	}
	exists, err := r.db.PartnerLink.Query().Where(
		partnerlink.IDEQ(partnerLinkID),
		partnerlink.UserID(userID),
		partnerlink.StatusEQ(partnerlink.StatusActive),
	).Exist(ctx)
	return err == nil && exists
}

func (r *Repository) GetActivePartnerPhone(ctx context.Context, partnerLinkID, userID string) string {
	if r.db == nil {
		for _, item := range r.store.Snapshot().Partners {
			if item.ID != partnerLinkID || item.UserID != userID || item.Status != "active" {
				continue
			}
			parts := strings.Split(item.Contact, "|")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[len(parts)-1])
			}
			return ""
		}
		return ""
	}
	item, err := r.db.PartnerLink.Query().Where(
		partnerlink.IDEQ(partnerLinkID),
		partnerlink.UserID(userID),
		partnerlink.StatusEQ(partnerlink.StatusActive),
	).Only(ctx)
	if err != nil {
		return ""
	}
	return value(item.PartnerPhone)
}
