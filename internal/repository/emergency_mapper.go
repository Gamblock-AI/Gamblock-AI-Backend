package repository

import (
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func emergencyRequestFromEnt(item *ent.EmergencyKeyRequest) model.EmergencyKeyRequest {
	return model.EmergencyKeyRequest{
		ID:               item.ID,
		RequestedBy:      item.RequestedBy,
		DeviceID:         value(item.DeviceID),
		ReviewedBy:       value(item.ReviewedBy),
		ApprovedBy:       value(item.ApprovedBy),
		Status:           item.Status.String(),
		RequestExpiresAt: item.RequestExpiresAt,
		KeyExpiresAt:     item.KeyExpiresAt,
		ReviewedAt:       item.ReviewedAt,
		ApprovedAt:       item.ApprovedAt,
		UsedAt:           item.UsedAt,
		GrantStartsAt:    item.GrantStartsAt,
		GrantExpiresAt:   item.GrantExpiresAt,
		GrantJTI:         value(item.GrantJti),
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
		KeyHash:          value(item.KeyHash),
	}
}
