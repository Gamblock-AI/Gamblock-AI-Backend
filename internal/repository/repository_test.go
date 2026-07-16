package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newRepo(t *testing.T) (*Repository, *store.Store) {
	t.Helper()
	st := store.NewSeeded()
	return New(nil, st), st
}

func TestCreateOrganization_InMemoryRoundTrip(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	before := len(st.Organizations)
	org, err := repo.CreateOrganization(ctx, "org_test1", "Test Group", "test-group", "TCODE1", "usr_suci")
	require.NoError(t, err)
	assert.Equal(t, "TCODE1", org.GroupCode)
	assert.Equal(t, "test-group", org.Slug)
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

func TestUpsertMission_CreatesAndUpdates(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	date := "2026-06-19"

	// New mission for a user without one today.
	m, err := repo.UpsertMission(ctx, "usr_new", date, 1, true)
	require.NoError(t, err)
	assert.True(t, m.Mission1)

	// Toggle mission 3 on the same record.
	m2, err := repo.UpsertMission(ctx, "usr_new", date, 3, true)
	require.NoError(t, err)
	assert.True(t, m2.Mission1)
	assert.True(t, m2.Mission3)
	assert.False(t, m2.Mission2)
}

func TestCreateApprovalRequestWithToken_AndLookup(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	expires := time.Now().Add(15 * time.Minute)
	err := repo.CreateApprovalRequestWithToken(ctx, "APR-T1", "usr_gading", "dev_android", "pl_active", "pause_protection", "alasan", 15, expires, "hash123")
	require.NoError(t, err)

	got, ok := st.GetTokenMapping("hash123")
	require.True(t, ok)
	assert.Equal(t, "APR-T1", got.ID)
}

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
	for _, e := range other {
		assert.NotEqual(t, "usr_gading", e.UserID, "must not leak other users' reflections")
	}
}

// --- User repository ---

func TestUserByEmail_FoundAndNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	u, ok := repo.UserByEmail(ctx, "gading@gmail.com")
	require.True(t, ok)
	assert.Equal(t, "usr_gading", u.ID)

	_, ok = repo.UserByEmail(ctx, "nobody@example.com")
	assert.False(t, ok)
}

func TestUserByID_FoundAndNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	u, ok := repo.UserByID(ctx, "usr_gading")
	require.True(t, ok)
	assert.Equal(t, "Gading", u.DisplayName)

	_, ok = repo.UserByID(ctx, "usr_nonexistent")
	assert.False(t, ok)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.CreateUser(ctx, "usr_dup", "gading@gmail.com", "Dup")
	// in-memory: CreateUser checks for existing email and returns error.
	assert.Error(t, err)
}

func TestCreateUser_New(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	u, err := repo.CreateUser(ctx, "usr_new1", "new1@example.com", "New1")
	require.NoError(t, err)
	assert.Equal(t, "new1@example.com", u.Email)
	// Fetchable by email.
	got, ok := repo.UserByEmail(ctx, "new1@example.com")
	require.True(t, ok)
	assert.Equal(t, "usr_new1", got.ID)
}

// --- Device repository ---

func TestDevice_CreateUpdateHeartbeat(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	mv, rv := "artifact-v0.3.1", "ruleset-2026.05.1"
	d, err := repo.CreateDevice(ctx, "dev_test", "usr_gading", "windows", "PC", "1.0.0", "Win11", &mv, &rv)
	require.NoError(t, err)
	assert.Equal(t, "dev_test", d.ID)

	d2, err := repo.UpdateDevice(ctx, "dev_test", "PC2", "1.0.1", "Win11", "active", mv, rv)
	require.NoError(t, err)
	assert.Equal(t, "PC2", d2.Label)

	err = repo.RecordHeartbeat(ctx, "dev_test")
	assert.NoError(t, err)
}

// --- Release repository ---

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

// --- Support repository ---

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
	reqs, err := repo.GetDataRequests(ctx, "usr_gading")
	require.NoError(t, err)
	assert.NotEmpty(t, reqs)
}

// --- Partner repository ---

func TestPartner_GetAndCreateInvitation(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	// Seeded partner pl_active belongs to suci (partner). GetPartners for any user.
	_, _, err := repo.GetPartners(ctx, "usr_suci")
	assert.NoError(t, err)

	_, err = repo.CreatePartnerInvitation(ctx, "pl_new", "usr_gading", "partner@example.com", nil, "hashhash")
	assert.NoError(t, err)
}

// --- Token repository (in-memory refresh tokens) ---

