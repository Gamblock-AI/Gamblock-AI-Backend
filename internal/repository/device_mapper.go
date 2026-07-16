package repository

import (
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func deviceFromEnt(item *ent.Device) model.Device {
	lastSeen := time.Time{}
	if item.LastSeenAt != nil {
		lastSeen = *item.LastSeenAt
	}
	return model.Device{
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
	}
}
