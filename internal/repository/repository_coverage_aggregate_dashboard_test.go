package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func repositoryCoverageDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

func repositoryCoverageHourlyValues(first, second float64) []any {
	values := make([]any, 24)
	values[0] = first
	values[23] = second
	return values
}

func TestRepositoryCoverage_AggregateDateWindowAndPrivacySafeTotals(t *testing.T) {
	now := repositoryCoverageDate(2026, time.July, 15)
	start := time.Date(2026, time.July, 13, 0, 0, 0, 0, time.UTC)
	st := &store.Store{
		Devices: []model.Device{{ID: "cov-device", UserID: "cov-user", ProtectionStatus: "active"}},
		AggregateEvents: []model.AggregateEvent{
			{ID: "cov-block-start", UserID: "cov-user", DeviceID: "cov-device", EventType: "block_count_sync", EventDate: start, Count: 2},
			{ID: "cov-intervention", UserID: "cov-user", DeviceID: "cov-device", EventType: "intervention_shown", EventDate: now.Add(-24 * time.Hour), Count: 3},
			{ID: "cov-tamper", UserID: "cov-user", DeviceID: "cov-device", EventType: "tamper_detected", EventDate: now, Count: 4},
			{ID: "cov-permission", UserID: "cov-user", DeviceID: "cov-device", EventType: "permission_revoked", EventDate: now, Count: 5},
			{ID: "cov-old", UserID: "cov-user", DeviceID: "cov-device", EventType: "block_count_sync", EventDate: now.Add(-4 * 24 * time.Hour), Count: 100},
			{ID: "cov-unknown", UserID: "cov-user", DeviceID: "cov-device", EventType: "raw_url", EventDate: now, Count: 1000},
			{ID: "cov-other-user", UserID: "other-user", DeviceID: "cov-device", EventType: "block_count_sync", EventDate: now, Count: 1000},
		},
	}
	repo := New(nil, st)

	analytics, err := repo.GetProtectionAnalytics(t.Context(), "cov-user", "cov-device", 3, now)
	require.NoError(t, err)
	assert.Equal(t, "cov-device", analytics.DeviceID)
	assert.Equal(t, "local_only", analytics.DataState)
	assert.Equal(t, model.ProtectionAnalyticsTotals{Blocked: 2, Interventions: 3, TamperEvents: 4, PermissionRevoked: 5}, analytics.Totals)
	assert.Equal(t, []int{2, 0, 0}, []int{analytics.Daily[0].Blocked, analytics.Daily[1].Blocked, analytics.Daily[2].Blocked})
	assert.Equal(t, 3, analytics.Daily[1].Interventions)
	assert.Equal(t, 4, analytics.Daily[2].TamperEvents)
	assert.Equal(t, 5, analytics.Daily[2].PermissionRevoked)

	emptyRepo := New(nil, &store.Store{Devices: st.Devices})
	empty, err := emptyRepo.GetProtectionAnalytics(t.Context(), "cov-user", "cov-device", 3, now)
	require.NoError(t, err)
	assert.Equal(t, "empty", empty.DataState)
	assert.Zero(t, empty.Totals)
	assert.EqualError(t, func() error {
		_, err := repo.GetProtectionAnalytics(t.Context(), "other-user", "cov-device", 3, now)
		return err
	}(), "device does not belong to user")
}

