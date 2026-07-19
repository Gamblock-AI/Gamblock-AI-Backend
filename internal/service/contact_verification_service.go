package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func (s *AuthService) BeginEmailVerification(ctx context.Context, user model.User) (string, error) {
	if user.EmailVerifiedAt != nil {
		return "", nil
	}
	rawToken := "email_" + uuid.NewString() + uuid.NewString()
	now := time.Now().UTC()
	if err := s.repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "verify_" + uuid.NewString()[:12], UserID: user.ID, Kind: "email",
		Destination: user.Email, TokenHash: HashRefreshToken(rawToken),
		ExpiresAt: now.Add(30 * time.Minute), CreatedAt: now,
	}); err != nil {
		return "", err
	}
	verificationURL := s.cfg.PublicWebBaseURL + "/verify-email?token=" + rawToken
	if err := NewEmailService(s.cfg).SendVerification(ctx, user.Email, verificationURL); err != nil {
		return "", err
	}
	if s.cfg.NotificationMode == "demo" {
		return verificationURL, nil
	}
	return "", nil
}

func (s *AuthService) ResendEmailVerification(ctx context.Context, userID string) (string, error) {
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok {
		return "", fmt.Errorf("user not found")
	}
	return s.BeginEmailVerification(ctx, user)
}

func (s *AuthService) ConfirmEmailVerification(ctx context.Context, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return fmt.Errorf("verification token is required")
	}
	verification, err := s.repo.ConsumeContactVerification(ctx, HashRefreshToken(rawToken), "email", time.Now().UTC())
	if err != nil {
		return err
	}
	return s.repo.MarkEmailVerified(ctx, verification.UserID, time.Now().UTC())
}

func (s *AuthService) BeginPhoneVerification(ctx context.Context, userID, phone string) (string, error) {
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.Role != "partner" {
		return "", fmt.Errorf("phone verification is only available for partner accounts")
	}
	phone = strings.ReplaceAll(strings.TrimSpace(phone), " ", "")
	if !e164Pattern.MatchString(phone) {
		return "", fmt.Errorf("phone must use E.164 format")
	}
	code, err := randomNumericCode(6)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := s.repo.SetPendingPhone(ctx, userID, phone); err != nil {
		return "", err
	}
	if err := s.repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "verify_" + uuid.NewString()[:12], UserID: userID, Kind: "phone",
		Destination: phone, TokenHash: HashRefreshToken(userID + ":" + code),
		ExpiresAt: now.Add(10 * time.Minute), CreatedAt: now,
	}); err != nil {
		return "", err
	}
	if err := NewWhatsAppService(s.cfg, s.logger).SendPhoneVerification(ctx, phone, code); err != nil {
		return "", err
	}
	if s.cfg.NotificationMode == "demo" {
		return code, nil
	}
	return "", nil
}

func (s *AuthService) ConfirmPhoneVerification(ctx context.Context, userID, code string) error {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return fmt.Errorf("verification code is invalid")
	}
	verification, err := s.repo.ConsumeContactVerification(ctx, HashRefreshToken(userID+":"+code), "phone", time.Now().UTC())
	if err != nil || verification.UserID != userID {
		return fmt.Errorf("verification code is invalid or expired")
	}
	return s.repo.MarkPhoneVerified(ctx, userID, verification.Destination, time.Now().UTC())
}

func randomNumericCode(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte('0' + value.Int64())
	}
	return string(result), nil
}
