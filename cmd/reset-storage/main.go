// Command reset-storage empties the runtime storage directories (media,
// avatars, exports, and release artifacts). It is meant to run together with a
// destructive database reset (migrate-fresh / migrate-down), where every
// dynamic file is orphaned by definition. The directories themselves are
// recreated so the API can keep writing to them.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
)

// resetDir removes the directory contents and recreates the directory. It
// refuses empty or filesystem-root paths to avoid wiping the wrong location.
func resetDir(dir string) error {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return nil
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return err
	}
	if abs == "/" {
		return nil
	}
	if err := os.RemoveAll(abs); err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return err
	}
	log.Printf("reset-storage: cleared %s", abs)
	return nil
}

func main() {
	cfg := config.Load()
	for _, dir := range []string{
		cfg.MediaStoragePath,
		cfg.AvatarStoragePath,
		cfg.ExportStoragePath,
		cfg.ArtifactStoragePath,
	} {
		if err := resetDir(dir); err != nil {
			log.Fatalf("reset storage %q: %v", dir, err)
		}
	}
	log.Println("dynamic storage directories reset")
}
