package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func midnightUTC(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func hourlyMetadata(counts ...int) map[string]any {
	hourly := make([]any, 24)
	for index := range hourly {
		if index < len(counts) {
			hourly[index] = float64(counts[index])
		} else {
			hourly[index] = float64(0)
		}
	}
	return map[string]any{"hourly": hourly}
}

func TestPartnerAnalytics_ConsentsAndHourly(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	st := &store.Store{
		AccountabilityGroups: []model.AccountabilityGroup{
			{ID: "grp", OwnerPartnerID: "prt", Name: "Group", Status: "active"},
		},
		AccountabilityMemberships: []model.AccountabilityMembership{
			{ID: "mbr_a", GroupID: "grp", StudentID: "stu_a", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: true}},
			{ID: "mbr_b", GroupID: "grp", StudentID: "stu_b", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: false}},
		},
		AggregateEvents: []model.AggregateEvent{
			{UserID: "stu_a", DeviceID: "dev_a", EventType: "block_count_sync", EventDate: midnightUTC(2026, time.August, 5), Count: 5, MetadataJSON: hourlyMetadata(1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 2, 0)},
			{UserID: "stu_b", DeviceID: "dev_b", EventType: "block_count_sync", EventDate: midnightUTC(2026, time.August, 5), Count: 100},
		},
	}
	repo := New(nil, st)

	summary, err := repo.PartnerAnalytics(t.Context(), "prt", "", 14, now)
	require.NoError(t, err)

	assert.Equal(t, 2, summary.MemberCount)
	assert.Equal(t, 1, summary.SharedMemberCount)
	assert.Equal(t, 5, summary.Totals.Blocked)
	assert.Equal(t, 1, summary.Hourly[0].Count)
	assert.Equal(t, 2, summary.Hourly[21].Count)
	assert.Equal(t, 2, summary.Hourly[22].Count)
	assert.Equal(t, 14, len(summary.Daily))
}

func TestPartnerAnalytics_UnknownGroupRejected(t *testing.T) {
	st := &store.Store{
		AccountabilityGroups: []model.AccountabilityGroup{
			{ID: "grp", OwnerPartnerID: "prt", Name: "Group", Status: "active"},
		},
	}
	repo := New(nil, st)

	_, err := repo.PartnerAnalytics(t.Context(), "prt", "other-group", 14, time.Now())
	require.Error(t, err)
}

func TestPlatformAnalytics_SumsAcrossUsersAndProtected(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	st := &store.Store{
		Devices: []model.Device{
			{UserID: "stu_a", ProtectionStatus: "active"},
			{UserID: "stu_b", ProtectionStatus: "active"},
			{UserID: "stu_c", ProtectionStatus: "inactive"},
		},
		AggregateEvents: []model.AggregateEvent{
			{UserID: "stu_a", EventType: "block_count_sync", EventDate: midnightUTC(2026, time.August, 5), Count: 3},
			{UserID: "stu_b", EventType: "block_count_sync", EventDate: midnightUTC(2026, time.August, 5), Count: 4},
			{UserID: "stu_a", EventType: "intervention_shown", EventDate: midnightUTC(2026, time.August, 5), Count: 2},
			{UserID: "stu_a", EventType: "tamper_detected", EventDate: midnightUTC(2026, time.August, 6), Count: 1},
			{UserID: "stu_a", EventType: "permission_revoked", EventDate: midnightUTC(2026, time.August, 6), Count: 1},
		},
	}
	repo := New(nil, st)

	summary, err := repo.PlatformAnalytics(t.Context(), 14, now)
	require.NoError(t, err)

	assert.Equal(t, 7, summary.Totals.Blocked)
	assert.Equal(t, 2, summary.Totals.Interventions)
	assert.Equal(t, 1, summary.Totals.TamperEvents)
	assert.Equal(t, 1, summary.Totals.PermissionRevoked)
	assert.Equal(t, 2, summary.ProtectedUsers)
	assert.Equal(t, 14, len(summary.Daily))
}

func TestBuildAnalyticsSummary_MergesHourlyAcrossDays(t *testing.T) {
	now := time.Date(2026, time.August, 5, 12, 0, 0, 0, time.UTC)
	events := []model.AggregateEvent{
		{EventType: "block_count_sync", EventDate: midnightUTC(2026, time.August, 5), Count: 2, MetadataJSON: hourlyMetadata(1, 1)},
		{EventType: "block_count_sync", EventDate: midnightUTC(2026, time.August, 4), Count: 3, MetadataJSON: hourlyMetadata(1)},
	}
	summary := buildAnalyticsSummary(events, 14, now, false)

	assert.Equal(t, 5, summary.Totals.Blocked)
	assert.Equal(t, 2, summary.Hourly[0].Count)
	assert.Equal(t, 1, summary.Hourly[1].Count)
	assert.Equal(t, "local_only", summary.DataState)
}
