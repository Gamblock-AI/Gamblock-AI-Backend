package model

import "time"

type ApprovalRequest struct {
	ID                       string     `json:"id"`
	UserID                   string     `json:"-"`
	DeviceID                 string     `json:"device_id"`
	PartnerLinkID            string     `json:"partner_link_id"`
	QuickTokenHash           string     `json:"-"`
	Action                   string     `json:"action"`
	ActionLabel              string     `json:"action_label"`
	ExpiresIn                string     `json:"expires_in"`
	Status                   string     `json:"status"`
	StatusLabel              string     `json:"status_label"`
	Reason                   string     `json:"reason"`
	RequestedDurationMinutes int        `json:"requested_duration_minutes"`
	ResolvedAt               *time.Time `json:"resolved_at,omitempty"`
	AppliedAt                *time.Time `json:"applied_at,omitempty"`
	GrantExpiresAt           *time.Time `json:"grant_expires_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
	ExpiresAt                time.Time  `json:"expires_at"`
}

type ApprovalGrant struct {
	RequestID      string    `json:"request_id"`
	DeviceID       string    `json:"device_id"`
	Action         string    `json:"action"`
	GrantStartsAt  time.Time `json:"grant_starts_at"`
	GrantExpiresAt time.Time `json:"grant_expires_at"`
}
