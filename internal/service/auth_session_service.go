package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (model.AuthResponse, error) {
	refreshTokenID, userID, deviceID, authTime, err := s.repo.GetActiveRefreshTokenSession(ctx, HashRefreshToken(rawRefresh))
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("invalid refresh token")
	}
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.DisabledAt != nil || user.MustChangePassword {
		return model.AuthResponse{}, fmt.Errorf("refresh token user not found")
	}
	if err := s.repo.RevokeRefreshTokenByID(ctx, refreshTokenID); err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPairAt(ctx, user, deviceID, authTime)
}

// Reauthenticate verifies the current user's password and issues a fresh
// access/refresh pair whose auth_time is now, satisfying recent-auth checks
// (e.g. partner group mutations) without logging the user out.
func (s *AuthService) Reauthenticate(ctx context.Context, userID, password string) (model.AuthResponse, error) {
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.DisabledAt != nil || user.MustChangePassword {
		return model.AuthResponse{}, fmt.Errorf("user is not active")
	}
	if user.PasswordHash == "" || !authn.VerifyPassword(password, user.PasswordHash) {
		return model.AuthResponse{}, fmt.Errorf("invalid credentials")
	}
	return s.authPair(ctx, user, nil)
}

func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	return s.repo.RevokeRefreshToken(ctx, HashRefreshToken(rawRefresh))
}

func (s *AuthService) authPair(ctx context.Context, user model.User, deviceID *string) (model.AuthResponse, error) {
	return s.authPairAt(ctx, user, deviceID, time.Now().UTC())
}

func (s *AuthService) authPairAt(ctx context.Context, user model.User, deviceID *string, authTime time.Time) (model.AuthResponse, error) {
	accessToken, err := s.issueTokenAt(user, authTime)
	if err != nil {
		return model.AuthResponse{}, err
	}
	rawRefresh, err := randomRefreshToken()
	if err != nil {
		return model.AuthResponse{}, err
	}
	expiresAt := time.Now().UTC().Add(s.cfg.JWTRefreshTTL)
	if err := s.repo.CreateRefreshTokenWithAuthTime(ctx, "rt_"+uuid.NewString(), user.ID, HashRefreshToken(rawRefresh), deviceID, authTime, expiresAt); err != nil {
		return model.AuthResponse{}, err
	}
	return model.AuthResponse{
		AccessToken:          accessToken,
		RefreshToken:         rawRefresh,
		TokenType:            "Bearer",
		ExpiresIn:            int(s.cfg.JWTAccessTTL.Seconds()),
		User:                 user,
		PasswordEnabled:      user.PasswordHash != "",
		VerificationRequired: user.PhoneVerifiedAt == nil,
	}, nil
}

func optionalDeviceID(deviceID string) *string {
	if deviceID == "" {
		return nil
	}
	return &deviceID
}
