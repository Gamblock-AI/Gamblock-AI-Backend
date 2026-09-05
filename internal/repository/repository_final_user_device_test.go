package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestRepositoryFinalUserDevice_InMemoryUserStatesAndInvalidUpdates(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)

	created, err := repo.CreateUserWithPassword(ctx, "final-user-basic", "final-basic@example.test", "Basic User", "hash-basic", "user")
	require.NoError(t, err)
	assert.Equal(t, "hash-basic", created.PasswordHash)
	assert.False(t, created.MustChangePassword)
	withPhone, err := repo.CreateUserWithPasswordAndPhone(
		ctx, "final-user-phone", "final-phone@example.test", "Phone User", "+628111000000",
		"hash-phone", "partner",
	)
	require.NoError(t, err)
	assert.Equal(t, "+628111000000", withPhone.PhoneE164)
	assert.Equal(t, "partner", withPhone.Role)

	provisioned, err := repo.CreateProvisionedUserWithPhone(
		ctx, "final-user-provisioned", "final-provisioned@example.test", "Provisioned User",
		"+628111000001", "hash-provisioned", "partner", true,
	)
	require.NoError(t, err)
	assert.True(t, provisioned.MustChangePassword)
	assert.Equal(t, "+628111000001", provisioned.PhoneE164)

	_, err = repo.CreateUser(ctx, "final-user-duplicate", "FINAL-PROVISIONED@example.test", "Duplicate")
	assert.EqualError(t, err, "email already exists")

	got, ok := repo.UserByEmail(ctx, "FINAL-PROVISIONED@EXAMPLE.TEST")
	require.True(t, ok)
	assert.Equal(t, provisioned.ID, got.ID)
	assert.Nil(t, got.EmailVerifiedAt)
	assert.Nil(t, got.PhoneVerifiedAt)
	_, ok = repo.UserByEmail(ctx, "")
	assert.False(t, ok)
	_, ok = repo.UserByID(ctx, "")
	assert.False(t, ok)

	updated, err := repo.UpdateUserDisplayName(ctx, provisioned.ID, "Renamed Provisioned")
	require.NoError(t, err)
	assert.Equal(t, "Renamed Provisioned", updated.DisplayName)
	assert.EqualError(t, func() error {
		_, updateErr := repo.UpdateUserDisplayName(ctx, "final-user-missing", "ignored")
		return updateErr
	}(), "user not found")

	avatarKey := "avatar/final-user.webp"
	updated, err = repo.UpdateUserAvatar(ctx, provisioned.ID, &avatarKey)
	require.NoError(t, err)
	assert.Equal(t, "/v1/users/final-user-provisioned/avatar", *updated.AvatarURL)
	storageKey, ok := repo.UserAvatarStorageKey(ctx, provisioned.ID)
	assert.True(t, ok)
	assert.Equal(t, avatarKey, storageKey)

	invalidAvatar := "uploads/final-user.webp"
	_, err = repo.UpdateUserAvatar(ctx, provisioned.ID, &invalidAvatar)
	require.NoError(t, err)
	_, ok = repo.UserAvatarStorageKey(ctx, provisioned.ID)
	assert.False(t, ok)
	_, ok = repo.UserAvatarStorageKey(ctx, "final-user-missing")
	assert.False(t, ok)
	_, err = repo.UpdateUserAvatar(ctx, "final-user-missing", nil)
	assert.EqualError(t, err, "user not found")

	require.NoError(t, repo.UpdateUserPasswordHash(ctx, provisioned.ID, "hash-updated"))
	got, ok = repo.UserByID(ctx, provisioned.ID)
	require.True(t, ok)
	assert.Equal(t, "hash-updated", got.PasswordHash)
	assert.False(t, got.MustChangePassword)
	assert.EqualError(t, repo.UpdateUserPasswordHash(ctx, "final-user-missing", "ignored"), "user not found")

	require.NoError(t, repo.MarkEmailVerified(ctx, provisioned.ID, now))
	require.NoError(t, repo.MarkPhoneVerified(ctx, provisioned.ID, "+628111000009", now))
	got, ok = repo.UserByID(ctx, provisioned.ID)
	require.True(t, ok)
	assert.Equal(t, now, *got.EmailVerifiedAt)
	assert.Equal(t, now, *got.PhoneVerifiedAt)
	assert.Equal(t, "+628111000009", got.PhoneE164)
	require.NoError(t, repo.SetPendingPhone(ctx, provisioned.ID, "+628111000010"))
	got, ok = repo.UserByID(ctx, provisioned.ID)
	require.True(t, ok)
	assert.Equal(t, "+628111000010", got.PhoneE164)
	assert.Nil(t, got.PhoneVerifiedAt)

	require.NoError(t, repo.SetAccountDisabled(ctx, provisioned.ID, true, now))
	got, ok = repo.UserByID(ctx, provisioned.ID)
	require.True(t, ok)
	assert.Equal(t, now, *got.DisabledAt)
	require.NoError(t, repo.SetAccountDisabled(ctx, provisioned.ID, false, now.Add(time.Minute)))
	got, ok = repo.UserByID(ctx, provisioned.ID)
	require.True(t, ok)
	assert.Nil(t, got.DisabledAt)

	assert.EqualError(t, repo.MarkEmailVerified(ctx, "final-user-missing", now), "user not found")
	assert.EqualError(t, repo.MarkPhoneVerified(ctx, "final-user-missing", "+1", now), "user not found")
	assert.EqualError(t, repo.SetPendingPhone(ctx, "final-user-missing", "+1"), "user not found")
	assert.EqualError(t, repo.SetAccountDisabled(ctx, "final-user-missing", true, now), "account not found")
}

