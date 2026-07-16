package repository

import (
	"context"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/checkin"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reflection"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetDashboardData(ctx context.Context, userID string, now time.Time) (model.DashboardSummary, model.ProtectionStatus, model.ProgressSnapshot, error) {
	user, ok := r.UserByID(ctx, userID)
	if !ok {
		return model.DashboardSummary{}, model.ProtectionStatus{}, model.ProgressSnapshot{}, nil
	}

	start := startOfDay(now.UTC()).AddDate(0, 0, -6)
	weeklyBlocks := make([]int, 7)
	moodPoints := make([]model.MoodPoint, 0)
	activityDays := make(map[string]struct{})
	blockedAttempts := 0
	dataState := "local_only"
	protection := model.ProtectionStatus{Mode: "inactive", RuntimeStatus: "not_connected"}
	reflectionCount := 0

	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, item := range snapshot.Devices {
			if item.UserID != userID {
				continue
			}
			protection.DeviceCount++
			if item.ProtectionStatus == "active" {
				protection.Mode = "active"
				protection.RuntimeStatus = "connected"
			}
			if protection.LastSync == nil || item.LastSeenAt.After(*protection.LastSync) {
				seen := item.LastSeenAt
				protection.LastSync = &seen
				protection.ModelVersion = item.ModelVersion
				protection.RulesetVersion = item.RulesetVersion
			}
		}
		for _, item := range snapshot.CheckIns {
			if item.UserID != userID || item.CreatedAt.Before(start) {
				continue
			}
			date := item.CreatedAt.UTC().Format("2006-01-02")
			moodPoints = append(moodPoints, model.MoodPoint{Date: date, Mood: item.Mood, Urge: item.Urge})
			activityDays[date] = struct{}{}
		}
		for _, item := range snapshot.JournalEntries {
			if item.UserID == userID {
				reflectionCount++
				if !item.CreatedAt.Before(start) {
					activityDays[item.CreatedAt.UTC().Format("2006-01-02")] = struct{}{}
				}
			}
		}
		for _, item := range snapshot.Missions {
			if item.UserID == userID && item.Date >= start.Format("2006-01-02") && (item.Mission1 || item.Mission2 || item.Mission3 || item.Mission4 || item.Mission5) {
				activityDays[item.Date] = struct{}{}
			}
		}
		for _, item := range snapshot.AggregateEvents {
			if item.UserID != userID || item.EventType != "block_count_sync" || item.EventDate.Before(start) {
				continue
			}
			index := int(startOfDay(item.EventDate).Sub(start).Hours() / 24)
			if index >= 0 && index < len(weeklyBlocks) {
				weeklyBlocks[index] += item.Count
				blockedAttempts += item.Count
			}
		}
	} else {
		dataState = "synced"
		devices, err := r.db.Device.Query().Where(device.UserID(userID)).All(ctx)
		if err != nil {
			return model.DashboardSummary{}, model.ProtectionStatus{}, model.ProgressSnapshot{}, err
		}
		for _, item := range devices {
			protection.DeviceCount++
			if item.ProtectionStatus == device.ProtectionStatusActive {
				protection.Mode = "active"
				protection.RuntimeStatus = "connected"
			}
			if item.LastSeenAt != nil && (protection.LastSync == nil || item.LastSeenAt.After(*protection.LastSync)) {
				seen := *item.LastSeenAt
				protection.LastSync = &seen
				protection.ModelVersion = value(item.ModelVersion)
				protection.RulesetVersion = value(item.RulesetVersion)
			}
		}
		events, err := r.db.AggregateEvent.Query().Where(
			aggregateevent.UserID(userID),
			aggregateevent.EventTypeEQ(aggregateevent.EventTypeBlockCountSync),
			aggregateevent.EventDateGTE(start),
		).All(ctx)
		if err != nil {
			return model.DashboardSummary{}, model.ProtectionStatus{}, model.ProgressSnapshot{}, err
		}
		for _, item := range events {
			index := int(startOfDay(item.EventDate).Sub(start).Hours() / 24)
			if index >= 0 && index < len(weeklyBlocks) {
				weeklyBlocks[index] += item.Count
				blockedAttempts += item.Count
			}
		}
		checkIns, err := r.db.CheckIn.Query().Where(checkin.UserID(userID), checkin.CreatedAtGTE(start)).All(ctx)
		if err != nil {
			return model.DashboardSummary{}, model.ProtectionStatus{}, model.ProgressSnapshot{}, err
		}
		for _, item := range checkIns {
			date := item.CreatedAt.UTC().Format("2006-01-02")
			moodPoints = append(moodPoints, model.MoodPoint{Date: date, Mood: item.MoodScore, Urge: item.UrgeScore})
			activityDays[date] = struct{}{}
		}
		reflections, err := r.db.Reflection.Query().Where(reflection.UserID(userID)).All(ctx)
		if err != nil {
			return model.DashboardSummary{}, model.ProtectionStatus{}, model.ProgressSnapshot{}, err
		}
		reflectionCount = len(reflections)
		for _, item := range reflections {
			if !item.CreatedAt.Before(start) {
				activityDays[item.CreatedAt.UTC().Format("2006-01-02")] = struct{}{}
			}
		}
	}

	sort.Slice(moodPoints, func(i, j int) bool { return moodPoints[i].Date < moodPoints[j].Date })
	streak := contiguousDays(activityDays, startOfDay(now.UTC()))
	summary := model.DashboardSummary{
		UserName: user.DisplayName, ProtectionLabel: protection.Mode,
		BlockedAttempts: blockedAttempts, ActiveDays: len(activityDays),
		CurrentStreak: streak, DataState: dataState,
	}
	progress := model.ProgressSnapshot{
		WeeklyBlocks: weeklyBlocks, MoodPoints: moodPoints,
		ActiveDays: len(activityDays), Reflections: reflectionCount, DataState: dataState,
	}
	return summary, protection, progress, nil
}

func startOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func contiguousDays(days map[string]struct{}, today time.Time) int {
	streak := 0
	for date := today; ; date = date.AddDate(0, 0, -1) {
		if _, ok := days[date.Format("2006-01-02")]; !ok {
			return streak
		}
		streak++
	}
}
