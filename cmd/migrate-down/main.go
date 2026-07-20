package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
)

func main() {
	if os.Getenv("CONFIRM_MIGRATE_DOWN") != "DROP_ALL_DATA" {
		log.Fatal("refusing destructive migration: set CONFIRM_MIGRATE_DOWN=DROP_ALL_DATA")
	}

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.DropPublicSchema(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("drop database schema: %v", err)
	}
	log.Println("database public schema dropped and recreated")
}
