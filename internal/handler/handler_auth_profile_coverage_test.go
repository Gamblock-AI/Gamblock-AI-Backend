package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func TestHandlerAuthProfileCoverage_AuthSessionBranches(t *testing.T) {
	h := newBranchHandler(t)

	login := invokeHandler(t, h, http.MethodPost, "/auth/login", `{"email":"gading@gmail.com","password":"password"}`, nil, nil, h.Login)
	require.Equal(t, http.StatusOK, login.Code)
	loginData := authProfileCoverageEnvelopeData(t, login)
	refreshToken, ok := loginData["refresh_token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, refreshToken)

	refreshed := invokeHandler(t, h, http.MethodPost, "/auth/refresh", `{"refresh_token":"`+refreshToken+`"}`, nil, nil, h.Refresh)
	require.Equal(t, http.StatusOK, refreshed.Code)
	refreshedData := authProfileCoverageEnvelopeData(t, refreshed)
	rotatedRefresh, ok := refreshedData["refresh_token"].(string)
	require.True(t, ok)

	// A rotated refresh token is single-use and must no longer authenticate.
	replayed := invokeHandler(t, h, http.MethodPost, "/auth/refresh", `{"refresh_token":"`+refreshToken+`"}`, nil, nil, h.Refresh)
	assertHandlerError(t, replayed, http.StatusUnauthorized, "invalid_refresh_token")

	logout := invokeHandler(t, h, http.MethodPost, "/auth/logout", `{"refresh_token":"`+rotatedRefresh+`"}`, nil, nil, h.Logout)
	require.Equal(t, http.StatusOK, logout.Code)
	assert.Contains(t, logout.Body.String(), `"revoked":true`)

	missingRefresh := invokeHandler(t, h, http.MethodPost, "/auth/refresh", `{}`, nil, nil, h.Refresh)
	assertHandlerError(t, missingRefresh, http.StatusBadRequest, "refresh_token_required")

	reauth := invokeHandler(t, h, http.MethodPost, "/auth/reauthenticate", `{"password":"password"}`, nil, nil, h.Reauthenticate)
	require.Equal(t, http.StatusOK, reauth.Code)
	badReauth := invokeHandler(t, h, http.MethodPost, "/auth/reauthenticate", `{"password":"wrong-password"}`, nil, nil, h.Reauthenticate)
	assertHandlerError(t, badReauth, http.StatusUnauthorized, "invalid_credentials")
	emptyReauth := invokeHandler(t, h, http.MethodPost, "/auth/reauthenticate", `{}`, nil, nil, h.Reauthenticate)
	assertHandlerError(t, emptyReauth, http.StatusBadRequest, "err_validation")

	registration := invokeHandler(t, h, http.MethodPost, "/auth/register", `{"email":"coverage-auth@example.com","password":"password2","name":"Coverage User","phone":"081234567890","role":"user"}`, nil, nil, h.Register)
	require.Equal(t, http.StatusCreated, registration.Code)
	duplicate := invokeHandler(t, h, http.MethodPost, "/auth/register", `{"email":"coverage-auth@example.com","password":"password2","name":"Coverage User","phone":"081234567890","role":"user"}`, nil, nil, h.Register)
	assertHandlerError(t, duplicate, http.StatusBadRequest, "registration_failed")
	invalidRegistration := invokeHandler(t, h, http.MethodPost, "/auth/register", `{"email":"bad@example.com","password":"short"}`, nil, nil, h.Register)
	assertHandlerError(t, invalidRegistration, http.StatusBadRequest, "validation_failed")

	devLogin := invokeHandler(t, h, http.MethodPost, "/auth/dev-login", `{}`, nil, nil, h.DevLogin)
	assertHandlerError(t, devLogin, http.StatusForbidden, "dev_login_failed")
	resetMissingEmail := invokeHandler(t, h, http.MethodPost, "/auth/password-reset/request", `{}`, nil, nil, h.RequestPasswordReset)
	assertHandlerError(t, resetMissingEmail, http.StatusBadRequest, "email_required")
	resetUnknown := invokeHandler(t, h, http.MethodPost, "/auth/password-reset/request", `{"email":"unknown@example.com"}`, nil, nil, h.RequestPasswordReset)
	require.Equal(t, http.StatusAccepted, resetUnknown.Code)
	assert.Contains(t, resetUnknown.Body.String(), `"accepted":true`)
	resetInvalid := invokeHandler(t, h, http.MethodPost, "/auth/password-reset/confirm", `{"email":"gading@gmail.com","code":"bad","new_password":"password2"}`, nil, nil, h.ConfirmPasswordReset)
	assertHandlerError(t, resetInvalid, http.StatusBadRequest, "password_reset_invalid")

	initialInvalid := invokeHandler(t, h, http.MethodPost, "/auth/first-login/password", `{"token":"bad","new_password":"short"}`, nil, nil, h.CompleteInitialPasswordChange)
	assertHandlerError(t, initialInvalid, http.StatusBadRequest, "initial_password_change_invalid")
}

func TestHandlerAuthProfileCoverage_PhoneVerificationAndPasswordBranches(t *testing.T) {
	r := newPhoneVerificationRouter(t)

	invalidRegister := postJSON(r, "/v1/auth/register", []byte(`{"email":"phone-invalid@example.com","password":"short","name":"Invalid","phone":"081234567890"}`))
	assert.Equal(t, http.StatusBadRequest, invalidRegister.Code)

	register := postJSON(r, "/v1/auth/register", []byte(`{"email":"phone-coverage@example.com","password":"password2","name":"Phone Coverage","phone":"081234567891","role":"user"}`))
	require.Equal(t, http.StatusCreated, register.Code)
	var registerEnvelope envelopeShape
	require.NoError(t, json.Unmarshal(register.Body.Bytes(), &registerEnvelope))
	registerData := registerEnvelope.Data.(map[string]any)
	verificationToken := registerData["verification_token"].(string)

	badCode := postJSON(r, "/v1/auth/phone-verification/verify", []byte(`{"verification_token":"`+verificationToken+`","code":"000000"}`))
	assert.Equal(t, http.StatusBadRequest, badCode.Code)
	assert.Contains(t, badCode.Body.String(), "phone_verification_failed")
	missingToken := postJSON(r, "/v1/auth/phone-verification/verify", []byte(`{"code":"123456"}`))
	assert.Equal(t, http.StatusBadRequest, missingToken.Code)
	assert.Contains(t, missingToken.Body.String(), "err_validation")
	resend := postJSON(r, "/v1/auth/phone-verification/verify/resend", []byte(`{"verification_token":"`+verificationToken+`"}`))
	require.Equal(t, http.StatusOK, resend.Code)

	h := newBranchHandler(t)
	passwordValidation := invokeHandler(t, h, http.MethodPatch, "/profile/password", `{"current_password":"","new_password":"short"}`, nil, nil, h.UpdatePassword)
	assertHandlerError(t, passwordValidation, http.StatusBadRequest, "password_validation_failed")
	wrongCurrent := invokeHandler(t, h, http.MethodPatch, "/profile/password", `{"current_password":"wrong","new_password":"new-password"}`, nil, nil, h.UpdatePassword)
	assertHandlerError(t, wrongCurrent, http.StatusBadRequest, "current_password_invalid")
	reuse := invokeHandler(t, h, http.MethodPatch, "/profile/password", `{"current_password":"password","new_password":"password"}`, nil, nil, h.UpdatePassword)
	assertHandlerError(t, reuse, http.StatusBadRequest, "password_reuse_not_allowed")
	updated := invokeHandler(t, h, http.MethodPatch, "/profile/password", `{"current_password":"password","new_password":"new-password"}`, nil, nil, h.UpdatePassword)
	require.Equal(t, http.StatusOK, updated.Code)
	assert.Contains(t, updated.Body.String(), `"reauth_required":true`)
}

func TestHandlerAuthProfileCoverage_DeviceOwnershipAndGrantBranches(t *testing.T) {
	h := newBranchHandler(t)

	created := invokeHandler(t, h, http.MethodPost, "/devices", `{"client_instance_id":"coverage-client","platform":"windows","label":"Coverage laptop","app_version":"2.0","os_version":"Linux","model_version":"model-2","ruleset_version":"rules-2"}`, nil, nil, h.CreateDevice)
	require.Equal(t, http.StatusCreated, created.Code)
	var createdEnvelope envelopeShape
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &createdEnvelope))
	createdID := createdEnvelope.Data.(map[string]any)["id"].(string)

	updated := invokeHandler(t, h, http.MethodPatch, "/devices/"+createdID, `{"label":"Updated laptop","protection_status":"attention"}`, gin.Params{{Key: "device_id", Value: createdID}}, nil, h.UpdateDevice)
	require.Equal(t, http.StatusOK, updated.Code)
	heartbeat := invokeHandler(t, h, http.MethodPost, "/devices/"+createdID+"/heartbeat", ``, gin.Params{{Key: "device_id", Value: createdID}}, nil, h.DeviceHeartbeat)
	require.Equal(t, http.StatusOK, heartbeat.Code)

	missingUpdate := invokeHandler(t, h, http.MethodPatch, "/devices/missing", `{}`, gin.Params{{Key: "device_id", Value: "missing"}}, nil, h.UpdateDevice)
	assertHandlerError(t, missingUpdate, http.StatusBadRequest, "device_update_failed")
	foreignUpdate := invokeHandler(t, h, http.MethodPatch, "/devices/dev_dery_android", `{}`, gin.Params{{Key: "device_id", Value: "dev_dery_android"}}, nil, h.UpdateDevice)
	assertHandlerError(t, foreignUpdate, http.StatusBadRequest, "device_update_failed")
	missingHeartbeat := invokeHandler(t, h, http.MethodPost, "/devices/missing/heartbeat", ``, gin.Params{{Key: "device_id", Value: "missing"}}, nil, h.DeviceHeartbeat)
	assertHandlerError(t, missingHeartbeat, http.StatusInternalServerError, "heartbeat_failed")

	challenge := invokeHandler(t, h, http.MethodPost, "/devices/dev_android/challenge", ``, gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.DeviceGrantKeyChallenge)
	assertHandlerError(t, challenge, http.StatusInternalServerError, "device_update_failed")
	malformed := invokeHandler(t, h, http.MethodPut, "/devices/dev_android/grant-key", `{`, gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.BindDeviceGrantKey)
	assertHandlerError(t, malformed, http.StatusBadRequest, "device_update_failed")
	unknownField := invokeHandler(t, h, http.MethodPut, "/devices/dev_android/grant-key", `{"challenge_token":"x","public_jwk":{},"proof":"y","unexpected":true}`, gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.BindDeviceGrantKey)
	assertHandlerError(t, unknownField, http.StatusBadRequest, "device_update_failed")
	missingField := invokeHandler(t, h, http.MethodPut, "/devices/dev_android/grant-key", `{"challenge_token":"x","public_jwk":{},"proof":""}`, gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.BindDeviceGrantKey)
	assertHandlerError(t, missingField, http.StatusBadRequest, "device_update_failed")
}

