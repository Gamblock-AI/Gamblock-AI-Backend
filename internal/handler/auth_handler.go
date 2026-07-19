package handler

import (
	"errors"
	"net/http"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Email == "" || input.Password == "" {
		h.respondCode(c, http.StatusBadRequest, "email_required")
		return
	}
	response, err := h.services.Auth.Login(c.Request.Context(), input.Email, input.Password)
	if err != nil {
		h.respondErrorErr(c, http.StatusUnauthorized, "invalid_credentials", err)
		return
	}
	if response.PasswordChangeRequired {
		h.respond(c, http.StatusOK, gin.H{
			"password_change_required": true,
			"password_change_token":    response.PasswordChangeToken,
		})
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) Register(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.Email == "" || input.Name == "" || len(input.Password) < 8 {
		h.respondCode(c, http.StatusBadRequest, "validation_failed")
		return
	}
	response, err := h.services.Auth.Register(c.Request.Context(), input.Email, input.Password, input.Name, input.Role)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "registration_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, response)
}

func (h *Handler) CompleteInitialPasswordChange(c *gin.Context) {
	var input struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Token == "" || len(input.NewPassword) < 8 {
		h.respondCode(c, http.StatusBadRequest, "initial_password_change_invalid")
		return
	}
	response, err := h.services.Auth.CompleteInitialPasswordChange(c.Request.Context(), input.Token, input.NewPassword)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "initial_password_change_invalid", err)
		return
	}
	h.respond(c, http.StatusOK, response)
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
		h.respondErrorErr(c, http.StatusForbidden, "dev_login_failed", err)
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) GoogleLogin(c *gin.Context) {
	var input struct {
		IDToken  string `json:"id_token"`
		DeviceID string `json:"device_id"`
		Role     string `json:"role"`
		Nonce    string `json:"nonce"`
	}
	_ = c.ShouldBindJSON(&input)
	if input.IDToken == "" {
		h.respondCode(c, http.StatusBadRequest, "google_token_required")
		return
	}
	response, err := h.services.Auth.GoogleLogin(c.Request.Context(), input.IDToken, input.DeviceID, input.Role, input.Nonce)
	if err != nil {
		code := "google_verification_failed"
		status := http.StatusUnauthorized
		if errors.Is(err, service.ErrGoogleLinkRequired) {
			code = "google_link_required"
			status = http.StatusConflict
		}
		h.respondErrorErr(c, status, code, err)
		return
	}
	h.respond(c, http.StatusOK, response)
}

func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Email == "" {
		h.respondCode(c, http.StatusBadRequest, "email_required")
		return
	}
	previewCode, err := h.services.Auth.RequestPasswordReset(c.Request.Context(), input.Email)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "password_reset_failed", err)
		return
	}
	data := gin.H{"accepted": true, "expires_in_seconds": 1800}
	if previewCode != "" {
		data["preview_code"] = previewCode
	}
	h.respond(c, http.StatusAccepted, data)
}

func (h *Handler) ConfirmPasswordReset(c *gin.Context) {
	var input struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Email == "" || input.Code == "" || len(input.NewPassword) < 8 {
		h.respondCode(c, http.StatusBadRequest, "password_reset_invalid")
		return
	}
	if err := h.services.Auth.ConfirmPasswordReset(c.Request.Context(), input.Email, input.Code, input.NewPassword); err != nil {
		code := "password_reset_failed"
		if errors.Is(err, service.ErrPasswordResetInvalid) || errors.Is(err, service.ErrPasswordReuse) {
			code = "password_reset_invalid"
		}
		h.respondErrorErr(c, http.StatusBadRequest, code, err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"reset": true})
}

func (h *Handler) LinkGoogle(c *gin.Context) {
	var input struct {
		CurrentPassword string `json:"current_password"`
		IDToken         string `json:"id_token"`
		Nonce           string `json:"nonce"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.CurrentPassword == "" || input.IDToken == "" {
		h.respondCode(c, http.StatusBadRequest, "google_link_failed")
		return
	}
	if err := h.services.Auth.LinkGoogle(c.Request.Context(), h.currentUserID(c), input.CurrentPassword, input.IDToken, input.Nonce); err != nil {
		code := "google_link_failed"
		if errors.Is(err, service.ErrCurrentPasswordInvalid) {
			code = "current_password_invalid"
		}
		h.respondErrorErr(c, http.StatusBadRequest, code, err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"google_linked": true})
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

func (h *Handler) ConfirmEmailVerification(c *gin.Context) {
	var input struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Token == "" {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.Auth.ConfirmEmailVerification(c.Request.Context(), input.Token); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "email_verification_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"verified": true})
}

func (h *Handler) ResendEmailVerification(c *gin.Context) {
	previewURL, err := h.services.Auth.ResendEmailVerification(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "email_verification_delivery_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"sent": true, "preview_url": previewURL})
}

func (h *Handler) BeginPhoneVerification(c *gin.Context) {
	var input struct {
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	previewCode, err := h.services.Auth.BeginPhoneVerification(c.Request.Context(), h.currentUserID(c), input.Phone)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "phone_verification_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"sent": true, "preview_code": previewCode})
}

func (h *Handler) ConfirmPhoneVerification(c *gin.Context) {
	var input struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	if err := h.services.Auth.ConfirmPhoneVerification(c.Request.Context(), h.currentUserID(c), input.Code); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "phone_verification_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"verified": true})
}
