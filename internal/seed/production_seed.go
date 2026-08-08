package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

// SeedProductionDefaults installs only public baseline content. It never
// creates demo users, activity, support cases, or operational records, and it
// does not overwrite administrator-managed content.
func SeedProductionDefaults(ctx context.Context, client *ent.Client, mediaPath string) error {
	_, err := SeedProductionDefaultsWithReport(ctx, client, mediaPath)
	return err
}

func SeedProductionDefaultsWithReport(ctx context.Context, client *ent.Client, mediaPath string) (LearningHubSeedReport, error) {
	moduleCount, err := client.PsychoeducationModule.Query().Count(ctx)
	if err != nil {
		return LearningHubSeedReport{}, err
	}
	if moduleCount == 0 {
		if err := SeedEducationModules(ctx, client, mediaPath); err != nil {
			return LearningHubSeedReport{}, err
		}
	}
	report, err := SeedLearningHubDefaultsWithReport(ctx, client, mediaPath)
	if err != nil {
		return LearningHubSeedReport{}, err
	}
	if err := SeedSiteSocialLinks(ctx, client); err != nil {
		return LearningHubSeedReport{}, err
	}
	return report, nil
}
