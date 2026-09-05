package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func learningCoverageDraft(slug string) model.LearningItemDraft {
	return model.LearningItemDraft{
		Slug: slug, Kind: "course", TitleID: "Kursus", TitleEN: "Course",
		SummaryID: "Ringkasan", SummaryEN: "Summary",
		Document: map[string]any{
			"provider":                    "Open Learning",
			"provider_description_id":     "Penyedia pembelajaran",
			"provider_description_en":     "Learning provider",
			"url":                         "https://example.com/course",
			"duration_minutes":            30,
			"reviewer_name":               "Reviewer",
			"reviewed_at":                 "2026-09-01",
			"outcomes_id":                 []string{"Hasil"},
			"outcomes_en":                 []string{"Outcome"},
			"clusters":                    []string{"software-engineering"},
			"programs":                    []string{"informatics"},
			"provider_logo_media_id":      "",
			"thumbnail_media_id":          "",
			"language":                    []string{"id", "en"},
			"additional_unknown_metadata": true,
		},
	}
}

func TestLearningHubValidationAndProjectionHelpers(t *testing.T) {
	assert.Equal(t, "open-learning", learningProviderSlug(" Open / Learning "))
	assert.Equal(t, "", learningProviderSlug("---"))
	assert.Equal(t, []string{"one", "two"}, documentStrings([]string{"one", "two"}))
	assert.Equal(t, []string{"one"}, documentStrings([]any{"one", " ", 3}))
	assert.Equal(t, []string{"one"}, documentStrings("one"))
	assert.Empty(t, documentStrings(nil))
	assert.Equal(t, 4, documentInt(int64(4)))
	assert.Equal(t, 4, documentInt(float64(4)))
	assert.Equal(t, 4, documentInt("4"))
	assert.Equal(t, 0, documentInt(true))
	require.NoError(t, validateTaxonomySlug("software-engineering"))
	require.ErrorIs(t, validateTaxonomySlug("bad slug"), ErrLearningHubAdminInvalid)

	draft := learningCoverageDraft("valid-course")
	require.NoError(t, validateLearningDraft(draft, false))
	require.NoError(t, validateLearningDraft(draft, true))
	invalid := draft
	invalid.Slug = "bad slug"
	require.ErrorIs(t, validateLearningDraft(invalid, false), ErrLearningHubAdminInvalid)
	invalid = draft
	invalid.Document["url"] = "http://example.com"
	require.ErrorIs(t, validateLearningDraft(invalid, false), ErrLearningHubAdminInvalid)
	invalid = draft
	invalid.Document["duration_minutes"] = 0
	require.ErrorIs(t, validateLearningDraft(invalid, false), ErrLearningHubAdminInvalid)
	invalid = draft
	delete(invalid.Document, "reviewer_name")
	require.ErrorIs(t, validateLearningDraft(invalid, true), ErrLearningHubAdminInvalid)
}

