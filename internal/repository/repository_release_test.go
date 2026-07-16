package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelease_CreateAndGet(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	before, err := repo.GetModelReleases(ctx)
	require.NoError(t, err)

	err = repo.CreateModelRelease(ctx, "rel_t", "artifact-v9", "all", "/p", "sha", "contract", 0.72, map[string]any{"x": 1})
	require.NoError(t, err)

	after, err := repo.GetModelReleases(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(before)+1, len(after))
}
