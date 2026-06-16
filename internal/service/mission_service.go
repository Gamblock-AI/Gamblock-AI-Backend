package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type MissionService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

func NewMissionService(repo *repository.Repository, logger *zap.Logger) *MissionService {
	return &MissionService{repo: repo, logger: logger}
}

func (s *MissionService) GetToday(ctx context.Context, userID string) (model.DailyMission, error) {
	today := time.Now().UTC().Format("2006-01-02")
	mission, err := s.repo.GetMissionByDate(ctx, userID, today)
	if err != nil {
		// Return empty mission for today
		return model.DailyMission{
			ID:     "mis_" + uuid.NewString()[:8],
			UserID: userID,
			Date:   today,
		}, nil
	}
	return mission, nil
}

func (s *MissionService) UpdateMission(ctx context.Context, userID string, missionNum int, completed bool) (model.DailyMission, error) {
	today := time.Now().UTC().Format("2006-01-02")
	return s.repo.UpsertMission(ctx, userID, today, missionNum, completed)
}
