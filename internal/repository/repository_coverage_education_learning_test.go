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

func newCoverageEducationLearningRepo(t *testing.T) (*Repository, *store.Store) {
	t.Helper()
	backing := store.New()
	return New(nil, backing), backing
}

func coverageEduLearningDocument(titleID, titleEN string) model.EducationDocument {
	return model.EducationDocument{
		Audience:         "student",
		ExperienceType:   "article",
		Category:         "self-regulation",
		EstimatedMinutes: 12,
		Translations: map[string]model.EducationTranslation{
			"id": {Title: titleID, Summary: "Ringkasan " + titleID},
			"en": {Title: titleEN, Summary: "Summary " + titleEN},
		},
		Sections: []model.EducationSection{{ID: "section-1", SortOrder: 1, Required: true}},
	}
}

func coverageEduLearningDraft(slug, titleID, titleEN string) model.LearningItemDraft {
	return model.LearningItemDraft{
		Slug:      slug,
		Kind:      "course",
		TitleID:   titleID,
		TitleEN:   titleEN,
		SummaryID: "Ringkasan " + titleID,
		SummaryEN: "Summary " + titleEN,
		Document: map[string]any{
			"provider":                "UTY Learning",
			"provider_description_id": "Materi kampus",
			"provider_description_en": "Campus material",
			"url":                     "https://example.com/course",
			"language":                []string{"id", "en"},
			"outcomes":                []string{"Satu langkah praktik"},
			"duration_minutes":        30,
			"clusters":                []string{"mindful"},
			"programs":                []string{"informatics"},
			"thumbnail_media_id":      "media-thumb",
		},
	}
}

