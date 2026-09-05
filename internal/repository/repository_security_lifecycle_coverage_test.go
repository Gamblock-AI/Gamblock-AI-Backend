package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositorySecurityLifecycle_ApprovalGrantPaths(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	_, err := repo.ApplyApprovedRequest(ctx, "approval-missing", "usr_other", "dev_other", now, "jti-missing")
	require.Error(t, err)

	resolvedAt := now.Add(-5 * time.Minute)
	partialAppliedAt := now.Add(-2 * time.Minute)
	partialExpiry := now.Add(20 * time.Minute)
	st.Lock()
	st.Approvals = append(st.Approvals,
		model.ApprovalRequest{
			ID: "approval-valid", UserID: "usr_cov", DeviceID: "dev_cov", Action: "pause_protection",
			Status: "approved", RequestedDurationMinutes: 30, ResolvedAt: &resolvedAt,
			ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		},
		model.ApprovalRequest{
			ID: "approval-partial", UserID: "usr_cov", DeviceID: "dev_partial", Action: "pause_protection",
			Status: "approved", RequestedDurationMinutes: 15, ResolvedAt: &resolvedAt,
			AppliedAt: &partialAppliedAt, GrantExpiresAt: &partialExpiry,
			ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		},
		model.ApprovalRequest{
			ID: "approval-expired", UserID: "usr_cov", DeviceID: "dev_expired", Action: "pause_protection",
			Status: "approved", RequestedDurationMinutes: 15,
			ResolvedAt: func() *time.Time { value := now.Add(-31 * time.Minute); return &value }(),
			ExpiresAt:  now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		},
		model.ApprovalRequest{
			ID: "approval-invalid-action", UserID: "usr_cov", DeviceID: "dev_invalid", Action: "unknown",
			Status: "approved", RequestedDurationMinutes: 15, ResolvedAt: &resolvedAt,
			ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		},
		model.ApprovalRequest{
			ID: "approval-invalid-duration", UserID: "usr_cov", DeviceID: "dev_duration", Action: "pause_protection",
			Status: "approved", RequestedDurationMinutes: 10, ResolvedAt: &resolvedAt,
			ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		},
	)
	st.Unlock()

	grant, err := repo.ApplyApprovedRequest(ctx, "approval-valid", "usr_cov", "dev_cov", now, "jti-valid")
	require.NoError(t, err)
	assert.Equal(t, "jti-valid", grant.GrantJTI)
	assert.Equal(t, now, grant.GrantStartsAt)
	assert.Equal(t, now.Add(30*time.Minute), grant.GrantExpiresAt)

	repeated, err := repo.ApplyApprovedRequest(ctx, "approval-valid", "usr_cov", "dev_cov", now.Add(time.Minute), "jti-replay")
	require.NoError(t, err)
	assert.Equal(t, "jti-valid", repeated.GrantJTI)
	assert.Equal(t, grant.GrantExpiresAt, repeated.GrantExpiresAt)

	partial, err := repo.ApplyApprovedRequest(ctx, "approval-partial", "usr_cov", "dev_partial", now, "jti-partial")
	require.NoError(t, err)
	assert.Equal(t, "jti-partial", partial.GrantJTI)
	assert.Equal(t, partialAppliedAt, partial.GrantStartsAt)
	assert.Equal(t, partialExpiry, partial.GrantExpiresAt)

	_, err = repo.ApplyApprovedRequest(ctx, "approval-expired", "usr_cov", "dev_expired", now, "jti-expired")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "window expired")
	_, err = repo.ApplyApprovedRequest(ctx, "approval-invalid-action", "usr_cov", "dev_invalid", now, "jti-invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be applied")
	_, err = repo.ApplyApprovedRequest(ctx, "approval-invalid-duration", "usr_cov", "dev_duration", now, "jti-invalid-duration")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duration")
	_, err = repo.ApplyApprovedRequest(ctx, "approval-valid", "usr_wrong", "dev_cov", now, "jti-owner")
	require.Error(t, err)
}

