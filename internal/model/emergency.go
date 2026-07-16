package model

import "time"

type EmergencyKeyRequest struct {
	ID               string     `json:"id"`
	RequestedBy      string     `json:"requested_by"`
	ApprovedBy       string     `json:"approved_by,omitempty"`
	Status           string     `json:"status"`
	RequestExpiresAt time.Time  `json:"request_expires_at"`
	KeyExpiresAt     *time.Time `json:"key_expires_at,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	KeyHash          string     `json:"-"`
}
