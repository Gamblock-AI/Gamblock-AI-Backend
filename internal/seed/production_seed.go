package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

// SeedProductionDefaults installs only public baseline content. It never
// creates demo users, activity, support cases, or operational records, and it
// does not overwrite administrator-managed education once content exists.
func SeedProductionDefaults(ctx context.Context, client *ent.Client, mediaPath string) error {
	moduleCount, err := client.PsychoeducationModule.Query().Count(ctx)
	if err != nil {
		return err
	}
	if moduleCount == 0 {
		if err := SeedEducationModules(ctx, client, mediaPath); err != nil {
			return err
		}
	}
	return SeedSiteSocialLinks(ctx, client)
}
