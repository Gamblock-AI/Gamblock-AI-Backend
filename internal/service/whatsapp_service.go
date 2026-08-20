package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

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
		client: &http.Client{Timeout: 15 * time.Second},
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

	messageBody := buildBatchMessage(summaries)
	return s.sendText(ctx, phone, messageBody)
}

func (s *WhatsAppService) SendSingleApproval(ctx context.Context, phone string, summary ApprovalSummary) error {
	return s.SendApprovalBatch(ctx, phone, []ApprovalSummary{summary})
}

func (s *WhatsAppService) SendPhoneVerification(ctx context.Context, phone, code string) error {
	if s.cfg.NotificationMode == "demo" {
		return nil
	}
	if phone == "" {
		return fmt.Errorf("whatsapp verification delivery is not configured")
	}
	return s.sendText(ctx, phone, "Kode verifikasi Gamblock-AI: "+code+". Berlaku 10 menit. Jangan bagikan kode ini.")
}

func (s *WhatsAppService) SendPasswordReset(ctx context.Context, phone, code string) error {
	return s.sendText(ctx, phone, "Kode pemulihan kata sandi Gamblock-AI: "+code+". Berlaku 30 menit. Jangan bagikan kode ini.")
}

func (s *WhatsAppService) SendDataRequestConfirmation(ctx context.Context, phone, confirmationURL string) error {
	return s.sendText(ctx, phone, "Konfirmasi permintaan penghapusan data Gamblock-AI melalui tautan berikut. Tautan berlaku 30 menit:\n"+confirmationURL)
}

func (s *WhatsAppService) SendDataExportReady(ctx context.Context, phone, accountURL string) error {
	return s.sendText(ctx, phone, "Ekspor data Gamblock-AI Anda sudah siap. Masuk ke akun untuk mengunduhnya:\n"+accountURL)
}

func (s *WhatsAppService) SendEmergencyKey(ctx context.Context, phone, key string) error {
	message := fmt.Sprintf("*Gamblock-AI - Kunci Akses Darurat*\n\nKode pemulihan darurat Anda adalah:\n*%s*\n\n📌 *Ketentuan & Kegunaan:*\n1. Masukkan kode ini pada menu Pemulihan Darurat di aplikasi Gamblock-AI dalam 24 jam.\n2. Setelah diaktivasi di aplikasi, Anda mendapatkan *akses penuh selama 10 menit* untuk mematikan Admin Perangkat, Aksesibilitas, atau mencopot pemasangan (uninstall) aplikasi tanpa dicegat proteksi.", key)
	return s.sendText(ctx, phone, message)
}

func (s *WhatsAppService) SendEmergencyRequestNotificationToAdmin(ctx context.Context, phone, requesterName, deviceID, requestID string) error {
	message := fmt.Sprintf("*Gamblock-AI - Permintaan Akses Darurat*\n\nAda permohonan akses darurat baru dari pengguna:\n- *Pengguna:* %s\n- *Perangkat:* %s\n- *ID Permintaan:* %s\n\nSilakan masuk ke panel admin (/admin/emergency) untuk meninjau dan menerbitkan kunci darurat.", requesterName, deviceID, requestID)
	return s.sendText(ctx, phone, message)
}

func (s *WhatsAppService) sendText(ctx context.Context, phone, message string) error {
	if s.cfg.NotificationMode == "demo" {
		return nil
	}
	if strings.TrimSpace(s.cfg.FonnteToken) == "" {
		return fmt.Errorf("Fonnte token is not configured")
	}
	target, err := normalizeFonnteTarget(phone, s.cfg.FonnteCountryCode)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("target", target)
	_ = writer.WriteField("message", message)
	_ = writer.WriteField("countryCode", "0")
	_ = writer.WriteField("preview", "false")
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to prepare Fonnte request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.FonnteBaseURL, "/")+"/send", &body)
	if err != nil {
		return fmt.Errorf("failed to create Fonnte request")
	}
	req.Header.Set("Authorization", s.cfg.FonnteToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("fonnte notification request failed", zap.Error(err))
		return fmt.Errorf("Fonnte notification delivery failed")
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return fmt.Errorf("Fonnte notification response unreadable")
	}
	var result struct {
		Status bool   `json:"status"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || resp.StatusCode >= http.StatusBadRequest || !result.Status {
		s.logger.Warn("fonnte notification rejected", zap.Int("status", resp.StatusCode))
		return fmt.Errorf("Fonnte notification delivery rejected")
	}
	return nil
}

func normalizeFonnteTarget(phone, countryCode string) (string, error) {
	value := strings.TrimSpace(phone)
	value = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(value)
	if strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	if strings.HasPrefix(value, "0") && countryCode != "" {
		value = strings.TrimPrefix(value, "0")
		value = countryCode + value
	}
	if len(value) < 8 || len(value) > 15 {
		return "", fmt.Errorf("phone must use a valid international format")
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return "", fmt.Errorf("phone must use a valid international format")
		}
	}
	return value, nil
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
