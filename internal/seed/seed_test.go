package seed

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningitem"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningrevision"
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
	firstAggregates, err := client.AggregateEvent.Query().Count(ctx)
	require.NoError(t, err)
	require.Greater(t, firstAggregates, 0)
	firstGroups, err := client.AccountabilityGroup.Query().Count(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, firstGroups, 1)

	require.NoError(t, Seed(ctx, client))
	second, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, first, second, "second seed must not duplicate users")
	secondAggregates, err := client.AggregateEvent.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, firstAggregates, secondAggregates, "second seed must not duplicate aggregate events")
}

func TestSeedProductionDefaults_DoesNotCreateDemoActivity(t *testing.T) {
	client := openSQLiteEnt(t)
	defer client.Close()
	ctx := context.Background()

	require.NoError(t, SeedProductionDefaults(ctx, client, t.TempDir()))

	users, err := client.User.Query().Count(ctx)
	require.NoError(t, err)
	aggregates, err := client.AggregateEvent.Query().Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, users)
	assert.Zero(t, aggregates)
}

func TestSeedLearningHubDefaults_RetrofitsPublishedMedia(t *testing.T) {
	client := openSQLiteEnt(t)
	defer client.Close()
	ctx := context.Background()
	mediaRoot := t.TempDir()

	_, err := SeedLearningHubDefaultsWithReport(ctx, client, mediaRoot)
	require.NoError(t, err)

	item, err := client.LearningItem.Query().Where(learningitem.IDEQ("item_autodesk-autocad-design")).Only(ctx)
	require.NoError(t, err)
	draft := cloneSeedDocument(item.DocumentJSON)
	delete(draft, "provider_logo_media_id")
	delete(draft, "thumbnail_media_id")
	_, err = client.LearningItem.UpdateOneID(item.ID).SetDocumentJSON(draft).Save(ctx)
	require.NoError(t, err)

	revision, err := client.LearningRevision.Query().Where(
		learningrevision.ItemIDEQ(item.ID),
		learningrevision.KindEQ(learningrevision.KindPublished),
	).Only(ctx)
	require.NoError(t, err)
	revisionDocument := cloneSeedDocument(revision.DocumentJSON)
	snapshot, ok := revisionDocument["_learning_item_snapshot"].(map[string]any)
	require.True(t, ok)
	publicDocument, ok := snapshot["document"].(map[string]any)
	require.True(t, ok)
	delete(publicDocument, "provider_logo_media_id")
	delete(publicDocument, "thumbnail_media_id")
	_, err = client.LearningRevision.UpdateOneID(revision.ID).SetDocumentJSON(revisionDocument).Save(ctx)
	require.NoError(t, err)

	_, err = SeedLearningHubDefaultsWithReport(ctx, client, mediaRoot)
	require.NoError(t, err)

	revision, err = client.LearningRevision.Get(ctx, revision.ID)
	require.NoError(t, err)
	snapshot, ok = revision.DocumentJSON["_learning_item_snapshot"].(map[string]any)
	require.True(t, ok)
	publicDocument, ok = snapshot["document"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "med_seed_lh_logo_autodesk", publicDocument["provider_logo_media_id"])
	assert.NotEmpty(t, publicDocument["thumbnail_media_id"])
}
