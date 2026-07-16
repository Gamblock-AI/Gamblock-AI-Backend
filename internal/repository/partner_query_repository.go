package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetPartners(ctx context.Context, userID string) (activePartner *model.Partner, items []model.Partner, err error) {
	if r.db == nil {
		for _, partner := range r.store.Snapshot().Partners {
			if partner.UserID != userID && partner.PartnerUserID != userID {
				continue
			}
			item := partnerForUser(partner, userID)
			items = append(items, item)
			if partner.Status == "active" {
				activeCopy := item
				activePartner = &activeCopy
			}
		}
		return activePartner, items, nil
	}
	rows, err := r.db.PartnerLink.Query().Where(
		partnerlink.Or(partnerlink.UserID(userID), partnerlink.PartnerUserID(userID)),
	).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		item := model.Partner{
			ID:               row.ID,
			RelationshipRole: "partner",
			Name:             row.PartnerEmail,
			Contact:          value(row.PartnerPhone),
			Status:           row.Status.String(),
			PartnerEmail:     row.PartnerEmail,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		}
		if row.UserID == userID {
			item.RelationshipRole = "owner"
		}
		items = append(items, item)
		if row.Status == partnerlink.StatusActive {
			activeCopy := item
			activePartner = &activeCopy
		}
	}
	return activePartner, items, nil
}

func (r *Repository) GetPartnerLinkByToken(ctx context.Context, inviteTokenHash string) (string, error) {
	oldestValid := time.Now().UTC().Add(-7 * 24 * time.Hour)
	if r.db == nil {
		for _, partner := range r.store.Snapshot().Partners {
			if partner.InviteTokenHash == inviteTokenHash && partner.Status == "invited" && partner.CreatedAt.After(oldestValid) {
				return partner.ID, nil
			}
		}
		return "", fmt.Errorf("invitation not found")
	}
	row, err := r.db.PartnerLink.Query().Where(
		partnerlink.InviteTokenHashEQ(inviteTokenHash),
		partnerlink.StatusEQ(partnerlink.StatusInvited),
		partnerlink.CreatedAtGT(oldestValid),
	).Only(ctx)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func (r *Repository) IsActivePartnerLinkOwnedBy(ctx context.Context, partnerLinkID, userID string) bool {
	if r.db == nil {
		for _, partner := range r.store.Snapshot().Partners {
			if partner.ID == partnerLinkID && partner.UserID == userID && partner.Status == "active" {
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
		for _, partner := range r.store.Snapshot().Partners {
			if partner.ID != partnerLinkID || partner.UserID != userID || partner.Status != "active" {
				continue
			}
			parts := strings.Split(partner.Contact, "|")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[len(parts)-1])
			}
			return ""
		}
		return ""
	}
	row, err := r.db.PartnerLink.Query().Where(
		partnerlink.IDEQ(partnerLinkID),
		partnerlink.UserID(userID),
		partnerlink.StatusEQ(partnerlink.StatusActive),
	).Only(ctx)
	if err != nil {
		return ""
	}
	return value(row.PartnerPhone)
}

func partnerForUser(partner model.Partner, userID string) model.Partner {
	role := "partner"
	if partner.UserID == userID {
		role = "owner"
	}
	return model.Partner{
		ID:               partner.ID,
		RelationshipRole: role,
		Name:             partner.Name,
		Contact:          partner.Contact,
		Status:           partner.Status,
		PartnerEmail:     partner.PartnerEmail,
		CreatedAt:        partner.CreatedAt,
		UpdatedAt:        partner.UpdatedAt,
	}
}
