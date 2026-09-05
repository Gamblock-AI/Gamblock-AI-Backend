package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func finalLearningHubEducationDocument(titleID, titleEN string) model.EducationDocument {
	return model.EducationDocument{
		Audience:         "student",
		ExperienceType:   "article",
		Category:         "self-regulation",
		EstimatedMinutes: 15,
		Translations: map[string]model.EducationTranslation{
			"id": {Title: titleID, Summary: "Ringkasan " + titleID},
			"en": {Title: titleEN, Summary: "Summary " + titleEN},
		},
		Sections: []model.EducationSection{{ID: "final-section", Required: true}},
	}
}

func finalLearningHubDraft(slug, titleID, titleEN string) model.LearningItemDraft {
	return model.LearningItemDraft{
		Slug:      slug,
		Kind:      "course",
		TitleID:   titleID,
		TitleEN:   titleEN,
		SummaryID: "Ringkasan " + titleID,
		SummaryEN: "Summary " + titleEN,
		Document: map[string]any{
			"provider":                "Final Learning Hub",
			"provider_description_id": "Materi teruji",
			"provider_description_en": "Tested material",
			"url":                     "https://example.com/final-learning",
			"language":                []string{"id", "en"},
			"outcomes":                []string{"Outcome 1"},
			"duration_minutes":        20,
			"thumbnail_media_id":      "final-thumbnail",
		},
	}
}

func finalLearningHubBool(value bool) *bool {
	return &value
}

func TestRepositoryFinalLearningHub_AdminItemsPaginationLifecycleAndRollback(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())

	first, err := repo.CreateAdminLearningItem(ctx, "final-admin", finalLearningHubDraft("final-first", "Materi pertama", "First material"))
	require.NoError(t, err)
	second, err := repo.CreateAdminLearningItem(ctx, "final-admin", finalLearningHubDraft("final-second", "Materi kedua", "Second material"))
	require.NoError(t, err)
	_, err = repo.CreateAdminLearningItem(ctx, "final-admin", finalLearningHubDraft("final-third", "Materi ketiga", "Third material"))
	require.NoError(t, err)

	page, err := repo.ListAdminLearningItemsPaginated(ctx, model.PaginationQuery{Status: "draft", Query: "KEDUA", Page: 1, Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, second.ID, page.Items[0].ID)
	assert.Equal(t, 1, page.TotalCount)
	assert.False(t, page.HasMore)

	emptyPage, err := repo.ListAdminLearningItemsPaginated(ctx, model.PaginationQuery{Status: "published", Page: 4, Limit: 2})
	require.NoError(t, err)
	assert.Empty(t, emptyPage.Items)
	assert.Zero(t, emptyPage.TotalCount)
	assert.Equal(t, 4, emptyPage.Page)

	bySlug, err := repo.GetAdminLearningItem(ctx, first.Slug)
	require.NoError(t, err)
	assert.Equal(t, first.ID, bySlug.ID)
	_, err = repo.GetAdminLearningItem(ctx, "final-missing-item")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)

	updatedDraft := finalLearningHubDraft("final-first-v2", "Materi pertama revisi", "First material revision")
	_, err = repo.UpdateAdminLearningItem(ctx, "final-editor", first.ID, 99, updatedDraft)
	assert.ErrorIs(t, err, ErrLearningAdminConflict)
	_, err = repo.UpdateAdminLearningItem(ctx, "final-editor", "final-missing-item", 1, updatedDraft)
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	updated, err := repo.UpdateAdminLearningItem(ctx, "final-editor", first.ID, 1, updatedDraft)
	require.NoError(t, err)
	assert.Equal(t, 2, updated.DraftRevision)
	assert.Equal(t, "final-first-v2", updated.Slug)

	published, err := repo.SetAdminLearningStatus(ctx, "final-reviewer", first.ID, "published")
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)
	assert.Equal(t, 2, published.PublishedRevision)
	assert.NotNil(t, published.PublishedAt)

	publishedPage, err := repo.ListAdminLearningItemsPaginated(ctx, model.PaginationQuery{Status: "published", Query: "revisi"})
	require.NoError(t, err)
	require.Len(t, publishedPage.Items, 1)
	assert.Equal(t, first.ID, publishedPage.Items[0].ID)

	latestDraft := finalLearningHubDraft("final-first-v3", "Materi paling baru", "Newest material")
	_, err = repo.UpdateAdminLearningItem(ctx, "final-editor", first.ID, 2, latestDraft)
	require.NoError(t, err)
	assert.Equal(t, "draft", mustFinalLearningHubAdminItem(t, repo, first.ID).Status)

	revisions, err := repo.ListLearningRevisions(ctx, first.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 4)
	var publishedRevision model.LearningRevision
	for _, revision := range revisions {
		if revision.Kind == "published" {
			publishedRevision = revision
			break
		}
	}
	require.NotEmpty(t, publishedRevision.ID)
	assert.Equal(t, "Final Learning Hub", publishedRevision.Document["provider"])

	rolledBack, err := repo.RollbackLearningItem(ctx, "final-rollback", first.ID, publishedRevision.ID)
	require.NoError(t, err)
	assert.Equal(t, "draft", rolledBack.Status)
	assert.Equal(t, "final-first-v2", rolledBack.Slug)
	assert.Equal(t, "Materi pertama revisi", rolledBack.TitleID)
	assert.Equal(t, "final-rollback", rolledBack.UpdatedBy)

	_, err = repo.RollbackLearningItem(ctx, "final-rollback", first.ID, "final-missing-revision")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	_, err = repo.ListLearningRevisions(ctx, "final-missing-item")
	assert.NoError(t, err)

	archived, err := repo.SetAdminLearningStatus(ctx, "final-reviewer", first.ID, "archived")
	require.NoError(t, err)
	assert.Equal(t, "archived", archived.Status)
	assert.NotNil(t, archived.ArchivedAt)

	all, err := repo.ListAdminLearningItems(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Empty(t, mustFinalLearningHubStore(t, repo).LearningItems)

	require.NoError(t, repo.DeleteAdminLearningItem(ctx, first.ID))
	assert.ErrorIs(t, repo.DeleteAdminLearningItem(ctx, first.ID), ErrLearningAdminNotFound)
}

