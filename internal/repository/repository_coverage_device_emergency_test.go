package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/emergencykeyrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	_ "modernc.org/sqlite"
)

func TestRepositoryCoverageDeviceEmergency_DeviceRegistrationOwnershipAndKeys(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())

	modelVersion := "model-cov-1"
	rulesetVersion := "ruleset-cov-1"
	created, err := repo.CreateDevice(ctx, "device-cov-owner", "user-cov-owner", "instance-cov-owner", "android", "Phone", "1.0", "Android", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "inactive", created.ProtectionStatus)
	assert.Equal(t, modelVersion, created.ModelVersion)
	assert.Equal(t, rulesetVersion, created.RulesetVersion)

	createdWithoutOptionalVersions, err := repo.CreateDevice(ctx, "device-cov-empty", "user-cov-owner", "instance-cov-empty", "windows", "Desktop", "2.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Empty(t, createdWithoutOptionalVersions.ModelVersion)
	assert.Empty(t, createdWithoutOptionalVersions.RulesetVersion)

	updated, err := repo.UpsertDevice(ctx, "ignored-id", "user-cov-owner", "instance-cov-owner", "windows", "Updated phone", "1.1", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "windows", updated.Platform)
	assert.Equal(t, "Updated phone", updated.Label)
	assert.Empty(t, updated.ModelVersion)
	assert.Empty(t, updated.RulesetVersion)

	newDevice, err := repo.UpsertDevice(ctx, "device-cov-new", "user-cov-owner", "instance-cov-new", "linux", "Linux", "3.0", "Linux", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "device-cov-new", newDevice.ID)

	fullyUpdated, err := repo.UpdateOwnedDevice(ctx, "user-cov-owner", created.ID, "Final", "1.2", "Windows 11", "active", "model-cov-2", "ruleset-cov-2")
	require.NoError(t, err)
	assert.Equal(t, "Final", fullyUpdated.Label)
	assert.Equal(t, "1.2", fullyUpdated.AppVersion)
	assert.Equal(t, "Windows 11", fullyUpdated.OSVersion)
	assert.Equal(t, "active", fullyUpdated.ProtectionStatus)
	assert.Equal(t, "model-cov-2", fullyUpdated.ModelVersion)
	assert.Equal(t, "ruleset-cov-2", fullyUpdated.RulesetVersion)

	_, err = repo.UpdateOwnedDevice(ctx, "other-user", created.ID, "no access", "", "", "", "", "")
	require.Error(t, err)
	_, err = repo.UpdateDevice(ctx, "missing-device", "", "", "", "", "", "")
	require.Error(t, err)

	require.NoError(t, repo.RecordOwnedHeartbeat(ctx, "user-cov-owner", created.ID))
	require.NoError(t, repo.RecordHeartbeat(ctx, created.ID))
	require.Error(t, repo.RecordOwnedHeartbeat(ctx, "other-user", created.ID))
	require.Error(t, repo.RecordHeartbeat(ctx, "missing-device"))
	assert.True(t, repo.IsDeviceOwnedBy(ctx, created.ID, "user-cov-owner"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, created.ID, "other-user"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "", "user-cov-owner"))

	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "user-cov-owner", created.ID, "{\"kty\":\"EC\"}", "thumb-cov-1"))
	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "user-cov-owner", created.ID, "{\"kty\":\"EC\"}", "thumb-cov-1"), "the same proof is idempotent")
	err = repo.BindOwnedDeviceGrantKey(ctx, "user-cov-owner", created.ID, "{\"kty\":\"EC\"}", "thumb-cov-2")
	require.ErrorIs(t, err, ErrDeviceGrantKeyConflict)
	err = repo.BindOwnedDeviceGrantKey(ctx, "user-cov-owner", created.ID, "{\"kty\":\"RSA\"}", "thumb-cov-1")
	require.ErrorIs(t, err, ErrDeviceGrantKeyConflict)
	err = repo.BindOwnedDeviceGrantKey(ctx, "other-user", created.ID, "{}", "thumb-other")
	require.Error(t, err)

	thumbprint, err := repo.OwnedDeviceGrantKeyThumbprint(ctx, "user-cov-owner", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "thumb-cov-1", thumbprint)
	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "other-user", created.ID)
	require.Error(t, err)
	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "user-cov-owner", createdWithoutOptionalVersions.ID)
	require.Error(t, err)
}

