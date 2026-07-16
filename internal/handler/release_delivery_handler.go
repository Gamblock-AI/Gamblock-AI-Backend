package handler

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

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
