package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertMission_CreatesAndUpdates(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	date := "2026-06-19"

	mission, err := repo.UpsertMission(ctx, "usr_new", date, 1, true)
	require.NoError(t, err)
	assert.True(t, mission.Mission1)

	updated, err := repo.UpsertMission(ctx, "usr_new", date, 3, true)
	require.NoError(t, err)
	assert.True(t, updated.Mission1)
	assert.True(t, updated.Mission3)
	assert.False(t, updated.Mission2)
}
