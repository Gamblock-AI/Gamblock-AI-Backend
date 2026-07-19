package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
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
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

func NewAdminService(repo *repository.Repository, logger *zap.Logger) *AdminService {
	return &AdminService{repo: repo, logger: logger}
}

func NewAdminServiceWithConfig(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *AdminService {
	return &AdminService{repo: repo, cfg: cfg, logger: logger}
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

var specialistOperatorRoles = map[string]bool{
	"content_admin": true, "model_release_operator": true, "support_operator": true,
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
		if err != nil || parsed.Scheme != "https" || parsed.User != nil || (parsed.Port() != "" && parsed.Port() != "443") || parsed.RawQuery != "" || parsed.Fragment != "" || !allowedSocialHost(item.Platform, parsed.Hostname()) {
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

func (s *AdminService) Operators(ctx context.Context) ([]model.OperatorAccount, []model.OperatorInvitation, error) {
	accounts, err := s.repo.ListOperatorAccounts(ctx)
	if err != nil {
		return nil, nil, err
	}
	invitations, err := s.repo.ListOperatorInvitations(ctx)
	return accounts, invitations, err
}

func (s *AdminService) InviteOperator(ctx context.Context, actorID, email, role, reason string) (model.OperatorInvitation, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !specialistOperatorRoles[role] || email == "" || !strings.Contains(email, "@") {
		return model.OperatorInvitation{}, "", fmt.Errorf("invalid operator invitation")
	}
	if _, exists := s.repo.UserByEmail(ctx, email); exists {
		return model.OperatorInvitation{}, "", fmt.Errorf("operator email is already registered")
	}
	rawToken := "operator_" + uuid.NewString() + uuid.NewString()
	now := time.Now().UTC()
	item := model.OperatorInvitation{ID: "opi_" + uuid.NewString()[:12], Email: email, Role: role,
		TokenHash: HashRefreshToken(rawToken), Status: "pending", InvitedBy: actorID,
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	if err := s.repo.SaveOperatorInvitation(ctx, item); err != nil {
		return model.OperatorInvitation{}, "", err
	}
	invitationURL := s.cfg.PublicWebBaseURL + "/operator/invitations/" + rawToken
	if err := NewEmailService(s.cfg).SendOperatorInvitation(ctx, email, invitationURL); err != nil {
		return model.OperatorInvitation{}, "", err
	}
	_ = s.audit(ctx, actorID, "operator_invited", "operator_invitation", item.ID, reason, map[string]any{"role": role})
	if s.cfg.NotificationMode == "demo" {
		return item, invitationURL, nil
	}
	return item, "", nil
}

func (s *AdminService) OperatorInvitation(ctx context.Context, rawToken string) (model.OperatorInvitation, error) {
	item, err := s.repo.OperatorInvitationByToken(ctx, HashRefreshToken(strings.TrimSpace(rawToken)))
	if err != nil || item.Status != "pending" || !time.Now().UTC().Before(item.ExpiresAt) {
		return model.OperatorInvitation{}, fmt.Errorf("operator invitation is invalid or expired")
	}
	item.TokenHash = ""
	return item, nil
}

func (s *AdminService) AcceptOperatorInvitation(ctx context.Context, rawToken, displayName, password string) (model.User, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len(password) < 8 {
		return model.User{}, fmt.Errorf("name and password are required")
	}
	passwordHash, err := authn.HashPassword(password)
	if err != nil {
		return model.User{}, err
	}
	return s.repo.AcceptOperatorInvitation(ctx, HashRefreshToken(strings.TrimSpace(rawToken)), displayName, passwordHash, time.Now().UTC())
}

func (s *AdminService) RevokeOperatorInvitation(ctx context.Context, actorID, invitationID, reason string) error {
	if err := s.repo.RevokeOperatorInvitation(ctx, invitationID); err != nil {
		return err
	}
	return s.audit(ctx, actorID, "operator_invitation_revoked", "operator_invitation", invitationID, reason, nil)
}

func (s *AdminService) UpdateOperator(ctx context.Context, actorID, operatorID, role string, disabled bool, reason string) error {
	if actorID == operatorID || !specialistOperatorRoles[role] {
		return fmt.Errorf("operator update is not allowed")
	}
	operator, ok := s.repo.UserByID(ctx, operatorID)
	if !ok || operator.Role == "platform_admin" || !specialistOperatorRoles[operator.Role] {
		return fmt.Errorf("operator update is not allowed")
	}
	if err := s.repo.UpdateOperatorAccount(ctx, operatorID, role, disabled, time.Now().UTC()); err != nil {
		return err
	}
	if err := s.repo.RevokeRefreshTokensForUser(ctx, operatorID); err != nil {
		return err
	}
	return s.audit(ctx, actorID, "operator_updated", "operator", operatorID, reason, map[string]any{"role": role, "disabled": disabled})
}

func (s *AdminService) Overview(ctx context.Context, role string) (model.AdminOverview, error) {
	overview := model.AdminOverview{Role: role}
	switch role {
	case "content_admin":
		modules, err := s.repo.GetEducationModules(ctx)
		if err != nil {
			return overview, err
		}
		for _, item := range modules {
			if item.Status == "in_review" {
				overview.ReviewContent++
			} else if item.Status == "draft" {
				overview.DraftContent++
			}
		}
	case "support_operator":
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
		requests, err := s.repo.GetAllDataRequests(ctx)
		if err != nil {
			return overview, err
		}
		for _, item := range requests {
			if item.Status == "failed" {
				overview.FailedDataRequests++
			}
		}
	case "model_release_operator":
		groups := [][]model.Release{}
		models, err := s.repo.GetModelReleases(ctx)
		if err != nil {
			return overview, err
		}
		groups = append(groups, models)
		rules, err := s.repo.GetRulesetReleases(ctx)
		if err != nil {
			return overview, err
		}
		groups = append(groups, rules)
		network, err := s.repo.GetNetworkRulesets(ctx)
		if err != nil {
			return overview, err
		}
		groups = append(groups, network)
		for _, group := range groups {
			for _, item := range group {
				if item.Status == "validated" {
					overview.ValidatedReleases++
				}
			}
		}
		rollouts, err := s.repo.ListReleaseRollouts(ctx)
		if err != nil {
			return overview, err
		}
		for _, item := range rollouts {
			if item.Status == "active" || item.Status == "staged" {
				overview.ActiveRollouts++
			}
		}
	case "platform_admin":
		requests, err := s.repo.GetPendingEmergencyKeyRequests(ctx, time.Now().UTC())
		if err != nil {
			return overview, err
		}
		overview.PendingEmergency = len(requests)
		operators, err := s.repo.ListOperatorAccounts(ctx)
		if err != nil {
			return overview, err
		}
		for _, item := range operators {
			if item.DisabledAt == nil {
				overview.ActiveOperators++
			}
		}
		links, err := s.repo.ListSiteSocialLinks(ctx, true)
		if err != nil {
			return overview, err
		}
		overview.VisibleSocialLinks = len(links)
	default:
		return overview, fmt.Errorf("operator role is not allowed")
	}
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

func (s *AdminService) GetModelReleases(ctx context.Context) ([]model.Release, error) {
	return s.repo.GetModelReleases(ctx)
}

func (s *AdminService) CreateModelRelease(ctx context.Context, version, platform, artifactPath, sha256Val, contract string, threshold float64, metrics map[string]any) error {
	id := "rel_model_" + uuid.NewString()[:8]
	return s.repo.CreateModelRelease(ctx, id, version, platform, artifactPath, sha256Val, contract, threshold, metrics)
}

func (s *AdminService) GetRulesetReleases(ctx context.Context) ([]model.Release, error) {
	return s.repo.GetRulesetReleases(ctx)
}

func (s *AdminService) CreateRulesetRelease(ctx context.Context, version, artifactPath, sha256Val string, rules map[string]any) error {
	id := "rel_rules_" + uuid.NewString()[:8]
	return s.repo.CreateRulesetRelease(ctx, id, version, artifactPath, sha256Val, rules)
}

func (s *AdminService) GetNetworkRulesets(ctx context.Context) ([]model.Release, error) {
	return s.repo.GetNetworkRulesets(ctx)
}

func (s *AdminService) GetPortalOverview(ctx context.Context) (model.PortalOverview, error) {
	return s.repo.GetPortalOverview(ctx)
}

func (s *AdminService) CreateNetworkRulesetRelease(ctx context.Context, version, artifactPath, sha256Val string, rules map[string]any) error {
	id := "rel_net_" + uuid.NewString()[:8]
	return s.repo.CreateNetworkRulesetRelease(ctx, id, version, artifactPath, sha256Val, rules)
}

func (s *AdminService) Releases(ctx context.Context) (map[string][]model.Release, []model.ReleaseRollout, error) {
	models, err := s.repo.GetModelReleases(ctx)
	if err != nil {
		return nil, nil, err
	}
	rulesets, err := s.repo.GetRulesetReleases(ctx)
	if err != nil {
		return nil, nil, err
	}
	network, err := s.repo.GetNetworkRulesets(ctx)
	if err != nil {
		return nil, nil, err
	}
	rollouts, err := s.repo.ListReleaseRollouts(ctx)
	return map[string][]model.Release{"model": models, "ruleset": rulesets, "network": network}, rollouts, err
}

func (s *AdminService) StoreReleaseArtifact(filename string, source io.Reader) (string, string, error) {
	filename = filepath.Base(strings.TrimSpace(filename))
	lower := strings.ToLower(filename)
	allowed := []string{".onnx", ".tflite", ".json", ".bin", ".zip", ".tar.gz"}
	extension := ""
	for _, candidate := range allowed {
		if strings.HasSuffix(lower, candidate) {
			extension = candidate
			break
		}
	}
	if extension == "" {
		return "", "", fmt.Errorf("release artifact type is not allowed")
	}
	if err := os.MkdirAll(s.cfg.ArtifactStoragePath, 0o750); err != nil {
		return "", "", err
	}
	relativePath := "uploads/" + time.Now().UTC().Format("2006/01/") + uuid.NewString() + extension
	absolutePath := filepath.Join(s.cfg.ArtifactStoragePath, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		return "", "", err
	}
	file, err := os.OpenFile(absolutePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(file, hash), source)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(absolutePath)
		if copyErr != nil {
			return "", "", copyErr
		}
		return "", "", closeErr
	}
	return relativePath, hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *AdminService) CreateRollout(ctx context.Context, actorID, kind, releaseID, platform string, percentage int, appVersion, reason string) (model.ReleaseRollout, error) {
	allowedPlatforms := map[string]bool{"android": true, "windows": true, "linux": true, "macos": true, "web": true, "all": true}
	if !allowedPlatforms[platform] || percentage < 1 || percentage > 100 || len(appVersion) > 80 {
		return model.ReleaseRollout{}, fmt.Errorf("invalid rollout cohort")
	}
	releases, rollouts, err := s.Releases(ctx)
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	validated := false
	for _, item := range releases[kind] {
		if item.ID == releaseID && item.Status == "validated" {
			validated = true
			break
		}
	}
	if !validated {
		return model.ReleaseRollout{}, fmt.Errorf("release must be validated before rollout")
	}
	for _, item := range rollouts {
		if item.Kind == kind && item.Platform == platform && (item.Status == "staged" || item.Status == "active" || item.Status == "paused") {
			return model.ReleaseRollout{}, fmt.Errorf("another rollout is already active for this kind and platform")
		}
	}
	item, err := s.repo.CreateReleaseRollout(ctx, model.ReleaseRollout{ID: "rollout_" + uuid.NewString()[:12], Kind: kind,
		ReleaseID: releaseID, Platform: platform, Percentage: percentage, AppVersionConstraint: strings.TrimSpace(appVersion), CreatedBy: actorID})
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	_ = s.audit(ctx, actorID, "release_rollout_staged", "release_rollout", item.ID, reason, map[string]any{"kind": kind, "percentage": percentage, "platform": platform})
	return item, nil
}

func (s *AdminService) TransitionRollout(ctx context.Context, actorID, rolloutID, action, reason string) (model.ReleaseRollout, error) {
	rollouts, err := s.repo.ListReleaseRollouts(ctx)
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	var current model.ReleaseRollout
	for _, item := range rollouts {
		if item.ID == rolloutID {
			current = item
			break
		}
	}
	if current.ID == "" {
		return model.ReleaseRollout{}, fmt.Errorf("rollout not found")
	}
	allowed := map[string]map[string]bool{
		"staged": {"active": true, "rolled_back": true},
		"active": {"paused": true, "completed": true, "rolled_back": true},
		"paused": {"active": true, "rolled_back": true},
	}
	if !allowed[current.Status][action] {
		return model.ReleaseRollout{}, fmt.Errorf("invalid rollout transition")
	}
	if strings.TrimSpace(reason) == "" && (action == "rolled_back" || action == "paused") {
		return model.ReleaseRollout{}, fmt.Errorf("transition reason is required")
	}
	item, err := s.repo.TransitionReleaseRollout(ctx, rolloutID, action, time.Now().UTC())
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	_ = s.audit(ctx, actorID, "release_rollout_"+action, "release_rollout", item.ID, reason, map[string]any{"kind": item.Kind, "release_id": item.ReleaseID})
	return item, nil
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
	return request, key, nil
}

func (s *AdminService) ValidateEmergencyKey(ctx context.Context, key, deviceID string) (model.EmergencyGrant, error) {
	now := time.Now().UTC()
	request, err := s.repo.UseEmergencyKey(ctx, HashRefreshToken(key), deviceID, now)
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
		RequestID: request.ID, DeviceID: deviceID,
		GrantStartsAt: now, GrantExpiresAt: now.Add(10 * time.Minute),
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
