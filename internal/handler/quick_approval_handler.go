package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) VerifyApprovalToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		h.respondCode(c, http.StatusBadRequest, "token_required")
		return
	}

	info, err := h.services.Accountability.VerifyQuickToken(c.Request.Context(), token)
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "invalid_token")
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
		h.respondCode(c, http.StatusBadRequest, "invalid_input")
		return
	}

	err := h.services.Accountability.ResolveByToken(c.Request.Context(), input.Token, input.Status)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "resolve_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"resolved": true, "status": input.Status})
}
