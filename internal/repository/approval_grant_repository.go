package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) ApplyApprovedRequest(ctx context.Context, id, userID, deviceID string, now time.Time) (model.ApprovalGrant, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if item.ID == id && item.UserID == userID && item.DeviceID == deviceID {
				return applyApprovalGrant(item, now)
			}
		}
		return model.ApprovalGrant{}, fmt.Errorf("approval request not found")
	}

	item, err := r.db.ApprovalRequest.Query().Where(
		approvalrequest.IDEQ(id),
		approvalrequest.UserID(userID),
		approvalrequest.DeviceIDEQ(deviceID),
	).Only(ctx)
	if err != nil {
		return model.ApprovalGrant{}, err
	}
	if item.Status != approvalrequest.StatusApproved || item.ResolvedAt == nil {
		return model.ApprovalGrant{}, fmt.Errorf("approval request is not approved")
	}
	if now.After(item.ResolvedAt.Add(30 * time.Minute)) {
		return model.ApprovalGrant{}, fmt.Errorf("approval apply window expired")
	}
	if item.AppliedAt != nil && item.GrantExpiresAt != nil {
		return model.ApprovalGrant{
			RequestID: id, DeviceID: deviceID, Action: item.Action.String(),
			GrantStartsAt: *item.AppliedAt, GrantExpiresAt: *item.GrantExpiresAt,
		}, nil
	}
	grantExpiresAt, err := approvalGrantExpiry(item.Action.String(), valueInt(item.RequestedDurationMinutes), now)
	if err != nil {
		return model.ApprovalGrant{}, err
	}
	changed, err := r.db.ApprovalRequest.Update().
		Where(
			approvalrequest.IDEQ(id),
			approvalrequest.UserID(userID),
			approvalrequest.DeviceIDEQ(deviceID),
			approvalrequest.StatusEQ(approvalrequest.StatusApproved),
			approvalrequest.AppliedAtIsNil(),
		).
		SetAppliedAt(now).
		SetGrantExpiresAt(grantExpiresAt).
		Save(ctx)
	if err != nil {
		return model.ApprovalGrant{}, err
	}
	if changed == 0 {
		reloaded, reloadErr := r.db.ApprovalRequest.Get(ctx, id)
		if reloadErr != nil || reloaded.AppliedAt == nil || reloaded.GrantExpiresAt == nil {
			return model.ApprovalGrant{}, fmt.Errorf("approval grant could not be applied")
		}
		return model.ApprovalGrant{
			RequestID: id, DeviceID: deviceID, Action: reloaded.Action.String(),
			GrantStartsAt: *reloaded.AppliedAt, GrantExpiresAt: *reloaded.GrantExpiresAt,
		}, nil
	}
	r.RefreshStore(ctx)
	return model.ApprovalGrant{
		RequestID: id, DeviceID: deviceID, Action: item.Action.String(),
		GrantStartsAt: now, GrantExpiresAt: grantExpiresAt,
	}, nil
}

func applyApprovalGrant(item *model.ApprovalRequest, now time.Time) (model.ApprovalGrant, error) {
	if item.Status != "approved" || item.ResolvedAt == nil {
		return model.ApprovalGrant{}, fmt.Errorf("approval request is not approved")
	}
	if now.After(item.ResolvedAt.Add(30 * time.Minute)) {
		return model.ApprovalGrant{}, fmt.Errorf("approval apply window expired")
	}
	if item.AppliedAt != nil && item.GrantExpiresAt != nil {
		return model.ApprovalGrant{
			RequestID: item.ID, DeviceID: item.DeviceID, Action: item.Action,
			GrantStartsAt: *item.AppliedAt, GrantExpiresAt: *item.GrantExpiresAt,
		}, nil
	}
	grantExpiresAt, err := approvalGrantExpiry(item.Action, item.RequestedDurationMinutes, now)
	if err != nil {
		return model.ApprovalGrant{}, err
	}
	item.AppliedAt = &now
	item.GrantExpiresAt = &grantExpiresAt
	item.UpdatedAt = now
	return model.ApprovalGrant{
		RequestID: item.ID, DeviceID: item.DeviceID, Action: item.Action,
		GrantStartsAt: now, GrantExpiresAt: grantExpiresAt,
	}, nil
}

func approvalGrantExpiry(action string, duration int, now time.Time) (time.Time, error) {
	switch action {
	case "pause_protection":
		if duration <= 0 {
			return time.Time{}, fmt.Errorf("pause duration is required")
		}
		return now.Add(time.Duration(duration) * time.Minute), nil
	case "uninstall_detected":
		return now.Add(10 * time.Minute), nil
	default:
		return time.Time{}, fmt.Errorf("approval action cannot be applied by a protection client")
	}
}
