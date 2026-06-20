package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/handler"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

// Register wires the full route table; this test asserts that key public
// endpoints respond (health/ready) and that the route table builds without
// panicking — a regression guard for route wiring.
func TestRegister_HealthAndReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please", AllowedOrigins: []string{"*"}}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := handler.New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(mid.RequestID())
	Register(r, h, mid)

	for _, tc := range []struct{ path, method string }{
		{"/healthz", http.MethodGet},
		{"/readyz", http.MethodGet},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equalf(t, http.StatusOK, w.Code, "%s should be 200", tc.path)
		assert.Contains(t, w.Body.String(), "status", "%s body should contain status", tc.path)
	}
}

// A protected v1 endpoint without auth returns 401 (routes + AuthRequired wired).
func TestRegister_ProtectedRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please", AllowedOrigins: []string{"*"}}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := handler.New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(mid.RequestID(), mid.AuthOptional())
	Register(r, h, mid)

	req := httptest.NewRequest(http.MethodGet, "/v1/reflections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
