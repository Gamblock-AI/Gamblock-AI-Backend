package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/operatorinvitation"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoverypracticesession"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	_ "modernc.org/sqlite" // CGO-free sqlite driver
)

func openSQLite(t *testing.T) *ent.Client {
	t.Helper()
	client, db := openSQLiteWithDB(t)
	t.Cleanup(func() { _ = db.Close() })
	return client
}

func openSQLiteWithDB(t *testing.T) (*ent.Client, *sql.DB) {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", databaseName))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}
	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client, db
}

func TestOpen_RejectsEmptyURLAndClosesValidClient(t *testing.T) {
	client, closeClient, err := Open("")
	assert.Nil(t, client)
	assert.Nil(t, closeClient)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is empty")

	client, closeClient, err = Open("postgres://user:password@127.0.0.1:1/test")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, closeClient)
	assert.NoError(t, closeClient())
}

func TestDropPublicSchema_RejectsEmptyURL(t *testing.T) {
	err := DropPublicSchema(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is empty")
}

func TestDropPublicSchema_ReturnsContextErrorBeforeOpeningTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := DropPublicSchema(ctx, "postgres://user:password@127.0.0.1:1/test")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestMigrate_Idempotent(t *testing.T) {
	client := openSQLite(t)
	defer client.Close()
	ctx := context.Background()
	// Migrate runs schema create (already done in openSQLite); re-running must not error.
	require.NoError(t, Migrate(ctx, client))
}

func TestMigrate_NormalizesLegacyRolesAndPendingInvitations(t *testing.T) {
	client, sqlDB := openSQLiteWithDB(t)
	defer client.Close()
	defer sqlDB.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	for _, id := range []string{"legacy-content", "legacy-owner"} {
		_, err := client.User.Create().SetID(id).SetEmail(id + "@example.com").SetDisplayName(id).SetPasswordHash("hash").Save(ctx)
		require.NoError(t, err)
	}
	_, err := sqlDB.ExecContext(ctx, "UPDATE users SET role = ? WHERE id = ?", "content_admin", "legacy-content")
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, "UPDATE users SET role = ? WHERE id = ?", "organization_owner", "legacy-owner")
	require.NoError(t, err)

	_, err = client.SupportCase.Create().SetID("legacy-case").SetUserID("legacy-content").SetType(supportcase.TypeTechnicalSupport).SetSummary("legacy support").Save(ctx)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO support_messages
		(id, support_case_id, author_id, author_role, content_encrypted, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, "legacy-message", "legacy-case", "legacy-content", "support_operator", "ciphertext", now)
	require.NoError(t, err)

	_, err = client.OperatorInvitation.Create().SetID("legacy-invitation").SetEmail("operator@example.com").SetRole(operatorinvitation.RoleAdmin).SetTokenHash("legacy-token").SetStatus(operatorinvitation.StatusPending).SetInvitedBy("legacy-owner").SetExpiresAt(now.Add(time.Hour)).Save(ctx)
	require.NoError(t, err)

	require.NoError(t, Migrate(ctx, client))

	contentUser, err := client.User.Get(ctx, "legacy-content")
	require.NoError(t, err)
	assert.Equal(t, "admin", contentUser.Role.String())
	ownerUser, err := client.User.Get(ctx, "legacy-owner")
	require.NoError(t, err)
	assert.Equal(t, "partner", ownerUser.Role.String())

	message, err := client.SupportMessage.Get(ctx, "legacy-message")
	require.NoError(t, err)
	assert.Equal(t, "admin", message.AuthorRole.String())
	invitation, err := client.OperatorInvitation.Get(ctx, "legacy-invitation")
	require.NoError(t, err)
	assert.Equal(t, operatorinvitation.RoleAdmin.String(), invitation.Role.String())
	assert.Equal(t, operatorinvitation.StatusRevoked.String(), invitation.Status.String())
}

