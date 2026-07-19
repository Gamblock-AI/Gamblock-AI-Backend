package service

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
)

type EmailService struct {
	cfg config.Config
}

func NewEmailService(cfg config.Config) *EmailService {
	return &EmailService{cfg: cfg}
}

func (s *EmailService) SendVerification(_ context.Context, recipient, verificationURL string) error {
	if s.cfg.NotificationMode == "demo" {
		return nil
	}
	if s.cfg.SMTPHost == "" || s.cfg.SMTPPort == "" || s.cfg.SMTPFrom == "" {
		return fmt.Errorf("SMTP is not configured")
	}
	addr := s.cfg.SMTPHost + ":" + s.cfg.SMTPPort
	var auth smtp.Auth
	if s.cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}
	body := "From: " + s.cfg.SMTPFrom + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: Verify your Gamblock-AI email\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Verify your email to activate accountability features:\r\n" + verificationURL + "\r\n\r\n" +
		"This link expires in 30 minutes. If you did not request it, ignore this email."
	return smtp.SendMail(addr, auth, s.cfg.SMTPFrom, []string{recipient}, []byte(body))
}

func (s *EmailService) SendOperatorInvitation(_ context.Context, recipient, invitationURL string) error {
	return s.sendLink(recipient, "Your Gamblock-AI operator invitation",
		"You were invited to a restricted Gamblock-AI operations role. Create a separate operator account using this link:",
		invitationURL, "This link expires in 24 hours. If you did not expect it, ignore this email.")
}

func (s *EmailService) SendDataRequestConfirmation(_ context.Context, recipient, confirmationURL string) error {
	return s.sendLink(recipient, "Confirm your Gamblock-AI data deletion",
		"Confirm the permanent deletion request from your authenticated Gamblock-AI account:",
		confirmationURL, "This link expires in 30 minutes. If you did not request deletion, ignore this email and contact support.")
}

func (s *EmailService) SendDataExportReady(_ context.Context, recipient, accountURL string) error {
	return s.sendLink(recipient, "Your Gamblock-AI data export is ready",
		"Your encrypted export is ready. Sign in to your account to download it:",
		accountURL, "The export is removed after seven days and never contains browsing history.")
}

func (s *EmailService) sendLink(recipient, subject, intro, link, footer string) error {
	if s.cfg.NotificationMode == "demo" {
		return nil
	}
	if s.cfg.SMTPHost == "" || s.cfg.SMTPPort == "" || s.cfg.SMTPFrom == "" {
		return fmt.Errorf("SMTP is not configured")
	}
	addr := s.cfg.SMTPHost + ":" + s.cfg.SMTPPort
	var auth smtp.Auth
	if s.cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	}
	body := "From: " + s.cfg.SMTPFrom + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		intro + "\r\n" + link + "\r\n\r\n" + footer
	return smtp.SendMail(addr, auth, s.cfg.SMTPFrom, []string{recipient}, []byte(body))
}