func TestRepositoryCoverage_AggregateIdempotencyAndMonotonicSnapshot(t *testing.T) {
	repo := New(nil, &store.Store{})
	event := model.AggregateEvent{
		ID: "cov-write-event", UserID: "cov-write-user", DeviceID: "cov-write-device",
		IdempotencyKey: "cov-write-key", EventType: "block_count_sync",
		EventDate: repositoryCoverageDate(2026, time.July, 15), Count: 4,
	}
	first, err := repo.SaveAggregateEvent(t.Context(), event)
	require.NoError(t, err)
	duplicate, err := repo.SaveAggregateEvent(t.Context(), model.AggregateEvent{
		ID: "cov-write-duplicate", UserID: "cov-write-user", IdempotencyKey: event.IdempotencyKey, Count: 99,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, duplicate.ID)
	assert.Equal(t, 4, duplicate.Count)

	snapshot, err := repo.SaveAggregateEventSnapshot(t.Context(), model.AggregateEvent{
		ID: "cov-write-snapshot", UserID: "cov-write-user", DeviceID: "cov-write-device",
		IdempotencyKey: "cov-write-snapshot-key", EventType: "block_count_sync", Count: 10,
		MetadataJSON: map[string]any{"revision": 1},
	})
	require.NoError(t, err)
	assert.Equal(t, 10, snapshot.Count)
	stale, err := repo.SaveAggregateEventSnapshot(t.Context(), model.AggregateEvent{
		UserID: "cov-write-user", IdempotencyKey: "cov-write-snapshot-key", Count: 2,
		MetadataJSON: map[string]any{"revision": 2},
	})
	require.NoError(t, err)
	assert.Equal(t, 10, stale.Count)
	higher, err := repo.SaveAggregateEventSnapshot(t.Context(), model.AggregateEvent{
		UserID: "cov-write-user", IdempotencyKey: "cov-write-snapshot-key", Count: 12,
		MetadataJSON: map[string]any{"revision": 3},
	})
	require.NoError(t, err)
	assert.Equal(t, 12, higher.Count)
	assert.Equal(t, 3, higher.MetadataJSON["revision"])
}

func TestRepositoryCoverage_AnalyticsConsentDateWindowAndHistogramValidation(t *testing.T) {
	now := repositoryCoverageDate(2026, time.July, 15)
	st := &store.Store{
		AccountabilityGroups: []model.AccountabilityGroup{
			{ID: "cov-group", OwnerPartnerID: "cov-partner", Status: "active"},
			{ID: "other-group", OwnerPartnerID: "other-partner", Status: "active"},
		},
		AccountabilityMemberships: []model.AccountabilityMembership{
			{ID: "cov-live-shared", GroupID: "cov-group", StudentID: "cov-shared", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: true}},
			{ID: "cov-live-private", GroupID: "cov-group", StudentID: "cov-private", Status: "active"},
			{ID: "cov-left-shared", GroupID: "cov-group", StudentID: "cov-left", Status: "left", Sharing: model.SharingPreferences{ProtectionActivity: true}},
			{ID: "cov-other-group", GroupID: "other-group", StudentID: "cov-other", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: true}},
		},
		AggregateEvents: []model.AggregateEvent{
			{UserID: "cov-shared", EventType: "block_count_sync", EventDate: now, Count: 7, MetadataJSON: map[string]any{"hourly": repositoryCoverageHourlyValues(2, 3)}},
			{UserID: "cov-shared", EventType: "intervention_shown", EventDate: now.Add(-24 * time.Hour), Count: 4},
			{UserID: "cov-shared", EventType: "block_count_sync", EventDate: now.Add(-10 * 24 * time.Hour), Count: 100},
			{UserID: "cov-private", EventType: "block_count_sync", EventDate: now, Count: 99},
			{UserID: "cov-left", EventType: "block_count_sync", EventDate: now, Count: 99},
			{UserID: "cov-shared", EventType: "not-an-analytics-event", EventDate: now, Count: 99},
		},
	}
	repo := New(nil, st)

	summary, err := repo.PartnerAnalytics(t.Context(), "cov-partner", "cov-group", 7, now)
	require.NoError(t, err)
	assert.Equal(t, 2, summary.MemberCount)
	assert.Equal(t, 1, summary.SharedMemberCount)
	assert.Equal(t, 7, summary.Totals.Blocked)
	assert.Equal(t, 4, summary.Totals.Interventions)
	assert.Equal(t, 2, summary.Hourly[0].Count)
	assert.Equal(t, 3, summary.Hourly[23].Count)
	assert.Equal(t, "local_only", summary.DataState)

	filtered, err := repo.PartnerAnalytics(t.Context(), "cov-partner", "", 7, now)
	require.NoError(t, err)
	assert.Equal(t, summary.Totals, filtered.Totals)
	assert.Equal(t, 7, filtered.PeriodDays)

	privateRepo := New(nil, &store.Store{
		AccountabilityGroups:      st.AccountabilityGroups[:1],
		AccountabilityMemberships: []model.AccountabilityMembership{{ID: "cov-private-only", GroupID: "cov-group", StudentID: "cov-private", Status: "active"}},
	})
	privateSummary, err := privateRepo.PartnerAnalytics(t.Context(), "cov-partner", "cov-group", 5, now)
	require.NoError(t, err)
	assert.Equal(t, 1, privateSummary.MemberCount)
	assert.Zero(t, privateSummary.SharedMemberCount)
	assert.Equal(t, "empty", privateSummary.DataState)
	assert.Len(t, privateSummary.Daily, 5)
	assert.EqualError(t, func() error {
		_, err := repo.PartnerAnalytics(t.Context(), "cov-partner", "missing-group", 7, now)
		return err
	}(), "group is not owned by the partner")

	filteredEvents, err := repo.aggregateEventsForUsers(t.Context(), nil, 7, now)
	require.NoError(t, err)
	assert.Empty(t, filteredEvents)
	malformed := buildAnalyticsSummary([]model.AggregateEvent{
		{EventType: "block_count_sync", EventDate: now, Count: 1, MetadataJSON: nil},
		{EventType: "block_count_sync", EventDate: now, Count: 1, MetadataJSON: map[string]any{"hourly": "not-a-slice"}},
		{EventType: "block_count_sync", EventDate: now, Count: 1, MetadataJSON: map[string]any{"hourly": []any{1.0}}},
		{EventType: "block_count_sync", EventDate: now, Count: 1, MetadataJSON: map[string]any{"hourly": func() []any {
			values := make([]any, 24)
			values[0], values[1], values[2] = 1.0, -1.0, 2
			return values
		}()}},
	}, 7, now, true)
	assert.Equal(t, 4, malformed.Totals.Blocked)
	assert.Equal(t, 1, malformed.Hourly[0].Count)
	assert.Zero(t, malformed.Hourly[1].Count)
	assert.Zero(t, malformed.Hourly[2].Count)
	assert.Equal(t, "synced", malformed.DataState)
}

