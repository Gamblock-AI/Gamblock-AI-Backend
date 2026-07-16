package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
	modules, err := h.services.Admin.GetEducationModules(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_modules_failed", err)
		return
	}
	h.respond(c, http.StatusOK, modules)
}

func (h *Handler) CreateAdminModule(c *gin.Context) {
	var input model.EducationModule
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if strings.TrimSpace(input.Slug) == "" || strings.TrimSpace(input.Title) == "" ||
		strings.TrimSpace(input.Summary) == "" || strings.TrimSpace(input.BodyMarkdown) == "" ||
		input.EstimatedMinutes < 1 {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	input.Status = "draft"
	if input.ID == "" {
		input.ID = "mod_" + uuid.NewString()[:8]
	}
	if err := h.services.Admin.CreateEducationModule(c.Request.Context(), input); err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "create_admin_module_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.ID, "status": "draft"})
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
