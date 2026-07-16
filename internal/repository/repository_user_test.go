package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserByEmail_FoundAndNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	user, ok := repo.UserByEmail(ctx, "gading@gmail.com")
	require.True(t, ok)
	assert.Equal(t, "usr_gading", user.ID)

	_, ok = repo.UserByEmail(ctx, "nobody@example.com")
	assert.False(t, ok)
}

func TestUserByID_FoundAndNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	user, ok := repo.UserByID(ctx, "usr_gading")
	require.True(t, ok)
	assert.Equal(t, "Gading", user.DisplayName)

	_, ok = repo.UserByID(ctx, "usr_nonexistent")
	assert.False(t, ok)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.CreateUser(context.Background(), "usr_dup", "gading@gmail.com", "Dup")
	assert.Error(t, err)
}

func TestCreateUser_New(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	user, err := repo.CreateUser(ctx, "usr_new1", "new1@example.com", "New1")
	require.NoError(t, err)
	assert.Equal(t, "new1@example.com", user.Email)

	got, ok := repo.UserByEmail(ctx, "new1@example.com")
	require.True(t, ok)
	assert.Equal(t, "usr_new1", got.ID)
}
