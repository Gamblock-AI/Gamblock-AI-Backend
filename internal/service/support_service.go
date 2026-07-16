package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type SupportService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

func NewSupportService(repo *repository.Repository, logger *zap.Logger) *SupportService {
	return &SupportService{repo: repo, logger: logger}
}

func (s *SupportService) GetSupportCases(ctx context.Context) ([]model.SupportCase, error) {
	return s.repo.GetSupportCases(ctx)
}

func (s *SupportService) GetSupportCasesForUser(ctx context.Context, userID string) ([]model.SupportCase, error) {
	return s.repo.GetSupportCasesForUser(ctx, userID)
}

func (s *SupportService) CreateSupportCase(ctx context.Context, userID, title, cType, priority string) error {
	title = strings.TrimSpace(title)
	allowedTypes := map[string]bool{
		"technical_support": true, "account_recovery": true, "partner_abuse": true,
		"stuck_approval": true, "device_recovery": true, "notification_failure": true,
		"organization_dispute": true, "accountability_guidance": true, "privacy_request": true,
	}
	allowedPriorities := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
	if title == "" || !allowedTypes[cType] || !allowedPriorities[priority] {
		return fmt.Errorf("invalid support case input")
	}
	id := "CASE-" + uuid.NewString()[:8]
	return s.repo.CreateSupportCase(ctx, id, userID, title, cType, priority)
}

func (s *SupportService) GetDataRequests(ctx context.Context, userID string) ([]model.DataRequest, error) {
	return s.repo.GetDataRequests(ctx, userID)
}

func (s *SupportService) CreateDataRequest(ctx context.Context, userID, requestType string) error {
	id := "DR-" + uuid.NewString()[:8]
	return s.repo.CreateDataRequest(ctx, id, userID, requestType)
}
