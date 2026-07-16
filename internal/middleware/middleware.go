package middleware

import (
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"go.uber.org/zap"
)

type Middleware struct {
	auth   *service.AuthService
	logger *zap.Logger
}

func New(auth *service.AuthService, logger *zap.Logger) *Middleware {
	return &Middleware{auth: auth, logger: logger}
}
