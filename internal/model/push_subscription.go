package model

import "time"

// PushSubscription is an opt-in Web Push delivery endpoint (RFC 8030). It
// contains only the subscription material needed to deliver notifications, not
// browsing data. The endpoint is unique per subscription; a user may hold
// several (one per installed browser).
type PushSubscription struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Endpoint  string     `json:"endpoint"`
	P256dh    string     `json:"p256dh"`
	AuthKey   string     `json:"auth_key"`
	UserAgent *string    `json:"-"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
