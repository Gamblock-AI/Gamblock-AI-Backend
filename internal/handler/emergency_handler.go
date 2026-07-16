package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type emergencyUnlockInput struct {
	EmergencyKey string `json:"emergency_key" binding:"required"`
	DeviceID     string `json:"device_id"`
}

func (h *Handler) RequestEmergencyKey(c *gin.Context) {
	request, err := h.services.Admin.RequestEmergencyKey(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "generate_key_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, request)
}

func (h *Handler) PendingEmergencyKeyRequests(c *gin.Context) {
	requests, err := h.services.Admin.GetPendingEmergencyKeyRequests(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "generate_key_failed", err)
		return
	}
	h.respond(c, http.StatusOK, requests)
}

func (h *Handler) ApproveEmergencyKeyRequest(c *gin.Context) {
	request, key, err := h.services.Admin.ApproveEmergencyKeyRequest(
		c.Request.Context(),
		c.Param("id"),
		h.currentUserID(c),
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "generate_key_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{
		"request":       request,
		"emergency_key": key,
		"expires_in":    "24 hours",
		"single_use":    true,
	})
}

func (h *Handler) EmergencyUnlock(c *gin.Context) {
	var input emergencyUnlockInput
	if err := c.ShouldBindJSON(&input); err != nil || input.EmergencyKey == "" {
		h.respondCode(c, http.StatusBadRequest, "emergency_key_required")
		return
	}

	err := h.services.Admin.ValidateEmergencyKey(c.Request.Context(), input.EmergencyKey, input.DeviceID)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "invalid_key", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{
		"unlocked": true,
		"message":  "Perangkat berhasil dibuka. Kunci darurat hanya berlaku satu kali.",
	})
}
