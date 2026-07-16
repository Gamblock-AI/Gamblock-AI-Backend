package handler

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

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
