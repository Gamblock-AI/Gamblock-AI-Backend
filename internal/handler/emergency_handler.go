package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type emergencyUnlockInput struct {
	EmergencyKey string `json:"emergency_key" binding:"required"`
	DeviceID     string `json:"device_id"`
}

func (h *Handler) GenerateEmergencyKey(c *gin.Context) {
	key, err := h.services.Admin.GenerateEmergencyKey(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "generate_key_failed", err.Error())
		return
	}
	h.respond(c, http.StatusCreated, gin.H{
		"emergency_key": key,
		"expires_in":    "24 jam",
		"single_use":    true,
	})
}

func (h *Handler) EmergencyUnlock(c *gin.Context) {
	var input emergencyUnlockInput
	if err := c.ShouldBindJSON(&input); err != nil || input.EmergencyKey == "" {
		h.respondError(c, http.StatusBadRequest, "emergency_key_required", "Kunci darurat diperlukan")
		return
	}

	err := h.services.Admin.ValidateEmergencyKey(c.Request.Context(), input.EmergencyKey, input.DeviceID)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "invalid_key", err.Error())
		return
	}
	h.respond(c, http.StatusOK, gin.H{
		"unlocked": true,
		"message":  "Perangkat berhasil dibuka. Kunci darurat hanya berlaku satu kali.",
	})
}
