package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func TestAggregateForMembershipAt_UsesSevenJakartaCalendarDates(t *testing.T) {
	now := time.Date(2026, time.August, 2, 18, 30, 0, 0, time.UTC)
	eventDate := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	snapshot := &store.Store{AggregateEvents: []model.AggregateEvent{
		{UserID: "student", DeviceID: "android", EventType: "block_count_sync", EventDate: eventDate(2026, time.August, 3), Count: 2},
		{UserID: "student", DeviceID: "windows", EventType: "block_count_sync", EventDate: eventDate(2026, time.July, 28), Count: 3},
		{UserID: "student", DeviceID: "android", EventType: "block_count_sync", EventDate: eventDate(2026, time.July, 27), Count: 100},
		{UserID: "student", DeviceID: "android", EventType: "block_count_sync", EventDate: eventDate(2026, time.August, 4), Count: 100},
		{UserID: "student", DeviceID: "android", EventType: "intervention_shown", EventDate: eventDate(2026, time.August, 3), Count: 50},
		{UserID: "other", DeviceID: "other-device", EventType: "block_count_sync", EventDate: eventDate(2026, time.August, 3), Count: 75},
	}, CheckIns: []model.CheckIn{
		{UserID: "student", CreatedAt: time.Date(2026, time.August, 3, 3, 0, 0, 0, time.UTC)},
		{UserID: "student", CreatedAt: time.Date(2026, time.July, 27, 18, 0, 0, 0, time.UTC)},
		{UserID: "student", CreatedAt: time.Date(2026, time.July, 26, 18, 0, 0, 0, time.UTC)},
		{UserID: "student", CreatedAt: time.Date(2026, time.August, 3, 18, 0, 0, 0, time.UTC)},
	}}
	membership := model.AccountabilityMembership{
		StudentID: "student",
		Sharing: model.SharingPreferences{
			ProtectionActivity: true,
			RecoveryEngagement: true,
		},
	}

	result := aggregateForMembershipAt(snapshot, membership, now)

	assert.Equal(t, 5, result.WeeklyBlockCount)
	assert.Equal(t, 2, result.CheckInDays)
}

func TestAggregateForMembershipAt_HidesProtectionActivityWithoutConsent(t *testing.T) {
	now := time.Date(2026, time.August, 2, 18, 30, 0, 0, time.UTC)
	snapshot := &store.Store{AggregateEvents: []model.AggregateEvent{{
		UserID: "student", DeviceID: "android", EventType: "block_count_sync",
		EventDate: time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC), Count: 9,
	}}}
	membership := model.AccountabilityMembership{StudentID: "student"}

	result := aggregateForMembershipAt(snapshot, membership, now)

	assert.Zero(t, result.WeeklyBlockCount)
}
