package api

import (
	"context"
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
	r.Use(gin.Recovery(), mid.RequestID(), mid.PrivacyGuard())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Accept", "X-Audit-Reason"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.Register(r, h, mid)

	// Start notification batch scheduler
	whatsapp := service.NewWhatsAppService(cfg, logger)
	go startNotificationBatcher(repo, whatsapp, cfg, logger)

	return r
}

func startNotificationBatcher(repo *repository.Repository, whatsapp *service.WhatsAppService, cfg config.Config, logger *zap.Logger) {
	interval := 6 * time.Hour
	if cfg.NotificationMode == "demo" {
		interval = 5 * time.Minute // faster in demo mode for testing
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("notification batcher started", zap.Duration("interval", interval))

	for range ticker.C {
		ctx := context.Background()
		pending, err := repo.GetPendingBatchApprovals(ctx)
		if err != nil {
			logger.Warn("batcher: failed to fetch pending approvals", zap.Error(err))
			continue
		}
		if len(pending) == 0 {
			continue
		}

		byPartner := make(map[string][]service.ApprovalSummary)
		for _, req := range pending {
			byPartner[req.PartnerPhone] = append(byPartner[req.PartnerPhone], service.ApprovalSummary{
				MemberName: req.MemberName,
				Action:     req.Action,
				QuickLink:  req.QuickLink,
			})
		}

		for phone, summaries := range byPartner {
			if err := whatsapp.SendApprovalBatch(ctx, phone, summaries); err != nil {
				logger.Warn("batcher: send failed", zap.String("phone", phone), zap.Error(err))
			}
		}
		logger.Info("batcher: sent", zap.Int("partners", len(byPartner)), zap.Int("requests", len(pending)))
	}
}
