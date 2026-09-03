package handler

import (
	"bytes"
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

func newFullRouter(t *testing.T, appEnv string) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{AppEnv: appEnv, JWTAccessSecret: "test-secret-very-long-please", JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9, JournalEncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID(), mid.PrivacyGuard())
	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())
	v1.POST("/auth/login", h.Login)
	v1.GET("/client/protection-status", mid.AuthRequired(), h.ClientProtectionStatus)
	v1.GET("/client/progress", mid.AuthRequired(), h.ClientProgressSnapshot)
	v1.POST("/client/aggregate-events", mid.AuthRequired(), h.RecordAggregateEvent)
	v1.GET("/portal/overview", mid.AuthRequired(), h.PortalOverview)
	v1.GET("/missions/today", mid.AuthRequired(), h.GetTodayMission)
	v1.POST("/missions/claim", mid.AuthRequired(), h.ClaimMission)
	v1.POST("/missions/custom", mid.AuthRequired(), h.CreateCustomMission)
	v1.GET("/approval-requests", mid.AuthRequired(), h.GetApprovalRequests)
	v1.POST("/approval-requests", mid.AuthRequired(), h.CreateApprovalRequest)
	v1.PUT("/devices/:device_id/grant-key", mid.AuthRequired(), h.BindDeviceGrantKey)
	v1.POST("/organizations", mid.AuthRequired(), h.CreateOrganization)
	v1.POST("/organizations/join", mid.AuthRequired(), h.JoinOrganization)
	v1.GET("/organizations/mine", mid.AuthRequired(), h.GetCurrentUserOrganization)
	return r, loginToken(t, r)
}

func loginToken(t *testing.T, r *gin.Engine) string {
	t.Helper()
	body := []byte(`{"email":"gading@gmail.com","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env.Data.(map[string]any)["access_token"].(string)
}

func TestHandler_ClientProtectionStatus(t *testing.T) {
	r, token := newFullRouter(t, "development")
	w := authedGet(r, "/v1/client/protection-status", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ClientProgress(t *testing.T) {
	r, token := newFullRouter(t, "development")
	w := authedGet(r, "/v1/client/progress", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_RecordAggregateEvent_AcceptsBlockedEventTimestamps(t *testing.T) {
	r, token := newFullRouter(t, "development")
	now := time.Now().UTC()
	body, err := json.Marshal(map[string]any{
		"device_id":           "dev_android",
		"event_type":          "block_count_sync",
		"event_date":          now.Format("2006-01-02"),
		"count":               3,
		"idempotency_key":     "handler-aggregate-1",
		"blocked_event_times": []string{now.Add(-time.Minute).Format(time.RFC3339)},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/client/aggregate-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Nil(t, env.Error)
	assert.NotNil(t, env.Data)
	assert.NotEmpty(t, env.RequestID)
}

func TestHandler_RecordAggregateEvent_RejectsInvalidBlockedTimestamp(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body, err := json.Marshal(map[string]any{
		"device_id":           "dev_android",
		"event_type":          "block_count_sync",
		"event_date":          time.Now().UTC().Format("2006-01-02"),
		"count":               1,
		"idempotency_key":     "handler-aggregate-2",
		"blocked_event_times": []string{"not-rfc3339"},
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/client/aggregate-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, "blocked_events_rejected", env.Error.Code)
}

func TestHandler_RecordAggregateEvent_RequiresAuthentication(t *testing.T) {
	r, _ := newFullRouter(t, "development")
	req := httptest.NewRequest(http.MethodPost, "/v1/client/aggregate-events", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "auth_required")
}

func TestHandler_RecordAggregateEvent_RejectsRawBrowsingPayload(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body, err := json.Marshal(map[string]any{
		"device_id":       "dev_android",
		"event_type":      "block_count_sync",
		"event_date":      time.Now().UTC().Format("2006-01-02"),
		"count":           1,
		"idempotency_key": "handler-aggregate-3",
		"url":             "https://example.com",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/client/aggregate-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "privacy_payload_rejected")
}

func TestHandler_PortalOverview(t *testing.T) {
	r, token := newFullRouter(t, "development")
	w := authedGet(r, "/v1/portal/overview", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetTodayMission(t *testing.T) {
	r, token := newFullRouter(t, "development")
	w := authedGet(r, "/v1/missions/today", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CreateCustomMission(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body, err := json.Marshal(map[string]any{
		"title": "Walk after class",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/missions/custom", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_ClaimMission(t *testing.T) {
	r, token := newFullRouter(t, "development")
	today := authedGet(r, "/v1/missions/today", token)
	require.Equal(t, http.StatusOK, today.Code)
	var missionEnvelope envelopeShape
	require.NoError(t, json.Unmarshal(today.Body.Bytes(), &missionEnvelope))
	tasks := missionEnvelope.Data.(map[string]any)["tasks"].([]any)
	body, err := json.Marshal(map[string]any{"mission_id": systemMissionID(t, tasks)})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/missions/claim", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func systemMissionID(t *testing.T, tasks []any) string {
	t.Helper()
	for _, rawTask := range tasks {
		task := rawTask.(map[string]any)
		if task["source"] == "system" {
			return task["id"].(string)
		}
	}
	t.Fatal("expected at least one system mission")
	return ""
}

func TestHandler_CreateCustomMission_InvalidPayload(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body := []byte(`{"title":""}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/missions/custom", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetApprovalRequests(t *testing.T) {
	r, token := newFullRouter(t, "development")
	w := authedGet(r, "/v1/approval-requests", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CreateApprovalRequest(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body := []byte(`{"action":"pause_protection","reason":"testing","requested_duration_minutes":15,"device_id":"dev_android","membership_id":"mbr_active"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/approval-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_BindDeviceGrantKeyRejectsUnknownFields(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body := []byte(`{"challenge_token":"signed-challenge","public_jwk":{"kty":"EC","crv":"P-256","x":"x","y":"y"},"proof":"signature","url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/devices/dev_android/grant-key", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "device_update_failed")
}

func TestHandler_CreateOrganization_NameRequired(t *testing.T) {
	r, token := newFullRouter(t, "production")
	body := []byte(`{"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_JoinOrganization_CodeRequired(t *testing.T) {
	r, token := newFullRouter(t, "production")
	body := []byte(`{"group_code":""}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetCurrentUserOrganization_None(t *testing.T) {
	r, token := newFullRouter(t, "development")
	// gading has no org -> 404 no_org
	w := authedGet(r, "/v1/organizations/mine", token)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
