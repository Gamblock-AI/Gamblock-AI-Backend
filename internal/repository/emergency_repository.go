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
		SetRequestExpiresAt(request.RequestExpiresAt).
		Save(ctx)
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
			if item.Status == "pending" && !now.Before(item.RequestExpiresAt) {
				item.Status = "expired"
				item.UpdatedAt = now
			}
			if item.Status == "pending" {
				items = append(items, *item)
			}
		}
		return items, nil
	}
	_, _ = r.db.EmergencyKeyRequest.Update().
		Where(
			emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusPending),
			emergencykeyrequest.RequestExpiresAtLTE(now),
		).
		SetStatus(emergencykeyrequest.StatusExpired).
		Save(ctx)
	rows, err := r.db.EmergencyKeyRequest.Query().
		Where(
			emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusPending),
			emergencykeyrequest.RequestExpiresAtGT(now),
		).
		Order(emergencykeyrequest.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.EmergencyKeyRequest, 0, len(rows))
	for _, row := range rows {
		items = append(items, emergencyRequestFromEnt(row))
	}
	return items, nil
}

func (r *Repository) ApproveEmergencyKeyRequest(ctx context.Context, id, approverID, keyHash string, now, keyExpiresAt time.Time) (model.EmergencyKeyRequest, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.EmergencyKeyRequests {
			item := &r.store.EmergencyKeyRequests[index]
			if item.ID != id {
				continue
			}
			if item.Status != "pending" || !now.Before(item.RequestExpiresAt) {
				return model.EmergencyKeyRequest{}, fmt.Errorf("request is not pending or has expired")
			}
			if item.RequestedBy == approverID {
				return model.EmergencyKeyRequest{}, fmt.Errorf("a different platform administrator must approve")
			}
			item.ApprovedBy = approverID
			item.ApprovedAt = &now
			item.KeyExpiresAt = &keyExpiresAt
			item.KeyHash = keyHash
			item.Status = "approved"
			item.UpdatedAt = now
			return *item, nil
		}
		return model.EmergencyKeyRequest{}, fmt.Errorf("request not found")
	}
	changed, err := r.db.EmergencyKeyRequest.Update().
		Where(
			emergencykeyrequest.IDEQ(id),
			emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusPending),
			emergencykeyrequest.RequestExpiresAtGT(now),
			emergencykeyrequest.RequestedByNEQ(approverID),
		).
		SetApprovedBy(approverID).
		SetApprovedAt(now).
		SetKeyHash(keyHash).
		SetKeyExpiresAt(keyExpiresAt).
		SetStatus(emergencykeyrequest.StatusApproved).
		Save(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	if changed != 1 {
		return model.EmergencyKeyRequest{}, fmt.Errorf("request requires an unexpired approval from a different platform administrator")
	}
	item, err := r.db.EmergencyKeyRequest.Get(ctx, id)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	return emergencyRequestFromEnt(item), nil
}

func (r *Repository) UseEmergencyKey(ctx context.Context, keyHash string, now time.Time) (model.EmergencyKeyRequest, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.EmergencyKeyRequests {
			item := &r.store.EmergencyKeyRequests[index]
			if item.KeyHash != keyHash {
				continue
			}
			if item.Status != "approved" || item.KeyExpiresAt == nil || !now.Before(*item.KeyExpiresAt) {
				return model.EmergencyKeyRequest{}, fmt.Errorf("emergency key is invalid, used, or expired")
			}
			item.Status = "used"
			item.UpdatedAt = now
			return *item, nil
		}
		return model.EmergencyKeyRequest{}, fmt.Errorf("emergency key not found")
	}
	changed, err := r.db.EmergencyKeyRequest.Update().
		Where(
			emergencykeyrequest.KeyHashEQ(keyHash),
			emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusApproved),
			emergencykeyrequest.KeyExpiresAtGT(now),
		).
		SetStatus(emergencykeyrequest.StatusUsed).
		Save(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	if changed != 1 {
		return model.EmergencyKeyRequest{}, fmt.Errorf("emergency key is invalid, used, or expired")
	}
	item, err := r.db.EmergencyKeyRequest.Query().Where(emergencykeyrequest.KeyHashEQ(keyHash)).Only(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	return emergencyRequestFromEnt(item), nil
}

func emergencyRequestFromEnt(item *ent.EmergencyKeyRequest) model.EmergencyKeyRequest {
	return model.EmergencyKeyRequest{
		ID:               item.ID,
		RequestedBy:      item.RequestedBy,
		ApprovedBy:       value(item.ApprovedBy),
		Status:           item.Status.String(),
		RequestExpiresAt: item.RequestExpiresAt,
		KeyExpiresAt:     item.KeyExpiresAt,
		ApprovedAt:       item.ApprovedAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
		KeyHash:          value(item.KeyHash),
	}
}
