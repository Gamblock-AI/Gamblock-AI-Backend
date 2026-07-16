package db

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func loadOperationsStore(ctx context.Context, client *ent.Client, out *store.Store) error {
	orgs, err := client.Organization.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range orgs {
		members, err := client.OrganizationMember.Query().
			Where(organizationmember.OrganizationIDEQ(item.ID)).
			Count(ctx)
		if err != nil {
			return err
		}
		out.Organizations = append(out.Organizations, store.Organization{
			ID:        item.ID,
			Name:      item.Name,
			Slug:      item.Slug,
			Status:    item.Status.String(),
			Members:   members,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	supportCases, err := client.SupportCase.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range supportCases {
		out.SupportCases = append(out.SupportCases, store.SupportCase{
			ID:        item.ID,
			UserID:    item.UserID,
			Title:     item.Summary,
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			Priority:  item.Priority.String(),
			Owner:     "",
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	dataRequests, err := client.DataRequest.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range dataRequests {
		out.DataRequests = append(out.DataRequests, store.DataRequest{
			ID:        item.ID,
			UserID:    item.UserID,
			Title:     humanDataRequestTitle(item.Type.String()),
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			CreatedAt: item.RequestedAt,
			UpdatedAt: time.Time{},
		})
	}

	audits, err := client.AuditLog.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range audits {
		out.AuditEvents = append(out.AuditEvents, store.AuditEvent{
			ID:        item.ID,
			Actor:     item.ActorEmail,
			Action:    item.Action,
			Target:    item.TargetID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: time.Time{},
		})
	}

	notifications, err := client.NotificationDelivery.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range notifications {
		out.NotificationEvents = append(out.NotificationEvents, store.NotificationItem{
			ID:        item.ID,
			Channel:   item.Channel.String(),
			Recipient: item.Recipient,
			Status:    item.Status.String(),
			Reason:    value(item.ApprovalRequestID),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return nil
}
