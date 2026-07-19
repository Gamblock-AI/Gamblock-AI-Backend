package model

import "time"

type JournalEntry struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Text           string    `json:"text"`
	Mood           string    `json:"mood,omitempty"`
	MoodScore      *int      `json:"mood_score,omitempty"`
	NextStep       string    `json:"next_step,omitempty"`
	Status         string    `json:"status"`
	IsFocus        bool      `json:"is_focus"`
	PayloadVersion int       `json:"payload_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ReflectionUpdate struct {
	Text      *string `json:"text"`
	MoodScore *int    `json:"mood_score"`
	NextStep  *string `json:"next_step"`
	Status    *string `json:"status"`
	IsFocus   *bool   `json:"is_focus"`
}
