package service

import (
	"context"
	"go.uber.org/zap"
	"github.com/google/uuid"

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

func (s *SupportService) CreateSupportCase(ctx context.Context, userID, title, cType, priority string) error {
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
