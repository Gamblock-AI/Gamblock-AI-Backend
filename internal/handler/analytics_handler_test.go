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

func newAnalyticsRouter(t *testing.T, appEnv string) (*gin.Engine, string) {
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
	accountability := v1.Group("/accountability")
	accountability.Use(mid.AuthRequired(), mid.RequireRoles("user", "partner"))
	accountability.GET("/analytics", h.AccountabilityAnalytics)
	admin := v1.Group("/admin")
	admin.Use(mid.AuthRequired(), mid.RequireRoles("admin"), mid.RequireVerifiedPhone())
	admin.GET("/analytics", h.AdminAnalytics)
	return r, ""
}

func loginAsEmail(t *testing.T, r *gin.Engine, email string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "password": "password"})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	return env.Data.(map[string]any)["access_token"].(string)
}

func TestHandler_PartnerAnalytics(t *testing.T) {
	r, _ := newAnalyticsRouter(t, "development")
	token := loginAsEmail(t, r, "suci@gmail.com")
	w := authedGet(r, "/v1/accountability/analytics?days=14", token)
	require.Equal(t, http.StatusOK, w.Code)

	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, float64(14), data["period_days"])
	assert.Equal(t, float64(2), data["member_count"])
	assert.Equal(t, float64(2), data["shared_member_count"])
	assert.Equal(t, float64(12), data["totals"].(map[string]any)["blocked"])
	assert.NotEmpty(t, data["daily"])
	assert.Len(t, data["hourly"], 24)
}

func TestHandler_AdminAnalytics_RequiresAdmin(t *testing.T) {
	r, _ := newAnalyticsRouter(t, "development")

	// A regular user is denied the admin analytics endpoint.
	userToken := loginAsEmail(t, r, "gading@gmail.com")
	w := authedGet(r, "/v1/admin/analytics", userToken)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// A verified admin receives the platform-wide summary.
	adminToken := loginAsEmail(t, r, "nasywa@gmail.com")
	w = authedGet(r, "/v1/admin/analytics?days=30", adminToken)
	require.Equal(t, http.StatusOK, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	data := env.Data.(map[string]any)
	assert.Equal(t, float64(30), data["period_days"])
	assert.Equal(t, float64(12), data["totals"].(map[string]any)["blocked"])
	assert.Equal(t, float64(2), data["protected_users"])
}

func TestHandler_AdminAnalytics_InvalidPeriod(t *testing.T) {
	r, _ := newAnalyticsRouter(t, "development")
	adminToken := loginAsEmail(t, r, "nasywa@gmail.com")
	w := authedGet(r, "/v1/admin/analytics?days=10", adminToken)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
