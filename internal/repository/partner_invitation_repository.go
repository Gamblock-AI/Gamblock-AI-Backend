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

func (r *Repository) AcceptPartnerInvitation(ctx context.Context, linkID, partnerUserID string) error {
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
			partner := &r.store.Partners[index]
			if partner.ID == linkID && partner.Status == "invited" && partner.UserID != partnerUserID && strings.EqualFold(partner.PartnerEmail, userEmail) {
				partner.PartnerUserID = partnerUserID
				partner.Status = "active"
				partner.UpdatedAt = time.Now().UTC()
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
			partner := &r.store.Partners[index]
			if partner.ID == partnerLinkID && (partner.UserID == userID || partner.PartnerUserID == userID) {
				partner.Status = "revoked"
				partner.UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("partner link not found")
	}
	item, err := r.db.PartnerLink.Query().Where(
		partnerlink.IDEQ(partnerLinkID),
		partnerlink.Or(partnerlink.UserID(userID), partnerlink.PartnerUserID(userID)),
	).Only(ctx)
	if err != nil {
		return err
	}
	_, err = item.Update().SetStatus(partnerlink.StatusRevoked).SetRevokedAt(time.Now().UTC()).Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
