package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
)

type WhatsAppService struct {
	cfg    config.Config
	logger *zap.Logger
	client *http.Client
}

func NewWhatsAppService(cfg config.Config, logger *zap.Logger) *WhatsAppService {
	return &WhatsAppService{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{},
	}
}

type ApprovalSummary struct {
	MemberName string
	Action     string
	QuickLink  string
}

func (s *WhatsAppService) SendApprovalBatch(ctx context.Context, phone string, summaries []ApprovalSummary) error {
	if s.cfg.NotificationMode == "demo" {
		s.logger.Info("whatsapp: demo mode - logging instead of sending",
			zap.Int("pending_requests", len(summaries)),
		)
		for _, summary := range summaries {
			s.logger.Info("whatsapp: pending approval",
				zap.String("member", summary.MemberName),
				zap.String("action", summary.Action),
			)
		}
		return nil
	}
	if phone == "" {
		return fmt.Errorf("partner phone is not configured")
	}

	if s.cfg.WhatsAppAPIKey == "" || s.cfg.WhatsAppPhoneID == "" {
		return fmt.Errorf("whatsapp API credentials are not configured")
	}

	messageBody := buildBatchMessage(summaries)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                phone,
		"type":              "text",
		"text": map[string]interface{}{
			"body": messageBody,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal whatsapp payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", s.cfg.WhatsAppBaseURL, s.cfg.WhatsAppPhoneID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.cfg.WhatsAppAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error("whatsapp: request failed", zap.Error(err))
		return fmt.Errorf("failed to send whatsapp request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Error("whatsapp: API rejected message",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return fmt.Errorf("whatsapp API returned status %d: %s", resp.StatusCode, string(body))
	}

	s.logger.Info("whatsapp: message sent successfully",
		zap.String("recipient", phone),
	)

	return nil
}

func (s *WhatsAppService) SendSingleApproval(ctx context.Context, phone string, summary ApprovalSummary) error {
	return s.SendApprovalBatch(ctx, phone, []ApprovalSummary{summary})
}

func (s *WhatsAppService) SendPhoneVerification(ctx context.Context, phone, code string) error {
	if s.cfg.NotificationMode == "demo" {
		return nil
	}
	if phone == "" || s.cfg.WhatsAppAPIKey == "" || s.cfg.WhatsAppPhoneID == "" {
		return fmt.Errorf("whatsapp verification delivery is not configured")
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                phone,
		"type":              "text",
		"text":              map[string]any{"body": "Kode verifikasi Gamblock-AI: " + code + ". Berlaku 10 menit. Jangan bagikan kode ini."},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to prepare verification message: %w", err)
	}
	url := fmt.Sprintf("%s/%s/messages", s.cfg.WhatsAppBaseURL, s.cfg.WhatsAppPhoneID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create verification request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.WhatsAppAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send verification message: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("whatsapp verification delivery was rejected")
	}
	return nil
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
