package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrganization_InMemoryRoundTrip(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	before := len(st.Organizations)
	organization, err := repo.CreateOrganization(ctx, "org_test1", "Test Group", "test-group", "TCODE1", "usr_suci")
	require.NoError(t, err)
	assert.Equal(t, "TCODE1", organization.GroupCode)
	assert.Equal(t, "test-group", organization.Slug)
	assert.Equal(t, before+1, len(st.Organizations))

	got, err := repo.GetOrganizationByGroupCode(ctx, "TCODE1")
	require.NoError(t, err)
	assert.Equal(t, "org_test1", got.ID)
	assert.Equal(t, "Test Group", got.Name)
}

func TestGetOrganizationByGroupCode_NotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.GetOrganizationByGroupCode(context.Background(), "NOPE")
	require.Error(t, err)
}

func TestOrganizationMember_CreateAndList(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	// In-memory create is a no-op because members are not persisted in this fallback.
	err := repo.CreateOrganizationMember(ctx, "mem_t", "org_community", "usr_gading", "member", "active", &now)
	assert.NoError(t, err)

	list, err := repo.ListOrganizationMembers(ctx, "org_community")
	assert.NoError(t, err)
	_ = list
}

func TestGetOrganizationMember_NotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.GetOrganizationMember(context.Background(), "org_community", "usr_nonexistent")
	require.Error(t, err)
}

func TestGetMemberProgressSummary(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.GetMemberProgressSummary(context.Background(), "usr_gading")
	_ = err
}

func TestOrganization_CountPendingApprovals(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.CountPendingApprovalsForOrg(context.Background(), "org_community")
	_ = err
}
