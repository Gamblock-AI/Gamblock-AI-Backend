package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type AdminService struct {
	repo        *repository.Repository
	logger      *zap.Logger
	grantSigner *ProtectionGrantSigner
	whatsapp    *WhatsAppService
}

func NewAdminService(repo *repository.Repository, cfg config.Config, whatsapp *WhatsAppService, logger *zap.Logger) *AdminService {
	if whatsapp == nil {
		whatsapp = NewWhatsAppService(cfg, logger)
	}
	return &AdminService{repo: repo, logger: logger, grantSigner: NewProtectionGrantSigner(cfg), whatsapp: whatsapp}
}

// PlatformAnalytics returns platform-wide aggregate analytics for admin
// dashboards. days must be 14 or 30.
func (s *AdminService) PlatformAnalytics(ctx context.Context, days int) (model.AnalyticsSummary, error) {
	if days != 14 && days != 30 {
		return model.AnalyticsSummary{}, fmt.Errorf("analytics period must be 14 or 30 days")
	}
	return s.repo.PlatformAnalytics(ctx, days, time.Now().UTC())
}

var socialHosts = map[string][]string{
	"instagram": {"instagram.com"},
	"tiktok":    {"tiktok.com"},
	"youtube":   {"youtube.com", "youtu.be"},
	"facebook":  {"facebook.com", "fb.com"},
	"linkedin":  {"linkedin.com"},
	"x":         {"x.com", "twitter.com"},
	"threads":   {"threads.net"},
	"github":    {"github.com"},
}

func (s *AdminService) PublicSocialLinks(ctx context.Context) ([]model.SiteSocialLink, error) {
	return s.repo.ListSiteSocialLinks(ctx, true)
}

func (s *AdminService) SiteSocialLinks(ctx context.Context) ([]model.SiteSocialLink, error) {
	return s.repo.ListSiteSocialLinks(ctx, false)
}

func (s *AdminService) ReplaceSiteSocialLinks(ctx context.Context, actorID, reason string, items []model.SiteSocialLink) ([]model.SiteSocialLink, error) {
	if len(items) > len(socialHosts) {
		return nil, fmt.Errorf("too many social links")
	}
	seen := map[string]bool{}
	for index := range items {
		item := &items[index]
		item.Platform = strings.ToLower(strings.TrimSpace(item.Platform))
		item.Label = strings.TrimSpace(item.Label)
		item.SortOrder = index
		if seen[item.Platform] || socialHosts[item.Platform] == nil || item.Label == "" || len(item.Label) > 80 {
			return nil, fmt.Errorf("invalid social link")
		}
		seen[item.Platform] = true
		if item.URL == nil || strings.TrimSpace(*item.URL) == "" {
			item.URL = nil
			item.Enabled = false
			continue
		}
		value := strings.TrimSpace(*item.URL)
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.User != nil || (parsed.Port() != "" && parsed.Port() != "443") || !allowedSocialQuery(item.Platform, parsed) || parsed.Fragment != "" || !allowedSocialHost(item.Platform, parsed.Hostname()) {
			return nil, fmt.Errorf("social link URL is not allowed")
		}
		item.URL = &value
	}
	if err := s.repo.ReplaceSiteSocialLinks(ctx, actorID, items); err != nil {
		return nil, err
	}
	_ = s.audit(ctx, actorID, "site_social_links_updated", "site_settings", "social_links", reason, map[string]any{"count": len(items)})
	return s.repo.ListSiteSocialLinks(ctx, false)
}

