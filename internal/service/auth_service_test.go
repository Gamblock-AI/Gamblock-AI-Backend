package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newAuthSvc(t *testing.T) (*AuthService, *store.Store) {
	t.Helper()
	cfg := config.Config{AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please-32bytes!", JWTAccessTTL: time.Hour, JWTRefreshTTL: 720 * time.Hour}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	return NewAuthService(repo, cfg, zap.NewNop()), st
}

func TestAuthService_LoginSeededUser(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "gading@gmail.com", resp.User.Email)
}

func TestAuthService_LoginUnknownFails(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.Login(context.Background(), "nobody@nowhere.xyz", "x")
	require.Error(t, err)
}

func TestAuthService_RegisterDuplicateEmailFails(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.Register(context.Background(), "gading@gmail.com", "password2", "Gading")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
}

func TestAuthService_RegisterNewUser(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Register(context.Background(), "newbie@example.com", "password2", "Newbie")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "newbie@example.com", resp.User.Email)
}

func TestAuthService_DevLoginDefaultEmail(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.DevLogin(context.Background(), "", "partner", "")
	require.NoError(t, err)
	assert.Equal(t, "partner", resp.User.Role)
}

func TestAuthService_RefreshAndLogout(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password")
	require.NoError(t, err)

	// Refresh returns a new pair.
	r2, err := svc.Refresh(context.Background(), resp.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, r2.AccessToken)

	// Logout revokes (no error expected even on unknown token path).
	err = svc.Logout(context.Background(), resp.RefreshToken)
	assert.NoError(t, err)
}

func TestAuthService_ParseAccessTokenRoundTrip(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password")
	require.NoError(t, err)
	claims, err := svc.ParseAccessToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "gading@gmail.com", claims.Email)
}

func TestAuthService_ParseAccessTokenInvalid(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.ParseAccessToken("not.a.valid.jwt")
	require.Error(t, err)
}

func TestAuthService_UpdatePasswordRevokesRefreshTokens(t *testing.T) {
	svc, _ := newAuthSvc(t)
	ctx := context.Background()
	session, err := svc.Login(ctx, "gading@gmail.com", "password")
	require.NoError(t, err)

	require.NoError(t, svc.UpdatePassword(ctx, "usr_gading", "password", "new-password"))
	_, err = svc.Refresh(ctx, session.RefreshToken)
	require.Error(t, err)
	_, err = svc.Login(ctx, "gading@gmail.com", "password")
	require.Error(t, err)
	_, err = svc.Login(ctx, "gading@gmail.com", "new-password")
	require.NoError(t, err)
}
