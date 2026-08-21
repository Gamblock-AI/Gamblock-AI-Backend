// Command seed-local-accounts installs only the core accounts for local zero-data testing.
// It is isolated from production/staging seeder assertions.
package main

import (
	"context"
	"log"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seedscale"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	client, closeDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		if err := closeDB(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := db.Migrate(ctx, client); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	log.Println("seeding clean local accounts...")
	if err := seedscale.SeedLocalAccounts(ctx, client); err != nil {
		log.Fatalf("seed local accounts failed: %v", err)
	}

	log.Printf("clean local accounts ready (users: %d)", len(seedscale.CoreLocalAccounts))
}
