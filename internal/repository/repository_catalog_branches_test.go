package repository

import (
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func coverageEducationDocument(title string) model.EducationDocument {
	return model.EducationDocument{
		Audience:         "student",
		ExperienceType:   "article",
		Category:         "awareness",
		EstimatedMinutes: 12,
		Translations: map[string]model.EducationTranslation{
			"id": {Title: title, Summary: "Ringkasan", LearningObjective: "Tujuan"},
			"en": {Title: "English title", Summary: "Summary", LearningObjective: "Objective"},
		},
	}
}

func TestEducationRepository_InMemoryQueriesPublishingProgressAndDelete(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	draft := model.EducationModule{
		ID: "mod_cov", Slug: "coverage-module", Title: "Coverage", Summary: "Coverage summary", Status: "draft",
		DraftDocument: coverageEducationDocument("Coverage"), DraftRevision: 1, CreatedBy: "usr_nasywa", UpdatedBy: "usr_nasywa",
	}
	require.NoError(t, repo.CreateEducationModule(ctx, draft))

	modules, err := repo.GetEducationModules(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, modules)
	got, err := repo.GetEducationModuleByID(ctx, "mod_cov")
	require.NoError(t, err)
	assert.Equal(t, "coverage-module", got.Slug)
	got, err = repo.GetEducationModuleBySlug(ctx, "COVERAGE-MODULE")
	require.NoError(t, err)
	assert.Equal(t, "mod_cov", got.ID)
	_, err = repo.GetEducationModuleByID(ctx, "missing-module")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	_, err = repo.GetEducationModuleBySlug(ctx, "missing-module")
	assert.ErrorIs(t, err, ErrEducationNotFound)

	page, err := repo.GetAdminEducationModules(ctx, model.PaginationQuery{Status: "draft", Query: "COVERAGE", Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, page.TotalCount)
	assert.Len(t, page.Items, 1)
	page, err = repo.GetAdminEducationModules(ctx, model.PaginationQuery{Query: "not-found", Page: 2, Limit: 2})
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	_, err = repo.UpdateEducationDraft(ctx, "mod_cov", 99, "bad", coverageEducationDocument("Bad"), "editor")
	assert.ErrorIs(t, err, ErrEducationConflict)
	updated, err := repo.UpdateEducationDraft(ctx, "mod_cov", 1, "coverage-module-v2", coverageEducationDocument("Updated"), "editor")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.DraftRevision)
	assert.Equal(t, "Updated", updated.Title)
	assert.Equal(t, "editor", updated.UpdatedBy)
	_, err = repo.UpdateEducationDraft(ctx, "missing-module", 1, "missing", coverageEducationDocument("Missing"), "editor")
	assert.ErrorIs(t, err, ErrEducationNotFound)

	published, err := repo.SetEducationStatus(ctx, "mod_cov", "published", "reviewer", true)
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)
	assert.NotNil(t, published.PublishedDocument)
	assert.Equal(t, 2, published.PublishedRevision)
	publishedModules, err := repo.GetPublishedEducationModules(ctx)
	require.NoError(t, err)
	foundPublished := false
	for _, item := range publishedModules {
		if item.ID == "mod_cov" {
			foundPublished = true
		}
	}
	assert.True(t, foundPublished)

	media := model.EducationMedia{ID: "media_cov", Kind: "image", Purpose: "thumbnail", MediaType: "image", MIMEType: "image/webp", StorageKey: "education/media_cov.webp", Status: "draft", CreatedBy: "editor"}
	require.NoError(t, repo.CreateEducationMedia(ctx, media))
	gotMedia, err := repo.GetEducationMedia(ctx, media.ID)
	require.NoError(t, err)
	assert.Equal(t, media.StorageKey, gotMedia.StorageKey)
	_, err = repo.GetEducationMedia(ctx, "missing-media")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	require.NoError(t, repo.PublishEducationMedia(ctx, []string{"media_cov", "unknown-media"}))
	assert.Equal(t, "published", st.Snapshot().EducationMedia[len(st.Snapshot().EducationMedia)-1].Status)
	assert.NoError(t, repo.PublishEducationMedia(ctx, nil))

	emptyProgress, err := repo.GetEducationProgress(ctx, "usr_gading", "mod_cov", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{}, emptyProgress.CompletedSectionIDs)
	progress, err := repo.SaveEducationProgress(ctx, model.EducationProgress{UserID: "usr_gading", ModuleID: "mod_cov", Revision: 2, ProgressPercent: 40})
	require.NoError(t, err)
	assert.NotEmpty(t, progress.ID)
	progress.CompletedSectionIDs = []string{"intro"}
	progress.ProgressPercent = 100
	updatedProgress, err := repo.SaveEducationProgress(ctx, progress)
	require.NoError(t, err)
	assert.Equal(t, progress.ID, updatedProgress.ID)
	assert.Equal(t, 100, updatedProgress.ProgressPercent)

	require.NoError(t, repo.DeleteEducationModule(ctx, "mod_cov"))
	_, err = repo.GetEducationModuleByID(ctx, "mod_cov")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	assert.EqualError(t, repo.DeleteEducationModule(ctx, "mod_cov"), ErrEducationNotFound.Error())
	assert.NotContains(t, st.Snapshot().EducationProgress, progress)
}