func TestHandlerAuthProfileCoverage_ProfileAndAvatarBranches(t *testing.T) {
	h := newBranchHandler(t)

	profile := invokeHandler(t, h, http.MethodGet, "/me", ``, nil, nil, h.GetProfile)
	require.Equal(t, http.StatusOK, profile.Code)
	assert.Contains(t, profile.Body.String(), `"password_enabled":true`)
	missingProfile := invokeHandler(t, h, http.MethodGet, "/me", ``, nil, map[string]any{"user_id": "missing-user"}, h.GetProfile)
	assertHandlerError(t, missingProfile, http.StatusNotFound, "profile_not_found")

	updated := invokeHandler(t, h, http.MethodPatch, "/me", `{"display_name":"  Profile Coverage  "}`, nil, nil, h.UpdateProfile)
	require.Equal(t, http.StatusOK, updated.Code)
	assert.Contains(t, updated.Body.String(), "Profile Coverage")
	malformed := invokeHandler(t, h, http.MethodPatch, "/me", `{`, nil, nil, h.UpdateProfile)
	assertHandlerError(t, malformed, http.StatusBadRequest, "err_validation")
	invalidName := invokeHandler(t, h, http.MethodPatch, "/me", `{"display_name":""}`, nil, nil, h.UpdateProfile)
	assertHandlerError(t, invalidName, http.StatusBadRequest, "profile_update_failed")

	deleteAvatar := invokeHandler(t, h, http.MethodDelete, "/me/avatar", ``, nil, nil, h.DeleteAvatar)
	require.Equal(t, http.StatusOK, deleteAvatar.Code)
	missingUpload := invokeHandler(t, h, http.MethodPost, "/me/avatar", ``, nil, nil, h.UploadAvatar)
	assertHandlerError(t, missingUpload, http.StatusBadRequest, "profile_update_failed")
	missingAvatar := invokeHandler(t, h, http.MethodGet, "/users/missing/avatar", ``, gin.Params{{Key: "id", Value: "missing"}}, nil, h.UserAvatar)
	assertHandlerError(t, missingAvatar, http.StatusNotFound, "profile_not_found")
}

