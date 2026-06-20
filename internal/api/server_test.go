package api

import (
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
		AllowedOrigins: []string{"*"},
		ArtifactStoragePath: "./var/artifacts",
	}
	st := store.NewSeeded()
	r := New(cfg, st, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}
