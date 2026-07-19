package handler

import (
	"net/http"
	"strings"

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

func (h *Handler) UploadAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarRequestBytes)
	file, _, err := c.Request.FormFile("avatar")
	if err != nil {
		h.respondCode(c, http.StatusBadRequest, "profile_update_failed")
		return
	}
	defer file.Close() //nolint:errcheck
	user, err := h.services.Client.UploadAvatar(c.Request.Context(), h.currentUserID(c), file)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "profile_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, user)
}

func (h *Handler) DeleteAvatar(c *gin.Context) {
	user, err := h.services.Client.DeleteAvatar(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "profile_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, user)
}

func (h *Handler) UserAvatar(c *gin.Context) {
	file, size, err := h.services.Client.AvatarFile(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "profile_not_found")
		return
	}
	defer file.Close() //nolint:errcheck
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Header("Content-Disposition", "inline")
	c.DataFromReader(http.StatusOK, size, "image/webp", file, nil)
}

const maxAvatarRequestBytes = (2 << 20) + (256 << 10)

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

func (h *Handler) UpdatePassword(c *gin.Context) {
	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.CurrentPassword == "" || len(input.NewPassword) < 8 {
		h.respondCode(c, http.StatusBadRequest, "password_validation_failed")
		return
	}
	if err := h.services.Auth.UpdatePassword(c.Request.Context(), h.currentUserID(c), input.CurrentPassword, input.NewPassword); err != nil {
		code := "password_update_failed"
		if strings.Contains(err.Error(), "current password") {
			code = "current_password_invalid"
		} else if strings.Contains(err.Error(), "different") {
			code = "password_reuse_not_allowed"
		}
		h.respondErrorErr(c, http.StatusBadRequest, code, err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"updated": true, "reauth_required": true})
}
