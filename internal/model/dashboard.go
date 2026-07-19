package model

import "time"

type DashboardSummary struct {
	UserName        string `json:"user_name"`
	ProtectionLabel string `json:"protection_label"`
	BlockedAttempts int    `json:"blocked_attempts"`
	ActiveDays      int    `json:"active_days"`
	CurrentStreak   int    `json:"current_streak"`
	DataState       string `json:"data_state"`
}

type ProtectionStatus struct {
	Mode           string     `json:"mode"`
	RuntimeStatus  string     `json:"runtime_status"`
	RulesetVersion string     `json:"ruleset_version,omitempty"`
	ModelVersion   string     `json:"model_version,omitempty"`
	LastSync       *time.Time `json:"last_sync,omitempty"`
	DeviceCount    int        `json:"device_count"`
}

type MoodPoint struct {
	Date string `json:"date"`
	Mood int    `json:"mood"`
	Urge int    `json:"urge"`
}

type ProgressActivityDay struct {
	Date      string `json:"date"`
	CheckIns  int    `json:"check_ins"`
	Practices int    `json:"practices"`
	Journals  int    `json:"journals"`
	Missions  int    `json:"missions"`
	Education int    `json:"education"`
	Reviews   int    `json:"reviews"`
}

type ProgressSnapshot struct {
	WeeklyBlocks   []int                 `json:"weekly_blocks"`
	RangeDays      int                   `json:"range_days"`
	DailyBlocks    []int                 `json:"daily_blocks"`
	MoodPoints     []MoodPoint           `json:"mood_points"`
	CheckInCount   int                   `json:"check_in_count"`
	TrendAvailable bool                  `json:"trend_available"`
	ActiveDays     int                   `json:"active_days"`
	Reflections    int                   `json:"reflections"`
	DataState      string                `json:"data_state"`
	ActivityDays   []ProgressActivityDay `json:"activity_days"`
}
