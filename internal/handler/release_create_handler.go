package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateModelRelease(c *gin.Context) {
	var input releaseInput
	if err := c.ShouldBindJSON(&input); err != nil || h.validateReleaseInput(&input, true) != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}
	if err := h.services.Admin.CreateModelRelease(
		c.Request.Context(),
		input.Version,
		input.Platform,
		input.ArtifactPath,
		input.SHA256,
		input.ContractVersion,
		input.Threshold,
		input.Metrics,
	); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_model_release_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "status": "validated"})
}

func (h *Handler) CreateRulesetRelease(c *gin.Context) {
	var input releaseInput
	if err := c.ShouldBindJSON(&input); err != nil || h.validateReleaseInput(&input, false) != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}
	if err := h.services.Admin.CreateRulesetRelease(
		c.Request.Context(),
		input.Version,
		input.ArtifactPath,
		input.SHA256,
		input.Rules,
	); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_ruleset_release_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "status": "validated"})
}

func (h *Handler) CreateNetworkRulesetRelease(c *gin.Context) {
	var input releaseInput
	if err := c.ShouldBindJSON(&input); err != nil || h.validateReleaseInput(&input, false) != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}
	if err := h.services.Admin.CreateNetworkRulesetRelease(
		c.Request.Context(),
		input.Version,
		input.ArtifactPath,
		input.SHA256,
		input.Rules,
	); err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_network_release_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "status": "validated"})
}
