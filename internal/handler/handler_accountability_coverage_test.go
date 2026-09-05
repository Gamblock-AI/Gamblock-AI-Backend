package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

type accountabilityCoverageHarness struct {
	r    *gin.Engine
	h    *Handler
	repo *repository.Repository
}

type accountabilityCoverageEnvelope struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"request_id"`
}

func newAccountabilityCoverageHarness(t *testing.T) *accountabilityCoverageHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv:               "development",
		JWTAccessSecret:      "test-secret-very-long-please",
		JWTAccessTTL:         3600 * time.Second,
		JWTRefreshTTL:        720 * time.Hour,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		NotificationMode:     "demo",
		PublicWebBaseURL:     "http://localhost:3000",
	}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID(), mid.PrivacyGuard())
	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())
	v1.POST("/auth/login", h.Login)
	v1.GET("/partners", mid.AuthRequired(), h.GetPartners)
	v1.POST("/partners/invitations", mid.AuthRequired(), h.CreatePartnerInvitation)
	v1.POST("/partners/invitations/:token/accept", mid.AuthRequired(), h.AcceptPartnerInvitation)
	v1.POST("/partners/:partner_link_id/revoke", mid.AuthRequired(), h.RevokePartner)

	accountability := v1.Group("/accountability")
	accountability.Use(mid.AuthRequired(), mid.RequireRoles("user", "partner"))
	accountability.GET("/workspace", h.AccountabilityWorkspace)
	accountability.GET("/summary", h.AccountabilitySummary)
	accountability.GET("/analytics", h.AccountabilityAnalytics)
	accountability.GET("/groups", h.AccountabilityGroups)
	accountability.GET("/members", h.AccountabilityMembers)
	accountability.GET("/analytics/members", h.AccountabilityAnalyticsMembers)
	accountability.GET("/exit-requests", h.AccountabilityExitRequests)
	accountability.GET("/contact-requests", h.AccountabilityContactRequests)
	accountability.GET("/flagged-members", mid.RequireRoles("partner"), h.FlaggedAccountabilityMembers)
	accountability.POST("/groups", mid.RequireRoles("partner"), h.CreateAccountabilityGroup)
	accountability.POST("/groups/preview", mid.RequireRoles("user"), h.PreviewAccountabilityGroup)
	accountability.POST("/groups/join", mid.RequireRoles("user"), h.JoinAccountabilityGroup)
	accountability.POST("/groups/:group_id/rotate-code", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.RotateAccountabilityGroupCode)
	accountability.POST("/groups/:group_id/delete", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.DeleteAccountabilityGroup)
	accountability.PATCH("/memberships/:membership_id/sharing", mid.RequireRoles("user"), h.UpdateAccountabilitySharing)
	accountability.POST("/memberships/:membership_id/leave", mid.RequireRoles("user"), h.RequestAccountabilityLeave)
	accountability.POST("/exit-requests/:request_id/cancel", mid.RequireRoles("user"), h.CancelAccountabilityLeave)
	accountability.POST("/memberships/:membership_id/remove", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.RemoveAccountabilityMember)
	accountability.POST("/exit-requests/:request_id/resolve", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.ResolveAccountabilityLeave)
	accountability.POST("/contact-requests", mid.RequireRoles("user"), h.CreatePartnerContactRequest)
	accountability.POST("/contact-requests/:request_id/transition", h.TransitionPartnerContactRequest)

	v1.GET("/approval-requests", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.GetApprovalRequests)
	v1.POST("/approval-requests", mid.AuthRequired(), mid.RequireRoles("user"), h.CreateApprovalRequest)
	v1.POST("/approval-requests/:id/cancel", mid.AuthRequired(), mid.RequireRoles("user"), h.CancelApprovalRequest)
	v1.POST("/approval-requests/:id/approve", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.ApproveApprovalRequest)
	v1.POST("/approval-requests/:id/deny", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.DenyApprovalRequest)
	v1.POST("/approval-requests/:id/apply", mid.AuthRequired(), mid.RequireRoles("user"), h.ApplyApprovalRequest)
	v1.GET("/approval-requests/verify/:token", h.VerifyApprovalToken)
	v1.POST("/approval-requests/:id/resolve-by-token", h.ResolveApprovalByToken)
	admin := v1.Group("/admin")
	admin.Use(mid.AuthRequired(), mid.RequireRoles("admin"), mid.RequireVerifiedPhone())
	admin.GET("/analytics", h.AdminAnalytics)

	v1.GET("/organizations/mine", mid.AuthRequired(), h.GetCurrentUserOrganization)
	v1.GET("/organizations/:id", mid.AuthRequired(), h.GetOrganization)
	v1.POST("/organizations", mid.AuthRequired(), h.CreateOrganization)
	v1.POST("/organizations/join", mid.AuthRequired(), h.JoinOrganization)
	v1.GET("/organizations/:id/members", mid.AuthRequired(), h.ListOrganizationMembers)
	v1.GET("/organizations/:id/analytics", mid.AuthRequired(), h.GetOrganizationAnalytics)
	v1.DELETE("/organizations/:id/members/:user_id", mid.AuthRequired(), h.RemoveOrganizationMember)

	return &accountabilityCoverageHarness{r: r, h: h, repo: repo}
}

func accountabilityCoverageLogin(t *testing.T, r *gin.Engine, email string) string {
	t.Helper()
	body := []byte(`{"email":"` + email + `","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env accountabilityCoverageEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	data, ok := env.Data.(map[string]any)
	require.True(t, ok)
	token, ok := data["access_token"].(string)
	require.True(t, ok)
	return token
}