func TestRepositorySecurityLifecycle_StandaloneRemovalClaims(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	for _, values := range [][4]string{
		{"", "usr_cov", "dev_cov", "jti"},
		{"req", "", "dev_cov", "jti"},
		{"req", "usr_cov", "", "jti"},
		{"req", "usr_cov", "dev_cov", ""},
	} {
		_, err := repo.IssueStandaloneRemovalGrant(ctx, values[0], values[1], values[2], values[3], now)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete")
	}

	grant, err := repo.IssueStandaloneRemovalGrant(ctx, "standalone-cov-1", "usr_standalone", "dev_standalone", "jti-1", now)
	require.NoError(t, err)
	assert.Equal(t, standaloneRemovalGrantWindow, grant.GrantExpiresAt.Sub(grant.GrantStartsAt))

	_, err = repo.IssueStandaloneRemovalGrant(ctx, "standalone-cov-2", "usr_standalone", "dev_standalone", "jti-2", now.Add(time.Minute))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already active")
	_, err = repo.IssueStandaloneRemovalGrant(ctx, "standalone-cov-3", "usr_standalone", "dev_other", "jti-3", now.Add(time.Minute))
	require.NoError(t, err)
}

func TestRepositorySecurityLifecycle_EmergencyTransitions(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	request, err := repo.CreateEmergencyKeyRequest(ctx, model.EmergencyKeyRequest{
		ID: "emergency-cov", RequestedBy: "usr_cov", DeviceID: "dev_cov", Status: "pending",
		RequestExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.Equal(t, "emergency-cov", request.ID)

	reviewed, err := repo.ReviewEmergencyKeyRequest(ctx, "emergency-cov", "usr_admin", now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "reviewed", reviewed.Status)
	assert.Equal(t, "usr_admin", reviewed.ReviewedBy)

	keyExpiry := now.Add(2 * time.Hour)
	approved, err := repo.ApproveEmergencyKeyRequest(ctx, "emergency-cov", "usr_admin", "key-hash-cov", now.Add(2*time.Minute), keyExpiry)
	require.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)
	assert.Equal(t, "key-hash-cov", approved.KeyHash)

	usable, err := repo.GetUsableEmergencyKeyRequest(ctx, "key-hash-cov", "dev_cov", now.Add(3*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "approved", usable.Status)
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "key-hash-cov", "dev_other", now.Add(3*time.Minute))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device")

	useAt := now.Add(4 * time.Minute)
	_, err = repo.UseEmergencyKey(ctx, "key-hash-cov", "dev_cov", useAt, useAt.Add(9*time.Minute), "grant-jti-cov")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata")
	used, err := repo.UseEmergencyKey(ctx, "key-hash-cov", "dev_cov", useAt, useAt.Add(10*time.Minute), "grant-jti-cov")
	require.NoError(t, err)
	assert.Equal(t, "used", used.Status)
	assert.Equal(t, "grant-jti-cov", used.GrantJTI)

	retry, err := repo.UseEmergencyKey(ctx, "key-hash-cov", "dev_cov", useAt.Add(time.Minute), useAt.Add(11*time.Minute), "another-jti")
	require.NoError(t, err)
	assert.Equal(t, "grant-jti-cov", retry.GrantJTI)
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "key-hash-cov", "dev_cov", useAt.Add(11*time.Minute))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")

	_, err = repo.GetCurrentEmergencyKeyRequest(ctx, "usr_unknown", "dev_unknown", now)
	require.Error(t, err)

	expiredID := "emergency-cov-expired"
	st.Lock()
	st.EmergencyKeyRequests = append(st.EmergencyKeyRequests, model.EmergencyKeyRequest{
		ID: expiredID, RequestedBy: "usr_cov", DeviceID: "dev_expired", Status: "pending",
		RequestExpiresAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	st.Unlock()
	_, err = repo.ReviewEmergencyKeyRequest(ctx, expiredID, "usr_admin", now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
	pending, err := repo.GetPendingEmergencyKeyRequests(ctx, now)
	require.NoError(t, err)
	for _, item := range pending {
		assert.NotEqual(t, expiredID, item.ID)
	}
	history, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Bucket: "history", Query: "grant"})
	require.NoError(t, err)
	assert.NotNil(t, history.Items)
}

func TestRepositorySecurityLifecycle_TokenAndContactOwnership(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	deviceID := "dev_token_cov"

	require.NoError(t, repo.CreateRefreshTokenWithAuthTime(ctx, "rt-cov", "usr_cov", "hash-cov", &deviceID, now, now.Add(time.Hour)))
	rtID, userID, gotDevice, authTime, err := repo.GetActiveRefreshTokenSession(ctx, "hash-cov")
	require.NoError(t, err)
	assert.Equal(t, "rt-cov", rtID)
	assert.Equal(t, "usr_cov", userID)
	require.NotNil(t, gotDevice)
	assert.Equal(t, deviceID, *gotDevice)
	assert.Equal(t, now, authTime)
	_, _, _, err = repo.GetActiveRefreshToken(ctx, "hash-cov")
	require.NoError(t, err)
	require.NoError(t, repo.ConsumeRefreshTokenByID(ctx, "rt-cov"))
	require.Error(t, repo.ConsumeRefreshTokenByID(ctx, "rt-cov"))

	require.NoError(t, repo.CreateRefreshToken(ctx, "rt-expired-cov", "usr_cov", "hash-expired-cov", nil, now.Add(-time.Minute)))
	_, _, _, err = repo.GetActiveRefreshToken(ctx, "hash-expired-cov")
	require.Error(t, err)
	require.NoError(t, repo.RevokeRefreshToken(ctx, "missing-hash-cov"))
	require.NoError(t, repo.CreateRefreshToken(ctx, "rt-user-1", "usr_cov", "hash-user-1", nil, now.Add(time.Hour)))
	require.NoError(t, repo.CreateRefreshToken(ctx, "rt-user-2", "usr_cov", "hash-user-2", nil, now.Add(time.Hour)))
	require.NoError(t, repo.RevokeRefreshTokensForUser(ctx, "usr_cov"))
	_, _, _, err = repo.GetActiveRefreshToken(ctx, "hash-user-1")
	require.Error(t, err)

	oldCreated := now.Add(-2 * time.Hour)
	newCreated := now.Add(-time.Hour)
	require.NoError(t, repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "contact-old-cov", UserID: "usr_cov", Kind: "phone", Destination: "+62000",
		TokenHash: "old-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: oldCreated,
		ConsumedAt: func() *time.Time { value := oldCreated; return &value }(),
	}))
	require.NoError(t, repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "contact-new-cov", UserID: "usr_cov", Kind: "phone", Destination: "+62000",
		TokenHash: "new-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: newCreated,
	}))
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+62000", "wrong-hash", now, 2)
	require.Error(t, err)
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+62000", "wrong-hash", now, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
	verified, err := repo.VerifyLatestContactCode(ctx, "phone", "+62000", "new-hash", now, 2)
	require.NoError(t, err)
	assert.Equal(t, "contact-new-cov", verified.ID)
	assert.NotNil(t, verified.ConsumedAt)

	require.NoError(t, repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "contact-consume-cov", UserID: "usr_cov", Kind: "email", Destination: "cov@example.test",
		TokenHash: "consume-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}))
	consumed, err := repo.ConsumeContactVerification(ctx, "consume-hash", "email", now)
	require.NoError(t, err)
	assert.Equal(t, "contact-consume-cov", consumed.ID)
	_, err = repo.ConsumeContactVerification(ctx, "consume-hash", "email", now)
	require.Error(t, err)
	require.NoError(t, repo.SaveContactVerification(ctx, model.ContactVerification{
		ID: "contact-invalidated-cov", UserID: "usr_cov", Kind: "email", Destination: "cov@example.test",
		TokenHash: "invalidated-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(time.Second),
	}))
	require.NoError(t, repo.InvalidateContactVerifications(ctx, "email", "cov@example.test", now.Add(2*time.Second)))
	_, err = repo.ConsumeContactVerification(ctx, "invalidated-hash", "email", now.Add(3*time.Second))
	require.Error(t, err)
	_, err = repo.ConsumeContactVerification(ctx, "missing-hash", "email", now)
	require.Error(t, err)

	require.NoError(t, repo.MarkEmailVerified(ctx, "usr_gading", now))
	require.NoError(t, repo.MarkPhoneVerified(ctx, "usr_gading", "+62111", now))
	require.NoError(t, repo.SetPendingPhone(ctx, "usr_gading", "+62222"))
	require.Error(t, repo.MarkEmailVerified(ctx, "missing-user-cov", now))
	require.Error(t, repo.MarkPhoneVerified(ctx, "missing-user-cov", "+62000", now))
	require.Error(t, repo.SetPendingPhone(ctx, "missing-user-cov", "+62000"))

	snapshot := st.Snapshot()
	for _, item := range snapshot.ContactVerifications {
		if item.ID == "contact-new-cov" {
			assert.Equal(t, 1, item.AttemptCount)
		}
	}
}

