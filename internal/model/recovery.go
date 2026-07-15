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
