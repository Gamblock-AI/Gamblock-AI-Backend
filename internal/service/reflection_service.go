package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type reflectionPayloadV2 struct {
	Version   int    `json:"version"`
	Text      string `json:"text"`
	MoodScore *int   `json:"mood_score,omitempty"`
	NextStep  string `json:"next_step,omitempty"`
}

type ReflectionService struct {
	repo        *repository.Repository
	cfg         config.Config
	logger      *zap.Logger
	encryptMode bool
}

func NewReflectionService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *ReflectionService {
	encryptMode := cfg.JournalEncryptionKey != ""
	return &ReflectionService{
		repo:        repo,
		cfg:         cfg,
		logger:      logger,
		encryptMode: encryptMode,
	}
}

func (s *ReflectionService) GetReflections(ctx context.Context, userID string) ([]model.JournalEntry, error) {
	entries, err := s.repo.GetReflections(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Decrypt if encryption is enabled
	if s.encryptMode {
		for i := range entries {
			decrypted, decErr := crypto.Decrypt(entries[i].Text, s.cfg.JournalEncryptionKey)
			if decErr != nil {
				s.logger.Error("failed to decrypt journal entry", zap.String("entry_id", entries[i].ID), zap.Error(decErr))
				return nil, fmt.Errorf("failed to decrypt journal entry")
			}
			var payload reflectionPayloadV2
			if json.Unmarshal([]byte(decrypted), &payload) == nil && payload.Version == 2 {
				entries[i].Text = payload.Text
				entries[i].MoodScore = payload.MoodScore
				entries[i].NextStep = payload.NextStep
				entries[i].PayloadVersion = 2
			} else {
				entries[i].Text = decrypted
				entries[i].PayloadVersion = 1
			}
			if entries[i].Status == "" {
				entries[i].Status = "active"
			}
		}
	}
	return entries, nil
}

func (s *ReflectionService) CreateReflection(ctx context.Context, userID, text, mood string) (model.JournalEntry, error) {
	return s.CreateReflectionEntry(ctx, userID, text, nil, "", false)
}

func (s *ReflectionService) CreateReflectionEntry(ctx context.Context, userID, text string, moodScore *int, nextStep string, isFocus bool) (model.JournalEntry, error) {
	if !s.encryptMode {
		s.logger.Error("encryption key is missing, refusing to store plaintext journal")
		return model.JournalEntry{}, fmt.Errorf("encryption is required but not configured")
	}
	text, nextStep = strings.TrimSpace(text), strings.TrimSpace(nextStep)
	if text == "" || len(text) > 8000 || len(nextStep) > 500 {
		return model.JournalEntry{}, fmt.Errorf("reflection content is invalid")
	}
	if moodScore != nil && (*moodScore < 1 || *moodScore > 5) {
		return model.JournalEntry{}, fmt.Errorf("reflection mood is invalid")
	}
	payload, marshalErr := json.Marshal(reflectionPayloadV2{Version: 2, Text: text, MoodScore: moodScore, NextStep: nextStep})
	if marshalErr != nil {
		return model.JournalEntry{}, fmt.Errorf("failed to encode journal entry")
	}
	encrypted, encErr := crypto.Encrypt(string(payload), s.cfg.JournalEncryptionKey)
	if encErr != nil {
		s.logger.Error("failed to encrypt journal, refusing to store plaintext", zap.Error(encErr))
		return model.JournalEntry{}, fmt.Errorf("failed to encrypt journal entry")
	}

	created, err := s.repo.CreateReflectionEntry(ctx, model.JournalEntry{
		UserID: userID, Text: encrypted, Status: "active", IsFocus: isFocus,
	})
	if err != nil {
		return model.JournalEntry{}, err
	}
	created.Text, created.MoodScore, created.NextStep, created.PayloadVersion = text, moodScore, nextStep, 2
	return created, nil
}

func (s *ReflectionService) UpdateReflection(ctx context.Context, userID, reflectionID string, input model.ReflectionUpdate) (model.JournalEntry, error) {
	entries, err := s.GetReflections(ctx, userID)
	if err != nil {
		return model.JournalEntry{}, err
	}
	var current *model.JournalEntry
	for index := range entries {
		if entries[index].ID == reflectionID {
			current = &entries[index]
			break
		}
	}
	if current == nil {
		return model.JournalEntry{}, fmt.Errorf("reflection not found")
	}
	if input.Text != nil {
		current.Text = strings.TrimSpace(*input.Text)
	}
	if input.MoodScore != nil {
		if *input.MoodScore < 1 || *input.MoodScore > 5 {
			return model.JournalEntry{}, fmt.Errorf("reflection mood is invalid")
		}
		current.MoodScore = input.MoodScore
	}
	if input.NextStep != nil {
		current.NextStep = strings.TrimSpace(*input.NextStep)
	}
	if input.Status != nil {
		if *input.Status != "active" && *input.Status != "archived" {
			return model.JournalEntry{}, fmt.Errorf("reflection status is invalid")
		}
		current.Status = *input.Status
	}
	if input.IsFocus != nil {
		current.IsFocus = *input.IsFocus
	}
	if current.Text == "" || len(current.Text) > 8000 || len(current.NextStep) > 500 {
		return model.JournalEntry{}, fmt.Errorf("reflection content is invalid")
	}
	payload, marshalErr := json.Marshal(reflectionPayloadV2{Version: 2, Text: current.Text, MoodScore: current.MoodScore, NextStep: current.NextStep})
	if marshalErr != nil {
		return model.JournalEntry{}, fmt.Errorf("failed to encode journal entry")
	}
	encrypted, encErr := crypto.Encrypt(string(payload), s.cfg.JournalEncryptionKey)
	if encErr != nil {
		return model.JournalEntry{}, fmt.Errorf("failed to encrypt journal entry")
	}
	persisted, err := s.repo.UpdateReflectionEntry(ctx, model.JournalEntry{
		ID: current.ID, UserID: userID, Text: encrypted, Status: current.Status, IsFocus: current.IsFocus,
	})
	if err != nil {
		return model.JournalEntry{}, err
	}
	persisted.Text, persisted.MoodScore, persisted.NextStep, persisted.PayloadVersion = current.Text, current.MoodScore, current.NextStep, 2
	return persisted, nil
}

func (s *ReflectionService) GetEducationModuleBySlug(ctx context.Context, slug string) (model.EducationModule, error) {
	return s.repo.GetEducationModuleBySlug(ctx, slug)
}
