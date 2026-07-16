package service

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newRepo(t *testing.T) (*repository.Repository, *store.Store) {
	t.Helper()
	st := store.NewSeeded()
	return repository.New(nil, st), st
}

func testCfg() config.Config {
	return config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please-32bytes!",
		JWTAccessTTL: time.Hour, JWTRefreshTTL: 720 * time.Hour,
		PublicWebBaseURL: "http://localhost:3000",
	}
}

// --- AccountabilityService ---

func TestAccountability_CreateApprovalRequestAndResolve(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewAccountabilityService(repo, testCfg(), NewWhatsAppService(testCfg(), zap.NewNop()), zap.NewNop())
	ctx := context.Background()

	err := svc.CreateApprovalRequest(ctx, "usr_gading", "dev_android", "pl_active", "pause_protection", "alasan", 30)
	require.NoError(t, err)

	list, err := svc.GetApprovalRequests(ctx, "usr_gading")
	require.NoError(t, err)
	require.NotEmpty(t, list)

	err = svc.ResolveApprovalAsPartner(ctx, list[0].ID, "approved", "usr_suci")
	assert.NoError(t, err)
}

// --- MissionService ---

func TestMission_GetTodayEmptyThenUpdate(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewMissionService(repo, zap.NewNop())
	ctx := context.Background()

	// New user -> empty mission for today.
	m, err := svc.GetToday(ctx, "usr_newbie")
	require.NoError(t, err)
	assert.Equal(t, "usr_newbie", m.UserID)
	assert.False(t, m.Mission1)

	// Update mission 2 -> true.
	m2, err := svc.UpdateMission(ctx, "usr_newbie", 2, true)
	require.NoError(t, err)
	assert.True(t, m2.Mission2)
}

// --- OrganizationService ---

func TestOrganization_CreateAndJoinByCode(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewOrganizationService(repo, zap.NewNop())
	ctx := context.Background()

	org, err := svc.Create(ctx, "Kelas TI 2024", "usr_suci")
	require.NoError(t, err)
	assert.Equal(t, "Kelas TI 2024", org.Name)
	assert.NotEmpty(t, org.GroupCode)
	assert.Equal(t, 1, org.Members)

	// A member joins by the generated group code.
	joined, err := svc.JoinByCode(ctx, org.GroupCode, "usr_gading")
	require.NoError(t, err)
	assert.Equal(t, org.ID, joined.ID)

	// Invalid code fails.
	_, err = svc.JoinByCode(ctx, "NOPE", "usr_gading")
	require.Error(t, err)
}

func TestOrganization_GetAnalytics(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewOrganizationService(repo, zap.NewNop())
	ctx := context.Background()
	org, err := svc.Create(ctx, "Grup Anal", "usr_suci")
	require.NoError(t, err)

	a, err := svc.GetAnalytics(ctx, org.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, a.TotalMembers, 1) // creator (+ possibly seeded) is a member
	assert.Len(t, a.WeeklyBlockTrend, 7)
}

// --- AdminService ---

func TestAdmin_EducationModuleCRUD(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewAdminService(repo, zap.NewNop())
	ctx := context.Background()

	before, err := svc.GetEducationModules(ctx)
	require.NoError(t, err)

	err = svc.CreateEducationModule(ctx, model.EducationModule{
		ID: "mod_test", Slug: "test-module", Title: "Test Module", Summary: "x",
		BodyMarkdown: "## x", EstimatedMinutes: 5, Status: "published",
	})
	require.NoError(t, err)

	after, err := svc.GetEducationModules(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(after), len(before))
}

func TestAdmin_EmergencyKeyGenerateAndValidate(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewAdminService(repo, zap.NewNop())
	ctx := context.Background()

	key, err := svc.GenerateEmergencyKey(ctx, "usr_nasywa")
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Wrong key fails.
	err = svc.ValidateEmergencyKey(ctx, "wrong-key", "dev_android")
	assert.Error(t, err)
}

// --- DeviceService ---

func TestDevice_CreateUpdateHeartbeat(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewDeviceService(repo, zap.NewNop())
	ctx := context.Background()

	mv, rv := "artifact-v0.3.1", "ruleset-2026.05.1"
	d, err := svc.CreateDevice(ctx, "usr_gading", "windows", "Gading PC", "1.0.0", "Windows 11", &mv, &rv)
	require.NoError(t, err)
	assert.Equal(t, "windows", d.Platform)

	d2, err := svc.UpdateDevice(ctx, d.ID, "Gading PC2", "1.0.1", "Windows 11", "active", mv, rv)
	require.NoError(t, err)
	assert.Equal(t, "Gading PC2", d2.Label)

	err = svc.RecordHeartbeat(ctx, d.ID)
	assert.NoError(t, err)
}