func TestHandlerAuthProfileCoverage_AggregatePrivacyAndAnalyticsBranches(t *testing.T) {
	h := newBranchHandler(t)
	today := time.Now().UTC().Format("2006-01-02")

	valid := invokeHandler(t, h, http.MethodPost, "/client/aggregate-events", `{"device_id":"dev_android","event_type":"block_count_sync","event_date":"`+today+`","count":0,"idempotency_key":"coverage-aggregate-1","snapshot":false}`, nil, nil, h.RecordAggregateEvent)
	require.Equal(t, http.StatusAccepted, valid.Code)
	malformed := invokeHandler(t, h, http.MethodPost, "/client/aggregate-events", `{`, nil, nil, h.RecordAggregateEvent)
	assertHandlerError(t, malformed, http.StatusBadRequest, "err_validation")
	serviceError := invokeHandler(t, h, http.MethodPost, "/client/aggregate-events", `{"device_id":"dev_android","event_type":"block_count_sync","event_date":"`+today+`","count":-1,"idempotency_key":"coverage-aggregate-2"}`, nil, nil, h.RecordAggregateEvent)
	assertHandlerError(t, serviceError, http.StatusBadRequest, "aggregate_event_rejected")
	blockedSaveError := invokeHandler(t, h, http.MethodPost, "/client/aggregate-events", `{"device_id":"dev_android","event_type":"block_count_sync","event_date":"`+today+`","count":1,"idempotency_key":"coverage-aggregate-3","blocked_event_times":["2020-01-01T00:00:00Z"]}`, nil, nil, h.RecordAggregateEvent)
	assertHandlerError(t, blockedSaveError, http.StatusBadRequest, "blocked_events_rejected")

	analytics := invokeHandler(t, h, http.MethodGet, "/client/protection-analytics?days=7&device_id=dev_android", ``, nil, nil, h.ProtectionAnalytics)
	require.Equal(t, http.StatusOK, analytics.Code)
	invalidPeriod := invokeHandler(t, h, http.MethodGet, "/client/protection-analytics?days=nope&device_id=dev_android", ``, nil, nil, h.ProtectionAnalytics)
	assertHandlerError(t, invalidPeriod, http.StatusBadRequest, "analytics_period_invalid")
	invalidServicePeriod := invokeHandler(t, h, http.MethodGet, "/client/protection-analytics?days=14&device_id=dev_android", ``, nil, nil, h.ProtectionAnalytics)
	assertHandlerError(t, invalidServicePeriod, http.StatusBadRequest, "protection_analytics_failed")
	missingDevice := invokeHandler(t, h, http.MethodGet, "/client/protection-analytics?days=7", ``, nil, nil, h.ProtectionAnalytics)
	assertHandlerError(t, missingDevice, http.StatusBadRequest, "protection_analytics_failed")

	parsed, err := parseBlockedEventTimes([]string{"2026-09-05T12:00:00+07:00"})
	require.NoError(t, err)
	assert.Equal(t, time.UTC, parsed[0].Location())
	_, err = parseBlockedEventTimes([]string{"invalid"})
	assert.Error(t, err)
}

