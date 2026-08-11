package service

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
)

const (
	protectionGrantIssuer      = "gamblock-ai-backend"
	protectionGrantAudience    = "gamblock-protection-native"
	protectionGrantType        = "gamblock-grant+jwt"
	deviceKeyChallengeAudience = "gamblock-device-key-enrollment"
	deviceKeyChallengeType     = "gamblock-device-key-challenge+jwt"
	protectionGrantVersion     = 1
)

var ErrProtectionGrantSigningUnavailable = errors.New("protection grant signing is unavailable")

type protectionGrantClaims struct {
	GrantVersion int                         `json:"grant_version"`
	RequestID    string                      `json:"request_id"`
	DeviceID     string                      `json:"device_id"`
	Action       string                      `json:"action"`
	Confirmation protectionGrantConfirmation `json:"cnf"`
	jwt.RegisteredClaims
}

type protectionGrantConfirmation struct {
	JWKThumbprint string `json:"jkt"`
}

type deviceKeyChallengeClaims struct {
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

type ProtectionGrantSigner struct {
	privateKey *ecdsa.PrivateKey
	keyID      string
	loadErr    error
}

func NewProtectionGrantSigner(cfg config.Config) *ProtectionGrantSigner {
	signer := &ProtectionGrantSigner{keyID: strings.TrimSpace(cfg.ProtectionGrantSigningKeyID)}
	encoded := strings.TrimSpace(cfg.ProtectionGrantSigningPrivateKey)
	if encoded == "" || signer.keyID == "" {
		signer.loadErr = fmt.Errorf("%w: signing key configuration is incomplete", ErrProtectionGrantSigningUnavailable)
		return signer
	}
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		signer.loadErr = fmt.Errorf("%w: decode signing key", ErrProtectionGrantSigningUnavailable)
		return signer
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		signer.loadErr = fmt.Errorf("%w: parse signing key", ErrProtectionGrantSigningUnavailable)
		return signer
	}
	privateKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || privateKey.Curve.Params().Name != "P-256" {
		signer.loadErr = fmt.Errorf("%w: signing key is not ECDSA P-256", ErrProtectionGrantSigningUnavailable)
		return signer
	}
	signer.privateKey = privateKey
	return signer
}

func (s *ProtectionGrantSigner) Ready() error {
	if s == nil {
		return fmt.Errorf("%w: signer is nil", ErrProtectionGrantSigningUnavailable)
	}
	if s.privateKey == nil || s.keyID == "" {
		if s.loadErr != nil {
			return s.loadErr
		}
		return fmt.Errorf("%w: signer is incomplete", ErrProtectionGrantSigningUnavailable)
	}
	return s.loadErr
}

func (s *ProtectionGrantSigner) Sign(requestID, deviceID, action, deviceKeyThumbprint, grantJTI string, startsAt, expiresAt time.Time) (string, error) {
	if err := s.Ready(); err != nil {
		return "", err
	}
	if requestID == "" || deviceID == "" || deviceKeyThumbprint == "" || grantJTI == "" || !expiresAt.After(startsAt) {
		return "", fmt.Errorf("invalid protection grant claims")
	}
	duration := expiresAt.Sub(startsAt)
	switch action {
	case "pause_protection":
		if duration != 15*time.Minute && duration != 30*time.Minute && duration != 60*time.Minute && duration != 120*time.Minute {
			return "", fmt.Errorf("pause protection grant has an unsupported duration")
		}
	case "uninstall_detected", "emergency_access":
		if duration > 10*time.Minute {
			return "", fmt.Errorf("removal grant exceeds maximum duration")
		}
	default:
		return "", fmt.Errorf("unsupported protection grant action")
	}
	claims := protectionGrantClaims{
		GrantVersion: protectionGrantVersion,
		RequestID:    requestID,
		DeviceID:     deviceID,
		Action:       action,
		Confirmation: protectionGrantConfirmation{JWKThumbprint: deviceKeyThumbprint},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    protectionGrantIssuer,
			Audience:  jwt.ClaimStrings{protectionGrantAudience},
			IssuedAt:  jwt.NewNumericDate(startsAt),
			NotBefore: jwt.NewNumericDate(startsAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        grantJTI,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = s.keyID
	token.Header["typ"] = protectionGrantType
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("%w: sign protection grant: %v", ErrProtectionGrantSigningUnavailable, err)
	}
	return signed, nil
}

func newProtectionGrantJTI() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("%w: generate protection grant nonce: %v", ErrProtectionGrantSigningUnavailable, err)
	}
	return base64.RawURLEncoding.EncodeToString(nonce), nil
}

func (s *ProtectionGrantSigner) IssueDeviceKeyChallenge(userID, deviceID string, now time.Time) (string, time.Time, error) {
	if err := s.Ready(); err != nil {
		return "", time.Time{}, err
	}
	if userID == "" || deviceID == "" {
		return "", time.Time{}, fmt.Errorf("invalid device key challenge claims")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, fmt.Errorf("%w: generate device key challenge nonce: %v", ErrProtectionGrantSigningUnavailable, err)
	}
	expiresAt := now.Add(5 * time.Minute)
	claims := deviceKeyChallengeClaims{
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    protectionGrantIssuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{deviceKeyChallengeAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        base64.RawURLEncoding.EncodeToString(nonce),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = s.keyID
	token.Header["typ"] = deviceKeyChallengeType
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("%w: sign device key challenge: %v", ErrProtectionGrantSigningUnavailable, err)
	}
	return signed, expiresAt, nil
}

func (s *ProtectionGrantSigner) VerifyDeviceKeyChallenge(raw, userID, deviceID string, now time.Time) error {
	if err := s.Ready(); err != nil {
		return err
	}
	parsed, err := jwt.ParseWithClaims(raw, &deviceKeyChallengeClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodES256 || token.Header["alg"] != jwt.SigningMethodES256.Alg() {
			return nil, fmt.Errorf("unexpected device key challenge algorithm")
		}
		if token.Header["kid"] != s.keyID || token.Header["typ"] != deviceKeyChallengeType {
			return nil, fmt.Errorf("unexpected device key challenge header")
		}
		return &s.privateKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}), jwt.WithIssuer(protectionGrantIssuer), jwt.WithAudience(deviceKeyChallengeAudience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil || !parsed.Valid {
		return fmt.Errorf("invalid or expired device key challenge")
	}
	claims, ok := parsed.Claims.(*deviceKeyChallengeClaims)
	if !ok || claims.Subject != userID || claims.DeviceID != deviceID || claims.ID == "" ||
		claims.IssuedAt == nil || claims.NotBefore == nil || claims.ExpiresAt == nil ||
		!claims.NotBefore.Time.Equal(claims.IssuedAt.Time) || claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != 5*time.Minute {
		return fmt.Errorf("device key challenge binding does not match")
	}
	return nil
}
