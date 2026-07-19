package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

var (
	ErrCurrentPasswordInvalid       = errors.New("current password is invalid")
	ErrPasswordReuse                = errors.New("new password must be different")
	ErrInitialPasswordChangeInvalid = errors.New("initial password token is invalid or expired")
)

func (s *AuthService) Login(ctx context.Context, email, password string) (model.AuthResponse, error) {
	user, ok := s.repo.UserByEmail(ctx, strings.TrimSpace(email))
	if !ok || user.DisabledAt != nil || !authn.VerifyPassword(password, user.PasswordHash) {
		return model.AuthResponse{}, fmt.Errorf("user not found or invalid credentials")
	}
	if user.MustChangePassword {
		token, err := s.issueInitialPasswordToken(user)
		if err != nil {
			return model.AuthResponse{}, err
		}
		return model.AuthResponse{PasswordChangeRequired: true, PasswordChangeToken: token}, nil
	}
	response, err := s.authPair(ctx, user, nil)
	response.VerificationRequired = user.EmailVerifiedAt == nil
	return response, err
}

// ActiveIdentity revalidates mutable account state for bearer-token requests.
// This makes account disablement and legacy-role migration effective
// immediately rather than waiting for an issued access token to expire.
func (s *AuthService) ActiveIdentity(ctx context.Context, userID string) (string, bool) {
	user, ok := s.repo.UserByID(ctx, userID)
	return user.Role, ok && user.DisabledAt == nil && !user.MustChangePassword
}

func (s *AuthService) HasVerifiedEmail(ctx context.Context, userID string) bool {
	user, ok := s.repo.UserByID(ctx, userID)
	return ok && user.DisabledAt == nil && user.EmailVerifiedAt != nil
}

func (s *AuthService) CompleteInitialPasswordChange(ctx context.Context, token, newPassword string) (model.AuthResponse, error) {
	if len(newPassword) < 8 {
		return model.AuthResponse{}, ErrInitialPasswordChangeInvalid
	}
	userID, err := s.parseInitialPasswordToken(token)
	if err != nil {
		return model.AuthResponse{}, ErrInitialPasswordChangeInvalid
	}
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.DisabledAt != nil || !user.MustChangePassword || authn.VerifyPassword(newPassword, user.PasswordHash) {
		return model.AuthResponse{}, ErrInitialPasswordChangeInvalid
	}
	passwordHash, err := authn.HashPassword(newPassword)
	if err != nil {
		return model.AuthResponse{}, err
	}
	if err := s.repo.UpdateUserPasswordHash(ctx, user.ID, passwordHash); err != nil {
		return model.AuthResponse{}, err
	}
	if err := s.repo.RevokeRefreshTokensForUser(ctx, user.ID); err != nil {
		return model.AuthResponse{}, err
	}
	user.MustChangePassword = false
	return s.authPair(ctx, user, nil)
}

func (s *AuthService) Register(ctx context.Context, email, password, name string, requestedRole ...string) (model.AuthResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	role := "user"
	if len(requestedRole) > 0 && requestedRole[0] != "" {
		role = requestedRole[0]
	}
	if len(password) < 8 {
		return model.AuthResponse{}, fmt.Errorf("password must contain at least 8 characters")
	}
	if _, ok := s.repo.UserByEmail(ctx, email); ok {
		return model.AuthResponse{}, fmt.Errorf("email already exists")
	}
	if role != "user" && role != "partner" {
		return model.AuthResponse{}, fmt.Errorf("role must be user or partner")
	}
	passwordHash, err := authn.HashPassword(password)
	if err != nil {
		return model.AuthResponse{}, err
	}
	user, err := s.repo.CreateUserWithPassword(ctx, "usr_"+uuid.NewString()[:8], email, name, passwordHash, role)
	if err != nil {
		return model.AuthResponse{}, err
	}
	response, err := s.authPair(ctx, user, nil)
	if err != nil {
		return model.AuthResponse{}, err
	}
	previewURL, deliveryErr := s.BeginEmailVerification(ctx, user)
	if deliveryErr != nil {
		s.logger.Warn("email verification delivery failed", zap.String("user_id", user.ID))
	}
	response.VerificationRequired = true
	response.VerificationPreviewURL = previewURL
	return response, nil
}

func (s *AuthService) DevLogin(ctx context.Context, email, role, deviceID string) (model.AuthResponse, error) {
	if s.cfg.IsProduction() || (!s.cfg.EnableDevLogin && s.cfg.AppEnv != "test") {
		return model.AuthResponse{}, fmt.Errorf("development login is disabled")
	}
	if email == "" {
		email = "gading@gmail.com"
	}
	user, ok := s.repo.UserByEmail(ctx, email)
	if !ok || user.DisabledAt != nil {
		return model.AuthResponse{}, fmt.Errorf("development user not found")
	}
	if role != "" && s.cfg.AppEnv == "test" {
		user.Role = role
	}
	return s.authPair(ctx, user, optionalDeviceID(deviceID))
}

func (s *AuthService) UpdatePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if currentPassword == "" || len(newPassword) < 8 {
		return fmt.Errorf("current password and a new password of at least 8 characters are required")
	}
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.PasswordHash == "" || !authn.VerifyPassword(currentPassword, user.PasswordHash) {
		return ErrCurrentPasswordInvalid
	}
	if authn.VerifyPassword(newPassword, user.PasswordHash) {
		return ErrPasswordReuse
	}
	passwordHash, err := authn.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUserPasswordHash(ctx, userID, passwordHash); err != nil {
		return err
	}
	return s.repo.RevokeRefreshTokensForUser(ctx, userID)
}
