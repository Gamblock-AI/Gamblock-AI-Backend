package model

import "time"

type AuditEvent struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	Actor      string         `json:"actor"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type"`
	Target     string         `json:"target"`
	Reason     string         `json:"reason"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
