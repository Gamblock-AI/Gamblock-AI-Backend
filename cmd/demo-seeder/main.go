// Command demo-seeder installs the four owner-approved demo accounts and
// their fixture data. It is intentionally separate from the production-safe
// seeder used by automatic deployments.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
)

const demoSeedConfirmation = "CREATE_FOUR_DEMO_ACCOUNTS"

type expectedDemoAccount struct {
	email string
	role  entuser.Role
}

var expectedDemoAccounts = map[string]expectedDemoAccount{
	"usr_gading": {email: "gading@gmail.com", role: entuser.RoleUser},
	"usr_dery":   {email: "dery@gmail.com", role: entuser.RoleUser},
	"usr_suci":   {email: "suci@gmail.com", role: entuser.RolePartner},
	"usr_nasywa": {email: "nasywa@gmail.com", role: entuser.RoleAdmin},
}

func validateDemoSeedTarget(ctx context.Context, client *ent.Client, allowEmpty bool) error {
	rows, err := client.User.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("query demo seed target: %w", err)
	}
	if len(rows) == 0 && allowEmpty {
		return nil
	}
	if len(rows) != len(expectedDemoAccounts) {
		return fmt.Errorf("demo seeder requires an empty database or exactly four known demo accounts")
	}
	for _, row := range rows {
		expected, ok := expectedDemoAccounts[row.ID]
		if !ok || row.Email != expected.email || row.Role != expected.role {
			return fmt.Errorf("database contains an account outside the approved demo fixture")
		}
	}
	return nil
}

func main() {
	if os.Getenv("CONFIRM_DEMO_SEED") != demoSeedConfirmation {
		log.Fatalf("refusing demo seed: set CONFIRM_DEMO_SEED=%s", demoSeedConfirmation)
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
	if err = validateDemoSeedTarget(ctx, client, true); err != nil {
		log.Fatalf("validate demo seed target: %v", err)
	}
	if err = db.Seed(ctx, client, cfg.MediaStoragePath); err != nil {
		log.Fatalf("seed demo database: %v", err)
	}
	if err = validateDemoSeedTarget(ctx, client, false); err != nil {
		log.Fatalf("validate seeded demo accounts: %v", err)
	}
	log.Printf("demo database ready: users=%d", len(expectedDemoAccounts))
}
