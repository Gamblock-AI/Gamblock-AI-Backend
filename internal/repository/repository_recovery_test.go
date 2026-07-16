package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReflections_OnlyOwnUser(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.CreateReflection(ctx, "usr_gading", "refleksi gading", "baik")
	require.NoError(t, err)

	got, err := repo.GetReflections(ctx, "usr_gading")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(got), 1)

	other, err := repo.GetReflections(ctx, "usr_dery")
	require.NoError(t, err)
	for _, entry := range other {
		assert.NotEqual(t, "usr_gading", entry.UserID, "must not leak other users' reflections")
	}
}
