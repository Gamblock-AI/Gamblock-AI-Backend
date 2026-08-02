// Command seed-learning-hub installs the idempotent public Learning Hub
// catalog without creating demo users or activity records.
package main

import (
	"context"
	"log"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	report, err := seed.SeedLearningHubDefaultsWithReport(ctx, client)
	if err != nil {
		log.Fatalf("seed learning hub: %v", err)
	}
	log.Printf("learning hub seed complete: inserted=%d skipped=%d", report.Inserted, report.Skipped)
}
