package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	appcrypto "github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

var (
	ErrLearningHubStateInvalid      = errors.New("learning hub state is invalid")
	ErrLearningHubCheckpointInvalid = errors.New("learning hub checkpoint is invalid")
	ErrLearningHubAdminInvalid      = errors.New("learning hub admin payload is invalid")
	ErrLearningHubAdminConflict     = errors.New("learning hub admin draft conflict")
	ErrLearningHubAdminNotFound     = errors.New("learning hub admin resource not found")
	ErrLearningHubTransitionInvalid = errors.New("learning hub editorial transition is invalid")
	ErrLearningHubTaxonomyConflict  = errors.New("learning hub taxonomy is in use")
)

var learningSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var learningKinds = map[string]bool{
	"course": true, "certification": true, "learning_path": true,
	"mini_project": true, "career_snapshot": true, "toolkit": true,
	"opportunity": true,
}

var learningItemStatuses = map[string]bool{
	"draft": true, "in_review": true, "published": true, "archived": true,
}

type LearningHubService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

func NewLearningHubService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *LearningHubService {
	return &LearningHubService{repo: repo, cfg: cfg, logger: logger}
}

func (s *LearningHubService) Catalog(ctx context.Context, userID, locale string) (model.LearningCatalog, error) {
	if locale != "en" {
		locale = "id"
	}
	return s.repo.GetLearningCatalog(ctx, userID, locale)
}

func (s *LearningHubService) Item(ctx context.Context, userID, slug, locale string) (model.LearningItem, error) {
	if locale != "en" {
		locale = "id"
	}
	return s.repo.GetLearningItemBySlug(ctx, userID, slug, locale)
}

func (s *LearningHubService) Progress(ctx context.Context, userID string) ([]model.LearningProgress, error) {
	return s.repo.GetLearningProgress(ctx, userID)
}

func (s *LearningHubService) SaveState(ctx context.Context, userID, itemID, state string) (model.LearningProgress, error) {
	state = strings.TrimSpace(state)
	if state != "saved" && state != "started" {
		return model.LearningProgress{}, ErrLearningHubStateInvalid
	}
	return s.repo.UpsertLearningState(ctx, userID, itemID, state)
}

func (s *LearningHubService) Checkpoint(ctx context.Context, userID, itemID string, input model.LearningCheckpointInput) (model.LearningCheckpointResult, error) {
	reflection, outcome := strings.TrimSpace(input.Reflection), strings.TrimSpace(input.Outcome)
	if reflection == "" && outcome == "" {
		return model.LearningCheckpointResult{}, ErrLearningHubCheckpointInvalid
	}
	if len([]rune(reflection)) > 2000 || len([]rune(outcome)) > 2000 {
		return model.LearningCheckpointResult{}, ErrLearningHubCheckpointInvalid
	}
	if s.cfg.JournalEncryptionKey == "" {
		return model.LearningCheckpointResult{}, fmt.Errorf("learning checkpoint encryption is not configured")
	}
	reflectionEncrypted, err := appcrypto.Encrypt(reflection, s.cfg.JournalEncryptionKey)
	if err != nil {
		return model.LearningCheckpointResult{}, fmt.Errorf("failed to encrypt learning reflection")
	}
	outcomeEncrypted, err := appcrypto.Encrypt(outcome, s.cfg.JournalEncryptionKey)
	if err != nil {
		return model.LearningCheckpointResult{}, fmt.Errorf("failed to encrypt learning outcome")
	}
	progress, granted, capReached, err := s.repo.CompleteLearningProgress(ctx, userID, itemID, reflectionEncrypted, outcomeEncrypted)
	if err != nil {
		if errors.Is(err, repository.ErrLearningItemNotFound) {
			return model.LearningCheckpointResult{}, err
		}
		return model.LearningCheckpointResult{}, err
	}
	experience, err := s.repo.GetLearningExperience(ctx, userID)
	if err != nil {
		return model.LearningCheckpointResult{}, err
	}
	return model.LearningCheckpointResult{Progress: progress, EXPGranted: granted, CapReached: capReached, Experience: experience}, nil
}

