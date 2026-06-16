package model

import "time"

type Device struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Platform         string    `json:"platform"`
	Label            string    `json:"label"`
	AppVersion       string    `json:"app_version"`
	OSVersion        string    `json:"os_version"`
	ModelVersion     string    `json:"model_version"`
	RulesetVersion   string    `json:"ruleset_version"`
	ProtectionStatus string    `json:"protection_status"`
	LastSeenAt       time.Time `json:"last_seen_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
