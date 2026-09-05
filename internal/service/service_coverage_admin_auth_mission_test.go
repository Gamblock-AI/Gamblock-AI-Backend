package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

func TestAdminService_CoverageForReadAuditSocialAndOverviewFlows(t *testing.T) {
	repo, _ := newRepo(t)
	cfg := testCfg()
	svc := NewAdminService(repo, cfg, NewWhatsAppService(cfg, zap.NewNop()), zap.NewNop())
	ctx := context.Background()

	_, err := svc.PlatformAnalytics(ctx, 7)
	require.Error(t, err)
	_, err = svc.PlatformAnalytics(ctx, 14)
	require.NoError(t, err)

	allLinks, err := svc.SiteSocialLinks(ctx)
	require.NoError(t, err)
	assert.NotNil(t, allLinks)
	publicLinks, err := svc.PublicSocialLinks(ctx)
	require.NoError(t, err)
	assert.NotNil(t, publicLinks)

	instagramURL := "https://instagram.com/gamblockai"
	facebookURL := "https://facebook.com/profile.php?id=61591544143202"
	links, err := svc.ReplaceSiteSocialLinks(ctx, "usr_nasywa", "coverage fixture", []model.SiteSocialLink{
		{Platform: " Instagram ", Label: "Instagram", URL: &instagramURL, Enabled: true},
		{Platform: "facebook", Label: "Facebook", URL: &facebookURL, Enabled: true},
		{Platform: "tiktok", Label: "TikTok", Enabled: false},
	})
	require.NoError(t, err)
	assert.Len(t, links, 3)
	publicLinks, err = svc.PublicSocialLinks(ctx)
	require.NoError(t, err)
	assert.Len(t, publicLinks, 2)

	tooMany := make([]model.SiteSocialLink, len(socialHosts)+1)
	for i := range tooMany {
		tooMany[i] = model.SiteSocialLink{Platform: "instagram", Label: "duplicate"}
	}
	_, err = svc.ReplaceSiteSocialLinks(ctx, "usr_nasywa", "invalid", tooMany)
	require.Error(t, err)
	assert.True(t, allowedSocialHost("subdomain.instagram.com", "instagram.com") == false)
	assert.True(t, allowedSocialHost("instagram", "cdn.instagram.com"))

	err = svc.RecordAudit(ctx, "missing", "manual", "account", "x", "reason", nil)
	require.Error(t, err)
	err = svc.RecordAudit(ctx, "usr_nasywa", "manual", "account", "usr_gading", "reason", map[string]any{"source": "test"})
	require.NoError(t, err)
	events, err := svc.AuditEvents(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	paged, err := svc.AuditEventsPaginated(ctx, model.PaginationQuery{Page: 1, Limit: 5})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, paged.TotalCount, 1)

	accounts, err := svc.Accounts(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, accounts)
	filtered, err := svc.AccountsPaginated(ctx, model.PaginationQuery{Role: model.RoleAdmin, Status: "active", Query: "nasywa"})
	require.NoError(t, err)
	assert.NotEmpty(t, filtered.Items)

	_, err = svc.Overview(ctx, model.RoleUser)
	require.Error(t, err)
	overview, err := svc.Overview(ctx, model.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, model.RoleAdmin, overview.Role)
	portal, err := svc.GetPortalOverview(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, portal.HealthyDevicesPercent, 0)

	_, err = svc.GenerateEmergencyKey(ctx, "usr_nasywa")
	require.Error(t, err)
}

