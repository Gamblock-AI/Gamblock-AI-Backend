package service

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/api/idtoken"
)

type GoogleTokenVerifier interface {
	Validate(context.Context, string, string) (*idtoken.Payload, error)
}

type googleTokenVerifier struct{}

func (googleTokenVerifier) Validate(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, token, audience)
}

type AuthEmailSender interface {
	SendVerification(context.Context, string, string) error
	SendPasswordReset(context.Context, string, string) error
}

type AuthService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
	google GoogleTokenVerifier
	email  AuthEmailSender
}

func NewAuthService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *AuthService {
	return &AuthService{repo: repo, cfg: cfg, logger: logger, google: googleTokenVerifier{}, email: NewEmailService(cfg)}
}

func NewAuthServiceWithDependencies(repo *repository.Repository, cfg config.Config, logger *zap.Logger, google GoogleTokenVerifier, email AuthEmailSender) *AuthService {
	service := NewAuthService(repo, cfg, logger)
	if google != nil {
		service.google = google
	}
	if email != nil {
		service.email = email
	}
	return service
}
