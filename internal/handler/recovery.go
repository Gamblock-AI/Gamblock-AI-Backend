package handler

import (
	"net/http"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetIntention(c *gin.Context) {
	intn, err := h.services.Recovery.GetActiveIntention(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "err_internal", err)
		return
	}
	h.respond(c, http.StatusOK, intn)
}

func (h *Handler) SaveIntention(c *gin.Context) {
	var req struct {
		Text   string `json:"intention_text"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}

	intn, err := h.services.Recovery.SaveIntention(c.Request.Context(), h.currentUserID(c), req.Text, req.Status)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "err_internal", err)
		return
	}
	h.respond(c, http.StatusOK, intn)
}

func (h *Handler) GetCheckIns(c *gin.Context) {
	list, err := h.services.Recovery.GetCheckIns(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "err_internal", err)
		return
	}
	h.respond(c, http.StatusOK, list)
}

func (h *Handler) CreateCheckIn(c *gin.Context) {
	var req struct {
		Mood        int    `json:"mood_score"`
		Urge        int    `json:"urge_score"`
		ContextText string `json:"context_text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}

	chk, err := h.services.Recovery.CreateCheckIn(c.Request.Context(), h.currentUserID(c), req.Mood, req.Urge, req.ContextText)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "err_internal", err)
		return
	}
	h.respond(c, http.StatusOK, chk)
}

func (h *Handler) GetRecoveryRecords(c *gin.Context) {
	items, err := h.services.Recovery.GetRecoveryRecords(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "recovery_records_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) SaveRecoveryRecord(c *gin.Context) {
	var input struct {
		ID         string         `json:"id"`
		Kind       string         `json:"kind"`
		RecordDate string         `json:"record_date"`
		Content    string         `json:"content"`
		Status     string         `json:"status"`
		Metadata   map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	item, err := h.services.Recovery.SaveRecoveryRecord(c.Request.Context(), h.currentUserID(c), input.ID, input.Kind, input.RecordDate, input.Content, input.Status, input.Metadata)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "recovery_record_save_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) GetRecoveryPractices(c *gin.Context) {
	items, err := h.services.Recovery.GetRecoveryPractices(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "recovery_practice_fetch_failed", err)
		return
	}
	h.respond(c, http.StatusOK, items)
}

func (h *Handler) CreateRecoveryPractice(c *gin.Context) {
	var input struct {
		PracticeKind    string `json:"practice_kind"`
		DurationSeconds int    `json:"duration_seconds"`
		Feedback        string `json:"feedback"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "recovery_practice_invalid")
		return
	}
	item, err := h.services.Recovery.SaveRecoveryPractice(c.Request.Context(), h.currentUserID(c), input.PracticeKind, input.DurationSeconds, input.Feedback)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "recovery_practice_invalid", err)
		return
	}
	h.respond(c, http.StatusCreated, item)
}

func (h *Handler) GetRecoverySpace(c *gin.Context) {
	item, err := h.services.Recovery.GetRecoverySpace(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "recovery_space_fetch_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) UpdateRecoverySpace(c *gin.Context) {
	var input struct {
		PlacedItems map[string]any `json:"placed_items"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	item, err := h.services.Recovery.UpdateRecoverySpace(c.Request.Context(), h.currentUserID(c), input.PlacedItems)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "recovery_space_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) GetCurrentWeeklyReview(c *gin.Context) {
	item, err := h.services.Recovery.GetCurrentWeeklyReview(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "weekly_review_fetch_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}

func (h *Handler) SaveCurrentWeeklyReview(c *gin.Context) {
	var input model.WeeklyReview
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	item, err := h.services.Recovery.SaveCurrentWeeklyReview(c.Request.Context(), h.currentUserID(c), input)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "weekly_review_save_failed", err)
		return
	}
	h.respond(c, http.StatusOK, item)
}