func TestRepositoryFinalLearningHub_PublicCatalogProgressAndEmptyBranches(t *testing.T) {
	ctx := context.Background()
	backing := store.New()
	backing.Users = []model.User{{ID: "final-student", ExperiencePoints: 40}}
	backing.LearningClusters = []model.LearningCluster{{ID: "final-cluster", Slug: "final-cluster", Title: "Klaster"}}
	backing.AcademicPrograms = []model.AcademicProgram{{ID: "final-program", Slug: "final-program", Name: "Program"}}
	backing.LearningItems = []model.LearningItem{
		{ID: "final-learning-1", Slug: "final-learning-1", Kind: "course", Title: "Course 1", Summary: "Summary 1"},
		{ID: "final-learning-2", Slug: "final-learning-2", Kind: "article", Title: "Course 2", Summary: "Summary 2"},
	}
	backing.LearningProgress = []model.LearningProgress{
		{UserID: "final-student", ItemID: "final-learning-1", State: "started"},
		{UserID: "final-student", ItemID: "final-learning-ignored", State: ""},
		{UserID: "other-student", ItemID: "final-learning-2", State: "completed"},
	}
	repo := New(nil, backing)

	catalog, err := repo.GetLearningCatalog(ctx, "final-student", "id")
	require.NoError(t, err)
	assert.Len(t, catalog.Clusters, 1)
	assert.Len(t, catalog.Programs, 1)
	assert.Len(t, catalog.Items, 2)
	assert.Len(t, catalog.Progress, 2)
	assert.Equal(t, 40, catalog.Experience.TotalEXP)
	assert.Equal(t, 1, catalog.Experience.Level)

	item, err := repo.GetLearningItemBySlug(ctx, "final-student", "final-learning-1", "id")
	require.NoError(t, err)
	require.NotNil(t, item.Progress)
	assert.Equal(t, "started", item.Progress.State)
	_, err = repo.GetLearningItemBySlug(ctx, "final-student", "final-missing-learning", "id")
	assert.ErrorIs(t, err, ErrLearningItemNotFound)

	progress, err := repo.GetLearningProgress(ctx, "final-student")
	require.NoError(t, err)
	require.Len(t, progress, 1)
	assert.Equal(t, "final-learning-1", progress[0].ItemID)

	_, err = repo.UpsertLearningState(ctx, "final-student", "final-learning-1", "invalid")
	assert.ErrorIs(t, err, ErrLearningStateInvalid)
	created, err := repo.UpsertLearningState(ctx, "final-student", "final-learning-2", "saved")
	require.NoError(t, err)
	assert.Equal(t, "saved", created.State)
	updated, err := repo.UpsertLearningState(ctx, "final-student", "final-learning-2", "started")
	require.NoError(t, err)
	assert.Equal(t, "started", updated.State)
	backing.LearningProgress = append(backing.LearningProgress, model.LearningProgress{UserID: "final-student", ItemID: "final-learning-completed", State: "completed"})
	completed, err := repo.UpsertLearningState(ctx, "final-student", "final-learning-completed", "saved")
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.State)

	_, _, _, err = repo.CompleteLearningProgress(ctx, "final-student", "final-learning-1", "", "")
	assert.ErrorIs(t, err, ErrLearningCheckpointInvalid)
	_, _, _, err = repo.CompleteLearningProgress(ctx, "final-student", "final-missing-learning", "reflection", "outcome")
	assert.ErrorIs(t, err, ErrLearningItemNotFound)
	completed, granted, capReached, err := repo.CompleteLearningProgress(ctx, "final-student", "final-learning-2", "reflection", "outcome")
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.State)
	assert.True(t, granted)
	assert.False(t, capReached)
	_, granted, capReached, err = repo.CompleteLearningProgress(ctx, "final-student", "final-learning-2", "new reflection", "new outcome")
	require.NoError(t, err)
	assert.False(t, granted)
	assert.False(t, capReached)

	experience, err := repo.GetLearningExperience(ctx, "final-student")
	require.NoError(t, err)
	assert.Equal(t, 50, experience.TotalEXP)
	unknown, err := repo.GetLearningExperience(ctx, "final-unknown-student")
	require.NoError(t, err)
	assert.Zero(t, unknown.TotalEXP)

	emptyRepo := New(nil, store.New())
	emptyCatalog, err := emptyRepo.GetLearningCatalog(ctx, "final-empty-student", "en")
	require.NoError(t, err)
	assert.Empty(t, emptyCatalog.Items)
	assert.Empty(t, emptyCatalog.Progress)
	assert.ErrorIs(t, func() error {
		_, err := emptyRepo.GetLearningItemBySlug(ctx, "final-empty-student", "missing", "en")
		return err
	}(), ErrLearningItemNotFound)
}

