//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reflection"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const integrationJournalKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func openPostgres(t *testing.T) (*ent.Client, *repository.Repository) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}

	client, closeDB, err := db.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = closeDB()
	})
	require.NoError(t, db.Migrate(context.Background(), client))
	return client, repository.New(client, store.New())
}

func createIntegrationUser(t *testing.T, client *ent.Client) *ent.User {
	t.Helper()
	now := time.Now().UTC()
	user, err := client.User.Create().
		SetID("integration-" + uuid.NewString()).
		SetEmail("integration-" + uuid.NewString() + "@test.local").
		SetDisplayName("Integration Test").
		SetRole(entuser.RoleUser).
		SetEmailVerifiedAt(now).
		SetPhoneE164("+628123456789").
		SetPhoneVerifiedAt(now).
		Save(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = client.Reflection.Delete().Where(reflection.UserIDEQ(user.ID)).Exec(ctx)
		_, _ = client.AggregateEvent.Delete().Where(aggregateevent.UserIDEQ(user.ID)).Exec(ctx)
		_ = client.User.DeleteOneID(user.ID).Exec(ctx)
	})
	return user
}

func TestPostgresMigrationAndEncryptedPersistence(t *testing.T) {
	client, repo := openPostgres(t)
	ctx := context.Background()
	// Migrate must be safe to run on an already-initialized production schema.
	require.NoError(t, db.Migrate(ctx, client))
	user := createIntegrationUser(t, client)

	reflectionService := service.NewReflectionService(repo, config.Config{JournalEncryptionKey: integrationJournalKey}, zap.NewNop())
	plain := "integration journal must remain encrypted at rest"
	created, err := reflectionService.CreateReflection(ctx, user.ID, plain, "calm")
	require.NoError(t, err)
	persisted, err := client.Reflection.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.NotEqual(t, plain, persisted.ContentEncrypted)

	entries, err := reflectionService.GetReflections(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, plain, entries[0].Text)
}

func TestPostgresTransactionRollback(t *testing.T) {
	client, _ := openPostgres(t)
	ctx := context.Background()
	id := "integration-rollback-" + uuid.NewString()
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	_, err = tx.User.Create().
		SetID(id).
		SetEmail(uuid.NewString() + "@rollback.test").
		SetDisplayName("Rolled Back").
		SetRole(entuser.RoleUser).
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	_, err = client.User.Get(ctx, id)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err))
}

func TestPostgresAggregateSnapshotConcurrentIdempotency(t *testing.T) {
	client, repo := openPostgres(t)
	user := createIntegrationUser(t, client)
	const writers = 16
	key := "integration:" + user.ID + ":aggregate"
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup

	for count := 1; count <= writers; count++ {
		count := count
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.SaveAggregateEventSnapshot(context.Background(), model.AggregateEvent{
				ID:     fmt.Sprintf("integration-aggregate-%d-%s", count, uuid.NewString()),
				UserID: user.ID, DeviceID: "integration-device", IdempotencyKey: key,
				EventType: "block_count_sync", EventDate: time.Now().UTC(), Count: count,
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	stored, err := client.AggregateEvent.Query().Where(aggregateevent.IdempotencyKeyEQ(key)).Only(context.Background())
	require.NoError(t, err)
	assert.Equal(t, writers, stored.Count)
	count, err := client.AggregateEvent.Query().Where(aggregateevent.IdempotencyKeyEQ(key)).Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
