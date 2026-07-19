package api

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/handler"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/routes"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func New(cfg config.Config, st *store.Store, logger *zap.Logger, clients ...*ent.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	var entClient *ent.Client
	if len(clients) > 0 {
		entClient = clients[0]
	}

	repo := repository.New(entClient, st)
	services := service.NewContainer(repo, cfg, logger)
	mid := middleware.New(services.Auth, logger)
	h := handler.New(services, mid, cfg, logger)

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Audit-Reason"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// CORS must wrap PrivacyGuard so even a rejected request carries the
	// appropriate browser-readable response headers.
	r.Use(mid.PrivacyGuard())

	routes.Register(r, h, mid)

	return r
}
