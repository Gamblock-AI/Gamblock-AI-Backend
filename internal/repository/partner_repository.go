package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetPartners(ctx context.Context, userID string) (activePartner *model.Partner, items []model.Partner, err error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var active *model.Partner
		var list []model.Partner
		for _, p := range snapshot.Partners {
			list = append(list, model.Partner{ID: p.ID, Name: p.Name, Contact: p.Contact, Status: p.Status, PartnerEmail: p.PartnerEmail, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt})
			if p.Status == "active" {
				active = &model.Partner{ID: p.ID, Name: p.Name, Contact: p.Contact, Status: p.Status, PartnerEmail: p.PartnerEmail, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
			}
		}
		return active, list, nil
	}

	rows, err := r.db.PartnerLink.Query().Where(partnerlink.UserID(userID)).All(ctx)
	if err != nil {
		return nil, nil, err
	}

	for _, item := range rows {
		p := model.Partner{
			ID:           item.ID,
			Name:         item.PartnerEmail,
			Contact:      value(item.PartnerPhone),
			Status:       item.Status.String(),
			PartnerEmail: item.PartnerEmail,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
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
		return model.Partner{ID: plID, Status: "invited", PartnerEmail: email, CreatedAt: now, UpdatedAt: now}, nil
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
	if r.db == nil {
		return "invite_demo", nil
	}
	row, err := r.db.PartnerLink.Query().
		Where(partnerlink.InviteTokenHashEQ(inviteTokenHash), partnerlink.StatusEQ(partnerlink.StatusInvited)).
		Only(ctx)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (r *Repository) AcceptPartnerInvitation(ctx context.Context, linkID string, partnerUserID string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.PartnerLink.UpdateOneID(linkID).
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

func (r *Repository) RevokePartner(ctx context.Context, partnerLinkID string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.PartnerLink.UpdateOneID(partnerLinkID).
		SetStatus(partnerlink.StatusRevoked).
		SetRevokedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
