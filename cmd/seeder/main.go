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

	if err = db.Migrate(ctx, client); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	report, err := seed.SeedProductionDefaultsWithReport(ctx, client, cfg.MediaStoragePath)
	if err != nil {
		log.Fatalf("seed production defaults: %v", err)
	}
	log.Printf("production database defaults ready: learning_hub_inserted=%d learning_hub_skipped=%d", report.Inserted, report.Skipped)
}
