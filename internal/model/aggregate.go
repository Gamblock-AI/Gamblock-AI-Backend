package model

import "time"

type AggregateEvent struct {
	ID             string         `json:"id"`
	UserID         string         `json:"-"`
	DeviceID       string         `json:"device_id,omitempty"`
	IdempotencyKey string         `json:"-"`
	EventType      string         `json:"event_type"`
	EventDate      time.Time      `json:"event_date"`
	Count          int            `json:"count"`
	MetadataJSON   map[string]any `json:"metadata_json,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// Hourly counts attached to a daily aggregate event. The 24-element slice is a
// per-hour histogram of the event type recorded in device-local time; it is an
// aggregate count per hour and never contains URLs, domains, or browsing data.
type HourlyDistribution struct {
	Hourly []int `json:"hourly"`
}
