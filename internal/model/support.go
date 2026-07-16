package model

import "time"

type SupportCase struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	Title     string    `json:"title"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	Owner     string    `json:"owner,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
