package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken_CreateGetRevoke(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Hour)
	require.NoError(t, repo.CreateRefreshToken(ctx, "rt_1", "usr_gading", "hash1", nil, expiresAt))

	refreshTokenID, userID, _, err := repo.GetActiveRefreshToken(ctx, "hash1")
	require.NoError(t, err)
	assert.Equal(t, "rt_1", refreshTokenID)
	assert.Equal(t, "usr_gading", userID)

	require.NoError(t, repo.RevokeRefreshTokenByID(ctx, "rt_1"))
	_, _, _, err = repo.GetActiveRefreshToken(ctx, "hash1")
	assert.Error(t, err)
}

func TestRefreshToken_UnknownHashFails(t *testing.T) {
	repo, _ := newRepo(t)
	_, _, _, err := repo.GetActiveRefreshToken(context.Background(), "nope")
	assert.Error(t, err)
}

func TestRefreshToken_RevokeByHash(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_ = repo.CreateRefreshToken(ctx, "rt_h", "usr_gading", "hashh", nil, time.Now().Add(time.Hour))
	assert.NoError(t, repo.RevokeRefreshToken(ctx, "hashh"))
}