func TestRepositoryCoverageDeviceEmergency_EmergencyFilteringAndExpiry(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())
	now := time.Date(2026, time.June, 10, 12, 0, 0, 0, time.UTC)

	requests := []model.EmergencyKeyRequest{
		deviceEmergencyCoverageRequest("emergency-cov-pending", "user-cov-a", "device-cov-a", "pending", now.Add(-2*time.Hour), now.Add(time.Hour)),
		deviceEmergencyCoverageRequest("emergency-cov-reviewed", "user-cov-b", "device-cov-b", "reviewed", now.Add(-time.Hour), now.Add(2*time.Hour)),
		deviceEmergencyCoverageRequest("emergency-cov-expired", "user-cov-c", "device-cov-c", "pending", now.Add(-3*time.Hour), now.Add(-time.Minute)),
		deviceEmergencyCoverageRequest("emergency-cov-history", "user-cov-a", "device-cov-a", "approved", now.Add(-30*time.Minute), now.Add(-3*time.Hour)),
	}
	keyExpires := now.Add(-time.Minute)
	requests[3].KeyHash = "hash-cov-history"
	requests[3].KeyExpiresAt = &keyExpires
	for _, request := range requests {
		_, err := repo.CreateEmergencyKeyRequest(ctx, request)
		require.NoError(t, err)
	}

	current, err := repo.GetCurrentEmergencyKeyRequest(ctx, "user-cov-a", "device-cov-a", now)
	require.NoError(t, err)
	assert.Equal(t, "emergency-cov-history", current.ID, "the newest matching request is selected")
	assert.Equal(t, "expired", current.Status, "approved requests expire when their key window closes")

	pending, err := repo.GetPendingEmergencyKeyRequests(ctx, now)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.Equal(t, "emergency-cov-pending", pending[0].ID)
	assert.Equal(t, "emergency-cov-reviewed", pending[1].ID)

	page, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Limit: 1, Query: "USER-COV-B"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "emergency-cov-reviewed", page.Items[0].ID)
	assert.Equal(t, 1, page.TotalCount)

	history, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Bucket: "history", Limit: 10})
	require.NoError(t, err)
	require.Len(t, history.Items, 2)
	assert.Equal(t, "emergency-cov-expired", history.Items[0].ID)
	assert.Equal(t, "emergency-cov-history", history.Items[1].ID)

	statusPage, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Status: "reviewed", Limit: 10})
	require.NoError(t, err)
	require.Len(t, statusPage.Items, 1)
	assert.Equal(t, "emergency-cov-reviewed", statusPage.Items[0].ID)

	_, err = repo.GetCurrentEmergencyKeyRequest(ctx, "unknown-user", "unknown-device", now)
	require.Error(t, err)
}

