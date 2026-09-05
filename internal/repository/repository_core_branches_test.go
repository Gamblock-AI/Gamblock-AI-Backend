package repository

import (
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_InMemoryLifecycleAndErrorPaths(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()

	created, err := repo.CreateProvisionedUserWithPhone(ctx, "usr_cov_user", "Coverage@Example.com", "Coverage User", "+628111111111", "hash", "user", true)
	require.NoError(t, err)
	assert.Equal(t, "usr_cov_user", created.ID)
	assert.True(t, created.MustChangePassword)
	assert.Equal(t, "+628111111111", created.PhoneE164)

	_, err = repo.CreateUser(ctx, "usr_cov_duplicate", "coverage@example.com", "Duplicate")
	require.EqualError(t, err, "email already exists")

	got, ok := repo.UserByEmail(ctx, "COVERAGE@example.COM")
	require.True(t, ok)
	assert.Equal(t, "usr_cov_user", got.ID)
	got, ok = repo.UserByID(ctx, "usr_cov_user")
	require.True(t, ok)
	assert.Equal(t, "Coverage User", got.DisplayName)
	_, ok = repo.UserByID(ctx, "missing-user")
	assert.False(t, ok)

	updated, err := repo.UpdateUserDisplayName(ctx, "usr_cov_user", "Updated User")
	require.NoError(t, err)
	assert.Equal(t, "Updated User", updated.DisplayName)
	_, err = repo.UpdateUserDisplayName(ctx, "missing-user", "Nope")
	assert.EqualError(t, err, "user not found")

	avatarKey := "avatar/usr_cov_user.webp"
	updated, err = repo.UpdateUserAvatar(ctx, "usr_cov_user", &avatarKey)
	require.NoError(t, err)
	assert.Equal(t, "/v1/users/usr_cov_user/avatar", *updated.AvatarURL)
	storageKey, ok := repo.UserAvatarStorageKey(ctx, "usr_cov_user")
	require.True(t, ok)
	assert.Equal(t, avatarKey, storageKey)

	invalidAvatar := "uploads/not-an-avatar.webp"
	_, err = repo.UpdateUserAvatar(ctx, "usr_cov_user", &invalidAvatar)
	require.NoError(t, err)
	_, ok = repo.UserAvatarStorageKey(ctx, "usr_cov_user")
	assert.False(t, ok)
	_, ok = repo.UserAvatarStorageKey(ctx, "missing-user")
	assert.False(t, ok)
	_, err = repo.UpdateUserAvatar(ctx, "usr_cov_user", nil)
	assert.NoError(t, err)
	_, err = repo.UpdateUserAvatar(ctx, "missing-user", nil)
	assert.EqualError(t, err, "user not found")

	require.NoError(t, repo.UpdateUserPasswordHash(ctx, "usr_cov_user", "new-hash"))
	assert.EqualError(t, repo.UpdateUserPasswordHash(ctx, "missing-user", "hash"), "user not found")
}

func TestContactVerification_InMemoryLatestCodeAndExpiryRules(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	old := model.ContactVerification{
		ID: "cv_old", UserID: "usr_gading", Kind: "phone", Destination: "+628123", TokenHash: "old-hash",
		ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-2 * time.Minute),
	}
	latest := old
	latest.ID, latest.TokenHash, latest.CreatedAt = "cv_latest", "latest-hash", now.Add(-time.Minute)
	expired := old
	expired.ID, expired.TokenHash, expired.ExpiresAt, expired.CreatedAt = "cv_expired", "expired-hash", now.Add(-time.Second), now.Add(-3*time.Minute)
	require.NoError(t, repo.SaveContactVerification(ctx, old))
	require.NoError(t, repo.SaveContactVerification(ctx, latest))
	require.NoError(t, repo.SaveContactVerification(ctx, expired))

	_, err := repo.VerifyLatestContactCode(ctx, "phone", "+628123", "wrong", now, 2)
	require.EqualError(t, err, "verification code is invalid or expired")
	snapshot := st.Snapshot()
	var latestStored model.ContactVerification
	for _, item := range snapshot.ContactVerifications {
		if item.ID == latest.ID {
			latestStored = item
		}
	}
	assert.Equal(t, 1, latestStored.AttemptCount)

	verified, err := repo.VerifyLatestContactCode(ctx, "phone", "+628123", "latest-hash", now, 2)
	require.NoError(t, err)
	assert.Equal(t, latest.ID, verified.ID)
	assert.NotNil(t, verified.ConsumedAt)

	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628123", "latest-hash", now, 2)
	require.EqualError(t, err, "verification code is invalid or expired")
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628123", "old-hash", now, 2)
	require.NoError(t, err)

	limited := latest
	limited.ID, limited.TokenHash, limited.CreatedAt, limited.ConsumedAt, limited.AttemptCount = "cv_limited", "limited-hash", now, nil, 1
	require.NoError(t, repo.SaveContactVerification(ctx, limited))
	_, err = repo.VerifyLatestContactCode(ctx, "phone", "+628123", "wrong", now, 1)
	require.EqualError(t, err, "verification attempt limit reached")

	_, err = repo.ConsumeContactVerification(ctx, "missing", "phone", now)
	require.EqualError(t, err, "verification token is invalid or expired")
	consumable := model.ContactVerification{ID: "cv_consume", UserID: "usr_gading", Kind: "email", Destination: "gading@example.com", TokenHash: "consume-hash", ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	require.NoError(t, repo.SaveContactVerification(ctx, consumable))
	consumed, err := repo.ConsumeContactVerification(ctx, "consume-hash", "email", now)
	require.NoError(t, err)
	assert.Equal(t, consumable.ID, consumed.ID)
	_, err = repo.ConsumeContactVerification(ctx, "consume-hash", "email", now)
	require.EqualError(t, err, "verification token is invalid or expired")

	require.NoError(t, repo.InvalidateContactVerifications(ctx, "email", "gading@example.com", now))
	snapshot = st.Snapshot()
	for _, item := range snapshot.ContactVerifications {
		if item.ID == consumable.ID {
			assert.NotNil(t, item.ConsumedAt)
		}
	}
}

func TestContactVerification_MarksOwnedUserContacts(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	verifiedAt := time.Now().UTC()

	require.NoError(t, repo.MarkEmailVerified(ctx, "usr_gading", verifiedAt))
	require.NoError(t, repo.MarkPhoneVerified(ctx, "usr_gading", "+628999999999", verifiedAt))
	require.NoError(t, repo.SetPendingPhone(ctx, "usr_gading", "+628888888888"))
	user, ok := repo.UserByID(ctx, "usr_gading")
	require.True(t, ok)
	assert.Equal(t, "+628888888888", user.PhoneE164)
	assert.Nil(t, user.PhoneVerifiedAt)

	assert.EqualError(t, repo.MarkEmailVerified(ctx, "missing-user", verifiedAt), "user not found")
	assert.EqualError(t, repo.MarkPhoneVerified(ctx, "missing-user", "+1", verifiedAt), "user not found")
	assert.EqualError(t, repo.SetPendingPhone(ctx, "missing-user", "+1"), "user not found")
}

func TestDeviceRepository_InMemoryUpsertOwnershipAndGrantKey(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()

	modelVersion := "model-cov"
	device, err := repo.CreateDevice(ctx, "dev_cov", "usr_gading", "client-cov", "android", "Coverage device", "2.0", "Android", &modelVersion, nil)
	require.NoError(t, err)
	assert.Equal(t, "inactive", device.ProtectionStatus)

	updated, err := repo.UpsertDevice(ctx, "dev_ignored", "usr_gading", "client-cov", "windows", "Updated device", "3.0", "Windows", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "dev_cov", updated.ID)
	assert.Equal(t, "windows", updated.Platform)
	assert.Empty(t, updated.ModelVersion)

	created, err := repo.UpsertDevice(ctx, "dev_cov_new", "usr_gading", "new-client", "android", "New device", "1.0", "Android", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "dev_cov_new", created.ID)

	updated, err = repo.UpdateOwnedDevice(ctx, "usr_gading", "dev_cov", "Owned label", "", "", "active", "model-2", "rules-2")
	require.NoError(t, err)
	assert.Equal(t, "Owned label", updated.Label)
	assert.Equal(t, "active", updated.ProtectionStatus)
	assert.EqualError(t, func() error {
		_, err := repo.UpdateOwnedDevice(ctx, "usr_dery", "dev_cov", "", "", "", "", "", "")
		return err
	}(), "device not found")
	assert.EqualError(t, func() error {
		_, err := repo.UpdateOwnedDevice(ctx, "usr_gading", "missing-device", "", "", "", "", "", "")
		return err
	}(), "device not found")

	require.NoError(t, repo.RecordOwnedHeartbeat(ctx, "usr_gading", "dev_cov"))
	assert.EqualError(t, repo.RecordOwnedHeartbeat(ctx, "usr_dery", "dev_cov"), "device not found")
	assert.True(t, repo.IsDeviceOwnedBy(ctx, "dev_cov", "usr_gading"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "dev_cov", "usr_dery"))
	assert.False(t, repo.IsDeviceOwnedBy(ctx, "", "usr_gading"))

	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "usr_gading", "dev_cov")
	assert.EqualError(t, err, "device grant key is not enrolled")
	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_gading", "dev_cov", "jwk-cov", "thumb-cov"))
	require.NoError(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_gading", "dev_cov", "jwk-cov", "thumb-cov"))
	assert.ErrorIs(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_gading", "dev_cov", "other-jwk", "other-thumb"), ErrDeviceGrantKeyConflict)
	thumbprint, err := repo.OwnedDeviceGrantKeyThumbprint(ctx, "usr_gading", "dev_cov")
	require.NoError(t, err)
	assert.Equal(t, "thumb-cov", thumbprint)
	assert.EqualError(t, repo.BindOwnedDeviceGrantKey(ctx, "usr_dery", "dev_cov", "jwk", "thumb"), "device not found")
	_, err = repo.OwnedDeviceGrantKeyThumbprint(ctx, "usr_dery", "dev_cov")
	assert.EqualError(t, err, "device not found")
	assert.GreaterOrEqual(t, len(st.Snapshot().Devices), 5)
}

func TestAggregateRepository_InMemoryIdempotencyAndAnalytics(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()
	event := model.AggregateEvent{ID: "agg_cov", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "cov-key", EventType: "block_count_sync", EventDate: now, Count: 7}

	first, err := repo.SaveAggregateEvent(ctx, event)
	require.NoError(t, err)
	second, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{ID: "agg_other", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "cov-key", EventType: "block_count_sync", EventDate: now, Count: 99})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 7, second.Count)

	otherUser, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{ID: "agg_other_user", UserID: "usr_dery", DeviceID: "dev_dery_android", IdempotencyKey: "cov-key", EventType: "block_count_sync", EventDate: now, Count: 3})
	require.NoError(t, err)
	assert.Equal(t, "agg_other_user", otherUser.ID)

	snapshotKey := "cov-snapshot"
	snapshot, err := repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{ID: "agg_snapshot", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: snapshotKey, EventType: "block_count_sync", EventDate: now, Count: 10, MetadataJSON: map[string]any{"version": 1}})
	require.NoError(t, err)
	assert.Equal(t, 10, snapshot.Count)
	stale, err := repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: snapshotKey, Count: 2, MetadataJSON: map[string]any{"version": 2}})
	require.NoError(t, err)
	assert.Equal(t, 10, stale.Count)
	higher, err := repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: snapshotKey, Count: 12, MetadataJSON: map[string]any{"version": 3}})
	require.NoError(t, err)
	assert.Equal(t, 12, higher.Count)
	assert.Equal(t, 3, higher.MetadataJSON["version"])

	analytics, err := repo.GetProtectionAnalytics(ctx, "usr_gading", "dev_android", 3, now)
	require.NoError(t, err)
	assert.Equal(t, "local_only", analytics.DataState)
	assert.GreaterOrEqual(t, analytics.Totals.Blocked, 12)
	_, err = repo.GetProtectionAnalytics(ctx, "usr_dery", "dev_android", 3, now)
	require.EqualError(t, err, "device does not belong to user")
}

