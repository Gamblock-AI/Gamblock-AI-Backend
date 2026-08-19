package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

const standaloneRemovalGrantWindow = 10 * time.Minute

// IssueStandaloneRemovalGrant records a single-use, short-lived approval grant
// for a device whose owner has no active accountability partnership. It is
// the only path that hands a removal grant to a partnerless student. The
// approval row is created in the approved+applied state so it can never be
// reused through the partner flow.
func (r *Repository) IssueStandaloneRemovalGrant(ctx context.Context, reqID, userID, deviceID, grantJTI string, now time.Time) (model.ApprovalGrant, error) {
	if reqID == "" || userID == "" || deviceID == "" || grantJTI == "" {
		return model.ApprovalGrant{}, fmt.Errorf("standalone removal grant claims are incomplete")
	}

	grantExpiresAt := now.Add(standaloneRemovalGrantWindow)
	reason := "Standalone removal without an active accountability partner."
	actionLabel := approvalActionLabel("uninstall_detected", 0)

	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if item.UserID == userID && item.DeviceID == deviceID &&
				item.MembershipID == "" && item.Status == "approved" &&
				item.GrantExpiresAt != nil && item.GrantExpiresAt.After(now) {
				return model.ApprovalGrant{}, fmt.Errorf("a standalone removal grant is already active")
			}
		}
		r.store.Approvals = append(r.store.Approvals, model.ApprovalRequest{
			ID:                       reqID,
			UserID:                   userID,
			DeviceID:                 deviceID,
			Action:                   "uninstall_detected",
			ActionLabel:              actionLabel,
			ExpiresIn:                humanExpiry(grantExpiresAt),
			Status:                   "approved",
			StatusLabel:              approvalStatusLabel("approved"),
			Reason:                   reason,
			RequestedDurationMinutes: 10,
			ResolvedAt:               &now,
			AppliedAt:                &now,
			GrantExpiresAt:           &grantExpiresAt,
			GrantJTI:                 grantJTI,
			CreatedAt:                now,
			UpdatedAt:                now,
			ExpiresAt:                grantExpiresAt,
		})
		return model.ApprovalGrant{
			RequestID: reqID, DeviceID: deviceID, Action: "uninstall_detected",
			GrantStartsAt: now, GrantExpiresAt: grantExpiresAt, GrantJTI: grantJTI,
		}, nil
	}

	existing, err := r.db.ApprovalRequest.Query().Where(
		approvalrequest.UserID(userID),
		approvalrequest.DeviceIDEQ(deviceID),
		approvalrequest.MembershipIDEQ(""),
		approvalrequest.StatusEQ(approvalrequest.StatusApproved),
		approvalrequest.GrantExpiresAtNotNil(),
		approvalrequest.GrantExpiresAtGT(now),
	).Exist(ctx)
	if err != nil {
		return model.ApprovalGrant{}, err
	}
	if existing {
		return model.ApprovalGrant{}, fmt.Errorf("a standalone removal grant is already active")
	}

	item, err := r.db.ApprovalRequest.Create().
		SetID(reqID).
		SetUserID(userID).
		SetDeviceID(deviceID).
		SetMembershipID("").
		SetAction(approvalrequest.ActionUninstallDetected).
		SetStatus(approvalrequest.StatusApproved).
		SetNillableReason(optional(reason)).
		SetRequestedDurationMinutes(10).
		SetResolvedAt(now).
		SetAppliedAt(now).
		SetGrantExpiresAt(grantExpiresAt).
		SetGrantJti(grantJTI).
		SetExpiresAt(grantExpiresAt).
		Save(ctx)
	if err != nil {
		return model.ApprovalGrant{}, err
	}
	r.RefreshStore(ctx)
	return model.ApprovalGrant{
		RequestID: item.ID, DeviceID: deviceID, Action: "uninstall_detected",
		GrantStartsAt: now, GrantExpiresAt: grantExpiresAt, GrantJTI: grantJTI,
	}, nil
}
