package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

var (
	ErrReminderPreferenceInvalid = errors.New("reminder preference is invalid")

	reminderTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
	allowedLocales      = map[string]bool{"id": true, "en": true}
)

// ReminderService owns the single opt-in daily reminder setting synced across
// the web, Android, and Windows surfaces.
type ReminderService struct {
	repo *repository.Repository
}

func NewReminderService(repo *repository.Repository) *ReminderService {
	return &ReminderService{repo: repo}
}

func (s *ReminderService) GetPreference(ctx context.Context, userID string) (model.ReminderPreference, error) {
	return s.repo.ReminderPreference(ctx, userID)
}

func (s *ReminderService) UpdatePreference(ctx context.Context, userID string, enabled bool, localTime, timezone, locale string) (model.ReminderPreference, error) {
	localTime = strings.TrimSpace(localTime)
	timezone = strings.TrimSpace(timezone)
	locale = strings.ToLower(strings.TrimSpace(locale))

	if localTime == "" {
		localTime = "19:00"
	}
	if !reminderTimePattern.MatchString(localTime) {
		return model.ReminderPreference{}, fmt.Errorf("%w: local_time must be HH:mm", ErrReminderPreferenceInvalid)
	}
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return model.ReminderPreference{}, fmt.Errorf("%w: invalid timezone", ErrReminderPreferenceInvalid)
	}
	if locale == "" || !allowedLocales[locale] {
		locale = "id"
	}
	return s.repo.UpsertReminderPreference(ctx, userID, enabled, localTime, timezone, locale)
}
