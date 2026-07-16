package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) CreateApprovalRequestWithToken(ctx context.Context, reqID, userID, deviceID, partnerLinkID, action, reason string, duration int, expiresAt time.Time, quickTokenHash string) (model.ApprovalRequest, error) {
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		entry := store.ApprovalRequest{
			ID:                       reqID,
			UserID:                   userID,
			DeviceID:                 deviceID,
			PartnerLinkID:            partnerLinkID,
			Action:                   action,
			ActionLabel:              approvalActionLabel(action, duration),
			ExpiresIn:                humanExpiry(expiresAt),
			Status:                   "pending",
			StatusLabel:              approvalStatusLabel("pending"),
			Reason:                   reason,
			RequestedDurationMinutes: duration,
			CreatedAt:                now,
			UpdatedAt:                now,
			ExpiresAt:                expiresAt,
		}
		r.store.SetTokenMapping(quickTokenHash, entry)
		r.store.Approvals = append(r.store.Approvals, entry)
		return entry, nil
	}
	item, err := r.db.ApprovalRequest.Create().
		SetID(reqID).
		SetUserID(userID).
		SetDeviceID(deviceID).
		SetPartnerLinkID(partnerLinkID).
		SetQuickTokenHash(quickTokenHash).
		SetAction(approvalrequest.Action(action)).
		SetStatus(approvalrequest.StatusPending).
		SetNillableReason(optional(reason)).
		SetRequestedDurationMinutes(duration).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return model.ApprovalRequest{}, err
	}
	r.store.SetTokenMapping(quickTokenHash, store.ApprovalRequest{
		ID: reqID, UserID: userID, DeviceID: deviceID, PartnerLinkID: partnerLinkID,
		Action: action, Status: "pending", RequestedDurationMinutes: duration,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ExpiresAt: expiresAt,
	})
	r.RefreshStore(ctx)
	return model.ApprovalRequest{
		ID: reqID, UserID: userID, DeviceID: deviceID, PartnerLinkID: partnerLinkID,
		Action: action, ActionLabel: approvalActionLabel(action, duration),
		ExpiresIn: humanExpiry(expiresAt), Status: "pending", StatusLabel: approvalStatusLabel("pending"),
		Reason: reason, RequestedDurationMinutes: duration,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ExpiresAt: expiresAt,
	}, nil
}

func (r *Repository) GetApprovalByQuickToken(ctx context.Context, tokenHash string) (store.ApprovalRequest, error) {
	if r.db != nil {
		item, err := r.db.ApprovalRequest.Query().Where(approvalrequest.QuickTokenHashEQ(tokenHash)).Only(ctx)
		if err != nil {
			return store.ApprovalRequest{}, err
		}
		return store.ApprovalRequest{
			ID:                       item.ID,
			UserID:                   item.UserID,
			DeviceID:                 value(item.DeviceID),
			PartnerLinkID:            item.PartnerLinkID,
			QuickTokenHash:           value(item.QuickTokenHash),
			Action:                   item.Action.String(),
			Status:                   item.Status.String(),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: valueInt(item.RequestedDurationMinutes),
			ResolvedAt:               item.ResolvedAt,
			AppliedAt:                item.AppliedAt,
			GrantExpiresAt:           item.GrantExpiresAt,
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
			ExpiresAt:                item.ExpiresAt,
		}, nil
	}
	entry, ok := r.store.GetTokenMapping(tokenHash)
	if !ok {
		return store.ApprovalRequest{}, fmt.Errorf("token not found")
	}
	return entry, nil
}

func (r *Repository) UpdateQuickTokenState(tokenHash string, request store.ApprovalRequest) {
	r.store.SetTokenMapping(tokenHash, request)
}
