package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetProfile(c *gin.Context) {
	user, err := h.services.Client.GetProfile(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "profile_not_found", err)
		return
	}
	h.respond(c, http.StatusOK, user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var input struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	user, err := h.services.Client.UpdateProfile(c.Request.Context(), h.currentUserID(c), input.DisplayName)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "profile_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, user)
}
