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
	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoverypracticesession"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func repositoryFinalDate(day int) time.Time {
	return time.Date(2026, time.September, day, 12, 0, 0, 0, time.UTC)
}

func TestRepositoryFinalRecoveryAnalytics_DashboardAndProgressWindows(t *testing.T) {
	ctx := context.Background()
	now := repositoryFinalDate(20)
	userID := "repository-final-user"

	st := store.New()
	st.Users = []model.User{{ID: userID, DisplayName: "Final Coverage User", ExperiencePoints: 30}}
	st.Devices = []model.Device{
		{ID: "repository-final-device-old", UserID: userID, ProtectionStatus: "inactive", LastSeenAt: now.Add(-72 * time.Hour), ModelVersion: "model-old", RulesetVersion: "rules-old"},
		{ID: "repository-final-device-active", UserID: userID, ProtectionStatus: "active", LastSeenAt: now.Add(-24 * time.Hour), ModelVersion: "model-new", RulesetVersion: "rules-new"},
		{ID: "repository-final-device-other", UserID: "other-final-user", ProtectionStatus: "active", LastSeenAt: now},
	}
	st.CheckIns = []model.CheckIn{
		{ID: "repository-final-checkin-old", UserID: userID, Mood: 1, Urge: 1, CreatedAt: now.Add(-10 * 24 * time.Hour)},
		{ID: "repository-final-checkin-late", UserID: userID, Mood: 5, Urge: 4, CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "repository-final-checkin-mid", UserID: userID, Mood: 3, Urge: 2, CreatedAt: now.Add(-24 * time.Hour)},
		{ID: "repository-final-checkin-now", UserID: userID, Mood: 4, Urge: 1, CreatedAt: now},
		{ID: "repository-final-checkin-other", UserID: "other-final-user", Mood: 5, CreatedAt: now},
	}
	st.JournalEntries = []model.JournalEntry{
		{ID: "repository-final-journal-old", UserID: userID, Text: "old", CreatedAt: now.Add(-10 * 24 * time.Hour)},
		{ID: "repository-final-journal-current", UserID: userID, Text: "current", CreatedAt: now.Add(-48 * time.Hour)},
		{ID: "repository-final-journal-other", UserID: "other-final-user", Text: "other", CreatedAt: now},
	}
	st.Missions = []model.DailyMission{
		{ID: "repository-final-mission-complete", UserID: userID, Date: now.Format("2006-01-02"), Mission1: true},
		{ID: "repository-final-mission-empty", UserID: userID, Date: now.Add(-24 * time.Hour).Format("2006-01-02")},
		{ID: "repository-final-mission-old", UserID: userID, Date: now.Add(-10 * 24 * time.Hour).Format("2006-01-02"), Mission1: true},
	}
	st.AggregateEvents = []model.AggregateEvent{
		{ID: "repository-final-aggregate-old", UserID: userID, EventType: "block_count_sync", EventDate: now.Add(-10 * 24 * time.Hour), Count: 100},
		{ID: "repository-final-aggregate-window", UserID: userID, EventType: "block_count_sync", EventDate: now.Add(-6 * 24 * time.Hour), Count: 2},
		{ID: "repository-final-aggregate-recent", UserID: userID, EventType: "block_count_sync", EventDate: now.Add(-24 * time.Hour), Count: 3},
		{ID: "repository-final-aggregate-now", UserID: userID, EventType: "block_count_sync", EventDate: now, Count: 1},
		{ID: "repository-final-aggregate-ignored", UserID: userID, EventType: "raw_url", EventDate: now, Count: 999},
		{ID: "repository-final-aggregate-other", UserID: "other-final-user", EventType: "block_count_sync", EventDate: now, Count: 999},
	}
	st.LearningProgress = []model.LearningProgress{
		{UserID: userID, ItemID: "repository-final-learning-created", CreatedAt: now.Add(-72 * time.Hour)},
		{UserID: userID, ItemID: "repository-final-learning-zero"},
		{UserID: userID, ItemID: "repository-final-learning-old", UpdatedAt: now.Add(-10 * 24 * time.Hour)},
		{UserID: "other-final-user", ItemID: "repository-final-learning-other", UpdatedAt: now},
	}
	st.EducationProgress = []model.EducationProgress{
		{ID: "repository-final-education-current", UserID: userID, UpdatedAt: now.Add(-24 * time.Hour)},
		{ID: "repository-final-education-old", UserID: userID, UpdatedAt: now.Add(-10 * 24 * time.Hour)},
	}
	st.RecoveryRecords = []model.RecoveryRecord{
		{ID: "repository-final-review", UserID: userID, Kind: "weekly_review", RecordDate: now.Add(-24 * time.Hour).Format("2006-01-02")},
		{ID: "repository-final-other-record", UserID: userID, Kind: "checkin", RecordDate: now.Format("2006-01-02")},
	}

	repo := New(nil, st)
	summary, protection, dashboardProgress, err := repo.GetDashboardData(ctx, userID, now)
	require.NoError(t, err)
	assert.Equal(t, "Final Coverage User", summary.UserName)
	assert.Equal(t, "active", summary.ProtectionLabel)
	assert.Equal(t, "local_only", summary.DataState)
	assert.Equal(t, 2, protection.DeviceCount)
	assert.Equal(t, "model-new", protection.ModelVersion)
	assert.Equal(t, "rules-new", protection.RulesetVersion)
	assert.Equal(t, 6, summary.BlockedAttempts)
	assert.Equal(t, 2, dashboardProgress.Reflections)
	assert.Len(t, dashboardProgress.WeeklyBlocks, 7)
	assert.Len(t, dashboardProgress.MoodPoints, 3)
	assert.LessOrEqual(t, dashboardProgress.MoodPoints[0].Date, dashboardProgress.MoodPoints[1].Date)

	progress, err := repo.GetProgressData(ctx, userID, 30, now)
	require.NoError(t, err)
	assert.Equal(t, 30, progress.RangeDays)
	assert.Len(t, progress.DailyBlocks, 30)
	assert.Len(t, progress.WeeklyBlocks, 7)
	assert.True(t, progress.TrendAvailable)
	assert.Equal(t, 4, progress.CheckInCount)
	assert.Equal(t, 2, progress.Reflections)
	assert.NotEmpty(t, progress.ActivityDays)
	var hasLearningFallback, hasEducation, hasReview bool
	for _, item := range progress.ActivityDays {
		if item.LearningHub > 0 {
			hasLearningFallback = true
		}
		if item.Education > 0 {
			hasEducation = true
		}
		if item.Reviews > 0 {
			hasReview = true
		}
	}
	assert.True(t, hasLearningFallback)
	assert.True(t, hasEducation)
	assert.True(t, hasReview)

	_, _, _, err = repo.GetDashboardData(ctx, "repository-final-missing", now)
	require.NoError(t, err)
	_, err = repo.GetProgressData(ctx, userID, 14, now)
	require.EqualError(t, err, "progress range must be 7, 30, or 90 days")
}

