package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
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
		Text      string `json:"text"`
		Mood      string `json:"mood"`
		MoodScore *int   `json:"mood_score"`
		NextStep  string `json:"next_step"`
		IsFocus   bool   `json:"is_focus"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Text == "" {
		h.respondCode(c, http.StatusBadRequest, "text_required")
		return
	}

	entry, err := h.services.Reflection.CreateReflectionEntry(c.Request.Context(), h.currentUserID(c), input.Text, input.MoodScore, input.NextStep, input.IsFocus)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "reflection_create_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, entry)
}

func (h *Handler) UpdateReflection(c *gin.Context) {
	var input model.ReflectionUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	entry, err := h.services.Reflection.UpdateReflection(c.Request.Context(), h.currentUserID(c), c.Param("id"), input)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "reflection_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, entry)
}

func (h *Handler) GetModules(c *gin.Context) {
	modules, err := h.services.Education.PublishedModules(c.Request.Context(), h.currentUserID(c), c.Query("locale"))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_modules_failed", err)
		return
	}
	h.respond(c, http.StatusOK, modules)
}

func (h *Handler) GetModuleDetail(c *gin.Context) {
	slug := c.Param("slug")
	module, err := h.services.Education.PublishedModule(c.Request.Context(), h.currentUserID(c), slug, c.Query("locale"))
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
		Detail   string `json:"detail"`
		Impact   string `json:"impact"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Summary == "" {
		h.respondCode(c, http.StatusBadRequest, "summary_required")
		return
	}

	item, err := h.services.Support.CreateThreadedSupportCase(c.Request.Context(), h.currentUserID(c), input.Summary, input.Detail, input.Type, input.Priority, input.Impact)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "support_case_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, item)
}

func (h *Handler) GetSupportCaseDetail(c *gin.Context) {
	item, err := h.services.Support.GetSupportCaseDetail(c.Request.Context(), h.currentUserID(c), currentRole(c), c.Param("id"))
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "support_case_not_found", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) ReplySupportCase(c *gin.Context) {
	var input struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	message, err := h.services.Support.Reply(c.Request.Context(), h.currentUserID(c), currentRole(c), c.Param("id"), input.Content)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "support_reply_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, message)
}

func (h *Handler) TransitionSupportCase(c *gin.Context) {
	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.Support.Transition(c.Request.Context(), h.currentUserID(c), currentRole(c), c.Param("id"), input.Status); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "support_transition_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"updated": true})
}

func currentRole(c *gin.Context) string {
	role, _ := c.Get("role")
	return fmt.Sprint(role)
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

	item, previewURL, err := h.services.Support.CreateDataRequestWithResult(c.Request.Context(), h.currentUserID(c), input.Type)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrDataRequestInvalid) {
			status = http.StatusBadRequest
		} else if errors.Is(err, service.ErrDataRequestConflict) {
			status = http.StatusConflict
		} else if errors.Is(err, service.ErrDataRequestForbidden) {
			status = http.StatusForbidden
		}
		h.respondErrorErr(c, status, "data_request_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"request": item, "confirmation_preview_url": previewURL})
}