func TestRepositoryFinalUserDevice_InMemoryContactAndDeviceStates(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())
	now := time.Date(2026, time.September, 5, 13, 0, 0, 0, time.UTC)

	old := model.ContactVerification{
		ID: "final-contact-old", UserID: "final-user", Kind: "phone", Destination: "+628111000020",
		TokenHash: "final-old-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-2 * time.Minute),
	}
	latest := old
	latest.ID = "final-contact-latest"
	latest.TokenHash = "final-latest-hash"
	latest.CreatedAt = now.Add(-time.Minute)
	require.NoError(t, repo.SaveContactVerification(ctx, old))
	require.NoError(t, repo.SaveContactVerification(ctx, latest))
	require.NoError(t, repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "final-contact-unrelated", UserID: "final-user", Kind: "email", Destination: "final@example.test",
		TokenHash: "final-unrelated-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}))

	_, err := repo.VerifyLatestContactCode(ctx, "phone", "+628111000020", "wrong-hash", now, 2)
	assert.EqualError(t, err, "verification code is invalid or expired")
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628111000020", "wrong-hash", now, 1)
	assert.EqualError(t, err, "verification attempt limit reached")
	verified, err := repo.VerifyLatestContactCode(ctx, "phone", "+628111000020", "final-latest-hash", now, 2)
	require.NoError(t, err)
	assert.Equal(t, latest.ID, verified.ID)
	assert.NotNil(t, verified.ConsumedAt)

	replacement := model.ContactVerification{
		ID: "final-contact-replacement", UserID: "final-user", Kind: "phone", Destination: "+628111000020",
		TokenHash: "final-replacement-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(time.Minute),
	}
	require.NoError(t, repo.SaveContactVerification(ctx, replacement))
	require.NoError(t, repo.InvalidateContactVerifications(ctx, "phone", "+628111000020", now.Add(2*time.Minute)))
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628111000020", "final-replacement-hash", now.Add(2*time.Minute), 3)
	assert.EqualError(t, err, "verification code is invalid or expired")
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "", "missing", now, 3)
	assert.EqualError(t, err, "verification code is invalid or expired")

	consumable := model.ContactVerification{
		ID: "final-contact-consumable", UserID: "final-user", Kind: "email", Destination: "final@example.test",
		TokenHash: "final-consume-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	require.NoError(t, repo.SaveContactVerification(ctx, consumable))
	consumed, err := repo.ConsumeContactVerification(ctx, "final-consume-hash", "email", now)
	require.NoError(t, err)
	assert.Equal(t, consumable.ID, consumed.ID)
	_, err = repo.ConsumeContactVerification(ctx, "final-consume-hash", "email", now)
	assert.EqualError(t, err, "verification token is invalid or expired")
	_, err = repo.ConsumeContactVerification(ctx, "final-unrelated-hash", "phone", now)
	assert.EqualError(t, err, "verification token is invalid or expired")

	expired := model.ContactVerification{
		ID: "final-contact-expired", UserID: "final-user", Kind: "email", Destination: "expired@example.test",
		TokenHash: "final-expired-hash", ExpiresAt: now.Add(-time.Second), CreatedAt: now,
	}
	require.NoError(t, repo.SaveContactVerification(ctx, expired))
	_, err = repo.ConsumeContactVerification(ctx, "final-expired-hash", "email", now)
	assert.EqualError(t, err, "verification token is invalid or expired")

	modelVersion := "final-model"
	rulesetVersion := "final-rules"
	created, err := repo.CreateDevice(ctx, "final-device-owner", "final-owner", "final-instance", "android", "Final phone", "1.0", "Android", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "inactive", created.ProtectionStatus)
	assert.Equal(t, modelVersion, created.ModelVersion)

	updated, err := repo.UpsertDevice(ctx, "ignored-id", "final-owner", "final-instance", "windows", "Updated phone", "2.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Empty(t, updated.ModelVersion)
	assert.Empty(t, updated.RulesetVersion)

	newDevice, err := repo.UpsertDevice(ctx, "final-device-new", "final-owner", "new-instance", "windows", "Desktop", "1.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "final-device-new", newDevice.ID)

	unchanged, err := repo.UpdateDevice(ctx, created.ID, "", "", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, "Updated phone", unchanged.Label)
	fullyUpdated, err := repo.UpdateOwnedDevice(ctx, "final-owner", created.ID, "Owned phone", "3.0", "Android 15", "active", "model-new", "rules-new")
	require.NoError(t, err)
	assert.Equal(t, "active", fullyUpdated.ProtectionStatus)
	assert.Equal(t, "model-new", fullyUpdated.ModelVersion)
	assert.Equal(t, "rules-new", fullyUpdated.RulesetVersion)
	_, err = repo.UpdateOwnedDevice(ctx, "other-owner", created.ID, "blocked", "", "", "", "", "")
	assert.EqualError(t, err, "device not found")
	_, err = repo.UpdateDevice(ctx, "final-device-missing", "", "", "", "", "", "")
	assert.EqualError(t, err, "device not found")

	require.NoError(t, repo.RecordHeartbeat(ctx, created.ID))
	require.NoError(t, repo.RecordOwnedHeartbeat(ctx, "final-owner", created.ID))
	assert.EqualError(t, repo.RecordOwnedHeartbeat(ctx, "other-owner", created.ID), "device not found")
	assert.EqualError(t, repo.RecordHeartbeat(ctx, "final-device-missing"), "device not found")
	assert.True(t, repo.IsDeviceOwnedBy(ctx, created.ID, "final-owner"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, created.ID, "other-owner"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "", "final-owner"))
}

func TestRepositoryFinalUserDevice_SQLitePersistenceConflictsAndEmptyStates(t *testing.T) {
	ctx := context.Background()
	client := repositoryFinalUserDeviceOpenSQLite(t)
	repo := New(client, store.New())
	now := time.Date(2026, time.September, 5, 14, 0, 0, 0, time.UTC)

	created, err := repo.CreateProvisionedUserWithPhone(ctx, "final-sql-user", "final-sql@example.test", "SQL User", "+628111000030", "sql-hash", "user", true)
	require.NoError(t, err)
	assert.True(t, created.MustChangePassword)
	assert.Equal(t, "+628111000030", created.PhoneE164)
	_, ok := repo.UserByEmail(ctx, "FINAL-SQL@EXAMPLE.TEST")
	assert.True(t, ok)
	_, ok = repo.UserByEmail(ctx, "missing-sql@example.test")
	assert.False(t, ok)
	_, ok = repo.UserByID(ctx, "missing-sql-user")
	assert.False(t, ok)
	_, err = repo.CreateUser(ctx, "final-sql-duplicate", "final-sql@example.test", "Duplicate")
	assert.Error(t, err)

	second, err := repo.CreateUserWithPassword(ctx, "final-sql-second", "second-sql@example.test", "Second SQL", "", "partner")
	require.NoError(t, err)
	assert.Empty(t, second.PasswordHash)

	require.NoError(t, repo.MarkEmailVerified(ctx, created.ID, now))
	require.NoError(t, repo.MarkPhoneVerified(ctx, created.ID, "+628111000031", now))
	require.NoError(t, repo.SetPendingPhone(ctx, created.ID, "+628111000032"))
	require.NoError(t, repo.SetAccountDisabled(ctx, created.ID, true, now))
	verified, ok := repo.UserByID(ctx, created.ID)
	require.True(t, ok)
	assert.Equal(t, now, *verified.EmailVerifiedAt)
	assert.Nil(t, verified.PhoneVerifiedAt)
	assert.Equal(t, now, *verified.DisabledAt)
	require.NoError(t, repo.SetAccountDisabled(ctx, created.ID, false, now.Add(time.Minute)))
	verified, ok = repo.UserByID(ctx, created.ID)
	require.True(t, ok)
	assert.Nil(t, verified.DisabledAt)
	assert.EqualError(t, repo.MarkEmailVerified(ctx, "missing-sql-user", now), "ent: user not found")
	assert.EqualError(t, repo.MarkPhoneVerified(ctx, "missing-sql-user", "+1", now), "ent: user not found")
	assert.EqualError(t, repo.SetPendingPhone(ctx, "missing-sql-user", "+1"), "ent: user not found")
	assert.EqualError(t, repo.SetAccountDisabled(ctx, "missing-sql-user", true, now), "account not found")

	avatarKey := "avatar/final-sql.webp"
	updated, err := repo.UpdateUserDisplayName(ctx, created.ID, "SQL Renamed")
	require.NoError(t, err)
	assert.Equal(t, "SQL Renamed", updated.DisplayName)
	updated, err = repo.UpdateUserAvatar(ctx, created.ID, &avatarKey)
	require.NoError(t, err)
	assert.Equal(t, "/v1/users/final-sql-user/avatar", *updated.AvatarURL)
	storageKey, ok := repo.UserAvatarStorageKey(ctx, created.ID)
	assert.True(t, ok)
	assert.Equal(t, avatarKey, storageKey)
	invalidAvatar := "uploads/final-sql.webp"
	_, err = repo.UpdateUserAvatar(ctx, created.ID, &invalidAvatar)
	require.NoError(t, err)
	_, ok = repo.UserAvatarStorageKey(ctx, created.ID)
	assert.False(t, ok)
	_, err = repo.UpdateUserAvatar(ctx, "missing-sql-user", nil)
	assert.Error(t, err)
	require.NoError(t, repo.UpdateUserPasswordHash(ctx, created.ID, "sql-updated-hash"))
	assert.Error(t, repo.UpdateUserPasswordHash(ctx, "missing-sql-user", "ignored"))

	contact := model.ContactVerification{
		ID: "final-sql-contact", UserID: created.ID, Kind: "phone", Destination: "+628111000032",
		TokenHash: "final-sql-contact-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	require.NoError(t, repo.SaveContactVerification(ctx, contact))
	assert.Error(t, repo.SaveContactVerification(ctx, contact), "duplicate verification IDs must be rejected")
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628111000032", "wrong", now, 2)
	assert.EqualError(t, err, "verification code is invalid or expired")
	verifiedContact, err := repo.VerifyLatestContactCode(ctx, "phone", "+628111000032", "final-sql-contact-hash", now, 2)
	require.NoError(t, err)
	assert.Equal(t, contact.ID, verifiedContact.ID)
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628111000032", "final-sql-contact-hash", now, 2)
	assert.EqualError(t, err, "verification code is invalid or expired")
	require.NoError(t, repo.InvalidateContactVerifications(ctx, "phone", "+628111000032", now))
	_, err = repo.ConsumeContactVerification(ctx, "missing-contact", "phone", now)
	assert.EqualError(t, err, "verification token is invalid or expired")
	consumeContact := model.ContactVerification{
		ID: "final-sql-consume-contact", UserID: created.ID, Kind: "email", Destination: "final-sql@example.test",
		TokenHash: "final-sql-consume-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	require.NoError(t, repo.SaveContactVerification(ctx, consumeContact))
	consumed, err := repo.ConsumeContactVerification(ctx, consumeContact.TokenHash, consumeContact.Kind, now)
	require.NoError(t, err)
	assert.Equal(t, consumeContact.ID, consumed.ID)

	modelVersion := "final-sql-model"
	rulesetVersion := "final-sql-rules"
	deviceItem, err := repo.CreateDevice(ctx, "final-sql-device", created.ID, "final-sql-instance", "android", "SQL phone", "1.0", "Android", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "inactive", deviceItem.ProtectionStatus)
	_, err = repo.CreateDevice(ctx, "final-sql-device", created.ID, "final-sql-instance-duplicate", "android", "Duplicate", "1.0", "Android", nil, nil)
	assert.Error(t, err, "duplicate device IDs must be rejected")
	upserted, err := repo.UpsertDevice(ctx, "ignored-device-id", created.ID, "final-sql-instance", "windows", "SQL desktop", "2.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, deviceItem.ID, upserted.ID)
	assert.Equal(t, modelVersion, upserted.ModelVersion, "nil optional updates preserve the SQL value")
	newDevice, err := repo.UpsertDevice(ctx, "final-sql-device-new", created.ID, "final-sql-instance-new", "windows", "New desktop", "1.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "final-sql-device-new", newDevice.ID)

	partial, err := repo.UpdateOwnedDevice(ctx, created.ID, deviceItem.ID, "", "2.1", "Android 15", "active", "", "")
	require.NoError(t, err)
	assert.Equal(t, "SQL desktop", partial.Label)
	assert.Equal(t, modelVersion, partial.ModelVersion)
	assert.EqualError(t, func() error {
		_, updateErr := repo.UpdateOwnedDevice(ctx, "other-sql-user", deviceItem.ID, "blocked", "", "", "", "", "")
		return updateErr
	}(), "device not found")
	_, err = repo.UpdateDevice(ctx, "missing-sql-device", "", "", "", "", "", "")
	assert.Error(t, err)
	require.NoError(t, repo.RecordHeartbeat(ctx, deviceItem.ID))
	require.NoError(t, repo.RecordOwnedHeartbeat(ctx, created.ID, deviceItem.ID))
	assert.Error(t, repo.RecordOwnedHeartbeat(ctx, "other-sql-user", deviceItem.ID))
	assert.Error(t, repo.RecordHeartbeat(ctx, "missing-sql-device"))
	assert.True(t, repo.IsDeviceOwnedBy(ctx, deviceItem.ID, created.ID))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, deviceItem.ID, "other-sql-user"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "", created.ID))
}

func repositoryFinalUserDeviceOpenSQLite(t *testing.T) *ent.Client {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_", ":", "_").Replace(t.Name())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", databaseName))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	client := ent.NewClient(ent.Driver(entsql.OpenDB("sqlite3", db)))
	require.NoError(t, client.Schema.Create(context.Background()))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
