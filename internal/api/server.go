package api

import (
	"context"
	"net/http"
	"net/url"
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

	if cfg.VAPIDPublicKey != "" && cfg.VAPIDPrivateKey != "" {
		scheduler := service.NewReminderScheduler(repo, services.Push, logger)
		go scheduler.Run(context.Background())
	}

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID())
	corsConfig := cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Audit-Reason"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	// Outside production, accept any localhost/127.0.0.1 origin regardless of
	// port so local browser clients (e.g. `flutter run -d chrome --web-port
	// 45051`) never need a `.env` CORS edit or a server restart. Production
	// stays strict: only the explicit AllowedOrigins list is accepted.
	if !cfg.IsProduction() {
		corsConfig.AllowOriginFunc = func(origin string) bool {
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			host := u.Hostname()
			return host == "localhost" || host == "127.0.0.1"
		}
	}
	r.Use(cors.New(corsConfig))
	// CORS must wrap PrivacyGuard so even a rejected request carries the
	// appropriate browser-readable response headers.
	r.Use(mid.PrivacyGuard())

	routes.Register(r, h, mid)

	return r
}
