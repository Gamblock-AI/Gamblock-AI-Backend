package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
)

func SeedDataRequests(ctx context.Context, client *ent.Client, now time.Time) error {
	if _, err := client.DataRequest.Create().SetID("DR-1042").SetUserID("usr_gading").SetType("export").SetStatus("completed").SetRequestedAt(now.Add(-24 * time.Hour)).SetCompletedAt(now.Add(-2 * time.Hour)).SetFailureCode("result_unavailable").Save(ctx); err != nil {
		return err
	}
	if _, err := client.DataRequest.Create().SetID("DR-1035").SetUserID("usr_gading").SetType("delete").SetStatus("processing").SetRequestedAt(now.Add(-6 * time.Hour)).Save(ctx); err != nil {
		return err
	}
	return nil
}
