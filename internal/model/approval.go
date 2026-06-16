package model

import "time"

type ApprovalRequest struct {
	ID                       string    `json:"id"`
	Action                   string    `json:"action"`
	ExpiresIn                string    `json:"expires_in"`
	Status                   string    `json:"status"`
	Reason                   string    `json:"reason"`
	Duration                 int       `json:"requested_duration_minutes"`
	RequestedDurationMinutes int       `json:"requested_duration_minutes"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
