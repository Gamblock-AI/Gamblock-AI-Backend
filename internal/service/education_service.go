package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

const (
	maxEducationImageBytes = 10 << 20
	maxEducationVideoBytes = 100 << 20
	maxEducationPDFBytes   = 25 << 20
)

var allowedRichTextNodes = map[string]bool{
	"doc": true, "paragraph": true, "text": true, "heading": true,
	"bulletList": true, "orderedList": true, "listItem": true,
	"blockquote": true, "horizontalRule": true, "hardBreak": true,
	"image": true, "video": true, "pdf": true, "table": true,
	"tableRow": true, "tableHeader": true, "tableCell": true,
}

var allowedRichTextMarks = map[string]bool{
	"bold": true, "italic": true, "underline": true, "strike": true,
	"link": true, "code": true,
}

type EducationService struct {
	repo *repository.Repository
	cfg  config.Config
}

func NewEducationService(repo *repository.Repository, cfg config.Config) *EducationService {
	return &EducationService{repo: repo, cfg: cfg}
}

func normalizeLocale(locale string) string {
	if strings.EqualFold(locale, "en") {
		return "en"
	}
	return "id"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateRichText(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		if nodeType, ok := typed["type"].(string); ok && !allowedRichTextNodes[nodeType] {
			return fmt.Errorf("unsupported rich-text node %q", nodeType)
		}
		if marks, ok := typed["marks"].([]any); ok {
			for _, mark := range marks {
				entry, ok := mark.(map[string]any)
				if !ok {
					return errors.New("invalid rich-text mark")
				}
				kind, _ := entry["type"].(string)
				if !allowedRichTextMarks[kind] {
					return fmt.Errorf("unsupported rich-text mark %q", kind)
				}
			}
		}
		for _, child := range typed {
			if err := validateRichText(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := validateRichText(child); err != nil {
				return err
			}
		}
	case string, float64, bool, nil:
		return nil
	default:
		return fmt.Errorf("invalid rich-text value %T", value)
	}
	return nil
}

func validateEducationDocument(document model.EducationDocument) error {
	if document.Audience == "" {
		document.Audience = "all"
	}
	if document.ExperienceType == "" {
		document.ExperienceType = "article"
	}
	if !slices.Contains([]string{"student", "partner", "all"}, document.Audience) {
		return errors.New("education audience is invalid")
	}
	if !slices.Contains([]string{"article", "partner_response_simulator"}, document.ExperienceType) {
		return errors.New("education experience type is invalid")
	}
	if document.ExperienceType == "partner_response_simulator" && document.Audience == "student" {
		return errors.New("partner simulator cannot target students")
	}
	if document.EstimatedMinutes < 1 || document.EstimatedMinutes > 120 {
		return errors.New("estimated minutes must be between 1 and 120")
	}
	if strings.TrimSpace(document.ReviewerName) == "" || strings.TrimSpace(document.ReviewedAt) == "" {
		return errors.New("review metadata is required")
	}
	if len(document.Sources) == 0 {
		return errors.New("at least one source is required")
	}
	if len(document.Thumbnails) == 0 || len(document.Thumbnails) > 8 {
		return errors.New("one to eight thumbnails are required")
	}
	if len(document.Sections) == 0 {
		return errors.New("at least one section is required")
	}
	for _, locale := range []string{"id", "en"} {
		translation, ok := document.Translations[locale]
		if !ok || strings.TrimSpace(translation.Title) == "" || strings.TrimSpace(translation.Summary) == "" || strings.TrimSpace(translation.LearningObjective) == "" || strings.TrimSpace(translation.Disclaimer) == "" {
			return fmt.Errorf("locale %s is incomplete", locale)
		}
		if strings.TrimSpace(translation.ReviewerRole) == "" && strings.TrimSpace(document.ReviewerRole) == "" {
			return fmt.Errorf("locale %s reviewer role is incomplete", locale)
		}
	}
	sectionIDs, checkIDs := map[string]bool{}, map[string]bool{}
	for _, section := range document.Sections {
		if section.ID == "" || sectionIDs[section.ID] {
			return errors.New("section ids must be unique")
		}
		sectionIDs[section.ID] = true
		for _, locale := range []string{"id", "en"} {
			translation, ok := section.Translations[locale]
			if !ok || strings.TrimSpace(translation.Title) == "" {
				return fmt.Errorf("section %s locale %s is incomplete", section.ID, locale)
			}
			if err := validateRichText(map[string]any(translation.Content)); err != nil {
				return fmt.Errorf("section %s: %w", section.ID, err)
			}
			check := translation.KnowledgeCheck
			if check == nil || check.ID == "" || strings.TrimSpace(check.Question) == "" || len(check.Choices) < 2 || check.CorrectChoiceID == "" {
				return fmt.Errorf("section %s locale %s needs a complete knowledge check", section.ID, locale)
			}
			if locale == "id" {
				if checkIDs[check.ID] {
					return errors.New("knowledge check ids must be unique")
				}
				checkIDs[check.ID] = true
			}
			if section.Translations["id"].KnowledgeCheck.ID != section.Translations["en"].KnowledgeCheck.ID {
				return fmt.Errorf("section %s check ids must match across locales", section.ID)
			}
			foundCorrect := false
			for _, choice := range check.Choices {
				if choice.ID == check.CorrectChoiceID {
					foundCorrect = true
				}
			}
			if !foundCorrect {
				return fmt.Errorf("section %s correct choice is missing", section.ID)
			}
		}
	}
	for _, thumbnail := range document.Thumbnails {
		if thumbnail.MediaID == "" || strings.TrimSpace(thumbnail.AltText["id"]) == "" || strings.TrimSpace(thumbnail.AltText["en"]) == "" {
			return errors.New("thumbnail and bilingual alt text are required")
		}
	}
	for _, source := range document.Sources {
		parsed, err := url.Parse(source.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" {
			return errors.New("sources must use valid HTTPS URLs")
		}
	}
	return nil
}

func collectDocumentMedia(document model.EducationDocument) (all []string, required []string) {
	seen, requiredSeen := map[string]bool{}, map[string]bool{}
	for _, thumbnail := range document.Thumbnails {
		if !seen[thumbnail.MediaID] {
			seen[thumbnail.MediaID] = true
			all = append(all, thumbnail.MediaID)
		}
	}
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if attrs, ok := typed["attrs"].(map[string]any); ok {
				id, _ := attrs["media_id"].(string)
				if id != "" && !seen[id] {
					seen[id] = true
					all = append(all, id)
				}
				isRequired, _ := attrs["required"].(bool)
				if id != "" && isRequired && !requiredSeen[id] {
					requiredSeen[id] = true
					required = append(required, id)
				}
			}
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	for _, section := range document.Sections {
		for _, translation := range section.Translations {
			walk(map[string]any(translation.Content))
		}
	}
	return all, required
}

func (s *EducationService) ensureMedia(ctx context.Context, document model.EducationDocument) ([]string, error) {
	ids, _ := collectDocumentMedia(document)
	thumbnailIDs := map[string]bool{}
	for _, thumbnail := range document.Thumbnails {
		thumbnailIDs[thumbnail.MediaID] = true
	}
	for _, id := range ids {
		media, err := s.repo.GetEducationMedia(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("media %s: %w", id, err)
		}
		if thumbnailIDs[id] && (media.Purpose != "thumbnail" || media.MediaType != "image" || media.Kind != "upload") {
			return nil, fmt.Errorf("thumbnail %s must be an uploaded image", id)
		}
	}
	return ids, nil
}

func (s *EducationService) AdminModules(ctx context.Context) ([]model.EducationModule, error) {
	return s.repo.GetEducationModules(ctx)
}

func (s *EducationService) AdminModule(ctx context.Context, id string) (model.EducationModule, error) {
	return s.repo.GetEducationModuleByID(ctx, id)
}

func (s *EducationService) CreateModule(ctx context.Context, actor, slug string, document model.EducationDocument) (model.EducationModule, error) {
	document.Audience, document.ExperienceType = normalizedEducationExperience(document)
	if strings.TrimSpace(slug) == "" || len(document.Translations) == 0 {
		return model.EducationModule{}, errors.New("slug and initial translations are required")
	}
	id := "mod_" + uuid.NewString()[:8]
	idTranslation := document.Translations["id"]
	module := model.EducationModule{ID: id, Slug: slug, Title: idTranslation.Title, Summary: idTranslation.Summary,
		EstimatedMinutes: document.EstimatedMinutes, Status: "draft", DraftDocument: document,
		DraftRevision: 1, CreatedBy: actor, UpdatedBy: actor, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.repo.CreateEducationModule(ctx, module); err != nil {
		return model.EducationModule{}, err
	}
	created, err := s.repo.GetEducationModuleByID(ctx, id)
	if err != nil {
		return model.EducationModule{}, err
	}
	if err = s.saveEducationRevision(ctx, created, "draft", actor); err != nil {
		return model.EducationModule{}, err
	}
	s.recordEducationAudit(ctx, actor, "education_module_created", created.ID, map[string]any{"revision": created.DraftRevision})
	return created, nil
}

func (s *EducationService) UpdateDraft(ctx context.Context, actor, id, slug string, expectedRevision int, document model.EducationDocument) (model.EducationModule, error) {
	document.Audience, document.ExperienceType = normalizedEducationExperience(document)
	if strings.TrimSpace(slug) == "" || document.EstimatedMinutes < 1 {
		return model.EducationModule{}, errors.New("slug and estimated minutes are required")
	}
	updated, err := s.repo.UpdateEducationDraft(ctx, id, expectedRevision, slug, document, actor)
	if err != nil {
		return model.EducationModule{}, err
	}
	if err = s.saveEducationRevision(ctx, updated, "draft", actor); err != nil {
		return model.EducationModule{}, err
	}
	s.recordEducationAudit(ctx, actor, "education_module_updated", updated.ID, map[string]any{"revision": updated.DraftRevision})
	return updated, nil
}

func (s *EducationService) SubmitReview(ctx context.Context, actor, id string) (model.EducationModule, error) {
	module, err := s.repo.GetEducationModuleByID(ctx, id)
	if err != nil {
		return model.EducationModule{}, err
	}
	if err = validateEducationDocument(module.DraftDocument); err != nil {
		return model.EducationModule{}, err
	}
	if _, err = s.ensureMedia(ctx, module.DraftDocument); err != nil {
		return model.EducationModule{}, err
	}
	updated, err := s.repo.SetEducationStatus(ctx, id, "in_review", actor, false)
	if err == nil {
		s.recordEducationAudit(ctx, actor, "education_module_submitted", id, map[string]any{"revision": updated.DraftRevision})
	}
	return updated, err
}

func (s *EducationService) Publish(ctx context.Context, actor, id string) (model.EducationModule, error) {
	module, err := s.repo.GetEducationModuleByID(ctx, id)
	if err != nil {
		return model.EducationModule{}, err
	}
	if module.Status != "in_review" {
		return model.EducationModule{}, errors.New("module must be in review before publishing")
	}
	if err = validateEducationDocument(module.DraftDocument); err != nil {
		return model.EducationModule{}, err
	}
	mediaIDs, err := s.ensureMedia(ctx, module.DraftDocument)
	if err != nil {
		return model.EducationModule{}, err
	}
	if err = s.repo.PublishEducationMedia(ctx, mediaIDs); err != nil {
		return model.EducationModule{}, err
	}
	published, err := s.repo.SetEducationStatus(ctx, id, "published", actor, true)
	if err != nil {
		return model.EducationModule{}, err
	}
	if err = s.saveEducationRevision(ctx, published, "published", actor); err != nil {
		return model.EducationModule{}, err
	}
	s.recordEducationAudit(ctx, actor, "education_module_published", id, map[string]any{"revision": published.PublishedRevision})
	return published, nil
}

func (s *EducationService) Archive(ctx context.Context, actor, id string) (model.EducationModule, error) {
	archived, err := s.repo.SetEducationStatus(ctx, id, "archived", actor, false)
	if err == nil {
		s.recordEducationAudit(ctx, actor, "education_module_archived", id, nil)
	}
	return archived, err
}

func (s *EducationService) Revisions(ctx context.Context, moduleID string) ([]model.EducationRevision, error) {
	if _, err := s.repo.GetEducationModuleByID(ctx, moduleID); err != nil {
		return nil, err
	}
	return s.repo.ListEducationRevisions(ctx, moduleID)
}

func (s *EducationService) Rollback(ctx context.Context, actor, moduleID, revisionID, reason string) (model.EducationModule, error) {
	if strings.TrimSpace(reason) == "" {
		return model.EducationModule{}, errors.New("rollback reason is required")
	}
	current, err := s.repo.GetEducationModuleByID(ctx, moduleID)
	if err != nil {
		return model.EducationModule{}, err
	}
	revision, err := s.repo.EducationRevisionByID(ctx, moduleID, revisionID)
	if err != nil {
		return model.EducationModule{}, err
	}
	updated, err := s.repo.UpdateEducationDraft(ctx, moduleID, current.DraftRevision, revision.Slug, revision.Document, actor)
	if err != nil {
		return model.EducationModule{}, err
	}
	if err = s.saveEducationRevision(ctx, updated, "rollback", actor); err != nil {
		return model.EducationModule{}, err
	}
	s.recordEducationAudit(ctx, actor, "education_module_rolled_back", moduleID, map[string]any{"source_revision_id": revisionID, "reason": strings.TrimSpace(reason), "revision": updated.DraftRevision})
	return updated, nil
}

func (s *EducationService) saveEducationRevision(ctx context.Context, module model.EducationModule, kind, actor string) error {
	revision, document := module.DraftRevision, module.DraftDocument
	if kind == "published" && module.PublishedDocument != nil {
		revision, document = module.PublishedRevision, *module.PublishedDocument
	}
	return s.repo.SaveEducationRevision(ctx, model.EducationRevision{
		ID: "edrev_" + uuid.NewString()[:12], ModuleID: module.ID, Revision: revision,
		Document: document, Slug: module.Slug, Kind: kind, CreatedBy: actor, CreatedAt: time.Now().UTC(),
	})
}

func (s *EducationService) recordEducationAudit(ctx context.Context, actorID, action, moduleID string, metadata map[string]any) {
	actor, ok := s.repo.UserByID(ctx, actorID)
	if !ok {
		return
	}
	_ = s.repo.SaveAuditEvent(ctx, model.AuditEvent{
		ID: "audit_" + uuid.NewString()[:12], ActorID: actor.ID, Actor: actor.Email,
		Action: action, TargetType: "education_module", Target: moduleID, Metadata: metadata, CreatedAt: time.Now().UTC(),
	})
}

func (s *EducationService) localizedModule(ctx context.Context, module model.EducationModule, locale, userID string) (model.LocalizedEducationModule, error) {
	if module.Status != "published" || module.PublishedDocument == nil {
		return model.LocalizedEducationModule{}, repository.ErrEducationNotFound
	}
	locale = normalizeLocale(locale)
	document := *module.PublishedDocument
	document.Audience, document.ExperienceType = normalizedEducationExperience(document)
	translation := document.Translations[locale]
	progress, err := s.repo.GetEducationProgress(ctx, userID, module.ID, module.PublishedRevision)
	if err != nil {
		return model.LocalizedEducationModule{}, err
	}
	sections := make([]model.LocalizedEducationSection, 0, len(document.Sections))
	for _, section := range document.Sections {
		localized := section.Translations[locale]
		check := localized.KnowledgeCheck
		if check != nil && !slices.Contains(progress.CorrectCheckIDs, check.ID) {
			copyCheck := *check
			copyCheck.CorrectChoiceID, copyCheck.Explanation = "", ""
			check = &copyCheck
		}
		sections = append(sections, model.LocalizedEducationSection{ID: section.ID, SortOrder: section.SortOrder,
			Required: section.Required, Title: localized.Title, Content: localized.Content, KnowledgeCheck: check})
	}
	sort.Slice(sections, func(i, j int) bool { return sections[i].SortOrder < sections[j].SortOrder })
	mediaIDs, _ := collectDocumentMedia(document)
	mediaURLs := map[string]string{}
	for _, id := range mediaIDs {
		media, mediaErr := s.repo.GetEducationMedia(ctx, id)
		if mediaErr != nil {
			continue
		}
		if media.Kind == "external" {
			mediaURLs[id] = media.ExternalURL
		} else {
			mediaURLs[id] = "/v1/education/media/" + id
		}
	}
	return model.LocalizedEducationModule{
		ID: module.ID, Slug: module.Slug, Locale: locale, Title: translation.Title, Summary: translation.Summary,
		LearningObjective: translation.LearningObjective, Disclaimer: translation.Disclaimer,
		Category: document.Category, Audience: document.Audience, ExperienceType: document.ExperienceType, EstimatedMinutes: document.EstimatedMinutes,
		ReviewerName: document.ReviewerName, ReviewerRole: firstNonEmpty(translation.ReviewerRole, document.ReviewerRole), ReviewedAt: document.ReviewedAt,
		Revision: module.PublishedRevision, Thumbnails: document.Thumbnails, ThumbnailURLs: mediaURLs,
		MediaURLs: mediaURLs, Sources: document.Sources, Sections: sections, Progress: progress, UpdatedAt: module.UpdatedAt,
	}, nil
}

func (s *EducationService) PublishedModules(ctx context.Context, userID, locale string) ([]model.LocalizedEducationModule, error) {
	modules, err := s.repo.GetPublishedEducationModules(ctx)
	if err != nil {
		return nil, err
	}
	localized := make([]model.LocalizedEducationModule, 0, len(modules))
	role := "user"
	if user, ok := s.repo.UserByID(ctx, userID); ok {
		role = user.Role
	}
	for _, module := range modules {
		if module.PublishedDocument == nil {
			continue
		}
		document := *module.PublishedDocument
		document.Audience, document.ExperienceType = normalizedEducationExperience(document)
		if !educationAudienceAllows(document.Audience, role) {
			continue
		}
		item, localizeErr := s.localizedModule(ctx, module, locale, userID)
		if localizeErr != nil {
			return nil, localizeErr
		}
		localized = append(localized, item)
	}
	return localized, nil
}

func (s *EducationService) PublishedModule(ctx context.Context, userID, slug, locale string) (model.LocalizedEducationModule, error) {
	module, err := s.repo.GetEducationModuleBySlug(ctx, slug)
	if err != nil {
		return model.LocalizedEducationModule{}, err
	}
	role := "user"
	if user, ok := s.repo.UserByID(ctx, userID); ok {
		role = user.Role
	}
	if module.PublishedDocument == nil {
		return model.LocalizedEducationModule{}, repository.ErrEducationNotFound
	}
	document := *module.PublishedDocument
	document.Audience, document.ExperienceType = normalizedEducationExperience(document)
	if !educationAudienceAllows(document.Audience, role) {
		return model.LocalizedEducationModule{}, repository.ErrEducationNotFound
	}
	return s.localizedModule(ctx, module, locale, userID)
}

func normalizedEducationExperience(document model.EducationDocument) (string, string) {
	audience, experience := document.Audience, document.ExperienceType
	if audience == "" {
		audience = "all"
	}
	if experience == "" {
		experience = "article"
	}
	return audience, experience
}

func educationAudienceAllows(audience, role string) bool {
	if audience == "all" {
		return role == "user" || role == "partner"
	}
	return (audience == "student" && role == "user") || (audience == "partner" && role == "partner")
}

func requiredItems(document model.EducationDocument) (sections, media, checks []string) {
	_, media = collectDocumentMedia(document)
	for _, section := range document.Sections {
		if section.Required {
			sections = append(sections, section.ID)
		}
		if check := section.Translations["id"].KnowledgeCheck; check != nil && check.Required {
			checks = append(checks, check.ID)
		}
	}
	return sections, media, checks
}

func calculateProgress(progress *model.EducationProgress, document model.EducationDocument) {
	sections, media, checks := requiredItems(document)
	total := len(sections) + len(media) + len(checks)
	done := 0
	for _, id := range sections {
		if slices.Contains(progress.CompletedSectionIDs, id) {
			done++
		}
	}
	for _, id := range media {
		if slices.Contains(progress.OpenedMediaIDs, id) {
			done++
		}
	}
	for _, id := range checks {
		if slices.Contains(progress.CorrectCheckIDs, id) {
			done++
		}
	}
	if total == 0 {
		progress.ProgressPercent = 0
	} else {
		progress.ProgressPercent = done * 100 / total
	}
	if done == total && total > 0 {
		now := time.Now().UTC()
		progress.ProgressPercent = 100
		if progress.CompletedAt == nil {
			progress.CompletedAt = &now
		}
	} else {
		progress.CompletedAt = nil
	}
}

func filterAllowed(values, allowed []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if slices.Contains(allowed, value) && !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}

func (s *EducationService) UpdateProgress(ctx context.Context, userID, moduleID string, revision int, input model.EducationProgressInput) (model.EducationProgress, error) {
	module, err := s.repo.GetEducationModuleByID(ctx, moduleID)
	if err != nil || module.PublishedDocument == nil || module.PublishedRevision != revision {
		return model.EducationProgress{}, repository.ErrEducationNotFound
	}
	progress, err := s.repo.GetEducationProgress(ctx, userID, moduleID, revision)
	if err != nil {
		return model.EducationProgress{}, err
	}
	sections, media, _ := requiredItems(*module.PublishedDocument)
	progress.CompletedSectionIDs = filterAllowed(input.CompletedSectionIDs, sections)
	progress.OpenedMediaIDs = filterAllowed(input.OpenedMediaIDs, media)
	calculateProgress(&progress, *module.PublishedDocument)
	return s.repo.SaveEducationProgress(ctx, progress)
}

func (s *EducationService) AnswerCheck(ctx context.Context, userID, moduleID string, revision int, checkID, choiceID, locale string) (model.EducationCheckResult, error) {
	module, err := s.repo.GetEducationModuleByID(ctx, moduleID)
	if err != nil || module.PublishedDocument == nil || module.PublishedRevision != revision {
		return model.EducationCheckResult{}, repository.ErrEducationNotFound
	}
	locale = normalizeLocale(locale)
	var check *model.EducationKnowledgeCheck
	for _, section := range module.PublishedDocument.Sections {
		candidate := section.Translations[locale].KnowledgeCheck
		if candidate != nil && candidate.ID == checkID {
			check = candidate
			break
		}
	}
	if check == nil {
		return model.EducationCheckResult{}, repository.ErrEducationNotFound
	}
	progress, err := s.repo.GetEducationProgress(ctx, userID, moduleID, revision)
	if err != nil {
		return model.EducationCheckResult{}, err
	}
	correct := check.CorrectChoiceID == choiceID
	if correct && !slices.Contains(progress.CorrectCheckIDs, checkID) {
		progress.CorrectCheckIDs = append(progress.CorrectCheckIDs, checkID)
	}
	calculateProgress(&progress, *module.PublishedDocument)
	progress, err = s.repo.SaveEducationProgress(ctx, progress)
	if err != nil {
		return model.EducationCheckResult{}, err
	}
	return model.EducationCheckResult{Correct: correct, Explanation: check.Explanation, Progress: progress}, nil
}

func mediaSpec(sniff []byte) (mediaType, mime string, maxBytes int64, ok bool) {
	mime = http.DetectContentType(sniff)
	switch mime {
	case "image/jpeg", "image/png", "image/webp":
		return "image", mime, maxEducationImageBytes, true
	case "video/mp4", "video/webm":
		return "video", mime, maxEducationVideoBytes, true
	case "application/pdf":
		return "pdf", mime, maxEducationPDFBytes, true
	default:
		return "", mime, 0, false
	}
}

func (s *EducationService) UploadMedia(ctx context.Context, actor, purpose, originalName string, reader io.Reader) (model.EducationMedia, error) {
	if purpose != "thumbnail" && purpose != "content" {
		return model.EducationMedia{}, errors.New("invalid media purpose")
	}
	if err := os.MkdirAll(s.cfg.MediaStoragePath, 0o750); err != nil {
		return model.EducationMedia{}, err
	}
	temp, err := os.CreateTemp(s.cfg.MediaStoragePath, ".education-upload-*")
	if err != nil {
		return model.EducationMedia{}, err
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(reader, maxEducationVideoBytes+1))
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return model.EducationMedia{}, err
	}
	file, err := os.Open(tempName)
	if err != nil {
		return model.EducationMedia{}, err
	}
	sniff := make([]byte, 512)
	n, readErr := file.Read(sniff)
	_ = file.Close()
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return model.EducationMedia{}, readErr
	}
	mediaType, mime, maxBytes, ok := mediaSpec(sniff[:n])
	if !ok || written > maxBytes {
		return model.EducationMedia{}, errors.New("unsupported media type or file too large")
	}
	if purpose == "thumbnail" && mediaType != "image" {
		return model.EducationMedia{}, errors.New("thumbnail must be an image")
	}
	extension := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "video/mp4": ".mp4", "video/webm": ".webm", "application/pdf": ".pdf"}[mime]
	id := "med_" + uuid.NewString()
	storageKey := id + extension
	target := filepath.Join(s.cfg.MediaStoragePath, storageKey)
	if err = os.Rename(tempName, target); err != nil {
		return model.EducationMedia{}, err
	}
	media := model.EducationMedia{ID: id, Kind: "upload", Purpose: purpose, MediaType: mediaType,
		MIMEType: mime, StorageKey: storageKey, OriginalName: filepath.Base(originalName), SizeBytes: written,
		SHA256: hex.EncodeToString(hash.Sum(nil)), Status: "draft", CreatedBy: actor, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err = s.repo.CreateEducationMedia(ctx, media); err != nil {
		_ = os.Remove(target)
		return model.EducationMedia{}, err
	}
	return media, nil
}

