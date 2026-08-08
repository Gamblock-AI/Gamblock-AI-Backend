package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type captureAuthNotification struct {
	resetCode string
}

func (sender *captureAuthNotification) SendPhoneVerification(context.Context, string, string) error {
	return nil
}
func (sender *captureAuthNotification) SendPasswordReset(_ context.Context, _ string, code string) error {
	sender.resetCode = code
	return nil
}

func newRecoveryAuthService(t *testing.T) (*AuthService, *captureAuthNotification) {
	t.Helper()
	cfg := config.Config{
		AppEnv: "test", NotificationMode: "demo",
		JWTAccessSecret: "test-secret-very-long-please-32bytes!",
		JWTAccessTTL:    time.Hour, JWTRefreshTTL: 720 * time.Hour,
	}
	email := &captureAuthNotification{}
	repo := repository.New(nil, store.NewSeeded())
	return NewAuthServiceWithDependencies(repo, cfg, zap.NewNop(), email), email
}

func TestPasswordResetIsNonEnumeratingAndSingleUse(t *testing.T) {
	ctx := context.Background()
	svc, email := newRecoveryAuthService(t)

	preview, err := svc.RequestPasswordReset(ctx, "unknown@example.com")
	require.NoError(t, err)
	assert.Empty(t, preview)

	session, err := svc.Login(ctx, "gading@gmail.com", "password")
	require.NoError(t, err)
	preview, err = svc.RequestPasswordReset(ctx, "GADING@gmail.com")
	require.NoError(t, err)
	require.Len(t, email.resetCode, 12)
	assert.Equal(t, email.resetCode, preview)

	require.NoError(t, svc.ConfirmPasswordReset(ctx, "gading@gmail.com", email.resetCode, "safe-password-2"))
	_, err = svc.Refresh(ctx, session.RefreshToken)
	require.Error(t, err)
	_, err = svc.Login(ctx, "gading@gmail.com", "safe-password-2")
	require.NoError(t, err)
	assert.ErrorIs(t, svc.ConfirmPasswordReset(ctx, "gading@gmail.com", email.resetCode, "another-password"), ErrPasswordResetInvalid)
}

func TestPasswordResetOnlyLatestCodeIsAccepted(t *testing.T) {
	ctx := context.Background()
	svc, email := newRecoveryAuthService(t)
	_, err := svc.RequestPasswordReset(ctx, "gading@gmail.com")
	require.NoError(t, err)
	first := email.resetCode
	_, err = svc.RequestPasswordReset(ctx, "gading@gmail.com")
	require.NoError(t, err)
	second := email.resetCode
	require.NotEqual(t, first, second)
	assert.True(t, errors.Is(svc.ConfirmPasswordReset(ctx, "gading@gmail.com", first, "safe-password-3"), ErrPasswordResetInvalid))
	require.NoError(t, svc.ConfirmPasswordReset(ctx, "gading@gmail.com", second, "safe-password-3"))
}
