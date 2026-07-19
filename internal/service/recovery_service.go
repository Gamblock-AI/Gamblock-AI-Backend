package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	appcrypto "github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type RecoveryRepository interface {
	GetIntention(ctx context.Context, userID string) (model.Intention, bool)
	SaveIntention(ctx context.Context, userID, text, status string) (model.Intention, error)
	GetCheckIns(ctx context.Context, userID string) ([]model.CheckIn, error)
	SaveCheckIn(ctx context.Context, userID string, mood, urge int, contextText string) (model.CheckIn, error)
}

type RecoveryService struct {
	repo       RecoveryRepository
	recordRepo *repository.Repository
	cfg        config.Config
}

func NewRecoveryService(repo RecoveryRepository) *RecoveryService {
	return &RecoveryService{repo: repo}
}

func NewRecoveryServiceWithConfig(repo *repository.Repository, cfg config.Config) *RecoveryService {
	return &RecoveryService{repo: repo, recordRepo: repo, cfg: cfg}
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
	if mood < 1 || mood > 5 || urge < 0 || urge > 5 {
		return model.CheckIn{}, fmt.Errorf("mood must be between 1 and 5 and urge between 0 and 5")
	}
	return s.repo.SaveCheckIn(ctx, userID, mood, urge, contextText)
}

func (s *RecoveryService) GetRecoveryRecords(ctx context.Context, userID string) ([]model.RecoveryRecord, error) {
	if s.recordRepo == nil {
		return []model.RecoveryRecord{}, nil
	}
	cutoff := time.Now().UTC().AddDate(-1, 0, 0)
	if err := s.recordRepo.DeleteExpiredRecoveryRecords(ctx, userID, cutoff); err != nil {
		return nil, err
	}
	items, err := s.recordRepo.ListRecoveryRecords(ctx, userID, cutoff)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Content == "" {
			continue
		}
		if s.cfg.JournalEncryptionKey == "" {
			return nil, fmt.Errorf("recovery encryption key is unavailable")
		}
		plain, decryptErr := appcrypto.Decrypt(items[i].Content, s.cfg.JournalEncryptionKey)
		if decryptErr != nil {
			return nil, fmt.Errorf("recovery record decryption failed")
		}
		items[i].Content = plain
	}
	return items, nil
}