func TestRepositorySecurityLifecycle_DeviceGrantAndOwnership(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	modelVersion, rulesetVersion := "model-cov", "rules-cov"
	device, err := repo.CreateDevice(ctx, "device-security-cov", "usr_cov", "instance-security-cov", "android", "Cov phone", "1.0", "Android", &modelVersion, &rulesetVersion)
	require.NoError(t, err)
	assert.Equal(t, "inactive", device.ProtectionStatus)

	upserted, err := repo.UpsertDevice(ctx, "ignored-device-id", "usr_cov", "instance-security-cov", "android", "Updated phone", "2.0", "Android 15", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "device-security-cov", upserted.ID)
	assert.Equal(t, "Updated phone", upserted.Label)
	assert.True(t, repo.IsDeviceOwnedBy(ctx, "device-security-cov", "usr_cov"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "device-security-cov", "usr_other"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "", "usr_cov"))

	updated, err := repo.UpdateOwnedDevice(ctx, "usr_cov", "device-security-cov", "Owned update", "2.1", "Android 15", "active", "model-2", "rules-2")
	require.NoError(t, err)
	assert.Equal(t, "active", updated.ProtectionStatus)
	_, err = repo.UpdateOwnedDevice(ctx, "usr_other", "device-security-cov", "bad", "", "", "", "", "")
	require.Error(t, err)
	require.NoError(t, repo.RecordOwnedHeartbeat(ctx, "usr_cov", "device-security-cov"))
	require.Error(t, repo.RecordOwnedHeartbeat(ctx, "usr_other", "device-security-cov"))
	_, err = repo.UpdateOwnedDevice(ctx, "usr_cov", "missing-device-cov", "", "", "", "", "", "")
	require.Error(t, err)

	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "usr_cov", "device-security-cov")
	require.Error(t, err)
	require.Error(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_other", "device-security-cov", "jwk", "thumb"))
	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_cov", "device-security-cov", "jwk", "thumb"))
	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_cov", "device-security-cov", "jwk", "thumb"))
	assert.ErrorIs(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_cov", "device-security-cov", "other-jwk", "other-thumb"), ErrDeviceGrantKeyConflict)
	thumbprint, err := repo.OwnedDeviceGrantKeyThumbprint(ctx, "usr_cov", "device-security-cov")
	require.NoError(t, err)
	assert.Equal(t, "thumb", thumbprint)
	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "usr_other", "device-security-cov")
	require.Error(t, err)
}

