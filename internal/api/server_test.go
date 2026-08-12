package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

// New must build a fully-wired router on the in-memory store without panicking,
// and healthz must respond 200. This guards the server bootstrap + route wiring.
func TestNew_BuildsRouterAndHealthz(t *testing.T) {
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		AllowedOrigins: []string{"*"},
	}
	st := store.NewSeeded()
	r := New(cfg, st, zap.NewNop())
	require.NotNil(t, r)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

// Ready endpoint reports db/storage config.
func TestNew_ReadyEndpoint(t *testing.T) {
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		AllowedOrigins:      []string{"*"},
		MediaStoragePath:    "./var/media",
	}
	st := store.NewSeeded()
	r := New(cfg, st, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

func TestNew_CORSHeadersWrapPrivacyGuardRejection(t *testing.T) {
	const origin = "http://localhost:3000"
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		AllowedOrigins: []string{origin},
	}
	r := New(cfg, store.NewSeeded(), zap.NewNop())
	body := []byte(`{"url":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/reflections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Body.String(), "privacy_payload_rejected")
}

func TestNew_PasswordChangePreflightAllowsLocalhostPatch(t *testing.T) {
	const origin = "http://localhost:3000"
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		AllowedOrigins: []string{origin},
	}
	r := New(cfg, store.NewSeeded(), zap.NewNop())
	req := httptest.NewRequest(http.MethodOptions, "/v1/me/password", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"))
}

// Outside production, any localhost/127.0.0.1 origin with any port is allowed,
// so a dev Flutter web server on a random port needs no .env CORS edit.
func TestNew_DevModeAllowsAnyLocalhostPortOrigin(t *testing.T) {
	const origin = "http://localhost:45678"
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		AllowedOrigins: []string{"http://localhost:3000"},
	}
	r := New(cfg, store.NewSeeded(), zap.NewNop())
	req := httptest.NewRequest(http.MethodOptions, "/v1/me/password", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"))
}

// Production stays strict: a random localhost port that is not in the explicit
// AllowedOrigins list must not be echoed.
func TestNew_ProductionRejectsUnexpectedLocalhostPortOrigin(t *testing.T) {
	const origin = "http://localhost:45678"
	cfg := config.Config{
		AppEnv: "production", JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		AllowedOrigins: []string{"http://localhost:3000"},
	}
	r := New(cfg, store.NewSeeded(), zap.NewNop())
	req := httptest.NewRequest(http.MethodOptions, "/v1/me/password", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}
