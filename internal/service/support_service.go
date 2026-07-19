package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	appcrypto "github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type SupportService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

var (
	ErrDataRequestInvalid   = errors.New("invalid data request")
	ErrDataRequestConflict  = errors.New("active data request already exists")
	ErrDataRequestForbidden = errors.New("data request is not allowed for this account")
)

func NewSupportService(repo *repository.Repository, logger *zap.Logger) *SupportService {
	return &SupportService{repo: repo, logger: logger}
}

func NewSupportServiceWithConfig(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *SupportService {
	return &SupportService{repo: repo, cfg: cfg, logger: logger}
}

func (s *SupportService) ensureRequesterRole(ctx context.Context, userID string) error {
	actor, ok := s.repo.UserByID(ctx, userID)
	if !ok || (actor.Role != model.RoleUser && actor.Role != model.RolePartner) {
		return fmt.Errorf("support requester role is not allowed")
	}
	return nil
}

func (s *SupportService) CreateThreadedSupportCase(ctx context.Context, userID, title, detail, cType, priority, impact string) (model.SupportCase, error) {
	if err := s.ensureRequesterRole(ctx, userID); err != nil {
		return model.SupportCase{}, err
	}
	title = strings.TrimSpace(title)
	detail = strings.TrimSpace(detail)
	allowedTypes := map[string]bool{
		"technical_support": true, "account_recovery": true, "partner_abuse": true,
		"stuck_approval": true, "device_recovery": true, "notification_failure": true,
		"organization_dispute": true, "accountability_guidance": true, "privacy_request": true,
		"safety_support": true,
	}
	allowedPriorities := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
	allowedImpacts := map[string]bool{"question": true, "degraded": true, "blocked": true, "safety": true}
	if title == "" || detail == "" || len(title) > 120 || len(detail) > 4000 || !allowedTypes[cType] || !allowedPriorities[priority] || !allowedImpacts[impact] {
		return model.SupportCase{}, fmt.Errorf("invalid support case input")
	}
	if s.cfg.JournalEncryptionKey == "" {
		return model.SupportCase{}, fmt.Errorf("support message encryption is required")
	}
	encrypted, err := appcrypto.Encrypt(detail, s.cfg.JournalEncryptionKey)
	if err != nil {
		return model.SupportCase{}, fmt.Errorf("support message encryption failed")
	}
	now := time.Now().UTC()
	item := model.SupportCase{
		ID: "CASE-" + uuid.NewString()[:8], UserID: userID, Title: title,
		Type: cType, Status: "waiting_support", Priority: priority, Impact: impact,
		CreatedAt: now, UpdatedAt: now,
	}
	return s.repo.CreateSupportCaseWithMessage(ctx, item, encrypted)
}

func (s *SupportService) GetSupportCaseDetail(ctx context.Context, actorID, actorMode, caseID string) (model.SupportCase, error) {
	if actorMode != "admin" {
		if err := s.ensureRequesterRole(ctx, actorID); err != nil {
			return model.SupportCase{}, err
		}
	}
	item, err := s.repo.GetSupportCaseDetail(ctx, caseID)
	if err != nil {
		return model.SupportCase{}, err
	}
	isOperator := actorMode == "admin"
	if isOperator && item.UserID == actorID {
		return model.SupportCase{}, fmt.Errorf("administrators cannot handle their own support cases")
	}
	if item.UserID != actorID && !isOperator {
		return model.SupportCase{}, fmt.Errorf("support case does not belong to actor")
	}
	if isOperator && item.Owner != actorID {
		return model.SupportCase{}, fmt.Errorf("support case must be claimed before opening the thread")
	}
	if err := s.decryptSupportMessages(item.Messages); err != nil {
		return model.SupportCase{}, err
	}
	return item, nil
}

func (s *SupportService) Reply(ctx context.Context, actorID, actorMode, caseID, content string) (model.SupportMessage, error) {
	if actorMode != "admin" {
		if err := s.ensureRequesterRole(ctx, actorID); err != nil {
			return model.SupportMessage{}, err
		}
	}
	item, err := s.repo.GetSupportCaseDetail(ctx, caseID)
	if err != nil {
		return model.SupportMessage{}, err
	}
	isOperator := actorMode == "admin"
	if isOperator && item.UserID == actorID {
		return model.SupportMessage{}, fmt.Errorf("administrators cannot handle their own support cases")
	}
	if item.UserID != actorID && !isOperator {
		return model.SupportMessage{}, fmt.Errorf("support case does not belong to actor")
	}
	if isOperator && item.Owner != actorID {
		return model.SupportMessage{}, fmt.Errorf("support case must be claimed before replying")
	}
	if item.Status == "closed" {
		return model.SupportMessage{}, fmt.Errorf("closed support cases cannot receive replies")
	}
	content = strings.TrimSpace(content)
	if content == "" || len(content) > 4000 {
		return model.SupportMessage{}, fmt.Errorf("support reply is invalid")
	}
	if s.cfg.JournalEncryptionKey == "" {
		return model.SupportMessage{}, fmt.Errorf("support message encryption is required")
	}
	encrypted, err := appcrypto.Encrypt(content, s.cfg.JournalEncryptionKey)
	if err != nil {
		return model.SupportMessage{}, fmt.Errorf("support message encryption failed")
	}
	role := "requester"
	nextStatus := "waiting_support"
	if isOperator {
		role = "admin"
		nextStatus = "waiting_user"
	}
	message, err := s.repo.AddSupportMessage(ctx, model.SupportMessage{
		ID: "msg_" + uuid.NewString()[:12], SupportCaseID: caseID, AuthorID: actorID,
		AuthorRole: role, Content: encrypted, CreatedAt: time.Now().UTC(),
	}, nextStatus)
	if err != nil {
		return model.SupportMessage{}, err
	}
	message.Content = content
	return message, nil
}

func (s *SupportService) Transition(ctx context.Context, actorID, actorMode, caseID, status string) error {
	if actorMode != "admin" {
		if err := s.ensureRequesterRole(ctx, actorID); err != nil {
			return err
		}
	}
	item, err := s.repo.GetSupportCaseDetail(ctx, caseID)
	if err != nil {
		return err
	}
	isOperator := actorMode == "admin"
	if isOperator && item.UserID == actorID {
		return fmt.Errorf("administrators cannot handle their own support cases")
	}
	if item.UserID != actorID && !isOperator {
		return fmt.Errorf("support case does not belong to actor")
	}
	if isOperator && item.Owner != actorID {
		return fmt.Errorf("support case must be claimed before changing status")
	}
	if isOperator {
		if status != "waiting_support" && status != "waiting_user" && status != "resolved" && status != "closed" {
			return fmt.Errorf("invalid support status")
		}
		return s.repo.TransitionSupportCase(ctx, caseID, status, actorID, time.Now().UTC())
	}
	if status == "closed" && item.Status == "resolved" {
		return s.repo.TransitionSupportCase(ctx, caseID, "closed", "", time.Now().UTC())
	}
	if status == "waiting_support" && item.Status == "resolved" && item.ResolvedAt != nil && time.Since(*item.ResolvedAt) <= 7*24*time.Hour {
		return s.repo.TransitionSupportCase(ctx, caseID, "waiting_support", "", time.Now().UTC())
	}
	return fmt.Errorf("requester cannot perform this support transition")
}

func (s *SupportService) decryptSupportMessages(items []model.SupportMessage) error {
	if len(items) == 0 {
		return nil
	}
	if s.cfg.JournalEncryptionKey == "" {
		return fmt.Errorf("support message encryption key is unavailable")
	}
	for i := range items {
		plain, err := appcrypto.Decrypt(items[i].Content, s.cfg.JournalEncryptionKey)
		if err != nil {
			s.logger.Error("failed to decrypt support message", zap.String("message_id", items[i].ID))
			return fmt.Errorf("support message decryption failed")
		}
		items[i].Content = plain
	}
	return nil
}

func (s *SupportService) GetSupportCases(ctx context.Context) ([]model.SupportCase, error) {
	return s.repo.GetSupportCases(ctx)
}

func (s *SupportService) GetSupportCasesForAdmin(ctx context.Context, adminID string) ([]model.SupportCase, error) {
	items, err := s.repo.GetSupportCases(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]model.SupportCase, 0, len(items))
	for _, item := range items {
		if item.UserID != adminID {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *SupportService) Claim(ctx context.Context, operatorID, caseID, reason string) (model.SupportCase, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 240 {
		return model.SupportCase{}, fmt.Errorf("claim reason is required")
	}
	item, err := s.repo.GetSupportCaseDetail(ctx, caseID)
	if err != nil {
		return model.SupportCase{}, err
	}
	if item.UserID == operatorID {
		return model.SupportCase{}, fmt.Errorf("administrators cannot claim their own support cases")
	}
	return s.repo.ClaimSupportCase(ctx, caseID, operatorID, reason, time.Now().UTC())
}

func (s *SupportService) ReleaseClaim(ctx context.Context, operatorID, caseID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 240 {
		return fmt.Errorf("release reason is required")
	}
	return s.repo.ReleaseSupportCase(ctx, caseID, operatorID, reason, time.Now().UTC())
}

func (s *SupportService) GetSupportCasesForUser(ctx context.Context, userID string) ([]model.SupportCase, error) {
	if err := s.ensureRequesterRole(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.GetSupportCasesForUser(ctx, userID)
}

func (s *SupportService) CreateSupportCase(ctx context.Context, userID, title, cType, priority string) error {
	if err := s.ensureRequesterRole(ctx, userID); err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	allowedTypes := map[string]bool{
		"technical_support": true, "account_recovery": true, "partner_abuse": true,
		"stuck_approval": true, "device_recovery": true, "notification_failure": true,
		"organization_dispute": true, "accountability_guidance": true, "privacy_request": true,
	}
	allowedPriorities := map[string]bool{"low": true, "normal": true, "high": true, "urgent": true}
	if title == "" || !allowedTypes[cType] || !allowedPriorities[priority] {
		return fmt.Errorf("invalid support case input")
	}
	id := "CASE-" + uuid.NewString()[:8]
	return s.repo.CreateSupportCase(ctx, id, userID, title, cType, priority)
}

func (s *SupportService) GetDataRequests(ctx context.Context, userID string) ([]model.DataRequest, error) {
	s.purgeExpiredDataExports(ctx)
	return s.repo.GetDataRequests(ctx, userID)
}

func (s *SupportService) GetAllDataRequests(ctx context.Context) ([]model.DataRequest, error) {
	s.purgeExpiredDataExports(ctx)
	return s.repo.GetAllDataRequests(ctx)
}

func (s *SupportService) purgeExpiredDataExports(ctx context.Context) {
	items, err := s.repo.GetAllDataRequests(ctx)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, item := range items {
		if item.Type != "export" || item.Status != "completed" {
			continue
		}
		expected := filepath.Join(s.cfg.ExportStoragePath, filepath.Base(item.ID)+".zip.enc")
		if item.ResultExpiresAt != nil && !now.Before(*item.ResultExpiresAt) {
			if item.ResultPath != "" && filepath.Clean(item.ResultPath) == filepath.Clean(expected) {
				_ = os.Remove(expected)
			}
			item.ResultPath, item.ResultExpiresAt, item.FailureCode, item.UpdatedAt = "", nil, "result_expired", now
			_ = s.repo.UpdateDataRequest(ctx, item)
			continue
		}
		validPath := item.ResultPath != "" && filepath.Clean(item.ResultPath) == filepath.Clean(expected)
		fileInfo, statErr := os.Stat(expected)
		validFile := statErr == nil && fileInfo.Mode().IsRegular()
		if item.ResultExpiresAt == nil || !validPath || !validFile {
			item.ResultPath, item.ResultExpiresAt, item.FailureCode, item.UpdatedAt = "", nil, "result_unavailable", now
			_ = s.repo.UpdateDataRequest(ctx, item)
		}
	}
}

func (s *SupportService) CreateDataRequest(ctx context.Context, userID, requestType string) error {
	_, _, err := s.CreateDataRequestWithResult(ctx, userID, requestType)
	return err
}

func (s *SupportService) CreateDataRequestWithResult(ctx context.Context, userID, requestType string) (model.DataRequest, string, error) {
	if requestType != "export" && requestType != "delete" && requestType != "retention_review" {
		return model.DataRequest{}, "", fmt.Errorf("%w: unsupported type", ErrDataRequestInvalid)
	}
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok {
		return model.DataRequest{}, "", fmt.Errorf("user not found")
	}
	if requestType == "delete" && user.Role != "user" && user.Role != "partner" {
		return model.DataRequest{}, "", fmt.Errorf("%w: operator account deletion requires an out-of-band administrator workflow", ErrDataRequestForbidden)
	}
	existingRequests, err := s.repo.GetDataRequests(ctx, userID)
	if err != nil {
		return model.DataRequest{}, "", err
	}
	for _, existing := range existingRequests {
		if existing.Type == requestType && existing.Status != "completed" && existing.Status != "failed" && existing.Status != "rejected" && existing.Status != "cancelled" {
			return model.DataRequest{}, "", ErrDataRequestConflict
		}
	}
	now := time.Now().UTC()
	item := model.DataRequest{ID: "DR-" + uuid.NewString()[:8], UserID: userID,
		Title: humanDataRequestTitleForService(requestType), Type: requestType, Status: "queued", CreatedAt: now, UpdatedAt: now}
	previewURL := ""
	if requestType == "delete" {
		rawToken := "delete_" + uuid.NewString() + uuid.NewString()
		expiresAt := now.Add(30 * time.Minute)
		item.Status, item.ConfirmationTokenHash, item.ConfirmationExpiresAt = "pending_confirmation", HashRefreshToken(rawToken), &expiresAt
		previewURL = s.cfg.PublicWebBaseURL + "/data-requests/confirm-delete?token=" + rawToken
	}
	if err := s.repo.CreateDataRequestRecord(ctx, item); err != nil {
		return model.DataRequest{}, "", err
	}
	if requestType == "delete" {
		if err := NewEmailService(s.cfg).SendDataRequestConfirmation(ctx, user.Email, previewURL); err != nil {
			return model.DataRequest{}, "", err
		}
		if s.cfg.NotificationMode != "demo" {
			previewURL = ""
		}
		return item, previewURL, nil
	}
	if requestType == "export" {
		processed, err := s.ProcessDataExport(ctx, item.ID)
		return processed, "", err
	}
	return item, "", nil
}

func humanDataRequestTitleForService(kind string) string {
	if kind == "export" {
		return "Export account data"
	}
	if kind == "delete" {
		return "Delete account and data"
	}
	return "Review data retention"
}

func (s *SupportService) ProcessDataExport(ctx context.Context, requestID string) (model.DataRequest, error) {
	item, err := s.repo.DataRequestByID(ctx, requestID)
	if err != nil || item.Type != "export" || (item.Status != "queued" && item.Status != "failed") {
		return model.DataRequest{}, fmt.Errorf("data export cannot be processed")
	}
	item.Status, item.FailureCode, item.UpdatedAt = "processing", "", time.Now().UTC()
	if err := s.repo.UpdateDataRequest(ctx, item); err != nil {
		return model.DataRequest{}, err
	}
	fail := func(code string, cause error) (model.DataRequest, error) {
		item.Status, item.FailureCode, item.RetryCount, item.UpdatedAt = "failed", code, item.RetryCount+1, time.Now().UTC()
		_ = s.repo.UpdateDataRequest(ctx, item)
		return item, cause
	}
	payload, err := s.repo.BuildUserExportSnapshot(ctx, item.UserID)
	if err != nil {
		return fail("snapshot_failed", err)
	}
	if err = s.decryptExportPayload(payload); err != nil {
		return fail("decryption_failed", err)
	}
	payload["manifest"] = map[string]any{"format": "gamblock-ai-account-export-v1", "generated_at": time.Now().UTC(), "browsing_history_included": false}
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fail("encoding_failed", err)
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entry, err := archive.Create("gamblock-ai-account-export.json")
	if err != nil {
		return fail("archive_failed", err)
	}
	if _, err = entry.Write(jsonData); err != nil {
		return fail("archive_failed", err)
	}
	if err = archive.Close(); err != nil {
		return fail("archive_failed", err)
	}
	if s.cfg.JournalEncryptionKey == "" {
		return fail("encryption_unavailable", fmt.Errorf("export encryption is unavailable"))
	}
	encrypted, err := appcrypto.Encrypt(buffer.String(), s.cfg.JournalEncryptionKey)
	if err != nil {
		return fail("encryption_failed", err)
	}
	if err = os.MkdirAll(s.cfg.ExportStoragePath, 0o750); err != nil {
		return fail("storage_failed", err)
	}
	path := filepath.Join(s.cfg.ExportStoragePath, filepath.Base(item.ID)+".zip.enc")
	if err = os.WriteFile(path, []byte(encrypted), 0o600); err != nil {
		return fail("storage_failed", err)
	}
	now, expires := time.Now().UTC(), time.Now().UTC().Add(7*24*time.Hour)
	item.Status, item.ResultPath, item.ResultExpiresAt, item.CompletedAt, item.UpdatedAt = "completed", path, &expires, &now, now
	if err = s.repo.UpdateDataRequest(ctx, item); err != nil {
		return fail("persistence_failed", err)
	}
	if user, ok := s.repo.UserByID(ctx, item.UserID); ok {
		_ = NewEmailService(s.cfg).SendDataExportReady(ctx, user.Email, s.cfg.PublicWebBaseURL+"/data-requests")
	}
	return item, nil
}

func (s *SupportService) decryptExportPayload(payload map[string]any) error {
	if s.cfg.JournalEncryptionKey == "" {
		return fmt.Errorf("export encryption key is unavailable")
	}
	if records, ok := payload["recovery_records"].([]model.RecoveryRecord); ok {
		for index := range records {
			if records[index].Content != "" {
				plain, err := appcrypto.Decrypt(records[index].Content, s.cfg.JournalEncryptionKey)
				if err != nil {
					return err
				}
				records[index].Content = plain
			}
		}
		payload["recovery_records"] = records
	}
	if reflections, ok := payload["reflections"].([]model.JournalEntry); ok {
		for index := range reflections {
			if reflections[index].Text != "" {
				plain, err := appcrypto.Decrypt(reflections[index].Text, s.cfg.JournalEncryptionKey)
				if err != nil {
					return err
				}
				reflections[index].Text = plain
			}
		}
		payload["reflections"] = reflections
	}
	if messages, ok := payload["support_messages"].([]model.SupportMessage); ok {
		for index := range messages {
			if messages[index].Content != "" {
				plain, err := appcrypto.Decrypt(messages[index].Content, s.cfg.JournalEncryptionKey)
				if err != nil {
					return err
				}
				messages[index].Content = plain
			}
		}
		payload["support_messages"] = messages
	}
	return nil
}

func (s *SupportService) DataExportFile(ctx context.Context, userID, requestID string) ([]byte, error) {
	s.purgeExpiredDataExports(ctx)
	item, err := s.repo.DataRequestByID(ctx, requestID)
	if err != nil || item.UserID != userID || item.Type != "export" || item.Status != "completed" || item.ResultExpiresAt == nil || !time.Now().UTC().Before(*item.ResultExpiresAt) {
		return nil, fmt.Errorf("data export is unavailable")
	}
	expected := filepath.Join(s.cfg.ExportStoragePath, filepath.Base(item.ID)+".zip.enc")
	if filepath.Clean(item.ResultPath) != filepath.Clean(expected) {
		return nil, fmt.Errorf("data export path is invalid")
	}
	encrypted, err := os.ReadFile(expected)
	if err != nil {
		return nil, fmt.Errorf("data export is unavailable")
	}
	plain, err := appcrypto.Decrypt(string(encrypted), s.cfg.JournalEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("data export decryption failed")
	}
	return []byte(plain), nil
}

func (s *SupportService) ConfirmAccountDeletion(ctx context.Context, userID, rawToken string) error {
	now := time.Now().UTC()
	item, err := s.repo.DataRequestByConfirmationToken(ctx, HashRefreshToken(strings.TrimSpace(rawToken)), now)
	if err != nil || item.UserID != userID || item.Type != "delete" || item.Status != "pending_confirmation" {
		return fmt.Errorf("deletion confirmation is invalid or expired")
	}
	item.Status, item.ConfirmationTokenHash, item.ConfirmationExpiresAt, item.ConfirmedAt, item.UpdatedAt = "processing", "", nil, &now, now
	if err := s.repo.UpdateDataRequest(ctx, item); err != nil {
		return err
	}
	if key, ok := s.repo.UserAvatarStorageKey(ctx, userID); ok && key != "" {
		_ = os.Remove(filepath.Join(s.cfg.AvatarStoragePath, filepath.Base(key)))
	}
	return s.repo.DeleteUserAccountData(ctx, userID, now)
}

func (s *SupportService) RetryDataRequest(ctx context.Context, requestID string) (model.DataRequest, error) {
	item, err := s.repo.DataRequestByID(ctx, requestID)
	if err != nil || item.Type != "export" || item.Status != "failed" || item.RetryCount >= 3 {
		return model.DataRequest{}, fmt.Errorf("data request cannot be retried")
	}
	item.Status, item.FailureCode, item.UpdatedAt = "queued", "", time.Now().UTC()
	if err := s.repo.UpdateDataRequest(ctx, item); err != nil {
		return model.DataRequest{}, err
	}
	return s.ProcessDataExport(ctx, requestID)
}

func (s *SupportService) RejectDataRequest(ctx context.Context, requestID, reason string) (model.DataRequest, error) {
	item, err := s.repo.DataRequestByID(ctx, requestID)
	if err != nil || (item.Status != "failed" && item.Status != "queued") || strings.TrimSpace(reason) == "" {
		return model.DataRequest{}, fmt.Errorf("data request cannot be rejected")
	}
	item.Status, item.FailureCode, item.UpdatedAt = "rejected", "operator_rejected", time.Now().UTC()
	if err := s.repo.UpdateDataRequest(ctx, item); err != nil {
		return model.DataRequest{}, err
	}
	return item, nil
}