func TestRepositorySecurityLifecycle_ReminderAndPushOwnership(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	defaultPreference, err := repo.ReminderPreference(ctx, "usr_reminder_cov")
	require.NoError(t, err)
	assert.False(t, defaultPreference.Enabled)
	assert.Equal(t, defaultReminderTimezone, defaultPreference.Timezone)

	preference, err := repo.UpsertReminderPreference(ctx, "usr_reminder_cov", true, "08:30", "Asia/Jakarta", "id")
	require.NoError(t, err)
	assert.True(t, preference.Enabled)
	updated, err := repo.UpsertReminderPreference(ctx, "usr_reminder_cov", false, "20:15", "Europe/London", "en")
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	require.NoError(t, repo.MarkReminderFired(ctx, "missing-reminder-cov", now))
	require.NoError(t, repo.MarkReminderFired(ctx, "usr_reminder_cov", now))
	preference, err = repo.UpsertReminderPreference(ctx, "usr_reminder_cov", true, "20:15", "Europe/London", "en")
	require.NoError(t, err)
	require.NotNil(t, preference.LastFiredAt)
	enabled, err := repo.EnabledReminderPreferences(ctx)
	require.NoError(t, err)
	assert.Len(t, enabled, 1)

	userAgent := "coverage-agent"
	subscription, err := repo.UpsertPushSubscription(ctx, "usr_reminder_cov", "https://push.example/cov", "p256dh-1", "auth-1", &userAgent)
	require.NoError(t, err)
	assert.Equal(t, "usr_reminder_cov", subscription.UserID)
	updatedSubscription, err := repo.UpsertPushSubscription(ctx, "usr_other", "https://push.example/cov", "p256dh-2", "auth-2", nil)
	require.NoError(t, err)
	assert.Equal(t, subscription.ID, updatedSubscription.ID)
	assert.Equal(t, "usr_other", updatedSubscription.UserID)
	assert.Nil(t, updatedSubscription.UserAgent)

	owned, err := repo.PushSubscriptionsForUser(ctx, "usr_other")
	require.NoError(t, err)
	assert.Len(t, owned, 1)
	wrongOwner, err := repo.PushSubscriptionsForUser(ctx, "usr_reminder_cov")
	require.NoError(t, err)
	assert.Empty(t, wrongOwner)
	require.NoError(t, repo.DeletePushSubscription(ctx, "usr_reminder_cov", "https://push.example/cov"))
	owned, err = repo.PushSubscriptionsForUser(ctx, "usr_other")
	require.NoError(t, err)
	assert.Len(t, owned, 1)
	require.NoError(t, repo.RemovePushSubscriptionByID(ctx, subscription.ID))
	require.NoError(t, repo.RemovePushSubscriptionByID(ctx, "missing-subscription-cov"))
	owned, err = repo.PushSubscriptionsForUser(ctx, "usr_other")
	require.NoError(t, err)
	assert.Empty(t, owned)
}

