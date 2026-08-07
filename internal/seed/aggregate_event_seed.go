package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
)

// SeedAggregateEvents installs privacy-safe demo counters only. Production
// defaults never call this function.
func SeedAggregateEvents(ctx context.Context, client *ent.Client, now time.Time) error {
	type aggregateFixture struct {
		studentID string
		deviceID  string
		daysAgo   int
		count     int
	}
	fixtures := []aggregateFixture{
		{studentID: "usr_gading", deviceID: "dev_android", daysAgo: 0, count: 3},
		{studentID: "usr_gading", deviceID: "dev_windows", daysAgo: 0, count: 1},
		{studentID: "usr_gading", deviceID: "dev_android", daysAgo: 1, count: 2},
		{studentID: "usr_gading", deviceID: "dev_windows", daysAgo: 2, count: 2},
		{studentID: "usr_gading", deviceID: "dev_android", daysAgo: 3, count: 1},
		{studentID: "usr_dery", deviceID: "dev_dery_android", daysAgo: 0, count: 2},
		{studentID: "usr_dery", deviceID: "dev_dery_android", daysAgo: 4, count: 1},
	}
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	for _, fixture := range fixtures {
		localDate := now.In(jakarta).AddDate(0, 0, -fixture.daysAgo)
		eventDate := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.UTC)
		id := fmt.Sprintf("agg_seed_%s_%s_%d", fixture.studentID, fixture.deviceID, fixture.daysAgo)
		if _, err := client.AggregateEvent.Get(ctx, id); err == nil {
			continue
		} else if !ent.IsNotFound(err) {
			return err
		}
		if _, err := client.AggregateEvent.Create().
			SetID(id).
			SetUserID(fixture.studentID).
			SetDeviceID(fixture.deviceID).
			SetIdempotencyKey(id + ":block_count_sync").
			SetEventType(aggregateevent.EventTypeBlockCountSync).
			SetEventDate(eventDate).
			SetCount(fixture.count).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
