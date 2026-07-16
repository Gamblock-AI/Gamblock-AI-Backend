package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/api/idtoken"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (s *AuthService) GoogleLogin(ctx context.Context, rawIDToken, deviceID string) (model.AuthResponse, error) {
	if s.cfg.GoogleClientID == "" {
		return model.AuthResponse{}, fmt.Errorf("GOOGLE_CLIENT_ID is not configured")
	}
	payload, err := idtoken.Validate(ctx, rawIDToken, s.cfg.GoogleClientID)
	if err != nil {
		return model.AuthResponse{}, err
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if email == "" {
		return model.AuthResponse{}, fmt.Errorf("google token has no email claim")
	}
	if !emailVerified {
		return model.AuthResponse{}, fmt.Errorf("google email is not verified")
	}
	if name == "" {
		name = email
	}

	user, err := s.repo.GetUserByGoogleSubject(ctx, payload.Subject)
	if err != nil {
		if _, exists := s.repo.UserByEmail(ctx, email); exists {
			return model.AuthResponse{}, fmt.Errorf("an existing account must link Google after password authentication")
		}
		var avatarURL *string
		if picture != "" {
			avatarURL = &picture
		}
		user, err = s.repo.CreateUserGoogle(ctx, "usr_"+uuid.NewString(), email, name, avatarURL, payload.Subject)
	}
	if err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPair(ctx, user, optionalDeviceID(deviceID))
}