func TestApprovalRepository_InMemoryTransitionsAndGrantLifecycle(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	request, err := repo.CreateApprovalRequestWithToken(ctx, "apr_cov", "usr_gading", "dev_android", "mbr_active", "pause_protection", "coverage", 15, now.Add(time.Hour), "quick-cov")
	require.NoError(t, err)
	assert.Equal(t, "Pause protection for 15 minutes", request.ActionLabel)
	quick, err := repo.GetApprovalByQuickToken(ctx, "quick-cov")
	require.NoError(t, err)
	assert.Equal(t, request.ID, quick.ID)
	_, err = repo.GetApprovalByQuickToken(ctx, "missing-quick")
	require.EqualError(t, err, "token not found")

	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "apr_cov", "usr_suci", "invalid", ""), "invalid approval status")
	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "apr_cov", "usr_dery", "approved", ""), "pending approval request not found")
	require.NoError(t, repo.ResolveApprovalAsPartner(ctx, "apr_cov", "usr_suci", "approved", "Take a breath"))
	grant, err := repo.ApplyApprovedRequest(ctx, "apr_cov", "usr_gading", "dev_android", now, "grant-cov")
	require.NoError(t, err)
	assert.Equal(t, "grant-cov", grant.GrantJTI)
	assert.Equal(t, "pause_protection", grant.Action)
	secondGrant, err := repo.ApplyApprovedRequest(ctx, "apr_cov", "usr_gading", "dev_android", now.Add(time.Minute), "different-jti")
	require.NoError(t, err)
	assert.Equal(t, "grant-cov", secondGrant.GrantJTI)
	_, err = repo.ApplyApprovedRequest(ctx, "apr_cov", "usr_dery", "dev_android", now, "wrong-user")
	require.EqualError(t, err, "approval request not found")

	pending, err := repo.CreateApprovalRequestWithToken(ctx, "apr_cancel", "usr_gading", "dev_android", "mbr_active", "pause_protection", "cancel", 30, now.Add(time.Hour), "quick-cancel")
	require.NoError(t, err)
	assert.Equal(t, "pending", pending.Status)
	assert.EqualError(t, repo.CancelApprovalRequest(ctx, "apr_cancel", "usr_dery"), "pending approval request not found")
	require.NoError(t, repo.CancelApprovalRequest(ctx, "apr_cancel", "usr_gading"))
	assert.EqualError(t, repo.CancelApprovalRequest(ctx, "apr_cancel", "usr_gading"), "pending approval request not found")

	_, err = repo.CreateApprovalRequestWithToken(ctx, "apr_membership", "usr_gading", "dev_android", "mbr_active", "pause_protection", "membership", 60, now.Add(time.Hour), "quick-membership")
	require.NoError(t, err)
	require.NoError(t, repo.CancelPendingApprovalsForMembership(ctx, "mbr_active", "usr_suci"))
	var memberRequest model.ApprovalRequest
	for _, item := range st.Snapshot().Approvals {
		if item.ID == "apr_membership" {
			memberRequest = item
		}
	}
	assert.Equal(t, "cancelled", memberRequest.Status)

	assert.EqualError(t, repo.UpdateApprovalRequest(ctx, "apr_cancel", "approved", "usr_suci"), "pending approval request not found")

	assert.Equal(t, "Allow protected app removal", approvalActionLabel("uninstall_detected", 0))
	assert.Equal(t, "unknown", approvalActionLabel("unknown", 0))
	assert.Equal(t, "custom", approvalStatusLabel("custom"))
	_, err = approvalGrantExpiry("pause_protection", 10, now)
	assert.Error(t, err)
	_, err = approvalGrantExpiry("unsupported", 15, now)
	assert.Error(t, err)
	_, err = applyApprovalGrant(&model.ApprovalRequest{ID: "bad", Status: "pending"}, now, "jti")
	assert.Error(t, err)

	expired := model.ApprovalRequest{ID: "expired", UserID: "usr_gading", DeviceID: "dev_android", Action: "pause_protection", Status: "approved", RequestedDurationMinutes: 15, ResolvedAt: timePtr(now.Add(-31 * time.Minute)), ExpiresAt: now.Add(time.Hour)}
	repo2 := New(nil, &store.Store{Approvals: []model.ApprovalRequest{expired}})
	_, err = repo2.ApplyApprovedRequest(ctx, "expired", "usr_gading", "dev_android", now, "expired-jti")
	require.EqualError(t, err, "approval apply window expired")
}