func TestRepositoryCoverage_PlatformAnalyticsFiltersEventsAndCountsProtectedUsers(t *testing.T) {
	now := repositoryCoverageDate(2026, time.July, 15)
	repo := New(nil, &store.Store{
		Devices: []model.Device{
			{ID: "cov-platform-device-a", UserID: "cov-platform-user", ProtectionStatus: "active"},
			{ID: "cov-platform-device-b", UserID: "cov-platform-user", ProtectionStatus: "active"},
			{ID: "cov-platform-device-c", UserID: "cov-platform-inactive", ProtectionStatus: "inactive"},
		},
		AggregateEvents: []model.AggregateEvent{
			{UserID: "cov-platform-user", EventType: "block_count_sync", EventDate: now, Count: 5},
			{UserID: "cov-platform-user", EventType: "intervention_shown", EventDate: now, Count: 4},
			{UserID: "cov-platform-user", EventType: "tamper_detected", EventDate: now.Add(-24 * time.Hour), Count: 3},
			{UserID: "cov-platform-user", EventType: "permission_revoked", EventDate: now.Add(-24 * time.Hour), Count: 2},
			{UserID: "cov-platform-user", EventType: "unsupported-event", EventDate: now, Count: 100},
			{UserID: "cov-platform-user", EventType: "block_count_sync", EventDate: now.Add(-4 * 24 * time.Hour), Count: 100},
		},
	})

	summary, err := repo.PlatformAnalytics(t.Context(), 3, now)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.ProtectedUsers)
	assert.Equal(t, model.AnalyticsTotals{Blocked: 5, Interventions: 4, TamperEvents: 3, PermissionRevoked: 2}, summary.Totals)
	assert.Equal(t, "local_only", summary.DataState)
	assert.Len(t, summary.Daily, 3)
}

