package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupport_CreateCaseAndDataRequest(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	before, err := repo.GetSupportCases(ctx)
	require.NoError(t, err)

	err = repo.CreateSupportCase(ctx, "case_t", "usr_gading", "judul", "device_recovery", "normal")
	require.NoError(t, err)

	after, err := repo.GetSupportCases(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(before)+1, len(after))

	err = repo.CreateDataRequest(ctx, "dr_t", "usr_gading", "export")
	require.NoError(t, err)
	requests, err := repo.GetDataRequests(ctx, "usr_gading")
	require.NoError(t, err)
	assert.NotEmpty(t, requests)
}
