package repository

import (
	"context"
	"fmt"
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

func (r *Repository) GetProgressData(ctx context.Context, userID string, days int, now time.Time) (model.ProgressSnapshot, error) {
	if days != 7 && days != 30 && days != 90 {
		return model.ProgressSnapshot{}, fmt.Errorf("progress range must be 7, 30, or 90 days")
	}
	if r.db != nil {
		r.RefreshStore(ctx)
	}
	start := startOfDay(now.UTC()).AddDate(0, 0, -(days - 1))
	dailyBlocks := make([]int, days)
	moodPoints := []model.MoodPoint{}
	activityDays := map[string]struct{}{}
	activityByDate := map[string]*model.ProgressActivityDay{}
	reflections := 0
	snapshot := r.store.Snapshot()
	for _, event := range snapshot.AggregateEvents {
		if event.UserID != userID || event.EventType != "block_count_sync" || event.EventDate.Before(start) {
			continue
		}
		index := int(startOfDay(event.EventDate).Sub(start).Hours() / 24)
		if index >= 0 && index < days {
			dailyBlocks[index] += event.Count
		}
	}
	for _, checkIn := range snapshot.CheckIns {
		if checkIn.UserID != userID || checkIn.CreatedAt.Before(start) {
			continue
		}
		date := checkIn.CreatedAt.UTC().Format("2006-01-02")
		moodPoints = append(moodPoints, model.MoodPoint{Date: date, Mood: checkIn.Mood, Urge: checkIn.Urge})
		activityDays[date] = struct{}{}
		activityForDate(activityByDate, date).CheckIns++
	}
	for _, reflection := range snapshot.JournalEntries {
		if reflection.UserID == userID && !reflection.CreatedAt.Before(start) {
			reflections++
			date := reflection.CreatedAt.UTC().Format("2006-01-02")
			activityDays[date] = struct{}{}
			activityForDate(activityByDate, date).Journals++
		}
	}
	for _, practice := range snapshot.RecoveryPracticeSessions {
		if practice.UserID == userID && !practice.CompletedAt.Before(start) {
			date := practice.CompletedAt.UTC().Format("2006-01-02")
			activityDays[date] = struct{}{}
			activityForDate(activityByDate, date).Practices++
		}
	}
	for _, mission := range snapshot.Missions {
		if mission.UserID != userID || mission.Date < start.Format("2006-01-02") {
			continue
		}
		count := 0
		for _, completed := range []bool{mission.Mission1, mission.Mission2, mission.Mission3, mission.Mission4, mission.Mission5} {
			if completed {
				count++
			}
		}
		if count > 0 {
			activityDays[mission.Date] = struct{}{}
			activityForDate(activityByDate, mission.Date).Missions += count
		}
	}
	for _, education := range snapshot.EducationProgress {
		if education.UserID == userID && !education.UpdatedAt.Before(start) {
			date := education.UpdatedAt.UTC().Format("2006-01-02")
			activityDays[date] = struct{}{}
			activityForDate(activityByDate, date).Education++
		}
	}
	for _, record := range snapshot.RecoveryRecords {
		if record.UserID == userID && record.Kind == "weekly_review" && record.RecordDate >= start.Format("2006-01-02") {
			activityDays[record.RecordDate] = struct{}{}
			activityForDate(activityByDate, record.RecordDate).Reviews++
		}
	}
	sort.Slice(moodPoints, func(i, j int) bool { return moodPoints[i].Date < moodPoints[j].Date })
	weekly := dailyBlocks
	if len(dailyBlocks) > 7 {
		weekly = dailyBlocks[len(dailyBlocks)-7:]
	}
	activityList := make([]model.ProgressActivityDay, 0, len(activityByDate))
	for _, item := range activityByDate {
		activityList = append(activityList, *item)
	}
	sort.Slice(activityList, func(i, j int) bool { return activityList[i].Date < activityList[j].Date })
	return model.ProgressSnapshot{
		WeeklyBlocks: weekly, RangeDays: days, DailyBlocks: dailyBlocks,
		MoodPoints: moodPoints, CheckInCount: len(moodPoints), TrendAvailable: len(moodPoints) >= 3,
		ActiveDays: len(activityDays), Reflections: reflections, DataState: "synced", ActivityDays: activityList,
	}, nil
}

func activityForDate(items map[string]*model.ProgressActivityDay, date string) *model.ProgressActivityDay {
	if items[date] == nil {
		items[date] = &model.ProgressActivityDay{Date: date}
	}
	return items[date]
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