func TestRepositoryFinalLearningHub_TaxonomyConflictsAndEmptyBranches(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())

	taxonomy, err := repo.GetLearningHubTaxonomy(ctx)
	require.NoError(t, err)
	assert.Equal(t, "uty", taxonomy.Institution.Slug)
	assert.Empty(t, taxonomy.Clusters)
	assert.Empty(t, taxonomy.Programs)
	_, err = repo.LearningClusterBySlug(ctx, "final-missing-cluster")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	_, err = repo.LearningProgramBySlug(ctx, "final-missing-program")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)

	activeCluster, err := repo.CreateLearningCluster(ctx, model.LearningClusterInput{Slug: "final-active", TitleID: "Aktif", TitleEN: "Active", DescriptionID: "Deskripsi", SortOrder: 1})
	require.NoError(t, err)
	assert.True(t, activeCluster.Active)
	_, err = repo.CreateLearningCluster(ctx, model.LearningClusterInput{Slug: "final-active", TitleID: "Duplikat"})
	assert.ErrorIs(t, err, ErrLearningAdminConflict)

	program, err := repo.CreateLearningProgram(ctx, model.AcademicProgramInput{Slug: "final-program", NameID: "Informatika", NameEN: "Informatics", Degree: "S1", PrimaryClusterSlug: "final-active", SortOrder: 1})
	require.NoError(t, err)
	assert.Equal(t, "Informatika", program.Name)
	assert.Equal(t, "Informatics", program.NameEN)
	_, err = repo.CreateLearningProgram(ctx, model.AcademicProgramInput{Slug: "final-program", Name: "Duplikat"})
	assert.ErrorIs(t, err, ErrLearningAdminConflict)

	_, err = repo.UpdateLearningCluster(ctx, activeCluster.ID, model.LearningClusterInput{Slug: "final-renamed", TitleID: "Baru"})
	assert.ErrorIs(t, err, ErrLearningTaxonomyConflict)
	_, err = repo.UpdateLearningCluster(ctx, "final-missing-cluster", model.LearningClusterInput{Slug: "missing"})
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	assert.True(t, mustFinalLearningHubClusterInUse(t, repo, "final-active"))
	assert.False(t, mustFinalLearningHubClusterInUse(t, repo, "final-unused"))

	inactiveCluster, err := repo.CreateLearningCluster(ctx, model.LearningClusterInput{Slug: "final-inactive", TitleID: "Tidak aktif", Active: finalLearningHubBool(false)})
	require.NoError(t, err)
	assert.False(t, inactiveCluster.Active)
	updatedCluster, err := repo.UpdateLearningCluster(ctx, inactiveCluster.ID, model.LearningClusterInput{Slug: "final-unused", TitleID: "Tidak terpakai", Active: finalLearningHubBool(true)})
	require.NoError(t, err)
	assert.True(t, updatedCluster.Active)
	assert.NoError(t, repo.HardDeleteLearningCluster(ctx, updatedCluster.ID))
	assert.ErrorIs(t, repo.HardDeleteLearningCluster(ctx, updatedCluster.ID), ErrLearningAdminNotFound)
	assert.ErrorIs(t, repo.HardDeleteLearningCluster(ctx, activeCluster.ID), ErrLearningTaxonomyConflict)

	inactiveProgram, err := repo.CreateLearningProgram(ctx, model.AcademicProgramInput{Slug: "final-inactive-program", Name: "Inactive", PrimaryClusterSlug: "final-active", Active: finalLearningHubBool(false)})
	require.NoError(t, err)
	assert.False(t, inactiveProgram.Active)
	updatedProgram, err := repo.UpdateLearningProgram(ctx, inactiveProgram.ID, model.AcademicProgramInput{Slug: "final-inactive-program-v2", Name: "Updated", NameEN: "Updated EN", PrimaryClusterSlug: "final-active", Active: finalLearningHubBool(true)})
	require.NoError(t, err)
	assert.True(t, updatedProgram.Active)
	_, err = repo.UpdateLearningProgram(ctx, inactiveProgram.ID, model.AcademicProgramInput{Slug: "final-program"})
	assert.ErrorIs(t, err, ErrLearningAdminConflict)
	_, err = repo.UpdateLearningProgram(ctx, "final-missing-program", model.AcademicProgramInput{Slug: "missing"})
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	assert.NoError(t, repo.DeactivateLearningProgram(ctx, inactiveProgram.ID))
	assert.Len(t, mustFinalLearningHubStore(t, repo).AcademicPrograms, 1)
	assert.NoError(t, repo.DeactivateLearningProgram(ctx, inactiveProgram.ID))
	assert.ErrorIs(t, repo.DeactivateLearningProgram(ctx, "final-missing-program"), ErrLearningAdminNotFound)

	gotCluster, err := repo.LearningClusterBySlug(ctx, "final-active")
	require.NoError(t, err)
	assert.Equal(t, activeCluster.ID, gotCluster.ID)
	gotProgram, err := repo.LearningProgramBySlug(ctx, "final-program")
	require.NoError(t, err)
	assert.Equal(t, program.ID, gotProgram.ID)
}

