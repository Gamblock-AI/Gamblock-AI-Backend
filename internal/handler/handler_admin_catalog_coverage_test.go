package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

type handlerAdminCatalogCoverageHarness struct {
	h  *Handler
	st *store.Store
}

func newHandlerAdminCatalogCoverageHarness(t *testing.T) handlerAdminCatalogCoverageHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv: "test", JWTAccessSecret: "test-secret-very-long-please-32bytes!",
		JWTAccessTTL: time.Hour, JWTRefreshTTL: 720 * time.Hour,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		NotificationMode:     "demo", PublicWebBaseURL: "http://localhost:3000",
		ExportStoragePath: t.TempDir(), MediaStoragePath: t.TempDir(),
		MediaEmbedHosts: []string{"www.youtube.com", "vimeo.com"},
	}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	return handlerAdminCatalogCoverageHarness{h: New(services, mid, cfg, zap.NewNop()), st: st}
}

func handlerAdminCatalogCoverageAdminValues() map[string]any {
	return map[string]any{"user_id": "usr_nasywa", "role": "admin"}
}

func handlerAdminCatalogCoverageFindUser(t *testing.T, st *store.Store, email string) model.User {
	t.Helper()
	st.RLock()
	defer st.RUnlock()
	for _, item := range st.Users {
		if item.Email == email {
			return item
		}
	}
	t.Fatalf("user %q not found", email)
	return model.User{}
}

func handlerAdminCatalogCoverageFindLearningItem(t *testing.T, st *store.Store, slug string) model.AdminLearningItem {
	t.Helper()
	st.RLock()
	defer st.RUnlock()
	for _, item := range st.AdminLearningItems {
		if item.Slug == slug {
			return item
		}
	}
	t.Fatalf("learning item %q not found", slug)
	return model.AdminLearningItem{}
}

func handlerAdminCatalogCoverageFindLearningRevision(t *testing.T, st *store.Store, itemID string) model.LearningRevision {
	t.Helper()
	st.RLock()
	defer st.RUnlock()
	for _, item := range st.LearningRevisions {
		if item.ItemID == itemID {
			return item
		}
	}
	t.Fatalf("learning revision for %q not found", itemID)
	return model.LearningRevision{}
}

func handlerAdminCatalogCoverageLearningDraft() model.LearningItemDraft {
	return model.LearningItemDraft{
		Slug: "coverage-course", Kind: "course", TitleID: "Kursus Coverage", TitleEN: "Coverage Course",
		SummaryID: "Ringkasan coverage", SummaryEN: "Coverage summary",
		Document: map[string]any{
			"provider":                "Open Learning",
			"provider_description_id": "Penyedia pembelajaran",
			"provider_description_en": "Learning provider",
			"url":                     "https://example.com/coverage-course",
			"duration_minutes":        30,
			"reviewer_name":           "Coverage Reviewer",
			"reviewed_at":             "2026-09-01",
			"outcomes_id":             []string{"Hasil belajar"},
			"outcomes_en":             []string{"Learning outcome"},
			"clusters":                []string{"coverage-cluster"},
			"programs":                []string{"coverage-program"},
			"language":                []string{"id", "en"},
		},
	}
}

func handlerAdminCatalogCoverageJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	return string(body)
}

