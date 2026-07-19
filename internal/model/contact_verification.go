package model

import "time"

type ContactVerification struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Kind         string     `json:"kind"`
	Destination  string     `json:"destination"`
	TokenHash    string     `json:"-"`
	AttemptCount int        `json:"-"`
	ExpiresAt    time.Time  `json:"expires_at"`
	ConsumedAt   *time.Time `json:"consumed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
