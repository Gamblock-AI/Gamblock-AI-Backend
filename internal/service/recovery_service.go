package service

import (
	"context"
	"fmt"
	"strings"

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
	text = strings.TrimSpace(text)
	if text == "" {
		return model.Intention{}, fmt.Errorf("intention is required")
	}
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "paused" && status != "archived" {
		return model.Intention{}, fmt.Errorf("invalid intention status")
	}
	return s.repo.SaveIntention(ctx, userID, text, status)
}

func (s *RecoveryService) GetCheckIns(ctx context.Context, userID string) ([]model.CheckIn, error) {
	return s.repo.GetCheckIns(ctx, userID)
}

func (s *RecoveryService) CreateCheckIn(ctx context.Context, userID string, mood, urge int, contextText string) (model.CheckIn, error) {
	if mood < 1 || mood > 5 || urge < 1 || urge > 5 {
		return model.CheckIn{}, fmt.Errorf("mood and urge must be between 1 and 5")
	}
	return s.repo.SaveCheckIn(ctx, userID, mood, urge, contextText)
}
