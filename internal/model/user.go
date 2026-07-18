package model

import "time"

type User struct {
	ID               string     `json:"id"`
	Email            string     `json:"email"`
	DisplayName      string     `json:"display_name"`
	Role             string     `json:"role"`
	ExperiencePoints int        `json:"-"`
	PasswordHash     string     `json:"-"`
	GoogleSubject    string     `json:"-"`
	DisabledAt       *time.Time `json:"-"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