func TestRepositoryCoverageEducationLifecycleMemory(t *testing.T) {
	ctx := context.Background()
	repo, backing := newCoverageEducationLearningRepo(t)
	initialDocument := coverageEduLearningDocument("Siklus dorongan", "Urge cycle")
	require.NoError(t, repo.CreateEducationModule(ctx, model.EducationModule{
		ID: "edu-coverage", Slug: "urge-cycle", Title: "Siklus dorongan", Summary: "Ringkasan",
		EstimatedMinutes: 10, Status: "published", DraftDocument: initialDocument, DraftRevision: 1,
		CreatedBy: "seed", UpdatedBy: "seed",
	}))

	modules, err := repo.GetEducationModules(ctx)
	require.NoError(t, err)
	require.Len(t, modules, 1)
	assert.Equal(t, "edu-coverage", modules[0].ID)

	adminPage, err := repo.GetAdminEducationModules(ctx, model.PaginationQuery{Status: "published", Query: "urge", Limit: 1})
	require.NoError(t, err)
	require.Len(t, adminPage.Items, 1)
	assert.Equal(t, 1, adminPage.TotalCount)
	filteredPage, err := repo.GetAdminEducationModules(ctx, model.PaginationQuery{Query: "does-not-match", Page: 2, Limit: 1})
	require.NoError(t, err)
	assert.Empty(t, filteredPage.Items)
	assert.Equal(t, 0, filteredPage.TotalCount)

	module, err := repo.GetEducationModuleByID(ctx, "edu-coverage")
	require.NoError(t, err)
	assert.Equal(t, "urge-cycle", module.Slug)
	module, err = repo.GetEducationModuleBySlug(ctx, "URGE-CYCLE")
	require.NoError(t, err)
	assert.Equal(t, "edu-coverage", module.ID)
	_, err = repo.GetEducationModuleByID(ctx, "missing-education")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	_, err = repo.GetEducationModuleBySlug(ctx, "missing-education")
	assert.ErrorIs(t, err, ErrEducationNotFound)

	updatedDocument := coverageEduLearningDocument("Pilihan sadar", "Mindful choice")
	_, err = repo.UpdateEducationDraft(ctx, "edu-coverage", 99, "mindful-choice", updatedDocument, "editor")
	assert.ErrorIs(t, err, ErrEducationConflict)
	updated, err := repo.UpdateEducationDraft(ctx, "edu-coverage", 1, "mindful-choice", updatedDocument, "editor")
	require.NoError(t, err)
	assert.Equal(t, 2, updated.DraftRevision)
	assert.Equal(t, "draft", updated.Status)
	assert.Equal(t, "Pilihan sadar", updated.Title)

	published, err := repo.SetEducationStatus(ctx, "edu-coverage", "published", "reviewer", true)
	require.NoError(t, err)
	require.NotNil(t, published.PublishedDocument)
	assert.Equal(t, 2, published.PublishedRevision)
	assert.Equal(t, "reviewer", published.UpdatedBy)

	publishedModules, err := repo.GetPublishedEducationModules(ctx)
	require.NoError(t, err)
	require.Len(t, publishedModules, 1)
	assert.Equal(t, 2, publishedModules[0].PublishedRevision)

	archived, err := repo.SetEducationStatus(ctx, "edu-coverage", "archived", "reviewer", false)
	require.NoError(t, err)
	assert.NotNil(t, archived.ArchivedAt)
	publishedModules, err = repo.GetPublishedEducationModules(ctx)
	require.NoError(t, err)
	assert.Empty(t, publishedModules)

	media := model.EducationMedia{ID: "media-coverage", Kind: "upload", Purpose: "thumbnail", MediaType: "image", MIMEType: "image/webp", StorageKey: "media-coverage.webp", Status: "draft", CreatedBy: "editor"}
	require.NoError(t, repo.CreateEducationMedia(ctx, media))
	gotMedia, err := repo.GetEducationMedia(ctx, media.ID)
	require.NoError(t, err)
	assert.Equal(t, media.StorageKey, gotMedia.StorageKey)
	_, err = repo.GetEducationMedia(ctx, "missing-media")
	assert.ErrorIs(t, err, ErrEducationNotFound)
	require.NoError(t, repo.PublishEducationMedia(ctx, nil))
	require.NoError(t, repo.PublishEducationMedia(ctx, []string{media.ID, "not-present"}))
	gotMedia, err = repo.GetEducationMedia(ctx, media.ID)
	require.NoError(t, err)
	assert.Equal(t, "published", gotMedia.Status)

	progress, err := repo.GetEducationProgress(ctx, "student-coverage", "edu-coverage", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{}, progress.CompletedSectionIDs)
	progress, err = repo.SaveEducationProgress(ctx, model.EducationProgress{
		UserID: "student-coverage", ModuleID: "edu-coverage", Revision: 2, ProgressPercent: 40,
	})
	require.NoError(t, err)
	require.NotEmpty(t, progress.ID)
	createdAt := progress.CreatedAt
	progress, err = repo.SaveEducationProgress(ctx, model.EducationProgress{
		ID: "ignored-on-update", UserID: "student-coverage", ModuleID: "edu-coverage", Revision: 2,
		CompletedSectionIDs: []string{"section-1"}, OpenedMediaIDs: []string{"media-coverage"},
		CorrectCheckIDs: []string{"check-1"}, ProgressPercent: 100,
	})
	require.NoError(t, err)
	assert.NotEqual(t, "ignored-on-update", progress.ID)
	assert.Equal(t, createdAt, progress.CreatedAt)
	assert.Equal(t, 100, progress.ProgressPercent)
	progress, err = repo.GetEducationProgress(ctx, "student-coverage", "edu-coverage", 2)
	require.NoError(t, err)
	assert.Equal(t, []string{"section-1"}, progress.CompletedSectionIDs)

	newer := time.Now().UTC()
	require.NoError(t, repo.SaveEducationRevision(ctx, model.EducationRevision{ID: "edu-rev-1", ModuleID: "edu-coverage", Revision: 1, Kind: "draft", Document: initialDocument, CreatedBy: "editor", CreatedAt: newer.Add(-time.Minute)}))
	require.NoError(t, repo.SaveEducationRevision(ctx, model.EducationRevision{ID: "edu-rev-2", ModuleID: "edu-coverage", Revision: 2, Kind: "published", Document: updatedDocument, CreatedBy: "reviewer", CreatedAt: newer}))
	require.NoError(t, repo.SaveEducationRevision(ctx, model.EducationRevision{ID: "duplicate", ModuleID: "edu-coverage", Revision: 2, Kind: "published", CreatedAt: newer.Add(time.Minute)}))
	revisions, err := repo.ListEducationRevisions(ctx, "edu-coverage")
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	assert.Equal(t, "edu-rev-2", revisions[0].ID)
	foundRevision, err := repo.EducationRevisionByID(ctx, "edu-coverage", "edu-rev-1")
	require.NoError(t, err)
	assert.Equal(t, 1, foundRevision.Revision)
	_, err = repo.EducationRevisionByID(ctx, "wrong-module", "edu-rev-1")
	assert.ErrorIs(t, err, ErrEducationNotFound)

	backing.EducationProgress = append(backing.EducationProgress, model.EducationProgress{UserID: "other", ModuleID: "edu-coverage", Revision: 2})
	backing.EducationRevisions = append(backing.EducationRevisions, model.EducationRevision{ID: "other-revision", ModuleID: "other-module"})
	require.NoError(t, repo.DeleteEducationModule(ctx, "edu-coverage"))
	assert.Empty(t, backing.Modules)
	assert.Empty(t, backing.EducationProgress)
	assert.Len(t, backing.EducationRevisions, 1)
	assert.Equal(t, "other-module", backing.EducationRevisions[0].ModuleID)
	assert.ErrorIs(t, repo.DeleteEducationModule(ctx, "edu-coverage"), ErrEducationNotFound)
}

