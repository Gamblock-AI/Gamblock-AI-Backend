package db

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

// LoadStore rebuilds the in-memory cache from ent. Required records fail the
// load; recovery and analytics records remain best-effort for backwards
// compatible databases that do not yet have every optional table.
func LoadStore(ctx context.Context, client *ent.Client) (*store.Store, error) {
	users, err := client.User.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return store.New(), nil
	}

	out := store.New()
	if err := loadIdentityStore(ctx, client, out, users); err != nil {
		return nil, err
	}
	if err := loadReleaseStore(ctx, client, out); err != nil {
		return nil, err
	}
	if err := loadOperationsStore(ctx, client, out); err != nil {
		return nil, err
	}
	loadRecoveryStore(ctx, client, out)
	return out, nil
}
