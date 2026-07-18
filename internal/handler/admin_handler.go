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

func (h *Handler) AdminModelReleases(c *gin.Context) {
	releases, err := h.services.Admin.GetModelReleases(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_model_releases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, releases)
}

func (h *Handler) AdminSupportCases(c *gin.Context) {
	cases, err := h.services.Support.GetSupportCases(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_support_cases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, cases)
}
