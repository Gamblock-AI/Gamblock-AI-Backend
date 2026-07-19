package model

import "time"

type Intention struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Text      string    `json:"intention_text"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID               string    `json:"id,omitempty"`
	WeekStart        string    `json:"week_start"`
	WhatHelped       []string  `json:"what_helped"`
	WhatWasHard      []string  `json:"what_was_hard"`
	Adjustment       string    `json:"adjustment"`
	NextMission      string    `json:"next_mission"`
	RecommendedSkill string    `json:"recommended_skill,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

type RecoveryUnlockEvidence struct {
	PracticeKinds   map[string]bool
	HasFocusJournal bool
	HasWeeklyReview bool
	ActiveDays      int
}