func TestMigrate_RollsBackEarlierUpdatesWhenLaterMigrationFails(t *testing.T) {
	client, sqlDB := openSQLiteWithDB(t)
	defer client.Close()
	defer sqlDB.Close()
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := client.User.Create().SetID("rollback-user").SetEmail("rollback@example.com").SetDisplayName("Rollback").SetPasswordHash("hash").Save(ctx)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, "UPDATE users SET role = ? WHERE id = ?", "content_admin", "rollback-user")
	require.NoError(t, err)
	_, err = client.SupportCase.Create().SetID("rollback-case").SetUserID("rollback-user").SetType(supportcase.TypeTechnicalSupport).SetSummary("rollback support").Save(ctx)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO support_messages
		(id, support_case_id, author_id, author_role, content_encrypted, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`, "rollback-message", "rollback-case", "rollback-user", "support_operator", "ciphertext", now)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(ctx, `CREATE TRIGGER fail_legacy_support_migration
		BEFORE UPDATE OF author_role ON support_messages
		WHEN NEW.author_role = 'admin'
		BEGIN SELECT RAISE(ABORT, 'forced migration failure'); END`)
	require.NoError(t, err)

	err = Migrate(ctx, client)
	require.Error(t, err)

	userRow, err := client.User.Get(ctx, "rollback-user")
	require.NoError(t, err)
	assert.Equal(t, "content_admin", userRow.Role.String(), "the user role update must be rolled back")
	message, err := client.SupportMessage.Get(ctx, "rollback-message")
	require.NoError(t, err)
	assert.Equal(t, "support_operator", message.AuthorRole.String(), "the failing update must not be committed")
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

func TestLoadStore_EmptyDatabaseReturnsEmptyStore(t *testing.T) {
	client := openSQLite(t)
	defer client.Close()

	loaded, err := LoadStore(context.Background(), client)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Empty(t, loaded.Users)
	assert.Empty(t, loaded.Devices)
}

func TestDatabaseOperations_ReturnErrorsAfterClientClose(t *testing.T) {
	client := openSQLite(t)
	require.NoError(t, client.Close())
	ctx := context.Background()

	_, err := LoadStore(ctx, client)
	assert.Error(t, err)
	assert.Error(t, Migrate(ctx, client))
	assert.Error(t, Seed(ctx, client))
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
	for input, expected := range map[string]string{
		"pending": "Pending partner approval", "approved": "Approved", "denied": "Denied",
		"expired": "Expired", "cancelled": "Cancelled", "weird": "weird",
	} {
		assert.Equal(t, expected, humanApprovalStatus(input))
	}
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

func TestLoaderHelpers_HandleMissionKeysAndOptionalValues(t *testing.T) {
	assert.Equal(t, 12, missionKeyNumber("mission_12"))
	assert.Equal(t, 0, missionKeyNumber("not-a-mission"))

	day := &store.DailyMission{}
	for number := 1; number <= 6; number++ {
		setMissionCompleted(day, number, true)
	}
	assert.True(t, day.Mission1)
	assert.True(t, day.Mission2)
	assert.True(t, day.Mission3)
	assert.True(t, day.Mission4)
	assert.True(t, day.Mission5)
	assert.True(t, day.Mission6)
	setMissionCompleted(day, 99, false)
	assert.True(t, day.Mission6, "unknown mission numbers must be ignored")

	var nilAdjustment *string
	assert.Equal(t, "", valueEnum(nilAdjustment))
	adjustment := "not_enough_time"
	assert.Equal(t, adjustment, valueEnum(&adjustment))
	assert.Equal(t, 0, valueInt(nil))
	number := 7
	assert.Equal(t, number, valueInt(&number))

	assert.Equal(t, "", recoveryFeedback(nil))
	feedback := recoverypracticesession.FeedbackLighter
	assert.Equal(t, feedback.String(), recoveryFeedback(&feedback))
}
