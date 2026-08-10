// Command reset-storage empties the runtime storage directories (media,
// avatars, and exports). It is meant to run together with a
// destructive database reset (migrate-fresh / migrate-down), where every
// dynamic file is orphaned by definition. The directories themselves are
// recreated so the API can keep writing to them.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
)

const resetStorageConfirmation = "DELETE_DYNAMIC_STORAGE"

// resetDir removes the directory contents and recreates the directory. It
// refuses empty or filesystem-root paths to avoid wiping the wrong location.
func resetDir(dir string) error {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return fmt.Errorf("storage path is empty")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return err
	}
	if abs == "/" {
		return fmt.Errorf("refusing to reset filesystem root")
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
	if os.Getenv("CONFIRM_RESET_STORAGE") != resetStorageConfirmation {
		log.Fatalf("refusing storage reset: set CONFIRM_RESET_STORAGE=%s", resetStorageConfirmation)
	}
	cfg := config.Load()
	for _, dir := range []string{
		cfg.MediaStoragePath,
		cfg.AvatarStoragePath,
		cfg.ExportStoragePath,
	} {
		if err := resetDir(dir); err != nil {
			log.Fatalf("reset storage %q: %v", dir, err)
		}
	}
	log.Println("dynamic storage directories reset")
}