func TestRepositorySecurityLifecycle_RecoveryOwnershipAndRetention(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	record, err := repo.SaveRecoveryRecord(ctx, model.RecoveryRecord{
		ID: "recovery-cov", UserID: "usr_recovery_cov", Kind: "weekly_review", RecordDate: "2026-09-05",
		Metadata: map[string]any{"outcome": "steady"}, Content: "encrypted-content", Status: "completed",
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, "recovery-cov", record.ID)
	record.Content = "encrypted-content-v2"
	record.Status = "updated"
	updated, err := repo.SaveRecoveryRecord(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, "encrypted-content-v2", updated.Content)

	oldRecord := model.RecoveryRecord{ID: "recovery-old-cov", UserID: "usr_recovery_cov", Kind: "checkin", RecordDate: "2026-09-01", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)}
	otherOld := oldRecord
	otherOld.ID = "recovery-other-old-cov"
	otherOld.UserID = "usr_other"
	_, err = repo.SaveRecoveryRecord(ctx, oldRecord)
	require.NoError(t, err)
	_, err = repo.SaveRecoveryRecord(ctx, otherOld)
	require.NoError(t, err)
	records, err := repo.ListRecoveryRecords(ctx, "usr_recovery_cov", now.Add(-2*time.Hour))
	require.NoError(t, err)
	assert.Len(t, records, 1)
	require.NoError(t, repo.DeleteExpiredRecoveryRecords(ctx, "usr_recovery_cov", now.Add(-2*time.Hour)))
	records, err = repo.ListRecoveryRecords(ctx, "usr_recovery_cov", time.Time{})
	require.NoError(t, err)
	for _, item := range records {
		assert.NotEqual(t, "recovery-old-cov", item.ID)
	}

	st.Lock()
	st.RecoveryPracticeSessions = append(st.RecoveryPracticeSessions,
		model.RecoveryPracticeSession{ID: "practice-old-cov", UserID: "usr_recovery_cov", PracticeKind: "breathing", CompletedAt: now.Add(-3 * time.Hour), CreatedAt: now.Add(-3 * time.Hour)},
		model.RecoveryPracticeSession{ID: "practice-new-cov", UserID: "usr_recovery_cov", PracticeKind: "grounding", CompletedAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour)},
		model.RecoveryPracticeSession{ID: "practice-other-cov", UserID: "usr_other", PracticeKind: "breathing", CompletedAt: now.Add(-3 * time.Hour), CreatedAt: now.Add(-3 * time.Hour)},
	)
	st.Unlock()
	practices, err := repo.ListRecoveryPracticeSessions(ctx, "usr_recovery_cov", now.Add(-2*time.Hour))
	require.NoError(t, err)
	assert.Len(t, practices, 1)
	require.NoError(t, repo.DeleteExpiredRecoveryPracticeSessions(ctx, "usr_recovery_cov", now.Add(-2*time.Hour)))
	practices, err = repo.ListRecoveryPracticeSessions(ctx, "usr_recovery_cov", time.Time{})
	require.NoError(t, err)
	assert.Len(t, practices, 1)

	space, found, err := repo.GetRecoverySpace(ctx, "usr_recovery_cov")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, space.ID)
	savedSpace, err := repo.SaveRecoverySpace(ctx, model.RecoverySpace{ID: "space-cov", UserID: "usr_recovery_cov", Theme: "calm", UnlockedItems: []string{"one"}, PlacedItems: map[string]any{"one": "center"}, UnlockRuleVersion: 1, CreatedAt: now})
	require.NoError(t, err)
	assert.Equal(t, "space-cov", savedSpace.ID)
	savedSpace.Theme = "focused"
	savedSpace.UnlockedItems = []string{"one", "two"}
	updatedSpace, err := repo.SaveRecoverySpace(ctx, savedSpace)
	require.NoError(t, err)
	assert.Equal(t, "focused", updatedSpace.Theme)
	assert.Equal(t, savedSpace.CreatedAt, updatedSpace.CreatedAt)

	granted, capReached, err := repo.GrantWeeklyReviewExperience(ctx, "usr_recovery_cov", "2026-09-01")
	require.NoError(t, err)
	assert.True(t, granted)
	assert.False(t, capReached)
	granted, capReached, err = repo.GrantWeeklyReviewExperience(ctx, "usr_recovery_cov", "2026-09-01")
	require.NoError(t, err)
	assert.False(t, granted)
	assert.False(t, capReached)
	for index := 2; index <= 5; index++ {
		granted, capReached, err = repo.GrantWeeklyReviewExperience(ctx, "usr_recovery_cov", "2026-09-0"+string(rune('0'+index)))
		require.NoError(t, err)
		assert.True(t, granted)
		assert.False(t, capReached)
	}
	granted, capReached, err = repo.GrantWeeklyReviewExperience(ctx, "usr_recovery_cov", "2026-09-06")
	require.NoError(t, err)
	assert.False(t, granted)
	assert.True(t, capReached)

	evidence := repo.RecoveryUnlockEvidence(ctx, "usr_gading")
	assert.GreaterOrEqual(t, evidence.TotalPractices, 1)
	assert.GreaterOrEqual(t, evidence.FocusJournals, 0)
}