func validateLearningDraft(draft model.LearningItemDraft, requireReview bool) error {
	draft.Slug = strings.TrimSpace(draft.Slug)
	if draft.Slug == "" || !learningSlugPattern.MatchString(draft.Slug) || len(draft.Slug) > 120 {
		return ErrLearningHubAdminInvalid
	}
	if !learningKinds[draft.Kind] || strings.TrimSpace(draft.TitleID) == "" || strings.TrimSpace(draft.TitleEN) == "" || strings.TrimSpace(draft.SummaryID) == "" || strings.TrimSpace(draft.SummaryEN) == "" {
		return ErrLearningHubAdminInvalid
	}
	if draft.Document == nil {
		return ErrLearningHubAdminInvalid
	}
	if rawURL, ok := draft.Document["url"].(string); ok && strings.TrimSpace(rawURL) != "" {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return ErrLearningHubAdminInvalid
		}
	}
	if rawDuration, ok := draft.Document["duration_minutes"]; ok {
		duration := documentInt(rawDuration)
		if duration < 1 || duration > 7200 {
			return ErrLearningHubAdminInvalid
		}
	}
	for _, key := range []string{"provider_description_id", "provider_description_en"} {
		if len([]rune(documentString(draft.Document, key))) > 200 {
			return ErrLearningHubAdminInvalid
		}
	}
	if requireReview {
		if strings.TrimSpace(documentString(draft.Document, "provider")) == "" || strings.TrimSpace(documentString(draft.Document, "url")) == "" {
			return ErrLearningHubAdminInvalid
		}
		if strings.TrimSpace(documentString(draft.Document, "reviewer_name")) == "" || strings.TrimSpace(documentString(draft.Document, "reviewed_at")) == "" {
			return ErrLearningHubAdminInvalid
		}
		if _, err := time.Parse("2006-01-02", documentString(draft.Document, "reviewed_at")); err != nil {
			return ErrLearningHubAdminInvalid
		}
		if len(documentStrings(draft.Document["outcomes_id"])) == 0 && len(documentStrings(draft.Document["outcomes_en"])) == 0 && len(documentStrings(draft.Document["outcomes"])) == 0 {
			return ErrLearningHubAdminInvalid
		}
		if len(documentStrings(draft.Document["clusters"])) == 0 || len(documentStrings(draft.Document["programs"])) == 0 {
			return ErrLearningHubAdminInvalid
		}
	}
	return nil
}

func documentString(document map[string]any, key string) string {
	value, _ := document[key].(string)
	return value
}

func documentInt(value any) int {
	switch number := value.(type) {
	case int:
		return number
	case int64:
		return int(number)
	case float64:
		return int(number)
	case string:
		parsed, _ := strconv.Atoi(number)
		return parsed
	default:
		return 0
	}
}

func documentStrings(value any) []string {
	result := make([]string, 0)
	switch values := value.(type) {
	case []string:
		return append(result, values...)
	case []any:
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, text)
			}
		}
	case string:
		if strings.TrimSpace(values) != "" {
			result = append(result, values)
		}
	}
	return result
}

func (s *LearningHubService) AdminItems(ctx context.Context, status string) ([]model.AdminLearningItem, error) {
	status = strings.TrimSpace(status)
	if status != "" && !learningItemStatuses[status] {
		return nil, ErrLearningHubAdminInvalid
	}
	return s.repo.ListAdminLearningItems(ctx, status)
}

func (s *LearningHubService) AdminItemsPaginated(ctx context.Context, query model.PaginationQuery) (model.PaginatedList[model.AdminLearningItem], error) {
	status := strings.TrimSpace(query.Status)
	if status != "" && !learningItemStatuses[status] {
		return model.PaginatedList[model.AdminLearningItem]{}, ErrLearningHubAdminInvalid
	}
	return s.repo.ListAdminLearningItemsPaginated(ctx, query)
}