func TestRepositoryCoverageDeviceEmergency_EmergencyTransitionsAndGrantLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := New(nil, store.New())
	now := time.Date(2026, time.June, 11, 8, 0, 0, 0, time.UTC)

	reviewID := "emergency-cov-review-transition"
	_, err := repo.CreateEmergencyKeyRequest(ctx, deviceEmergencyCoverageRequest(reviewID, "user-cov", "device-cov", "pending", now, now.Add(time.Hour)))
	require.NoError(t, err)
	reviewed, err := repo.ReviewEmergencyKeyRequest(ctx, reviewID, "admin-cov", now.Add(10*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "reviewed", reviewed.Status)
	assert.Equal(t, "admin-cov", reviewed.ReviewedBy)

	keyExpires := now.Add(2 * time.Hour)
	approved, err := repo.ApproveEmergencyKeyRequest(ctx, reviewID, "admin-cov", "hash-cov-transition", now.Add(20*time.Minute), keyExpires)
	require.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)
	assert.Equal(t, "hash-cov-transition", approved.KeyHash)

	usable, err := repo.GetUsableEmergencyKeyRequest(ctx, "hash-cov-transition", "device-cov", now.Add(30*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, reviewID, usable.ID)
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "hash-cov-transition", "wrong-device", now.Add(30*time.Minute))
	require.Error(t, err)

	grantExpires := now.Add(40 * time.Minute)
	used, err := repo.UseEmergencyKey(ctx, "hash-cov-transition", "device-cov", now.Add(30*time.Minute), grantExpires, "grant-cov-1")
	require.NoError(t, err)
	assert.Equal(t, "used", used.Status)
	assert.Equal(t, "grant-cov-1", used.GrantJTI)

	retry, err := repo.GetUsableEmergencyKeyRequest(ctx, "hash-cov-transition", "device-cov", now.Add(35*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "grant-cov-1", retry.GrantJTI)
	retried, err := repo.UseEmergencyKey(ctx, "hash-cov-transition", "device-cov", now.Add(35*time.Minute), now.Add(45*time.Minute), "grant-cov-retry")
	require.NoError(t, err)
	assert.Equal(t, "grant-cov-1", retried.GrantJTI, "a retry preserves the original grant identity")

	_, err = repo.UseEmergencyKey(ctx, "hash-cov-transition", "device-cov", now.Add(30*time.Minute), now.Add(50*time.Minute), "bad-window")
	require.Error(t, err)
	_, err = repo.UseEmergencyKey(ctx, "hash-cov-transition", "wrong-device", now.Add(35*time.Minute), now.Add(45*time.Minute), "grant-cov-2")
	require.Error(t, err)
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "hash-cov-transition", "device-cov", now.Add(2*time.Hour))
	require.Error(t, err)

	expiredID := "emergency-cov-expired-transition"
	_, err = repo.CreateEmergencyKeyRequest(ctx, deviceEmergencyCoverageRequest(expiredID, "user-cov", "device-cov", "pending", now, now.Add(time.Hour)))
	require.NoError(t, err)
	_, err = repo.ReviewEmergencyKeyRequest(ctx, expiredID, "admin-cov", now.Add(2*time.Hour))
	require.Error(t, err)
	_, err = repo.ApproveEmergencyKeyRequest(ctx, "not-found", "admin-cov", "hash-missing", now, keyExpires)
	require.Error(t, err)

	reviewedID := "emergency-cov-reviewed-approve"
	existingReview := deviceEmergencyCoverageRequest(reviewedID, "user-cov", "device-cov-2", "reviewed", now, now.Add(time.Hour))
	existingReview.ReviewedBy = "existing-reviewer"
	_, err = repo.CreateEmergencyKeyRequest(ctx, existingReview)
	require.NoError(t, err)
	approvedReviewed, err := repo.ApproveEmergencyKeyRequest(ctx, reviewedID, "admin-cov", "hash-cov-reviewed", now, keyExpires)
	require.NoError(t, err)
	assert.Equal(t, "existing-reviewer", approvedReviewed.ReviewedBy, "an existing reviewer is preserved")

	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "missing-hash", "device-cov", now)
	require.Error(t, err)
}