func TestHandlerAdminCatalogCoverage_AdminOperationsAndSupport(t *testing.T) {
	harness := newHandlerAdminCatalogCoverageHarness(t)
	h, st := harness.h, harness.st
	admin := handlerAdminCatalogCoverageAdminValues()

	w := invokeHandler(t, h, http.MethodGet, "/v1/portal/overview", "", nil, admin, h.PortalOverview)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/modules", "", nil, admin, h.AdminModules)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/modules?page=1&limit=2&q=impulse", "", nil, admin, h.AdminModules)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	document := st.Modules[0].DraftDocument
	createModuleBody := handlerAdminCatalogCoverageJSON(t, map[string]any{
		"slug": "coverage-admin-module", "document": document,
	})
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules", createModuleBody, nil, admin, h.CreateAdminModule)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/support-cases", "", nil, admin, h.AdminSupportCases)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/support-cases?page=1&limit=1&bucket=active", "", nil, admin, h.AdminSupportCases)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = invokeHandler(t, h, http.MethodGet, "/v1/public/site-social-links", "", nil, nil, h.PublicSiteSocialLinks)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/site-social-links", "", nil, admin, h.AdminSiteSocialLinks)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/overview", "", nil, admin, h.AdminOverview)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/audit-events", "", nil, admin, h.AdminAuditEvents)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/audit-events?page=1&limit=2&q=education", "", nil, admin, h.AdminAuditEvents)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/accounts", "", nil, admin, h.AdminAccounts)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/accounts?page=1&limit=2&role=admin", "", nil, admin, h.AdminAccounts)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	githubURL := "https://github.com/gamblock-ai"
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/site-social-links", handlerAdminCatalogCoverageJSON(t, map[string]any{
		"reason": "coverage social link update",
		"items":  []model.SiteSocialLink{{Platform: "github", Label: "GitHub", URL: &githubURL, Enabled: true}},
	}), nil, admin, h.ReplaceAdminSiteSocialLinks)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	createAccountBody := `{"email":"coverage-admin@example.com","phone":"+6281234567890","display_name":"Coverage Operator","role":"admin","reason":"coverage account lifecycle"}`
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/accounts", createAccountBody, nil, admin, h.CreateAdminAccount)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := handlerAdminCatalogCoverageFindUser(t, st, "coverage-admin@example.com")
	w = invokeHandler(t, h, http.MethodPatch, "/v1/admin/accounts/"+created.ID, `{"disabled":true,"reason":"coverage disable"}`, gin.Params{{Key: "id", Value: created.ID}}, admin, h.UpdateAdminAccount)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/operator/invitations/retired", "", nil, nil, h.RetiredOperatorInvitation)
	assert.Equal(t, http.StatusGone, w.Code)

	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/data-requests", "", nil, admin, h.AdminDataRequests)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/data-requests?page=1&limit=2&status=completed", "", nil, admin, h.AdminDataRequests)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	st.Lock()
	st.DataRequests = append(st.DataRequests,
		model.DataRequest{ID: "DR-COVERAGE-RETRY", UserID: "usr_gading", Type: "export", Status: "failed", RetryCount: 0, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
		model.DataRequest{ID: "DR-COVERAGE-REJECT", UserID: "usr_dery", Type: "export", Status: "queued", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	)
	for index := range st.SupportCases {
		if st.SupportCases[index].ID == "CASE-1087" || st.SupportCases[index].ID == "CASE-1084" {
			st.SupportCases[index].Owner = ""
		}
	}
	st.Unlock()
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/data-requests/DR-COVERAGE-RETRY/retry", "", gin.Params{{Key: "id", Value: "DR-COVERAGE-RETRY"}}, admin, h.RetryAdminDataRequest)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/data-requests/DR-COVERAGE-REJECT/reject", `{"reason":"coverage rejection"}`, gin.Params{{Key: "id", Value: "DR-COVERAGE-REJECT"}}, admin, h.RejectAdminDataRequest)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/claim", `{"reason":"coverage investigation"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.ClaimAdminSupportCase)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/support-cases/CASE-1087", "", gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.GetAdminSupportCaseDetail)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/messages", `{"content":"Coverage response"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.ReplyAdminSupportCase)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/transition", `{"status":"resolved"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.TransitionAdminSupportCase)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1084/claim", `{"reason":"coverage second case"}`, gin.Params{{Key: "id", Value: "CASE-1084"}}, admin, h.ClaimAdminSupportCase)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1084/release", `{"reason":"coverage release"}`, gin.Params{{Key: "id", Value: "CASE-1084"}}, admin, h.ReleaseAdminSupportCase)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// The seeded reflections are intentionally plaintext demo fixtures; remove
	// them before exercising the encrypted export path.
	st.Lock()
	st.JournalEntries = nil
	st.Unlock()
	export, _, err := h.services.Support.CreateDataRequestWithResult(t.Context(), "usr_gading", "export")
	require.NoError(t, err)
	w = invokeHandler(t, h, http.MethodGet, "/v1/data-requests/"+export.ID+"/download", "", gin.Params{{Key: "id", Value: export.ID}}, map[string]any{"user_id": "usr_gading", "role": "user"}, h.DownloadDataExport)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "application/zip", w.Header().Get("Content-Type"))
}

func TestHandlerAdminCatalogCoverage_EducationEditorialAndProgress(t *testing.T) {
	harness := newHandlerAdminCatalogCoverageHarness(t)
	h, st := harness.h, harness.st
	admin := handlerAdminCatalogCoverageAdminValues()

	newDocument := st.Modules[1].DraftDocument
	w := invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules", handlerAdminCatalogCoverageJSON(t, map[string]any{
		"slug": "coverage-education-handler", "document": newDocument,
	}), nil, admin, h.CreateAdminModule)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created model.EducationModule
	st.RLock()
	for _, item := range st.Modules {
		if item.Slug == "coverage-education-handler" {
			created = item
		}
	}
	st.RUnlock()
	require.NotEmpty(t, created.ID)

	updatedDocument := created.DraftDocument
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/modules/"+created.ID, handlerAdminCatalogCoverageJSON(t, map[string]any{
		"slug": "coverage-education-handler-updated", "expected_revision": created.DraftRevision, "document": updatedDocument,
	}), gin.Params{{Key: "id", Value: created.ID}}, admin, h.UpdateAdminModule)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules/"+created.ID+"/submit-review", "", gin.Params{{Key: "id", Value: created.ID}}, admin, h.SubmitAdminModuleReview)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules/"+created.ID+"/publish", "", gin.Params{{Key: "id", Value: created.ID}}, admin, h.PublishAdminModule)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/modules/"+created.ID, "", gin.Params{{Key: "id", Value: created.ID}}, admin, h.AdminModuleDetail)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/modules/"+created.ID+"/revisions", "", gin.Params{{Key: "id", Value: created.ID}}, admin, h.AdminModuleRevisions)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var revisions []model.EducationRevision
	var revisionEnvelope struct {
		Data []model.EducationRevision `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &revisionEnvelope))
	revisions = revisionEnvelope.Data
	require.NotEmpty(t, revisions)
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules/"+created.ID+"/revisions/"+revisions[0].ID+"/rollback", `{"reason":"coverage rollback"}`, gin.Params{{Key: "id", Value: created.ID}, {Key: "revision_id", Value: revisions[0].ID}}, admin, h.RollbackAdminModule)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	part, err := writer.CreateFormFile("file", "coverage.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("\x89PNG\r\n\x1a\ncoverage"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("purpose", "thumbnail"))
	require.NoError(t, writer.Close())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/media", upload.String(), nil, admin, h.UploadAdminEducationMedia)
	// invokeHandler creates an application/json request, so exercise the multipart
	// handler with a real request while keeping all setup in this test file.
	_ = w
	multipartRequest := httptest.NewRequest(http.MethodPost, "/v1/admin/content/media", &upload)
	multipartRequest.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = multipartRequest
	context.Set("request_id", "handler-admin-catalog-media")
	context.Set("user_id", "usr_nasywa")
	context.Set("role", "admin")
	h.UploadAdminEducationMedia(context)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())

	var uploaded model.EducationMedia
	st.RLock()
	for _, item := range st.EducationMedia {
		if item.OriginalName == "coverage.png" {
			uploaded = item
		}
	}
	st.RUnlock()
	require.NotEmpty(t, uploaded.ID)
	st.Lock()
	for index := range st.EducationMedia {
		if st.EducationMedia[index].ID == uploaded.ID {
			st.EducationMedia[index].Status = "published"
		}
	}
	st.Unlock()
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/media/external", `{"purpose":"content","media_type":"video","url":"https://www.youtube.com/watch?v=coverage"}`, nil, admin, h.RegisterAdminExternalMedia)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	for _, fn := range []gin.HandlerFunc{h.PublishedEducationMedia, h.AdminEducationMedia} {
		w = invokeHandler(t, h, http.MethodGet, "/v1/education/media/"+uploaded.ID, "", gin.Params{{Key: "id", Value: uploaded.ID}}, admin, fn)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}

	var studentModule model.EducationModule
	st.RLock()
	for _, item := range st.Modules {
		if item.PublishedDocument != nil {
			studentModule = item
			break
		}
	}
	st.RUnlock()
	require.NotNil(t, studentModule.PublishedDocument)
	section := studentModule.PublishedDocument.Sections[0]
	w = invokeHandler(t, h, http.MethodPut, "/v1/psychoeducation/modules/"+studentModule.ID+"/revisions/1/progress", `{"completed_section_ids":["`+section.ID+`"]}`, gin.Params{{Key: "id", Value: studentModule.ID}, {Key: "revision", Value: "1"}}, map[string]any{"user_id": "usr_gading", "role": "user"}, h.UpdateEducationProgress)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	if section.Translations["id"].KnowledgeCheck != nil {
		check := section.Translations["id"].KnowledgeCheck
		w = invokeHandler(t, h, http.MethodPost, "/v1/psychoeducation/modules/"+studentModule.ID+"/revisions/1/checks/"+check.ID+"/answer", handlerAdminCatalogCoverageJSON(t, map[string]string{"choice_id": check.Choices[0].ID}), gin.Params{{Key: "id", Value: studentModule.ID}, {Key: "revision", Value: "1"}, {Key: "check_id", Value: check.ID}}, map[string]any{"user_id": "usr_gading", "role": "user"}, h.AnswerEducationCheck)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/modules/"+created.ID, "", gin.Params{{Key: "id", Value: created.ID}}, admin, h.DeleteAdminModule)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestHandlerAdminCatalogCoverage_LearningHubEditorialAndStudentFlows(t *testing.T) {
	harness := newHandlerAdminCatalogCoverageHarness(t)
	h, st := harness.h, harness.st
	admin := handlerAdminCatalogCoverageAdminValues()

	w := invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/taxonomy/clusters", `{"slug":"coverage-cluster","title_id":"Cluster Coverage","title_en":"Coverage Cluster"}`, nil, admin, h.CreateAdminLearningHubCluster)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var cluster model.AdminLearningCluster
	st.RLock()
	for _, item := range st.AdminLearningClusters {
		if item.Slug == "coverage-cluster" {
			cluster = item
		}
	}
	st.RUnlock()
	require.NotEmpty(t, cluster.ID)
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/learning-hub/taxonomy/clusters/"+cluster.ID, `{"slug":"coverage-cluster","title_id":"Cluster Updated","title_en":"Updated Cluster"}`, gin.Params{{Key: "id", Value: cluster.ID}}, admin, h.UpdateAdminLearningHubCluster)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/taxonomy/programs", `{"slug":"coverage-program","name":"Coverage Program","degree":"S1","primary_cluster_slug":"coverage-cluster"}`, nil, admin, h.CreateAdminLearningHubProgram)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var program model.AdminAcademicProgram
	st.RLock()
	for _, item := range st.AdminAcademicPrograms {
		if item.Slug == "coverage-program" {
			program = item
		}
	}
	st.RUnlock()
	require.NotEmpty(t, program.ID)
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/learning-hub/taxonomy/programs/"+program.ID, `{"slug":"coverage-program","name":"Coverage Program Updated","degree":"S1","primary_cluster_slug":"coverage-cluster"}`, gin.Params{{Key: "id", Value: program.ID}}, admin, h.UpdateAdminLearningHubProgram)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	draft := handlerAdminCatalogCoverageLearningDraft()
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/items", handlerAdminCatalogCoverageJSON(t, draft), nil, admin, h.CreateAdminLearningHubItem)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	item := handlerAdminCatalogCoverageFindLearningItem(t, st, draft.Slug)
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/items", "", nil, admin, h.AdminLearningHubItems)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/items?page=1&limit=5&q=coverage", "", nil, admin, h.AdminLearningHubItems)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/items/"+item.ID, "", gin.Params{{Key: "id", Value: item.ID}}, admin, h.AdminLearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/learning-hub/items/"+item.ID, handlerAdminCatalogCoverageJSON(t, map[string]any{"expected_revision": item.DraftRevision, "draft": draft}), gin.Params{{Key: "id", Value: item.ID}}, admin, h.UpdateAdminLearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/items/"+item.ID+"/submit-review", "", gin.Params{{Key: "id", Value: item.ID}}, admin, h.SubmitAdminLearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/items/"+item.ID+"/publish", "", gin.Params{{Key: "id", Value: item.ID}}, admin, h.PublishAdminLearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/items/"+item.ID+"/revisions", "", gin.Params{{Key: "id", Value: item.ID}}, admin, h.AdminLearningHubRevisions)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	revision := handlerAdminCatalogCoverageFindLearningRevision(t, st, item.ID)
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/items/"+item.ID+"/revisions/"+revision.ID+"/rollback", `{"reason":"coverage rollback"}`, gin.Params{{Key: "id", Value: item.ID}, {Key: "revision_id", Value: revision.ID}}, admin, h.RollbackAdminLearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/taxonomy", "", nil, admin, h.AdminLearningHubTaxonomy)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	values := map[string]any{"user_id": "usr_gading", "role": "user"}
	for _, request := range []struct {
		method string
		path   string
		body   string
		fn     gin.HandlerFunc
	}{
		{http.MethodGet, "/v1/learning-hub/catalog", "", h.LearningHubCatalog},
		{http.MethodGet, "/v1/learning-hub/providers?page=1&limit=5", "", h.LearningHubProviders},
		{http.MethodGet, "/v1/learning-hub/items?provider=open-learning", "", h.LearningHubItemsByProvider},
		{http.MethodGet, "/v1/learning-hub/progress", "", h.LearningHubProgress},
	} {
		w = invokeHandler(t, h, request.method, request.path, request.body, nil, values, request.fn)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
	w = invokeHandler(t, h, http.MethodGet, "/v1/learning-hub/items/"+draft.Slug, "", gin.Params{{Key: "slug", Value: draft.Slug}}, values, h.LearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPut, "/v1/learning-hub/items/"+item.ID+"/state", `{"state":"started"}`, gin.Params{{Key: "id", Value: item.ID}}, values, h.UpdateLearningHubState)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/learning-hub/items/"+item.ID+"/checkpoint", `{"reflection":"A useful reflection","outcome":"I will practice this"}`, gin.Params{{Key: "id", Value: item.ID}}, values, h.CreateLearningHubCheckpoint)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/learning-hub/items/"+item.ID, "", gin.Params{{Key: "id", Value: item.ID}}, admin, h.DeleteAdminLearningHubItem)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/learning-hub/taxonomy/programs/"+program.ID, "", gin.Params{{Key: "id", Value: program.ID}}, admin, h.DeleteAdminLearningHubProgram)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/learning-hub/taxonomy/clusters/"+cluster.ID, "", gin.Params{{Key: "id", Value: cluster.ID}}, admin, h.DeleteAdminLearningHubCluster)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestHandlerAdminCatalogCoverage_EmergencyLifecycle(t *testing.T) {
	harness := newHandlerAdminCatalogCoverageHarness(t)
	h, st := harness.h, harness.st
	admin := handlerAdminCatalogCoverageAdminValues()
	user := map[string]any{"user_id": "usr_gading", "role": "user"}
	st.Lock()
	st.Devices = append(st.Devices, model.Device{ID: "dev_coverage", UserID: "usr_gading", Platform: "android", ProtectionStatus: "active"})
	st.Unlock()

	w := invokeHandler(t, h, http.MethodPost, "/v1/emergency-key-requests", `{"device_id":"dev_coverage"}`, nil, user, h.RequestEmergencyKey)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/emergency-key-requests/current?device_id=dev_coverage", "", nil, user, h.CurrentEmergencyKeyRequest)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/emergency-key-requests", "", nil, admin, h.PendingEmergencyKeyRequests)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/emergency-key-requests?page=1&limit=5&status=pending", "", nil, admin, h.PendingEmergencyKeyRequests)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/emergency-key-requests/emk_req_0001/review", "", gin.Params{{Key: "id", Value: "emk_req_0001"}}, admin, h.ReviewEmergencyKeyRequest)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/emergency-key-requests/emk_req_0001/approve", "", gin.Params{{Key: "id", Value: "emk_req_0001"}}, admin, h.ApproveEmergencyKeyRequest)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestHandlerAdminCatalogCoverage_ErrorAndValidationBranches(t *testing.T) {
	harness := newHandlerAdminCatalogCoverageHarness(t)
	h, st := harness.h, harness.st
	admin := handlerAdminCatalogCoverageAdminValues()
	user := map[string]any{"user_id": "usr_gading", "role": "user"}

	w := invokeHandler(t, h, http.MethodPut, "/v1/admin/site-social-links", `{`, nil, admin, h.ReplaceAdminSiteSocialLinks)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/site-social-links", `{"reason":"bad url","items":[{"platform":"github","label":"GitHub","url":"http://evil.example"}]}`, nil, admin, h.ReplaceAdminSiteSocialLinks)
	assertHandlerError(t, w, http.StatusBadRequest, "site_social_links_failed")
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/overview", "", nil, map[string]any{"user_id": "usr_gading", "role": "user"}, h.AdminOverview)
	assertHandlerError(t, w, http.StatusForbidden, "admin_overview_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/accounts", `{`, nil, admin, h.CreateAdminAccount)
	assertHandlerError(t, w, http.StatusBadRequest, "validation_failed")
	w = invokeHandler(t, h, http.MethodPatch, "/v1/admin/accounts/usr_gading", `{"role":"admin"}`, gin.Params{{Key: "id", Value: "usr_gading"}}, admin, h.UpdateAdminAccount)
	assertHandlerError(t, w, http.StatusBadRequest, "validation_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/claim", `{`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.ClaimAdminSupportCase)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/release", `{`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.ReleaseAdminSupportCase)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/data-requests/missing/retry", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.RetryAdminDataRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "data_request_retry_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/data-requests/missing/reject", `{`, gin.Params{{Key: "id", Value: "missing"}}, admin, h.RejectAdminDataRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodGet, "/v1/data-requests/missing/download", "", gin.Params{{Key: "id", Value: "missing"}}, user, h.DownloadDataExport)
	assertHandlerError(t, w, http.StatusNotFound, "data_export_unavailable")
	w = invokeHandler(t, h, http.MethodPost, "/v1/data-requests/confirm-delete", `{`, nil, user, h.ConfirmAccountDeletion)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/v1/data-requests/confirm-delete", `{"token":"invalid"}`, nil, user, h.ConfirmAccountDeletion)
	assertHandlerError(t, w, http.StatusBadRequest, "account_deletion_failed")

	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/modules/missing", `{`, gin.Params{{Key: "id", Value: "missing"}}, admin, h.UpdateAdminModule)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules/missing/submit-review", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.SubmitAdminModuleReview)
	assertHandlerError(t, w, http.StatusNotFound, "module_not_found")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules/missing/publish", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.PublishAdminModule)
	assertHandlerError(t, w, http.StatusNotFound, "module_not_found")
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/modules/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.DeleteAdminModule)
	assertHandlerError(t, w, http.StatusNotFound, "module_not_found")
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/modules/missing/revisions", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.AdminModuleRevisions)
	assertHandlerError(t, w, http.StatusNotFound, "module_not_found")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/modules/missing/revisions/missing/rollback", `{"reason":""}`, gin.Params{{Key: "id", Value: "missing"}, {Key: "revision_id", Value: "missing"}}, admin, h.RollbackAdminModule)
	assertHandlerError(t, w, http.StatusBadRequest, "education_validation_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/media/external", `{`, nil, admin, h.RegisterAdminExternalMedia)
	assertHandlerError(t, w, http.StatusBadRequest, "err_validation")
	w = invokeHandler(t, h, http.MethodGet, "/v1/education/media/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.PublishedEducationMedia)
	assertHandlerError(t, w, http.StatusNotFound, "education_media_not_found")
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/media/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.AdminEducationMedia)
	assertHandlerError(t, w, http.StatusNotFound, "education_media_not_found")

	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/items?status=invalid", "", nil, admin, h.AdminLearningHubItems)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/learning-hub/items/missing", `{"expected_revision":0}`, gin.Params{{Key: "id", Value: "missing"}}, admin, h.UpdateAdminLearningHubItem)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/learning-hub/items/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.DeleteAdminLearningHubItem)
	assertHandlerError(t, w, http.StatusNotFound, "learning_hub_admin_not_found")
	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/content/learning-hub/items/missing/revisions", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.AdminLearningHubRevisions)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/items/missing/revisions/missing/rollback", `{"reason":""}`, gin.Params{{Key: "id", Value: "missing"}, {Key: "revision_id", Value: "missing"}}, admin, h.RollbackAdminLearningHubItem)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/taxonomy/clusters", `{`, nil, admin, h.CreateAdminLearningHubCluster)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/learning-hub/taxonomy/clusters/missing", `{"slug":"bad slug"}`, gin.Params{{Key: "id", Value: "missing"}}, admin, h.UpdateAdminLearningHubCluster)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/learning-hub/taxonomy/clusters/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.DeleteAdminLearningHubCluster)
	assertHandlerError(t, w, http.StatusNotFound, "learning_hub_admin_not_found")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/content/learning-hub/taxonomy/programs", `{`, nil, admin, h.CreateAdminLearningHubProgram)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodPut, "/v1/admin/content/learning-hub/taxonomy/programs/missing", `{"slug":"bad slug"}`, gin.Params{{Key: "id", Value: "missing"}}, admin, h.UpdateAdminLearningHubProgram)
	assertHandlerError(t, w, http.StatusBadRequest, "learning_hub_admin_validation_failed")
	w = invokeHandler(t, h, http.MethodDelete, "/v1/admin/content/learning-hub/taxonomy/programs/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.DeleteAdminLearningHubProgram)
	assertHandlerError(t, w, http.StatusNotFound, "learning_hub_admin_not_found")

	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/emergency-key-requests/missing", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.ReviewEmergencyKeyRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "emergency_review_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/emergency-key-requests/missing/approve", "", gin.Params{{Key: "id", Value: "missing"}}, admin, h.ApproveEmergencyKeyRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "generate_key_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/devices/unlock", `{"emergency_key":"invalid","device_id":"dev_android"}`, nil, nil, h.EmergencyUnlock)
	assertHandlerError(t, w, http.StatusInternalServerError, "invalid_key")
	w = invokeHandler(t, h, http.MethodPost, "/v1/emergency-key-requests", `{"device_id":"missing"}`, nil, user, h.RequestEmergencyKey)
	assertHandlerError(t, w, http.StatusBadRequest, "emergency_request_failed")
	w = invokeHandler(t, h, http.MethodGet, "/v1/emergency-key-requests/current?device_id=missing", "", nil, user, h.CurrentEmergencyKeyRequest)
	assertHandlerError(t, w, http.StatusNotFound, "emergency_request_not_found")

	w = invokeHandler(t, h, http.MethodGet, "/v1/admin/support-cases/CASE-1087", "", gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.GetAdminSupportCaseDetail)
	assertHandlerError(t, w, http.StatusNotFound, "support_case_not_found")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/messages", `{"content":"unclaimed"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.ReplyAdminSupportCase)
	assertHandlerError(t, w, http.StatusBadRequest, "support_reply_failed")
	w = invokeHandler(t, h, http.MethodPost, "/v1/admin/support-cases/CASE-1087/transition", `{"status":"resolved"}`, gin.Params{{Key: "id", Value: "CASE-1087"}}, admin, h.TransitionAdminSupportCase)
	assertHandlerError(t, w, http.StatusBadRequest, "support_transition_failed")
	w = invokeHandler(t, h, http.MethodGet, "/v1/data-requests", "", nil, map[string]any{"user_id": "usr_nasywa", "role": "admin"}, h.GetDataRequests)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = invokeHandler(t, h, http.MethodPost, "/v1/data-requests", `{"type":"unsupported"}`, nil, user, h.CreateDataRequest)
	assertHandlerError(t, w, http.StatusBadRequest, "data_request_failed")

	// Keep the fixture referenced so this test remains explicit about using an
	// isolated in-memory backend rather than an external database.
	assert.NotNil(t, st)
}

