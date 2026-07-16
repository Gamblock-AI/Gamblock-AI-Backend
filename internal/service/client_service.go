package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/google/uuid"
)

type ClientService struct {
	repo *repository.Repository
}

func (s *ClientService) RecordAggregate(ctx context.Context, userID, deviceID, eventType, date, idempotencyKey string, count int) (model.AggregateEvent, error) {
	allowed := map[string]bool{
		"intervention_shown": true, "block_count_sync": true, "tamper_detected": true,
		"permission_revoked": true, "model_updated": true, "ruleset_updated": true,
	}
	eventDate, err := time.Parse("2006-01-02", date)
	if err != nil || !allowed[eventType] || count < 0 || count > 1_000_000 || len(idempotencyKey) < 8 || len(idempotencyKey) > 120 {
		return model.AggregateEvent{}, fmt.Errorf("invalid aggregate event")
	}
	now := time.Now().UTC()
	if eventDate.After(now.Add(24*time.Hour)) || eventDate.Before(now.AddDate(0, 0, -90)) {
		return model.AggregateEvent{}, fmt.Errorf("aggregate date is outside the accepted window")
	}
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, userID) {
		return model.AggregateEvent{}, fmt.Errorf("device does not belong to user")
	}
	return s.repo.SaveAggregateEvent(ctx, model.AggregateEvent{
		ID: "agg_" + uuid.NewString(), UserID: userID, DeviceID: deviceID,
		IdempotencyKey: userID + ":" + idempotencyKey,
		EventType:      eventType, EventDate: eventDate, Count: count,
	})
}

func (s *ClientService) GetProfile(ctx context.Context, userID string) (model.User, error) {
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok {
		return model.User{}, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *ClientService) UpdateProfile(ctx context.Context, userID, displayName string) (model.User, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len([]rune(displayName)) > 80 {
		return model.User{}, fmt.Errorf("display name must contain 1-80 characters")
	}
	return s.repo.UpdateUserDisplayName(ctx, userID, displayName)
}

func NewClientService(repo *repository.Repository) *ClientService {
	return &ClientService{repo: repo}
}

func (s *ClientService) Dashboard(ctx context.Context, userID string) (model.DashboardSummary, model.ProtectionStatus, model.ProgressSnapshot, error) {
	return s.repo.GetDashboardData(ctx, userID, time.Now().UTC())
}
