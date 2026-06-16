package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type DeviceService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

func NewDeviceService(repo *repository.Repository, logger *zap.Logger) *DeviceService {
	return &DeviceService{repo: repo, logger: logger}
}

func (s *DeviceService) CreateDevice(ctx context.Context, userID, platformVal, label, appVersion, osVersion string, modelVersion, rulesetVersion *string) (model.Device, error) {
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
	return s.repo.CreateDevice(ctx, id, userID, platformVal, label, appVersion, osVersion, modelVersion, rulesetVersion)
}

func (s *DeviceService) UpdateDevice(ctx context.Context, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	return s.repo.UpdateDevice(ctx, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion)
}

func (s *DeviceService) RecordHeartbeat(ctx context.Context, deviceID string) error {
	s.logger.Info("device heartbeat", zap.String("device_id", deviceID))
	return s.repo.RecordHeartbeat(ctx, deviceID)
}
