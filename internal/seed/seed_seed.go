package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

func Seed(ctx context.Context, client *ent.Client, mediaPath ...string) error {
	count, err := client.User.Query().Count(ctx)
	if err != nil {
		return err
	}
	if err := SeedEducationModules(ctx, client, mediaPath...); err != nil {
		return err
	}
	if err := SeedSiteSocialLinks(ctx, client); err != nil {
		return err
	}
	if _, err := SeedLearningHubDefaultsWithReport(ctx, client, mediaPath...); err != nil {
		return err
	}
	now := time.Now().UTC()
	// Idempotent demo recovery data (Niat quiz, check-in, missions, learning)
	// so the SPK recommendation for demo students has enough input. It runs
	// unconditionally to backfill existing demo databases as well.
	if err := SeedDemoRecoveryData(ctx, client, now); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if err := SeedUsers(ctx, client); err != nil {
		return err
	}
	if err := SeedDevices(ctx, client, now); err != nil {
		return err
	}
	if err := SeedPartners(ctx, client, now); err != nil {
		return err
	}
	if err := SeedAccountabilityGroups(ctx, client, now); err != nil {
		return err
	}
	if err := SeedAggregateEvents(ctx, client, now); err != nil {
		return err
	}
	if err := SeedApprovals(ctx, client, now); err != nil {
		return err
	}
	if err := SeedReleases(ctx, client, now); err != nil {
		return err
	}
	if err := SeedOrganizations(ctx, client); err != nil {
		return err
	}
	if err := SeedSupportCases(ctx, client); err != nil {
		return err
	}
	if err := SeedDataRequests(ctx, client, now); err != nil {
		return err
	}
	if err := SeedReportRollups(ctx, client, now); err != nil {
		return err
	}
	if err := SeedAuditLogs(ctx, client); err != nil {
		return err
	}
	if err := SeedNotifications(ctx, client); err != nil {
		return err
	}
	return nil
}
