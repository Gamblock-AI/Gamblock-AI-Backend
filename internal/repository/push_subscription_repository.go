package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entpush "github.com/gamblock-ai/gamblock-ai-backend/ent/pushsubscription"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/google/uuid"
)

func (r *Repository) UpsertPushSubscription(ctx context.Context, userID, endpoint, p256dh, authKey string, userAgent *string) (model.PushSubscription, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.PushSubscriptions {
			if r.store.PushSubscriptions[index].Endpoint == endpoint {
				r.store.PushSubscriptions[index].UserID = userID
				r.store.PushSubscriptions[index].P256dh = p256dh
				r.store.PushSubscriptions[index].AuthKey = authKey
				r.store.PushSubscriptions[index].UserAgent = userAgent
				r.store.PushSubscriptions[index].UpdatedAt = time.Now().UTC()
				return r.store.PushSubscriptions[index], nil
			}
		}
		sub := model.PushSubscription{
			ID:        "psu_" + uuid.NewString(),
			UserID:    userID,
			Endpoint:  endpoint,
			P256dh:    p256dh,
			AuthKey:   authKey,
			UserAgent: userAgent,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		r.store.PushSubscriptions = append(r.store.PushSubscriptions, sub)
		return sub, nil
	}
	existing, err := r.db.PushSubscription.Query().Where(entpush.EndpointEQ(endpoint)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return model.PushSubscription{}, err
	}
	var row *ent.PushSubscription
	if err == nil {
		row, err = r.db.PushSubscription.UpdateOneID(existing.ID).
			SetUserID(userID).
			SetP256dh(p256dh).
			SetAuthKey(authKey).
			SetNillableUserAgent(userAgent).
			Save(ctx)
	} else {
		row, err = r.db.PushSubscription.Create().
			SetID("psu_" + uuid.NewString()).
			SetUserID(userID).
			SetEndpoint(endpoint).
			SetP256dh(p256dh).
			SetAuthKey(authKey).
			SetNillableUserAgent(userAgent).
			Save(ctx)
	}
	if err != nil {
		return model.PushSubscription{}, err
	}
	r.RefreshStore(ctx)
	return pushSubscriptionFromEnt(row), nil
}

func (r *Repository) DeletePushSubscription(ctx context.Context, userID, endpoint string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.PushSubscriptions {
			if r.store.PushSubscriptions[index].Endpoint == endpoint &&
				r.store.PushSubscriptions[index].UserID == userID {
				r.store.PushSubscriptions = append(r.store.PushSubscriptions[:index], r.store.PushSubscriptions[index+1:]...)
				return nil
			}
		}
		return nil
	}
	if _, err := r.db.PushSubscription.Delete().
		Where(entpush.EndpointEQ(endpoint), entpush.UserIDEQ(userID)).
		Exec(ctx); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

// RemovePushSubscriptionByID deletes a subscription the pusher has declared
// invalid (404/410), regardless of the current user, to keep delivery honest.
func (r *Repository) RemovePushSubscriptionByID(ctx context.Context, subscriptionID string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.PushSubscriptions {
			if r.store.PushSubscriptions[index].ID == subscriptionID {
				r.store.PushSubscriptions = append(r.store.PushSubscriptions[:index], r.store.PushSubscriptions[index+1:]...)
				return nil
			}
		}
		return nil
	}
	if _, err := r.db.PushSubscription.Delete().Where(entpush.IDEQ(subscriptionID)).Exec(ctx); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) PushSubscriptionsForUser(ctx context.Context, userID string) ([]model.PushSubscription, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		var out []model.PushSubscription
		for _, sub := range r.store.PushSubscriptions {
			if sub.UserID == userID {
				out = append(out, sub)
			}
		}
		return out, nil
	}
	rows, err := r.db.PushSubscription.Query().Where(entpush.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.PushSubscription, 0, len(rows))
	for _, row := range rows {
		out = append(out, pushSubscriptionFromEnt(row))
	}
	return out, nil
}

func pushSubscriptionFromEnt(row *ent.PushSubscription) model.PushSubscription {
	return model.PushSubscription{
		ID:        row.ID,
		UserID:    row.UserID,
		Endpoint:  row.Endpoint,
		P256dh:    row.P256dh,
		AuthKey:   row.AuthKey,
		UserAgent: row.UserAgent,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
