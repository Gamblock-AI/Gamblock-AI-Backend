package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (s *AuthService) Login(ctx context.Context, email, password string) (model.AuthResponse, error) {
	user, ok := s.repo.UserByEmail(ctx, strings.TrimSpace(email))
	if !ok || user.DisabledAt != nil || !authn.VerifyPassword(password, user.PasswordHash) {
		return model.AuthResponse{}, fmt.Errorf("user not found or invalid credentials")
	}
	return s.authPair(ctx, user, nil)
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (model.AuthResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if len(password) < 8 {
		return model.AuthResponse{}, fmt.Errorf("password must contain at least 8 characters")
	}
	if _, ok := s.repo.UserByEmail(ctx, email); ok {
		return model.AuthResponse{}, fmt.Errorf("email already exists")
	}
	passwordHash, err := authn.HashPassword(password)
	if err != nil {
		return model.AuthResponse{}, err
	}
	user, err := s.repo.CreateUserWithPassword(ctx, "usr_"+uuid.NewString()[:8], email, name, passwordHash)
	if err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPair(ctx, user, nil)
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
		return fmt.Errorf("current password is invalid")
	}
	if authn.VerifyPassword(newPassword, user.PasswordHash) {
		return fmt.Errorf("new password must be different")
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
