package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (h *Handler) PortalOverview(c *gin.Context) {
	overview, err := h.services.Admin.GetPortalOverview(c.Request.Context())
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "portal_overview_failed", err)
		return
	}
	h.respond(c, http.StatusOK, overview)
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

	if strings.TrimSpace(input.Slug) == "" || strings.TrimSpace(input.Title) == "" ||
		strings.TrimSpace(input.Summary) == "" || strings.TrimSpace(input.BodyMarkdown) == "" ||
		input.EstimatedMinutes < 1 {
		h.respondCode(c, http.StatusBadRequest, "err_validation")
		return
	}
	input.Status = "draft"
	if input.ID == "" {
		input.ID = "mod_" + uuid.NewString()[:8]
	}

	err := h.services.Admin.CreateEducationModule(c.Request.Context(), input)
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "create_admin_module_failed", err)
		return
	}
	h.respond(c, http.StatusCreated, gin.H{"id": input.ID, "status": "draft"})
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
	summary, _, _, err := h.services.Client.Dashboard(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "dashboard_summary_failed", err)
		return
	}
	h.respond(c, http.StatusOK, summary)
}

func (h *Handler) ClientProtectionStatus(c *gin.Context) {
	_, protection, _, err := h.services.Client.Dashboard(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "protection_status_failed", err)
		return
	}
	h.respond(c, http.StatusOK, protection)
}

func (h *Handler) ClientProgressSnapshot(c *gin.Context) {
	_, _, progress, err := h.services.Client.Dashboard(c.Request.Context(), h.currentUserID(c))
	if err != nil {
		h.respondErrorErr(c, http.StatusInternalServerError, "progress_snapshot_failed", err)
		return
	}
	h.respond(c, http.StatusOK, progress)
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
	if err := c.ShouldBindJSON(&input); err != nil || h.validateReleaseInput(&input, true) != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}

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
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "status": "validated"})
}

func (h *Handler) CreateRulesetRelease(c *gin.Context) {
	var input releaseInput
	if err := c.ShouldBindJSON(&input); err != nil || h.validateReleaseInput(&input, false) != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}

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
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "status": "validated"})
}

func (h *Handler) CreateNetworkRulesetRelease(c *gin.Context) {
	var input releaseInput
	if err := c.ShouldBindJSON(&input); err != nil || h.validateReleaseInput(&input, false) != nil {
		h.respondCode(c, http.StatusBadRequest, "release_validation_failed")
		return
	}

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
	h.respond(c, http.StatusCreated, gin.H{"id": input.Version, "status": "validated"})
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
	release, found := h.findRelease(c, kind, version)
	if !found || release.Status != "published" {
		h.respondCode(c, http.StatusNotFound, "release_not_found")
		return
	}
	path, err := h.safeArtifactPath(release.ArtifactPath)
	if err != nil {
		h.respondCode(c, http.StatusNotFound, "release_not_found")
		return
	}
	artifact, err := os.ReadFile(path)
	if err != nil || sha256Hex(artifact) != strings.ToLower(release.SHA256) {
		h.respondCode(c, http.StatusServiceUnavailable, "artifact_unavailable")
		return
	}
	c.Header("X-Artifact-Version", version)
	c.Header("X-Artifact-Kind", kind)
	c.Header("X-Artifact-SHA256", release.SHA256)
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/octet-stream", artifact)
}

func (h *Handler) latestRelease(c *gin.Context, kind string) {
	release, found := h.findRelease(c, kind, "")
	if !found {
		h.respondCode(c, http.StatusNotFound, "release_not_found")
		return
	}

	size := int64(0)
	if path, err := h.safeArtifactPath(release.ArtifactPath); err == nil {
		if info, statErr := os.Stat(path); statErr == nil {
			size = info.Size()
		}
	}
	h.respond(c, http.StatusOK, gin.H{
		"id":               release.ID,
		"version":          release.Version,
		"platform":         release.Platform,
		"sha256":           release.SHA256,
		"status":           release.Status,
		"download_url":     release.DownloadURL,
		"size_bytes":       size,
		"contract_version": release.ContractVersion,
		"threshold":        release.Threshold,
		"metrics":          release.Metrics,
	})
}

func (h *Handler) validateReleaseInput(input *releaseInput, modelRelease bool) error {
	input.Version = strings.TrimSpace(input.Version)
	input.ArtifactPath = strings.TrimSpace(input.ArtifactPath)
	input.SHA256 = strings.ToLower(strings.TrimSpace(input.SHA256))
	if input.Version == "" || input.ArtifactPath == "" || len(input.SHA256) != 64 {
		return fmt.Errorf("version, artifact path, and SHA-256 are required")
	}
	if _, err := hex.DecodeString(input.SHA256); err != nil {
		return fmt.Errorf("invalid SHA-256")
	}
	if input.ContractVersion == "" {
		input.ContractVersion = "v1"
	}
	if modelRelease && (input.Platform == "" || input.Threshold <= 0 || input.Threshold > 1) {
		return fmt.Errorf("platform and threshold are required")
	}
	if input.Metrics == nil {
		input.Metrics = map[string]any{}
	}
	if input.Rules == nil {
		input.Rules = map[string]any{}
	}
	path, err := h.safeArtifactPath(input.ArtifactPath)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil || sha256Hex(content) != input.SHA256 {
		return fmt.Errorf("artifact missing or checksum mismatch")
	}
	return nil
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (h *Handler) safeArtifactPath(relative string) (string, error) {
	root, err := filepath.Abs(h.cfg.ArtifactStoragePath)
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(root, relative))
	if err != nil || (candidate != root && !strings.HasPrefix(candidate, root+string(os.PathSeparator))) {
		return "", fmt.Errorf("artifact path escapes storage root")
	}
	return candidate, nil
}

func (h *Handler) findRelease(c *gin.Context, kind, version string) (model.Release, bool) {
	var list []model.Release
	switch kind {
	case "model":
		list, _ = h.services.Admin.GetModelReleases(c.Request.Context())
	case "ruleset":
		list, _ = h.services.Admin.GetRulesetReleases(c.Request.Context())
	case "network-rulesets":
		list, _ = h.services.Admin.GetNetworkRulesets(c.Request.Context())
	}
	for _, release := range list {
		if release.Status == "published" && (version == "" || release.Version == version) {
			return release, true
		}
	}
	return model.Release{}, false
}