// --- SupportService ---

func TestSupport_CreateSupportCaseAndDataRequest(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewSupportService(repo, zap.NewNop())
	ctx := context.Background()

	err := svc.CreateSupportCase(ctx, "usr_gading", "tidak bisa login", "device_recovery", "normal")
	assert.NoError(t, err)

	err = svc.CreateDataRequest(ctx, "usr_gading", "export")
	assert.NoError(t, err)

	cases, err := svc.GetSupportCases(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, cases)

	reqs, err := svc.GetDataRequests(ctx, "usr_gading")
	require.NoError(t, err)
	assert.NotEmpty(t, reqs)
}

// --- WhatsAppService ---

func TestWhatsApp_DemoModeIsNoOp(t *testing.T) {
	cfg := testCfg() // NotificationMode defaults to "" -> but service checks "demo"
	svc := NewWhatsAppService(cfg, zap.NewNop())
	// demo mode OR empty phone -> no error (no-op)
	err := svc.SendSingleApproval(context.Background(), "", ApprovalSummary{MemberName: "x", Action: "pause"})
	assert.NoError(t, err)

	err = svc.SendApprovalBatch(context.Background(), "", []ApprovalSummary{{}})
	assert.NoError(t, err)
}

// --- Service edge/error branches ---

func TestAccountability_ResolveByToken_Invalid(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewAccountabilityService(repo, testCfg(), NewWhatsAppService(testCfg(), zap.NewNop()), zap.NewNop())
	err := svc.ResolveByToken(context.Background(), "no-such-token", "approved")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token tidak valid")
}

func TestAccountability_VerifyQuickToken_Invalid(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewAccountabilityService(repo, testCfg(), NewWhatsAppService(testCfg(), zap.NewNop()), zap.NewNop())
	_, err := svc.VerifyQuickToken(context.Background(), "no-such-token")
	require.Error(t, err)
}

func TestOrganization_RemoveMember_NotFound(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewOrganizationService(repo, zap.NewNop())
	err := svc.RemoveMember(context.Background(), "org_community", "usr_nonexistent", "usr_suci")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "anggota tidak ditemukan")
}

func TestOrganization_GetByID_NotFound(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewOrganizationService(repo, zap.NewNop())
	_, err := svc.GetByID(context.Background(), "org_nonexistent")
	require.Error(t, err)
}

func TestOrganization_GetByUserID_None(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewOrganizationService(repo, zap.NewNop())
	_, err := svc.GetByUserID(context.Background(), "usr_nobody")
	require.Error(t, err)
}

func TestAdmin_ReleasesCreateAndGet(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewAdminService(repo, zap.NewNop())
	ctx := context.Background()

	beforeModel, _ := svc.GetModelReleases(ctx)
	err := svc.CreateModelRelease(ctx, "artifact-v9", "all", "/p", "sha", "contract", 0.72, map[string]any{"x": 1})
	require.NoError(t, err)
	afterModel, _ := svc.GetModelReleases(ctx)
	assert.Equal(t, len(beforeModel)+1, len(afterModel))

	err = svc.CreateRulesetRelease(ctx, "ruleset-v9", "/p", "sha", map[string]any{"rules": 5})
	require.NoError(t, err)
	_, err = svc.GetRulesetReleases(ctx)
	assert.NoError(t, err)

	err = svc.CreateNetworkRulesetRelease(ctx, "net-v9", "/p", "sha", map[string]any{"domains": 0})
	require.NoError(t, err)
	_, err = svc.GetNetworkRulesets(ctx)
	assert.NoError(t, err)
}

func TestDevice_CreateMissingOptionalVersions(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewDeviceService(repo, zap.NewNop())
	d, err := svc.CreateDevice(context.Background(), "usr_gading", "android", "Phone", "1.0.0", "Android 15", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "android", d.Platform)
}

func TestMission_GetTodaySeededUser(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewMissionService(repo, zap.NewNop())
	m, err := svc.GetToday(context.Background(), "usr_gading")
	require.NoError(t, err)
	// Seeded user has a mission for today's date.
	assert.Equal(t, "usr_gading", m.UserID)
}
