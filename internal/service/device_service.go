package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type DeviceService struct {
	repo        *repository.Repository
	logger      *zap.Logger
	grantSigner *ProtectionGrantSigner
}

func NewDeviceService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *DeviceService {
	return &DeviceService{repo: repo, logger: logger, grantSigner: NewProtectionGrantSigner(cfg)}
}

func (s *DeviceService) CreateDevice(ctx context.Context, userID, clientInstanceID, platformVal, label, appVersion, osVersion string, modelVersion, rulesetVersion *string) (model.Device, error) {
	if clientInstanceID == "" {
		return model.Device{}, fmt.Errorf("client instance id is required")
	}
	if platformVal == "" {
		platformVal = "android"
	}
	if label == "" {
		label = "Protected device"
	}
	if appVersion == "" {
		appVersion = "1.0.0"
	}
	if osVersion == "" {
		osVersion = "Unknown OS"
	}
	id := "dev_" + uuid.NewString()
	return s.repo.UpsertDevice(ctx, id, userID, clientInstanceID, platformVal, label, appVersion, osVersion, modelVersion, rulesetVersion)
}

func (s *DeviceService) UpdateDevice(ctx context.Context, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	return s.repo.UpdateDevice(ctx, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion)
}

func (s *DeviceService) UpdateOwnedDevice(ctx context.Context, userID, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	return s.repo.UpdateOwnedDevice(ctx, userID, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion)
}

func (s *DeviceService) RecordHeartbeat(ctx context.Context, deviceID string) error {
	s.logger.Info("device heartbeat", zap.String("device_id", deviceID))
	return s.repo.RecordHeartbeat(ctx, deviceID)
}

func (s *DeviceService) RecordOwnedHeartbeat(ctx context.Context, userID, deviceID string) error {
	s.logger.Info("device heartbeat", zap.String("device_id", deviceID), zap.String("user_id", userID))
	return s.repo.RecordOwnedHeartbeat(ctx, userID, deviceID)
}

func (s *DeviceService) IssueGrantKeyChallenge(ctx context.Context, userID, deviceID string) (model.DeviceGrantKeyChallenge, error) {
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, userID) {
		return model.DeviceGrantKeyChallenge{}, fmt.Errorf("device does not belong to user")
	}
	now := time.Now().UTC()
	token, expiresAt, err := s.grantSigner.IssueDeviceKeyChallenge(userID, deviceID, now)
	if err != nil {
		return model.DeviceGrantKeyChallenge{}, err
	}
	return model.DeviceGrantKeyChallenge{ChallengeToken: token, ExpiresAt: expiresAt}, nil
}

func (s *DeviceService) BindGrantKey(ctx context.Context, userID, deviceID, challengeToken string, publicJWK json.RawMessage, proof string) (model.DeviceGrantKeyBinding, error) {
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, userID) {
		return model.DeviceGrantKeyBinding{}, fmt.Errorf("device does not belong to user")
	}
	challengeToken = strings.TrimSpace(challengeToken)
	proof = strings.TrimSpace(proof)
	if challengeToken == "" || len(challengeToken) > 4096 || proof == "" || len(proof) > 512 {
		return model.DeviceGrantKeyBinding{}, fmt.Errorf("device grant key enrollment payload is invalid")
	}
	now := time.Now().UTC()
	if err := s.grantSigner.VerifyDeviceKeyChallenge(challengeToken, userID, deviceID, now); err != nil {
		return model.DeviceGrantKeyBinding{}, err
	}
	parsed, canonicalJWK, thumbprint, err := parseDeviceGrantPublicJWK(publicJWK)
	if err != nil {
		return model.DeviceGrantKeyBinding{}, err
	}
	proofBytes, err := base64.RawURLEncoding.DecodeString(proof)
	if err != nil || len(proofBytes) != 64 {
		return model.DeviceGrantKeyBinding{}, fmt.Errorf("device grant key proof must be a 64-byte base64url ES256 signature")
	}
	message := []byte("gamblock-device-key-v1\n" + deviceID + "\n" + challengeToken)
	digest := sha256.Sum256(message)
	r := new(big.Int).SetBytes(proofBytes[:32])
	signatureS := new(big.Int).SetBytes(proofBytes[32:])
	if !ecdsa.Verify(parsed, digest[:], r, signatureS) {
		return model.DeviceGrantKeyBinding{}, fmt.Errorf("device grant key proof is invalid")
	}
	if err := s.repo.BindOwnedDeviceGrantKey(ctx, userID, deviceID, canonicalJWK, thumbprint); err != nil {
		return model.DeviceGrantKeyBinding{}, err
	}
	return model.DeviceGrantKeyBinding{DeviceID: deviceID, Thumbprint: thumbprint, Bound: true}, nil
}

type deviceGrantPublicJWK struct {
	KTY string `json:"kty"`
	CRV string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func parseDeviceGrantPublicJWK(raw json.RawMessage) (*ecdsa.PublicKey, string, string, error) {
	if len(raw) == 0 || len(raw) > 1024 {
		return nil, "", "", fmt.Errorf("device grant public JWK is required")
	}
	var jwk deviceGrantPublicJWK
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&jwk); err != nil {
		return nil, "", "", fmt.Errorf("device grant public JWK is invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, "", "", fmt.Errorf("device grant public JWK is invalid")
	}
	if jwk.KTY != "EC" || jwk.CRV != "P-256" {
		return nil, "", "", fmt.Errorf("device grant public JWK must use EC P-256")
	}
	xBytes, xErr := base64.RawURLEncoding.DecodeString(jwk.X)
	yBytes, yErr := base64.RawURLEncoding.DecodeString(jwk.Y)
	if xErr != nil || yErr != nil || len(xBytes) != 32 || len(yBytes) != 32 {
		return nil, "", "", fmt.Errorf("device grant public JWK coordinates are invalid")
	}
	publicKey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xBytes), Y: new(big.Int).SetBytes(yBytes)}
	if !publicKey.Curve.IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, "", "", fmt.Errorf("device grant public JWK point is invalid")
	}
	canonicalBytes, err := json.Marshal(jwk)
	if err != nil {
		return nil, "", "", fmt.Errorf("encode device grant public JWK: %w", err)
	}
	thumbprintInput := []byte(`{"crv":"P-256","kty":"EC","x":"` + jwk.X + `","y":"` + jwk.Y + `"}`)
	digest := sha256.Sum256(thumbprintInput)
	return publicKey, string(canonicalBytes), base64.RawURLEncoding.EncodeToString(digest[:]), nil
}
