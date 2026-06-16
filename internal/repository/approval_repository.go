package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetApprovalRequests(ctx context.Context, userID string) ([]model.ApprovalRequest, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.ApprovalRequest
		for _, a := range snapshot.Approvals {
			list = append(list, model.ApprovalRequest{
				ID:                       a.ID,
				Action:                   a.Action,
				ExpiresIn:                a.ExpiresIn,
				Status:                   a.Status,
				Reason:                   a.Reason,
				RequestedDurationMinutes: a.RequestedDurationMinutes,
				CreatedAt:                a.CreatedAt,
				UpdatedAt:                a.UpdatedAt,
			})
		}
		return list, nil
	}

	rows, err := r.db.ApprovalRequest.Query().
		Where(approvalrequest.UserID(userID)).
		Order(ent.Desc(approvalrequest.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.ApprovalRequest
	for _, item := range rows {
		list = append(list, model.ApprovalRequest{
			ID:                       item.ID,
			Action:                   item.Action.String(),
			ExpiresIn:                humanExpiry(item.ExpiresAt),
			Status:                   humanApprovalStatus(item.Status.String()),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: valueInt(item.RequestedDurationMinutes),
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) CreateApprovalRequest(ctx context.Context, reqID, userID, deviceID, partnerLinkID, action, reason string, duration int, expiresAt time.Time) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.ApprovalRequest.Create().
		SetID(reqID).
		SetUserID(userID).
		SetDeviceID(deviceID).
		SetPartnerLinkID(partnerLinkID).
		SetAction(approvalrequest.Action(action)).
		SetStatus(approvalrequest.StatusPending).
		SetNillableReason(optional(reason)).
		SetRequestedDurationMinutes(duration).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) UpdateApprovalRequest(ctx context.Context, id, status, resolvedBy string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.ApprovalRequest.UpdateOneID(id).
		SetStatus(approvalrequest.Status(status)).
		SetResolvedBy(resolvedBy).
		SetResolvedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) CreateApprovalRequestWithToken(ctx context.Context, reqID, userID, deviceID, partnerLinkID, action, reason string, duration int, expiresAt time.Time, quickTokenHash string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		entry := store.ApprovalRequest{
			ID:                       reqID,
			Action:                   humanApprovalAction(action, duration),
			ExpiresIn:                humanExpiry(expiresAt),
			Status:                   "Pending partner approval",
			Reason:                   reason,
			RequestedDurationMinutes: duration,
			CreatedAt:                time.Now().UTC(),
			UpdatedAt:                time.Now().UTC(),
		}
		r.store.SetTokenMapping(quickTokenHash, entry)
		r.store.Approvals = append(r.store.Approvals, entry)
		return nil
	}
	_, err := r.db.ApprovalRequest.Create().
		SetID(reqID).
		SetUserID(userID).
		SetDeviceID(deviceID).
		SetPartnerLinkID(partnerLinkID).
		SetAction(approvalrequest.Action(action)).
		SetStatus(approvalrequest.StatusPending).
		SetNillableReason(optional(reason)).
		SetRequestedDurationMinutes(duration).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		return err
	}
	r.store.SetTokenMapping(quickTokenHash, store.ApprovalRequest{
		ID:     reqID,
		Status: "pending",
	})
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) GetApprovalByQuickToken(ctx context.Context, tokenHash string) (store.ApprovalRequest, error) {
	entry, ok := r.store.GetTokenMapping(tokenHash)
	if !ok {
		return store.ApprovalRequest{}, fmt.Errorf("token not found")
	}
	return entry, nil
}

type PendingBatchApproval struct {
	MemberName   string
	Action       string
	QuickLink    string
	PartnerPhone string
}

func (r *Repository) GetPendingBatchApprovals(ctx context.Context) ([]PendingBatchApproval, error) {
	r.store.RLock()
	defer r.store.RUnlock()
	var results []PendingBatchApproval
	for _, a := range r.store.Approvals {
		if a.Status == "Pending partner approval" {
			results = append(results, PendingBatchApproval{
				MemberName:   "Member",
				Action:       a.Action,
				QuickLink:    "https://gamblock.id/approve/demo-token",
				PartnerPhone: "",
			})
		}
	}
	if r.db != nil {
		rows, err := r.db.ApprovalRequest.Query().Where(approvalrequest.StatusEQ("pending")).All(ctx)
		if err == nil {
			for _, row := range rows {
				results = append(results, PendingBatchApproval{
					MemberName:   row.UserID,
					Action:       row.Action.String(),
					QuickLink:    "https://gamblock.id/approve/" + row.ID,
					PartnerPhone: "",
				})
			}
		}
	}
	return results, nil
}
