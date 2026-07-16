package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
