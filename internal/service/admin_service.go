package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
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

func (s *AdminService) CreateNetworkRulesetRelease(ctx context.Context, version, artifactPath, sha256Val string, rules map[string]any) error {
	id := "rel_net_" + uuid.NewString()[:8]
	return s.repo.CreateNetworkRulesetRelease(ctx, id, version, artifactPath, sha256Val, rules)
}

type emergencyKey struct {
	KeyHash   string
	CreatedAt time.Time
	CreatedBy string
	Used      bool
}

var (
	emergencyKeys   = make(map[string]*emergencyKey)
	emergencyKeysMu sync.RWMutex
)

func (s *AdminService) GenerateEmergencyKey(ctx context.Context, createdBy string) (string, error) {
	key, err := generateEmergencyKeyString()
	if err != nil {
		return "", err
	}
	keyHash := HashRefreshToken(key)
	emergencyKeysMu.Lock()
	emergencyKeys[keyHash] = &emergencyKey{
		KeyHash:   keyHash,
		CreatedAt: time.Now().UTC(),
		CreatedBy: createdBy,
		Used:      false,
	}
	emergencyKeysMu.Unlock()
	s.logger.Info("emergency key generated", zap.String("created_by", createdBy))
	return key, nil
}

func (s *AdminService) ValidateEmergencyKey(ctx context.Context, key, deviceID string) error {
	keyHash := HashRefreshToken(key)
	emergencyKeysMu.Lock()
	defer emergencyKeysMu.Unlock()

	ek, ok := emergencyKeys[keyHash]
	if !ok {
		return fmt.Errorf("kunci darurat tidak valid")
	}
	if ek.Used {
		return fmt.Errorf("kunci darurat sudah digunakan")
	}
	if time.Since(ek.CreatedAt) > 24*time.Hour {
		return fmt.Errorf("kunci darurat sudah kadaluarsa (berlaku 24 jam)")
	}

	ek.Used = true
	s.logger.Info("emergency key used for device unlock",
		zap.String("device_id", deviceID),
		zap.String("created_by", ek.CreatedBy),
	)
	return nil
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
