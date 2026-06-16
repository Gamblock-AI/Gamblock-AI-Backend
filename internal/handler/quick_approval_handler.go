package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) VerifyApprovalToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		h.respondError(c, http.StatusBadRequest, "token_required", "Token validasi diperlukan")
		return
	}

	info, err := h.services.Accountability.VerifyQuickToken(c.Request.Context(), token)
	if err != nil {
		h.respondError(c, http.StatusNotFound, "invalid_token", "Token tidak valid atau sudah kadaluarsa")
		return
	}
	h.respond(c, http.StatusOK, info)
}

type resolveTokenInput struct {
	Token  string `json:"token"`
	Status string `json:"status"` // "approved" or "denied"
}

func (h *Handler) ResolveApprovalByToken(c *gin.Context) {
	var input resolveTokenInput
	_ = c.ShouldBindJSON(&input)
	if input.Token == "" || (input.Status != "approved" && input.Status != "denied") {
		h.respondError(c, http.StatusBadRequest, "invalid_input", "Token dan status (approved/denied) diperlukan")
		return
	}

	err := h.services.Accountability.ResolveByToken(c.Request.Context(), input.Token, input.Status)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "resolve_failed", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{"resolved": true, "status": input.Status})
}
