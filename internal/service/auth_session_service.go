package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (model.AuthResponse, error) {
	refreshTokenID, userID, deviceID, err := s.repo.GetActiveRefreshToken(ctx, HashRefreshToken(rawRefresh))
	if err != nil {
		return model.AuthResponse{}, fmt.Errorf("invalid refresh token")
	}
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.DisabledAt != nil {
		return model.AuthResponse{}, fmt.Errorf("refresh token user not found")
	}
	if err := s.repo.RevokeRefreshTokenByID(ctx, refreshTokenID); err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPair(ctx, user, deviceID)
}

func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	return s.repo.RevokeRefreshToken(ctx, HashRefreshToken(rawRefresh))
}

func (s *AuthService) authPair(ctx context.Context, user model.User, deviceID *string) (model.AuthResponse, error) {
	accessToken, err := s.issueToken(user)
	if err != nil {
		return model.AuthResponse{}, err
	}
	rawRefresh, err := randomRefreshToken()
	if err != nil {
		return model.AuthResponse{}, err
	}
	expiresAt := time.Now().UTC().Add(s.cfg.JWTRefreshTTL)
	if err := s.repo.CreateRefreshToken(ctx, "rt_"+uuid.NewString(), user.ID, HashRefreshToken(rawRefresh), deviceID, expiresAt); err != nil {
		return model.AuthResponse{}, err
	}
	return model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
		User:         user,
	}, nil
}

func optionalDeviceID(deviceID string) *string {
	if deviceID == "" {
		return nil
	}
	return &deviceID
}
