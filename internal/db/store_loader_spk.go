package db

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

// loadSpkStore copies SPK intervention records, blocked-event timestamps, and
// preferences into the in-memory cache. It is best-effort so databases that
// predate these tables can still load; the SPK service falls back to ent.
func loadSpkStore(ctx context.Context, client *ent.Client, out *store.Store) {
	records, err := client.InterventionRecord.Query().All(ctx)
	if err != nil {
		return
	}
	for _, item := range records {
		out.InterventionRecords = append(out.InterventionRecords, store.InterventionRecord{
			ID:                       item.ID,
			UserID:                   item.UserID,
			InterventionKey:          item.InterventionKey,
			ResponseType:             item.ResponseType,
			SupportLevel:             item.SupportLevel.String(),
			EngagementLevel:          item.EngagementLevel.String(),
			ReadinessLevel:           item.ReadinessLevel,
			Status:                   item.Status.String(),
			RecommendedAt:            item.RecommendedAt,
			CompletedAt:              item.CompletedAt,
			EffectivenessStatus:      item.EffectivenessStatus.String(),
			PersonalizedMessage:      value(item.PersonalizedMessage),
			PersonalizedExplanation:  value(item.PersonalizedExplanation),
			LLMUsed:                  item.LlmUsed,
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
		})
	}

	events, err := client.BlockedEvent.Query().All(ctx)
	if err != nil {
		return
	}
	for _, item := range events {
		out.BlockedEvents = append(out.BlockedEvents, store.BlockedEvent{
			ID:         item.ID,
			UserID:     item.UserID,
			DeviceID:   value(item.DeviceID),
			OccurredAt: item.OccurredAt,
			CreatedAt:  item.CreatedAt,
		})
	}

	prefs, err := client.SpkPreference.Query().All(ctx)
	if err != nil {
		return
	}
	for _, item := range prefs {
		out.SpkPreferences = append(out.SpkPreferences, store.SpkPreference{
			ID:                        item.ID,
			UserID:                    item.UserID,
			SpkRecommendationEnabled:  item.SpkRecommendationEnabled,
			SpkUseProtection:          item.SpkUseProtection,
			SpkUseRecovery:            item.SpkUseRecovery,
			SpkUsePersonal:            item.SpkUsePersonal,
			LLMPersonalizationEnabled: item.LlmPersonalizationEnabled,
			CreatedAt:                 item.CreatedAt,
			UpdatedAt:                 item.UpdatedAt,
		})
	}
}
