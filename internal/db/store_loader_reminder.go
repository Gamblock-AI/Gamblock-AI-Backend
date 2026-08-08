package db

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

// loadReminderStore copies opt-in reminder and push-subscription rows into the
// in-memory cache. It is best-effort so databases that predate these tables can
// still load; the scheduler and settings endpoints simply fall back to ent.
func loadReminderStore(ctx context.Context, client *ent.Client, out *store.Store) {
	prefs, err := client.ReminderPreference.Query().All(ctx)
	if err != nil {
		return
	}
	for _, item := range prefs {
		out.ReminderPreferences = append(out.ReminderPreferences, store.ReminderPreference{
			ID:          item.ID,
			UserID:      item.UserID,
			Enabled:     item.Enabled,
			LocalTime:   item.LocalTime,
			Timezone:    item.Timezone,
			Locale:      item.Locale,
			LastFiredAt: item.LastFiredAt,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	subscriptions, err := client.PushSubscription.Query().All(ctx)
	if err != nil {
		return
	}
	for _, item := range subscriptions {
		out.PushSubscriptions = append(out.PushSubscriptions, store.PushSubscription{
			ID:        item.ID,
			UserID:    item.UserID,
			Endpoint:  item.Endpoint,
			P256dh:    item.P256dh,
			AuthKey:   item.AuthKey,
			UserAgent: item.UserAgent,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
}
