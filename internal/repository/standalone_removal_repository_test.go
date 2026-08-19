package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueStandaloneRemovalGrant_SingleUseWithinWindow(t *testing.T) {
	repo, _ := newRepo(t)
	now := time.Now().UTC()

	first, err := repo.IssueStandaloneRemovalGrant(
		context.Background(), "SRL-1", "usr_partnerless", "dev_android", "jti-1", now,
	)
	require.NoError(t, err)
	assert.Equal(t, "uninstall_detected", first.Action)
	assert.Equal(t, "SRL-1", first.RequestID)
	assert.True(t, first.GrantExpiresAt.After(now))

	_, err = repo.IssueStandaloneRemovalGrant(
		context.Background(), "SRL-2", "usr_partnerless", "dev_android", "jti-2", now.Add(time.Minute),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already active")
}

func TestIssueStandaloneRemovalGrant_AllowsAfterExpiry(t *testing.T) {
	repo, _ := newRepo(t)
	now := time.Now().UTC()

	_, err := repo.IssueStandaloneRemovalGrant(
		context.Background(), "SRL-3", "usr_again", "dev_android", "jti-3", now,
	)
	require.NoError(t, err)

	_, err = repo.IssueStandaloneRemovalGrant(
		context.Background(), "SRL-4", "usr_again", "dev_android", "jti-4", now.Add(standaloneRemovalGrantWindow+time.Minute),
	)
	require.NoError(t, err)
}