func TestRepositoryCoverage_DashboardAndProgressDateWindows(t *testing.T) {
	now := repositoryCoverageDate(2026, time.July, 15)
	st := &store.Store{
		Users: []model.User{{ID: "cov-dashboard-user", DisplayName: "Dashboard Coverage"}},
		Devices: []model.Device{
			{ID: "cov-active-device", UserID: "cov-dashboard-user", ProtectionStatus: "active", LastSeenAt: now.Add(-48 * time.Hour), ModelVersion: "model-old", RulesetVersion: "rules-old"},
			{ID: "cov-inactive-device", UserID: "cov-dashboard-user", ProtectionStatus: "inactive", LastSeenAt: now.Add(-24 * time.Hour), ModelVersion: "model-new", RulesetVersion: "rules-new"},
		},
		CheckIns: []model.CheckIn{
			{ID: "cov-check-today", UserID: "cov-dashboard-user", Mood: 5, Urge: 1, CreatedAt: now},
			{ID: "cov-check-yesterday", UserID: "cov-dashboard-user", Mood: 4, Urge: 2, CreatedAt: now.Add(-24 * time.Hour)},
			{ID: "cov-check-two-days", UserID: "cov-dashboard-user", Mood: 3, Urge: 3, CreatedAt: now.Add(-48 * time.Hour)},
			{ID: "cov-check-old", UserID: "cov-dashboard-user", Mood: 1, Urge: 5, CreatedAt: now.Add(-40 * 24 * time.Hour)},
			{ID: "cov-check-other", UserID: "other-user", Mood: 1, Urge: 1, CreatedAt: now},
		},
		JournalEntries: []model.JournalEntry{
			{ID: "cov-journal-current", UserID: "cov-dashboard-user", CreatedAt: now.Add(-24 * time.Hour)},
			{ID: "cov-journal-old", UserID: "cov-dashboard-user", CreatedAt: now.Add(-40 * 24 * time.Hour)},
			{ID: "cov-journal-other", UserID: "other-user", CreatedAt: now},
		},
		Missions: []model.DailyMission{
			{ID: "cov-mission-current", UserID: "cov-dashboard-user", Date: now.Format("2006-01-02"), Mission1: true},
			{ID: "cov-mission-empty", UserID: "cov-dashboard-user", Date: now.Format("2006-01-02")},
			{ID: "cov-mission-old", UserID: "cov-dashboard-user", Date: now.Add(-40 * 24 * time.Hour).Format("2006-01-02"), Mission1: true},
		},
		AggregateEvents: []model.AggregateEvent{
			{ID: "cov-dashboard-block-old", UserID: "cov-dashboard-user", EventType: "block_count_sync", EventDate: now.Add(-8 * 24 * time.Hour), Count: 100},
			{ID: "cov-dashboard-block-week", UserID: "cov-dashboard-user", EventType: "block_count_sync", EventDate: now.Add(-6 * 24 * time.Hour), Count: 2},
			{ID: "cov-dashboard-block-today", UserID: "cov-dashboard-user", EventType: "block_count_sync", EventDate: now, Count: 3},
			{ID: "cov-dashboard-other-event", UserID: "cov-dashboard-user", EventType: "tamper_detected", EventDate: now, Count: 50},
		},
		LearningProgress: []model.LearningProgress{
			{UserID: "cov-dashboard-user", ItemID: "cov-learning-created", CreatedAt: now.Add(-3 * 24 * time.Hour)},
			{UserID: "cov-dashboard-user", ItemID: "cov-learning-updated", CreatedAt: now.Add(-5 * 24 * time.Hour), UpdatedAt: now.Add(-2 * 24 * time.Hour)},
			{UserID: "cov-dashboard-user", ItemID: "cov-learning-empty"},
			{UserID: "other-user", ItemID: "cov-learning-other", UpdatedAt: now},
		},
		EducationProgress: []model.EducationProgress{
			{ID: "cov-education-current", UserID: "cov-dashboard-user", UpdatedAt: now.Add(-4 * 24 * time.Hour)},
			{ID: "cov-education-other", UserID: "other-user", UpdatedAt: now},
		},
		RecoveryRecords: []model.RecoveryRecord{
			{ID: "cov-review-current", UserID: "cov-dashboard-user", Kind: "weekly_review", RecordDate: now.Add(-5 * 24 * time.Hour).Format("2006-01-02")},
			{ID: "cov-record-other-kind", UserID: "cov-dashboard-user", Kind: "check_in", RecordDate: now.Format("2006-01-02")},
			{ID: "cov-review-other", UserID: "other-user", Kind: "weekly_review", RecordDate: now.Format("2006-01-02")},
		},
	}
	repo := New(nil, st)

	missing, protection, progress, err := repo.GetDashboardData(t.Context(), "missing-user", now)
	require.NoError(t, err)
	assert.Zero(t, missing)
	assert.Zero(t, protection)
	assert.Zero(t, progress)

	dashboard, protection, progress, err := repo.GetDashboardData(t.Context(), "cov-dashboard-user", now)
	require.NoError(t, err)
	assert.Equal(t, "Dashboard Coverage", dashboard.UserName)
	assert.Equal(t, "active", dashboard.ProtectionLabel)
	assert.Equal(t, "local_only", dashboard.DataState)
	assert.Equal(t, 5, dashboard.BlockedAttempts)
	assert.Equal(t, 3, dashboard.CurrentStreak)
	assert.Equal(t, 2, protection.DeviceCount)
	assert.Equal(t, "model-new", protection.ModelVersion)
	assert.Equal(t, "rules-new", protection.RulesetVersion)
	assert.Equal(t, now.Add(-24*time.Hour), *protection.LastSync)
	assert.Equal(t, 2, progress.Reflections)
	assert.Len(t, progress.WeeklyBlocks, 7)
	assert.Equal(t, 2, progress.WeeklyBlocks[0])
	assert.Equal(t, 3, progress.WeeklyBlocks[6])

	progress, err = repo.GetProgressData(t.Context(), "cov-dashboard-user", 30, now)
	require.NoError(t, err)
	assert.Equal(t, 30, progress.RangeDays)
	assert.Len(t, progress.DailyBlocks, 30)
	assert.Len(t, progress.WeeklyBlocks, 7)
	assert.Equal(t, 3, progress.CheckInCount)
	assert.True(t, progress.TrendAvailable)
	assert.Equal(t, 1, progress.Reflections)
	assert.Equal(t, "synced", progress.DataState)
	assert.NotEmpty(t, progress.ActivityDays)
	assert.EqualError(t, func() error {
		_, err := repo.GetProgressData(t.Context(), "cov-dashboard-user", 8, now)
		return err
	}(), "progress range must be 7, 30, or 90 days")
}

