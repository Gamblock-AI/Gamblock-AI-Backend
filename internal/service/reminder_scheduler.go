package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

// ReminderScheduler runs a once-per-minute tick that delivers the opt-in daily
// Web Push reminder to users whose local time has just reached their chosen
// reminder minute. It is intended for a single API process; a horizontally
// scaled deployment would need a distributed lock or cron.
type ReminderScheduler struct {
	repo   *repository.Repository
	push   *PushService
	logger *zap.Logger
}

func NewReminderScheduler(repo *repository.Repository, push *PushService, logger *zap.Logger) *ReminderScheduler {
	return &ReminderScheduler{repo: repo, push: push, logger: logger}
}

func (s *ReminderScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Tick(ctx)
		}
	}
}

// Tick evaluates every enabled preference once and fires the ones whose local
// minute matches now and have not already fired today.
func (s *ReminderScheduler) Tick(ctx context.Context) {
	preferences, err := s.repo.EnabledReminderPreferences(ctx)
	if err != nil {
		s.logger.Error("reminder scheduler could not load preferences", zap.Error(err))
		return
	}
	now := time.Now()
	for _, pref := range preferences {
		location, err := time.LoadLocation(pref.Timezone)
		if err != nil {
			continue
		}
		localNow := now.In(location)
		hour, minute := parseReminderTime(pref.LocalTime)
		if localNow.Hour() != hour || localNow.Minute() != minute {
			continue
		}
		if pref.LastFiredAt != nil && sameCalendarDay(*pref.LastFiredAt, localNow, location) {
			continue
		}
		payload := dailyReminderPayload(pref.Locale)
		sent, err := s.push.SendToUser(ctx, pref.UserID, payload)
		if err != nil {
			s.logger.Info("daily reminder push failed",
				zap.String("user_id", pref.UserID), zap.Error(err))
			continue
		}
		s.logger.Info("daily reminder delivered",
			zap.String("user_id", pref.UserID), zap.Int("subscriptions", sent))
		if err := s.repo.MarkReminderFired(ctx, pref.UserID, now.UTC()); err != nil {
			s.logger.Info("reminder fired marker failed", zap.Error(err))
		}
	}
}

func parseReminderTime(value string) (int, int) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return 19, 0
	}
	hour, errH := strconv.Atoi(parts[0])
	minute, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 19, 0
	}
	return hour, minute
}

// sameCalendarDay reports whether the given UTC timestamp falls on the same
// calendar day as the reference time in the reference location.
func sameCalendarDay(ts, ref time.Time, loc *time.Location) bool {
	a := ts.In(loc)
	return a.Year() == ref.Year() && a.YearDay() == ref.YearDay()
}

type reminderMessages struct {
	title string
	body  string
}

var reminderCopies = map[string][]reminderMessages{
	"id": {
		{title: "Pengingat harianmu", body: "Jangan putus konsistensimu — isi check-in harianmu."},
		{title: "Satu langkah kecil", body: "Kamu sudah kuat hari ini. Satu langkah kecil tetap kemajuan."},
		{title: "Waktunya check-in", body: "Sempatkan refleksi singkat untuk mendukung pemulihanmu."},
		{title: "Tetap konsisten", body: "Tetap tenang, tetap konsisten. Check-in harian menunggumu."},
	},
	"en": {
		{title: "Your daily nudge", body: "Don't break your rhythm — complete your daily check-in."},
		{title: "One small step", body: "You've got this. One small step keeps your recovery moving."},
		{title: "Time to check in", body: "Take a quick moment to reflect and support your recovery."},
		{title: "Stay consistent", body: "Stay calm, stay consistent. Your daily check-in awaits."},
	},
}

func dailyReminderPayload(locale string) PushPayload {
	if locale != "en" {
		locale = "id"
	}
	copies := reminderCopies[locale]
	dayOfYear := time.Now().YearDay()
	message := copies[dayOfYear%len(copies)]
	return PushPayload{
		Title: message.title,
		Body:  message.body,
		Icon:  "/icons/icon-192.png",
		URL:   "/" + locale + "/recovery",
	}
}
