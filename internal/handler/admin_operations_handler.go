package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (h *Handler) PublicSiteSocialLinks(c *gin.Context) {
	items, err := h.services.Admin.PublicSocialLinks(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "site_social_links_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) AdminOverview(c *gin.Context) {
	item, err := h.services.Admin.Overview(c.Request.Context(), currentRole(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusForbidden, "admin_overview_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) AdminSiteSocialLinks(c *gin.Context) {
	items, err := h.services.Admin.SiteSocialLinks(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "site_social_links_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) ReplaceAdminSiteSocialLinks(c *gin.Context) {
	var input struct {
		Items  []model.SiteSocialLink `json:"items"`
		Reason string                 `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Reason) == "" {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	items, err := h.services.Admin.ReplaceSiteSocialLinks(c.Request.Context(), h.currentUserID(c), input.Reason, input.Items)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "site_social_links_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) AdminAuditEvents(c *gin.Context) {
	items, err := h.services.Admin.AuditEvents(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "audit_events_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) AdminAccounts(c *gin.Context) {
	accounts, err := h.services.Admin.Accounts(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "admin_accounts_fetch_failed", err)
		return
	}
	h.respond(c, http.StatusOK, accounts)
}

func (h *Handler) CreateAdminAccount(c *gin.Context) {
	var input struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		Reason      string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "validation_failed")
		return
	}
	user, temporaryPassword, err := h.services.Admin.CreateAccount(c.Request.Context(), h.currentUserID(c), input.Email, input.DisplayName, input.Role, input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "admin_account_create_failed", err)
		return
	}
	previewURL, deliveryErr := h.services.Auth.BeginEmailVerification(c.Request.Context(), user)
	if deliveryErr != nil {
		previewURL = ""
	}
	h.respond(c, http.StatusCreated, gin.H{
		"account": gin.H{
			"id": user.ID, "email": user.Email, "display_name": user.DisplayName,
			"role": user.Role, "must_change_password": true, "created_at": user.CreatedAt,
		},
		"temporary_password":       temporaryPassword,
		"verification_preview_url": previewURL,
	})
}

func (h *Handler) UpdateAdminAccount(c *gin.Context) {
	var input struct {
		Disabled bool    `json:"disabled"`
		Reason   string  `json:"reason"`
		Role     *string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Role != nil {
		h.respondCode(c, http.StatusBadRequest, "validation_failed")
		return
	}
	if err := h.services.Admin.UpdateAccount(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.Disabled, input.Reason); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "admin_account_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"updated": true})
}

func (h *Handler) RetiredOperatorInvitation(c *gin.Context) {
	h.respondCode(c, http.StatusGone, "operator_invitation_retired")
}

func (h *Handler) ClaimAdminSupportCase(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	item, err := h.services.Support.Claim(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusConflict, "support_claim_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) ReleaseAdminSupportCase(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.Support.ReleaseClaim(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.Reason); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "support_release_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"released": true})
}

func (h *Handler) AdminDataRequests(c *gin.Context) {
	items, err := h.services.Support.GetAllDataRequests(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_data_requests_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) RetryAdminDataRequest(c *gin.Context) {
	item, err := h.services.Support.RetryDataRequest(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "data_request_retry_failed", err)
		return
	}
	_ = h.services.Admin.RecordAudit(c.Request.Context(), h.currentUserID(c), "data_request_retried", "data_request", item.ID, "operator retry", nil)
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) RejectAdminDataRequest(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	item, err := h.services.Support.RejectDataRequest(c.Request.Context(), c.Param("id"), input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "data_request_reject_failed", err)
		return
	}
	_ = h.services.Admin.RecordAudit(c.Request.Context(), h.currentUserID(c), "data_request_rejected", "data_request", item.ID, input.Reason, nil)
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) AdminReleases(c *gin.Context) {
	releases, rollouts, err := h.services.Admin.Releases(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_releases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"releases": releases, "rollouts": rollouts})
}

func (h *Handler) UploadAdminReleaseArtifact(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<20)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}
	defer file.Close() //nolint:errcheck
	path, checksum, err := h.services.Admin.StoreReleaseArtifact(header.Filename, file)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "release_validation_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"artifact_path": path, "sha256": checksum})
}

func (h *Handler) CreateAdminRollout(c *gin.Context) {
	var input struct {
		Kind                 string `json:"kind"`
		ReleaseID            string `json:"release_id"`
		Platform             string `json:"platform"`
		Percentage           int    `json:"percentage"`
		AppVersionConstraint string `json:"app_version_constraint"`
		Reason               string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Reason) == "" {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	item, err := h.services.Admin.CreateRollout(c.Request.Context(), h.currentUserID(c), input.Kind, input.ReleaseID, input.Platform, input.Percentage, input.AppVersionConstraint, input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "release_rollout_create_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, item)
}

func (h *Handler) TransitionAdminRollout(c *gin.Context) {
	var input struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	item, err := h.services.Admin.TransitionRollout(c.Request.Context(), h.currentUserID(c), c.Param("id"), input.Action, input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "release_rollout_transition_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) DownloadDataExport(c *gin.Context) {
	content, err := h.services.Support.DataExportFile(c.Request.Context(), h.currentUserID(c), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "data_export_unavailable", err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="gamblock-ai-account-export.zip"`)
	c.Data(http.StatusOK, "application/zip", content)
}

func (h *Handler) ConfirmAccountDeletion(c *gin.Context) {
	var input struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.Support.ConfirmAccountDeletion(c.Request.Context(), h.currentUserID(c), input.Token); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "account_deletion_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"deleted": true})
}