func allowedSocialHost(platform, host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, allowed := range socialHosts[platform] {
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

// Facebook profile IDs are represented as https://facebook.com/profile.php?id=<numeric-id>.
// This is the sole query-bearing social URL accepted by the public-link contract.
func allowedSocialQuery(platform string, parsed *url.URL) bool {
	if parsed.RawQuery == "" {
		return true
	}
	if platform != "facebook" || parsed.Path != "/profile.php" {
		return false
	}
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || len(values) != 1 || len(values["id"]) != 1 || values.Get("id") == "" {
		return false
	}
	_, err = strconv.ParseUint(values.Get("id"), 10, 64)
	return err == nil
}

func (s *AdminService) AuditEvents(ctx context.Context) ([]model.AuditEvent, error) {
	_ = s.repo.PurgeAuditEventsBefore(ctx, time.Now().UTC().AddDate(-2, 0, 0))
	return s.repo.ListAuditEvents(ctx, 200)
}

func (s *AdminService) audit(ctx context.Context, actorID, action, targetType, targetID, reason string, metadata map[string]any) error {
	actor, ok := s.repo.UserByID(ctx, actorID)
	if !ok {
		return fmt.Errorf("audit actor not found")
	}
	return s.repo.SaveAuditEvent(ctx, model.AuditEvent{
		ID: "audit_" + uuid.NewString()[:12], ActorID: actor.ID, Actor: actor.Email,
		Action: action, TargetType: targetType, Target: targetID, Reason: strings.TrimSpace(reason),
		Metadata: metadata, CreatedAt: time.Now().UTC(),
	})
}

func (s *AdminService) RecordAudit(ctx context.Context, actorID, action, targetType, targetID, reason string, metadata map[string]any) error {
	return s.audit(ctx, actorID, action, targetType, targetID, reason, metadata)
}

func (s *AdminService) Accounts(ctx context.Context) ([]model.AdminAccount, error) {
	return s.repo.ListAdminAccounts(ctx)
}

func (s *AdminService) CreateAccount(ctx context.Context, actorID, email, phone, displayName, role, reason string) (model.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	phone = normalizePhone(phone)
	displayName = strings.TrimSpace(displayName)
	if !model.IsAccountRole(role) || email == "" || !strings.Contains(email, "@") || !e164Pattern.MatchString(phone) || displayName == "" || strings.TrimSpace(reason) == "" {
		return model.User{}, "", fmt.Errorf("invalid account")
	}
	if _, exists := s.repo.UserByEmail(ctx, email); exists {
		return model.User{}, "", fmt.Errorf("email already exists")
	}
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return model.User{}, "", err
	}
	temporaryPassword := "Gm!" + base64.RawURLEncoding.EncodeToString(random)
	passwordHash, err := authn.HashPassword(temporaryPassword)
	if err != nil {
		return model.User{}, "", err
	}
	user, err := s.repo.CreateProvisionedUserWithPhone(ctx, "usr_"+uuid.NewString()[:12], email, displayName, phone, passwordHash, role, true)
	if err != nil {
		return model.User{}, "", err
	}
	if err := s.audit(ctx, actorID, "account_created", "account", user.ID, reason, map[string]any{"role": role}); err != nil {
		s.logger.Error("failed to audit account creation", zap.String("account_id", user.ID), zap.Error(err))
	}
	return user, temporaryPassword, nil
}

func (s *AdminService) UpdateAccount(ctx context.Context, actorID, accountID string, disabled bool, reason string) error {
	if actorID == accountID || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("account update is not allowed")
	}
	account, ok := s.repo.UserByID(ctx, accountID)
	if !ok || !model.IsAccountRole(account.Role) {
		return fmt.Errorf("account not found")
	}
	if err := s.repo.SetAccountDisabled(ctx, accountID, disabled, time.Now().UTC()); err != nil {
		return err
	}
	if disabled {
		if err := s.repo.RevokeRefreshTokensForUser(ctx, accountID); err != nil {
			return err
		}
	}
	return s.audit(ctx, actorID, "account_status_updated", "account", accountID, reason, map[string]any{"disabled": disabled})
}

func (s *AdminService) Overview(ctx context.Context, role string) (model.AdminOverview, error) {
	overview := model.AdminOverview{Role: role}
	if role != model.RoleAdmin {
		return overview, fmt.Errorf("admin role is required")
	}
	modules, err := s.repo.GetEducationModules(ctx)
	if err != nil {
		return overview, err
	}
	for _, item := range modules {
		if item.Status == "in_review" {
			overview.ReviewContent++
		}
		if item.Status == "draft" {
			overview.DraftContent++
		}
	}
	cases, err := s.repo.GetSupportCases(ctx)
	if err != nil {
		return overview, err
	}
	for _, item := range cases {
		if item.Status != "resolved" && item.Status != "closed" {
			overview.OpenSupport++
			if item.Owner == "" {
				overview.UnassignedSupport++
			}
		}
	}
	dataRequests, err := s.repo.GetAllDataRequests(ctx)
	if err != nil {
		return overview, err
	}
	for _, item := range dataRequests {
		if item.Status == "failed" {
			overview.FailedDataRequests++
		}
	}
	emergency, err := s.repo.GetPendingEmergencyKeyRequests(ctx, time.Now().UTC())
	if err != nil {
		return overview, err
	}
	overview.PendingEmergency = len(emergency)
	accounts, err := s.repo.ListAdminAccounts(ctx)
	if err != nil {
		return overview, err
	}
	for _, item := range accounts {
		if item.Role == model.RoleAdmin && item.DisabledAt == nil {
			overview.ActiveOperators++
		}
	}
	links, err := s.repo.ListSiteSocialLinks(ctx, true)
	if err != nil {
		return overview, err
	}
	overview.VisibleSocialLinks = len(links)
	return overview, nil
}

