package service

import (
	"context"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

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
			if decErr == nil {
				entries[i].Text = decrypted
			}
		}
	}
	return entries, nil
}

func (s *ReflectionService) CreateReflection(ctx context.Context, userID, text, mood string) (model.JournalEntry, error) {
	storageText := text
	if s.encryptMode {
		encrypted, encErr := crypto.Encrypt(text, s.cfg.JournalEncryptionKey)
		if encErr == nil {
			storageText = encrypted
		} else {
			s.logger.Warn("failed to encrypt journal, storing plaintext", zap.Error(encErr))
		}
	}
	return s.repo.CreateReflection(ctx, userID, storageText, mood)
}

func (s *ReflectionService) GetEducationModuleBySlug(ctx context.Context, slug string) (model.EducationModule, error) {
	return s.repo.GetEducationModuleBySlug(ctx, slug)
}
