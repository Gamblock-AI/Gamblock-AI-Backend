package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

func newBranchHandler(t *testing.T) *Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv:               "development",
		JWTAccessSecret:      "test-secret-very-long-please",
		JWTAccessTTL:         3600e9,
		JWTRefreshTTL:        720 * 3600e9,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
	repo := repository.New(nil, store.NewSeeded())
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	return New(services, mid, cfg, zap.NewNop())
}

func invokeHandler(t *testing.T, h *Handler, method, path, body string, params gin.Params, values map[string]any, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Params = params
	c.Set("request_id", "handler-branch-test")
	c.Set("user_id", "usr_gading")
	c.Set("role", "user")
	for key, value := range values {
		c.Set(key, value)
	}
	fn(c)
	return w
}

func assertHandlerError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, w.Code, w.Body.String())
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, code, env.Error.Code)
	assert.Equal(t, "handler-branch-test", env.RequestID)
}

func TestHandler_ErrorMappingTables(t *testing.T) {
	t.Run("education", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{name: "not found", err: repository.ErrEducationNotFound, status: http.StatusNotFound, code: "module_not_found"},
			{name: "conflict", err: repository.ErrEducationConflict, status: http.StatusConflict, code: "education_conflict"},
			{name: "validation", err: errors.New("invalid document"), status: http.StatusBadRequest, code: "education_validation_failed"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				status, code := educationStatus(tc.err)
				assert.Equal(t, tc.status, status)
				assert.Equal(t, tc.code, code)
			})
		}
	})

	t.Run("learning hub", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{name: "state", err: service.ErrLearningHubStateInvalid, status: http.StatusBadRequest, code: "learning_hub_state_invalid"},
			{name: "checkpoint", err: repository.ErrLearningCheckpointInvalid, status: http.StatusBadRequest, code: "learning_hub_checkpoint_invalid"},
			{name: "not found", err: repository.ErrLearningItemNotFound, status: http.StatusNotFound, code: "learning_hub_item_not_found"},
			{name: "unexpected", err: errors.New("storage unavailable"), status: http.StatusInternalServerError, code: "learning_hub_mutation_failed"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				status, code := learningHubError(tc.err)
				assert.Equal(t, tc.status, status)
				assert.Equal(t, tc.code, code)
			})
		}
	})

	t.Run("learning hub admin", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{name: "validation", err: service.ErrLearningHubAdminInvalid, status: http.StatusBadRequest, code: "learning_hub_admin_validation_failed"},
			{name: "transition", err: service.ErrLearningHubTransitionInvalid, status: http.StatusBadRequest, code: "learning_hub_admin_validation_failed"},
			{name: "draft conflict", err: repository.ErrLearningAdminConflict, status: http.StatusConflict, code: "learning_hub_admin_conflict"},
			{name: "taxonomy conflict", err: service.ErrLearningHubTaxonomyConflict, status: http.StatusConflict, code: "learning_hub_taxonomy_conflict"},
			{name: "not found", err: repository.ErrLearningAdminNotFound, status: http.StatusNotFound, code: "learning_hub_admin_not_found"},
			{name: "unexpected", err: errors.New("storage unavailable"), status: http.StatusInternalServerError, code: "learning_hub_admin_failed"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				status, code := learningHubAdminError(tc.err)
				assert.Equal(t, tc.status, status)
				assert.Equal(t, tc.code, code)
			})
		}
	})

	t.Run("mission mutation", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			err    error
			status int
			code   string
		}{
			{name: "limit", err: service.ErrCustomMissionLimit, status: http.StatusConflict, code: "custom_mission_limit"},
			{name: "invalid", err: service.ErrCustomMissionInvalid, status: http.StatusBadRequest, code: "custom_mission_invalid"},
			{name: "not editable", err: service.ErrCustomMissionNotEditable, status: http.StatusConflict, code: "custom_mission_not_editable"},
			{name: "unexpected", err: errors.New("write failed"), status: http.StatusConflict, code: "mission_update_failed"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.code, missionMutationCode(tc.err))
				assert.Equal(t, tc.status, missionMutationStatus(tc.err))
			})
		}
	})
}

