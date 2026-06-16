package model

import "time"

type DailyMission struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"`
	Mission1  bool      `json:"mission_1"`
	Mission2  bool      `json:"mission_2"`
	Mission3  bool      `json:"mission_3"`
	Mission4  bool      `json:"mission_4"`
	Mission5  bool      `json:"mission_5"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