func TestRepositoryFinalRecoveryAnalytics_AggregateAndAnalyticsPrivacy(t *testing.T) {
	ctx := context.Background()
	now := repositoryFinalDate(20)
	st := store.New()
	st.Devices = []model.Device{{ID: "repository-final-analytics-device", UserID: "repository-final-analytics-user", ProtectionStatus: "active"}}
	st.AggregateEvents = []model.AggregateEvent{
		{ID: "repository-final-protection-block", UserID: "repository-final-analytics-user", DeviceID: "repository-final-analytics-device", IdempotencyKey: "repository-final-event", EventType: "block_count_sync", EventDate: now, Count: 4, MetadataJSON: map[string]any{"hourly": []any{float64(2)}}},
		{ID: "repository-final-protection-intervention", UserID: "repository-final-analytics-user", DeviceID: "repository-final-analytics-device", EventType: "intervention_shown", EventDate: now.Add(-24 * time.Hour), Count: 2},
		{ID: "repository-final-protection-tamper", UserID: "repository-final-analytics-user", DeviceID: "repository-final-analytics-device", EventType: "tamper_detected", EventDate: now.Add(-48 * time.Hour), Count: 3},
		{ID: "repository-final-protection-permission", UserID: "repository-final-analytics-user", DeviceID: "repository-final-analytics-device", EventType: "permission_revoked", EventDate: now, Count: 1},
		{ID: "repository-final-protection-old", UserID: "repository-final-analytics-user", DeviceID: "repository-final-analytics-device", EventType: "block_count_sync", EventDate: now.Add(-4 * 24 * time.Hour), Count: 100},
		{ID: "repository-final-protection-raw", UserID: "repository-final-analytics-user", DeviceID: "repository-final-analytics-device", EventType: "raw_url", EventDate: now, Count: 999},
	}
	repo := New(nil, st)
	analytics, err := repo.GetProtectionAnalytics(ctx, "repository-final-analytics-user", "repository-final-analytics-device", 3, now)
	require.NoError(t, err)
	assert.Equal(t, "local_only", analytics.DataState)
	assert.Equal(t, model.ProtectionAnalyticsTotals{Blocked: 4, Interventions: 2, TamperEvents: 3, PermissionRevoked: 1}, analytics.Totals)
	_, err = repo.GetProtectionAnalytics(ctx, "repository-final-analytics-user", "repository-final-unknown-device", 3, now)
	require.EqualError(t, err, "device does not belong to user")

	first, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{ID: "repository-final-new-event", UserID: "repository-final-analytics-user", IdempotencyKey: "repository-final-new-key", EventType: "block_count_sync", EventDate: now, Count: 8})
	require.NoError(t, err)
	duplicate, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{ID: "repository-final-duplicate", UserID: "repository-final-analytics-user", IdempotencyKey: "repository-final-new-key", Count: 99})
	require.NoError(t, err)
	assert.Equal(t, first.ID, duplicate.ID)
	_, err = repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{ID: "repository-final-snapshot", UserID: "repository-final-analytics-user", IdempotencyKey: "repository-final-snapshot-key", Count: 10})
	require.NoError(t, err)
	stale, err := repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{UserID: "repository-final-analytics-user", IdempotencyKey: "repository-final-snapshot-key", Count: 1})
	require.NoError(t, err)
	assert.Equal(t, 10, stale.Count)
	higher, err := repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{UserID: "repository-final-analytics-user", IdempotencyKey: "repository-final-snapshot-key", Count: 11})
	require.NoError(t, err)
	assert.Equal(t, 11, higher.Count)
	assert.Empty(t, mustRepositoryFinalAggregateEventsForUsers(t, repo, ctx, nil, 3, now))

	st.AccountabilityGroups = []model.AccountabilityGroup{{ID: "repository-final-group", OwnerPartnerID: "repository-final-partner", Status: "active"}}
	st.AccountabilityMemberships = []model.AccountabilityMembership{
		{ID: "repository-final-live-private", GroupID: "repository-final-group", StudentID: "repository-final-private", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: false}},
		{ID: "repository-final-live-shared", GroupID: "repository-final-group", StudentID: "repository-final-shared", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: false}},
		{ID: "repository-final-left-shared", GroupID: "repository-final-group", StudentID: "repository-final-left", Status: "left", Sharing: model.SharingPreferences{ProtectionActivity: true}},
	}
	st.AggregateEvents = append(st.AggregateEvents, model.AggregateEvent{ID: "repository-final-shared-event", UserID: "repository-final-shared", EventType: "block_count_sync", EventDate: now, Count: 6})
	empty, err := repo.PartnerAnalytics(ctx, "repository-final-partner", "repository-final-group", 3, now)
	require.NoError(t, err)
	assert.Equal(t, 2, empty.MemberCount)
	assert.Equal(t, 0, empty.SharedMemberCount)
	assert.Equal(t, "empty", empty.DataState)

	st.AccountabilityMemberships[0].Sharing.ProtectionActivity = true
	st.AccountabilityMemberships[1].Sharing.ProtectionActivity = true
	partner, err := repo.PartnerAnalytics(ctx, "repository-final-partner", "repository-final-group", 3, now)
	require.NoError(t, err)
	assert.Equal(t, 2, partner.SharedMemberCount)
	assert.Equal(t, 6, partner.Totals.Blocked)
	platform, err := repo.PlatformAnalytics(ctx, 3, now)
	require.NoError(t, err)
	assert.Equal(t, 1, platform.ProtectedUsers)
	assert.Equal(t, "local_only", platform.DataState)
	_, err = repo.PartnerAnalytics(ctx, "repository-final-partner", "repository-final-missing-group", 3, now)
	require.EqualError(t, err, "group is not owned by the partner")
}

