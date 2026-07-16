package model

import "time"

type ApprovalRequest struct {
	ID                       string    `json:"id"`
	UserID                   string    `json:"-"`
	DeviceID                 string    `json:"-"`
	PartnerLinkID            string    `json:"-"`
	QuickTokenHash           string    `json:"-"`
	Action                   string    `json:"action"`
	ExpiresIn                string    `json:"expires_in"`
	Status                   string    `json:"status"`
	Reason                   string    `json:"reason"`
	RequestedDurationMinutes int       `json:"requested_duration_minutes"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
	ExpiresAt                time.Time `json:"expires_at"`
}
