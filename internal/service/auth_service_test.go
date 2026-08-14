package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func TestAuthService_ProvisionedUserMustChangePasswordBeforeSession(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, err := authn.HashPassword("temporary-password")
	require.NoError(t, err)
	_, err = repo.CreateProvisionedUser(context.Background(), "usr_provisioned", "provisioned@example.com", "Provisioned", hash, "user", true)
	require.NoError(t, err)

	login, err := svc.Login(context.Background(), "provisioned@example.com", "temporary-password", "")
	require.NoError(t, err)
	assert.True(t, login.PasswordChangeRequired)
	assert.Empty(t, login.AccessToken)
	assert.NotEmpty(t, login.PasswordChangeToken)

	session, err := svc.CompleteInitialPasswordChange(context.Background(), login.PasswordChangeToken, "permanent-password", "")
	require.NoError(t, err)
	assert.NotEmpty(t, session.AccessToken)
	assert.Equal(t, "user", session.User.Role)
	_, err = svc.CompleteInitialPasswordChange(context.Background(), login.PasswordChangeToken, "another-password", "")
	require.Error(t, err)
}

func newAuthSvc(t *testing.T) (*AuthService, *store.Store) {
	t.Helper()
	cfg := config.Config{
		AppEnv: "test", NotificationMode: "demo",
		JWTAccessSecret: "test-secret-very-long-please-32bytes!",
		JWTAccessTTL: time.Hour, JWTRefreshTTL: 720 * time.Hour,
	}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	return NewAuthService(repo, cfg, zap.NewNop()), st
}

func TestAuthService_LoginSeededUser(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password", "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, "gading@gmail.com", resp.User.Email)
	assert.True(t, resp.PasswordEnabled)
}

func TestAuthService_LoginUnknownFails(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.Login(context.Background(), "nobody@nowhere.xyz", "x", "")
	require.Error(t, err)
}

func TestAuthService_RegisterDuplicateEmailFails(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.Register(context.Background(), "gading@gmail.com", "password2", "Gading", "+6281200000010")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email already exists")
}

func TestAuthService_RegisterNewUser(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Register(context.Background(), "newbie@example.com", "password2", "Newbie", "+6281200000011")
	require.NoError(t, err)
	assert.True(t, resp.VerificationRequired)
	assert.Empty(t, resp.AccessToken)
	assert.Empty(t, resp.RefreshToken)
	assert.NotEmpty(t, resp.VerificationToken)
	assert.Equal(t, "newbie@example.com", resp.User.Email)
	assert.NotEmpty(t, resp.VerificationPreviewCode, "demo mode returns the code as preview")
}

func TestAuthService_LoginUnverifiedReturnsVerification(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, err := authn.HashPassword("password2")
	require.NoError(t, err)
	_, err = repo.CreateUserWithPasswordAndPhone(context.Background(), "usr_unverif", "unverif@example.com", "Unverified", "+6281200000099", hash, "user")
	require.NoError(t, err)

	resp, err := svc.Login(context.Background(), "unverif@example.com", "password2", "")
	require.NoError(t, err)
	assert.True(t, resp.VerificationRequired)
	assert.Empty(t, resp.AccessToken)
	assert.Empty(t, resp.RefreshToken)
	assert.NotEmpty(t, resp.VerificationToken)
	assert.NotEmpty(t, resp.VerificationPreviewCode)
}

func TestAuthService_VerifyPhoneWithToken(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, _ := authn.HashPassword("password2")
	_, _ = repo.CreateUserWithPasswordAndPhone(context.Background(), "usr_unverif", "unverif@example.com", "Unverified", "+6281200000099", hash, "user")

	resp, err := svc.Login(context.Background(), "unverif@example.com", "password2", "")
	require.NoError(t, err)
	require.NotEmpty(t, resp.VerificationPreviewCode)

	// Verify issues a session (no re-login needed) once the phone is confirmed.
	verified, err := svc.VerifyPhoneWithToken(context.Background(), resp.VerificationToken, resp.VerificationPreviewCode, "")
	require.NoError(t, err)
	assert.NotEmpty(t, verified.AccessToken)
	assert.NotEmpty(t, verified.RefreshToken)
	assert.False(t, verified.VerificationRequired)
	assert.NotEmpty(t, verified.User.PhoneVerifiedAt)

	session, err := svc.Login(context.Background(), "unverif@example.com", "password2", "")
	require.NoError(t, err)
	assert.False(t, session.VerificationRequired)
	assert.NotEmpty(t, session.AccessToken)
}