func TestAdminService_CoverageForEmergencyRequestReviewAndAuthorization(t *testing.T) {
	repo, _ := newRepo(t)
	cfg := testCfg()
	svc := NewAdminService(repo, cfg, NewWhatsAppService(cfg, zap.NewNop()), zap.NewNop())
	deviceSvc := NewDeviceService(repo, cfg, zap.NewNop())
	ctx := context.Background()

	device, err := deviceSvc.CreateDevice(ctx, "usr_dery", "admin-coverage-device", "android", "Admin coverage", "1.0", "Android", nil, nil)
	require.NoError(t, err)

	_, err = svc.GetCurrentEmergencyKeyRequest(ctx, "usr_gading", device.ID)
	require.Error(t, err)
	_, err = svc.RequestEmergencyKey(ctx, "usr_gading", device.ID)
	require.Error(t, err)

	request, err := svc.RequestEmergencyKey(ctx, "usr_dery", device.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", request.Status)
	current, err := svc.GetCurrentEmergencyKeyRequest(ctx, "usr_dery", device.ID)
	require.NoError(t, err)
	assert.Equal(t, request.ID, current.ID)
	pending, err := svc.GetPendingEmergencyKeyRequests(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, pending)
	paged, err := svc.GetPendingEmergencyKeyRequestsPaginated(ctx, model.PaginationQuery{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, paged.TotalCount, 1)

	reviewed, err := svc.ReviewEmergencyKeyRequest(ctx, request.ID, "usr_nasywa")
	require.NoError(t, err)
	assert.Equal(t, "reviewed", reviewed.Status)
	_, err = svc.RequestEmergencyKey(ctx, "usr_dery", device.ID)
	require.Error(t, err, "reviewed request must remain active")
}

func TestAuthService_CoverageForIdentityAndReauthentication(t *testing.T) {
	svc, st := newAuthSvc(t)
	ctx := context.Background()

	role, active := svc.ActiveIdentity(ctx, "usr_gading")
	assert.Equal(t, model.RoleUser, role)
	assert.True(t, active)
	_, active = svc.ActiveIdentity(ctx, "missing")
	assert.False(t, active)
	assert.True(t, svc.HasVerifiedPhone(ctx, "usr_gading"))
	assert.False(t, svc.HasVerifiedPhone(ctx, "missing"))

	_, err := svc.Reauthenticate(ctx, "usr_gading", "wrong-password")
	require.Error(t, err)
	refreshed, err := svc.Reauthenticate(ctx, "usr_gading", "password")
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)
	assert.NotEmpty(t, refreshed.RefreshToken)

	_, err = svc.issueToken(model.User{ID: "usr_gading", Email: "gading@gmail.com", Role: model.RoleUser})
	require.NoError(t, err)

	// An account that is pending an initial password change cannot reauthenticate.
	repo := repository.New(nil, st)
	user, ok := repo.UserByID(ctx, "usr_gading")
	require.True(t, ok)
	user.MustChangePassword = true
	st.Lock()
	for index := range st.Users {
		if st.Users[index].ID == user.ID {
			st.Users[index].MustChangePassword = true
		}
	}
	st.Unlock()
	_, err = svc.Reauthenticate(ctx, user.ID, "password")
	require.Error(t, err)
}

func TestMissionService_CoverageForCustomMissionStateTransitions(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewMissionServiceWithConfig(repo, testCfg(), zap.NewNop())
	ctx := context.Background()

	_, err := svc.CreateCustomMission(ctx, "usr_dery", "   ")
	require.ErrorIs(t, err, ErrCustomMissionInvalid)
	_, err = svc.CreateCustomMission(ctx, "usr_dery", strings.Repeat("x", customMissionTitleMaxLength+1))
	require.ErrorIs(t, err, ErrCustomMissionInvalid)

	created, err := svc.CreateCustomMission(ctx, "usr_dery", "  Batasi waktu bermain  ")
	require.NoError(t, err)
	require.Len(t, created.Tasks, 5)
	var custom model.DailyMissionTask
	for _, task := range created.Tasks {
		if task.Source == "custom" {
			custom = task
		}
	}
	require.NotEmpty(t, custom.ID)
	assert.Equal(t, "Batasi waktu bermain", custom.Title)

	updated, err := svc.UpdateCustomMission(ctx, "usr_dery", custom.ID, "Batasi waktu belajar")
	require.NoError(t, err)
	assert.Equal(t, "Batasi waktu belajar", findMissionTask(updated.Tasks, custom.ID).Title)

	claimed, err := svc.ClaimMissionByID(ctx, "usr_dery", custom.ID)
	require.NoError(t, err)
	assert.True(t, findMissionTask(claimed.Tasks, custom.ID).Completed)

	_, err = svc.UpdateCustomMission(ctx, "usr_dery", "missing-custom", "new title")
	require.ErrorIs(t, err, ErrCustomMissionNotEditable)

	second, err := svc.CreateCustomMission(ctx, "usr_dery", "Mission yang dihapus")
	require.NoError(t, err)
	var secondID string
	for _, task := range second.Tasks {
		if task.Source == "custom" {
			secondID = task.ID
		}
	}
	deleted, err := svc.DeleteCustomMission(ctx, "usr_dery", secondID)
	require.NoError(t, err)
	assert.Nil(t, findMissionTask(deleted.Tasks, secondID))
	_, err = svc.DeleteCustomMission(ctx, "usr_dery", secondID)
	require.ErrorIs(t, err, ErrCustomMissionNotEditable)
}

func TestPushService_CoverageForSubscriptionLifecycle(t *testing.T) {
	repo, _ := newRepo(t)
	cfg := testCfg()
	svc := NewPushService(repo, cfg, zap.NewNop())
	ctx := context.Background()
	ua := "coverage-browser"

	first, err := svc.UpsertSubscription(ctx, "usr_gading", "https://push.example/one", "p256dh-one", "auth-one", &ua)
	require.NoError(t, err)
	second, err := svc.UpsertSubscription(ctx, "usr_gading", first.Endpoint, "p256dh-two", "auth-two", nil)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	sent, err := svc.SendToUser(ctx, "usr_gading", PushPayload{Title: "Reminder", Body: "Body", URL: "/id/recovery"})
	require.NoError(t, err)
	assert.Equal(t, 1, sent, "unconfigured VAPID delivery is a safe no-op")
	assert.False(t, svc.configured())

	require.NoError(t, svc.DeleteSubscription(ctx, "other-user", first.Endpoint))
	require.NoError(t, svc.DeleteSubscription(ctx, "usr_gading", first.Endpoint))
	sent, err = svc.SendToUser(ctx, "usr_gading", PushPayload{Title: "No subscriptions"})
	require.NoError(t, err)
	assert.Zero(t, sent)
}

func TestDeepSeekService_CoverageForValidationAndResponseHelpers(t *testing.T) {
	assert.ErrorIs(t, validateTranslateInput(nil, "id", "en"), ErrTranslationEmptyText)
	assert.ErrorIs(t, validateTranslateInput([]string{"x"}, "fr", "en"), ErrInvalidLanguage)
	assert.ErrorIs(t, validateTranslateInput([]string{"x"}, "id", "id"), ErrSameLanguage)
	assert.ErrorIs(t, validateTranslateInput([]string{"x"}, "id", "fr"), ErrInvalidLanguage)
	assert.ErrorIs(t, validateTranslateInput([]string{strings.Repeat("x", maxCharsPerText+1)}, "id", "en"), ErrTextTooLong)
	tooMany := make([]string, maxTextsPerRequest+1)
	assert.ErrorIs(t, validateTranslateInput(tooMany, "id", "en"), ErrTranslationMaxTexts)
	assert.NoError(t, validateTranslateInput([]string{"  hello  ", ""}, "en", "id"))

	assert.Equal(t, "hello"+translationBatchSep+"world", buildBatchUserMessage([]string{"hello", "world"}))
	assert.Contains(t, buildBatchSystemPrompt("id", "en"), "Bahasa Indonesia")
	assert.Equal(t, "English", languageNameCode("en"))
	assert.Equal(t, "xx", languageNameCode("xx"))

	parsed, err := parseBatchResponse("satu"+translationBatchSep+"two", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"satu", "two"}, parsed)
	_, err = parseBatchResponse("only-one", 2)
	require.Error(t, err)

	svc := NewDeepSeekService(testCfg(), zap.NewNop())
	_, err = svc.BatchTranslate(context.Background(), []string{"hello"}, "id", "id")
	require.ErrorIs(t, err, ErrSameLanguage)
}

func TestReminderScheduler_CoverageForConstructionAndCancellation(t *testing.T) {
	repo, _ := newRepo(t)
	push := NewPushService(repo, testCfg(), zap.NewNop())
	scheduler := NewReminderScheduler(repo, push, zap.NewNop())
	assert.NotNil(t, scheduler)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scheduler.Run(ctx)
	scheduler.Tick(context.Background())
	assert.Equal(t, 19, func() int { hour, _ := parseReminderTime("invalid"); return hour }())
	assert.Equal(t, 19, func() int { hour, _ := parseReminderTime("25:70"); return hour }())
	assert.Equal(t, 7, func() int { hour, _ := parseReminderTime("07:05"); return hour }())
	assert.True(t, sameCalendarDay(time.Now().UTC(), time.Now().UTC(), time.UTC))
}
