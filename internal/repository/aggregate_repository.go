package repository

import (
	"context"
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
