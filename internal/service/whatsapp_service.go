package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
)

type WhatsAppService struct {
	cfg    config.Config
	logger *zap.Logger
}

func NewWhatsAppService(cfg config.Config, logger *zap.Logger) *WhatsAppService {
	return &WhatsAppService{cfg: cfg, logger: logger}
}

type ApprovalSummary struct {
	MemberName string
	Action     string
	QuickLink  string
}

func (s *WhatsAppService) SendApprovalBatch(ctx context.Context, phone string, summaries []ApprovalSummary) error {
	if s.cfg.NotificationMode == "demo" || phone == "" {
		s.logger.Info("whatsapp: demo mode - logging instead of sending",
			zap.String("recipient", phone),
			zap.Int("pending_requests", len(summaries)),
		)
		for _, summary := range summaries {
			s.logger.Info("whatsapp: pending approval",
				zap.String("member", summary.MemberName),
				zap.String("action", summary.Action),
				zap.String("quick_link", summary.QuickLink),
			)
		}
		return nil
	}

	message := buildBatchMessage(summaries)
	s.logger.Info("whatsapp: sending batch message",
		zap.String("recipient", phone),
		zap.String("message", message),
	)

	// TODO: Integrate with WhatsApp Business API when credentials available
	// POST to https://graph.facebook.com/v18.0/{PHONE_ID}/messages
	return fmt.Errorf("whatsapp API not yet configured - set NOTIFICATION_MODE=demo for local testing")
}

func (s *WhatsAppService) SendSingleApproval(ctx context.Context, phone string, summary ApprovalSummary) error {
	return s.SendApprovalBatch(ctx, phone, []ApprovalSummary{summary})
}

func buildBatchMessage(summaries []ApprovalSummary) string {
	msg := "*Gamblock AI - Permohonan Izin Pencopotan*\n\n"
	msg += fmt.Sprintf("Anda memiliki *%d* permohonan yang menunggu persetujuan:\n\n", len(summaries))
	for i, s := range summaries {
		msg += fmt.Sprintf("%d. *%s* - %s\n   %s\n\n", i+1, s.MemberName, s.Action, s.QuickLink)
	}
	msg += "Klik tautan di atas untuk menyetujui atau menolak."
	return msg
}
