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

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(phone)
	if strings.HasPrefix(phone, "00") {
		phone = "+" + strings.TrimPrefix(phone, "00")
	}
	if strings.HasPrefix(phone, "0") {
		phone = "+62" + strings.TrimPrefix(phone, "0")
	}
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	return phone
}

func (s *AuthService) BeginPhoneVerification(ctx context.Context, userID, phone string) (string, error) {
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || user.DisabledAt != nil {
		return "", fmt.Errorf("user not found")
	}
	if strings.TrimSpace(phone) == "" {
		phone = user.PhoneE164
	}
	phone = normalizePhone(phone)
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
	if err := s.notification.SendPhoneVerification(ctx, phone, code); err != nil {
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