func (s *RecoveryService) SaveRecoveryRecord(ctx context.Context, userID, id, kind, recordDate, content, status string, metadata map[string]any) (model.RecoveryRecord, error) {
	if s.recordRepo == nil {
		return model.RecoveryRecord{}, fmt.Errorf("recovery record persistence is unavailable")
	}
	allowedKinds := map[string]bool{
		"roadmap": true, "coping_plan": true, "weekly_review": true, "practice_log": true,
		"saved_resource": true, "reminder": true, "urge_practice": true,
		"grounding_practice": true, "mission_reflection": true,
		"saved_skill": true, "reminder_preferences": true,
	}
	if !allowedKinds[kind] || !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(recordDate) {
		return model.RecoveryRecord{}, fmt.Errorf("recovery record kind or date is invalid")
	}
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "archived" {
		return model.RecoveryRecord{}, fmt.Errorf("recovery record status is invalid")
	}
	content = strings.TrimSpace(content)
	if len(content) > 8000 {
		return model.RecoveryRecord{}, fmt.Errorf("recovery record content is too long")
	}
	if kind == "reminder_preferences" || kind == "reminder" {
		if _, exists := metadata["enabled"]; !exists {
			metadata["enabled"] = false
		}
	}
	encrypted := ""
	var err error
	if content != "" {
		if s.cfg.JournalEncryptionKey == "" {
			return model.RecoveryRecord{}, fmt.Errorf("recovery encryption is required")
		}
		encrypted, err = appcrypto.Encrypt(content, s.cfg.JournalEncryptionKey)
		if err != nil {
			return model.RecoveryRecord{}, fmt.Errorf("recovery record encryption failed")
		}
	}
	now := time.Now().UTC()
	if id == "" {
		id = "rec_" + uuid.NewString()[:12]
	}
	item, err := s.recordRepo.SaveRecoveryRecord(ctx, model.RecoveryRecord{
		ID: id, UserID: userID, Kind: kind, RecordDate: recordDate,
		Metadata: metadata, Content: encrypted, Status: status, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return model.RecoveryRecord{}, err
	}
	item.Content = content
	return item, nil
}

func (s *RecoveryService) GetRecoveryPractices(ctx context.Context, userID string) ([]model.RecoveryPracticeSession, error) {
	if s.recordRepo == nil {
		return []model.RecoveryPracticeSession{}, nil
	}
	cutoff := time.Now().UTC().AddDate(-1, 0, 0)
	if err := s.recordRepo.DeleteExpiredRecoveryPracticeSessions(ctx, userID, cutoff); err != nil {
		return nil, err
	}
	return s.recordRepo.ListRecoveryPracticeSessions(ctx, userID, cutoff)
}

func (s *RecoveryService) SaveRecoveryPractice(ctx context.Context, userID, kind string, durationSeconds int, feedback string) (model.RecoveryPracticeSession, error) {
	allowedKinds := map[string]bool{"urge_surfing": true, "grounding_54321": true, "focus_sprint": true}
	allowedFeedback := map[string]bool{"": true, "lighter": true, "same": true, "heavier": true, "prefer_not_say": true}
	if !allowedKinds[kind] || !allowedFeedback[feedback] || durationSeconds < 30 || durationSeconds > 7200 {
		return model.RecoveryPracticeSession{}, fmt.Errorf("recovery practice is invalid")
	}
	now := time.Now().UTC()
	item := model.RecoveryPracticeSession{
		ID: "practice_" + uuid.NewString()[:12], UserID: userID, PracticeKind: kind,
		DurationSeconds: durationSeconds, Feedback: feedback, CompletedAt: now, CreatedAt: now,
	}
	return s.recordRepo.SaveRecoveryPracticeSession(ctx, item)
}

func (s *RecoveryService) GetRecoverySpace(ctx context.Context, userID string) (model.RecoverySpace, error) {
	item, found, err := s.recordRepo.GetRecoverySpace(ctx, userID)
	if err != nil {
		return model.RecoverySpace{}, err
	}
	now := time.Now().UTC()
	if !found {
		item = model.RecoverySpace{
			ID: "space_" + uuid.NewString()[:12], UserID: userID, Theme: "dorm_room",
			UnlockedItems: []string{}, PlacedItems: map[string]any{}, UnlockRuleVersion: 1,
			CreatedAt: now, UpdatedAt: now,
		}
	}
	evidence := s.recordRepo.RecoveryUnlockEvidence(ctx, userID)
	unlocked := map[string]bool{}
	for _, key := range item.UnlockedItems {
		unlocked[key] = true
	}
	if evidence.PracticeKinds["grounding_54321"] {
		unlocked["plant"] = true
	}
	if evidence.PracticeKinds["urge_surfing"] {
		unlocked["curtain"] = true
	}
	if evidence.PracticeKinds["focus_sprint"] {
		unlocked["desk_lamp"] = true
	}
	if evidence.HasFocusJournal {
		unlocked["note_board"] = true
	}
	if evidence.HasWeeklyReview {
		unlocked["wall_art"] = true
	}
	if evidence.ActiveDays >= 5 {
		unlocked["gami_figure"] = true
	}
	item.UnlockedItems = item.UnlockedItems[:0]
	for key := range unlocked {
		item.UnlockedItems = append(item.UnlockedItems, key)
	}
	sort.Strings(item.UnlockedItems)
	item.UpdatedAt = now
	return s.recordRepo.SaveRecoverySpace(ctx, item)
}

func (s *RecoveryService) UpdateRecoverySpace(ctx context.Context, userID string, placedItems map[string]any) (model.RecoverySpace, error) {
	item, err := s.GetRecoverySpace(ctx, userID)
	if err != nil {
		return model.RecoverySpace{}, err
	}
	if placedItems == nil {
		placedItems = map[string]any{}
	}
	allowed := map[string]bool{}
	for _, key := range item.UnlockedItems {
		allowed[key] = true
	}
	for key := range placedItems {
		if !allowed[key] {
			return model.RecoverySpace{}, fmt.Errorf("recovery space item is not unlocked")
		}
	}
	item.PlacedItems = placedItems
	item.UpdatedAt = time.Now().UTC()
	return s.recordRepo.SaveRecoverySpace(ctx, item)
}

func (s *RecoveryService) GetCurrentWeeklyReview(ctx context.Context, userID string) (model.WeeklyReview, error) {
	weekStart := jakartaWeekStart(time.Now().UTC())
	items, err := s.GetRecoveryRecords(ctx, userID)
	if err != nil {
		return model.WeeklyReview{}, err
	}
	for _, item := range items {
		if item.Kind == "weekly_review" && item.RecordDate == weekStart {
			var review model.WeeklyReview
			if json.Unmarshal([]byte(item.Content), &review) != nil {
				return model.WeeklyReview{}, fmt.Errorf("weekly review payload is invalid")
			}
			review.ID, review.UpdatedAt = item.ID, item.UpdatedAt
			return review, nil
		}
	}
	return model.WeeklyReview{WeekStart: weekStart, WhatHelped: []string{}, WhatWasHard: []string{}}, nil
}

func (s *RecoveryService) SaveCurrentWeeklyReview(ctx context.Context, userID string, review model.WeeklyReview) (model.WeeklyReview, error) {
	review.WeekStart = jakartaWeekStart(time.Now().UTC())
	if len(review.WhatHelped) > 6 || len(review.WhatWasHard) > 6 || len(review.Adjustment) > 500 || len(review.NextMission) > 300 || len(review.RecommendedSkill) > 200 {
		return model.WeeklyReview{}, fmt.Errorf("weekly review is invalid")
	}
	payload, err := json.Marshal(review)
	if err != nil {
		return model.WeeklyReview{}, fmt.Errorf("weekly review encoding failed")
	}
	item, err := s.SaveRecoveryRecord(ctx, userID, review.ID, "weekly_review", review.WeekStart, string(payload), "active", map[string]any{"schema_version": 2})
	if err != nil {
		return model.WeeklyReview{}, err
	}
	review.ID, review.UpdatedAt = item.ID, item.UpdatedAt
	return review, nil
}

func jakartaWeekStart(now time.Time) string {
	location := time.FixedZone("Asia/Jakarta", 7*60*60)
	local := now.In(location)
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	return time.Date(local.Year(), local.Month(), local.Day()-daysSinceMonday, 0, 0, 0, 0, location).Format("2006-01-02")
}
