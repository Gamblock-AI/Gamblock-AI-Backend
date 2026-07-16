package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	_ "modernc.org/sqlite" // CGO-free sqlite driver
)

func openSQLite(t *testing.T) *ent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:dbtest?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = err
	}
	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestMigrate_Idempotent(t *testing.T) {
	client := openSQLite(t)
	defer client.Close()
	ctx := context.Background()
	// Migrate runs schema create (already done in openSQLite); re-running must not error.
	require.NoError(t, Migrate(ctx, client))
}

func TestSeedAndLoadStore(t *testing.T) {
	client := openSQLite(t)
	defer client.Close()
	ctx := context.Background()

	require.NoError(t, Seed(ctx, client))

	st, err := LoadStore(ctx, client)
	require.NoError(t, err)
	require.NotNil(t, st)
	assert.NotEmpty(t, st.Users)
	assert.NotEmpty(t, st.Devices)
}

// --- pure helpers ---

func TestHumanExpiry(t *testing.T) {
	assert.Contains(t, humanExpiry(time.Now().Add(20*time.Minute)), "Expires")
	assert.Contains(t, humanExpiry(time.Now().Add(-time.Hour)), "Reviewed")
}

func TestHumanPublished(t *testing.T) {
	now := time.Now()
	assert.Equal(t, "Published", humanPublished(&now))
	assert.Equal(t, "Not published", humanPublished(nil))
}

func TestHumanApprovalStatus(t *testing.T) {
	assert.Equal(t, "Pending partner approval", humanApprovalStatus("pending"))
	assert.Equal(t, "Approved", humanApprovalStatus("approved"))
	assert.Equal(t, "weird", humanApprovalStatus("weird"))
}

func TestHumanApprovalAction(t *testing.T) {
	assert.Contains(t, humanApprovalAction("pause_protection", 15), "Pause protection")
	assert.Equal(t, "Permission revoked detected", humanApprovalAction("uninstall_detected", 0))
	assert.Equal(t, "other", humanApprovalAction("other", 0))
}

func TestHumanDataRequestTitle(t *testing.T) {
	assert.Equal(t, "Export account data", humanDataRequestTitle("export"))
	assert.Equal(t, "Delete archived support notes", humanDataRequestTitle("delete"))
	assert.Equal(t, "Data request", humanDataRequestTitle("x"))
}

func TestValueAndEnsureDefaults(t *testing.T) {
	assert.Equal(t, "", value(nil))
	s := "x"
	assert.Equal(t, "x", value(&s))
}