func TestLearningHubCatalogStateCheckpointAndEditorialLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, st := newRepo(t)
	// NewSeeded has no Learning Hub editorial rows, so this test exercises the
	// same create/publish path used by the admin CMS on a clean store.
	svc := NewLearningHubService(repo, testCfg(), zap.NewNop())
	cluster, err := svc.CreateCluster(ctx, "usr_nasywa", model.LearningClusterInput{Slug: "software-engineering", TitleID: "Rekayasa Perangkat Lunak", TitleEN: "Software Engineering"})
	require.NoError(t, err)
	assert.True(t, cluster.Active)
	cluster, err = svc.UpdateCluster(ctx, "usr_nasywa", cluster.ID, model.LearningClusterInput{Slug: "software-engineering", TitleID: "RPL", TitleEN: "Software Engineering"})
	require.NoError(t, err)
	assert.Equal(t, "RPL", cluster.TitleID)
	program, err := svc.CreateProgram(ctx, "usr_nasywa", model.AcademicProgramInput{Slug: "informatics", NameID: "Informatika", Degree: "S1", PrimaryClusterSlug: "software-engineering"})
	require.NoError(t, err)
	assert.Equal(t, "Informatika", program.Name)

	draft := learningCoverageDraft("coverage-course")
	item, err := svc.CreateAdminItem(ctx, "usr_nasywa", draft)
	require.NoError(t, err)
	assert.Equal(t, "draft", item.Status)
	updated, err := svc.UpdateAdminItem(ctx, "usr_nasywa", item.ID, item.DraftRevision, draft)
	require.NoError(t, err)
	assert.Greater(t, updated.DraftRevision, item.DraftRevision)
	_, err = svc.UpdateAdminItem(ctx, "usr_nasywa", item.ID, item.DraftRevision, draft)
	require.ErrorIs(t, err, ErrLearningHubAdminConflict)
	inReview, err := svc.SubmitAdminItemReview(ctx, "usr_nasywa", item.ID)
	require.NoError(t, err)
	assert.Equal(t, "in_review", inReview.Status)
	published, err := svc.PublishAdminItem(ctx, "usr_nasywa", item.ID)
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)
	// The memory repository keeps the public projection lightweight; populate
	// the same provider fields that the DB projection derives from the document.
	st.Lock()
	for index := range st.LearningItems {
		if st.LearningItems[index].ID == item.ID {
			st.LearningItems[index].Provider = "Open Learning"
			st.LearningItems[index].ProviderDescription = "Learning provider"
		}
	}
	st.Unlock()

	items, err := svc.AdminItems(ctx, "published")
	require.NoError(t, err)
	assert.NotEmpty(t, items)
	page, err := svc.AdminItemsPaginated(ctx, model.PaginationQuery{Status: "published", Query: "coverage"})
	require.NoError(t, err)
	assert.NotEmpty(t, page.Items)
	_, err = svc.AdminItems(ctx, "invalid")
	require.ErrorIs(t, err, ErrLearningHubAdminInvalid)
	cat, err := svc.Catalog(ctx, "usr_gading", "en")
	require.NoError(t, err)
	assert.NotEmpty(t, cat.Items)
	providers, err := svc.Providers(ctx, "usr_gading", "en", model.PaginationQuery{Query: "open"})
	require.NoError(t, err)
	assert.NotEmpty(t, providers.Items)
	itemsByProvider, err := svc.ItemsByProvider(ctx, "usr_gading", "en", "open-learning", model.PaginationQuery{})
	require.NoError(t, err)
	assert.NotEmpty(t, itemsByProvider.Items)

	progress, err := svc.SaveState(ctx, "usr_gading", item.ID, " started ")
	require.NoError(t, err)
	assert.Equal(t, "started", progress.State)
	_, err = svc.SaveState(ctx, "usr_gading", item.ID, "completed")
	require.ErrorIs(t, err, ErrLearningHubStateInvalid)
	checkpoint, err := svc.Checkpoint(ctx, "usr_gading", item.ID, model.LearningCheckpointInput{Reflection: "learned", Outcome: "practice"})
	require.NoError(t, err)
	assert.True(t, checkpoint.EXPGranted)
	assert.NotEmpty(t, checkpoint.Progress.CompletedAt)
	_, err = svc.Checkpoint(ctx, "usr_gading", "missing", model.LearningCheckpointInput{Reflection: "x"})
	require.Error(t, err)
	_, err = svc.Checkpoint(ctx, "usr_gading", item.ID, model.LearningCheckpointInput{})
	require.ErrorIs(t, err, ErrLearningHubCheckpointInvalid)
	_, err = svc.Checkpoint(ctx, "usr_gading", item.ID, model.LearningCheckpointInput{Reflection: strings.Repeat("x", 2001)})
	require.ErrorIs(t, err, ErrLearningHubCheckpointInvalid)

	revisions, err := svc.AdminRevisions(ctx, item.ID)
	require.NoError(t, err)
	require.NotEmpty(t, revisions)
	_, err = svc.RollbackAdminItem(ctx, "usr_nasywa", item.ID, revisions[0].ID, "restore source revision")
	require.NoError(t, err)
	_, err = svc.RollbackAdminItem(ctx, "usr_nasywa", item.ID, revisions[0].ID, "")
	require.ErrorIs(t, err, ErrLearningHubAdminInvalid)

	err = svc.DeleteCluster(ctx, "usr_nasywa", cluster.ID)
	require.ErrorIs(t, err, repository.ErrLearningTaxonomyConflict)
	err = svc.DeleteProgram(ctx, "usr_nasywa", program.ID)
	require.NoError(t, err)
	err = svc.DeleteCluster(ctx, "usr_nasywa", cluster.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, cat.Experience)
}

