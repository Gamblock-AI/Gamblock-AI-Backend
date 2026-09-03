package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/handler"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newAccessMatrixRouter(t *testing.T) (*gin.Engine, *service.AuthService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please-32bytes!",
		JWTAccessTTL: time.Hour, JWTRefreshTTL: 720 * time.Hour,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		AllowedOrigins:       []string{"*"},
	}
	repo := repository.New(nil, store.NewSeeded())
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := handler.New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(mid.RequestID(), mid.PrivacyGuard())
	Register(r, h, mid)
	return r, services.Auth
}

func accessMatrixToken(t *testing.T, auth *service.AuthService, email string) string {
	t.Helper()
	response, err := auth.Login(context.Background(), email, "password", "")
	require.NoError(t, err)
	require.NotEmpty(t, response.AccessToken)
	return response.AccessToken
}

func staleAccessMatrixToken(t *testing.T) string {
	t.Helper()
	now := time.Now().UTC()
	claims := model.Claims{
		UserID: "usr_suci", Email: "suci@gmail.com", Role: "partner",
		AuthTime: jwt.NewNumericDate(now.Add(-time.Hour)),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "usr_suci", IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), Issuer: "gamblock-ai-backend",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret-very-long-please-32bytes!"))
	require.NoError(t, err)
	return token
}

func TestRegister_AccessMatrixEnforcesAuthRolesAndRecentAuth(t *testing.T) {
	r, auth := newAccessMatrixRouter(t)
	userToken := accessMatrixToken(t, auth, "gading@gmail.com")
	partnerToken := accessMatrixToken(t, auth, "suci@gmail.com")
	adminToken := accessMatrixToken(t, auth, "nasywa@gmail.com")

	tests := []struct {
		name      string
		method    string
		path      string
		token     string
		status    int
		errorCode string
	}{
		{name: "public health", method: http.MethodGet, path: "/healthz", status: http.StatusOK},
		{name: "missing auth", method: http.MethodGet, path: "/v1/client/dashboard-summary", status: http.StatusUnauthorized, errorCode: "auth_required"},
		{name: "user cannot read admin analytics", method: http.MethodGet, path: "/v1/admin/analytics", token: userToken, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "partner cannot read admin analytics", method: http.MethodGet, path: "/v1/admin/analytics", token: partnerToken, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "admin can read admin analytics", method: http.MethodGet, path: "/v1/admin/analytics?days=14", token: adminToken, status: http.StatusOK},
		{name: "user can read student recommendation", method: http.MethodGet, path: "/v1/client/spk-recommendation", token: userToken, status: http.StatusOK},
		{name: "partner cannot read student recommendation", method: http.MethodGet, path: "/v1/client/spk-recommendation", token: partnerToken, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "stale partner auth is rejected", method: http.MethodPost, path: "/v1/accountability/groups/grp_demo/rotate-code", token: staleAccessMatrixToken(t), status: http.StatusUnauthorized, errorCode: "recent_auth_required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			assert.Equal(t, tt.status, response.Code)
			if tt.errorCode != "" {
				assert.Contains(t, response.Body.String(), tt.errorCode)
			}
		})
	}
}
