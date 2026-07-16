package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type AdminService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

func NewAdminService(repo *repository.Repository, logger *zap.Logger) *AdminService {
	return &AdminService{repo: repo, logger: logger}
}

func (s *AdminService) GetEducationModules(ctx context.Context) ([]model.EducationModule, error) {
	return s.repo.GetEducationModules(ctx)
}

func (s *AdminService) CreateEducationModule(ctx context.Context, m model.EducationModule) error {
	if m.ID == "" {
		m.ID = "mod_" + uuid.NewString()[:8]
	}
	return s.repo.CreateEducationModule(ctx, m)
}

func (s *AdminService) GetModelReleases(ctx context.Context) ([]model.Release, error) {
	return s.repo.GetModelReleases(ctx)
}

func (s *AdminService) CreateModelRelease(ctx context.Context, version, platform, artifactPath, sha256Val, contract string, threshold float64, metrics map[string]any) error {
	id := "rel_model_" + uuid.NewString()[:8]
	return s.repo.CreateModelRelease(ctx, id, version, platform, artifactPath, sha256Val, contract, threshold, metrics)
}

func (s *AdminService) GetRulesetReleases(ctx context.Context) ([]model.Release, error) {
	return s.repo.GetRulesetReleases(ctx)
}

func (s *AdminService) CreateRulesetRelease(ctx context.Context, version, artifactPath, sha256Val string, rules map[string]any) error {
	id := "rel_rules_" + uuid.NewString()[:8]
	return s.repo.CreateRulesetRelease(ctx, id, version, artifactPath, sha256Val, rules)
}

func (s *AdminService) GetNetworkRulesets(ctx context.Context) ([]model.Release, error) {
	return s.repo.GetNetworkRulesets(ctx)
}

func (s *AdminService) GetPortalOverview(ctx context.Context) (model.PortalOverview, error) {
	return s.repo.GetPortalOverview(ctx)
}

func (s *AdminService) CreateNetworkRulesetRelease(ctx context.Context, version, artifactPath, sha256Val string, rules map[string]any) error {
	id := "rel_net_" + uuid.NewString()[:8]
	return s.repo.CreateNetworkRulesetRelease(ctx, id, version, artifactPath, sha256Val, rules)
}

func (s *AdminService) GenerateEmergencyKey(ctx context.Context, createdBy string) (string, error) {
	return "", fmt.Errorf("direct generation is disabled; request approval from a second platform administrator")
}

func (s *AdminService) RequestEmergencyKey(ctx context.Context, requestedBy, deviceID string) (model.EmergencyKeyRequest, error) {
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, requestedBy) {
		return model.EmergencyKeyRequest{}, fmt.Errorf("device does not belong to user")
	}
	now := time.Now().UTC()
	if current, err := s.repo.GetCurrentEmergencyKeyRequest(ctx, requestedBy, deviceID, now); err == nil &&
		(current.Status == "pending" || current.Status == "reviewed" || current.Status == "approved") {
		return model.EmergencyKeyRequest{}, fmt.Errorf("an active emergency request already exists")
	}
	request := model.EmergencyKeyRequest{
		ID:               "ekr_" + uuid.NewString()[:8],
		RequestedBy:      requestedBy,
		DeviceID:         deviceID,
		Status:           "pending",
		RequestExpiresAt: now.Add(30 * time.Minute),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	created, err := s.repo.CreateEmergencyKeyRequest(ctx, request)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	s.logger.Info("emergency key requested", zap.String("requested_by", requestedBy), zap.String("request_id", created.ID))
	return created, nil
}

func (s *AdminService) GetCurrentEmergencyKeyRequest(ctx context.Context, requestedBy, deviceID string) (model.EmergencyKeyRequest, error) {
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, requestedBy) {
		return model.EmergencyKeyRequest{}, fmt.Errorf("device does not belong to user")
	}
	return s.repo.GetCurrentEmergencyKeyRequest(ctx, requestedBy, deviceID, time.Now().UTC())
}

func (s *AdminService) GetPendingEmergencyKeyRequests(ctx context.Context) ([]model.EmergencyKeyRequest, error) {
	return s.repo.GetPendingEmergencyKeyRequests(ctx, time.Now().UTC())
}

func (s *AdminService) ReviewEmergencyKeyRequest(ctx context.Context, requestID, reviewedBy string) (model.EmergencyKeyRequest, error) {
	request, err := s.repo.ReviewEmergencyKeyRequest(ctx, requestID, reviewedBy, time.Now().UTC())
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	s.logger.Info("emergency key request reviewed",
		zap.String("request_id", request.ID),
		zap.String("requested_by", request.RequestedBy),
		zap.String("reviewed_by", reviewedBy),
	)
	return request, nil
}

func (s *AdminService) ApproveEmergencyKeyRequest(ctx context.Context, requestID, approvedBy string) (model.EmergencyKeyRequest, string, error) {
	key, err := generateEmergencyKeyString()
	if err != nil {
		return model.EmergencyKeyRequest{}, "", err
	}
	now := time.Now().UTC()
	request, err := s.repo.ApproveEmergencyKeyRequest(
		ctx,
		requestID,
		approvedBy,
		HashRefreshToken(key),
		now,
		now.Add(24*time.Hour),
	)
	if err != nil {
		return model.EmergencyKeyRequest{}, "", err
	}
	s.logger.Info("emergency key approved",
		zap.String("request_id", request.ID),
		zap.String("requested_by", request.RequestedBy),
		zap.String("reviewed_by", request.ReviewedBy),
		zap.String("approved_by", approvedBy),
	)
	return request, key, nil
}

func (s *AdminService) ValidateEmergencyKey(ctx context.Context, key, deviceID string) (model.EmergencyGrant, error) {
	now := time.Now().UTC()
	request, err := s.repo.UseEmergencyKey(ctx, HashRefreshToken(key), deviceID, now)
	if err != nil {
		return model.EmergencyGrant{}, err
	}
	s.logger.Info("emergency key used for device unlock",
		zap.String("device_id", deviceID),
		zap.String("request_id", request.ID),
		zap.String("requested_by", request.RequestedBy),
		zap.String("approved_by", request.ApprovedBy),
	)
	return model.EmergencyGrant{
		RequestID: request.ID, DeviceID: deviceID,
		GrantStartsAt: now, GrantExpiresAt: now.Add(10 * time.Minute),
	}, nil
}

func generateEmergencyKeyString() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 12)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}
