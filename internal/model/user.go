package model

import "time"

type User struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	DisplayName        string     `json:"display_name"`
	AvatarURL          *string    `json:"avatar_url,omitempty"`
	Role               string     `json:"role"`
	MustChangePassword bool       `json:"-"`
	EmailVerifiedAt    *time.Time `json:"email_verified_at,omitempty"`
	PhoneE164          string     `json:"phone_e164,omitempty"`
	PhoneVerifiedAt    *time.Time `json:"phone_verified_at,omitempty"`
	ExperiencePoints   int        `json:"-"`
	PasswordHash       string     `json:"-"`
	GoogleSubject      string     `json:"-"`
	DisabledAt         *time.Time `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