func TestAuthService_VerifyPhoneWithTokenWrongCode(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, _ := authn.HashPassword("password2")
	_, _ = repo.CreateUserWithPasswordAndPhone(context.Background(), "usr_unverif", "unverif@example.com", "Unverified", "+6281200000099", hash, "user")
	resp, _ := svc.Login(context.Background(), "unverif@example.com", "password2", "")

	_, err := svc.VerifyPhoneWithToken(context.Background(), resp.VerificationToken, "000000", "")
	require.Error(t, err)
}

func TestAuthService_ResendPhoneVerification(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, _ := authn.HashPassword("password2")
	_, _ = repo.CreateUserWithPasswordAndPhone(context.Background(), "usr_unverif", "unverif@example.com", "Unverified", "+6281200000099", hash, "user")
	resp, _ := svc.Login(context.Background(), "unverif@example.com", "password2", "")

	code, err := svc.ResendPhoneVerification(context.Background(), resp.VerificationToken)
	require.NoError(t, err)
	assert.NotEmpty(t, code)

	_, err = svc.VerifyPhoneWithToken(context.Background(), resp.VerificationToken, code, "")
	require.NoError(t, err)
}

func TestAuthService_DevLoginDefaultEmail(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.DevLogin(context.Background(), "", "partner", "")
	require.NoError(t, err)
	assert.Equal(t, "partner", resp.User.Role)
}

func TestAuthService_RefreshAndLogout(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password", "")
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
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password", "")
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
	session, err := svc.Login(ctx, "gading@gmail.com", "password", "")
	require.NoError(t, err)

	require.NoError(t, svc.UpdatePassword(ctx, "usr_gading", "password", "new-password"))
	_, err = svc.Refresh(ctx, session.RefreshToken)
	require.Error(t, err)
	_, err = svc.Login(ctx, "gading@gmail.com", "password", "")
	require.Error(t, err)
	_, err = svc.Login(ctx, "gading@gmail.com", "new-password", "")
	require.NoError(t, err)
}

// The native client must refuse sessions for non-student accounts even when
// credentials are valid.
func TestAuthService_LoginNativeRejectsPartner(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.Login(context.Background(), "suci@gmail.com", "password", NativeClientType)
	require.ErrorIs(t, err, ErrStudentOnly)
}

func TestAuthService_LoginNativeRejectsAdmin(t *testing.T) {
	svc, _ := newAuthSvc(t)
	_, err := svc.Login(context.Background(), "nasywa@gmail.com", "password", NativeClientType)
	require.ErrorIs(t, err, ErrStudentOnly)
}

// The web flow stays role-agnostic: partner login succeeds with no client type.
func TestAuthService_LoginWebAllowsPartner(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "suci@gmail.com", "password", "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "partner", resp.User.Role)
}

// The native client still accepts student accounts.
func TestAuthService_LoginNativeAllowsStudent(t *testing.T) {
	svc, _ := newAuthSvc(t)
	resp, err := svc.Login(context.Background(), "gading@gmail.com", "password", NativeClientType)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.Equal(t, "user", resp.User.Role)
}

// OTP verification through the native client must also refuse non-students.
func TestAuthService_VerifyPhoneWithTokenNativeRejectsPartner(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, _ := authn.HashPassword("password2")
	_, _ = repo.CreateUserWithPasswordAndPhone(context.Background(), "usr_partner_unverif", "partner-unverif@example.com", "Partner", "+6281200000098", hash, "partner")

	resp, err := svc.Login(context.Background(), "partner-unverif@example.com", "password2", "")
	require.NoError(t, err)
	require.True(t, resp.VerificationRequired)
	require.NotEmpty(t, resp.VerificationPreviewCode)

	_, err = svc.VerifyPhoneWithToken(context.Background(), resp.VerificationToken, resp.VerificationPreviewCode, NativeClientType)
	require.ErrorIs(t, err, ErrStudentOnly)
}

// OTP verification through the web flow keeps working for partner accounts.
func TestAuthService_VerifyPhoneWithTokenWebAllowsPartner(t *testing.T) {
	svc, st := newAuthSvc(t)
	repo := repository.New(nil, st)
	hash, _ := authn.HashPassword("password2")
	_, _ = repo.CreateUserWithPasswordAndPhone(context.Background(), "usr_partner_unverif", "partner-unverif@example.com", "Partner", "+6281200000098", hash, "partner")

	resp, err := svc.Login(context.Background(), "partner-unverif@example.com", "password2", "")
	require.NoError(t, err)
	require.True(t, resp.VerificationRequired)
	require.NotEmpty(t, resp.VerificationPreviewCode)

	verified, err := svc.VerifyPhoneWithToken(context.Background(), resp.VerificationToken, resp.VerificationPreviewCode, "")
	require.NoError(t, err)
	assert.NotEmpty(t, verified.AccessToken)
	assert.Equal(t, "partner", verified.User.Role)
}