func TestHandlerAuthProfileCoverage_SumberdayaJournalAndSupportBranches(t *testing.T) {
	h := newAuthProfileCoveragePlainHandler(t)

	reflections := invokeHandler(t, h, http.MethodGet, "/reflections", ``, nil, nil, h.GetReflections)
	require.Equal(t, http.StatusOK, reflections.Code)
	daily := invokeHandler(t, h, http.MethodGet, "/journal/today", ``, nil, nil, h.GetDailyJournal)
	require.Equal(t, http.StatusOK, daily.Code)
	dailyList := invokeHandler(t, h, http.MethodGet, "/journals", ``, nil, nil, h.GetDailyJournals)
	require.Equal(t, http.StatusOK, dailyList.Code)

	document := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Journal coverage"}]}]}`
	journal := invokeHandler(t, h, http.MethodPost, "/journal", `{"document":`+document+`}`, nil, nil, h.UpsertDailyJournal)
	require.Equal(t, http.StatusOK, journal.Code)
	createdReflection := invokeHandler(t, h, http.MethodPost, "/reflection", `{"text":"Reflection coverage","mood_score":4,"next_step":"Continue","is_focus":true}`, nil, nil, h.CreateReflection)
	require.Equal(t, http.StatusCreated, createdReflection.Code)
	reflectionData := authProfileCoverageEnvelopeData(t, createdReflection)
	reflectionID := reflectionData["id"].(string)
	updatedReflection := invokeHandler(t, h, http.MethodPatch, "/reflection/"+reflectionID, `{"text":"Updated reflection","mood_score":5,"next_step":"Rest","status":"archived","is_focus":false}`, gin.Params{{Key: "id", Value: reflectionID}}, nil, h.UpdateReflection)
	require.Equal(t, http.StatusOK, updatedReflection.Code)
	invalidJournal := invokeHandler(t, h, http.MethodPost, "/journal", `{"document":{"type":"unsupported"}}`, nil, nil, h.UpsertDailyJournal)
	assertHandlerError(t, invalidJournal, http.StatusBadRequest, "reflection_create_failed")
	invalidReflection := invokeHandler(t, h, http.MethodPost, "/reflection", `{}`, nil, nil, h.CreateReflection)
	assertHandlerError(t, invalidReflection, http.StatusBadRequest, "text_required")

	modules := invokeHandler(t, h, http.MethodGet, "/modules?locale=id", ``, nil, nil, h.GetModules)
	require.Equal(t, http.StatusOK, modules.Code)
	modulesPage := invokeHandler(t, h, http.MethodGet, "/modules?page=1&limit=2&q=focus&category=skill", ``, nil, nil, h.GetModules)
	require.Equal(t, http.StatusOK, modulesPage.Code)

	support := invokeHandler(t, h, http.MethodGet, "/support-cases", ``, nil, nil, h.GetSupportCases)
	require.Equal(t, http.StatusOK, support.Code)
	supportPage := invokeHandler(t, h, http.MethodGet, "/support-cases?page=1&limit=2&q=case&type=technical_support&status=open&bucket=active", ``, nil, nil, h.GetSupportCases)
	require.Equal(t, http.StatusOK, supportPage.Code)
	createdCase := invokeHandler(t, h, http.MethodPost, "/support-cases", `{"summary":"Coverage case","detail":"Need help","type":"technical_support","priority":"normal","impact":"blocked"}`, nil, nil, h.CreateSupportCase)
	require.Equal(t, http.StatusCreated, createdCase.Code)
	invalidReply := invokeHandler(t, h, http.MethodPost, "/support-cases/case/reply", `{`, gin.Params{{Key: "id", Value: "case"}}, nil, h.ReplySupportCase)
	assertHandlerError(t, invalidReply, http.StatusBadRequest, "err_validation")
	invalidTransition := invokeHandler(t, h, http.MethodPost, "/support-cases/case/transition", `{`, gin.Params{{Key: "id", Value: "case"}}, nil, h.TransitionSupportCase)
	assertHandlerError(t, invalidTransition, http.StatusBadRequest, "err_validation")
}