func mustRepositoryFinalAggregateEventsForUsers(t *testing.T, repo *Repository, ctx context.Context, userIDs []string, days int, now time.Time) []model.AggregateEvent {
	t.Helper()
	events, err := repo.aggregateEventsForUsers(ctx, userIDs, days, now)
	require.NoError(t, err)
	return events
}

func TestRepositoryFinalRecoveryAnalytics_RecoveryRetentionAndReflection(t *testing.T) {
	ctx := context.Background()
	now := repositoryFinalDate(20)
	repo := New(nil, store.New())

	intention, found := repo.GetIntention(ctx, "repository-final-recovery-user")
	assert.False(t, found)
	assert.Empty(t, intention.ID)
	fullQuiz := model.Intention{SchoolImpact: "high", MoneySpent: "medium", ScreenTime: "often", QuitAttempts: "two", QuitMotivation: "high"}
	createdIntention, err := repo.SaveIntention(ctx, "repository-final-recovery-user", "first intention", "active", fullQuiz)
	require.NoError(t, err)
	assert.Equal(t, "first intention", createdIntention.Text)
	updatedIntention, err := repo.SaveIntention(ctx, "repository-final-recovery-user", "updated intention", "active", model.Intention{})
	require.NoError(t, err)
	assert.Empty(t, updatedIntention.SchoolImpact)
	assert.Empty(t, updatedIntention.QuitMotivation)
	gotIntention, found := repo.GetIntention(ctx, "repository-final-recovery-user")
	require.True(t, found)
	assert.Equal(t, "updated intention", gotIntention.Text)

	checkIns, err := repo.GetCheckIns(ctx, "repository-final-recovery-user")
	require.NoError(t, err)
	assert.Empty(t, checkIns)
	checkIn, err := repo.SaveCheckIn(ctx, "repository-final-recovery-user", 3, 2, "first context")
	require.NoError(t, err)
	updatedCheckIn, err := repo.SaveCheckIn(ctx, "repository-final-recovery-user", 5, 1, "updated context")
	require.NoError(t, err)
	assert.Equal(t, checkIn.ID, updatedCheckIn.ID)
	assert.Equal(t, 5, updatedCheckIn.Mood)

	oldRecord := model.RecoveryRecord{ID: "repository-final-recovery-old", UserID: "repository-final-recovery-user", Kind: "weekly_review", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)}
	newRecord := model.RecoveryRecord{ID: "repository-final-recovery-new", UserID: "repository-final-recovery-user", Kind: "weekly_review", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	otherRecord := model.RecoveryRecord{ID: "repository-final-recovery-other", UserID: "other-final-user", Kind: "weekly_review", CreatedAt: now.Add(-48 * time.Hour)}
	for _, record := range []model.RecoveryRecord{oldRecord, newRecord, otherRecord} {
		_, err = repo.SaveRecoveryRecord(ctx, record)
		require.NoError(t, err)
	}
	records, err := repo.ListRecoveryRecords(ctx, "repository-final-recovery-user", now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, newRecord.ID, records[0].ID)
	require.NoError(t, repo.DeleteExpiredRecoveryRecords(ctx, "repository-final-recovery-user", now.Add(-24*time.Hour)))
	records, err = repo.ListRecoveryRecords(ctx, "repository-final-recovery-user", time.Time{})
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, newRecord.ID, records[0].ID)

	practiceOld := model.RecoveryPracticeSession{ID: "repository-final-practice-old", UserID: "repository-final-recovery-user", PracticeKind: "breathing", CompletedAt: now.Add(-48 * time.Hour), CreatedAt: now.Add(-48 * time.Hour)}
	practiceNew := model.RecoveryPracticeSession{ID: "repository-final-practice-new", UserID: "repository-final-recovery-user", PracticeKind: "grounding", Feedback: "lighter", CompletedAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour)}
	repoStore := store.New()
	repoStore.Users = []model.User{{ID: "repository-final-recovery-user", ExperiencePoints: 40}}
	repoStore.CheckIns = []model.CheckIn{checkIn}
	repoStore.RecoveryRecords = []model.RecoveryRecord{newRecord}
	repoStore.RecoveryPracticeSessions = []model.RecoveryPracticeSession{practiceOld, practiceNew}
	repoStore.JournalEntries = []model.JournalEntry{{ID: "repository-final-focus", UserID: "repository-final-recovery-user", IsFocus: true, CreatedAt: now}}
	repoStore.Missions = []model.DailyMission{{ID: "repository-final-evidence-mission", UserID: "repository-final-recovery-user", Date: now.Format("2006-01-02"), Mission1: true, Mission2: true}}
	evidenceRepo := New(nil, repoStore)
	practices, err := evidenceRepo.ListRecoveryPracticeSessions(ctx, "repository-final-recovery-user", now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Len(t, practices, 1)
	assert.Equal(t, "lighter", practices[0].Feedback)
	require.NoError(t, evidenceRepo.DeleteExpiredRecoveryPracticeSessions(ctx, "repository-final-recovery-user", now.Add(-24*time.Hour)))
	practices, err = evidenceRepo.ListRecoveryPracticeSessions(ctx, "repository-final-recovery-user", time.Time{})
	require.NoError(t, err)
	require.Len(t, practices, 1)

	space, found, err := evidenceRepo.GetRecoverySpace(ctx, "repository-final-recovery-user")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, space.ID)
	savedSpace, err := evidenceRepo.SaveRecoverySpace(ctx, model.RecoverySpace{ID: "repository-final-space", UserID: "repository-final-recovery-user", Theme: "calm", UnlockedItems: []string{"one"}, PlacedItems: map[string]any{"one": "center"}, UnlockRuleVersion: 1, CreatedAt: now})
	require.NoError(t, err)
	savedSpace.Theme = "focused"
	updatedSpace, err := evidenceRepo.SaveRecoverySpace(ctx, savedSpace)
	require.NoError(t, err)
	assert.Equal(t, "focused", updatedSpace.Theme)
	assert.Equal(t, savedSpace.CreatedAt, updatedSpace.CreatedAt)

	evidence := evidenceRepo.RecoveryUnlockEvidence(ctx, "repository-final-recovery-user")
	assert.True(t, evidence.HasFocusJournal)
	assert.True(t, evidence.HasWeeklyReview)
	assert.Equal(t, 1, evidence.TotalPractices)
	assert.Equal(t, 2, evidence.MissionsClaimed)
	assert.Equal(t, 40, evidence.ExperiencePoints)
	assert.GreaterOrEqual(t, evidence.ActiveDays, 3)

	emptyReflectionRepo := New(nil, store.New())
	emptyReflections, err := emptyReflectionRepo.GetReflections(ctx, "repository-final-recovery-user")
	require.NoError(t, err)
	assert.Empty(t, emptyReflections)
	firstReflection, err := evidenceRepo.CreateReflectionEntry(ctx, model.JournalEntry{UserID: "repository-final-recovery-user", Text: "first", Status: "active"})
	require.NoError(t, err)
	focusReflection, err := evidenceRepo.CreateReflectionEntry(ctx, model.JournalEntry{UserID: "repository-final-recovery-user", Text: "focus", Status: "active", IsFocus: true})
	require.NoError(t, err)
	updatedReflection, err := evidenceRepo.UpdateReflectionEntry(ctx, model.JournalEntry{ID: firstReflection.ID, UserID: "repository-final-recovery-user", Text: "updated", Status: "archived", IsFocus: true})
	require.NoError(t, err)
	assert.Equal(t, "updated", updatedReflection.Text)
	assert.True(t, updatedReflection.IsFocus)
	_, err = evidenceRepo.UpdateReflectionEntry(ctx, model.JournalEntry{ID: "repository-final-missing-reflection", UserID: "repository-final-recovery-user"})
	require.EqualError(t, err, "reflection not found")
	reflections, err := evidenceRepo.GetReflections(ctx, "repository-final-recovery-user")
	require.NoError(t, err)
	assert.Len(t, reflections, 3)
	assert.True(t, focusReflection.IsFocus)
}

