package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func TestDevice_CreateUpdateHeartbeat(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	modelVersion, rulesetVersion := "artifact-v0.3.1", "ruleset-2026.05.1"
	device, err := repo.CreateDevice(ctx, "dev_test", "usr_gading", "instance-repo-test", "windows", "PC", "1.0.0", "Win11", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "dev_test", device.ID)

	updated, err := repo.UpdateDevice(ctx, "dev_test", "PC2", "1.0.1", "Win11", "active", modelVersion, rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "PC2", updated.Label)
	assert.NoError(t, repo.RecordHeartbeat(ctx, "dev_test"))
}

func TestProtectionAnalytics_AggregatesOwnedDeviceOnly(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	today := time.Now().UTC()
	for index, eventType := range []string{"block_count_sync", "intervention_shown", "tamper_detected", "permission_revoked"} {
		_, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{
			ID: "agg_test_" + eventType, UserID: "usr_gading", DeviceID: "dev_android",
			IdempotencyKey: "analytics-" + eventType, EventType: eventType, EventDate: today, Count: index + 1,
		})
		require.NoError(t, err)
	}
	analytics, err := repo.GetProtectionAnalytics(ctx, "usr_gading", "dev_android", 7, today)
	require.NoError(t, err)
	assert.Equal(t, 1, analytics.Totals.Blocked)
	assert.Equal(t, 2, analytics.Totals.Interventions)
	assert.Equal(t, 3, analytics.Totals.TamperEvents)
	assert.Equal(t, 4, analytics.Totals.PermissionRevoked)

	_, err = repo.GetProtectionAnalytics(ctx, "usr_suci", "dev_android", 7, today)
	require.Error(t, err)
}

func TestDevice_UpdateNonexistent(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.UpdateDevice(context.Background(), "dev_nonexistent", "L", "1", "OS", "active", "m", "r")
	_ = err
}
