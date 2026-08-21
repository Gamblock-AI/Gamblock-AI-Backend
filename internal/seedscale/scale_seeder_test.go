package seedscale

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/datarequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/emergencykeyrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/intention"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationmodule"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryspace"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/spkpreference"
	_ "modernc.org/sqlite"
)

func openTestSQLiteEnt(t *testing.T) *ent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:seedscaletest?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestSeedLocalAccounts(t *testing.T) {
	client := openTestSQLiteEnt(t)
	defer client.Close()
	ctx := context.Background()

	require.NoError(t, SeedLocalAccounts(ctx, client))
	count, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(CoreLocalAccounts), count)

	// Test idempotency
	require.NoError(t, SeedLocalAccounts(ctx, client))
	count2, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, count, count2)
}

func TestSeedScaleDatabase_PopulatesAllTables(t *testing.T) {
	client := openTestSQLiteEnt(t)
	defer client.Close()
	ctx := context.Background()

	// Seed with a base count of 500 for testing
	reports, err := SeedScaleDatabase(ctx, client, ScaleSeedOptions{BaseCount: 500})
	require.NoError(t, err)
	assert.Len(t, reports, 49, "must report all 49 tables")

	for _, r := range reports {
		if r.TableName == "SiteSocialLink" {
			assert.Equal(t, 8, r.Count, "SiteSocialLink must have 8 unique platforms")
		} else {
			assert.GreaterOrEqual(t, r.Count, 500, "table %s should have >= 500 records", r.TableName)
			assert.LessOrEqual(t, r.Count, 2000, "table %s should have <= 2000 records", r.TableName)
		}
	}

	// Verify core developer accounts have relational records
	for _, uid := range []string{"usr_gading", "usr_dery", "usr_suci", "usr_nasywa"} {
		// Device
		devCount, err := client.Device.Query().Where(device.UserIDEQ(uid)).Count(ctx)
		require.NoError(t, err)
		assert.Greater(t, devCount, 0, "user %s must have a device", uid)

		// Intention
		intExists, err := client.Intention.Query().Where(intention.UserIDEQ(uid)).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, intExists, "user %s must have an intention", uid)

		// RecoverySpace
		rspExists, err := client.RecoverySpace.Query().Where(recoveryspace.UserIDEQ(uid)).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, rspExists, "user %s must have a recovery space", uid)

		// SpkPreference
		spkExists, err := client.SpkPreference.Query().Where(spkpreference.UserIDEQ(uid)).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, spkExists, "user %s must have an SPK preference", uid)
	}

	// Verify realistic status variations exist for UI dashboards
	reviewModules, err := client.PsychoeducationModule.Query().Where(psychoeducationmodule.StatusEQ(psychoeducationmodule.StatusInReview)).Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, reviewModules, 0, "must have modules in review")

	pendingEmergency, err := client.EmergencyKeyRequest.Query().Where(emergencykeyrequest.StatusEQ(emergencykeyrequest.StatusPending)).Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, pendingEmergency, 0, "must have pending emergency key requests")

	failedDataRequests, err := client.DataRequest.Query().Where(datarequest.StatusEQ(datarequest.StatusFailed)).Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, failedDataRequests, 0, "must have failed data requests")

	interventions, err := client.AggregateEvent.Query().Where(aggregateevent.EventTypeEQ(aggregateevent.EventTypeInterventionShown)).Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, interventions, 0, "must have intervention events")
}