func TestRepositoryCoverage_ExportScopesAccountAndSupportRecords(t *testing.T) {
	st := &store.Store{
		Users: []model.User{
			{ID: "cov-export-user", Email: "export@example.com"},
			{ID: "other-export-user", Email: "other@example.com"},
		},
		Devices: []model.Device{
			{ID: "cov-export-device", UserID: "cov-export-user"},
			{ID: "other-export-device", UserID: "other-export-user"},
		},
		AggregateEvents: []model.AggregateEvent{
			{ID: "cov-export-aggregate", UserID: "cov-export-user", EventType: "block_count_sync"},
			{ID: "other-export-aggregate", UserID: "other-export-user", EventType: "block_count_sync"},
		},
		Intentions:                []model.Intention{{ID: "cov-export-intention", UserID: "cov-export-user"}, {ID: "other-export-intention", UserID: "other-export-user"}},
		CheckIns:                  []model.CheckIn{{ID: "cov-export-check", UserID: "cov-export-user"}, {ID: "other-export-check", UserID: "other-export-user"}},
		RecoveryRecords:           []model.RecoveryRecord{{ID: "cov-export-recovery", UserID: "cov-export-user"}, {ID: "other-export-recovery", UserID: "other-export-user"}},
		RecoveryPracticeSessions:  []model.RecoveryPracticeSession{{ID: "cov-export-practice", UserID: "cov-export-user"}, {ID: "other-export-practice", UserID: "other-export-user"}},
		RecoverySpaces:            []model.RecoverySpace{{ID: "cov-export-space", UserID: "cov-export-user"}, {ID: "other-export-space", UserID: "other-export-user"}},
		JournalEntries:            []model.JournalEntry{{ID: "cov-export-journal", UserID: "cov-export-user"}, {ID: "other-export-journal", UserID: "other-export-user"}},
		Missions:                  []model.DailyMission{{ID: "cov-export-mission", UserID: "cov-export-user"}, {ID: "other-export-mission", UserID: "other-export-user"}},
		EducationProgress:         []model.EducationProgress{{ID: "cov-export-education", UserID: "cov-export-user"}, {ID: "other-export-education", UserID: "other-export-user"}},
		AccountabilityMemberships: []model.AccountabilityMembership{{ID: "cov-export-membership", StudentID: "cov-export-user"}, {ID: "other-export-membership", StudentID: "other-export-user"}},
		AccountabilityGroups:      []model.AccountabilityGroup{{ID: "cov-export-group", OwnerPartnerID: "cov-export-user"}, {ID: "other-export-group", OwnerPartnerID: "other-export-user"}},
		Approvals: []model.ApprovalRequest{
			{ID: "cov-export-approval", UserID: "cov-export-user"},
			{ID: "cov-export-resolved", UserID: "other-export-user", ResolvedBy: "cov-export-user"},
			{ID: "other-export-approval", UserID: "other-export-user"},
		},
		SupportCases: []model.SupportCase{{ID: "cov-export-case", UserID: "cov-export-user"}, {ID: "other-export-case", UserID: "other-export-user"}},
		SupportMessages: []model.SupportMessage{
			{ID: "cov-export-message", SupportCaseID: "cov-export-case"},
			{ID: "other-export-message", SupportCaseID: "other-export-case"},
		},
	}
	repo := New(nil, st)

	snapshot, err := repo.BuildUserExportSnapshot(t.Context(), "cov-export-user")
	require.NoError(t, err)
	assert.Equal(t, "cov-export-user", snapshot["account"].(model.User).ID)
	for _, key := range []string{
		"devices", "aggregate_protection_events", "intentions", "check_ins", "recovery_records",
		"recovery_practice_sessions", "recovery_spaces", "reflections", "missions", "education_progress",
		"accountability_memberships", "owned_accountability_groups", "approval_requests", "support_cases", "support_messages",
	} {
		expectedLength := 1
		if key == "approval_requests" {
			expectedLength = 2
		}
		assert.Len(t, snapshot[key], expectedLength, key)
	}
	assert.NotContains(t, snapshot, "urls")
	assert.NotContains(t, snapshot, "browsing_history")
	_, err = repo.BuildUserExportSnapshot(t.Context(), "missing-export-user")
	assert.EqualError(t, err, "user not found")
}

