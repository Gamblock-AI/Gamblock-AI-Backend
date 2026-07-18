package db

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func loadIdentityStore(ctx context.Context, client *ent.Client, out *store.Store, users []*ent.User) error {
	for _, item := range users {
		out.Users = append(out.Users, store.User{
			ID:               item.ID,
			Email:            item.Email,
			DisplayName:      item.DisplayName,
			Role:             item.Role.String(),
			ExperiencePoints: item.ExperiencePoints,
			PasswordHash:     value(item.PasswordHash),
			GoogleSubject:    value(item.GoogleSubject),
			DisabledAt:       item.DisabledAt,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}

	devices, err := client.Device.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range devices {
		lastSeen := time.Time{}
		if item.LastSeenAt != nil {
			lastSeen = *item.LastSeenAt
		}
		out.Devices = append(out.Devices, store.Device{
			ID:               item.ID,
			UserID:           item.UserID,
			ClientInstanceID: value(item.ClientInstanceID),
			Platform:         item.Platform.String(),
			Label:            item.Label,
			AppVersion:       item.AppVersion,
			OSVersion:        item.OsVersion,
			ModelVersion:     value(item.ModelVersion),
			RulesetVersion:   value(item.RulesetVersion),
			ProtectionStatus: item.ProtectionStatus.String(),
			LastSeenAt:       lastSeen,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}

	partners, err := client.PartnerLink.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range partners {
		contact := item.PartnerEmail
		if phone := value(item.PartnerPhone); phone != "" {
			contact += " | " + phone
		}
		out.Partners = append(out.Partners, store.Partner{
			ID:              item.ID,
			UserID:          item.UserID,
			PartnerUserID:   value(item.PartnerUserID),
			InviteTokenHash: value(item.InviteTokenHash),
			Name:            item.PartnerEmail,
			Contact:         contact,
			Status:          item.Status.String(),
			PartnerEmail:    item.PartnerEmail,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}

	approvals, err := client.ApprovalRequest.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range approvals {
		duration := valueInt(item.RequestedDurationMinutes)
		out.Approvals = append(out.Approvals, store.ApprovalRequest{
			ID:                       item.ID,
			UserID:                   item.UserID,
			DeviceID:                 value(item.DeviceID),
			PartnerLinkID:            item.PartnerLinkID,
			QuickTokenHash:           value(item.QuickTokenHash),
			Action:                   item.Action.String(),
			ActionLabel:              humanApprovalAction(item.Action.String(), duration),
			ExpiresIn:                humanExpiry(item.ExpiresAt),
			Status:                   item.Status.String(),
			StatusLabel:              humanApprovalStatus(item.Status.String()),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: duration,
			ResolvedAt:               item.ResolvedAt,
			AppliedAt:                item.AppliedAt,
			GrantExpiresAt:           item.GrantExpiresAt,
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
			ExpiresAt:                item.ExpiresAt,
		})
	}
	return nil
}
