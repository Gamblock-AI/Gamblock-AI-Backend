package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/emergencykeyrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) ReviewEmergencyKeyRequest(ctx context.Context, id, reviewerID string, now time.Time) (model.EmergencyKeyRequest, error) {
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
			item.ReviewedBy = reviewerID
			item.ReviewedAt = &now
			item.Status = "reviewed"
			item.UpdatedAt = now
			return *item, nil
		}
		return model.EmergencyKeyRequest{}, fmt.Errorf("request not found")
	}
	changed, err := r.db.EmergencyKeyRequest.Update().Where(
		emergencykeyrequest.IDEQ(id),
		emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusPending),
		emergencykeyrequest.RequestExpiresAtGT(now),
	).SetReviewedBy(reviewerID).SetReviewedAt(now).SetStatus(emergencykeyrequest.StatusReviewed).Save(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	if changed != 1 {
		return model.EmergencyKeyRequest{}, fmt.Errorf("request is not pending or has expired")
	}
	item, err := r.db.EmergencyKeyRequest.Get(ctx, id)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	return emergencyRequestFromEnt(item), nil
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
			if item.Status != "reviewed" || !now.Before(item.RequestExpiresAt) {
				return model.EmergencyKeyRequest{}, fmt.Errorf("request is not reviewed or has expired")
			}
			if item.ReviewedBy == "" || item.ReviewedBy == approverID {
				return model.EmergencyKeyRequest{}, fmt.Errorf("a second platform administrator must approve")
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
	changed, err := r.db.EmergencyKeyRequest.Update().Where(
		emergencykeyrequest.IDEQ(id),
		emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusReviewed),
		emergencykeyrequest.RequestExpiresAtGT(now),
		emergencykeyrequest.ReviewedByNotNil(),
		emergencykeyrequest.ReviewedByNEQ(approverID),
	).SetApprovedBy(approverID).SetApprovedAt(now).SetKeyHash(keyHash).SetKeyExpiresAt(keyExpiresAt).SetStatus(emergencykeyrequest.StatusApproved).Save(ctx)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	if changed != 1 {
		return model.EmergencyKeyRequest{}, fmt.Errorf("request requires review and approval by two different platform administrators")
	}
	item, err := r.db.EmergencyKeyRequest.Get(ctx, id)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	return emergencyRequestFromEnt(item), nil
}

func (r *Repository) UseEmergencyKey(ctx context.Context, keyHash, deviceID string, now time.Time) (model.EmergencyKeyRequest, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.EmergencyKeyRequests {
			item := &r.store.EmergencyKeyRequests[index]
			if item.KeyHash != keyHash {
				continue
			}
			if item.DeviceID != deviceID {
				return model.EmergencyKeyRequest{}, fmt.Errorf("emergency key is not valid for this device")
			}
			if item.Status != "approved" || item.KeyExpiresAt == nil || !now.Before(*item.KeyExpiresAt) {
				return model.EmergencyKeyRequest{}, fmt.Errorf("emergency key is invalid, used, or expired")
			}
			item.Status = "used"
			item.UsedAt = &now
			item.UpdatedAt = now
			return *item, nil
		}
		return model.EmergencyKeyRequest{}, fmt.Errorf("emergency key not found")
	}
	changed, err := r.db.EmergencyKeyRequest.Update().Where(
		emergencykeyrequest.KeyHashEQ(keyHash),
		emergencykeyrequest.DeviceIDEQ(deviceID),
		emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusApproved),
		emergencykeyrequest.KeyExpiresAtGT(now),
	).SetStatus(emergencykeyrequest.StatusUsed).SetUsedAt(now).Save(ctx)
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
