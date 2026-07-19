package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
)

func SeedSupportCases(ctx context.Context, client *ent.Client) error {
	if _, err := client.SupportCase.Create().SetID("CASE-1087").SetUserID("usr_gading").SetType(supportcase.TypeDeviceRecovery).SetStatus(supportcase.StatusWaitingUser).SetPriority(supportcase.PriorityNormal).SetSummary("Setup and permissions").Save(ctx); err != nil {
		return err
	}
	if _, err := client.SupportCase.Create().SetID("CASE-1084").SetUserID("usr_gading").SetType(supportcase.TypeStuckApproval).SetStatus(supportcase.StatusWaitingSupport).SetPriority(supportcase.PriorityHigh).SetSummary("Partner approval issue").Save(ctx); err != nil {
		return err
	}
	return nil
}
