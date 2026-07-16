package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateApprovalRequestWithToken_AndLookup(t *testing.T) {
	repo, st := newRepo(t)
	request, err := repo.CreateApprovalRequestWithToken(
		context.Background(), "APR-T1", "usr_gading", "dev_android", "pl_active",
		"pause_protection", "alasan", 15, time.Now().Add(15*time.Minute), "hash123",
	)
	require.NoError(t, err)
	assert.Equal(t, "pause_protection", request.Action)

	got, ok := st.GetTokenMapping("hash123")
	require.True(t, ok)
	assert.Equal(t, "APR-T1", got.ID)
}