func (s *LearningHubService) AdminItem(ctx context.Context, id string) (model.AdminLearningItem, error) {
	return s.repo.GetAdminLearningItem(ctx, id)
}

func (s *LearningHubService) CreateAdminItem(ctx context.Context, actor string, draft model.LearningItemDraft) (model.AdminLearningItem, error) {
	draft.Slug, draft.Kind = strings.ToLower(strings.TrimSpace(draft.Slug)), strings.TrimSpace(draft.Kind)
	if err := validateLearningDraft(draft, false); err != nil {
		return model.AdminLearningItem{}, err
	}
	created, err := s.repo.CreateAdminLearningItem(ctx, actor, draft)
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_item_created", created.ID, map[string]any{"revision": created.DraftRevision})
	}
	return created, err
}

func (s *LearningHubService) UpdateAdminItem(ctx context.Context, actor, id string, expectedRevision int, draft model.LearningItemDraft) (model.AdminLearningItem, error) {
	draft.Slug, draft.Kind = strings.ToLower(strings.TrimSpace(draft.Slug)), strings.TrimSpace(draft.Kind)
	if expectedRevision < 1 {
		return model.AdminLearningItem{}, ErrLearningHubAdminInvalid
	}
	if err := validateLearningDraft(draft, false); err != nil {
		return model.AdminLearningItem{}, err
	}
	updated, err := s.repo.UpdateAdminLearningItem(ctx, actor, id, expectedRevision, draft)
	if errors.Is(err, repository.ErrLearningAdminConflict) {
		return model.AdminLearningItem{}, ErrLearningHubAdminConflict
	}
	if errors.Is(err, repository.ErrLearningAdminNotFound) {
		return model.AdminLearningItem{}, ErrLearningHubAdminNotFound
	}
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_item_updated", id, map[string]any{"revision": updated.DraftRevision})
	}
	return updated, err
}

func (s *LearningHubService) SubmitAdminItemReview(ctx context.Context, actor, id string) (model.AdminLearningItem, error) {
	item, err := s.repo.GetAdminLearningItem(ctx, id)
	if err != nil {
		return model.AdminLearningItem{}, ErrLearningHubAdminNotFound
	}
	if item.Status != "draft" {
		return model.AdminLearningItem{}, ErrLearningHubTransitionInvalid
	}
	draft := model.LearningItemDraft{Slug: item.Slug, Kind: item.Kind, TitleID: item.TitleID, TitleEN: item.TitleEN, SummaryID: item.SummaryID, SummaryEN: item.SummaryEN, Document: item.DraftDocument}
	if err := validateLearningDraft(draft, true); err != nil {
		return model.AdminLearningItem{}, err
	}
	updated, err := s.repo.SetAdminLearningStatus(ctx, actor, id, "in_review")
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_item_submitted", id, map[string]any{"revision": updated.DraftRevision})
	}
	return updated, err
}

func (s *LearningHubService) PublishAdminItem(ctx context.Context, actor, id string) (model.AdminLearningItem, error) {
	item, err := s.repo.GetAdminLearningItem(ctx, id)
	if err != nil {
		return model.AdminLearningItem{}, ErrLearningHubAdminNotFound
	}
	if item.Status == "archived" {
		return model.AdminLearningItem{}, ErrLearningHubTransitionInvalid
	}
	draft := model.LearningItemDraft{Slug: item.Slug, Kind: item.Kind, TitleID: item.TitleID, TitleEN: item.TitleEN, SummaryID: item.SummaryID, SummaryEN: item.SummaryEN, Document: item.DraftDocument}
	if err := validateLearningDraft(draft, true); err != nil {
		return model.AdminLearningItem{}, err
	}
	var mediaIDs []string
	if logo := strings.TrimSpace(documentString(draft.Document, "provider_logo_media_id")); logo != "" {
		mediaIDs = append(mediaIDs, logo)
	}
	if thumb := strings.TrimSpace(documentString(draft.Document, "thumbnail_media_id")); thumb != "" {
		mediaIDs = append(mediaIDs, thumb)
	}
	if len(mediaIDs) > 0 {
		_ = s.repo.PublishEducationMedia(ctx, mediaIDs)
	}
	updated, err := s.repo.SetAdminLearningStatus(ctx, actor, id, "published")
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_item_published", id, map[string]any{"revision": updated.PublishedRevision})
	}
	return updated, err
}