func (s *AdminService) GetEducationModules(ctx context.Context) ([]model.EducationModule, error) {
	return s.repo.GetEducationModules(ctx)
}

func (s *AdminService) CreateEducationModule(ctx context.Context, m model.EducationModule) error {
	if m.ID == "" {
		m.ID = "mod_" + uuid.NewString()[:8]
	}
	return s.repo.CreateEducationModule(ctx, m)
}

func (s *AdminService) GetPortalOverview(ctx context.Context) (model.PortalOverview, error) {
	return s.repo.GetPortalOverview(ctx)
}

func (s *AdminService) GenerateEmergencyKey(ctx context.Context, createdBy string) (string, error) {
	return "", fmt.Errorf("direct generation is disabled; request approval from a second platform administrator")
}

func (s *AdminService) RequestEmergencyKey(ctx context.Context, requestedBy, deviceID string) (model.EmergencyKeyRequest, error) {
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, requestedBy) {
		return model.EmergencyKeyRequest{}, fmt.Errorf("device does not belong to user")
	}
	now := time.Now().UTC()
	if current, err := s.repo.GetCurrentEmergencyKeyRequest(ctx, requestedBy, deviceID, now); err == nil &&
		(current.Status == "pending" || current.Status == "reviewed" || current.Status == "approved") {
		return model.EmergencyKeyRequest{}, fmt.Errorf("an active emergency request already exists")
	}
	request := model.EmergencyKeyRequest{
		ID:               "ekr_" + uuid.NewString()[:8],
		RequestedBy:      requestedBy,
		DeviceID:         deviceID,
		Status:           "pending",
		RequestExpiresAt: now.Add(30 * time.Minute),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	created, err := s.repo.CreateEmergencyKeyRequest(ctx, request)
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	s.logger.Info("emergency key requested", zap.String("requested_by", requestedBy), zap.String("request_id", created.ID))

	if s.whatsapp != nil {
		requesterName := requestedBy
		if user, ok := s.repo.UserByID(ctx, requestedBy); ok {
			if strings.TrimSpace(user.DisplayName) != "" {
				requesterName = fmt.Sprintf("%s (%s)", user.DisplayName, user.Email)
			} else if strings.TrimSpace(user.Email) != "" {
				requesterName = user.Email
			}
		}
		if adminPhones, err := s.repo.ListActiveAdminPhones(ctx); err == nil {
			for _, phone := range adminPhones {
				if err := s.whatsapp.SendEmergencyRequestNotificationToAdmin(ctx, phone, requesterName, deviceID, created.ID); err != nil {
					s.logger.Warn("failed to notify admin of emergency request via whatsapp",
						zap.String("request_id", created.ID),
						zap.Error(err),
					)
				}
			}
		}
	}

	return created, nil
}

func (s *AdminService) GetCurrentEmergencyKeyRequest(ctx context.Context, requestedBy, deviceID string) (model.EmergencyKeyRequest, error) {
	if !s.repo.IsDeviceOwnedBy(ctx, deviceID, requestedBy) {
		return model.EmergencyKeyRequest{}, fmt.Errorf("device does not belong to user")
	}
	return s.repo.GetCurrentEmergencyKeyRequest(ctx, requestedBy, deviceID, time.Now().UTC())
}

func (s *AdminService) GetPendingEmergencyKeyRequests(ctx context.Context) ([]model.EmergencyKeyRequest, error) {
	return s.repo.GetPendingEmergencyKeyRequests(ctx, time.Now().UTC())
}

