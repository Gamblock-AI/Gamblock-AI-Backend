package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) SaveAggregateEvent(ctx context.Context, event model.AggregateEvent) (model.AggregateEvent, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for _, existing := range r.store.AggregateEvents {
			if existing.UserID == event.UserID && existing.IdempotencyKey == event.IdempotencyKey {
				return existing, nil
			}
		}
		event.CreatedAt = time.Now().UTC()
		r.store.AggregateEvents = append(r.store.AggregateEvents, event)
		return event, nil
	}
	existing, err := r.db.AggregateEvent.Query().Where(aggregateevent.IdempotencyKeyEQ(event.IdempotencyKey)).Only(ctx)
	if err == nil {
		return model.AggregateEvent{ID: existing.ID, UserID: existing.UserID, DeviceID: value(existing.DeviceID), IdempotencyKey: existing.IdempotencyKey, EventType: existing.EventType.String(), EventDate: existing.EventDate, Count: existing.Count, CreatedAt: existing.CreatedAt}, nil
	}
	item, err := r.db.AggregateEvent.Create().
		SetID(event.ID).
		SetUserID(event.UserID).
		SetNillableDeviceID(optional(event.DeviceID)).
		SetIdempotencyKey(event.IdempotencyKey).
		SetEventType(aggregateevent.EventType(event.EventType)).
		SetEventDate(event.EventDate).
		SetCount(event.Count).
		Save(ctx)
	if err != nil {
		return model.AggregateEvent{}, err
	}
	return model.AggregateEvent{ID: item.ID, UserID: item.UserID, DeviceID: value(item.DeviceID), IdempotencyKey: item.IdempotencyKey, EventType: item.EventType.String(), EventDate: item.EventDate, Count: item.Count, CreatedAt: item.CreatedAt}, nil
}

func (r *Repository) GetProtectionAnalytics(ctx context.Context, userID, deviceID string, days int, now time.Time) (model.ProtectionAnalytics, error) {
	if !r.IsDeviceOwnedBy(ctx, deviceID, userID) {
		return model.ProtectionAnalytics{}, fmt.Errorf("device does not belong to user")
	}
	start := startOfDay(now.UTC()).AddDate(0, 0, -(days - 1))
	daily := make([]model.ProtectionAnalyticsDay, days)
	byDate := make(map[string]int, days)
	for index := range daily {
		date := start.AddDate(0, 0, index).Format("2006-01-02")
		daily[index].Date = date
		byDate[date] = index
	}
	analytics := model.ProtectionAnalytics{
		DeviceID: deviceID, PeriodDays: days, Daily: daily, DataState: "empty",
	}
	add := func(event model.AggregateEvent) {
		index, ok := byDate[event.EventDate.UTC().Format("2006-01-02")]
		if !ok {
			return
		}
		switch event.EventType {
		case "block_count_sync":
			analytics.Daily[index].Blocked += event.Count
			analytics.Totals.Blocked += event.Count
		case "intervention_shown":
			analytics.Daily[index].Interventions += event.Count
			analytics.Totals.Interventions += event.Count
		case "tamper_detected":
			analytics.Daily[index].TamperEvents += event.Count
			analytics.Totals.TamperEvents += event.Count
		case "permission_revoked":
			analytics.Daily[index].PermissionRevoked += event.Count
			analytics.Totals.PermissionRevoked += event.Count
		}
	}
	if r.db == nil {
		for _, event := range r.store.Snapshot().AggregateEvents {
			if event.UserID == userID && event.DeviceID == deviceID && !event.EventDate.Before(start) {
				add(event)
			}
		}
		analytics.DataState = "local_only"
	} else {
		rows, err := r.db.AggregateEvent.Query().Where(
			aggregateevent.UserID(userID),
			aggregateevent.DeviceIDEQ(deviceID),
			aggregateevent.EventDateGTE(start),
		).All(ctx)
		if err != nil {
			return model.ProtectionAnalytics{}, err
		}
		for _, item := range rows {
			add(model.AggregateEvent{
				EventType: item.EventType.String(), EventDate: item.EventDate, Count: item.Count,
			})
		}
		analytics.DataState = "synced"
	}
	if analytics.Totals == (model.ProtectionAnalyticsTotals{}) {
		analytics.DataState = "empty"
	}
	return analytics, nil
}
