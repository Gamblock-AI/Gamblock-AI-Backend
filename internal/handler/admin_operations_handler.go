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
	var query model.PaginationQuery
	_ = c.ShouldBindQuery(&query)

	if c.Query("page") != "" || c.Query("limit") != "" || c.Query("action") != "" || c.Query("actor") != "" || c.Query("q") != "" {
		res, err := h.services.Admin.AuditEventsPaginated(c.Request.Context(), query)
		if err != nil {
			h.respondErrorErr(c, http.StatusInternalServerError, "audit_events_failed", err)
			return
		}
		h.respond(c, http.StatusOK, res)
		return
	}

	items, err := h.services.Admin.AuditEvents(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "audit_events_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) AdminAccounts(c *gin.Context) {
	var query model.PaginationQuery
	_ = c.ShouldBindQuery(&query)

	if c.Query("page") != "" || c.Query("limit") != "" || c.Query("role") != "" || c.Query("q") != "" {
		res, err := h.services.Admin.AccountsPaginated(c.Request.Context(), query)
		if err != nil {
			h.respondErrorErr(c, http.StatusInternalServerError, "admin_accounts_fetch_failed", err)
			return
		}
		h.respond(c, http.StatusOK, res)
		return
	}

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
		Phone       string `json:"phone"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		Reason      string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "validation_failed")
		return
	}
	user, temporaryPassword, err := h.services.Admin.CreateAccount(c.Request.Context(), h.currentUserID(c), input.Email, input.Phone, input.DisplayName, input.Role, input.Reason)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "admin_account_create_failed", err)
		return
	}
	previewCode, deliveryErr := h.services.Auth.BeginPhoneVerification(c.Request.Context(), user.ID, user.PhoneE164)
	if deliveryErr != nil {
		previewCode = ""
	}
	h.respond(c, http.StatusCreated, gin.H{
		"account": gin.H{
			"id": user.ID, "email": user.Email, "display_name": user.DisplayName,
			"role": user.Role, "must_change_password": true, "created_at": user.CreatedAt,
		},
		"temporary_password":              temporaryPassword,
		"phone_verification_preview_code": previewCode,
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
	var query model.PaginationQuery
	_ = c.ShouldBindQuery(&query)

	if c.Query("page") != "" || c.Query("limit") != "" || c.Query("status") != "" || c.Query("type") != "" {
		res, err := h.services.Support.GetAllDataRequestsPaginated(c.Request.Context(), query)
		if err != nil {
			h.respondErrorErr(c, http.StatusInternalServerError, "fetch_data_requests_failed", err)
			return
		}
		h.respond(c, http.StatusOK, res)
		return
	}

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
