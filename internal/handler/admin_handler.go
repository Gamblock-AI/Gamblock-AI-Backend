package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (h *Handler) PortalOverview(c *gin.Context) {
	releases, _ := h.services.Admin.GetModelReleases(c.Request.Context())
	rulesets, _ := h.services.Admin.GetRulesetReleases(c.Request.Context())
	supportCount := 0
	cases, err := h.services.Support.GetSupportCases(c.Request.Context())
	if err == nil {
		supportCount = len(cases)
	}
	latestModel := "artifact-v0.3.1"
	if len(releases) > 0 {
		latestModel = releases[0].Version
	}
	latestRuleset := "ruleset-2026.05.1"
	if len(rulesets) > 0 {
		latestRuleset = rulesets[0].Version
	}

	h.respond(c, http.StatusOK, gin.H{
		"protected_users":         128,
		"partner_approvals":       4,
		"healthy_devices_percent": 94,
		"open_support":            supportCount,
		"model_release":           latestModel,
		"ruleset_release":         latestRuleset,
	})
}

func (h *Handler) AdminModules(c *gin.Context) {
	modules, err := h.services.Admin.GetEducationModules(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_modules_failed", err)
		return
	}
	h.respond(c, http.StatusOK, modules)
}

func (h *Handler) CreateAdminModule(c *gin.Context) {
	var input model.EducationModule
	if err := c.ShouldBindJSON(&input); err != nil {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}

	if input.Status == "" {
		input.Status = "published" // Default directly to published for prototyping
	}

	err := h.services.Admin.CreateEducationModule(c.Request.Context(), input)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "create_admin_module_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.ID, "published": true})
}

func (h *Handler) AdminModelReleases(c *gin.Context) {
	releases, err := h.services.Admin.GetModelReleases(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_model_releases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, releases)
}

func (h *Handler) AdminSupportCases(c *gin.Context) {
	cases, err := h.services.Support.GetSupportCases(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "fetch_admin_support_cases_failed", err)
		return
	}
	h.respond(c, http.StatusOK, cases)
}

func (h *Handler) ClientDashboardSummary(c *gin.Context) {
	h.respond(c, http.StatusOK, gin.H{
		"user_name":        "Gading",
		"protection_label": "High",
		"blocked_attempts": 142,
		"active_days":      12,
		"current_streak":   7,
	})
}

func (h *Handler) ClientProtectionStatus(c *gin.Context) {
	releases, _ := h.services.Admin.GetModelReleases(c.Request.Context())
	rulesets, _ := h.services.Admin.GetRulesetReleases(c.Request.Context())
	latestModel := "artifact-v0.3.1"
	if len(releases) > 0 {
		latestModel = releases[0].Version
	}
	latestRuleset := "ruleset-2026.05.1"
	if len(rulesets) > 0 {
		latestRuleset = rulesets[0].Version
	}

	h.respond(c, http.StatusOK, gin.H{
		"mode":            "Active",
		"runtime_status":  "Local runtime ready",
		"ruleset_version": latestRuleset,
		"model_version":   latestModel,
		"last_sync":       "API sync: 2 minutes ago",
	})
}

func (h *Handler) ClientProgressSnapshot(c *gin.Context) {
	h.respond(c, http.StatusOK, gin.H{
		"weekly_blocks": []int{3, 1, 4, 2, 0, 5, 3},
		"moods":         []string{"Calm", "Tense", "Focused", "Tired"},
		"active_days":   12,
		"reflections":   7,
	})
}

type releaseInput struct {
	Version         string         `json:"version"`
	Platform        string         `json:"platform"`
	ArtifactPath    string         `json:"artifact_path"`
	SHA256          string         `json:"sha256"`
	ContractVersion string         `json:"contract_version"`
	Threshold       float64        `json:"threshold"`
	Metrics         map[string]any `json:"metrics"`
	Rules           map[string]any `json:"rules"`
}

func (h *Handler) CreateModelRelease(c *gin.Context) {
	var input releaseInput
	_ = c.ShouldBindJSON(&input)
	defaultsForRelease(&input, "model", "artifacts/model/fallback.onnx")

	err := h.services.Admin.CreateModelRelease(
		c.Request.Context(),
		input.Version,
		input.Platform,
		input.ArtifactPath,
		input.SHA256,
		input.ContractVersion,
		input.Threshold,
		input.Metrics,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_model_release_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "published": true})
}

