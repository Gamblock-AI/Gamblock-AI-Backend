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

type envelopeShape struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"request_id"`
}

// Build a full router on the in-memory store (no DB), with production config so
// error messages are gated to friendly text.
func newTestRouter(t *testing.T, appEnv string) (*gin.Engine, *store.Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv: appEnv, JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
	st := store.NewSeeded()
	st.JournalEntries = nil
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID(), mid.PrivacyGuard())
	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())
	v1.POST("/auth/login", h.Login)
	v1.POST("/reflections", mid.AuthRequired(), h.CreateReflection)
	v1.GET("/reflections", mid.AuthRequired(), h.GetReflections)
	return r, st
}

// Production env gate: error response must NOT leak err.Error(); must be friendly.
func TestLoginError_ProductionFriendlyNoLeak(t *testing.T) {
	r, _ := newTestRouter(t, "production")
	body := []byte(`{"email":"nobody@nowhere.xyz","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "invalid_credentials", env.Error.Code)
	assert.Equal(t, "Email atau kata sandi salah. Silakan periksa kembali.", env.Error.Message)
	assert.NotContains(t, env.Error.Message, "user not found")
	assert.NotEmpty(t, env.RequestID)
}

// Development env gate: error response includes technical detail for debugging.
func TestLoginError_DevelopmentTechnical(t *testing.T) {
	r, _ := newTestRouter(t, "development")
	body := []byte(`{"email":"nobody@nowhere.xyz","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Contains(t, env.Error.Message, "user not found", "dev must surface technical detail")
	assert.Contains(t, env.Error.Message, "invalid_credentials")
}

// Validation path uses respondCode -> friendly catalog message in production.
func TestLoginError_MissingEmail(t *testing.T) {
	r, _ := newTestRouter(t, "production")
	body := []byte(`{"password":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, "email_required", env.Error.Code)
	assert.Equal(t, "Email wajib diisi.", env.Error.Message)
}

// Reflection create succeeds with a token-authenticated dev login and returns
// the created entry in the envelope data field.
func TestReflection_CreateAndGet(t *testing.T) {
	r, _ := newTestRouter(t, "development")

	// Login as the seeded member to obtain a bearer token.
	loginBody := []byte(`{"email":"gading@gmail.com","password":"password"}`)
	lreq := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(loginBody))
	lreq.Header.Set("Content-Type", "application/json")
	lw := httptest.NewRecorder()
	r.ServeHTTP(lw, lreq)
	require.Equal(t, http.StatusOK, lw.Code, "seeded user must log in")
	var loginEnv envelopeShape
	require.NoError(t, json.Unmarshal(lw.Body.Bytes(), &loginEnv))
	token := loginEnv.Data.(map[string]any)["access_token"].(string)

	// Create reflection with a URL in the text (Bug #2 regression).
	cref := []byte(`{"text":"saya hampir buka https://example.com tapi tahan","mood":"cemas"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/reflections", bytes.NewReader(cref))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "reflection with URL must be accepted")

	// List reflections.
	greq := httptest.NewRequest(http.MethodGet, "/v1/reflections", nil)
	greq.Header.Set("Authorization", "Bearer "+token)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, greq)
	require.Equal(t, http.StatusOK, gw.Code)
}
