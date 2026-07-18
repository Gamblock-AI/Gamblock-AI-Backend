package db

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
)

func Open(databaseURL string) (*ent.Client, func() error, error) {
	if databaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL is empty")
	}
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, nil, err
	}
	driver := entsql.OpenDB(dialect.Postgres, sqlDB)
	return ent.NewClient(ent.Driver(driver)), sqlDB.Close, nil
}

func Migrate(ctx context.Context, client *ent.Client) error {
	return client.Schema.Create(ctx)
}

func Seed(ctx context.Context, client *ent.Client, mediaPath ...string) error {
	return seed.Seed(ctx, client, mediaPath...)
}