func handlerAdminCatalogCoverageAdminRouter(t *testing.T) (*gin.Engine, *Handler, *service.AuthService) {
	t.Helper()
	harness := newHandlerAdminCatalogCoverageHarness(t)
	mid := harness.h.middleware
	r := gin.New()
	r.Use(mid.RequestID(), mid.PrivacyGuard())
	admin := r.Group("/v1/admin")
	admin.Use(mid.AuthRequired(), mid.RequireRoles("admin"), mid.RequireVerifiedPhone())
	admin.GET("/overview", harness.h.AdminOverview)
	admin.PUT("/site-social-links", mid.RequireRecentAuth(15*time.Minute), harness.h.ReplaceAdminSiteSocialLinks)
	return r, harness.h, harness.h.services.Auth
}

func handlerAdminCatalogCoverageToken(t *testing.T, auth *service.AuthService, email string) string {
	t.Helper()
	response, err := auth.Login(t.Context(), email, "password", "")
	require.NoError(t, err)
	return response.AccessToken
}

func handlerAdminCatalogCoverageStaleToken(t *testing.T) string {
	t.Helper()
	now := time.Now().UTC()
	claims := model.Claims{
		UserID: "usr_nasywa", Email: "nasywa@gmail.com", Role: "admin",
		AuthTime: jwt.NewNumericDate(now.Add(-time.Hour)),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "usr_nasywa", IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), Issuer: "gamblock-ai-backend",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret-very-long-please-32bytes!"))
	require.NoError(t, err)
	return token
}