func TestRepositoryCoverageLearningCatalogProgressMemory(t *testing.T) {
	ctx := context.Background()
	repo, backing := newCoverageEducationLearningRepo(t)
	backing.Users = []model.User{{ID: "learning-student", ExperiencePoints: 20}}
	backing.LearningClusters = []model.LearningCluster{
		{ID: "cluster-2", Slug: "skills", Title: "Keterampilan", SortOrder: 2},
		{ID: "cluster-1", Slug: "mindful", Title: "Kesadaran", SortOrder: 1},
	}
	backing.AcademicPrograms = []model.AcademicProgram{{ID: "program-1", Slug: "informatics", Name: "Informatika"}}
	backing.LearningItems = []model.LearningItem{
		{ID: "learning-1", Slug: "pause-technique", Kind: "course", Title: "Teknik jeda", Summary: "Jeda singkat"},
		{ID: "learning-2", Slug: "money-map", Kind: "article", Title: "Peta uang", Summary: "Keuangan"},
	}
	backing.LearningProgress = []model.LearningProgress{
		{UserID: "learning-student", ItemID: "learning-1", State: "started"},
		{UserID: "learning-student", ItemID: "", State: "started"},
		{UserID: "another-student", ItemID: "learning-2", State: "completed"},
	}

	catalog, err := repo.GetLearningCatalog(ctx, "learning-student", "en")
	require.NoError(t, err)
	assert.Len(t, catalog.Clusters, 2)
	assert.Len(t, catalog.Programs, 1)
	assert.Len(t, catalog.Items, 2)
	require.Len(t, catalog.Progress, 2)
	assert.Equal(t, "learning-1", catalog.Items[0].Progress.ItemID)
	assert.Equal(t, 20, catalog.Experience.TotalEXP)
	assert.Equal(t, 1, catalog.Experience.Level)

	item, err := repo.GetLearningItemBySlug(ctx, "learning-student", "pause-technique", "id")
	require.NoError(t, err)
	require.NotNil(t, item.Progress)
	assert.Equal(t, "started", item.Progress.State)
	_, err = repo.GetLearningItemBySlug(ctx, "learning-student", "missing-learning-item", "id")
	assert.ErrorIs(t, err, ErrLearningItemNotFound)

	progressRows, err := repo.GetLearningProgress(ctx, "learning-student")
	require.NoError(t, err)
	require.Len(t, progressRows, 1)
	assert.Equal(t, "learning-1", progressRows[0].ItemID)

	_, err = repo.UpsertLearningState(ctx, "learning-student", "learning-1", "invalid")
	assert.ErrorIs(t, err, ErrLearningStateInvalid)
	created, err := repo.UpsertLearningState(ctx, "learning-student", "learning-2", "saved")
	require.NoError(t, err)
	assert.Equal(t, "saved", created.State)
	updated, err := repo.UpsertLearningState(ctx, "learning-student", "learning-2", "started")
	require.NoError(t, err)
	assert.Equal(t, "started", updated.State)
	backing.LearningProgress = append(backing.LearningProgress, model.LearningProgress{UserID: "learning-student", ItemID: "completed-item", State: "completed"})
	completedState, err := repo.UpsertLearningState(ctx, "learning-student", "completed-item", "saved")
	require.NoError(t, err)
	assert.Equal(t, "completed", completedState.State)

	_, granted, capReached, err := repo.CompleteLearningProgress(ctx, "learning-student", "learning-1", "reflection", "outcome")
	require.NoError(t, err)
	assert.True(t, granted)
	assert.False(t, capReached)
	experience, err := repo.GetLearningExperience(ctx, "learning-student")
	require.NoError(t, err)
	assert.Equal(t, 30, experience.TotalEXP)
	_, granted, capReached, err = repo.CompleteLearningProgress(ctx, "learning-student", "learning-1", "new reflection", "new outcome")
	require.NoError(t, err)
	assert.False(t, granted)
	assert.False(t, capReached)

	_, _, _, err = repo.CompleteLearningProgress(ctx, "learning-student", "learning-2", "", "")
	assert.ErrorIs(t, err, ErrLearningCheckpointInvalid)
	_, _, _, err = repo.CompleteLearningProgress(ctx, "learning-student", "missing-item", "reflection", "outcome")
	assert.ErrorIs(t, err, ErrLearningItemNotFound)

	today := time.Now().In(time.FixedZone("Asia/Jakarta", 7*60*60)).Format("2006-01-02")
	for index := 0; index < 5; index++ {
		backing.ExperienceGrants = append(backing.ExperienceGrants, model.ExperienceGrant{ID: "cap-" + string(rune('a'+index)), UserID: "learning-student", SourceKind: "other", SourceID: "cap-" + string(rune('a'+index)), GrantDate: today, Amount: 10})
	}
	_, granted, capReached, err = repo.CompleteLearningProgress(ctx, "learning-student", "learning-2", "reflection", "outcome")
	require.NoError(t, err)
	assert.False(t, granted)
	assert.True(t, capReached)

	unknownExperience, err := repo.GetLearningExperience(ctx, "unknown-student")
	require.NoError(t, err)
	assert.Equal(t, 0, unknownExperience.TotalEXP)
}

