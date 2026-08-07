package model

import "time"

type Intention struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Text           string    `json:"intention_text"`
	Status         string    `json:"status"`
	SchoolImpact   string    `json:"school_impact,omitempty"`
	MoneySpent     string    `json:"money_spent,omitempty"`
	ScreenTime     string    `json:"screen_time,omitempty"`
	QuitAttempts   string    `json:"quit_attempts,omitempty"`
	QuitMotivation string    `json:"quit_motivation,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CheckIn struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Mood      int       `json:"mood_score"`
	Urge      int       `json:"urge_score"`
	Context   string    `json:"context_text,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RecoveryRecord struct {
	ID         string         `json:"id"`
	UserID     string         `json:"-"`
	Kind       string         `json:"kind"`
	RecordDate string         `json:"record_date"`
	Metadata   map[string]any `json:"metadata"`
	Content    string         `json:"content,omitempty"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type RecoveryPracticeSession struct {
	ID              string    `json:"id"`
	UserID          string    `json:"-"`
	PracticeKind    string    `json:"practice_kind"`
	DurationSeconds int       `json:"duration_seconds"`
	Feedback        string    `json:"feedback,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
	CreatedAt       time.Time `json:"created_at"`
	// Response-only reward fields; never persisted.
	ExpAwarded int                 `json:"exp_awarded,omitempty"`
	Experience *ExperienceProgress `json:"experience,omitempty"`
}

type RecoverySpace struct {
	ID                string         `json:"id"`
	UserID            string         `json:"-"`
	Theme             string         `json:"theme"`
	UnlockedItems     []string       `json:"unlocked_items"`
	PlacedItems       map[string]any `json:"placed_items"`
	UnlockRuleVersion int            `json:"unlock_rule_version"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type WeeklyReview struct {
	ID               string   `json:"id,omitempty"`
	WeekStart        string   `json:"week_start"`
	WhatHelped       []string `json:"what_helped"`
	WhatWasHard      []string `json:"what_was_hard"`
	Adjustment       string   `json:"adjustment"`
	NextMission      string   `json:"next_mission"`
	RecommendedSkill string   `json:"recommended_skill,omitempty"`
	// These normalized fields are the website's privacy-safe structured
	// weekly-review contract. Legacy fields above remain readable for existing
	// records and older clients.
	IntentionID       string    `json:"intention_id,omitempty"`
	Outcome           string    `json:"outcome,omitempty"`
	HelpfulAction     string    `json:"helpful_action,omitempty"`
	NextMissionNumber int       `json:"next_mission_number,omitempty"`
	SelectedSkillID   string    `json:"selected_skill_id,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

type WeeklyReviewSaveResult struct {
	Review     WeeklyReview       `json:"review"`
	EXPGranted bool               `json:"exp_granted"`
	CapReached bool               `json:"cap_reached"`
	Experience ExperienceProgress `json:"experience"`
}

type RecoveryUnlockEvidence struct {
	PracticeKinds    map[string]bool
	HasFocusJournal  bool
	HasWeeklyReview  bool
	ActiveDays       int
	TotalPractices   int
	FocusJournals    int
	WeeklyReviews    int
	MissionsClaimed  int
	ExperiencePoints int
}
