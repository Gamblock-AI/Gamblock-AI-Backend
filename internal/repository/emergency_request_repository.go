package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/emergencykeyrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateEmergencyKeyRequest(ctx context.Context, request model.EmergencyKeyRequest) (model.EmergencyKeyRequest, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.EmergencyKeyRequests = append(r.store.EmergencyKeyRequests, request)
		return request, nil
	}
	item, err := r.db.EmergencyKeyRequest.Create().
		SetID(request.ID).
		SetRequestedBy(request.RequestedBy).
		SetDeviceID(request.DeviceID).
		SetRequestExpiresAt(request.RequestExpiresAt).
		Save(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	return emergencyRequestFromEnt(item), nil
}

func (r *Repository) GetCurrentEmergencyKeyRequest(ctx context.Context, userID, deviceID string, now time.Time) (model.EmergencyKeyRequest, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		var current *model.EmergencyKeyRequest
		for index := range r.store.EmergencyKeyRequests {
			item := &r.store.EmergencyKeyRequests[index]
			if item.RequestedBy != userID || item.DeviceID != deviceID {
				continue
			}
			expireEmergencyRequest(item, now, true)
			if current == nil || item.CreatedAt.After(current.CreatedAt) {
				copy := *item
				current = &copy
			}
		}
		if current == nil {
			return model.EmergencyKeyRequest{}, fmt.Errorf("emergency request not found")
		}
		return *current, nil
	}
	expireEmergencyRequests(ctx, r, now, userID, deviceID, true)
	item, err := r.db.EmergencyKeyRequest.Query().Where(
		emergencykeyrequest.RequestedBy(userID),
		emergencykeyrequest.DeviceIDEQ(deviceID),
	).Order(ent.Desc(emergencykeyrequest.FieldCreatedAt)).First(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	return emergencyRequestFromEnt(item), nil
}

func (r *Repository) GetPendingEmergencyKeyRequests(ctx context.Context, now time.Time) ([]model.EmergencyKeyRequest, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		items := make([]model.EmergencyKeyRequest, 0)
		for index := range r.store.EmergencyKeyRequests {
			item := &r.store.EmergencyKeyRequests[index]
			expireEmergencyRequest(item, now, false)
			if item.Status == "pending" || item.Status == "reviewed" {
				items = append(items, *item)
			}
		}
		return items, nil
	}
	expireEmergencyRequests(ctx, r, now, "", "", false)
	rows, err := r.db.EmergencyKeyRequest.Query().Where(
		emergencykeyrequest.StatusIn(emergencykeyrequest.StatusPending, emergencykeyrequest.StatusReviewed),
		emergencykeyrequest.RequestExpiresAtGT(now),
	).Order(emergencykeyrequest.ByCreatedAt()).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.EmergencyKeyRequest, 0, len(rows))
	for _, row := range rows {
		items = append(items, emergencyRequestFromEnt(row))
	}
	return items, nil
}

func expireEmergencyRequest(item *model.EmergencyKeyRequest, now time.Time, expireApproved bool) {
	if (item.Status == "pending" || item.Status == "reviewed") && !now.Before(item.RequestExpiresAt) {
		item.Status = "expired"
		item.UpdatedAt = now
	}
	if expireApproved && item.Status == "approved" && item.KeyExpiresAt != nil && !now.Before(*item.KeyExpiresAt) {
		item.Status = "expired"
		item.UpdatedAt = now
	}
}

func expireEmergencyRequests(ctx context.Context, r *Repository, now time.Time, userID, deviceID string, expireApproved bool) {
	requestUpdate := r.db.EmergencyKeyRequest.Update().Where(
		emergencykeyrequest.StatusIn(emergencykeyrequest.StatusPending, emergencykeyrequest.StatusReviewed),
		emergencykeyrequest.RequestExpiresAtLTE(now),
	)
	var keyUpdate = r.db.EmergencyKeyRequest.Update().Where(
		emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusApproved),
		emergencykeyrequest.KeyExpiresAtLTE(now),
	)
	if userID != "" {
		requestUpdate.Where(emergencykeyrequest.RequestedBy(userID))
		if expireApproved {
			keyUpdate.Where(emergencykeyrequest.RequestedBy(userID))
		}
	}
	if deviceID != "" {
		requestUpdate.Where(emergencykeyrequest.DeviceIDEQ(deviceID))
		if expireApproved {
			keyUpdate.Where(emergencykeyrequest.DeviceIDEQ(deviceID))
		}
	}
	_, _ = requestUpdate.SetStatus(emergencykeyrequest.StatusExpired).Save(ctx)
	if expireApproved {
		_, _ = keyUpdate.SetStatus(emergencykeyrequest.StatusExpired).Save(ctx)
	}
}