func TestRepositoryFinalRecoveryAnalytics_ReminderPushAndEmptyStates(t *testing.T) {
	ctx := context.Background()
	now := repositoryFinalDate(20)
	repo := New(nil, store.New())

	preference, err := repo.ReminderPreference(ctx, "repository-final-reminder-user")
	require.NoError(t, err)
	assert.False(t, preference.Enabled)
	assert.Equal(t, defaultReminderTimezone, preference.Timezone)
	preference, err = repo.UpsertReminderPreference(ctx, "repository-final-reminder-user", true, "08:30", "Asia/Jakarta", "id")
	require.NoError(t, err)
	assert.True(t, preference.Enabled)
	require.NoError(t, repo.MarkReminderFired(ctx, "repository-final-reminder-user", now))
	preference, err = repo.UpsertReminderPreference(ctx, "repository-final-reminder-user", false, "20:00", "UTC", "en")
	require.NoError(t, err)
	assert.False(t, preference.Enabled)
	assert.NotNil(t, preference.LastFiredAt)
	require.NoError(t, repo.MarkReminderFired(ctx, "repository-final-missing-reminder", now))
	assert.Empty(t, mustRepositoryFinalEnabledReminders(t, repo, ctx))
	preference, err = repo.UpsertReminderPreference(ctx, "repository-final-reminder-user", true, "20:00", "UTC", "en")
	require.NoError(t, err)
	assert.Len(t, mustRepositoryFinalEnabledReminders(t, repo, ctx), 1)

	userAgent := "repository-final-agent"
	subscription, err := repo.UpsertPushSubscription(ctx, "repository-final-push-user", "https://push.final/endpoint", "p256dh-one", "auth-one", &userAgent)
	require.NoError(t, err)
	assert.Equal(t, "repository-final-push-user", subscription.UserID)
	updated, err := repo.UpsertPushSubscription(ctx, "repository-final-other-user", subscription.Endpoint, "p256dh-two", "auth-two", nil)
	require.NoError(t, err)
	assert.Equal(t, subscription.ID, updated.ID)
	assert.Nil(t, updated.UserAgent)
	assert.Len(t, mustRepositoryFinalPushSubscriptions(t, repo, ctx, "repository-final-other-user"), 1)
	assert.Empty(t, mustRepositoryFinalPushSubscriptions(t, repo, ctx, "repository-final-push-user"))
	require.NoError(t, repo.DeletePushSubscription(ctx, "repository-final-push-user", subscription.Endpoint))
	assert.Len(t, mustRepositoryFinalPushSubscriptions(t, repo, ctx, "repository-final-other-user"), 1)
	require.NoError(t, repo.RemovePushSubscriptionByID(ctx, subscription.ID))
	require.NoError(t, repo.RemovePushSubscriptionByID(ctx, "repository-final-missing-subscription"))
	assert.Empty(t, mustRepositoryFinalPushSubscriptions(t, repo, ctx, "repository-final-other-user"))
}