func accountabilityCoverageRequest(t *testing.T, r *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func accountabilityCoverageData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env accountabilityCoverageEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	data, ok := env.Data.(map[string]any)
	require.True(t, ok, w.Body.String())
	return data
}

func accountabilityCoverageError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, w.Code, w.Body.String())
	var env accountabilityCoverageEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.NotNil(t, env.Error)
	assert.Equal(t, code, env.Error.Code)
}

func accountabilityCoverageInvoke(t *testing.T, h *Handler, method, path, body, userID, role string, params gin.Params, fn gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Params = params
	c.Set("request_id", "accountability-coverage-test")
	c.Set("user_id", userID)
	c.Set("role", role)
	fn(c)
	return w
}

func TestHandlerAccountabilityCoverage_WorkspaceCollectionsAndRoles(t *testing.T) {
	harness := newAccountabilityCoverageHarness(t)
	userToken := accountabilityCoverageLogin(t, harness.r, "gading@gmail.com")
	partnerToken := accountabilityCoverageLogin(t, harness.r, "suci@gmail.com")
	adminToken := accountabilityCoverageLogin(t, harness.r, "nasywa@gmail.com")

	for _, path := range []string{
		"/v1/accountability/workspace",
		"/v1/accountability/summary",
		"/v1/accountability/groups?page=1&limit=5&status=active&q=kelas",
		"/v1/accountability/members?page=1&limit=5&protection=all&q=gad",
		"/v1/accountability/analytics/members?page=1&limit=5&group_id=all&q=gad",
		"/v1/accountability/exit-requests?status=all",
		"/v1/accountability/contact-requests?bucket=incoming",
	} {
		w := accountabilityCoverageRequest(t, harness.r, http.MethodGet, path, userToken, "")
		assert.Equal(t, http.StatusOK, w.Code, path+": "+w.Body.String())
	}
	for _, path := range []string{
		"/v1/accountability/workspace",
		"/v1/accountability/summary",
		"/v1/accountability/groups",
		"/v1/accountability/members",
		"/v1/accountability/analytics/members",
		"/v1/accountability/exit-requests",
		"/v1/accountability/contact-requests?bucket=history",
		"/v1/accountability/flagged-members",
	} {
		w := accountabilityCoverageRequest(t, harness.r, http.MethodGet, path, partnerToken, "")
		assert.Equal(t, http.StatusOK, w.Code, path+": "+w.Body.String())
	}

	assert.Equal(t, http.StatusForbidden, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/accountability/workspace", adminToken, "").Code)
	assert.Equal(t, http.StatusForbidden, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/accountability/flagged-members", userToken, "").Code)
}

func TestHandlerAccountabilityCoverage_GroupMutationsOwnershipAndContacts(t *testing.T) {
	harness := newAccountabilityCoverageHarness(t)
	userToken := accountabilityCoverageLogin(t, harness.r, "gading@gmail.com")
	deryToken := accountabilityCoverageLogin(t, harness.r, "dery@gmail.com")
	partnerToken := accountabilityCoverageLogin(t, harness.r, "suci@gmail.com")

	assert.Equal(t, http.StatusForbidden, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups", userToken, `{"name":"Nope"}`).Code)
	w := accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups", partnerToken, `{"name":"Study Support","description":"A focused support group"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	createdGroupID := accountabilityCoverageData(t, w)["id"].(string)

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/preview", deryToken, `{"code":"GAMBLOCK42"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/join", deryToken, `{"code":"GAMBLOCK42","confirmed":true}`)
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/preview", userToken, `{`), http.StatusBadRequest, "err_validation")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/preview", userToken, `{"code":"invalid"}`), http.StatusBadRequest, "accountability_code_invalid")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/join", userToken, `{"code":"GAMBLOCK42","confirmed":false}`), http.StatusBadRequest, "accountability_join_failed")

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/grp_demo/rotate-code", partnerToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/groups/"+createdGroupID+"/delete", partnerToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPatch, "/v1/accountability/memberships/mbr_active/sharing", userToken, `{"protection_health":false,"protection_activity":true,"recovery_engagement":false,"education_progress":true}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPatch, "/v1/accountability/memberships/mbr_active/sharing", userToken, `{`), http.StatusBadRequest, "err_validation")

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/contact-requests", userToken, `{"membership_id":"mbr_active","category":"check_in","message":"Boleh bantu cek progres saya?"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	contactID := accountabilityCoverageData(t, w)["id"].(string)
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/contact-requests/"+contactID+"/transition", partnerToken, `{"status":"acknowledged"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/contact-requests/"+contactID+"/transition", userToken, `{"status":"closed"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/contact-requests", userToken, `{"membership_id":"mbr_active","category":"invalid"}`), http.StatusBadRequest, "partner_contact_create_failed")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/contact-requests/unknown/transition", userToken, `{"status":"acknowledged"}`), http.StatusBadRequest, "partner_contact_transition_failed")

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/memberships/mbr_active/leave", userToken, `{"kind":"normal","reason":"Need a different support arrangement"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	exitID := accountabilityCoverageData(t, w)["id"].(string)
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/exit-requests/"+exitID+"/cancel", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/memberships/mbr_active/leave", userToken, `{"kind":"normal","reason":"Pending review"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	exitID = accountabilityCoverageData(t, w)["id"].(string)
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/exit-requests/"+exitID+"/resolve", partnerToken, `{"decision":"approved"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	harness = newAccountabilityCoverageHarness(t)
	partnerToken = accountabilityCoverageLogin(t, harness.r, "suci@gmail.com")
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/accountability/memberships/mbr_active/remove", partnerToken, `{"reason":"Membership review"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestHandlerAccountabilityCoverage_PartnerInvitationsAndApprovals(t *testing.T) {
	harness := newAccountabilityCoverageHarness(t)
	userToken := accountabilityCoverageLogin(t, harness.r, "gading@gmail.com")
	deryToken := accountabilityCoverageLogin(t, harness.r, "dery@gmail.com")
	partnerToken := accountabilityCoverageLogin(t, harness.r, "suci@gmail.com")

	w := accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/partners", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/partners/invitations", deryToken, `{"email":"suci@gmail.com","phone":"+6281200000000"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	invitation := accountabilityCoverageData(t, w)
	linkID := invitation["id"].(string)
	inviteURL := invitation["invite_url"].(string)
	token := inviteURL[len("http://localhost:3000/partner/invitations/"):]
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/partners/invitations/"+token+"/accept", partnerToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/partners/"+linkID+"/revoke", deryToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/partners/invitations", userToken, `{"email":""}`), http.StatusBadRequest, "partner_email_required")

	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/approval-requests", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/approval-requests?page=1&limit=5&status=pending&q=Troubleshooting", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests", userToken, `{}`), http.StatusBadRequest, "action_required")

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests", userToken, `{"action":"pause_protection","reason":"temporary pause","requested_duration_minutes":15,"device_id":"dev_android","membership_id":"mbr_active"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	approvalID := accountabilityCoverageData(t, w)["id"].(string)
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/"+approvalID+"/approve", partnerToken, `{"supportive_response":"Approved for setup."}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests", userToken, `{"action":"pause_protection","reason":"second request","requested_duration_minutes":30,"device_id":"dev_android","membership_id":"mbr_active"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	approvalID = accountabilityCoverageData(t, w)["id"].(string)
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/"+approvalID+"/deny", partnerToken, `{"supportive_response":"Please wait."}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/APR-2401/cancel", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/missing/cancel", userToken, ""), http.StatusBadRequest, "approval_cancel_failed")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/APR-2398/apply", userToken, `{"device_id":"dev_android"}`), http.StatusInternalServerError, "approval_apply_failed")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/APR-2398/apply", userToken, `{}`), http.StatusBadRequest, "device_id_required")
}

func TestHandlerAccountabilityCoverage_AnalyticsAndQuickApproval(t *testing.T) {
	harness := newAccountabilityCoverageHarness(t)
	userToken := accountabilityCoverageLogin(t, harness.r, "gading@gmail.com")
	partnerToken := accountabilityCoverageLogin(t, harness.r, "suci@gmail.com")
	adminToken := accountabilityCoverageLogin(t, harness.r, "nasywa@gmail.com")

	w := accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/accountability/analytics?days=30&group_id=grp_demo", partnerToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/accountability/analytics?days=nope", partnerToken, ""), http.StatusBadRequest, "analytics_period_invalid")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/accountability/analytics?days=7", partnerToken, ""), http.StatusBadRequest, "analytics_failed")

	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/admin/analytics?days=30", adminToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/admin/analytics?days=nope", adminToken, ""), http.StatusBadRequest, "analytics_period_invalid")
	assert.Equal(t, http.StatusForbidden, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/admin/analytics", userToken, "").Code)

	quickToken := "quick-accountability-coverage-token"
	_, err := harness.repo.CreateApprovalRequestWithToken(
		context.Background(), "APR-QUICK-COVERAGE", "usr_gading", "dev_android", "mbr_active",
		"pause_protection", "quick review", 15, time.Now().UTC().Add(time.Hour), service.HashRefreshToken(quickToken),
	)
	require.NoError(t, err)
	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/approval-requests/verify/"+quickToken, "", "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/APR-QUICK-COVERAGE/resolve-by-token", "", `{"token":"`+quickToken+`","status":"approved"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/APR-QUICK-COVERAGE/resolve-by-token", "", `{"token":"`+quickToken+`","status":"denied"}`), http.StatusBadRequest, "resolve_failed")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/approval-requests/unknown/resolve-by-token", "", `{"token":"","status":"invalid"}`), http.StatusBadRequest, "invalid_input")

	w = accountabilityCoverageInvoke(t, harness.h, http.MethodGet, "/v1/approval-requests/verify/", "", "usr_gading", "user", nil, harness.h.VerifyApprovalToken)
	accountabilityCoverageError(t, w, http.StatusBadRequest, "token_required")
}