func TestHandler_HealthReadyAndRole(t *testing.T) {
	h := newBranchHandler(t)

	w := invokeHandler(t, h, http.MethodGet, "/health", "", nil, nil, h.Health)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)

	w = invokeHandler(t, h, http.MethodGet, "/ready", "", nil, nil, h.Ready)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"database_configured":false`)
	assert.Contains(t, w.Body.String(), `"storage_configured":false`)

	w = invokeHandler(t, h, http.MethodGet, "/admin", "", nil, map[string]any{"role": "admin"}, func(c *gin.Context) {
		assert.Equal(t, "admin", currentRole(c))
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_DeviceProfileReminderAndPushBranches(t *testing.T) {
	h := newBranchHandler(t)

	w := invokeHandler(t, h, http.MethodPost, "/devices", `{}`, nil, nil, h.CreateDevice)
	assertHandlerError(t, w, http.StatusBadRequest, "device_create_failed")

	w = invokeHandler(t, h, http.MethodPost, "/devices", `{"client_instance_id":"branch-client","platform":"android"}`, nil, nil, h.CreateDevice)
	assert.Equal(t, http.StatusCreated, w.Code)

	w = invokeHandler(t, h, http.MethodPatch, "/devices/dev_android", `{"label":"Updated Android","protection_status":"active"}`, gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.UpdateDevice)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/devices/dev_android/heartbeat", "", gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.DeviceHeartbeat)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/devices/missing/challenge", "", gin.Params{{Key: "device_id", Value: "missing"}}, nil, h.DeviceGrantKeyChallenge)
	assertHandlerError(t, w, http.StatusBadRequest, "device_update_failed")

	w = invokeHandler(t, h, http.MethodPut, "/devices/dev_android/grant-key", `{"challenge_token":"x","public_jwk":{},"proof":"y"} {}`, gin.Params{{Key: "device_id", Value: "dev_android"}}, nil, h.BindDeviceGrantKey)
	assertHandlerError(t, w, http.StatusBadRequest, "device_update_failed")

	w = invokeHandler(t, h, http.MethodPatch, "/profile", `{"display_name":"Gading Updated"}`, nil, nil, h.UpdateProfile)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPatch, "/password", `{"current_password":"","new_password":"short"}`, nil, nil, h.UpdatePassword)
	assertHandlerError(t, w, http.StatusBadRequest, "password_validation_failed")

	w = invokeHandler(t, h, http.MethodGet, "/avatar/missing", "", gin.Params{{Key: "id", Value: "missing"}}, nil, h.UserAvatar)
	assertHandlerError(t, w, http.StatusNotFound, "profile_not_found")

	w = invokeHandler(t, h, http.MethodGet, "/reminder", "", nil, nil, h.GetReminderPreference)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPut, "/reminder", `{"enabled":true,"local_time":"bad","timezone":"Asia/Jakarta","locale":"id"}`, nil, nil, h.UpdateReminderPreference)
	assertHandlerError(t, w, http.StatusBadRequest, "reminder_preference_invalid")

	w = invokeHandler(t, h, http.MethodPut, "/reminder", `{"enabled":true,"local_time":"09:30","timezone":"Asia/Jakarta","locale":"id"}`, nil, nil, h.UpdateReminderPreference)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/push", `{"endpoint":""}`, nil, nil, h.UpsertPushSubscription)
	assertHandlerError(t, w, http.StatusBadRequest, "push_subscription_invalid")

	w = invokeHandler(t, h, http.MethodPost, "/push", `{"endpoint":"https://push.example/sub","p256dh":"p256dh","auth_key":"auth"}`, nil, nil, h.UpsertPushSubscription)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodDelete, "/push", `{"endpoint":"https://push.example/sub"}`, nil, nil, h.DeletePushSubscription)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RecoveryAndJournalBranches(t *testing.T) {
	h := newBranchHandler(t)

	w := invokeHandler(t, h, http.MethodGet, "/intention", "", nil, nil, h.GetIntention)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/intention", `{"intention_text":"Saya ingin menjaga fokus belajar","status":"active","quit_motivation":"study"}`, nil, nil, h.SaveIntention)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/check-ins", `{"mood_score":4,"urge_score":1,"context_text":"Hari produktif"}`, nil, nil, h.CreateCheckIn)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodGet, "/check-ins", "", nil, nil, h.GetCheckIns)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/recovery-records", `{"kind":"roadmap","record_date":"2026-09-05","content":"Langkah belajar minggu ini","status":"active"}`, nil, nil, h.SaveRecoveryRecord)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodGet, "/recovery-records", "", nil, nil, h.GetRecoveryRecords)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodGet, "/recovery-practices", "", nil, nil, h.GetRecoveryPractices)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodGet, "/recovery-space", "", nil, nil, h.GetRecoverySpace)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodGet, "/weekly-review", "", nil, nil, h.GetCurrentWeeklyReview)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodPut, "/weekly-review", `{"outcome":"helped","helpful_action":"walk","adjustment":"continue","next_mission_number":2,"selected_skill_id":"skill"}`, nil, nil, h.SaveCurrentWeeklyReview)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/journal", `{}`, nil, nil, h.UpsertDailyJournal)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/reflection", `{}`, nil, nil, h.CreateReflection)
	assertHandlerError(t, w, http.StatusBadRequest, "text_required")
	w = invokeHandler(t, h, http.MethodPatch, "/reflection/missing", `{}`, gin.Params{{Key: "id", Value: "missing"}}, nil, h.UpdateReflection)
	assertHandlerError(t, w, http.StatusBadRequest, "reflection_update_failed")
}

func TestHandler_LearningHubBranches(t *testing.T) {
	h := newBranchHandler(t)

	for _, tc := range []struct {
		name string
		fn   gin.HandlerFunc
	}{
		{name: "catalog", fn: h.LearningHubCatalog},
		{name: "providers", fn: h.LearningHubProviders},
		{name: "items", fn: h.LearningHubItemsByProvider},
		{name: "progress", fn: h.LearningHubProgress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := invokeHandler(t, h, http.MethodGet, "/learning", "", nil, nil, tc.fn)
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		})
	}

	w := invokeHandler(t, h, http.MethodGet, "/learning/missing", "", gin.Params{{Key: "slug", Value: "missing"}}, nil, h.LearningHubItem)
	assertHandlerError(t, w, http.StatusNotFound, "learning_hub_item_not_found")

	w = invokeHandler(t, h, http.MethodPut, "/learning/item", `{"state":"   "}`, gin.Params{{Key: "id", Value: "missing"}}, nil, h.UpdateLearningHubState)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_state_invalid")
	w = invokeHandler(t, h, http.MethodPost, "/learning/item/checkpoint", `{`, gin.Params{{Key: "id", Value: "missing"}}, nil, h.CreateLearningHubCheckpoint)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_checkpoint_invalid")

	w = invokeHandler(t, h, http.MethodGet, "/admin/learning/missing", "", gin.Params{{Key: "id", Value: "missing"}}, map[string]any{"role": "admin"}, h.AdminLearningHubItem)
	assertHandlerError(t, w, http.StatusNotFound, "learning_hub_admin_not_found")
	w = invokeHandler(t, h, http.MethodPost, "/admin/learning/missing/unknown", "", gin.Params{{Key: "id", Value: "missing"}, {Key: "action", Value: "unknown"}}, map[string]any{"role": "admin"}, h.TransitionAdminLearningHubItem)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPost, "/admin/learning", `{`, nil, map[string]any{"role": "admin"}, h.CreateAdminLearningHubItem)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPost, "/admin/learning/cluster", `{`, nil, map[string]any{"role": "admin"}, h.CreateAdminLearningHubCluster)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPost, "/admin/learning/program", `{`, nil, map[string]any{"role": "admin"}, h.CreateAdminLearningHubProgram)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
}

func TestHandler_EducationMissionOrganizationAndEmergencyBranches(t *testing.T) {
	h := newBranchHandler(t)

	w := invokeHandler(t, h, http.MethodGet, "/admin/modules/missing", "", gin.Params{{Key: "id", Value: "missing"}}, map[string]any{"role": "admin"}, h.AdminModuleDetail)
	assertHandlerError(t, w, http.StatusNotFound, "module_not_found")
	w = invokeHandler(t, h, http.MethodPost, "/admin/modules", `{`, nil, map[string]any{"role": "admin"}, h.CreateAdminModule)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/admin/modules", `{"slug":" ","document":{"estimated_minutes":0}}`, nil, map[string]any{"role": "admin"}, h.CreateAdminModule)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/admin/media", "", nil, map[string]any{"role": "admin"}, h.UploadAdminEducationMedia)
	assertHandlerError(t, w, http.StatusBadRequest, "education_media_invalid")
	w = invokeHandler(t, h, http.MethodPost, "/admin/external-media", `{}`, nil, map[string]any{"role": "admin"}, h.RegisterAdminExternalMedia)
	assertHandlerError(t, w, http.StatusBadRequest, "education_media_invalid")
	w = invokeHandler(t, h, http.MethodPut, "/education/progress", `{}`, gin.Params{{Key: "id", Value: "missing"}, {Key: "revision", Value: "bad"}}, nil, h.UpdateEducationProgress)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/education/check", `{"choice_id":""}`, gin.Params{{Key: "id", Value: "missing"}, {Key: "revision", Value: "1"}, {Key: "check_id", Value: "check"}}, nil, h.AnswerEducationCheck)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")

	w = invokeHandler(t, h, http.MethodPut, "/mission", `{"mission_number":0}`, nil, nil, h.UpdateMission)
	assertHandlerError(t, w, http.StatusBadRequest, "invalid_mission")
	w = invokeHandler(t, h, http.MethodPost, "/mission/claim", `{"mission_id":"   "}`, nil, nil, h.ClaimMission)
	assertHandlerError(t, w, http.StatusBadRequest, "invalid_mission")
	w = invokeHandler(t, h, http.MethodPatch, "/mission/missing", `{}`, gin.Params{{Key: "id", Value: "missing"}}, nil, h.UpdateCustomMission)
	assertHandlerError(t, w, http.StatusBadRequest, "custom_mission_invalid")
	w = invokeHandler(t, h, http.MethodDelete, "/mission/missing", "", gin.Params{{Key: "id", Value: "missing"}}, nil, h.DeleteCustomMission)
	assertHandlerError(t, w, http.StatusConflict, "custom_mission_not_editable")

	w = invokeHandler(t, h, http.MethodGet, "/organization/org_uty", "", gin.Params{{Key: "id", Value: "org_uty"}}, nil, h.GetOrganization)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodGet, "/organization/missing", "", gin.Params{{Key: "id", Value: "missing"}}, nil, h.GetOrganization)
	assertHandlerError(t, w, http.StatusNotFound, "org_not_found")
	w = invokeHandler(t, h, http.MethodGet, "/organization/missing/members", "", gin.Params{{Key: "id", Value: "missing"}}, nil, h.ListOrganizationMembers)
	assert.Equal(t, http.StatusOK, w.Code)

	w = invokeHandler(t, h, http.MethodPost, "/emergency", `{}`, nil, nil, h.RequestEmergencyKey)
	assertHandlerError(t, w, http.StatusBadRequest, "device_id_required")
	w = invokeHandler(t, h, http.MethodGet, "/emergency", "", nil, nil, h.CurrentEmergencyKeyRequest)
	assertHandlerError(t, w, http.StatusNotFound, "emergency_request_not_found")
	w = invokeHandler(t, h, http.MethodGet, "/emergency/pending", "", nil, map[string]any{"role": "admin"}, h.PendingEmergencyKeyRequests)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodPost, "/emergency/unlock", `{"emergency_key":"","device_id":"dev_android"}`, nil, nil, h.EmergencyUnlock)
	assertHandlerError(t, w, http.StatusBadRequest, "emergency_key_required")
}

func TestHandler_ApprovalAndSupportBranches(t *testing.T) {
	h := newBranchHandler(t)

	w := invokeHandler(t, h, http.MethodGet, "/approvals", "", nil, nil, h.GetApprovalRequests)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodGet, "/approvals", "", nil, nil, h.GetApprovalRequests)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodPost, "/approval/missing/cancel", "", gin.Params{{Key: "id", Value: "missing"}}, nil, h.CancelApprovalRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "approval_cancel_failed")
	w = invokeHandler(t, h, http.MethodPost, "/approval/apply", `{}`, gin.Params{{Key: "id", Value: "missing"}}, nil, h.ApplyApprovalRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "device_id_required")
	w = invokeHandler(t, h, http.MethodGet, "/quick/missing", "", gin.Params{{Key: "token", Value: "missing"}}, nil, h.VerifyApprovalToken)
	assertHandlerError(t, w, http.StatusNotFound, "invalid_token")
	w = invokeHandler(t, h, http.MethodPost, "/quick/resolve", `{"token":"x","status":"invalid"}`, nil, nil, h.ResolveApprovalByToken)
	assertHandlerError(t, w, http.StatusBadRequest, "invalid_input")

	w = invokeHandler(t, h, http.MethodGet, "/support/CASE-1087", "", gin.Params{{Key: "id", Value: "CASE-1087"}}, nil, h.GetSupportCaseDetail)
	assert.Equal(t, http.StatusOK, w.Code)
	w = invokeHandler(t, h, http.MethodPost, "/support/CASE-1087/reply", `{"content":"Terima kasih"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, nil, h.ReplySupportCase)
	assert.Equal(t, http.StatusCreated, w.Code)
	w = invokeHandler(t, h, http.MethodPost, "/support/CASE-1087/transition", `{"status":"waiting_support"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, nil, h.TransitionSupportCase)
	assertHandlerError(t, w, http.StatusBadRequest, "support_transition_failed")
	w = invokeHandler(t, h, http.MethodGet, "/support/missing", "", gin.Params{{Key: "id", Value: "missing"}}, nil, h.GetSupportCaseDetail)
	assertHandlerError(t, w, http.StatusNotFound, "support_case_not_found")
}