func TestRefreshTokenRepository_InMemorySessionAndConsumption(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	authTime := time.Now().UTC().Add(-time.Minute)
	deviceID := "dev_android"
	require.NoError(t, repo.CreateRefreshTokenWithAuthTime(ctx, "rt_cov", "usr_gading", "hash-cov", &deviceID, authTime, time.Now().Add(time.Hour)))
	id, userID, gotDevice, gotAuthTime, err := repo.GetActiveRefreshTokenSession(ctx, "hash-cov")
	require.NoError(t, err)
	assert.Equal(t, "rt_cov", id)
	assert.Equal(t, "usr_gading", userID)
	assert.Equal(t, &deviceID, gotDevice)
	assert.Equal(t, authTime, gotAuthTime)

	require.NoError(t, repo.ConsumeRefreshTokenByID(ctx, "rt_cov"))
	assert.EqualError(t, repo.ConsumeRefreshTokenByID(ctx, "rt_cov"), "refresh token already consumed or expired")
	_, _, _, _, err = repo.GetActiveRefreshTokenSession(ctx, "hash-cov")
	assert.Error(t, err)

	require.NoError(t, repo.CreateRefreshToken(ctx, "rt_cov_2", "usr_gading", "hash-cov-2", nil, time.Now().Add(time.Hour)))
	require.NoError(t, repo.CreateRefreshToken(ctx, "rt_cov_3", "usr_gading", "hash-cov-3", nil, time.Now().Add(time.Hour)))
	require.NoError(t, repo.RevokeRefreshTokensForUser(ctx, "usr_gading"))
	_, _, _, err = repo.GetActiveRefreshToken(ctx, "hash-cov-2")
	assert.Error(t, err)
	assert.NoError(t, repo.RevokeRefreshToken(ctx, "missing-hash"))
	assert.NoError(t, repo.RevokeRefreshTokenByID(ctx, "missing-id"))
}

