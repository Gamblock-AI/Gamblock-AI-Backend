package handler

import (
	"bytes"
	"encoding/json"
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

func newFullRouter(t *testing.T, appEnv string) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{AppEnv: appEnv, JWTAccessSecret: "test-secret-very-long-please", JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9}
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
	v1.GET("/portal/overview", mid.AuthRequired(), h.PortalOverview)
	v1.GET("/missions/today", mid.AuthRequired(), h.GetTodayMission)
	v1.PATCH("/missions", mid.AuthRequired(), h.UpdateMission)
	v1.GET("/approval-requests", mid.AuthRequired(), h.GetApprovalRequests)
	v1.POST("/approval-requests", mid.AuthRequired(), h.CreateApprovalRequest)
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

func TestHandler_UpdateMission(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body := []byte(`{"mission_number":1,"completed":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/missions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateMission_InvalidNum(t *testing.T) {
	r, token := newFullRouter(t, "development")
	body := []byte(`{"mission_number":99,"completed":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/missions", bytes.NewReader(body))
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
	body := []byte(`{"action":"pause_protection","reason":"testing","requested_duration_minutes":15}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/approval-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
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