func TestLearningRepository_InMemoryDocumentSafetyAndStateTransitions(t *testing.T) {
	now := time.Now().UTC()
	st := &store.Store{
		Users:            []model.User{{ID: "usr_learning", ExperiencePoints: 90}},
		LearningClusters: []model.LearningCluster{{ID: "cluster_cov", Slug: "cluster", Title: "Cluster", SortOrder: 2}},
		AcademicPrograms: []model.AcademicProgram{{ID: "program_cov", Slug: "program", Name: "Program", SortOrder: 1}},
		LearningItems: []model.LearningItem{
			{ID: "item_cov", Slug: "coverage-item", Kind: "course", Title: "Coverage item", Summary: "Summary"},
			{ID: "item_other", Slug: "other-item", Kind: "course", Title: "Other", Summary: "Other"},
		},
		LearningProgress: []model.LearningProgress{
			{UserID: "usr_learning", ItemID: "item_cov", State: "saved", CreatedAt: now, UpdatedAt: now},
			{UserID: "other-user", ItemID: "item_cov", State: "started"},
			{UserID: "usr_learning", ItemID: "", State: "started"},
		},
	}
	repo := New(nil, st)
	ctx := t.Context()

	document := map[string]any{"provider": "UTY", "url": "https://example.com/course", "unsafe": "ignored by parser", "language": []any{"id", 42, "en"}, "outcomes": []string{"one", "two"}, "duration_minutes": 15}
	revision := learningRevisionDocument(document, "coverage-item", "course", "Judul", "Title", "Ringkasan", "Summary")
	content, snapshot := unpackLearningRevisionDocument(revision)
	require.NotNil(t, snapshot)
	assert.Equal(t, document["provider"], content["provider"])
	assert.Equal(t, "fallback", snapshotString(snapshot, "missing", "fallback"))
	assert.Equal(t, document, cloneLearningDocument(document))
	assert.Nil(t, cloneLearningDocument(nil))
	assert.Equal(t, "https://example.com", safeLearningURL("https://example.com"))
	assert.Empty(t, safeLearningURL("http://example.com"))
	assert.Empty(t, safeLearningURL("javascript:alert(1)"))
	assert.Equal(t, "/v1/education/media/media_cov", learningMediaURL("media_cov", "https://fallback.example"))
	assert.Equal(t, "https://fallback.example", learningMediaURL("", "https://fallback.example"))
	assert.Empty(t, learningMediaURL("", "ftp://example.com"))
	assert.Equal(t, []string{"a"}, stringSlice([]any{"a", 1}))
	assert.Equal(t, []string{"a", "b"}, stringSlice([]string{"a", "b"}))
	assert.Nil(t, stringSlice(42))

	item, err := repo.GetLearningItemBySlug(ctx, "usr_learning", "coverage-item", "id")
	require.NoError(t, err)
	assert.Equal(t, "coverage-item", item.Slug)
	assert.NotNil(t, item.Progress)
	_, err = repo.GetLearningItemBySlug(ctx, "usr_learning", "missing", "id")
	assert.ErrorIs(t, err, ErrLearningItemNotFound)

	catalog, err := repo.GetLearningCatalog(ctx, "usr_learning", "en")
	require.NoError(t, err)
	assert.Len(t, catalog.Items, 2)
	assert.Len(t, catalog.Progress, 2)
	assert.Equal(t, 90, catalog.Experience.TotalEXP)
	progress, err := repo.GetLearningProgress(ctx, "usr_learning")
	require.NoError(t, err)
	assert.Len(t, progress, 1)

	_, err = repo.UpsertLearningState(ctx, "usr_learning", "item_cov", "invalid")
	assert.ErrorIs(t, err, ErrLearningStateInvalid)
	updated, err := repo.UpsertLearningState(ctx, "usr_learning", "item_cov", "started")
	require.NoError(t, err)
	assert.Equal(t, "started", updated.State)
	created, err := repo.UpsertLearningState(ctx, "usr_learning", "item_other", "saved")
	require.NoError(t, err)
	assert.Equal(t, "saved", created.State)
	for index := range st.LearningProgress {
		if st.LearningProgress[index].UserID == "usr_learning" && st.LearningProgress[index].ItemID == "item_other" {
			st.LearningProgress[index].State = "completed"
		}
	}
	completedPreserved, err := repo.UpsertLearningState(ctx, "usr_learning", "item_other", "started")
	require.NoError(t, err)
	assert.Equal(t, "completed", completedPreserved.State)

	_, _, _, err = repo.CompleteLearningProgress(ctx, "usr_learning", "item_other", "", "")
	assert.ErrorIs(t, err, ErrLearningCheckpointInvalid)
	_, _, _, err = repo.CompleteLearningProgress(ctx, "usr_learning", "missing", "reflection", "")
	assert.ErrorIs(t, err, ErrLearningItemNotFound)
	completed, granted, capReached, err := repo.CompleteLearningProgress(ctx, "usr_learning", "item_cov", "encrypted reflection", "encrypted outcome")
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.State)
	assert.True(t, granted)
	assert.False(t, capReached)
	assert.Equal(t, 100, repoUserExperience(t, repo, "usr_learning"))
	completedAgain, granted, capReached, err := repo.CompleteLearningProgress(ctx, "usr_learning", "item_cov", "new", "new")
	require.NoError(t, err)
	assert.Equal(t, completed.ItemID, completedAgain.ItemID)
	assert.False(t, granted)
	assert.False(t, capReached)

	experience, err := repo.GetLearningExperience(ctx, "missing-user")
	require.NoError(t, err)
	assert.Equal(t, 0, experience.TotalEXP)
}

func repoUserExperience(t *testing.T, repo *Repository, userID string) int {
	t.Helper()
	experience, err := repo.GetLearningExperience(t.Context(), userID)
	require.NoError(t, err)
	return experience.TotalEXP
}
