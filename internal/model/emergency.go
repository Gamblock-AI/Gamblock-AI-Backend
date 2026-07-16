package model

import "time"

type EmergencyKeyRequest struct {
	ID               string     `json:"id"`
	RequestedBy      string     `json:"requested_by"`
	DeviceID         string     `json:"device_id"`
	ReviewedBy       string     `json:"reviewed_by,omitempty"`
	ApprovedBy       string     `json:"approved_by,omitempty"`
	Status           string     `json:"status"`
	RequestExpiresAt time.Time  `json:"request_expires_at"`
	KeyExpiresAt     *time.Time `json:"key_expires_at,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	UsedAt           *time.Time `json:"used_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	KeyHash          string     `json:"-"`
}

type EmergencyGrant struct {
	RequestID      string    `json:"request_id"`
	DeviceID       string    `json:"device_id"`
	GrantStartsAt  time.Time `json:"grant_starts_at"`
	GrantExpiresAt time.Time `json:"grant_expires_at"`
}
