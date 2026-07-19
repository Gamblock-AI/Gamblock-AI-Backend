package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitygroup"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitymembership"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
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

func (r *Repository) CancelPendingApprovalsForMembership(ctx context.Context, membershipID, resolvedBy string) error {
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.Approvals {
			item := &r.store.Approvals[i]
			if item.MembershipID == membershipID && item.Status == "pending" {
				item.Status = "cancelled"
				item.StatusLabel = approvalStatusLabel("cancelled")
				item.ResolvedAt = &now
				item.UpdatedAt = now
			}
		}
		return nil
	}
	_, err := r.db.ApprovalRequest.Update().Where(
		approvalrequest.MembershipIDEQ(membershipID),
		approvalrequest.StatusEQ(approvalrequest.StatusPending),
	).SetStatus(approvalrequest.StatusCancelled).SetResolvedBy(resolvedBy).SetResolvedAt(now).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) ResolveApprovalAsPartner(ctx context.Context, id, partnerUserID, status, supportiveResponse string) error {
	if status != "approved" && status != "denied" {
		return fmt.Errorf("invalid approval status")
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		groupIDs := map[string]bool{}
		for _, group := range r.store.AccountabilityGroups {
			if group.OwnerPartnerID == partnerUserID {
				groupIDs[group.ID] = true
			}
		}
		membershipIDs := map[string]bool{}
		for _, membership := range r.store.AccountabilityMemberships {
			if groupIDs[membership.GroupID] {
				membershipIDs[membership.ID] = true
			}
		}
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			allowed := membershipIDs[item.MembershipID]
			if item.ID != id || !allowed || item.Status != "pending" || !time.Now().UTC().Before(item.ExpiresAt) {
				continue
			}
			now := time.Now().UTC()
			item.Status = status
			item.StatusLabel = approvalStatusLabel(status)
			item.SupportiveResponse = supportiveResponse
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
	membership, err := r.db.AccountabilityMembership.Query().Where(
		accountabilitymembership.IDEQ(value(item.MembershipID)),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("membership for request was not found")
	}
	allowed, err := r.db.AccountabilityGroup.Query().Where(
		accountabilitygroup.IDEQ(membership.GroupID),
		accountabilitygroup.OwnerPartnerIDEQ(partnerUserID),
	).Exist(ctx)
	if err != nil || !allowed {
		return fmt.Errorf("partner is not authorized for this request")
	}
	_, err = item.Update().
		SetStatus(approvalrequest.Status(status)).
		SetNillableSupportiveResponse(optional(supportiveResponse)).
		SetResolvedBy(partnerUserID).
		SetResolvedAt(time.Now().UTC()).
		Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}
