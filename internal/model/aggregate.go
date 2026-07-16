package model

import "time"

type AggregateEvent struct {
	ID             string    `json:"id"`
	UserID         string    `json:"-"`
	DeviceID       string    `json:"device_id,omitempty"`
	IdempotencyKey string    `json:"-"`
	EventType      string    `json:"event_type"`
	EventDate      time.Time `json:"event_date"`
	Count          int       `json:"count"`
	CreatedAt      time.Time `json:"created_at"`
}