func TestRepositoryCoverageDeviceEmergency_Mappers(t *testing.T) {
	now := time.Date(2026, time.June, 12, 9, 0, 0, 0, time.UTC)

	withoutOptionals := deviceFromEnt(&ent.Device{
		ID: "mapped-device-empty", UserID: "mapped-user", Platform: device.PlatformAndroid,
		ProtectionStatus: device.ProtectionStatusInactive, CreatedAt: now, UpdatedAt: now,
	})
	assert.Equal(t, "mapped-device-empty", withoutOptionals.ID)
	assert.Empty(t, withoutOptionals.ClientInstanceID)
	assert.Empty(t, withoutOptionals.ModelVersion)
	assert.Empty(t, withoutOptionals.RulesetVersion)
	assert.Empty(t, withoutOptionals.GrantPublicJWK)
	assert.True(t, withoutOptionals.LastSeenAt.IsZero())

	lastSeen := now.Add(time.Minute)
	deviceClient := "client-mapped"
	deviceModel := "model-mapped"
	deviceRules := "rules-mapped"
	deviceJWK := "jwk-mapped"
	deviceThumb := "thumb-mapped"
	withOptionals := deviceFromEnt(&ent.Device{
		ID: "mapped-device-full", UserID: "mapped-user", ClientInstanceID: &deviceClient,
		Platform: device.PlatformWindows, Label: "Mapped desktop", AppVersion: "1.2",
		OsVersion: "Windows", ModelVersion: &deviceModel, RulesetVersion: &deviceRules,
		GrantPublicJwk: &deviceJWK, GrantKeyThumbprint: &deviceThumb,
		ProtectionStatus: device.ProtectionStatusActive, LastSeenAt: &lastSeen,
		CreatedAt: now, UpdatedAt: lastSeen,
	})
	assert.Equal(t, deviceClient, withOptionals.ClientInstanceID)
	assert.Equal(t, deviceJWK, withOptionals.GrantPublicJWK)
	assert.Equal(t, deviceThumb, withOptionals.GrantKeyThumbprint)
	assert.Equal(t, lastSeen, withOptionals.LastSeenAt)

	emergencyEmpty := emergencyRequestFromEnt(&ent.EmergencyKeyRequest{
		ID: "mapped-emergency-empty", RequestedBy: "mapped-user",
		Status: emergencykeyrequest.StatusPending, RequestExpiresAt: now,
		CreatedAt: now, UpdatedAt: now,
	})
	assert.Equal(t, "mapped-emergency-empty", emergencyEmpty.ID)
	assert.Empty(t, emergencyEmpty.DeviceID)
	assert.Empty(t, emergencyEmpty.KeyHash)
	assert.Nil(t, emergencyEmpty.KeyExpiresAt)

	emergencyDevice := "mapped-device-full"
	reviewer := "mapped-reviewer"
	approver := "mapped-approver"
	keyHash := "mapped-key-hash"
	grantJTI := "mapped-grant-jti"
	keyExpiry := now.Add(time.Hour)
	reviewedAt := now.Add(2 * time.Minute)
	approvedAt := now.Add(3 * time.Minute)
	usedAt := now.Add(4 * time.Minute)
	grantStart := now.Add(5 * time.Minute)
	grantExpiry := now.Add(15 * time.Minute)
	emergencyFull := emergencyRequestFromEnt(&ent.EmergencyKeyRequest{
		ID: "mapped-emergency-full", RequestedBy: "mapped-user", DeviceID: &emergencyDevice,
		ReviewedBy: &reviewer, ApprovedBy: &approver, Status: emergencykeyrequest.StatusUsed,
		KeyHash: &keyHash, RequestExpiresAt: now, KeyExpiresAt: &keyExpiry,
		ReviewedAt: &reviewedAt, ApprovedAt: &approvedAt, UsedAt: &usedAt,
		GrantStartsAt: &grantStart, GrantExpiresAt: &grantExpiry, GrantJti: &grantJTI,
		CreatedAt: now, UpdatedAt: grantStart,
	})
	assert.Equal(t, emergencyDevice, emergencyFull.DeviceID)
	assert.Equal(t, keyHash, emergencyFull.KeyHash)
	assert.Equal(t, grantJTI, emergencyFull.GrantJTI)
	assert.Equal(t, grantExpiry, *emergencyFull.GrantExpiresAt)
	assert.Equal(t, approvedAt, *emergencyFull.ApprovedAt)
}

