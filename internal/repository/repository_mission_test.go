package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertMission_CreatesAndUpdates(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	date := "2026-06-19"

	mission, points, err := repo.UpsertMission(
		ctx, "usr_gading", date, time.Time{}, time.Time{}, 1, true, 10,
	)
	require.NoError(t, err)
	assert.True(t, mission.Mission1)
	assert.Equal(t, 30, points)

	updated, points, err := repo.UpsertMission(
		ctx, "usr_gading", date, time.Time{}, time.Time{}, 3, true, 10,
	)
	require.NoError(t, err)
	assert.True(t, updated.Mission1)
	assert.True(t, updated.Mission3)
	assert.False(t, updated.Mission2)
	assert.Equal(t, 40, points)

	_, repeatedPoints, err := repo.UpsertMission(
		ctx, "usr_gading", date, time.Time{}, time.Time{}, 3, true, 10,
	)
	require.NoError(t, err)
	assert.Equal(t, points, repeatedPoints)

	undone, undonePoints, err := repo.UpsertMission(
		ctx, "usr_gading", date, time.Time{}, time.Time{}, 3, false, 10,
	)
	require.NoError(t, err)
	assert.False(t, undone.Mission3)
	assert.Equal(t, 30, undonePoints)
}
