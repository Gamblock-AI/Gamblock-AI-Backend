package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (s *AuthService) ParseAccessToken(tokenValue string) (*model.Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenValue, &model.Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTAccessSecret), nil
	}, jwt.WithIssuer("gamblock-ai-backend"))
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("invalid access token")
	}
	claims, ok := parsed.Claims.(*model.Claims)
	if !ok || claims.UserID == "" || claims.Email == "" || claims.Role == "" {
		return nil, fmt.Errorf("invalid access token claims")
	}
	return claims, nil
}

func (s *AuthService) issueToken(user model.User) (string, error) {
	return s.issueTokenAt(user, time.Now().UTC())
}

func (s *AuthService) issueTokenAt(user model.User, authTime time.Time) (string, error) {
	now := time.Now().UTC()
	claims := model.Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     user.Role,
		AuthTime: jwt.NewNumericDate(authTime),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessTTL)),
			Issuer:    "gamblock-ai-backend",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTAccessSecret))
}

func randomRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

const initialPasswordTokenTTL = 10 * time.Minute

type initialPasswordClaims struct {
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

func (s *AuthService) issueInitialPasswordToken(user model.User) (string, error) {
	now := time.Now().UTC()
	claims := initialPasswordClaims{
		Purpose: "initial_password_change",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: user.ID, IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(initialPasswordTokenTTL)),
			Issuer:    "gamblock-ai-backend", Audience: jwt.ClaimStrings{"initial-password"},
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTAccessSecret))
}

func (s *AuthService) parseInitialPasswordToken(raw string) (string, error) {
	parsed, err := jwt.ParseWithClaims(raw, &initialPasswordClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTAccessSecret), nil
	}, jwt.WithIssuer("gamblock-ai-backend"), jwt.WithAudience("initial-password"))
	if err != nil || !parsed.Valid {
		return "", ErrInitialPasswordChangeInvalid
	}
	claims, ok := parsed.Claims.(*initialPasswordClaims)
	if !ok || claims.Purpose != "initial_password_change" || claims.Subject == "" {
		return "", ErrInitialPasswordChangeInvalid
	}
	return claims.Subject, nil
}

// phoneVerificationTokenTTL outlives the 10-minute contact code so a resend
// (which issues a fresh code) can still complete within the token window.
const phoneVerificationTokenTTL = 30 * time.Minute

type phoneVerificationClaims struct {
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

// issuePhoneVerificationToken issues a short-lived, single-purpose token that
// lets an unverified account complete WhatsApp verification without a session.
func (s *AuthService) issuePhoneVerificationToken(user model.User) (string, error) {
	now := time.Now().UTC()
	claims := phoneVerificationClaims{
		Purpose: "phone_verification",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: user.ID, IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(phoneVerificationTokenTTL)),
			Issuer:    "gamblock-ai-backend", Audience: jwt.ClaimStrings{"phone-verification"},
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTAccessSecret))
}

func (s *AuthService) parsePhoneVerificationToken(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("phone verification token is invalid or expired")
	}
	parsed, err := jwt.ParseWithClaims(raw, &phoneVerificationClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTAccessSecret), nil
	}, jwt.WithIssuer("gamblock-ai-backend"), jwt.WithAudience("phone-verification"))
	if err != nil || !parsed.Valid {
		return "", fmt.Errorf("phone verification token is invalid or expired")
	}
	claims, ok := parsed.Claims.(*phoneVerificationClaims)
	if !ok || claims.Purpose != "phone_verification" || claims.Subject == "" {
		return "", fmt.Errorf("phone verification token is invalid or expired")
	}
	return claims.Subject, nil
}
