package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetApprovalRequests(ctx context.Context, userID string) ([]model.ApprovalRequest, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		partnerLinks := make(map[string]struct{})
		for _, link := range snapshot.Partners {
			if link.PartnerUserID == userID && link.Status == "active" {
				partnerLinks[link.ID] = struct{}{}
			}
		}
		var list []model.ApprovalRequest
		for _, a := range snapshot.Approvals {
			if _, partnerRequest := partnerLinks[a.PartnerLinkID]; a.UserID != userID && !partnerRequest {
				continue
			}
			list = append(list, model.ApprovalRequest{
				ID:                       a.ID,
				UserID:                   a.UserID,
				DeviceID:                 a.DeviceID,
				PartnerLinkID:            a.PartnerLinkID,
				Action:                   a.Action,
				ExpiresIn:                a.ExpiresIn,
				Status:                   a.Status,
				Reason:                   a.Reason,
				RequestedDurationMinutes: a.RequestedDurationMinutes,
				CreatedAt:                a.CreatedAt,
				UpdatedAt:                a.UpdatedAt,
				ExpiresAt:                a.ExpiresAt,
			})
		}
		return list, nil
	}

	partnerLinkIDs, err := r.db.PartnerLink.Query().Where(partnerlink.PartnerUserID(userID), partnerlink.StatusEQ(partnerlink.StatusActive)).IDs(ctx)
	if err != nil {
		return nil, err
	}
	predicate := approvalrequest.UserID(userID)
	if len(partnerLinkIDs) > 0 {
		predicate = approvalrequest.Or(predicate, approvalrequest.PartnerLinkIDIn(partnerLinkIDs...))
	}
	rows, err := r.db.ApprovalRequest.Query().
		Where(predicate).
		Order(ent.Desc(approvalrequest.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.ApprovalRequest
	for _, item := range rows {
		list = append(list, model.ApprovalRequest{
			ID:                       item.ID,
			UserID:                   item.UserID,
			DeviceID:                 value(item.DeviceID),
			PartnerLinkID:            item.PartnerLinkID,
			Action:                   item.Action.String(),
			ExpiresIn:                humanExpiry(item.ExpiresAt),
			Status:                   humanApprovalStatus(item.Status.String()),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: valueInt(item.RequestedDurationMinutes),
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
			ExpiresAt:                item.ExpiresAt,
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

func (r *Repository) CancelApprovalRequest(ctx context.Context, id, userID string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if item.ID == id && item.UserID == userID && item.Status == "pending" {
				item.Status = "cancelled"
				item.UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("pending approval request not found")
	}
	item, err := r.db.ApprovalRequest.Query().Where(
		approvalrequest.IDEQ(id), approvalrequest.UserID(userID),
		approvalrequest.StatusEQ(approvalrequest.StatusPending),
	).Only(ctx)
	if err != nil {
		return err
	}
	_, err = item.Update().SetStatus(approvalrequest.StatusCancelled).SetResolvedBy(userID).SetResolvedAt(time.Now().UTC()).Save(ctx)
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
		links := make(map[string]struct{})
		for _, link := range r.store.Partners {
			if link.PartnerUserID == partnerUserID && link.Status == "active" {
				links[link.ID] = struct{}{}
			}
		}
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if _, allowed := links[item.PartnerLinkID]; item.ID == id && allowed && item.Status == "pending" && time.Now().UTC().Before(item.ExpiresAt) {
				item.Status = status
				item.UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("pending approval request not found")
	}
	item, err := r.db.ApprovalRequest.Query().Where(
		approvalrequest.IDEQ(id), approvalrequest.StatusEQ(approvalrequest.StatusPending),
		approvalrequest.ExpiresAtGT(time.Now().UTC()),
	).Only(ctx)
	if err != nil {
		return err
	}
	allowed, err := r.db.PartnerLink.Query().Where(
		partnerlink.IDEQ(item.PartnerLinkID), partnerlink.PartnerUserID(partnerUserID),
		partnerlink.StatusEQ(partnerlink.StatusActive),
	).Exist(ctx)
	if err != nil || !allowed {
		return fmt.Errorf("partner is not authorized for this request")
	}
	_, err = item.Update().SetStatus(approvalrequest.Status(status)).SetResolvedBy(partnerUserID).SetResolvedAt(time.Now().UTC()).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) CreateApprovalRequestWithToken(ctx context.Context, reqID, userID, deviceID, partnerLinkID, action, reason string, duration int, expiresAt time.Time, quickTokenHash string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		entry := store.ApprovalRequest{
			ID:                       reqID,
			UserID:                   userID,
			DeviceID:                 deviceID,
			PartnerLinkID:            partnerLinkID,
			Action:                   humanApprovalAction(action, duration),
			ExpiresIn:                humanExpiry(expiresAt),
			Status:                   "pending",
			Reason:                   reason,
			RequestedDurationMinutes: duration,
			CreatedAt:                time.Now().UTC(),
			UpdatedAt:                time.Now().UTC(),
			ExpiresAt:                expiresAt,
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
		SetQuickTokenHash(quickTokenHash).
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
		ID: reqID, UserID: userID, DeviceID: deviceID, PartnerLinkID: partnerLinkID,
		Status: "pending", ExpiresAt: expiresAt,
	})
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) GetApprovalByQuickToken(ctx context.Context, tokenHash string) (store.ApprovalRequest, error) {
	if r.db != nil {
		item, err := r.db.ApprovalRequest.Query().Where(approvalrequest.QuickTokenHashEQ(tokenHash)).Only(ctx)
		if err != nil {
			return store.ApprovalRequest{}, err
		}
		return store.ApprovalRequest{
			ID: item.ID, UserID: item.UserID, DeviceID: value(item.DeviceID),
			PartnerLinkID: item.PartnerLinkID, QuickTokenHash: value(item.QuickTokenHash),
			Action: item.Action.String(), Status: item.Status.String(), Reason: value(item.Reason),
			RequestedDurationMinutes: valueInt(item.RequestedDurationMinutes),
			CreatedAt:                item.CreatedAt, UpdatedAt: item.UpdatedAt, ExpiresAt: item.ExpiresAt,
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