func TestRepositoryCoverage_DeleteAnonymizesAndPreservesOtherAccount(t *testing.T) {
	now := repositoryCoverageDate(2026, time.July, 15)
	st := &store.Store{
		Users: []model.User{
			{ID: "cov-delete-user", Email: "delete@example.com", PhoneE164: "+628111111111"},
			{ID: "other-delete-user", Email: "other-delete@example.com"},
		},
		ContactVerifications:      []model.ContactVerification{{ID: "cov-delete-contact", UserID: "cov-delete-user"}, {ID: "other-delete-contact", UserID: "other-delete-user"}},
		Devices:                   []model.Device{{ID: "cov-delete-device", UserID: "cov-delete-user"}, {ID: "other-delete-device", UserID: "other-delete-user"}},
		Partners:                  []model.Partner{{ID: "cov-delete-partner", UserID: "cov-delete-user"}, {ID: "other-delete-partner", UserID: "other-delete-user"}},
		AccountabilityGroups:      []model.AccountabilityGroup{{ID: "cov-delete-group", OwnerPartnerID: "cov-delete-user"}, {ID: "other-delete-group", OwnerPartnerID: "other-delete-user"}},
		AccountabilityMemberships: []model.AccountabilityMembership{{ID: "cov-delete-membership", GroupID: "cov-delete-group", StudentID: "other-delete-user"}, {ID: "other-delete-membership", GroupID: "other-delete-group", StudentID: "other-delete-user"}},
		MembershipExitRequests:    []model.MembershipExitRequest{{ID: "cov-delete-exit", MembershipID: "cov-delete-membership"}, {ID: "other-delete-exit", MembershipID: "other-delete-membership"}},
		PartnerContactRequests:    []model.PartnerContactRequest{{ID: "cov-delete-contact-request", MembershipID: "cov-delete-membership"}, {ID: "other-delete-contact-request", MembershipID: "other-delete-membership"}},
		Approvals:                 []model.ApprovalRequest{{ID: "cov-delete-approval", UserID: "cov-delete-user"}, {ID: "other-delete-approval", UserID: "other-delete-user"}},
		SupportCases:              []model.SupportCase{{ID: "cov-delete-case", UserID: "cov-delete-user"}, {ID: "other-delete-case", UserID: "other-delete-user"}},
		SupportMessages:           []model.SupportMessage{{ID: "cov-delete-message", SupportCaseID: "cov-delete-case"}, {ID: "other-delete-message", SupportCaseID: "other-delete-case"}},
		JournalEntries:            []model.JournalEntry{{ID: "cov-delete-journal", UserID: "cov-delete-user"}, {ID: "other-delete-journal", UserID: "other-delete-user"}},
		Missions:                  []model.DailyMission{{ID: "cov-delete-mission", UserID: "cov-delete-user"}, {ID: "other-delete-mission", UserID: "other-delete-user"}},
		Intentions:                []model.Intention{{ID: "cov-delete-intention", UserID: "cov-delete-user"}, {ID: "other-delete-intention", UserID: "other-delete-user"}},
		CheckIns:                  []model.CheckIn{{ID: "cov-delete-check", UserID: "cov-delete-user"}, {ID: "other-delete-check", UserID: "other-delete-user"}},
		RecoveryRecords:           []model.RecoveryRecord{{ID: "cov-delete-recovery", UserID: "cov-delete-user"}, {ID: "other-delete-recovery", UserID: "other-delete-user"}},
		RecoveryPracticeSessions:  []model.RecoveryPracticeSession{{ID: "cov-delete-practice", UserID: "cov-delete-user"}, {ID: "other-delete-practice", UserID: "other-delete-user"}},
		RecoverySpaces:            []model.RecoverySpace{{ID: "cov-delete-space", UserID: "cov-delete-user"}, {ID: "other-delete-space", UserID: "other-delete-user"}},
		AggregateEvents:           []model.AggregateEvent{{ID: "cov-delete-aggregate", UserID: "cov-delete-user"}, {ID: "other-delete-aggregate", UserID: "other-delete-user"}},
		DataRequests: []model.DataRequest{
			{ID: "cov-delete-request", UserID: "cov-delete-user", Type: "delete", Status: "processing", ConfirmationTokenHash: "hash", ResultPath: "private.zip"},
			{ID: "cov-export-request", UserID: "cov-delete-user", Type: "export", Status: "processing", ResultPath: "export.zip"},
			{ID: "other-delete-request", UserID: "other-delete-user", Type: "delete", Status: "processing"},
		},
		AuditEvents: []model.AuditEvent{{ID: "cov-delete-audit", ActorID: "cov-delete-user", Actor: "Delete User"}, {ID: "other-delete-audit", ActorID: "other-delete-user", Actor: "Other User"}},
	}
	repo := New(nil, st)

	assert.EqualError(t, repo.DeleteUserAccountData(t.Context(), "missing-delete-user", now), "user not found")
	require.NoError(t, repo.DeleteUserAccountData(t.Context(), "cov-delete-user", now))
	_, ok := repo.UserByID(t.Context(), "cov-delete-user")
	assert.False(t, ok)
	_, ok = repo.UserByID(t.Context(), "other-delete-user")
	assert.True(t, ok)

	snapshot := st.Snapshot()
	for _, item := range snapshot.ContactVerifications {
		assert.NotEqual(t, "cov-delete-user", item.UserID)
	}
	for _, item := range snapshot.Devices {
		assert.NotEqual(t, "cov-delete-user", item.UserID)
	}
	for _, item := range snapshot.JournalEntries {
		assert.NotEqual(t, "cov-delete-user", item.UserID)
	}
	assert.Len(t, snapshot.AccountabilityGroups, 1)
	assert.Equal(t, "other-delete-group", snapshot.AccountabilityGroups[0].ID)
	assert.Len(t, snapshot.AccountabilityMemberships, 1)
	assert.Len(t, snapshot.SupportCases, 1)
	assert.Len(t, snapshot.SupportMessages, 1)
	assert.Len(t, snapshot.AggregateEvents, 1)

	var deletedRequest, exportRequest model.DataRequest
	for _, item := range snapshot.DataRequests {
		switch item.ID {
		case "cov-delete-request":
			deletedRequest = item
		case "cov-export-request":
			exportRequest = item
		}
	}
	assert.Contains(t, deletedRequest.UserID, "deleted:")
	assert.Equal(t, "completed", deletedRequest.Status)
	assert.Equal(t, now, *deletedRequest.CompletedAt)
	assert.Empty(t, deletedRequest.ConfirmationTokenHash)
	assert.Empty(t, deletedRequest.ResultPath)
	assert.Contains(t, exportRequest.UserID, "deleted:")
	assert.Equal(t, "processing", exportRequest.Status)
	assert.Empty(t, exportRequest.ResultPath)

	var deletedAudit model.AuditEvent
	for _, item := range snapshot.AuditEvents {
		if item.ID == "cov-delete-audit" {
			deletedAudit = item
		}
	}
	assert.Contains(t, deletedAudit.ActorID, "deleted:")
	assert.Equal(t, "deleted-account", deletedAudit.Actor)
}

