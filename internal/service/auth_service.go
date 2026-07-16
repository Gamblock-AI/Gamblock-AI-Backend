package service

import (
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"go.uber.org/zap"
)

type AuthService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

func NewAuthService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *AuthService {
	return &AuthService{repo: repo, cfg: cfg, logger: logger}
}