func TestReflectionAndRecoveryRepositories_InMemoryStateAndErrors(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()

	entries, err := repo.GetReflections(ctx, "usr_gading")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "active", entries[0].Status)

	focus, err := repo.CreateReflectionEntry(ctx, model.JournalEntry{UserID: "usr_gading", Text: "focus", Mood: "calm", Status: "active", IsFocus: true})
	require.NoError(t, err)
	second, err := repo.CreateReflectionEntry(ctx, model.JournalEntry{UserID: "usr_gading", Text: "second", Mood: "steady", Status: "active", IsFocus: true})
	require.NoError(t, err)
	assert.NotEqual(t, focus.ID, second.ID)
	updated, err := repo.UpdateReflectionEntry(ctx, model.JournalEntry{ID: second.ID, UserID: "usr_gading", Text: "updated", Status: "archived", IsFocus: false})
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Text)
	assert.Equal(t, "archived", updated.Status)
	_, err = repo.UpdateReflectionEntry(ctx, model.JournalEntry{ID: "missing", UserID: "usr_gading"})
	require.EqualError(t, err, "reflection not found")
	_, err = repo.UpdateReflectionEntry(ctx, model.JournalEntry{ID: second.ID, UserID: "usr_dery"})
	require.EqualError(t, err, "reflection not found")

	intention, ok := repo.GetIntention(ctx, "usr_gading")
	assert.False(t, ok)
	assert.Empty(t, intention.ID)
	quiz := model.Intention{SchoolImpact: "medium", MoneySpent: "low", ScreenTime: "sometimes", QuitAttempts: "one", QuitMotivation: "high"}
	intention, err = repo.SaveIntention(ctx, "usr_gading", "I want to focus", "active", quiz)
	require.NoError(t, err)
	assert.Equal(t, "I want to focus", intention.Text)
	intention, err = repo.SaveIntention(ctx, "usr_gading", "Updated intention", "active", model.Intention{SchoolImpact: "high"})
	require.NoError(t, err)
	assert.Equal(t, "Updated intention", intention.Text)
	assert.Equal(t, "high", intention.SchoolImpact)

	checkIn, err := repo.SaveCheckIn(ctx, "usr_gading", 4, 2, "some context")
	require.NoError(t, err)
	checkIn, err = repo.SaveCheckIn(ctx, "usr_gading", 5, 1, "updated context")
	require.NoError(t, err)
	assert.Equal(t, 5, checkIn.Mood)
	assert.Equal(t, "updated context", checkIn.Context)
	checkIns, err := repo.GetCheckIns(ctx, "usr_gading")
	require.NoError(t, err)
	require.Len(t, checkIns, 1)

	sessions, err := repo.ListRecoveryPracticeSessions(ctx, "usr_gading", time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.NoError(t, repo.DeleteExpiredRecoveryPracticeSessions(ctx, "usr_gading", time.Now()))
	sessions, err = repo.ListRecoveryPracticeSessions(ctx, "usr_gading", time.Time{})
	require.NoError(t, err)
	assert.Empty(t, sessions)

	space, found, err := repo.GetRecoverySpace(ctx, "usr_gading")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, space.ID)
	saved, err := repo.SaveRecoverySpace(ctx, model.RecoverySpace{ID: "space-cov", UserID: "usr_gading", Theme: "calm", UnlockedItems: []string{"a"}, PlacedItems: map[string]any{"a": 1}, UnlockRuleVersion: 1})
	require.NoError(t, err)
	assert.Equal(t, "space-cov", saved.ID)
	saved, err = repo.SaveRecoverySpace(ctx, model.RecoverySpace{ID: "space-new-id", UserID: "usr_gading", Theme: "focus", UnlockedItems: []string{"a", "b"}, UnlockRuleVersion: 2})
	require.NoError(t, err)
	assert.Equal(t, "space-new-id", saved.ID)
	assert.Equal(t, "focus", saved.Theme)
}

