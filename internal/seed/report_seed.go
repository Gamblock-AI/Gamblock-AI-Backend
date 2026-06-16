package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

func SeedReportRollups(ctx context.Context, client *ent.Client, now time.Time) error {
	if _, err := client.ReportRollup.Create().SetID("rollup_platform_week").SetScope("platform").SetScopeID("platform").SetPeriod("weekly").SetPeriodStart(now.AddDate(0, 0, -7)).SetMetricsJSON(map[string]any{"active_devices": 121, "aggregate_interventions": 42, "tamper_events": 3, "content_completion_percent": 68}).Save(ctx); err != nil {
		return err
	}
	return nil
}
