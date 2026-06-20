package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Email == "" {
		h.respondCode(c, http.StatusBadRequest, "email_required")
		return
	}
	response, err := h.services.Auth.Login(c.Request.Context(), input.Email, input.Password)
	if err != nil {
		h.respondErrorErr(c, http.StatusUnauthorized, "invalid_credentials", err)
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) Register(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Email == "" || input.Name == "" {
		h.respondCode(c, http.StatusBadRequest, "validation_failed")
		return
	}
	response, err := h.services.Auth.Register(c.Request.Context(), input.Email, input.Password, input.Name)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "registration_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, response)
}

func (h *Handler) DevLogin(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		DeviceID string `json:"device_id"`
	}
	_ = c.ShouldBindJSON(&input)
	response, err := h.services.Auth.DevLogin(c.Request.Context(), input.Email, input.Role, input.DeviceID)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "dev_login_failed", err)
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) GoogleLogin(c *gin.Context) {
	var input struct {
		IDToken  string `json:"id_token"`
		DeviceID string `json:"device_id"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.IDToken == "" {
		h.respondCode(c, http.StatusBadRequest, "google_token_required")
		return
	}
	response, err := h.services.Auth.GoogleLogin(c.Request.Context(), input.IDToken, input.DeviceID)
	if err != nil {
		h.respondErrorErr(c, http.StatusUnauthorized, "google_verification_failed", err)
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) Refresh(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.RefreshToken == "" {
		h.respondCode(c, http.StatusBadRequest, "refresh_token_required")
		return
	}
	response, err := h.services.Auth.Refresh(c.Request.Context(), input.RefreshToken)
	if err != nil {
		h.respondErrorErr(c, http.StatusUnauthorized, "invalid_refresh_token", err)
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) Logout(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.ShouldBindJSON(&input)
	if err := h.services.Auth.Logout(c.Request.Context(), input.RefreshToken); err != nil {
		h.respondCode(c, http.StatusInternalServerError, "logout_failed")
		return
	}
	h.respond(c, http.StatusOK, gin.H{"revoked": true})
}