func mustRepositoryFinalEnabledReminders(t *testing.T, repo *Repository, ctx context.Context) []store.ReminderPreference {
	t.Helper()
	items, err := repo.EnabledReminderPreferences(ctx)
	require.NoError(t, err)
	return items
}

func mustRepositoryFinalPushSubscriptions(t *testing.T, repo *Repository, ctx context.Context, userID string) []model.PushSubscription {
	t.Helper()
	items, err := repo.PushSubscriptionsForUser(ctx, userID)
	require.NoError(t, err)
	return items
}

func TestRepositoryFinalRecoveryAnalytics_ExportPrivacyAndDeletion(t *testing.T) {
	ctx := context.Background()
	now := repositoryFinalDate(20)
	userID := "repository-final-delete-user"
	otherID := "repository-final-survivor"
	st := store.New()
	st.Users = []model.User{
		{ID: userID, Email: "delete-final@example.test", DisplayName: "Delete Final"},
		{ID: otherID, Email: "survivor-final@example.test", DisplayName: "Survivor Final"},
	}
	st.Devices = []model.Device{{ID: "repository-final-delete-device", UserID: userID}, {ID: "repository-final-survivor-device", UserID: otherID}}
	st.AggregateEvents = []model.AggregateEvent{{ID: "repository-final-delete-aggregate", UserID: userID}}
	st.Intentions = []model.Intention{{ID: "repository-final-delete-intention", UserID: userID}}
	st.CheckIns = []model.CheckIn{{ID: "repository-final-delete-checkin", UserID: userID}}
	st.RecoveryRecords = []model.RecoveryRecord{{ID: "repository-final-delete-record", UserID: userID}}
	st.RecoveryPracticeSessions = []model.RecoveryPracticeSession{{ID: "repository-final-delete-practice", UserID: userID}}
	st.RecoverySpaces = []model.RecoverySpace{{ID: "repository-final-delete-space", UserID: userID}}
	st.JournalEntries = []model.JournalEntry{{ID: "repository-final-delete-journal", UserID: userID}}
	st.Missions = []model.DailyMission{{ID: "repository-final-delete-mission", UserID: userID}}
	st.EducationProgress = []model.EducationProgress{{ID: "repository-final-delete-education", UserID: userID}}
	st.AccountabilityGroups = []model.AccountabilityGroup{{ID: "repository-final-delete-group", OwnerPartnerID: userID}, {ID: "repository-final-survivor-group", OwnerPartnerID: otherID}}
	st.AccountabilityMemberships = []model.AccountabilityMembership{
		{ID: "repository-final-delete-membership", GroupID: "repository-final-delete-group", StudentID: userID},
		{ID: "repository-final-survivor-membership", GroupID: "repository-final-survivor-group", StudentID: otherID},
	}
	st.MembershipExitRequests = []model.MembershipExitRequest{{ID: "repository-final-delete-exit", MembershipID: "repository-final-delete-membership", RequestedBy: userID}, {ID: "repository-final-survivor-exit", MembershipID: "repository-final-survivor-membership", RequestedBy: otherID}}
	st.PartnerContactRequests = []model.PartnerContactRequest{{ID: "repository-final-delete-contact", MembershipID: "repository-final-delete-membership", StudentID: userID}, {ID: "repository-final-survivor-contact", MembershipID: "repository-final-survivor-membership", StudentID: otherID}}
	st.Approvals = []model.ApprovalRequest{{ID: "repository-final-delete-approval", UserID: userID, MembershipID: "repository-final-delete-membership"}, {ID: "repository-final-survivor-approval", UserID: otherID, MembershipID: "repository-final-survivor-membership"}}
	st.SupportCases = []model.SupportCase{{ID: "repository-final-delete-case", UserID: userID}, {ID: "repository-final-survivor-case", UserID: otherID}}
	st.SupportMessages = []model.SupportMessage{{ID: "repository-final-delete-message", SupportCaseID: "repository-final-delete-case"}, {ID: "repository-final-survivor-message", SupportCaseID: "repository-final-survivor-case"}}
	st.Partners = []model.Partner{{ID: "repository-final-delete-partner", UserID: userID, PartnerUserID: otherID}, {ID: "repository-final-survivor-partner", UserID: otherID}}
	st.ContactVerifications = []model.ContactVerification{{ID: "repository-final-delete-verification", UserID: userID}}
	st.DataRequests = []model.DataRequest{
		{ID: "repository-final-delete-request", UserID: userID, Type: "delete", Status: "processing", ConfirmationTokenHash: "secret", ResultPath: "exports/delete.zip"},
		{ID: "repository-final-export-request", UserID: userID, Type: "export", Status: "completed", ConfirmationTokenHash: "secret-export", ResultPath: "exports/export.zip"},
		{ID: "repository-final-survivor-request", UserID: otherID, Type: "delete", Status: "processing"},
	}
	st.AuditEvents = []model.AuditEvent{{ID: "repository-final-delete-audit", ActorID: userID, Actor: "delete-final@example.test"}, {ID: "repository-final-survivor-audit", ActorID: otherID, Actor: "survivor-final@example.test"}}

	repo := New(nil, st)
	export, err := repo.BuildUserExportSnapshot(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, export["account"].(model.User).ID)
	for _, key := range []string{"devices", "aggregate_protection_events", "intentions", "check_ins", "recovery_records", "recovery_practice_sessions", "recovery_spaces", "reflections", "missions", "education_progress", "accountability_memberships", "owned_accountability_groups", "approval_requests", "support_cases", "support_messages"} {
		assert.Len(t, export[key], 1, key)
	}
	_, err = repo.BuildUserExportSnapshot(ctx, "repository-final-unknown-user")
	require.EqualError(t, err, "user not found")

	require.NoError(t, repo.DeleteUserAccountData(ctx, userID, now))
	_, found := repo.UserByID(ctx, userID)
	assert.False(t, found)
	assert.Len(t, st.Snapshot().Users, 1)
	assert.Len(t, st.Snapshot().Devices, 1)
	assert.Len(t, st.Snapshot().AccountabilityGroups, 1)
	assert.Len(t, st.Snapshot().SupportCases, 1)
	snapshot := st.Snapshot()
	for _, request := range snapshot.DataRequests {
		if request.ID == "repository-final-delete-request" || request.ID == "repository-final-export-request" {
			assert.Contains(t, request.UserID, "deleted:")
			assert.Empty(t, request.ConfirmationTokenHash)
			assert.Empty(t, request.ResultPath)
		}
	}
	for _, audit := range snapshot.AuditEvents {
		if audit.ID == "repository-final-delete-audit" {
			assert.Contains(t, audit.ActorID, "deleted:")
			assert.Equal(t, "deleted-account", audit.Actor)
		}
	}
	assert.Equal(t, "completed", snapshot.DataRequests[0].Status)
	assert.Equal(t, otherID, snapshot.DataRequests[len(snapshot.DataRequests)-1].UserID)
	assert.EqualError(t, repo.DeleteUserAccountData(ctx, userID, now), "user not found")
}