func TestRepositoryCoverageLearningAdminRevisionAndTaxonomyMemory(t *testing.T) {
	ctx := context.Background()
	repo, backing := newCoverageEducationLearningRepo(t)
	draft := coverageEduLearningDraft("mindful-course", "Kursus sadar", "Mindful course")
	created, err := repo.CreateAdminLearningItem(ctx, "admin-coverage", draft)
	require.NoError(t, err)
	assert.Equal(t, "draft", created.Status)
	assert.Equal(t, 1, created.DraftRevision)

	byID, err := repo.GetAdminLearningItem(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, draft.TitleID, byID.TitleID)
	bySlug, err := repo.GetAdminLearningItem(ctx, draft.Slug)
	require.NoError(t, err)
	assert.Equal(t, created.ID, bySlug.ID)
	_, err = repo.GetAdminLearningItem(ctx, "missing-admin-item")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)

	page, err := repo.ListAdminLearningItemsPaginated(ctx, model.PaginationQuery{Status: "draft", Query: "mindful", Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, 1, page.TotalCount)
	allDrafts, err := repo.ListAdminLearningItems(ctx, "draft")
	require.NoError(t, err)
	require.Len(t, allDrafts, 1)

	updatedDraft := coverageEduLearningDraft("mindful-course-v2", "Kursus sadar dua", "Mindful course two")
	_, err = repo.UpdateAdminLearningItem(ctx, "editor-coverage", created.ID, 99, updatedDraft)
	assert.ErrorIs(t, err, ErrLearningAdminConflict)
	updated, err := repo.UpdateAdminLearningItem(ctx, "editor-coverage", created.ID, 1, updatedDraft)
	require.NoError(t, err)
	assert.Equal(t, 2, updated.DraftRevision)
	assert.Equal(t, updatedDraft.Slug, updated.Slug)

	published, err := repo.SetAdminLearningStatus(ctx, "reviewer-coverage", created.ID, "published")
	require.NoError(t, err)
	assert.Equal(t, 2, published.PublishedRevision)
	assert.NotNil(t, published.PublishedAt)
	require.Len(t, backing.LearningItems, 1)

	revisions, err := repo.ListLearningRevisions(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 3)
	assert.Equal(t, "published", revisions[0].Kind)
	rollbackSource := revisions[len(revisions)-1].ID

	_, err = repo.UpdateAdminLearningItem(ctx, "editor-coverage", created.ID, 2, coverageEduLearningDraft("latest-course", "Versi terakhir", "Latest version"))
	require.NoError(t, err)
	rolledBack, err := repo.RollbackLearningItem(ctx, "rollback-actor", created.ID, rollbackSource)
	require.NoError(t, err)
	assert.Equal(t, "draft", rolledBack.Status)
	assert.Equal(t, "mindful-course", rolledBack.Slug)
	assert.Equal(t, "rollback-actor", rolledBack.UpdatedBy)

	archived, err := repo.SetAdminLearningStatus(ctx, "reviewer-coverage", created.ID, "archived")
	require.NoError(t, err)
	assert.NotNil(t, archived.ArchivedAt)
	assert.Empty(t, backing.LearningItems)
	require.NoError(t, repo.DeleteAdminLearningItem(ctx, created.ID))
	assert.ErrorIs(t, repo.DeleteAdminLearningItem(ctx, created.ID), ErrLearningAdminNotFound)
	_, err = repo.RollbackLearningItem(ctx, "actor", "missing-item", rollbackSource)
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)

	fallbackStore := store.New()
	fallbackStore.LearningItems = []model.LearningItem{{ID: "published-memory", Slug: "published-memory", Title: "Published", Summary: "Summary"}}
	fallbackRepo := New(nil, fallbackStore)
	fallbackRows, err := fallbackRepo.ListAdminLearningItems(ctx, "")
	require.NoError(t, err)
	require.Len(t, fallbackRows, 1)
	fallbackPage, err := fallbackRepo.ListAdminLearningItemsPaginated(ctx, model.PaginationQuery{Query: "published"})
	require.NoError(t, err)
	assert.Len(t, fallbackPage.Items, 1)

	taxonomyRepo, taxonomyStore := newCoverageEducationLearningRepo(t)
	taxonomy, err := taxonomyRepo.GetLearningHubTaxonomy(ctx)
	require.NoError(t, err)
	assert.Equal(t, "uty", taxonomy.Institution.Slug)
	assert.Empty(t, taxonomy.Clusters)

	cluster, err := taxonomyRepo.CreateLearningCluster(ctx, model.LearningClusterInput{Slug: "mindful", TitleID: "Kesadaran", TitleEN: "Mindfulness", DescriptionID: "Deskripsi", DescriptionEN: "Description", SortOrder: 1})
	require.NoError(t, err)
	assert.True(t, cluster.Active)
	_, err = taxonomyRepo.CreateLearningCluster(ctx, model.LearningClusterInput{Slug: "mindful", TitleID: "Duplikat"})
	assert.ErrorIs(t, err, ErrLearningAdminConflict)

	program, err := taxonomyRepo.CreateLearningProgram(ctx, model.AcademicProgramInput{Slug: "informatics", NameID: "Informatika", NameEN: "Informatics", Degree: "S1", PrimaryClusterSlug: "mindful", SortOrder: 1})
	require.NoError(t, err)
	assert.Equal(t, "inst_uty", program.InstitutionID)
	_, err = taxonomyRepo.CreateLearningProgram(ctx, model.AcademicProgramInput{Slug: "informatics", Name: "Duplikat"})
	assert.ErrorIs(t, err, ErrLearningAdminConflict)

	_, err = taxonomyRepo.UpdateLearningCluster(ctx, cluster.ID, model.LearningClusterInput{Slug: "mindful-renamed", TitleID: "Baru"})
	assert.ErrorIs(t, err, ErrLearningTaxonomyConflict)
	unusedCluster, err := taxonomyRepo.CreateLearningCluster(ctx, model.LearningClusterInput{Slug: "unused", TitleID: "Tidak dipakai", Active: func() *bool { value := false; return &value }()})
	require.NoError(t, err)
	updatedCluster, err := taxonomyRepo.UpdateLearningCluster(ctx, unusedCluster.ID, model.LearningClusterInput{Slug: "unused-renamed", TitleID: "Dipakai nanti", Active: func() *bool { value := true; return &value }()})
	require.NoError(t, err)
	assert.Equal(t, "unused-renamed", updatedCluster.Slug)
	assert.Len(t, taxonomyStore.LearningClusters, 2)

	assert.ErrorIs(t, taxonomyRepo.HardDeleteLearningCluster(ctx, cluster.ID), ErrLearningTaxonomyConflict)
	require.NoError(t, taxonomyRepo.HardDeleteLearningCluster(ctx, unusedCluster.ID))
	assert.ErrorIs(t, taxonomyRepo.HardDeleteLearningCluster(ctx, unusedCluster.ID), ErrLearningAdminNotFound)

	updatedProgram, err := taxonomyRepo.UpdateLearningProgram(ctx, program.ID, model.AcademicProgramInput{Slug: "informatics-v2", Name: "Informatika Baru", NameEN: "New Informatics", Degree: "S1", PrimaryClusterSlug: "mindful", SortOrder: 2})
	require.NoError(t, err)
	assert.Equal(t, "informatics-v2", updatedProgram.Slug)
	_, err = taxonomyRepo.UpdateLearningProgram(ctx, "missing-program", model.AcademicProgramInput{Slug: "missing"})
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	require.NoError(t, taxonomyRepo.DeactivateLearningProgram(ctx, program.ID))
	assert.Empty(t, taxonomyStore.AcademicPrograms)
	_, err = taxonomyRepo.UpdateLearningCluster(ctx, cluster.ID, model.LearningClusterInput{Slug: "mindful-renamed", TitleID: "Renamed", Active: func() *bool { value := true; return &value }()})
	require.NoError(t, err)

	gotCluster, err := taxonomyRepo.LearningClusterBySlug(ctx, "mindful-renamed")
	require.NoError(t, err)
	assert.Equal(t, cluster.ID, gotCluster.ID)
	gotProgram, err := taxonomyRepo.LearningProgramBySlug(ctx, "informatics-v2")
	require.NoError(t, err)
	assert.Equal(t, program.ID, gotProgram.ID)
	_, err = taxonomyRepo.LearningClusterBySlug(ctx, "missing-cluster")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)
	_, err = taxonomyRepo.LearningProgramBySlug(ctx, "missing-program")
	assert.ErrorIs(t, err, ErrLearningAdminNotFound)

	legacyTaxonomyStore := store.New()
	legacyTaxonomyStore.LearningClusters = []model.LearningCluster{{ID: "legacy-cluster", Slug: "legacy", Title: "Legacy"}}
	legacyTaxonomyStore.AcademicPrograms = []model.AcademicProgram{{ID: "legacy-program", Slug: "legacy-program", Name: "Legacy Program"}}
	legacyTaxonomyRepo := New(nil, legacyTaxonomyStore)
	legacyCluster, err := legacyTaxonomyRepo.LearningClusterBySlug(ctx, "legacy")
	require.NoError(t, err)
	assert.True(t, legacyCluster.Active)
	legacyProgram, err := legacyTaxonomyRepo.LearningProgramBySlug(ctx, "legacy-program")
	require.NoError(t, err)
	assert.Equal(t, "legacy-program", legacyProgram.Slug)
}
