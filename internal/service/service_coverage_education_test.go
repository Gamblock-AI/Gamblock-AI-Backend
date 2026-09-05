package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEducationValidationHelpers(t *testing.T) {
	assert.Equal(t, "en", normalizeLocale("EN"))
	assert.Equal(t, "id", normalizeLocale("nl"))
	assert.Equal(t, "first", firstNonEmpty(" ", "first", "second"))
	assert.Empty(t, firstNonEmpty(" ", ""))

	require.NoError(t, validateRichText(map[string]any{"type": "paragraph", "content": []any{"text", 1.0, true, nil}}))
	require.EqualError(t, validateRichText(map[string]any{"type": "iframe"}), `unsupported rich-text node "iframe"`)
	require.EqualError(t, validateRichText(map[string]any{"marks": []any{"bad"}}), "invalid rich-text mark")
	require.EqualError(t, validateRichText(map[string]any{"marks": []any{map[string]any{"type": "blink"}}}), `unsupported rich-text mark "blink"`)
	require.EqualError(t, validateRichText(struct{}{}), "invalid rich-text value struct {}")

	valid := model.EducationDocument{
		Audience: "all", ExperienceType: "article", EstimatedMinutes: 10,
		ReviewerName: "Reviewer", ReviewerRole: "Counselor", ReviewedAt: "2026-09-01",
		Translations: map[string]model.EducationTranslation{
			"id": {Title: "Judul", Summary: "Ringkasan", LearningObjective: "Tujuan", Disclaimer: "Catatan"},
			"en": {Title: "Title", Summary: "Summary", LearningObjective: "Objective", Disclaimer: "Disclaimer"},
		},
		Sections: []model.EducationSection{{ID: "one", Required: true, Translations: map[string]model.EducationSectionTranslation{
			"id": {Title: "Bagian", Content: model.RichTextDocument{"type": "paragraph"}, KnowledgeCheck: &model.EducationKnowledgeCheck{ID: "check", Question: "?", Choices: []model.EducationChoice{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectChoiceID: "a"}},
			"en": {Title: "Section", Content: model.RichTextDocument{"type": "paragraph"}, KnowledgeCheck: &model.EducationKnowledgeCheck{ID: "check", Question: "?", Choices: []model.EducationChoice{{ID: "a", Text: "A"}, {ID: "b", Text: "B"}}, CorrectChoiceID: "a"}},
		}}},
		Thumbnails: []model.EducationThumbnail{{MediaID: "thumb", AltText: map[string]string{"id": "id", "en": "en"}}},
		Videos:     []model.EducationVideo{{MediaID: "video", AltText: map[string]string{"id": "id", "en": "en"}}},
		Sources:    []model.EducationSource{{URL: "https://example.com/source"}},
	}
	require.NoError(t, validateEducationDocument(valid))
	invalid := valid
	invalid.Audience = "admin"
	require.EqualError(t, validateEducationDocument(invalid), "education audience is invalid")
	invalid = valid
	invalid.ExperienceType, invalid.Audience = "partner_response_simulator", "student"
	require.EqualError(t, validateEducationDocument(invalid), "partner simulator cannot target students")
	invalid = valid
	invalid.Thumbnails = nil
	require.EqualError(t, validateEducationDocument(invalid), "one to eight thumbnails are required")
	invalid = valid
	invalid.Sources[0].URL = "http://example.com"
	require.EqualError(t, validateEducationDocument(invalid), "sources must use valid HTTPS URLs")

	allIDs, requiredIDs := collectDocumentMedia(valid)
	assert.ElementsMatch(t, []string{"thumb", "video"}, allIDs)
	assert.Empty(t, requiredIDs)
	sections, media, checks := requiredItems(valid)
	assert.Equal(t, []string{"one"}, sections)
	assert.Empty(t, media, "thumbnail media is not required until it is embedded as required content")
	assert.Empty(t, checks)
	progress := model.EducationProgress{CompletedSectionIDs: []string{"one"}, OpenedMediaIDs: []string{"thumb", "video"}}
	calculateProgress(&progress, valid)
	assert.Equal(t, 100, progress.ProgressPercent)
	require.NotNil(t, progress.CompletedAt)
	assert.Equal(t, []string{"one"}, filterAllowed([]string{"one", "one", "unknown"}, []string{"one"}))
}

func TestEducationEditorialLifecycleAndProgress(t *testing.T) {
	ctx := context.Background()
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	svc := NewEducationService(repo, testCfg())
	doc := st.Modules[1].DraftDocument
	created, err := svc.CreateModule(ctx, "usr_nasywa", "coverage-education", doc)
	require.NoError(t, err)
	assert.Equal(t, "draft", created.Status)
	_, err = svc.SubmitReview(ctx, "usr_nasywa", created.ID)
	require.NoError(t, err)
	published, err := svc.Publish(ctx, "usr_nasywa", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)

	admin, err := svc.AdminModule(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, admin.ID)
	bySlug, err := svc.AdminModule(ctx, "coverage-education")
	require.NoError(t, err)
	assert.Equal(t, created.ID, bySlug.ID)
	mods, err := svc.AdminModules(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, mods)
	page, err := svc.AdminModulesPaginated(ctx, model.PaginationQuery{Query: "coverage"})
	require.NoError(t, err)
	assert.Len(t, page.Items, 1)

	localized, err := svc.PublishedModule(ctx, "usr_gading", "coverage-education", "EN")
	require.NoError(t, err)
	assert.Equal(t, "en", localized.Locale)
	require.NotEmpty(t, localized.Sections)
	check := localized.Sections[0].KnowledgeCheck
	require.NotNil(t, check)
	assert.Empty(t, check.CorrectChoiceID, "answers must be hidden before a correct answer")
	modules, err := svc.PublishedModules(ctx, "usr_gading", "id")
	require.NoError(t, err)
	assert.NotEmpty(t, modules)
	filtered, err := svc.PublishedModulesPaginated(ctx, "usr_gading", "id", model.PaginationQuery{Query: doc.Translations["id"].Title})
	require.NoError(t, err)
	assert.NotEmpty(t, filtered.Items)

	progress, err := svc.UpdateProgress(ctx, "usr_gading", created.ID, published.PublishedRevision, model.EducationProgressInput{
		CompletedSectionIDs: []string{localized.Sections[0].ID, "not-allowed"},
		OpenedMediaIDs:      []string{"not-allowed"},
	})
	require.NoError(t, err)
	assert.Equal(t, 25, progress.ProgressPercent)
	answer, err := svc.AnswerCheck(ctx, "usr_gading", created.ID, published.PublishedRevision, check.ID, check.Choices[0].ID, "id")
	require.NoError(t, err)
	assert.Equal(t, check.CorrectChoiceID == check.Choices[0].ID, answer.Correct)
	wrong, err := svc.AnswerCheck(ctx, "usr_gading", created.ID, published.PublishedRevision, check.ID, "wrong", "id")
	require.NoError(t, err)
	assert.False(t, wrong.Correct)
	_, err = svc.AnswerCheck(ctx, "usr_gading", created.ID, 999, check.ID, "wrong", "id")
	require.ErrorIs(t, err, repository.ErrEducationNotFound)

	revisions, err := svc.Revisions(ctx, created.ID)
	require.NoError(t, err)
	require.NotEmpty(t, revisions)
	_, err = svc.Rollback(ctx, "usr_nasywa", created.ID, revisions[0].ID, "restore reviewed draft")
	require.NoError(t, err)
	_, err = svc.Rollback(ctx, "usr_nasywa", created.ID, revisions[0].ID, "")
	require.EqualError(t, err, "rollback reason is required")
}

func TestEducationAudienceAndMediaWorkflows(t *testing.T) {
	ctx := context.Background()
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	cfg := testCfg()
	cfg.MediaStoragePath = t.TempDir()
	cfg.MediaEmbedHosts = []string{"www.youtube.com", "vimeo.com"}
	svc := NewEducationService(repo, cfg)

	assert.True(t, educationAudienceAllows("all", "user"))
	assert.True(t, educationAudienceAllows("partner", "partner"))
	assert.False(t, educationAudienceAllows("student", "partner"))
	audience, experience := normalizedEducationExperience(model.EducationDocument{})
	assert.Equal(t, "all", audience)
	assert.Equal(t, "article", experience)

	_, err := svc.UploadMedia(ctx, "usr_nasywa", "invalid", "x", bytes.NewReader([]byte("x")))
	require.EqualError(t, err, "invalid media purpose")
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x01}
	media, err := svc.UploadMedia(ctx, "usr_nasywa", "thumbnail", "../cover.png", bytes.NewReader(png))
	require.NoError(t, err)
	assert.Equal(t, "cover.png", media.OriginalName)
	_, path, err := svc.MediaFile(ctx, media.ID, true)
	require.NoError(t, err)
	assert.Contains(t, path, media.StorageKey)
	_, _, err = svc.MediaFile(ctx, media.ID, false)
	require.ErrorIs(t, err, repository.ErrEducationNotFound)

	video, err := svc.RegisterExternalMedia(ctx, "usr_nasywa", "content", "video", "https://www.youtube.com/watch?v=abc")
	require.NoError(t, err)
	assert.Equal(t, "external", video.Kind)
	_, err = svc.RegisterExternalMedia(ctx, "usr_nasywa", "thumbnail", "image", "https://www.youtube.com/a")
	require.EqualError(t, err, "invalid external media")
	_, err = svc.RegisterExternalMedia(ctx, "usr_nasywa", "content", "video", "https://evil.example/a")
	require.EqualError(t, err, "external media host is not allowed")
	assert.Equal(t, "image", mustMediaSpec(t, []byte("\xff\xd8\xff")))
	assert.Equal(t, "video", mustMediaSpec(t, []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2', 0, 0, 0, 0, 'm', 'p', '4', '2', 0, 0, 0, 0}))
	assert.Equal(t, "pdf", mustMediaSpec(t, []byte("%PDF-1.7")))
	_, _, _, ok := mediaSpec([]byte("plain text"))
	assert.False(t, ok)
}

func mustMediaSpec(t *testing.T, sniff []byte) string {
	t.Helper()
	mediaType, _, _, ok := mediaSpec(sniff)
	require.True(t, ok)
	return mediaType
}

func TestEducationAdminModuleUnknownAndInvalidProgress(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewEducationService(repo, testCfg())
	_, err := svc.AdminModule(ctx, "missing")
	require.Error(t, err)
	_, err = svc.PublishedModule(ctx, "usr_nasywa", "missing", "id")
	require.Error(t, err)
	_, err = svc.UpdateProgress(ctx, "usr_gading", "missing", 1, model.EducationProgressInput{})
	require.ErrorIs(t, err, repository.ErrEducationNotFound)
	_, err = svc.CreateModule(ctx, "usr_nasywa", "", model.EducationDocument{})
	require.EqualError(t, err, "slug and initial translations are required")
	_, err = svc.UpdateDraft(ctx, "usr_nasywa", "missing", "x", 1, model.EducationDocument{})
	require.EqualError(t, err, "slug and estimated minutes are required")
	err = svc.DeleteModule(ctx, "usr_nasywa", "missing")
	require.Error(t, err)
}

func TestEducationRawMediaSpecs(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", []byte{0xff, 0xd8, 0xff, 0xe0}, "image"},
		{"webp", []byte("RIFFxxxxWEBPVP"), "image"},
		{"webm", []byte{0x1a, 0x45, 0xdf, 0xa3, 0x93, 0x42, 0x82, 0x88, 0x77, 0x65, 0x62, 0x6d}, "video"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, _, ok := mediaSpec(tc.data)
			require.True(t, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}
