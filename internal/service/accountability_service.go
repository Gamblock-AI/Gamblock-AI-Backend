package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

func (s *AccountabilityService) CreatePartnerInvitation(ctx context.Context, userID, email, phone string) (model.Partner, error) {
	plID := "pl_" + uuid.NewString()
	rawToken := "pinv_" + uuid.NewString()
	tokenHash := HashRefreshToken(rawToken)
	var pVal *string
	if phone != "" {
		pVal = &phone
	}
	partner, err := s.repo.CreatePartnerInvitation(ctx, plID, userID, email, pVal, tokenHash)
	if err != nil {
		return model.Partner{}, err
	}
	return partner, nil
}

func (s *AccountabilityService) AcceptInvitation(ctx context.Context, token, partnerUserID string) error {
	tokenHash := HashRefreshToken(token)
	linkID, err := s.repo.GetPartnerLinkByToken(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("invalid token or invitation already accepted")
	}
	return s.repo.AcceptPartnerInvitation(ctx, linkID, partnerUserID)
}

func (s *AccountabilityService) RevokePartner(ctx context.Context, partnerLinkID string) error {
	return s.repo.RevokePartner(ctx, partnerLinkID)
}

func (s *AccountabilityService) GetApprovalRequests(ctx context.Context, userID string) ([]model.ApprovalRequest, error) {
	return s.repo.GetApprovalRequests(ctx, userID)
}

func (s *AccountabilityService) CreateApprovalRequest(ctx context.Context, userID, deviceID, partnerLinkID, action, reason string, duration int) error {
	reqID := "APR-" + uuid.NewString()[:8]
	quickToken := generateQuickToken()
	quickTokenHash := HashRefreshToken(quickToken)
	expiresAt := time.Now().UTC().Add(30 * time.Minute)

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
	_ = s.whatsapp.SendSingleApproval(ctx, "", summary)

	return nil
}

func (s *AccountabilityService) ResolveApprovalRequest(ctx context.Context, id, status, resolvedBy string) error {
	return s.repo.UpdateApprovalRequest(ctx, id, status, resolvedBy)
}

func (s *AccountabilityService) VerifyQuickToken(ctx context.Context, token string) (map[string]any, error) {
	tokenHash := HashRefreshToken(token)
	req, err := s.repo.GetApprovalByQuickToken(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("token tidak valid atau sudah digunakan")
	}

	return map[string]any{
		"request_id":                req.ID,
		"action":                    req.Action,
		"reason":                    req.Reason,
		"requested_duration_minutes": req.Duration,
		"status":                    req.Status,
		"created_at":                req.CreatedAt,
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
	return s.repo.UpdateApprovalRequest(ctx, req.ID, status, "quick_token")
}

func generateQuickToken() string {
	return "qapp_" + uuid.NewString()
}
