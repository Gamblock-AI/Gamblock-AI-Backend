package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (h *Handler) PortalOverview(c *gin.Context) {
	overview, err := h.services.Admin.GetPortalOverview(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "portal_overview_failed", err)
		return
	}
	h.respond(c, http.StatusOK, overview)
}

func (h *Handler) AdminModules(c *gin.Context) {
	var query model.PaginationQuery
	_ = c.ShouldBindQuery(&query)

	if hasPaginationQuery(c) || c.Query("status") != "" || c.Query("q") != "" {
		res, err := h.services.Education.AdminModulesPaginated(c.Request.Context(), query)
		if err != nil {
			h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_modules_failed", err)
			return
		}
		h.respond(c, http.StatusOK, res)
		return
	}

	modules, err := h.services.Education.AdminModules(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_modules_failed", err)
		return
	}
	h.respond(c, http.StatusOK, modules)
}

func (h *Handler) CreateAdminModule(c *gin.Context) {
	var input struct {
		Slug     string                  `json:"slug"`
		Document model.EducationDocument `json:"document"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if strings.TrimSpace(input.Slug) == "" || input.Document.EstimatedMinutes < 1 {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	module, err := h.services.Education.CreateModule(c.Request.Context(), h.currentUserID(c), input.Slug, input.Document)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "create_admin_module_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, module)
}

func (h *Handler) AdminSupportCases(c *gin.Context) {
	var query model.PaginationQuery
	_ = c.ShouldBindQuery(&query)

	if hasPaginationQuery(c) || c.Query("status") != "" || c.Query("priority") != "" || c.Query("q") != "" || c.Query("bucket") != "" || c.Query("assignee") != "" {
		res, err := h.services.Support.GetSupportCasesForAdminPaginated(c.Request.Context(), h.currentUserID(c), query)
		if err != nil {
			h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_support_cases_failed", err)
			return
		}
		h.respond(c, http.StatusOK, res)
		return
	}

	cases, err := h.services.Support.GetSupportCasesForAdmin(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_support_cases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, cases)
}
