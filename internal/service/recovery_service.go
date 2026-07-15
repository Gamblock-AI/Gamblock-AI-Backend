package service

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

type RecoveryRepository interface {
	GetIntention(ctx context.Context, userID string) (model.Intention, bool)
	SaveIntention(ctx context.Context, userID, text, status string) (model.Intention, error)
	GetCheckIns(ctx context.Context, userID string) ([]model.CheckIn, error)
	SaveCheckIn(ctx context.Context, userID string, mood, urge int, contextText string) (model.CheckIn, error)
}

type RecoveryService struct {
	repo RecoveryRepository
}

func NewRecoveryService(repo RecoveryRepository) *RecoveryService {
	return &RecoveryService{repo: repo}
}

func (s *RecoveryService) GetActiveIntention(ctx context.Context, userID string) (model.Intention, error) {
	intn, ok := s.repo.GetIntention(ctx, userID)
	if !ok {
		// Return empty object if no active intention found, not an error
		return model.Intention{}, nil
	}
	return intn, nil
}

func (s *RecoveryService) SaveIntention(ctx context.Context, userID, text, status string) (model.Intention, error) {
	if status == "" {
		status = "active"
	}
	return s.repo.SaveIntention(ctx, userID, text, status)
}

func (s *RecoveryService) GetCheckIns(ctx context.Context, userID string) ([]model.CheckIn, error) {
	return s.repo.GetCheckIns(ctx, userID)
}

func (s *RecoveryService) CreateCheckIn(ctx context.Context, userID string, mood, urge int, contextText string) (model.CheckIn, error) {
	return s.repo.SaveCheckIn(ctx, userID, mood, urge, contextText)
}