func TestRepositorySecurityLifecycle_SPKRecordsAndPrivacySafeInputs(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	preference, err := repo.SpkPreference(ctx, "usr_spk_cov")
	require.NoError(t, err)
	assert.True(t, preference.SpkRecommendationEnabled)
	preference, err = repo.UpsertSpkPreference(ctx, "usr_spk_cov", model.SpkPreference{
		SpkRecommendationEnabled: false, SpkUseProtection: false, SpkUseRecovery: true,
		SpkUsePersonal: false, LLMPersonalizationEnabled: false,
	})
	require.NoError(t, err)
	assert.False(t, preference.SpkRecommendationEnabled)
	preference, err = repo.UpsertSpkPreference(ctx, "usr_spk_cov", model.SpkPreference{SpkRecommendationEnabled: true, SpkUseProtection: true, SpkUseRecovery: false, SpkUsePersonal: true, LLMPersonalizationEnabled: true})
	require.NoError(t, err)
	assert.True(t, preference.SpkRecommendationEnabled)
	assert.False(t, preference.SpkUseRecovery)

	noRecord, err := repo.TodayInterventionRecord(ctx, "usr_spk_cov", now.Add(-time.Hour), now)
	require.NoError(t, err)
	assert.Nil(t, noRecord)
	record := model.InterventionRecord{
		ID: "intervention-cov", UserID: "usr_spk_cov", InterventionKey: "pause", ResponseType: "breathing",
		SupportLevel: "moderate", EngagementLevel: "medium", ReadinessLevel: "ready", Status: "recommended",
		RecommendedAt: now.Add(-30 * time.Minute), EffectivenessStatus: "unknown",
	}
	stored, err := repo.UpsertInterventionRecord(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, record.ID, stored.ID)
	records, err := repo.InterventionRecords(ctx, "usr_spk_cov")
	require.NoError(t, err)
	assert.Len(t, records, 1)
	today, err := repo.TodayInterventionRecord(ctx, "usr_spk_cov", now.Add(-time.Hour), now)
	require.NoError(t, err)
	require.NotNil(t, today)
	assert.Equal(t, record.ID, today.ID)
	_, err = repo.CompleteIntervention(ctx, "usr_other", record.ID)
	require.Error(t, err)
	completed, err := repo.CompleteIntervention(ctx, "usr_spk_cov", record.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
	record.PersonalizedMessage = "Take a calm pause."
	record.LLMUsed = true
	_, err = repo.UpdateInterventionRecord(ctx, "usr_other", record)
	require.Error(t, err)
	updated, err := repo.UpdateInterventionRecord(ctx, "usr_spk_cov", record)
	require.NoError(t, err)
	assert.Equal(t, "Take a calm pause.", updated.PersonalizedMessage)
	require.Error(t, repo.UpdateInterventionEffectiveness(ctx, "usr_other", record.ID, "helped"))
	require.NoError(t, repo.UpdateInterventionEffectiveness(ctx, "usr_spk_cov", record.ID, "helped"))
	assert.Error(t, repo.UpdateInterventionEffectiveness(ctx, "usr_spk_cov", "missing-intervention-cov", "helped"))

	assert.Equal(t, 0, func() int {
		count, saveErr := repo.SaveBlockedEvents(ctx, "usr_spk_cov", "dev_cov", nil)
		require.NoError(t, saveErr)
		return count
	}())
	eventAt := now.Add(-time.Hour)
	created, err := repo.SaveBlockedEvents(ctx, "usr_spk_cov", "dev_cov", []time.Time{eventAt, eventAt, eventAt.Add(time.Minute)})
	require.NoError(t, err)
	assert.Equal(t, 2, created)
	created, err = repo.SaveBlockedEvents(ctx, "usr_other", "dev_other", []time.Time{eventAt})
	require.NoError(t, err)
	assert.Equal(t, 1, created)

	st.Lock()
	st.AggregateEvents = append(st.AggregateEvents,
		model.AggregateEvent{ID: "spk-count-1", UserID: "usr_spk_cov", EventType: "block_count_sync", EventDate: now.Add(-30 * time.Minute), Count: 3},
		model.AggregateEvent{ID: "spk-count-2", UserID: "usr_spk_cov", EventType: "block_count_sync", EventDate: now.Add(-20 * time.Minute), Count: 2},
		model.AggregateEvent{ID: "spk-other-type", UserID: "usr_spk_cov", EventType: "intervention_shown", EventDate: now.Add(-10 * time.Minute), Count: 9},
	)
	st.Unlock()
	counts, err := repo.BlockCountsByDate(ctx, "usr_spk_cov", now.Add(-time.Hour), now)
	require.NoError(t, err)
	assert.Equal(t, 5, counts[now.Format("2006-01-02")])

	_, err = repo.SaveIntention(ctx, "usr_gading", "I want to protect my studies", "active", model.Intention{
		SchoolImpact: "moderate", MoneySpent: "low", ScreenTime: "high", QuitAttempts: "some", QuitMotivation: "ready",
	})
	require.NoError(t, err)
	_, err = repo.SaveBlockedEvents(ctx, "usr_gading", "dev_android", []time.Time{eventAt.Add(2 * time.Minute)})
	require.NoError(t, err)
	input, err := repo.SpkInputData(ctx, "usr_gading", now)
	require.NoError(t, err)
	assert.True(t, input.HasActivity)
	assert.True(t, input.HasIntention)
	assert.Equal(t, "I want to protect my studies", input.PersonalIntention)
	assert.NotEmpty(t, input.BlockedEventTimes)
	assert.NotContains(t, input.PersonalIntention, "http")

	_, _, _, err = repo.GetActiveRefreshToken(ctx, "missing-security-cov")
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrDeviceGrantKeyConflict))
}