func TestHandlerAuthProfileCoverage_RecoveryReminderAndPushValidation(t *testing.T) {
	h := newBranchHandler(t)

	invalidIntention := invokeHandler(t, h, http.MethodPost, "/intention", `{`, nil, nil, h.SaveIntention)
	assertHandlerError(t, invalidIntention, http.StatusBadRequest, "err_validation")
	invalidCheckIn := invokeHandler(t, h, http.MethodPost, "/check-ins", `{`, nil, nil, h.CreateCheckIn)
	assertHandlerError(t, invalidCheckIn, http.StatusBadRequest, "err_validation")
	serviceCheckInError := invokeHandler(t, h, http.MethodPost, "/check-ins", `{"mood_score":6,"urge_score":1}`, nil, nil, h.CreateCheckIn)
	assertHandlerError(t, serviceCheckInError, http.StatusInternalServerError, "err_internal")
	invalidRecord := invokeHandler(t, h, http.MethodPost, "/recovery-records", `{`, nil, nil, h.SaveRecoveryRecord)
	assertHandlerError(t, invalidRecord, http.StatusBadRequest, "err_validation")
	invalidWeeklyReview := invokeHandler(t, h, http.MethodPut, "/weekly-review", `{`, nil, nil, h.SaveCurrentWeeklyReview)
	assertHandlerError(t, invalidWeeklyReview, http.StatusBadRequest, "err_validation")
	invalidWeeklyReviewValue := invokeHandler(t, h, http.MethodPut, "/weekly-review", `{"outcome":"not-a-valid-outcome"}`, nil, nil, h.SaveCurrentWeeklyReview)
	assertHandlerError(t, invalidWeeklyReviewValue, http.StatusBadRequest, "weekly_review_save_failed")

	invalidReminderJSON := invokeHandler(t, h, http.MethodPut, "/reminder", `{`, nil, nil, h.UpdateReminderPreference)
	assertHandlerError(t, invalidReminderJSON, http.StatusBadRequest, "err_validation")
	defaultReminder := invokeHandler(t, h, http.MethodPut, "/reminder", `{"enabled":false,"local_time":"","timezone":"","locale":"xx"}`, nil, nil, h.UpdateReminderPreference)
	require.Equal(t, http.StatusOK, defaultReminder.Code)
	invalidPushJSON := invokeHandler(t, h, http.MethodPost, "/push", `{`, nil, nil, h.UpsertPushSubscription)
	assertHandlerError(t, invalidPushJSON, http.StatusBadRequest, "push_subscription_invalid")
	invalidDeletePush := invokeHandler(t, h, http.MethodDelete, "/push", `{}`, nil, nil, h.DeletePushSubscription)
	assertHandlerError(t, invalidDeletePush, http.StatusBadRequest, "push_subscription_invalid")

	// Recovery reads remain available even when there are no practice records.
	practices := invokeHandler(t, h, http.MethodGet, "/recovery-practices", ``, nil, nil, h.GetRecoveryPractices)
	require.Equal(t, http.StatusOK, practices.Code)
	space := invokeHandler(t, h, http.MethodGet, "/recovery-space", ``, nil, nil, h.GetRecoverySpace)
	require.Equal(t, http.StatusOK, space.Code)
}

func authProfileCoverageEnvelopeData(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope envelopeShape
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	data, ok := envelope.Data.(map[string]any)
	require.True(t, ok, "expected object response data, got %T", envelope.Data)
	return data
}

func newAuthProfileCoveragePlainHandler(t *testing.T) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv:               "development",
		JWTAccessSecret:      "test-secret-very-long-please",
		JWTAccessTTL:         3600e9,
		JWTRefreshTTL:        720 * 3600e9,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
	seed := store.NewSeeded()
	seed.JournalEntries = nil
	repo := repository.New(nil, seed)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	return New(services, mid, cfg, zap.NewNop())
}