func TestLearningHubAuthorizationAndNotFoundPaths(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewLearningHubService(repo, testCfg(), zap.NewNop())
	_, err := svc.CreateAdminItem(ctx, "usr_nasywa", model.LearningItemDraft{Slug: "", Kind: "course"})
	require.ErrorIs(t, err, ErrLearningHubAdminInvalid)
	_, err = svc.UpdateAdminItem(ctx, "usr_nasywa", "missing", 0, learningCoverageDraft("course"))
	require.ErrorIs(t, err, ErrLearningHubAdminInvalid)
	_, err = svc.UpdateAdminItem(ctx, "usr_nasywa", "missing", 1, learningCoverageDraft("course"))
	require.ErrorIs(t, err, ErrLearningHubAdminNotFound)
	_, err = svc.SubmitAdminItemReview(ctx, "usr_nasywa", "missing")
	require.ErrorIs(t, err, ErrLearningHubAdminNotFound)
	_, err = svc.PublishAdminItem(ctx, "usr_nasywa", "missing")
	require.ErrorIs(t, err, ErrLearningHubAdminNotFound)
	err = svc.DeleteAdminItem(ctx, "usr_nasywa", "missing")
	require.ErrorIs(t, err, repository.ErrLearningAdminNotFound)
	_, err = svc.CreateProgram(ctx, "usr_nasywa", model.AcademicProgramInput{Slug: "prog", Name: "Program", Degree: "S1", PrimaryClusterSlug: "missing"})
	require.ErrorIs(t, err, ErrLearningHubAdminInvalid)
	_, err = svc.UpdateCluster(ctx, "usr_nasywa", "missing", model.LearningClusterInput{Slug: "cluster", TitleID: "x", TitleEN: "x"})
	require.Error(t, err)
	_, err = svc.Taxonomy(ctx)
	require.NoError(t, err)
}

func TestSupportServiceThreadedWorkflowAndAuthorization(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewSupportServiceWithConfig(repo, testCfg(), zap.NewNop())
	_, err := svc.CreateThreadedSupportCase(ctx, "usr_nasywa", "title", "detail", "technical_support", "normal", "question")
	require.EqualError(t, err, "support requester role is not allowed")
	created, err := svc.CreateThreadedSupportCase(ctx, "usr_gading", " Setup issue ", " I need help ", "technical_support", "normal", "blocked")
	require.NoError(t, err)
	assert.Equal(t, "waiting_support", created.Status)
	assert.Empty(t, created.Messages)

	_, err = svc.GetSupportCaseDetail(ctx, "usr_dery", "user", created.ID)
	require.EqualError(t, err, "support case does not belong to actor")
	_, err = svc.GetSupportCaseDetail(ctx, "usr_nasywa", "admin", created.ID)
	require.EqualError(t, err, "support case must be claimed before opening the thread")
	claimed, err := svc.Claim(ctx, "usr_nasywa", created.ID, " investigate ")
	require.NoError(t, err)
	assert.Equal(t, "usr_nasywa", claimed.Owner)
	detail, err := svc.GetSupportCaseDetail(ctx, "usr_nasywa", "admin", created.ID)
	require.NoError(t, err)
	assert.Len(t, detail.Messages, 1)
	adminReply, err := svc.Reply(ctx, "usr_nasywa", "admin", created.ID, "We are checking")
	require.NoError(t, err)
	assert.Equal(t, "admin", adminReply.AuthorRole)
	require.NoError(t, svc.Transition(ctx, "usr_nasywa", "admin", created.ID, "resolved"))
	_, err = svc.Reply(ctx, "usr_nasywa", "admin", created.ID, "too late")
	require.EqualError(t, err, "closed or resolved support cases cannot receive replies")
	require.NoError(t, svc.Transition(ctx, "usr_gading", "user", created.ID, "waiting_support"))
	_, err = svc.Reply(ctx, "usr_gading", "user", created.ID, "Thank you")
	require.NoError(t, err)

	_, err = svc.Claim(ctx, "usr_nasywa", created.ID, "")
	require.EqualError(t, err, "claim reason is required")
	_, err = svc.GetSupportCaseDetail(ctx, "usr_nasywa", "admin", "missing")
	require.Error(t, err)
	all, err := svc.GetSupportCases(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, all)
	adminCases, err := svc.GetSupportCasesForAdmin(ctx, "usr_nasywa")
	require.NoError(t, err)
	assert.NotEmpty(t, adminCases)
	userCases, err := svc.GetSupportCasesForUser(ctx, "usr_gading")
	require.NoError(t, err)
	assert.NotEmpty(t, userCases)
	userPage, err := svc.GetSupportCasesForUserPaginated(ctx, "usr_gading", model.PaginationQuery{Bucket: "active", Query: "setup"})
	require.NoError(t, err)
	assert.NotNil(t, userPage.Items)
	adminPage, err := svc.GetSupportCasesForAdminPaginated(ctx, "usr_nasywa", model.PaginationQuery{Bucket: "history"})
	require.NoError(t, err)
	assert.NotNil(t, adminPage.Items)
}

