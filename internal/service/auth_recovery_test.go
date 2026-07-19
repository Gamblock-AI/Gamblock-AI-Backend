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
	"google.golang.org/api/idtoken"
)

type captureAuthEmail struct {
	resetCode string
}

type fakeGoogleVerifier struct {
	payload *idtoken.Payload
	err     error
}

func (verifier fakeGoogleVerifier) Validate(context.Context, string, string) (*idtoken.Payload, error) {
	return verifier.payload, verifier.err
}

func (sender *captureAuthEmail) SendVerification(context.Context, string, string) error { return nil }
func (sender *captureAuthEmail) SendPasswordReset(_ context.Context, _ string, code string) error {
	sender.resetCode = code
	return nil
}

func TestGoogleLoginRequiresExplicitLinkForExistingEmail(t *testing.T) {
	cfg := config.Config{
		AppEnv: "test", GoogleClientIDs: []string{"android-client", "windows-client"},
		JWTAccessSecret: "test-secret-very-long-please-32bytes!", JWTAccessTTL: time.Hour, JWTRefreshTTL: time.Hour,
	}
	verifier := fakeGoogleVerifier{payload: &idtoken.Payload{
		Audience: "android-client", Subject: "google-existing", Claims: map[string]any{
			"email": "gading@gmail.com", "name": "Gading", "email_verified": true, "nonce": "expected",
		},
	}}
	repo := repository.New(nil, store.NewSeeded())
	svc := NewAuthServiceWithDependencies(repo, cfg, zap.NewNop(), verifier, &captureAuthEmail{})

	_, err := svc.GoogleLogin(context.Background(), "token", "", "user", "expected")
	assert.ErrorIs(t, err, ErrGoogleLinkRequired)
	_, err = svc.GoogleLogin(context.Background(), "token", "", "user", "wrong")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGoogleLinkRequired)
}

func newRecoveryAuthService(t *testing.T) (*AuthService, *captureAuthEmail) {
	t.Helper()
	cfg := config.Config{
		AppEnv: "test", NotificationMode: "demo",
		JWTAccessSecret: "test-secret-very-long-please-32bytes!",
		JWTAccessTTL:    time.Hour, JWTRefreshTTL: 720 * time.Hour,
	}
	email := &captureAuthEmail{}
	repo := repository.New(nil, store.NewSeeded())
	return NewAuthServiceWithDependencies(repo, cfg, zap.NewNop(), nil, email), email
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