func TestHandlerAdminCatalogCoverage_AdminRoleAndRecentAuthGuards(t *testing.T) {
	r, _, auth := handlerAdminCatalogCoverageAdminRouter(t)
	adminToken := handlerAdminCatalogCoverageToken(t, auth, "nasywa@gmail.com")
	userToken := handlerAdminCatalogCoverageToken(t, auth, "gading@gmail.com")

	for _, tc := range []struct {
		name  string
		path  string
		token string
		want  int
		code  string
	}{
		{name: "missing authentication", path: "/v1/admin/overview", want: http.StatusUnauthorized, code: "auth_required"},
		{name: "user role denied", path: "/v1/admin/overview", token: userToken, want: http.StatusForbidden, code: "forbidden"},
		{name: "admin access allowed", path: "/v1/admin/overview", token: adminToken, want: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			assert.Equal(t, tc.want, response.Code, response.Body.String())
			if tc.code != "" {
				assert.Contains(t, response.Body.String(), tc.code)
			}
		})
	}

	staleRequest := httptest.NewRequest(http.MethodPut, "/v1/admin/site-social-links", bytes.NewBufferString(`{"reason":"stale","items":[]}`))
	staleRequest.Header.Set("Content-Type", "application/json")
	staleRequest.Header.Set("Authorization", "Bearer "+handlerAdminCatalogCoverageStaleToken(t))
	staleResponse := httptest.NewRecorder()
	r.ServeHTTP(staleResponse, staleRequest)
	assert.Equal(t, http.StatusUnauthorized, staleResponse.Code)
	assert.Contains(t, staleResponse.Body.String(), "recent_auth_required")

	freshRequest := httptest.NewRequest(http.MethodPut, "/v1/admin/site-social-links", bytes.NewBufferString(`{"reason":"fresh auth","items":[]}`))
	freshRequest.Header.Set("Content-Type", "application/json")
	freshRequest.Header.Set("Authorization", "Bearer "+adminToken)
	freshResponse := httptest.NewRecorder()
	r.ServeHTTP(freshResponse, freshRequest)
	assert.Equal(t, http.StatusOK, freshResponse.Code, freshResponse.Body.String())
}