func TestSupportServiceValidationAndDataRequestTransitions(t *testing.T) {
	ctx := context.Background()
	repo, st := newRepo(t)
	cfg := testCfg()
	cfg.NotificationMode = "demo"
	cfg.ExportStoragePath = t.TempDir()
	cfg.AvatarStoragePath = t.TempDir()
	svc := NewSupportServiceWithConfig(repo, cfg, zap.NewNop())
	err := svc.CreateSupportCase(ctx, "usr_nasywa", "title", "technical_support", "normal")
	require.Error(t, err)
	err = svc.CreateSupportCase(ctx, "usr_gading", "", "technical_support", "normal")
	require.EqualError(t, err, "invalid support case input")
	require.NoError(t, svc.CreateSupportCase(ctx, "usr_gading", "Question", "technical_support", "normal"))

	_, _, err = svc.CreateDataRequestWithResult(ctx, "usr_gading", "unsupported")
	require.ErrorIs(t, err, ErrDataRequestInvalid)
	_, _, err = svc.CreateDataRequestWithResult(ctx, "missing", "export")
	require.Error(t, err)
	_, _, err = svc.CreateDataRequestWithResult(ctx, "usr_nasywa", "delete")
	require.ErrorIs(t, err, ErrDataRequestForbidden)
	item, _, err := svc.CreateDataRequestWithResult(ctx, "usr_gading", "delete")
	require.NoError(t, err)
	assert.Equal(t, "pending_confirmation", item.Status)
	_, _, err = svc.CreateDataRequestWithResult(ctx, "usr_gading", "delete")
	require.ErrorIs(t, err, ErrDataRequestConflict)

	// Seeded demo journal/support rows are intentionally plaintext fixtures;
	// remove them for the encrypted export workflow under test.
	st.Lock()
	st.JournalEntries = nil
	st.RecoveryRecords = nil
	st.SupportMessages = nil
	st.Unlock()
	_, _, err = svc.CreateDataRequestWithResult(ctx, "usr_gading", "export")
	require.NoError(t, err)
	exports, err := svc.GetDataRequests(ctx, "usr_gading")
	require.NoError(t, err)
	assert.NotEmpty(t, exports)
	all, err := svc.GetAllDataRequests(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, all)
	page, err := svc.GetAllDataRequestsPaginated(ctx, model.PaginationQuery{PageSize: 10})
	require.NoError(t, err)
	assert.NotNil(t, page.Items)

	// A completed export with an expired result is made unavailable by the
	// background purge and can no longer be downloaded.
	expired := time.Now().UTC().Add(-time.Hour)
	st.Lock()
	st.DataRequests = append(st.DataRequests, model.DataRequest{ID: "DR-expired", UserID: "usr_gading", Type: "export", Status: "completed", ResultPath: cfg.ExportStoragePath + "/DR-expired.zip.enc", ResultExpiresAt: &expired})
	st.Unlock()
	_, err = svc.DataExportFile(ctx, "usr_gading", "DR-expired")
	require.EqualError(t, err, "data export is unavailable")

	_, err = svc.ProcessDataExport(ctx, "missing")
	require.EqualError(t, err, "data export cannot be processed")
	_, err = svc.RetryDataRequest(ctx, "missing")
	require.EqualError(t, err, "data request cannot be retried")
	_, err = svc.RejectDataRequest(ctx, "missing", "reason")
	require.EqualError(t, err, "data request cannot be rejected")
	assert.Equal(t, "Delete account and data", humanDataRequestTitleForService("delete"))
	assert.Equal(t, "Review data retention", humanDataRequestTitleForService("retention_review"))
}

func TestSupportServiceDecryptFallbackAndRequesterTransitions(t *testing.T) {
	ctx := context.Background()
	repo, st := newRepo(t)
	svc := NewSupportServiceWithConfig(repo, testCfg(), zap.NewNop())
	created, err := svc.CreateThreadedSupportCase(ctx, "usr_gading", "Help", "Details", "technical_support", "normal", "question")
	require.NoError(t, err)
	claimed, err := svc.Claim(ctx, "usr_nasywa", created.ID, "review")
	require.NoError(t, err)
	require.NoError(t, svc.Transition(ctx, "usr_nasywa", "admin", claimed.ID, "resolved"))
	// The requester can close a recently resolved case or reopen it for one
	// more support turn, but cannot use an unrelated transition.
	require.NoError(t, svc.Transition(ctx, "usr_gading", "user", created.ID, "closed"))
	err = svc.ReleaseClaim(ctx, "usr_nasywa", created.ID, "release")
	require.Error(t, err)

	st.Lock()
	st.SupportMessages = append(st.SupportMessages, model.SupportMessage{ID: "bad-cipher", SupportCaseID: created.ID, Content: strings.Repeat("a", 64)})
	st.Unlock()
	caseDetail, err := svc.GetSupportCaseDetail(ctx, "usr_gading", "user", created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, caseDetail.Messages)

	noKey := NewSupportServiceWithConfig(repo, configWithoutJournalKey(), zap.NewNop())
	_, err = noKey.Reply(ctx, "usr_gading", "user", created.ID, "message")
	require.Error(t, err)
}

func configWithoutJournalKey() (cfg config.Config) {
	cfg = testCfg()
	cfg.JournalEncryptionKey = ""
	return cfg
}