func repositoryFinalRecoveryOpenSQLite(t *testing.T) *ent.Client {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	database, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", databaseName))
	require.NoError(t, err)
	database.SetMaxOpenConns(1)
	driver := entsql.OpenDB("sqlite3", database)
	client := ent.NewClient(ent.Driver(driver))
	require.NoError(t, client.Schema.Create(context.Background()))
	t.Cleanup(func() {
		_ = client.Close()
		_ = database.Close()
	})
	return client
}

func TestRepositoryFinalRecoveryAnalytics_SQLitePersistenceMappersAndRetention(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	client := repositoryFinalRecoveryOpenSQLite(t)
	repo := New(client, store.New())
	user, err := repo.CreateUser(ctx, "repository-final-sqlite-user", "repository-final-sqlite@example.test", "SQLite Final")
	require.NoError(t, err)
	device, err := repo.CreateDevice(ctx, "repository-final-sqlite-device", user.ID, "repository-final-sqlite-instance", "android", "SQLite device", "1.0", "Android", nil, nil)
	require.NoError(t, err)
	_, err = repo.UpdateOwnedDevice(ctx, user.ID, device.ID, "SQLite active device", "1.0", "Android", "active", "model-sqlite", "rules-sqlite")
	require.NoError(t, err)

	event, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{ID: "repository-final-sqlite-event", UserID: user.ID, DeviceID: device.ID, IdempotencyKey: "repository-final-sqlite-key", EventType: "block_count_sync", EventDate: now, Count: 4, MetadataJSON: map[string]any{"hourly": []any{float64(4)}}})
	require.NoError(t, err)
	assert.Equal(t, "repository-final-sqlite-event", event.ID)
	duplicate, err := repo.SaveAggregateEvent(ctx, model.AggregateEvent{ID: "repository-final-sqlite-duplicate", UserID: user.ID, IdempotencyKey: "repository-final-sqlite-key", Count: 99})
	require.NoError(t, err)
	assert.Equal(t, event.ID, duplicate.ID)
	snapshot, err := repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{ID: "repository-final-sqlite-snapshot", UserID: user.ID, DeviceID: device.ID, IdempotencyKey: "repository-final-sqlite-snapshot-key", EventType: "block_count_sync", EventDate: now, Count: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, snapshot.Count)
	snapshot, err = repo.SaveAggregateEventSnapshot(ctx, model.AggregateEvent{UserID: user.ID, IdempotencyKey: "repository-final-sqlite-snapshot-key", Count: 5, MetadataJSON: map[string]any{"updated": true}})
	require.NoError(t, err)
	assert.Equal(t, 5, snapshot.Count)
	assert.True(t, snapshot.MetadataJSON["updated"].(bool))
	databaseEvents, err := repo.aggregateEventsForUsers(ctx, []string{user.ID}, 3, now)
	require.NoError(t, err)
	assert.Len(t, databaseEvents, 2)
	protection, err := repo.GetProtectionAnalytics(ctx, user.ID, device.ID, 3, now)
	require.NoError(t, err)
	assert.Equal(t, "synced", protection.DataState)
	assert.Equal(t, 9, protection.Totals.Blocked)

	quiz := model.Intention{SchoolImpact: "happened", MoneySpent: "under_500k", ScreenTime: "under_1h", QuitAttempts: "once", QuitMotivation: "determined"}
	intention, err := repo.SaveIntention(ctx, user.ID, "SQLite intention", "active", quiz)
	require.NoError(t, err)
	assert.Equal(t, "happened", intention.SchoolImpact)
	intention, err = repo.SaveIntention(ctx, user.ID, "SQLite fully updated intention", "active", quiz)
	require.NoError(t, err)
	assert.Equal(t, "under_500k", intention.MoneySpent)
	intention, err = repo.SaveIntention(ctx, user.ID, "SQLite updated intention", "active", model.Intention{})
	require.NoError(t, err)
	assert.Empty(t, intention.MoneySpent)
	loadedIntention, found := repo.GetIntention(ctx, user.ID)
	require.True(t, found)
	assert.Equal(t, "SQLite updated intention", loadedIntention.Text)

	checkIn, err := repo.SaveCheckIn(ctx, user.ID, 3, 2, "SQLite context")
	require.NoError(t, err)
	checkIn, err = repo.SaveCheckIn(ctx, user.ID, 4, 1, "SQLite updated context")
	require.NoError(t, err)
	assert.Equal(t, 4, checkIn.Mood)
	checkIns, err := repo.GetCheckIns(ctx, user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, checkIns)
	var foundUpdatedCheckIn bool
	for _, item := range checkIns {
		if item.Mood == 4 && item.Context == "SQLite updated context" {
			foundUpdatedCheckIn = true
		}
	}
	assert.True(t, foundUpdatedCheckIn)

	reflectionEntry, err := repo.CreateReflectionEntry(ctx, model.JournalEntry{UserID: user.ID, Text: "SQLite encrypted payload", Mood: "calm", Status: "active", IsFocus: true})
	require.NoError(t, err)
	reflections, err := repo.GetReflections(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, reflections, 1)
	assert.Equal(t, reflectionEntry.Text, reflections[0].Text)
	reflectionEntry.Text = "SQLite updated payload"
	updatedReflection, err := repo.UpdateReflectionEntry(ctx, reflectionEntry)
	require.NoError(t, err)
	assert.Equal(t, reflectionEntry.Text, updatedReflection.Text)

	oldPracticeAt := now.Add(-48 * time.Hour)
	newPracticeAt := now.Add(-time.Hour)
	_, err = client.RecoveryPracticeSession.Create().SetID("repository-final-sqlite-practice-old").SetUserID(user.ID).
		SetPracticeKind(recoverypracticesession.PracticeKindUrgeSurfing).SetDurationSeconds(60).
		SetCompletedAt(oldPracticeAt).SetCreatedAt(oldPracticeAt).Save(ctx)
	require.NoError(t, err)
	feedback := recoverypracticesession.FeedbackLighter
	_, err = client.RecoveryPracticeSession.Create().SetID("repository-final-sqlite-practice-new").SetUserID(user.ID).
		SetPracticeKind(recoverypracticesession.PracticeKindGrounding54321).SetDurationSeconds(120).
		SetNillableFeedback(&feedback).SetCompletedAt(newPracticeAt).SetCreatedAt(newPracticeAt).Save(ctx)
	require.NoError(t, err)
	practices, err := repo.ListRecoveryPracticeSessions(ctx, user.ID, now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Len(t, practices, 1)
	assert.Equal(t, "lighter", practices[0].Feedback)
	allPractices, err := repo.ListRecoveryPracticeSessions(ctx, user.ID, time.Time{})
	require.NoError(t, err)
	require.Len(t, allPractices, 2)
	assert.Empty(t, allPractices[1].Feedback)
	require.NoError(t, repo.DeleteExpiredRecoveryPracticeSessions(ctx, user.ID, now.Add(-24*time.Hour)))
	practices, err = repo.ListRecoveryPracticeSessions(ctx, user.ID, time.Time{})
	require.NoError(t, err)
	require.Len(t, practices, 1)

	savedSpace, err := repo.SaveRecoverySpace(ctx, model.RecoverySpace{ID: "repository-final-sqlite-space", UserID: user.ID, Theme: "dorm_room", UnlockedItems: []string{"one"}, PlacedItems: map[string]any{"one": "desk"}, UnlockRuleVersion: 1})
	require.NoError(t, err)
	savedSpace.Theme = "sunrise_study"
	savedSpace, err = repo.SaveRecoverySpace(ctx, savedSpace)
	require.NoError(t, err)
	assert.Equal(t, "sunrise_study", savedSpace.Theme)
	loadedSpace, found, err := repo.GetRecoverySpace(ctx, user.ID)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, savedSpace.ID, loadedSpace.ID)
	canceledContext, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = repo.GetRecoverySpace(canceledContext, user.ID)
	assert.ErrorIs(t, err, context.Canceled)

	reminder, err := repo.UpsertReminderPreference(ctx, user.ID, true, "08:00", "Asia/Jakarta", "id")
	require.NoError(t, err)
	assert.True(t, reminder.Enabled)
	require.NoError(t, repo.MarkReminderFired(ctx, user.ID, now))
	reminder, err = repo.UpsertReminderPreference(ctx, user.ID, false, "20:00", "UTC", "en")
	require.NoError(t, err)
	assert.False(t, reminder.Enabled)
	assert.NotNil(t, reminder.LastFiredAt)
	reminder, err = repo.UpsertReminderPreference(ctx, user.ID, true, "20:00", "UTC", "en")
	require.NoError(t, err)
	assert.Len(t, mustRepositoryFinalEnabledReminders(t, repo, ctx), 1)
	canceledContext, cancel = context.WithCancel(ctx)
	cancel()
	_, err = repo.ReminderPreference(canceledContext, user.ID)
	assert.ErrorIs(t, err, context.Canceled)

	userAgent := "repository-final-sqlite-agent"
	subscription, err := repo.UpsertPushSubscription(ctx, user.ID, "https://push.final/sqlite", "p256dh", "auth", &userAgent)
	require.NoError(t, err)
	assert.Equal(t, user.ID, subscription.UserID)
	subscription, err = repo.UpsertPushSubscription(ctx, user.ID, subscription.Endpoint, "p256dh-updated", "auth-updated", nil)
	require.NoError(t, err)
	require.NotNil(t, subscription.UserAgent)
	assert.Equal(t, userAgent, *subscription.UserAgent)
	assert.Len(t, mustRepositoryFinalPushSubscriptions(t, repo, ctx, user.ID), 1)
	invalidSubscription, err := repo.UpsertPushSubscription(ctx, user.ID, "https://push.final/sqlite-invalid", "p256dh-invalid", "auth-invalid", nil)
	require.NoError(t, err)
	require.NoError(t, repo.RemovePushSubscriptionByID(ctx, invalidSubscription.ID))
	require.NoError(t, repo.DeletePushSubscription(ctx, user.ID, subscription.Endpoint))
	assert.Empty(t, mustRepositoryFinalPushSubscriptions(t, repo, ctx, user.ID))

	_, err = client.AggregateEvent.Query().Where(aggregateevent.UserID(user.ID)).Count(ctx)
	require.NoError(t, err)
	repo.RefreshStore(ctx)
	export, err := repo.BuildUserExportSnapshot(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, export["account"].(model.User).ID)
	assert.Len(t, export["devices"], 1)

	dashboard, protectionStatus, progress, err := repo.GetDashboardData(ctx, user.ID, now)
	require.NoError(t, err)
	assert.Equal(t, "synced", dashboard.DataState)
	assert.Equal(t, 1, protectionStatus.DeviceCount)
	assert.Equal(t, "synced", progress.DataState)
	assert.Equal(t, 9, dashboard.BlockedAttempts)

	platform, err := repo.PlatformAnalytics(ctx, 3, now)
	require.NoError(t, err)
	assert.Equal(t, 1, platform.ProtectedUsers)
	assert.Equal(t, 9, platform.Totals.Blocked)

	require.NoError(t, repo.DeleteUserAccountData(ctx, user.ID, now))
	_, found = repo.UserByID(ctx, user.ID)
	assert.False(t, found)
}
