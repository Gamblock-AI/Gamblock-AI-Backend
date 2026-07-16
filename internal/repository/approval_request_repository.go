package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
)

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
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if item.ID == id && item.Status == "pending" {
				now := time.Now().UTC()
				item.Status = status
				item.StatusLabel = approvalStatusLabel(status)
				item.ResolvedAt = &now
				item.UpdatedAt = now
				return nil
			}
		}
		return fmt.Errorf("pending approval request not found")
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

func (r *Repository) CancelApprovalRequest(ctx context.Context, id, userID string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if item.ID == id && item.UserID == userID && item.Status == "pending" {
				item.Status = "cancelled"
				item.StatusLabel = approvalStatusLabel("cancelled")
				now := time.Now().UTC()
				item.ResolvedAt = &now
				item.UpdatedAt = now
				return nil
			}
		}
		return fmt.Errorf("pending approval request not found")
	}
	item, err := r.db.ApprovalRequest.Query().Where(
		approvalrequest.IDEQ(id),
		approvalrequest.UserID(userID),
		approvalrequest.StatusEQ(approvalrequest.StatusPending),
	).Only(ctx)
	if err != nil {
		return err
	}
	_, err = item.Update().
		SetStatus(approvalrequest.StatusCancelled).
		SetResolvedBy(userID).
		SetResolvedAt(time.Now().UTC()).
		Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) ResolveApprovalAsPartner(ctx context.Context, id, partnerUserID, status string) error {
	if status != "approved" && status != "denied" {
		return fmt.Errorf("invalid approval status")
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		links := activePartnerLinkIDs(r.store.Partners, partnerUserID)
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			_, allowed := links[item.PartnerLinkID]
			if item.ID != id || !allowed || item.Status != "pending" || !time.Now().UTC().Before(item.ExpiresAt) {
				continue
			}
			now := time.Now().UTC()
			item.Status = status
			item.StatusLabel = approvalStatusLabel(status)
			item.ResolvedAt = &now
			item.UpdatedAt = now
			return nil
		}
		return fmt.Errorf("pending approval request not found")
	}
	item, err := r.db.ApprovalRequest.Query().Where(
		approvalrequest.IDEQ(id),
		approvalrequest.StatusEQ(approvalrequest.StatusPending),
		approvalrequest.ExpiresAtGT(time.Now().UTC()),
	).Only(ctx)
	if err != nil {
		return err
	}
	allowed, err := r.db.PartnerLink.Query().Where(
		partnerlink.IDEQ(item.PartnerLinkID),
		partnerlink.PartnerUserID(partnerUserID),
		partnerlink.StatusEQ(partnerlink.StatusActive),
	).Exist(ctx)
	if err != nil || !allowed {
		return fmt.Errorf("partner is not authorized for this request")
	}
	_, err = item.Update().
		SetStatus(approvalrequest.Status(status)).
		SetResolvedBy(partnerUserID).
		SetResolvedAt(time.Now().UTC()).
		Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}