func TestHandlerAccountabilityCoverage_OrganizationsAndOwnership(t *testing.T) {
	harness := newAccountabilityCoverageHarness(t)
	userToken := accountabilityCoverageLogin(t, harness.r, "gading@gmail.com")
	deryToken := accountabilityCoverageLogin(t, harness.r, "dery@gmail.com")
	adminToken := accountabilityCoverageLogin(t, harness.r, "nasywa@gmail.com")

	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/mine", userToken, ""), http.StatusNotFound, "no_org")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/organizations", userToken, `{"name":""}`), http.StatusBadRequest, "name_required")
	w := accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/organizations", userToken, `{"name":"Campus Focus Group"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	org := accountabilityCoverageData(t, w)
	orgID := org["id"].(string)
	groupCode := org["group_code"].(string)

	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/"+orgID, userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/mine", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/"+orgID+"/members", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/"+orgID+"/analytics", userToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/organizations/join", deryToken, `{"group_code":"bad-code"}`), http.StatusBadRequest, "join_failed")
	w = accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/organizations/join", deryToken, `{"group_code":"`+groupCode+`"}`)
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/"+orgID+"/members", deryToken, "")
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodDelete, "/v1/organizations/"+orgID+"/members/usr_dery", adminToken, ""), http.StatusBadRequest, "remove_member_failed")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodDelete, "/v1/organizations/"+orgID+"/members/usr_gading", userToken, ""), http.StatusBadRequest, "remove_member_failed")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodGet, "/v1/organizations/missing", userToken, ""), http.StatusNotFound, "org_not_found")
	accountabilityCoverageError(t, accountabilityCoverageRequest(t, harness.r, http.MethodPost, "/v1/organizations/join", userToken, `{`), http.StatusBadRequest, "group_code_required")
}
