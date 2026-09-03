package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Refresh rotation is a single-use operation even when several requests race
// with the same refresh token. This protects the session boundary from replay.
func TestAuthService_ConcurrentRefreshConsumesTokenOnce(t *testing.T) {
	svc, _ := newRecoveryAuthService(t)
	session, err := svc.Login(context.Background(), "gading@gmail.com", "password", "")
	require.NoError(t, err)

	const attempts = 16
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, refreshErr := svc.Refresh(context.Background(), session.RefreshToken)
			results <- refreshErr
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	failures := 0
	for refreshErr := range results {
		if refreshErr == nil {
			successes++
		} else {
			failures++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, attempts-1, failures)
}