func (s *AdminService) ReviewEmergencyKeyRequest(ctx context.Context, requestID, reviewedBy string) (model.EmergencyKeyRequest, error) {
	request, err := s.repo.ReviewEmergencyKeyRequest(ctx, requestID, reviewedBy, time.Now().UTC())
	if err != nil {
		return model.EmergencyKeyRequest{}, err
	}
	s.logger.Info("emergency key request reviewed",
		zap.String("request_id", request.ID),
		zap.String("requested_by", request.RequestedBy),
		zap.String("reviewed_by", reviewedBy),
	)
	return request, nil
}

func (s *AdminService) ApproveEmergencyKeyRequest(ctx context.Context, requestID, approvedBy string) (model.EmergencyKeyRequest, string, error) {
	key, err := generateEmergencyKeyString()
	if err != nil {
		return model.EmergencyKeyRequest{}, "", err
	}
	now := time.Now().UTC()
	request, err := s.repo.ApproveEmergencyKeyRequest(
		ctx,
		requestID,
		approvedBy,
		HashRefreshToken(key),
		now,
		now.Add(24*time.Hour),
	)
	if err != nil {
		return model.EmergencyKeyRequest{}, "", err
	}
	s.logger.Info("emergency key approved",
		zap.String("request_id", request.ID),
		zap.String("requested_by", request.RequestedBy),
		zap.String("reviewed_by", request.ReviewedBy),
		zap.String("approved_by", approvedBy),
	)

	if s.whatsapp != nil {
		if user, ok := s.repo.UserByID(ctx, request.RequestedBy); ok && strings.TrimSpace(user.PhoneE164) != "" {
			if err := s.whatsapp.SendEmergencyKey(ctx, user.PhoneE164, key); err != nil {
				s.logger.Warn("failed to deliver emergency key via whatsapp",
					zap.String("request_id", request.ID),
					zap.String("user_id", request.RequestedBy),
					zap.Error(err),
				)
			} else {
				s.logger.Info("emergency key delivered via whatsapp",
					zap.String("request_id", request.ID),
					zap.String("user_id", request.RequestedBy),
				)
			}
		}
	}

	return request, key, nil
}

func (s *AdminService) ValidateEmergencyKey(ctx context.Context, key, deviceID string) (model.EmergencyGrant, error) {
	if err := s.grantSigner.Ready(); err != nil {
		return model.EmergencyGrant{}, err
	}
	now := time.Now().UTC()
	keyHash := HashRefreshToken(key)
	request, err := s.repo.GetUsableEmergencyKeyRequest(ctx, keyHash, deviceID, now)
	if err != nil {
		return model.EmergencyGrant{}, err
	}
	thumbprint, err := s.repo.OwnedDeviceGrantKeyThumbprint(ctx, request.RequestedBy, deviceID)
	if err != nil {
		return model.EmergencyGrant{}, err
	}
	grantJTI, err := newProtectionGrantJTI()
	if err != nil {
		return model.EmergencyGrant{}, err
	}
	request, err = s.repo.UseEmergencyKey(ctx, keyHash, deviceID, now, now.Add(10*time.Minute), grantJTI)
	if err != nil {
		return model.EmergencyGrant{}, err
	}
	if request.GrantStartsAt == nil || request.GrantExpiresAt == nil || request.GrantJTI == "" {
		return model.EmergencyGrant{}, fmt.Errorf("emergency grant metadata is incomplete")
	}
	grantToken, err := s.grantSigner.Sign(
		request.ID,
		deviceID,
		"emergency_access",
		thumbprint,
		request.GrantJTI,
		*request.GrantStartsAt,
		*request.GrantExpiresAt,
	)
	if err != nil {
		return model.EmergencyGrant{}, err
	}
	s.logger.Info("emergency key used for device unlock",
		zap.String("device_id", deviceID),
		zap.String("request_id", request.ID),
		zap.String("requested_by", request.RequestedBy),
		zap.String("approved_by", request.ApprovedBy),
	)
	return model.EmergencyGrant{
		RequestID: request.ID, DeviceID: deviceID, Action: "emergency_access",
		GrantStartsAt: *request.GrantStartsAt, GrantExpiresAt: *request.GrantExpiresAt,
		GrantJTI: request.GrantJTI, GrantToken: grantToken,
	}, nil
}

func generateEmergencyKeyString() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 12)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}
