package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

func SeedAuditLogs(ctx context.Context, client *ent.Client) error {
	if _, err := client.AuditLog.Create().SetID("audit_1").SetActorID("usr_nasywa").SetActorEmail("nasywa@gmail.com").SetAction("education_module_published").SetTargetType("education_module").SetTargetID("mod_impulse_cycle").SetReason("seed").Save(ctx); err != nil {
		return err
	}
	return nil
}
