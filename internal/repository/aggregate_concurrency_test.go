package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/stretchr/testify/require"
)

func TestSaveAggregateEventSnapshot_ConcurrentUpdatesRemainMonotonic(t *testing.T) {
	repo, st := newRepo(t)
	const writers = 32
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup

	for count := 1; count <= writers; count++ {
		count := count
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.SaveAggregateEventSnapshot(context.Background(), model.AggregateEvent{
				ID:     fmt.Sprintf("agg-concurrent-%d", count),
				UserID: "usr_gading", DeviceID: "dev_android",
				IdempotencyKey: "usr_gading:concurrent:dev_android",
				EventType:      "block_count_sync", EventDate: time.Now().UTC(), Count: count,
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	result, err := repo.SaveAggregateEventSnapshot(context.Background(), model.AggregateEvent{
		UserID: "usr_gading", DeviceID: "dev_android",
		IdempotencyKey: "usr_gading:concurrent:dev_android",
		EventType:      "block_count_sync", EventDate: time.Now().UTC(), Count: 0,
	})
	require.NoError(t, err)
	require.Equal(t, writers, result.Count)

	stored := 0
	for _, event := range st.Snapshot().AggregateEvents {
		if event.IdempotencyKey == "usr_gading:concurrent:dev_android" {
			stored++
		}
	}
	require.Equal(t, 1, stored)
}
