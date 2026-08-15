// Command seed-accounts installs only the four owner-approved demo accounts
// without any content fixtures. It is the production seeding path: production
// runs migrate-up plus this users-only seeder, while staging runs the full
// demo seeder and the baseline content seeders. It fails closed when the
// database contains any account outside the approved fixture.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
)

const seedAccountsConfirmation = "CREATE_FOUR_DEMO_ACCOUNTS"

func main() {
	if os.Getenv("CONFIRM_SEED_ACCOUNTS") != seedAccountsConfirmation {
		log.Fatalf("refusing account seed: set CONFIRM_SEED_ACCOUNTS=%s", seedAccountsConfirmation)
	}

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate configuration: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	if err = seed.ValidateDemoSeedTarget(ctx, client, true); err != nil {
		log.Fatalf("validate account seed target: %v", err)
	}
	if err = seed.SeedUsers(ctx, client); err != nil {
		log.Fatalf("seed demo accounts: %v", err)
	}
	if err = seed.ValidateDemoSeedTarget(ctx, client, false); err != nil {
		log.Fatalf("validate seeded demo accounts: %v", err)
	}
	log.Printf("demo accounts ready: users=%d", len(seed.ExpectedDemoAccounts))
}
