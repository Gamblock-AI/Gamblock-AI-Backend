package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetReflections(c *gin.Context) {
	entries, err := h.services.Reflection.GetReflections(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_reflections_failed", err)
		return
	}
	h.respond(c, http.StatusOK, entries)
}

func (h *Handler) CreateReflection(c *gin.Context) {
	var input struct {
		Text string `json:"text"`
		Mood string `json:"mood"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Text == "" {
		h.respondCode(c, http.StatusBadRequest, "text_required")
		return
	}

	entry, err := h.services.Reflection.CreateReflection(c.Request.Context(), h.currentUserID(c), input.Text, input.Mood)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "reflection_create_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, entry)
}

func (h *Handler) GetModules(c *gin.Context) {
	modules, err := h.services.Admin.GetEducationModules(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_modules_failed", err)
		return
	}
	h.respond(c, http.StatusOK, modules)
}

func (h *Handler) GetModuleDetail(c *gin.Context) {
	slug := c.Param("slug")
	module, err := h.services.Reflection.GetEducationModuleBySlug(c.Request.Context(), slug)
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "module_not_found", err)
		return
	}
	h.respond(c, http.StatusOK, module)
}

func (h *Handler) GetSupportCases(c *gin.Context) {
	cases, err := h.services.Support.GetSupportCasesForUser(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_support_cases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, cases)
}

func (h *Handler) CreateSupportCase(c *gin.Context) {
	var input struct {
		Type     string `json:"type"`
		Summary  string `json:"summary"`
		Priority string `json:"priority"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Summary == "" {
		h.respondCode(c, http.StatusBadRequest, "summary_required")
		return
	}

	err := h.services.Support.CreateSupportCase(c.Request.Context(), h.currentUserID(c), input.Summary, input.Type, input.Priority)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "support_case_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"created": true})
}

func (h *Handler) GetDataRequests(c *gin.Context) {
	requests, err := h.services.Support.GetDataRequests(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_data_requests_failed", err)
		return
	}
	h.respond(c, http.StatusOK, requests)
}

func (h *Handler) CreateDataRequest(c *gin.Context) {
	var input struct {
		Type string `json:"type"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Type == "" {
		h.respondCode(c, http.StatusBadRequest, "type_required")
		return
	}

	err := h.services.Support.CreateDataRequest(c.Request.Context(), h.currentUserID(c), input.Type)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "data_request_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"requested": true})
}