func (h *Handler) CreateRulesetRelease(c *gin.Context) {
	var input releaseInput
	_ = c.ShouldBindJSON(&input)
	defaultsForRelease(&input, "ruleset", "artifacts/ruleset/default.json")

	err := h.services.Admin.CreateRulesetRelease(
		c.Request.Context(),
		input.Version,
		input.ArtifactPath,
		input.SHA256,
		input.Rules,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_ruleset_release_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "published": true})
}

func (h *Handler) CreateNetworkRulesetRelease(c *gin.Context) {
	var input releaseInput
	_ = c.ShouldBindJSON(&input)
	defaultsForRelease(&input, "network-rulesets", "artifacts/network/dns-blocklist.txt")

	err := h.services.Admin.CreateNetworkRulesetRelease(
		c.Request.Context(),
		input.Version,
		input.ArtifactPath,
		input.SHA256,
		input.Rules,
	)
	if err != nil {
		h.respondErrorErr(c, http.StatusBadRequest, "create_network_release_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "published": true})
}

func (h *Handler) DownloadModelRelease(c *gin.Context) {
	h.downloadArtifact(c, "model", c.Param("version"))
}

func (h *Handler) DownloadRulesetRelease(c *gin.Context) {
	h.downloadArtifact(c, "ruleset", c.Param("version"))
}

func (h *Handler) DownloadNetworkRulesetRelease(c *gin.Context) {
	h.downloadArtifact(c, "network-rulesets", c.Param("version"))
}

func (h *Handler) LatestModelRelease(c *gin.Context) {
	h.latestRelease(c, "model")
}

func (h *Handler) LatestRulesetRelease(c *gin.Context) {
	h.latestRelease(c, "ruleset")
}

func (h *Handler) LatestNetworkRulesetRelease(c *gin.Context) {
	h.latestRelease(c, "network-rulesets")
}

func (h *Handler) downloadArtifact(c *gin.Context, kind, version string) {
	artifact := demoArtifactBytes(kind, version)
	c.Header("X-Artifact-Version", version)
	c.Header("X-Artifact-Kind", kind)
	c.Header("X-Artifact-SHA256", demoArtifactSHA256(kind, version))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/octet-stream", artifact)
}

func (h *Handler) latestRelease(c *gin.Context, kind string) {
	var release model.Release
	var found bool
	
	switch kind {
	case "model":
		list, _ := h.services.Admin.GetModelReleases(c.Request.Context())
		if len(list) > 0 {
			release = list[0]
			found = true
		}
	case "ruleset":
		list, _ := h.services.Admin.GetRulesetReleases(c.Request.Context())
		if len(list) > 0 {
			release = list[0]
			found = true
		}
	case "network-rulesets":
		list, _ := h.services.Admin.GetNetworkRulesets(c.Request.Context())
		if len(list) > 0 {
			release = list[0]
			found = true
		}
	}

	if !found {
		h.respondCode(c, http.StatusNotFound, "release_not_found")
		return
	}

	h.respond(c, http.StatusOK, gin.H{
		"id":               release.ID,
		"version":          release.Version,
		"platform":         release.Platform,
		"sha256":           demoArtifactSHA256(kind, release.Version),
		"status":           release.Status,
		"download_url":     release.DownloadURL,
		"size_bytes":       len(demoArtifactBytes(kind, release.Version)),
		"contract_version": "v1",
		"threshold":        0.72,
		"metrics":          release.Metrics,
		"expires_at":       time.Now().UTC().Add(15 * time.Minute),
	})
}

func defaultsForRelease(input *releaseInput, version, artifactPath string) {
	if input.Version == "" {
		input.Version = version + "-" + uuid.NewString()[:6]
	}
	if input.ArtifactPath == "" {
		input.ArtifactPath = artifactPath
	}
	if input.SHA256 == "" {
		input.SHA256 = "pending-checksum"
	}
	if input.ContractVersion == "" {
		input.ContractVersion = "v1"
	}
	if input.Threshold <= 0 {
		input.Threshold = 0.72
	}
	if input.Metrics == nil {
		input.Metrics = map[string]any{"source": "admin"}
	}
	if input.Rules == nil {
		input.Rules = map[string]any{"source": "admin"}
	}
}

func demoArtifactBytes(kind, version string) []byte {
	return []byte(fmt.Sprintf("Demo binary payload for kind: %s, version: %s", kind, version))
}

func demoArtifactSHA256(kind, version string) string {
	sum := sha256.Sum256(demoArtifactBytes(kind, version))
	return hex.EncodeToString(sum[:])
}