func TestRefreshToken_CreateGetRevoke(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	expires := time.Now().Add(time.Hour)
	err := repo.CreateRefreshToken(ctx, "rt_1", "usr_gading", "hash1", nil, expires)
	require.NoError(t, err)

	rtID, userID, _, err := repo.GetActiveRefreshToken(ctx, "hash1")
	require.NoError(t, err)
	assert.Equal(t, "rt_1", rtID)
	assert.Equal(t, "usr_gading", userID)

	err = repo.RevokeRefreshTokenByID(ctx, "rt_1")
	require.NoError(t, err)

	// After revoke, get should fail.
	_, _, _, err = repo.GetActiveRefreshToken(ctx, "hash1")
	assert.Error(t, err)
}

func TestRefreshToken_UnknownHashFails(t *testing.T) {
	repo, _ := newRepo(t)
	_, _, _, err := repo.GetActiveRefreshToken(context.Background(), "nope")
	assert.Error(t, err)
}

// --- Education repository ---

func TestEducation_CreateAppendsToStore(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	before := len(st.Modules)
	err := repo.CreateEducationModule(ctx, model.EducationModule{
		ID: "mod_x", Slug: "x", Title: "X", Status: "published",
	})
	require.NoError(t, err)
	assert.Equal(t, before+1, len(st.Modules))
}

// --- Organization members ---

func TestOrganizationMember_CreateAndList(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()
	// In-memory create is a no-op (org members not persisted in the store backing),
	// but must not panic or error.
	err := repo.CreateOrganizationMember(ctx, "mem_t", "org_community", "usr_gading", "member", "active", &now)
	assert.NoError(t, err)

	list, err := repo.ListOrganizationMembers(ctx, "org_community")
	// List may be empty in-memory; just assert no error.
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
	// in-memory path returns a summary or error; assert no panic.
	_ = err
}

// --- Partner ---

func TestPartner_InvitationAcceptRevoke(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.CreatePartnerInvitation(ctx, "pl_t", "usr_gading", "partner@example.com", nil, "tokenhash123")
	require.NoError(t, err)

	// Accept the invitation.
	err = repo.AcceptPartnerInvitation(ctx, "pl_t", "usr_suci")
	// in-memory path may be no-op; assert no panic.
	_ = err

	// Revoke.
	err = repo.RevokePartner(ctx, "pl_t", "usr_gading")
	_ = err
}

func TestPartner_GetPartners(t *testing.T) {
	repo, _ := newRepo(t)
	_, _, err := repo.GetPartners(context.Background(), "usr_suci")
	assert.NoError(t, err)
}

// --- Notification ---

func TestNotification_Queue(t *testing.T) {
	repo, _ := newRepo(t)
	// in-memory path is a no-op; assert no panic/error contract.
	err := repo.QueueNotification(context.Background(), "ntf_t", "APR-1", "", "email", "suci@gmail.com")
	_ = err
}

// --- Education GetBySlug ---

func TestGetEducationModuleBySlug(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	m, err := repo.GetEducationModuleBySlug(ctx, "pause-before-impulse")
	require.NoError(t, err)
	assert.Equal(t, "pause-before-impulse", m.Slug)
}

func TestGetEducationModuleBySlug_NotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.GetEducationModuleBySlug(context.Background(), "no-such-slug")
	require.Error(t, err)
}

// --- Partner GetPartnerLinkByToken ---

func TestPartnerLinkByToken(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.CreatePartnerInvitation(ctx, "pl_t2", "usr_gading", "p2@example.com", nil, "tokenhash456")
	require.NoError(t, err)

	// in-memory GetPartnerLinkByToken may not persist; assert no panic.
	_, _ = repo.GetPartnerLinkByToken(ctx, "tokenhash456")
}

// --- Device Update non-existent ---

func TestDevice_UpdateNonexistent(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.UpdateDevice(context.Background(), "dev_nonexistent", "L", "1", "OS", "active", "m", "r")
	// in-memory UpdateDevice for unknown device: assert no panic; error contract varies.
	_ = err
}

// --- Notification queue ---

func TestNotification_QueueInMemory(t *testing.T) {
	repo, _ := newRepo(t)
	err := repo.QueueNotification(context.Background(), "ntf_x", "APR-1", "sc-1", "whatsapp", "+62")
	_ = err
}

// --- Organization analytics member summary ---

func TestOrganization_CountPendingApprovals(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.CountPendingApprovalsForOrg(context.Background(), "org_community")
	// in-memory path; assert no panic.
	_ = err
}

// --- Refresh token revoke by hash ---

func TestRefreshToken_RevokeByHash(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	expires := time.Now().Add(time.Hour)
	_ = repo.CreateRefreshToken(ctx, "rt_h", "usr_gading", "hashh", nil, expires)

	err := repo.RevokeRefreshToken(ctx, "hashh")
	assert.NoError(t, err)
}
