package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

func SeedNotifications(ctx context.Context, client *ent.Client) error {
	if _, err := client.NotificationDelivery.Create().SetID("ntf_1").SetApprovalRequestID("APR-2401").SetChannel("email").SetRecipient("suci@gmail.com").SetStatus("sent").SetAttemptCount(1).Save(ctx); err != nil {
		return err
	}
	return nil
}
