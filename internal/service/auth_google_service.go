package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

var (
	ErrGoogleLinkRequired = errors.New("existing account must link Google")
	ErrGoogleLinkFailed   = errors.New("google account could not be linked")
)

func (s *AuthService) GoogleLogin(ctx context.Context, rawIDToken, deviceID, requestedRole, nonce string) (model.AuthResponse, error) {
	payload, err := s.verifyGoogleToken(ctx, rawIDToken, nonce)
	if err != nil {
		return model.AuthResponse{}, err
	}
	email, _ := payload.Claims["email"].(string)
	email = strings.TrimSpace(strings.ToLower(email))
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
		if requestedRole != "user" && requestedRole != "partner" {
			requestedRole = "user"
		}
		if _, exists := s.repo.UserByEmail(ctx, email); exists {
			return model.AuthResponse{}, ErrGoogleLinkRequired
		}
		var avatarURL *string
		if picture != "" {
			avatarURL = &picture
		}
		user, err = s.repo.CreateUserGoogle(ctx, "usr_"+uuid.NewString(), email, name, avatarURL, payload.Subject, requestedRole)
	}
	if err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPair(ctx, user, optionalDeviceID(deviceID))
}

func (s *AuthService) LinkGoogle(ctx context.Context, userID, currentPassword, rawIDToken, nonce string) error {
	user, exists := s.repo.UserByID(ctx, userID)
	if !exists || user.DisabledAt != nil || user.PasswordHash == "" || !authn.VerifyPassword(currentPassword, user.PasswordHash) {
		return ErrCurrentPasswordInvalid
	}
	payload, err := s.verifyGoogleToken(ctx, rawIDToken, nonce)
	if err != nil {
		return ErrGoogleLinkFailed
	}
	email, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	if !verified || !strings.EqualFold(strings.TrimSpace(email), user.Email) {
		return ErrGoogleLinkFailed
	}
	if linked, err := s.repo.GetUserByGoogleSubject(ctx, payload.Subject); err == nil && linked.ID != user.ID {
		return ErrGoogleLinkFailed
	}
	if err := s.repo.LinkUserGoogleSubject(ctx, user.ID, payload.Subject); err != nil {
		return ErrGoogleLinkFailed
	}
	return nil
}

func (s *AuthService) verifyGoogleToken(ctx context.Context, rawIDToken, nonce string) (*idtoken.Payload, error) {
	clientIDs := s.cfg.GoogleClientIDs
	if len(clientIDs) == 0 && strings.TrimSpace(s.cfg.GoogleClientID) != "" {
		clientIDs = []string{strings.TrimSpace(s.cfg.GoogleClientID)}
	}
	if len(clientIDs) == 0 {
		return nil, fmt.Errorf("Google client IDs are not configured")
	}
	var lastErr error
	for _, audience := range clientIDs {
		payload, err := s.google.Validate(ctx, rawIDToken, audience)
		if err != nil {
			lastErr = err
			continue
		}
		if nonce != "" {
			claim, _ := payload.Claims["nonce"].(string)
			if claim == "" || claim != nonce {
				return nil, fmt.Errorf("google nonce mismatch")
			}
		}
		return payload, nil
	}
	return nil, fmt.Errorf("google token audience is not allowed: %w", lastErr)
}
