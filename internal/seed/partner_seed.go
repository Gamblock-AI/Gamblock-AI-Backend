package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
)

func SeedPartners(ctx context.Context, client *ent.Client, now time.Time) error {
	if _, err := client.PartnerLink.Create().SetID("pl_active").SetUserID("usr_gading").SetPartnerUserID("usr_suci").SetPartnerEmail("suci@gmail.com").SetPartnerPhone("+62 812-0000-0000").SetStatus(partnerlink.StatusActive).SetAcceptedAt(now.Add(-14 * 24 * time.Hour)).Save(ctx); err != nil {
		return err
	}
	return nil
}
