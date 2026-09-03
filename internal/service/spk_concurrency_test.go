package service

import (
	"context"
	"sync"
	"testing"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/require"
)

// Multiple dashboard tabs must share one SPK computation and one daily record.
func TestSpkService_ConcurrentRecommendIsSingleflight(t *testing.T) {
	st := store.NewSeeded()
	svc := newSpkTestService(st)

	const requests = 16
	start := make(chan struct{})
	ids := make(chan string, requests)
	errs := make(chan error, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			recommendation, err := svc.Recommend(context.Background(), "usr_gading")
			if err != nil {
				errs <- err
				return
			}
			ids <- recommendation.RecommendationID
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var firstID string
	for id := range ids {
		if firstID == "" {
			firstID = id
		}
		require.Equal(t, firstID, id)
	}
	require.NotEmpty(t, firstID)

	count := 0
	for _, record := range st.Snapshot().InterventionRecords {
		if record.UserID == "usr_gading" {
			count++
		}
	}
	require.Equal(t, 1, count)
}
