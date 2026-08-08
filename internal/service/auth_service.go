package service

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"go.uber.org/zap"
)

type AuthNotificationSender interface {
	SendPhoneVerification(context.Context, string, string) error
	SendPasswordReset(context.Context, string, string) error
}

type AuthService struct {
	repo         *repository.Repository
	cfg          config.Config
	logger       *zap.Logger
	notification AuthNotificationSender
}

func NewAuthService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *AuthService {
	return &AuthService{repo: repo, cfg: cfg, logger: logger, notification: NewWhatsAppService(cfg, logger)}
}

func NewAuthServiceWithDependencies(repo *repository.Repository, cfg config.Config, logger *zap.Logger, notification AuthNotificationSender) *AuthService {
	service := NewAuthService(repo, cfg, logger)
	if notification != nil {
		service.notification = notification
	}
	return service
}