func TestRepositoryCoverage_PortalMetricsAndEmptyOverview(t *testing.T) {
	st := &store.Store{
		Devices: []model.Device{
			{ID: "cov-portal-active-a", UserID: "portal-user-a", ProtectionStatus: "active"},
			{ID: "cov-portal-active-b", UserID: "portal-user-a", ProtectionStatus: "active"},
			{ID: "cov-portal-inactive", UserID: "portal-user-b", ProtectionStatus: "inactive"},
		},
		Approvals: []model.ApprovalRequest{
			{ID: "cov-portal-pending", Status: "pending"},
			{ID: "cov-portal-legacy-pending", Status: "Pending partner approval"},
			{ID: "cov-portal-approved", Status: "approved"},
		},
		SupportCases: []model.SupportCase{
			{ID: "cov-portal-open", Status: "waiting_support"},
			{ID: "cov-portal-resolved", Status: "resolved"},
			{ID: "cov-portal-closed", Status: "closed"},
		},
	}
	repo := New(nil, st)

	overview, err := repo.GetPortalOverview(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, overview.ProtectedUsers)
	assert.Equal(t, 2, overview.PartnerApprovals)
	assert.Equal(t, 67, overview.HealthyDevicesPercent)
	assert.Equal(t, 1, overview.OpenSupport)
	assert.Equal(t, 0, percentage(0, 0))
	assert.Equal(t, 50, percentage(1, 2))

	emptyOverview, err := New(nil, &store.Store{}).GetPortalOverview(t.Context())
	require.NoError(t, err)
	assert.Zero(t, emptyOverview)
}
