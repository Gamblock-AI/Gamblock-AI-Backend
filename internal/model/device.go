package model

import "time"

type Device struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	ClientInstanceID   string    `json:"client_instance_id,omitempty"`
	Platform           string    `json:"platform"`
	Label              string    `json:"label"`
	AppVersion         string    `json:"app_version"`
	OSVersion          string    `json:"os_version"`
	ModelVersion       string    `json:"model_version"`
	RulesetVersion     string    `json:"ruleset_version"`
	GrantPublicJWK     string    `json:"-"`
	GrantKeyThumbprint string    `json:"-"`
	ProtectionStatus   string    `json:"protection_status"`
	LastSeenAt         time.Time `json:"last_seen_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type DeviceGrantKeyChallenge struct {
	ChallengeToken string    `json:"challenge_token"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type DeviceGrantKeyBinding struct {
	DeviceID   string `json:"device_id"`
	Thumbprint string `json:"thumbprint"`
	Bound      bool   `json:"bound"`
}
