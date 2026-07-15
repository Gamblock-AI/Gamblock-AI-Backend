package handler

import (
	"net/http"

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
