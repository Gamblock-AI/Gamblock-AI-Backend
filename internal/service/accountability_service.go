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

func (s *AccountabilityService) CreateApprovalRequest(ctx context.Context, userID, deviceID, membershipID, action, reason string, duration int) (model.ApprovalRequest, error) {
	allowedActions := map[string]bool{
		"uninstall_detected": true, "pause_protection": true,
	}
	membership, membershipErr := s.repo.MembershipByID(ctx, membershipID)
	if !allowedActions[action] || membershipErr != nil || membership.StudentID != userID || membership.Status != "active" {
		return model.ApprovalRequest{}, fmt.Errorf("invalid approval request relationship or action")
	}
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, userID) {
		return model.ApprovalRequest{}, fmt.Errorf("device does not belong to user")
	}
	if action == "pause_protection" {
		allowedDurations := map[int]bool{15: true, 30: true, 60: true, 120: true}
		if !allowedDurations[duration] {
			return model.ApprovalRequest{}, fmt.Errorf("pause duration must be 15, 30, 60, or 120 minutes")
		}
	} else if duration != 0 {
		return model.ApprovalRequest{}, fmt.Errorf("requested duration is only valid for pause protection")
	}
	reqID := "APR-" + uuid.NewString()[:8]
	quickToken := generateQuickToken()
	quickTokenHash := HashRefreshToken(quickToken)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	request, err := s.repo.CreateApprovalRequestWithToken(ctx, reqID, userID, deviceID, membershipID, action, reason, duration, expiresAt, quickTokenHash)
	if err != nil {
		return model.ApprovalRequest{}, err
	}

	// Generate Quick Link
	quickLink := fmt.Sprintf("%s/approve/%s", s.cfg.PublicWebBaseURL, quickToken)

	// Send WhatsApp notification to partner
	summary := ApprovalSummary{
		MemberName: userID,
		Action:     action,
		QuickLink:  quickLink,
	}
	group, groupErr := s.repo.AccountabilityGroupByID(ctx, membership.GroupID)
	phone := ""
	if groupErr == nil {
		if partner, ok := s.repo.UserByID(ctx, group.OwnerPartnerID); ok && partner.PhoneVerifiedAt != nil {
			phone = partner.PhoneE164
		}
	}
	if phone != "" {
		if err := s.whatsapp.SendSingleApproval(ctx, phone, summary); err != nil {
			s.logger.Warn("approval notification was not delivered", zap.String("request_id", reqID), zap.Error(err))
		}
	}

	return request, nil
}

func (s *AccountabilityService) CancelApprovalRequest(ctx context.Context, id, userID string) error {
	return s.repo.CancelApprovalRequest(ctx, id, userID)
}

func (s *AccountabilityService) ResolveApprovalAsPartner(ctx context.Context, id, status, partnerUserID string, response ...string) error {
	supportiveResponse := ""
	if len(response) > 0 {
		supportiveResponse = response[0]
	}
	if len(strings.TrimSpace(supportiveResponse)) > 500 {
		return fmt.Errorf("supportive response is too long")
	}
	return s.repo.ResolveApprovalAsPartner(ctx, id, partnerUserID, status, strings.TrimSpace(supportiveResponse))
}

func (s *AccountabilityService) ApplyApprovedRequest(ctx context.Context, id, userID, deviceID string) (model.ApprovalGrant, error) {
	if deviceID == "" {
		return model.ApprovalGrant{}, fmt.Errorf("device id is required")
	}
	return s.repo.ApplyApprovedRequest(ctx, id, userID, deviceID, time.Now().UTC())
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
