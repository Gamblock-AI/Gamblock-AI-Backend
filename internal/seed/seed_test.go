package seed

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	_ "modernc.org/sqlite" // registers the "sqlite" driver (CGO-free)
)

// Open an ent client backed by modernc sqlite (in-memory, CGO-free).
func openSQLiteEnt(t *testing.T) *ent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:seedtest?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	// Ensure foreign keys on (modernc is strict about this for ent schema with FKs).
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		// non-fatal on some configs
		_ = err
	}
	drv := entsql.OpenDB("sqlite3", db) // ent sqlite dialect name
	client := ent.NewClient(ent.Driver(drv))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

// Seed is idempotent: running it twice must not duplicate users.
func TestSeed_Idempotent(t *testing.T) {
	client := openSQLiteEnt(t)
	defer client.Close()
	ctx := context.Background()

	require.NoError(t, Seed(ctx, client))
	first, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	require.Greater(t, first, 0)

	require.NoError(t, Seed(ctx, client))
	second, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, first, second, "second seed must not duplicate users")
}
