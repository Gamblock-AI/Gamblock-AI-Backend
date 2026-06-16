package repository

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/notificationdelivery"
)

func (r *Repository) QueueNotification(ctx context.Context, id, appReqID, supportCaseID, channel, recipient string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.NotificationDelivery.Create().
		SetID(id).
		SetNillableApprovalRequestID(optional(appReqID)).
		SetNillableSupportCaseID(optional(supportCaseID)).
		SetChannel(notificationdelivery.Channel(channel)).
		SetRecipient(recipient).
		SetStatus(notificationdelivery.StatusQueued).
		Save(ctx)
	return err
}
