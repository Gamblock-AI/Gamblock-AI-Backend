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

func newSpkRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv: "development", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
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
	v1.GET("/client/spk-recommendation", mid.AuthRequired(), mid.RequireRoles("user"), h.GetSpkRecommendation)
	v1.POST("/client/spk-interventions/:id/complete", mid.AuthRequired(), mid.RequireRoles("user"), h.CompleteSpkIntervention)
	v1.GET("/client/spk-preference", mid.AuthRequired(), mid.RequireRoles("user"), h.GetSpkPreference)
	v1.PUT("/client/spk-preference", mid.AuthRequired(), mid.RequireRoles("user"), h.UpdateSpkPreference)

	loginBody := []byte(`{"email":"gading@gmail.com","password":"password"}`)
	lreq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
	lreq.Header.Set("Content-Type", "application/json")
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, lreq)
	require.Equal(t, http.StatusOK, lw.Code)
	var loginEnv envelopeShape
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &loginEnv))
	token := loginEnv.Data.(map[string]any)["access_token"].(string)
	return r, token
}

func TestSpk_RecommendationEndpoint(t *testing.T) {
	r, token := newSpkRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/client/spk-recommendation", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.NotNil(t, env.Data)
	data := env.Data.(map[string]any)
	assert.NotEmpty(t, data["recommendation_id"])
	assert.NotEmpty(t, data["feature"])
	assert.Equal(t, "partial", data["data_state"])
	feature := data["feature"].(map[string]any)
	assert.NotEmpty(t, feature["feature_id"])
}

func TestSpk_PreferenceRoundTrip(t *testing.T) {
	r, token := newSpkRouter(t)

	getReq := httptest.NewRequest(http.MethodGet, "/v1/client/spk-preference", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code)

	putBody := []byte(`{"spk_recommendation_enabled":true,"spk_use_protection":true,"spk_use_recovery":true,"spk_use_personal":true,"llm_personalization_enabled":true}`)
	putReq := httptest.NewRequest(http.MethodPut, "/v1/client/spk-preference", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer "+token)
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	require.Equal(t, http.StatusOK, putW.Code)

	var env envelopeShape
	require.NoError(t, json.Unmarshal(putW.Body.Bytes(), &env))
	assert.Equal(t, true, env.Data.(map[string]any)["llm_personalization_enabled"])
}
