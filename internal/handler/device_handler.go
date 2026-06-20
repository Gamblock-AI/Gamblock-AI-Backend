package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateDevice(c *gin.Context) {
	var input struct {
		Platform       string `json:"platform"`
		Label          string `json:"label"`
		AppVersion     string `json:"app_version"`
		OSVersion      string `json:"os_version"`
		ModelVersion   string `json:"model_version"`
		RulesetVersion string `json:"ruleset_version"`
	}
	_ = c.ShouldBindJSON(&input)
	
	var modelVal, rulesetVal *string
	if input.ModelVersion != "" {
		modelVal = &input.ModelVersion
	}
	if input.RulesetVersion != "" {
		rulesetVal = &input.RulesetVersion
	}

	dev, err := h.services.Device.CreateDevice(
		c.Request.Context(),
		h.currentUserID(c),
		input.Platform,
		input.Label,
		input.AppVersion,
		input.OSVersion,
		modelVal,
		rulesetVal,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "device_create_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": dev.ID, "registered": true, "status": dev.ProtectionStatus})
}

func (h *Handler) UpdateDevice(c *gin.Context) {
	var input struct {
		Label            string `json:"label"`
		AppVersion       string `json:"app_version"`
		OSVersion        string `json:"os_version"`
		ProtectionStatus string `json:"protection_status"`
		ModelVersion     string `json:"model_version"`
		RulesetVersion   string `json:"ruleset_version"`
	}
	_ = c.ShouldBindJSON(&input)

	dev, err := h.services.Device.UpdateDevice(
		c.Request.Context(),
		c.Param("device_id"),
		input.Label,
		input.AppVersion,
		input.OSVersion,
		input.ProtectionStatus,
		input.ModelVersion,
		input.RulesetVersion,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "device_update_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"id": dev.ID, "updated": true, "status": dev.ProtectionStatus})
}

func (h *Handler) DeviceHeartbeat(c *gin.Context) {
	deviceID := c.Param("device_id")
	err := h.services.Device.RecordHeartbeat(c.Request.Context(), deviceID)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "heartbeat_failed", err)
		return
	}
	h.respond(c, http.StatusOK, gin.H{"heartbeat": "ok", "device_id": deviceID})
}
