package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseReminderTime(t *testing.T) {
	assert.Equal(t, 19, hourOf("19:00"))
	assert.Equal(t, 0, minuteOf("19:00"))
	assert.Equal(t, 6, hourOf("06:30"))
	assert.Equal(t, 30, minuteOf("06:30"))
	// Invalid values fall back to the 19:00 default.
	assert.Equal(t, 19, hourOf("nope"))
	assert.Equal(t, 0, minuteOf("25:99"))
}

func TestDailyReminderPayload(t *testing.T) {
	payload := dailyReminderPayload("id")
	assert.NotEmpty(t, payload.Title)
	assert.NotEmpty(t, payload.Body)
	assert.Equal(t, "/icons/icon-192.png", payload.Icon)
	assert.Equal(t, "/id/recovery", payload.URL)

	en := dailyReminderPayload("en")
	assert.Equal(t, "/en/recovery", en.URL)

	unknown := dailyReminderPayload("fr")
	assert.Equal(t, "/id/recovery", unknown.URL)
}

func TestSameCalendarDay(t *testing.T) {
	loc := time.FixedZone("test", 7*3600)
	ref := time.Date(2026, 8, 9, 23, 30, 0, 0, loc)
	same := time.Date(2026, 8, 9, 16, 0, 0, 0, time.UTC)
	nextDay := time.Date(2026, 8, 10, 0, 30, 0, 0, loc)
	assert.True(t, sameCalendarDay(same, ref, loc))
	assert.False(t, sameCalendarDay(nextDay, ref, loc))
}

func hourOf(value string) int {
	h, _ := parseReminderTime(value)
	return h
}

func minuteOf(value string) int {
	_, m := parseReminderTime(value)
	return m
}