func (s *LearningHubService) ArchiveAdminItem(ctx context.Context, actor, id string) (model.AdminLearningItem, error) {
	item, err := s.repo.GetAdminLearningItem(ctx, id)
	if err != nil {
		return model.AdminLearningItem{}, ErrLearningHubAdminNotFound
	}
	if item.Status != "published" && item.Status != "in_review" {
		return model.AdminLearningItem{}, ErrLearningHubTransitionInvalid
	}
	updated, err := s.repo.SetAdminLearningStatus(ctx, actor, id, "archived")
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_item_archived", id, nil)
	}
	return updated, err
}

func (s *LearningHubService) AdminRevisions(ctx context.Context, id string) ([]model.LearningRevision, error) {
	return s.repo.ListLearningRevisions(ctx, id)
}

func (s *LearningHubService) RollbackAdminItem(ctx context.Context, actor, id, revisionID, reason string) (model.AdminLearningItem, error) {
	if strings.TrimSpace(reason) == "" || len([]rune(reason)) > 500 {
		return model.AdminLearningItem{}, ErrLearningHubAdminInvalid
	}
	updated, err := s.repo.RollbackLearningItem(ctx, actor, id, revisionID)
	if errors.Is(err, repository.ErrLearningAdminNotFound) {
		return model.AdminLearningItem{}, ErrLearningHubAdminNotFound
	}
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_item_rolled_back", id, map[string]any{"revision_id": revisionID, "reason": strings.TrimSpace(reason), "revision": updated.DraftRevision})
	}
	return updated, err
}

func (s *LearningHubService) Taxonomy(ctx context.Context) (model.LearningHubTaxonomy, error) {
	return s.repo.GetLearningHubTaxonomy(ctx)
}

func validateTaxonomySlug(slug string) error {
	if slug == "" || len(slug) > 100 || !learningSlugPattern.MatchString(slug) {
		return ErrLearningHubAdminInvalid
	}
	return nil
}

func (s *LearningHubService) CreateCluster(ctx context.Context, actor string, input model.LearningClusterInput) (model.AdminLearningCluster, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	if err := validateTaxonomySlug(input.Slug); err != nil || strings.TrimSpace(input.TitleID) == "" || strings.TrimSpace(input.TitleEN) == "" {
		return model.AdminLearningCluster{}, ErrLearningHubAdminInvalid
	}
	cluster, err := s.repo.CreateLearningCluster(ctx, input)
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_cluster_created", cluster.ID, map[string]any{"slug": cluster.Slug})
	}
	return cluster, err
}

func (s *LearningHubService) UpdateCluster(ctx context.Context, actor, id string, input model.LearningClusterInput) (model.AdminLearningCluster, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	if err := validateTaxonomySlug(input.Slug); err != nil || strings.TrimSpace(input.TitleID) == "" || strings.TrimSpace(input.TitleEN) == "" {
		return model.AdminLearningCluster{}, ErrLearningHubAdminInvalid
	}
	cluster, err := s.repo.UpdateLearningCluster(ctx, id, input)
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_cluster_updated", id, map[string]any{"slug": cluster.Slug})
	}
	return cluster, err
}

func (s *LearningHubService) DeleteCluster(ctx context.Context, actor, id string) error {
	if err := s.repo.HardDeleteLearningCluster(ctx, id); err != nil {
		return err
	}
	s.recordLearningAudit(ctx, actor, "learning_hub_cluster_deleted", id, nil)
	return nil
}

