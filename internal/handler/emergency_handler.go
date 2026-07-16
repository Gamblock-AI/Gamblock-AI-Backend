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
	var input struct {
		DeviceID string `json:"device_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.DeviceID == "" {
		h.respondCode(c, http.StatusBadRequest, "device_id_required")
		return
	}
	request, err := h.services.Admin.RequestEmergencyKey(c.Request.Context(), h.currentUserID(c), input.DeviceID)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "emergency_request_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, request)
}

func (h *Handler) CurrentEmergencyKeyRequest(c *gin.Context) {
	request, err := h.services.Admin.GetCurrentEmergencyKeyRequest(
		c.Request.Context(),
		h.currentUserID(c),
		c.Query("device_id"),
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusNotFound, "emergency_request_not_found", err)
		return
	}
	h.respond(c, http.StatusOK, request)
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

func (h *Handler) ReviewEmergencyKeyRequest(c *gin.Context) {
	request, err := h.services.Admin.ReviewEmergencyKeyRequest(
		c.Request.Context(),
		c.Param("id"),
		h.currentUserID(c),
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "emergency_review_failed", err)
		return
	}
	h.respond(c, http.StatusOK, request)
}

func (h *Handler) EmergencyUnlock(c *gin.Context) {
	var input emergencyUnlockInput
	if err := c.ShouldBindJSON(&input); err != nil || input.EmergencyKey == "" {
		h.respondCode(c, http.StatusBadRequest, "emergency_key_required")
		return
	}

	if input.DeviceID == "" {
		h.respondCode(c, http.StatusBadRequest, "device_id_required")
		return
	}
	grant, err := h.services.Admin.ValidateEmergencyKey(c.Request.Context(), input.EmergencyKey, input.DeviceID)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "invalid_key", err)
		return
	}
	h.respond(c, http.StatusOK, grant)
}
