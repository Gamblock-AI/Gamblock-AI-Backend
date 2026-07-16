package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// GetApprovalRequests returns requests owned by the user and requests the user
// may resolve as an active partner.
func (r *Repository) GetApprovalRequests(ctx context.Context, userID string) ([]model.ApprovalRequest, error) {
	if r.db == nil {
		now := time.Now().UTC()
		r.store.Lock()
		for index := range r.store.Approvals {
			item := &r.store.Approvals[index]
			if item.Status == "pending" && !now.Before(item.ExpiresAt) {
				item.Status = "expired"
				item.StatusLabel = approvalStatusLabel(item.Status)
				item.UpdatedAt = now
			}
		}
		r.store.Unlock()

		snapshot := r.store.Snapshot()
		partnerLinks := activePartnerLinkIDs(snapshot.Partners, userID)
		var list []model.ApprovalRequest
		for _, item := range snapshot.Approvals {
			if _, isPartnerRequest := partnerLinks[item.PartnerLinkID]; item.UserID != userID && !isPartnerRequest {
				continue
			}
			list = append(list, model.ApprovalRequest{
				ID:                       item.ID,
				UserID:                   item.UserID,
				DeviceID:                 item.DeviceID,
				PartnerLinkID:            item.PartnerLinkID,
				Action:                   item.Action,
				ActionLabel:              approvalActionLabel(item.Action, item.RequestedDurationMinutes),
				ExpiresIn:                item.ExpiresIn,
				Status:                   item.Status,
				StatusLabel:              approvalStatusLabel(item.Status),
				Reason:                   item.Reason,
				RequestedDurationMinutes: item.RequestedDurationMinutes,
				ResolvedAt:               item.ResolvedAt,
				AppliedAt:                item.AppliedAt,
				GrantExpiresAt:           item.GrantExpiresAt,
				CreatedAt:                item.CreatedAt,
				UpdatedAt:                item.UpdatedAt,
				ExpiresAt:                item.ExpiresAt,
			})
		}
		return list, nil
	}

	partnerLinkIDs, err := r.db.PartnerLink.Query().
		Where(partnerlink.PartnerUserID(userID), partnerlink.StatusEQ(partnerlink.StatusActive)).
		IDs(ctx)
	if err != nil {
		return nil, err
	}
	_, _ = r.db.ApprovalRequest.Update().
		Where(
			approvalrequest.StatusEQ(approvalrequest.StatusPending),
			approvalrequest.ExpiresAtLTE(time.Now().UTC()),
		).
		SetStatus(approvalrequest.StatusExpired).
		Save(ctx)

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
			ActionLabel:              approvalActionLabel(item.Action.String(), valueInt(item.RequestedDurationMinutes)),
			ExpiresIn:                humanExpiry(item.ExpiresAt),
			Status:                   item.Status.String(),
			StatusLabel:              approvalStatusLabel(item.Status.String()),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: valueInt(item.RequestedDurationMinutes),
			ResolvedAt:               item.ResolvedAt,
			AppliedAt:                item.AppliedAt,
			GrantExpiresAt:           item.GrantExpiresAt,
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
			ExpiresAt:                item.ExpiresAt,
		})
	}
	return list, nil
}
