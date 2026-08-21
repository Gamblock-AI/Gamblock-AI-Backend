// Command seed-scale populates all database tables with 500 to 2000 realistic records
// for high-volume local testing and benchmarking. It is strictly for local environments
// and is isolated from staging and production seeder pipelines.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seedscale"
)

func main() {
	countFlag := flag.Int("count", 600, "Base number of records per table (500-2000)")
	flag.Parse()

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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

	start := time.Now()
	log.Printf("starting scale seeding (target: %d rows per table)...", *countFlag)

	reports, err := seedscale.SeedScaleDatabase(ctx, client, seedscale.ScaleSeedOptions{
		BaseCount: *countFlag,
	})
	if err != nil {
		log.Fatalf("scale seeding failed: %v", err)
	}

	totalRows := 0
	log.Println("--------------------------------------------------")
	log.Println("Scale Seeding Summary:")
	log.Println("--------------------------------------------------")
	for _, r := range reports {
		log.Printf("  • %-30s : %5d rows", r.TableName, r.Count)
		totalRows += r.Count
	}
	log.Println("--------------------------------------------------")
	log.Printf("Scale seeding complete! Total: %d records across %d tables in %s", totalRows, len(reports), time.Since(start).Round(time.Millisecond))
}