func TestRepositoryCoverageDeviceEmergency_SQLitePersistenceAndMappers(t *testing.T) {
	client := deviceEmergencyCoverageOpenSQLite(t)
	ctx := context.Background()
	repo := New(client, store.New())

	modelVersion := "sqlite-model"
	rulesetVersion := "sqlite-rules"
	created, err := repo.CreateDevice(ctx, "sqlite-device", "sqlite-user", "sqlite-instance", "android", "SQLite phone", "1.0", "Android", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, modelVersion, created.ModelVersion)
	assert.Equal(t, rulesetVersion, created.RulesetVersion)

	updated, err := repo.UpsertDevice(ctx, "ignored", "sqlite-user", "sqlite-instance", "android", "Updated SQLite phone", "1.1", "Android 15", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "Updated SQLite phone", updated.Label)
	assert.Equal(t, "Android 15", updated.OSVersion)

	newDevice, err := repo.UpsertDevice(ctx, "sqlite-device-new", "sqlite-user", "sqlite-instance-new", "windows", "SQLite desktop", "2.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "sqlite-device-new", newDevice.ID)

	_, err = repo.UpdateOwnedDevice(ctx, "wrong-user", created.ID, "blocked", "", "", "", "", "")
	require.Error(t, err)
	updated, err = repo.UpdateOwnedDevice(ctx, "sqlite-user", created.ID, "", "1.2", "Android 16", "active", "", "")
	require.NoError(t, err)
	assert.Equal(t, "active", updated.ProtectionStatus)
	assert.Equal(t, "sqlite-model", updated.ModelVersion, "empty optional update does not erase the stored model")
	require.NoError(t, repo.RecordOwnedHeartbeat(ctx, "sqlite-user", created.ID))
	require.Error(t, repo.RecordOwnedHeartbeat(ctx, "wrong-user", created.ID))
	assert.True(t, repo.IsDeviceOwnedBy(ctx, created.ID, "sqlite-user"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, created.ID, "wrong-user"))

	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "sqlite-user", created.ID, "sqlite-jwk", "sqlite-thumb"))
	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "sqlite-user", created.ID, "sqlite-jwk", "sqlite-thumb"))
	require.ErrorIs(t, repo.BindOwnedDeviceGrantKey(ctx, "sqlite-user", created.ID, "other-jwk", "other-thumb"), ErrDeviceGrantKeyConflict)
	thumbprint, err := repo.OwnedDeviceGrantKeyThumbprint(ctx, "sqlite-user", created.ID)
	require.NoError(t, err)
	assert.Equal(t, "sqlite-thumb", thumbprint)
	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "sqlite-user", newDevice.ID)
	require.Error(t, err)

	now := time.Date(2026, time.June, 13, 10, 0, 0, 0, time.UTC)
	request, err := repo.CreateEmergencyKeyRequest(ctx, model.EmergencyKeyRequest{
		ID: "sqlite-emergency", RequestedBy: "sqlite-user", DeviceID: created.ID,
		RequestExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", request.Status)

	reviewed, err := repo.ReviewEmergencyKeyRequest(ctx, request.ID, "sqlite-admin", now.Add(5*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "reviewed", reviewed.Status)
	pending, err := repo.GetPendingEmergencyKeyRequests(ctx, now.Add(6*time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, request.ID, pending[0].ID)
	pendingPage, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now.Add(6*time.Minute), model.PaginationQuery{Status: "reviewed", Limit: 1})
	require.NoError(t, err)
	require.Len(t, pendingPage.Items, 1)
	assert.Equal(t, request.ID, pendingPage.Items[0].ID)
	keyExpiry := now.Add(2 * time.Hour)
	approved, err := repo.ApproveEmergencyKeyRequest(ctx, request.ID, "sqlite-admin", "sqlite-key-hash", now.Add(10*time.Minute), keyExpiry)
	require.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)
	assert.Equal(t, created.ID, approved.DeviceID)
	current, err := repo.GetCurrentEmergencyKeyRequest(ctx, "sqlite-user", created.ID, now.Add(20*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "approved", current.Status)

	usable, err := repo.GetUsableEmergencyKeyRequest(ctx, "sqlite-key-hash", created.ID, now.Add(20*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, request.ID, usable.ID)
	grantExpiry := now.Add(30 * time.Minute)
	used, err := repo.UseEmergencyKey(ctx, "sqlite-key-hash", created.ID, now.Add(20*time.Minute), grantExpiry, "sqlite-grant-jti")
	require.NoError(t, err)
	assert.Equal(t, "used", used.Status)
	assert.Equal(t, "sqlite-grant-jti", used.GrantJTI)
	retry, err := repo.UseEmergencyKey(ctx, "sqlite-key-hash", created.ID, now.Add(25*time.Minute), now.Add(35*time.Minute), "different-jti")
	require.NoError(t, err)
	assert.Equal(t, "sqlite-grant-jti", retry.GrantJTI)
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "sqlite-key-hash", "wrong-device", now.Add(25*time.Minute))
	require.Error(t, err)
}

func deviceEmergencyCoverageRequest(id, userID, deviceID, status string, createdAt, requestExpiresAt time.Time) model.EmergencyKeyRequest {
	return model.EmergencyKeyRequest{
		ID: id, RequestedBy: userID, DeviceID: deviceID, Status: status,
		RequestExpiresAt: requestExpiresAt, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func deviceEmergencyCoverageOpenSQLite(t *testing.T) *ent.Client {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", databaseName))
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(driver))
	require.NoError(t, client.Schema.Create(context.Background()))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