func (s *EducationService) RegisterExternalMedia(ctx context.Context, actor, purpose, mediaType, rawURL string) (model.EducationMedia, error) {
	if purpose != "content" || (mediaType != "image" && mediaType != "video" && mediaType != "pdf") {
		return model.EducationMedia{}, errors.New("invalid external media")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" {
		return model.EducationMedia{}, errors.New("external media must use an allowed HTTPS URL")
	}
	allowed := false
	for _, host := range s.cfg.MediaEmbedHosts {
		if strings.EqualFold(host, parsed.Hostname()) {
			allowed = true
			break
		}
	}
	if !allowed {
		return model.EducationMedia{}, errors.New("external media host is not allowed")
	}
	media := model.EducationMedia{ID: "med_" + uuid.NewString(), Kind: "external", Purpose: purpose,
		MediaType: mediaType, MIMEType: "text/uri-list", ExternalURL: parsed.String(), Status: "draft",
		CreatedBy: actor, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err = s.repo.CreateEducationMedia(ctx, media); err != nil {
		return model.EducationMedia{}, err
	}
	return media, nil
}

func (s *EducationService) MediaFile(ctx context.Context, id string, allowDraft bool) (model.EducationMedia, string, error) {
	media, err := s.repo.GetEducationMedia(ctx, id)
	if err != nil {
		return model.EducationMedia{}, "", err
	}
	if media.Kind != "upload" || (!allowDraft && media.Status != "published") {
		return model.EducationMedia{}, "", repository.ErrEducationNotFound
	}
	root, err := filepath.Abs(s.cfg.MediaStoragePath)
	if err != nil {
		return model.EducationMedia{}, "", err
	}
	path, err := filepath.Abs(filepath.Join(root, media.StorageKey))
	if err != nil || (!strings.HasPrefix(path, root+string(os.PathSeparator)) && path != root) {
		return model.EducationMedia{}, "", errors.New("media path escapes storage root")
	}
	return media, path, nil
}