func TestRepositoryFinalLearningHub_EducationMediaAndRevisionBranches(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())

	modules, err := repo.GetEducationModules(ctx)
	require.NoError(t, err)
	assert.Empty(t, modules)
	published, err := repo.GetPublishedEducationModules(ctx)
	require.NoError(t, err)
	assert.Empty(t, published)
	adminPage, err := repo.GetAdminEducationModules(ctx, model.PaginationQuery{Status: "published", Query: "missing", Page: 2, Limit: 1})
	require.NoError(t, err)
	assert.Empty(t, adminPage.Items)
	_, err = repo.GetEducationModuleByID(ctx, "final-missing-education")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	_, err = repo.GetEducationModuleBySlug(ctx, "final-missing-education")
	assert.ErrorIs(t, err, ErrEducationNotFound)

	document := finalLearningHubEducationDocument("Kesadaran", "Awareness")
	require.NoError(t, repo.CreateEducationModule(ctx, model.EducationModule{ID: "final-education", Slug: "final-awareness", Title: "Kesadaran", Summary: "Ringkasan", Status: "draft", DraftDocument: document, DraftRevision: 1, CreatedBy: "final-editor", UpdatedBy: "final-editor"}))
	page, err := repo.GetAdminEducationModules(ctx, model.PaginationQuery{Status: "draft", Query: "AWARE", Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)

	_, err = repo.UpdateEducationDraft(ctx, "final-education", 99, "wrong", document, "final-editor")
	assert.ErrorIs(t, err, ErrEducationConflict)
	updatedDocument := finalLearningHubEducationDocument("Pilihan sadar", "Mindful choice")
	updated, err := repo.UpdateEducationDraft(ctx, "final-education", 1, "final-mindful-choice", updatedDocument, "final-editor")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.DraftRevision)
	assert.Equal(t, "draft", updated.Status)

	updated, err = repo.SetEducationStatus(ctx, "final-education", "published", "final-reviewer", true)
	require.NoError(t, err)
	assert.Equal(t, 2, updated.PublishedRevision)
	require.NotNil(t, updated.PublishedDocument)
	published, err = repo.GetPublishedEducationModules(ctx)
	require.NoError(t, err)
	assert.Len(t, published, 1)

	media := model.EducationMedia{ID: "final-media", Kind: "upload", Purpose: "thumbnail", MediaType: "image", MIMEType: "image/webp", StorageKey: "final.webp", SizeBytes: 42, Status: "draft", CreatedBy: "final-editor"}
	require.NoError(t, repo.CreateEducationMedia(ctx, media))
	gotMedia, err := repo.GetEducationMedia(ctx, media.ID)
	require.NoError(t, err)
	assert.Equal(t, "final.webp", gotMedia.StorageKey)
	_, err = repo.GetEducationMedia(ctx, "final-missing-media")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	require.NoError(t, repo.PublishEducationMedia(ctx, nil))
	require.NoError(t, repo.PublishEducationMedia(ctx, []string{media.ID, "final-not-present"}))
	gotMedia, err = repo.GetEducationMedia(ctx, media.ID)
	require.NoError(t, err)
	assert.Equal(t, "published", gotMedia.Status)

	progress, err := repo.GetEducationProgress(ctx, "final-student", "final-education", 2)
	require.NoError(t, err)
	assert.Empty(t, progress.CompletedSectionIDs)
	progress, err = repo.SaveEducationProgress(ctx, model.EducationProgress{UserID: "final-student", ModuleID: "final-education", Revision: 2, ProgressPercent: 25})
	require.NoError(t, err)
	createdAt := progress.CreatedAt
	progress, err = repo.SaveEducationProgress(ctx, model.EducationProgress{ID: "final-replaced", UserID: "final-student", ModuleID: "final-education", Revision: 2, CompletedSectionIDs: []string{"final-section"}, ProgressPercent: 100})
	require.NoError(t, err)
	assert.NotEqual(t, "final-replaced", progress.ID)
	assert.Equal(t, createdAt, progress.CreatedAt)
	assert.Equal(t, 100, progress.ProgressPercent)

	finalRevision := model.EducationRevision{ID: "final-revision-1", ModuleID: "final-education", Revision: 1, Document: document, Slug: "final-awareness", Kind: "draft", CreatedBy: "final-editor", CreatedAt: time.Now().UTC()}
	require.NoError(t, repo.SaveEducationRevision(ctx, finalRevision))
	require.NoError(t, repo.SaveEducationRevision(ctx, finalRevision))
	revisions, err := repo.ListEducationRevisions(ctx, "final-education")
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	assert.Equal(t, finalRevision.ID, revisions[0].ID)
	gotRevision, err := repo.EducationRevisionByID(ctx, "final-education", finalRevision.ID)
	require.NoError(t, err)
	assert.Equal(t, finalRevision.Slug, gotRevision.Slug)
	_, err = repo.EducationRevisionByID(ctx, "other-module", finalRevision.ID)
	assert.ErrorIs(t, err, ErrEducationNotFound)

	archived, err := repo.SetEducationStatus(ctx, "final-education", "archived", "final-reviewer", false)
	require.NoError(t, err)
	assert.NotNil(t, archived.ArchivedAt)
	published, err = repo.GetPublishedEducationModules(ctx)
	require.NoError(t, err)
	assert.Empty(t, published)

	assert.ErrorIs(t, repo.DeleteEducationModule(ctx, "final-missing-education"), ErrEducationNotFound)
	require.NoError(t, repo.DeleteEducationModule(ctx, "final-education"))
	assert.Empty(t, mustFinalLearningHubStore(t, repo).EducationRevisions)
	assert.Empty(t, mustFinalLearningHubStore(t, repo).EducationProgress)
}

func mustFinalLearningHubStore(t *testing.T, repo *Repository) *store.Store {
	t.Helper()
	if repo.store == nil {
		t.Fatal("expected an in-memory repository")
	}
	return repo.store
}

func mustFinalLearningHubAdminItem(t *testing.T, repo *Repository, id string) model.AdminLearningItem {
	t.Helper()
	item, err := repo.GetAdminLearningItem(context.Background(), id)
	require.NoError(t, err)
	return item
}

func mustFinalLearningHubClusterInUse(t *testing.T, repo *Repository, slug string) bool {
	t.Helper()
	inUse, err := repo.LearningClusterInUse(context.Background(), slug)
	require.NoError(t, err)
	return inUse
}
