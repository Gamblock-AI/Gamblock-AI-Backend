package model

import "time"

// ReminderPreference holds the single opt-in daily reminder setting synced
// across the web, Android, and Windows surfaces. It stores only non-sensitive
// delivery scheduling data (enabled flag, local time, IANA timezone).
type ReminderPreference struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Enabled     bool       `json:"enabled"`
	LocalTime   string     `json:"local_time"`
	Timezone    string     `json:"timezone"`
	Locale      string     `json:"locale"`
	LastFiredAt *time.Time `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
