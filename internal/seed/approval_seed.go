package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
)

func SeedApprovals(ctx context.Context, client *ent.Client, now time.Time) error {
	if _, err := client.ApprovalRequest.Create().SetID("APR-2401").SetUserID("usr_gading").SetDeviceID("dev_android").SetMembershipID("mbr_active").SetAction(approvalrequest.ActionPauseProtection).SetStatus(approvalrequest.StatusPending).SetReason("Troubleshooting app setup").SetRequestedDurationMinutes(15).SetExpiresAt(now.Add(23 * time.Minute)).Save(ctx); err != nil {
		return err
	}
	if _, err := client.ApprovalRequest.Create().SetID("APR-2398").SetUserID("usr_gading").SetDeviceID("dev_windows").SetMembershipID("mbr_active").SetAction(approvalrequest.ActionUninstallDetected).SetStatus(approvalrequest.StatusApproved).SetReason("Accessibility service disabled").SetRequestedDurationMinutes(0).SetExpiresAt(now.Add(-20 * time.Hour)).SetResolvedBy("usr_suci").SetResolvedAt(now.Add(-18 * time.Hour)).Save(ctx); err != nil {
		return err
	}
	return nil
}