func TestSPKRepository_InMemoryPreferenceInterventionsAndBlockedEvents(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	pref, err := repo.SpkPreference(ctx, "usr_gading")
	require.NoError(t, err)
	assert.True(t, pref.SpkRecommendationEnabled)
	pref.SpkRecommendationEnabled = false
	pref.SpkUseProtection = false
	stored, err := repo.UpsertSpkPreference(ctx, "usr_gading", pref)
	require.NoError(t, err)
	assert.False(t, stored.SpkRecommendationEnabled)
	assert.False(t, stored.SpkUseProtection)

	record := model.InterventionRecord{ID: "int-cov", UserID: "usr_gading", InterventionKey: "breathing", ResponseType: "exercise", SupportLevel: "light", EngagementLevel: "steady", Status: "recommended", RecommendedAt: now}
	created, err := repo.UpsertInterventionRecord(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, "int-cov", created.ID)
	list, err := repo.InterventionRecords(ctx, "usr_gading")
	require.NoError(t, err)
	assert.Contains(t, list, created)
	today, err := repo.TodayInterventionRecord(ctx, "usr_gading", now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	require.NotNil(t, today)
	require.NoError(t, repo.UpdateInterventionEffectiveness(ctx, "usr_gading", "int-cov", "helpful"))
	updated, err := repo.UpdateInterventionRecord(ctx, "usr_gading", model.InterventionRecord{ID: "int-cov", UserID: "usr_gading", InterventionKey: "grounding", ResponseType: "education", SupportLevel: "medium", EngagementLevel: "high", RecommendedAt: now, PersonalizedMessage: "message", LLMUsed: true})
	require.NoError(t, err)
	assert.Equal(t, "grounding", updated.InterventionKey)
	completed, err := repo.CompleteIntervention(ctx, "usr_gading", "int-cov")
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.Status)
	_, err = repo.CompleteIntervention(ctx, "usr_dery", "int-cov")
	require.EqualError(t, err, "intervention record not found")
	assert.EqualError(t, repo.UpdateInterventionEffectiveness(ctx, "usr_dery", "int-cov", "not-helpful"), "intervention record not found")

	createdCount, err := repo.SaveBlockedEvents(ctx, "usr_gading", "dev_android", []time.Time{now, now, now.Add(time.Minute)})
	require.NoError(t, err)
	assert.Equal(t, 2, createdCount)
	createdCount, err = repo.SaveBlockedEvents(ctx, "usr_gading", "dev_android", nil)
	require.NoError(t, err)
	assert.Zero(t, createdCount)
	counts, err := repo.BlockCountsByDate(ctx, "usr_gading", now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, counts[now.Format("2006-01-02")], 0)
	_, err = repo.UpdateInterventionRecord(ctx, "usr_dery", model.InterventionRecord{ID: "missing", UserID: "usr_dery"})
	assert.EqualError(t, err, "intervention record not found")
}

func timePtr(value time.Time) *time.Time {
	return &value
}
