package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	passwordResetTTL         = 30 * time.Minute
	passwordResetMaxAttempts = 5
	passwordResetAlphabet    = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

var ErrPasswordResetInvalid = errors.New("password reset code is invalid or expired")

// RequestPasswordReset deliberately returns success for unknown accounts so
// callers cannot use it to discover registered email addresses.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	user, exists := s.repo.UserByEmail(ctx, email)
	if !exists || user.DisabledAt != nil {
		return "", nil
	}
	code, err := randomReadableCode(12)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := s.repo.InvalidateContactVerifications(ctx, "password_reset", email, now); err != nil {
		return "", err
	}
	if err := s.repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "reset_" + uuid.NewString()[:12], UserID: user.ID, Kind: "password_reset",
		Destination: email, TokenHash: HashRefreshToken(email + ":" + code),
		ExpiresAt: now.Add(passwordResetTTL), CreatedAt: now,
	}); err != nil {
		return "", err
	}
	if err := s.email.SendPasswordReset(ctx, email, code); err != nil {
		// SMTP is optional at startup. Keep the public response indistinguishable
		// from an unknown email when delivery is unavailable.
		s.logger.Warn("password reset delivery failed", zap.String("user_id", user.ID))
		return "", nil
	}
	if s.cfg.NotificationMode == "demo" {
		return code, nil
	}
	return "", nil
}

func (s *AuthService) ConfirmPasswordReset(ctx context.Context, email, code, newPassword string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	code = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	if email == "" || len(code) != 12 || len(newPassword) < 8 {
		return ErrPasswordResetInvalid
	}
	verification, err := s.repo.VerifyLatestContactCode(ctx, "password_reset", email, HashRefreshToken(email+":"+code), time.Now().UTC(), passwordResetMaxAttempts)
	if err != nil {
		return ErrPasswordResetInvalid
	}
	user, exists := s.repo.UserByID(ctx, verification.UserID)
	if !exists || user.DisabledAt != nil || !strings.EqualFold(user.Email, email) {
		return ErrPasswordResetInvalid
	}
	if user.PasswordHash != "" && authn.VerifyPassword(newPassword, user.PasswordHash) {
		return ErrPasswordReuse
	}
	passwordHash, err := authn.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	if err := s.repo.UpdateUserPasswordHash(ctx, user.ID, passwordHash); err != nil {
		return err
	}
	return s.repo.RevokeRefreshTokensForUser(ctx, user.ID)
}

func randomReadableCode(length int) (string, error) {
	result := make([]byte, length)
	for index := range result {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordResetAlphabet))))
		if err != nil {
			return "", err
		}
		result[index] = passwordResetAlphabet[value.Int64()]
	}
	return string(result), nil
}