func (s *LearningHubService) CreateProgram(ctx context.Context, actor string, input model.AcademicProgramInput) (model.AdminAcademicProgram, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.PrimaryClusterSlug = strings.ToLower(strings.TrimSpace(input.PrimaryClusterSlug))
	if strings.TrimSpace(input.Name) == "" && strings.TrimSpace(input.NameID) != "" {
		input.Name = strings.TrimSpace(input.NameID)
	}
	if strings.TrimSpace(input.NameEN) == "" && strings.TrimSpace(input.Name) != "" {
		input.NameEN = strings.TrimSpace(input.Name)
	}
	if err := validateTaxonomySlug(input.Slug); err != nil {
		return model.AdminAcademicProgram{}, ErrLearningHubAdminInvalid
	}
	if err := validateTaxonomySlug(input.PrimaryClusterSlug); err != nil || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Degree) == "" {
		return model.AdminAcademicProgram{}, ErrLearningHubAdminInvalid
	}
	cluster, err := s.repo.LearningClusterBySlug(ctx, input.PrimaryClusterSlug)
	if errors.Is(err, repository.ErrLearningAdminNotFound) || !cluster.Active {
		return model.AdminAcademicProgram{}, ErrLearningHubAdminInvalid
	}
	if err != nil {
		return model.AdminAcademicProgram{}, err
	}
	program, err := s.repo.CreateLearningProgram(ctx, input)
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_program_created", program.ID, map[string]any{"slug": program.Slug})
	}
	return program, err
}

func (s *LearningHubService) UpdateProgram(ctx context.Context, actor, id string, input model.AcademicProgramInput) (model.AdminAcademicProgram, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.PrimaryClusterSlug = strings.ToLower(strings.TrimSpace(input.PrimaryClusterSlug))
	if strings.TrimSpace(input.Name) == "" && strings.TrimSpace(input.NameID) != "" {
		input.Name = strings.TrimSpace(input.NameID)
	}
	if strings.TrimSpace(input.NameEN) == "" && strings.TrimSpace(input.Name) != "" {
		input.NameEN = strings.TrimSpace(input.Name)
	}
	if err := validateTaxonomySlug(input.Slug); err != nil {
		return model.AdminAcademicProgram{}, ErrLearningHubAdminInvalid
	}
	if err := validateTaxonomySlug(input.PrimaryClusterSlug); err != nil || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Degree) == "" {
		return model.AdminAcademicProgram{}, ErrLearningHubAdminInvalid
	}
	cluster, err := s.repo.LearningClusterBySlug(ctx, input.PrimaryClusterSlug)
	if errors.Is(err, repository.ErrLearningAdminNotFound) || !cluster.Active {
		return model.AdminAcademicProgram{}, ErrLearningHubAdminInvalid
	}
	if err != nil {
		return model.AdminAcademicProgram{}, err
	}
	program, err := s.repo.UpdateLearningProgram(ctx, id, input)
	if err == nil {
		s.recordLearningAudit(ctx, actor, "learning_hub_program_updated", id, map[string]any{"slug": program.Slug})
	}
	return program, err
}

func (s *LearningHubService) DeleteProgram(ctx context.Context, actor, id string) error {
	if err := s.repo.DeactivateLearningProgram(ctx, id); err != nil {
		return err
	}
	s.recordLearningAudit(ctx, actor, "learning_hub_program_archived", id, nil)
	return nil
}

func (s *LearningHubService) recordLearningAudit(ctx context.Context, actorID, action, target string, metadata map[string]any) {
	actor, ok := s.repo.UserByID(ctx, actorID)
	if !ok {
		return
	}
	_ = s.repo.SaveAuditEvent(ctx, model.AuditEvent{ID: "audit_" + uuid.NewString()[:12], ActorID: actor.ID, Actor: actor.Email, Action: action, TargetType: "learning_hub", Target: target, Metadata: metadata, CreatedAt: time.Now().UTC()})
}
