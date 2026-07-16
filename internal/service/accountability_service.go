package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type AccountabilityService struct {
	repo     *repository.Repository
	cfg      config.Config
	whatsapp *WhatsAppService
	logger   *zap.Logger
}

func NewAccountabilityService(repo *repository.Repository, cfg config.Config, whatsapp *WhatsAppService, logger *zap.Logger) *AccountabilityService {
	return &AccountabilityService{repo: repo, cfg: cfg, whatsapp: whatsapp, logger: logger}
}

func (s *AccountabilityService) GetPartners(ctx context.Context, userID string) (*model.Partner, []model.Partner, error) {
	return s.repo.GetPartners(ctx, userID)
}

func (s *AccountabilityService) CreatePartnerInvitation(ctx context.Context, userID, email, phone string) (model.Partner, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return model.Partner{}, "", fmt.Errorf("partner email is required")
	}
	owner, ok := s.repo.UserByID(ctx, userID)
	if !ok || strings.EqualFold(owner.Email, email) {
		return model.Partner{}, "", fmt.Errorf("partner must use a different account")
	}
	_, existing, err := s.repo.GetPartners(ctx, userID)
	if err != nil {
		return model.Partner{}, "", err
	}
	for _, relationship := range existing {
		if strings.EqualFold(relationship.PartnerEmail, email) && (relationship.Status == "active" || relationship.Status == "invited") {
			return model.Partner{}, "", fmt.Errorf("an active relationship or invitation already exists for this email")
		}
	}
	plID := "pl_" + uuid.NewString()
	rawToken := "pinv_" + uuid.NewString()
	tokenHash := HashRefreshToken(rawToken)
	var pVal *string
	if phone != "" {
		pVal = &phone
	}
	partner, err := s.repo.CreatePartnerInvitation(ctx, plID, userID, email, pVal, tokenHash)
	if err != nil {
		return model.Partner{}, "", err
	}
	inviteURL := fmt.Sprintf("%s/partner/invitations/%s", s.cfg.PublicWebBaseURL, rawToken)
	return partner, inviteURL, nil
}

func (s *AccountabilityService) AcceptInvitation(ctx context.Context, token, partnerUserID string) error {
	tokenHash := HashRefreshToken(token)
	linkID, err := s.repo.GetPartnerLinkByToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("invalid token or invitation already accepted")
	}
	return s.repo.AcceptPartnerInvitation(ctx, linkID, partnerUserID)
}

func (s *AccountabilityService) RevokePartner(ctx context.Context, partnerLinkID, userID string) error {
	return s.repo.RevokePartner(ctx, partnerLinkID, userID)
}

func (s *AccountabilityService) GetApprovalRequests(ctx context.Context, userID string) ([]model.ApprovalRequest, error) {
	return s.repo.GetApprovalRequests(ctx, userID)
}

func (s *AccountabilityService) CreateApprovalRequest(ctx context.Context, userID, deviceID, partnerLinkID, action, reason string, duration int) error {
	allowedActions := map[string]bool{
		"disable_protection": true, "remove_partner": true, "uninstall_detected": true,
		"reset_settings": true, "pause_protection": true, "emergency_access": true,
	}
	if !allowedActions[action] || !s.repo.IsActivePartnerLinkOwnedBy(ctx, partnerLinkID, userID) {
		return fmt.Errorf("invalid approval request relationship or action")
	}
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, userID) {
		return fmt.Errorf("device does not belong to user")
	}
	if duration < 0 || duration > 24*60 {
		return fmt.Errorf("requested duration is outside the allowed range")
	}
	reqID := "APR-" + uuid.NewString()[:8]
	quickToken := generateQuickToken()
	quickTokenHash := HashRefreshToken(quickToken)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	if err := s.repo.CreateApprovalRequestWithToken(ctx, reqID, userID, deviceID, partnerLinkID, action, reason, duration, expiresAt, quickTokenHash); err != nil {
		return err
	}

	// Generate Quick Link
	quickLink := fmt.Sprintf("%s/approve/%s", s.cfg.PublicWebBaseURL, quickToken)

	// Send WhatsApp notification to partner
	summary := ApprovalSummary{
		MemberName: userID,
		Action:     action,
		QuickLink:  quickLink,
	}
	phone := s.repo.GetActivePartnerPhone(ctx, partnerLinkID, userID)
	if phone != "" {
		if err := s.whatsapp.SendSingleApproval(ctx, phone, summary); err != nil {
			s.logger.Warn("approval notification was not delivered", zap.String("request_id", reqID), zap.Error(err))
		}
	}

	return nil
}

func (s *AccountabilityService) CancelApprovalRequest(ctx context.Context, id, userID string) error {
	return s.repo.CancelApprovalRequest(ctx, id, userID)
}

func (s *AccountabilityService) ResolveApprovalAsPartner(ctx context.Context, id, status, partnerUserID string) error {
	return s.repo.ResolveApprovalAsPartner(ctx, id, partnerUserID, status)
}

func (s *AccountabilityService) VerifyQuickToken(ctx context.Context, token string) (map[string]any, error) {
	tokenHash := HashRefreshToken(token)
	req, err := s.repo.GetApprovalByQuickToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("token tidak valid atau sudah digunakan")
	}
	if req.Status != "pending" || req.ExpiresAt.IsZero() || !time.Now().UTC().Before(req.ExpiresAt) {
		return nil, fmt.Errorf("token tidak valid atau sudah kedaluwarsa")
	}

	return map[string]any{
		"request_id":                 req.ID,
		"action":                     req.Action,
		"reason":                     req.Reason,
		"requested_duration_minutes": req.RequestedDurationMinutes,
		"status":                     req.Status,
		"created_at":                 req.CreatedAt,
	}, nil
}

func (s *AccountabilityService) ResolveByToken(ctx context.Context, token, status string) error {
	tokenHash := HashRefreshToken(token)
	req, err := s.repo.GetApprovalByQuickToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("token tidak valid")
	}
	if req.Status != "pending" {
		return fmt.Errorf("permohonan sudah diproses sebelumnya")
	}
	if req.ExpiresAt.IsZero() || !time.Now().UTC().Before(req.ExpiresAt) {
		return fmt.Errorf("permohonan sudah kedaluwarsa")
	}
	if err := s.repo.UpdateApprovalRequest(ctx, req.ID, status, "quick_token"); err != nil {
		return err
	}
	req.Status = status
	s.repo.UpdateQuickTokenState(tokenHash, req)
	return nil
}

func generateQuickToken() string {
	return "qapp_" + uuid.NewString()
}
