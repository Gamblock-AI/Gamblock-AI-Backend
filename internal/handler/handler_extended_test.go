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

// Router with the full set of GET endpoints exercised here.
func newExtendedRouter(t *testing.T, appEnv string) (*gin.Engine, string) {
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
	v1.GET("/reflections", mid.AuthRequired(), h.GetReflections)
	v1.POST("/reflections", mid.AuthRequired(), h.CreateReflection)
	v1.GET("/psychoeducation/modules", mid.AuthRequired(), h.GetModules)
	v1.GET("/psychoeducation/modules/:slug", mid.AuthRequired(), h.GetModuleDetail)
	v1.GET("/support-cases", mid.AuthRequired(), h.GetSupportCases)
	v1.GET("/approval-requests/verify/:token", h.VerifyApprovalToken)
	v1.GET("/client/dashboard-summary", mid.AuthRequired(), h.ClientDashboardSummary)
	v1.GET("/client/protection-status", mid.AuthRequired(), h.ClientProtectionStatus)
	return r, loginAsGading(t, r)
}

// Login as the seeded member and return the bearer token.
func loginAsGading(t *testing.T, r *gin.Engine) string {
	t.Helper()
	body := []byte(`{"email":"gading@gmail.com","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "seeded user must log in")
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env.Data.(map[string]any)["access_token"].(string)
}

func authedGet(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_GetReflections(t *testing.T) {
	r, token := newExtendedRouter(t, "development")
	w := authedGet(r, "/v1/reflections", token)
	require.Equal(t, http.StatusOK, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	// Seeded user has journal entries.
	assert.NotNil(t, env.Data)
}

func TestHandler_GetModules(t *testing.T) {
	r, token := newExtendedRouter(t, "development")
	w := authedGet(r, "/v1/psychoeducation/modules", token)
	require.Equal(t, http.StatusOK, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.NotNil(t, env.Data)
}

func TestHandler_GetSupportCases(t *testing.T) {
	r, token := newExtendedRouter(t, "development")
	w := authedGet(r, "/v1/support-cases", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_ClientDashboardSummary(t *testing.T) {
	r, token := newExtendedRouter(t, "development")
	w := authedGet(r, "/v1/client/dashboard-summary", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_QuickApprovalVerifyInvalidToken(t *testing.T) {
	r, _ := newExtendedRouter(t, "development")
	// PrivacyGuard is exempt for verify path; invalid token -> 404 with friendly msg.
	w := authedGet(r, "/v1/approval-requests/verify/nonexistent-token", "")
	require.Equal(t, http.StatusNotFound, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "invalid_token", env.Error.Code)
}

func TestHandler_UnauthorizedWithoutToken(t *testing.T) {
	r, _ := newExtendedRouter(t, "development")
	w := authedGet(r, "/v1/reflections", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
