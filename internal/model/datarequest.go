package model

import "time"

type DataRequest struct {
	ID                    string     `json:"id"`
	UserID                string     `json:"-"`
	Title                 string     `json:"title"`
	Type                  string     `json:"type"`
	Status                string     `json:"status"`
	ConfirmationTokenHash string     `json:"-"`
	ConfirmationExpiresAt *time.Time `json:"-"`
	ConfirmedAt           *time.Time `json:"confirmed_at,omitempty"`
	ResultPath            string     `json:"-"`
	ResultExpiresAt       *time.Time `json:"result_expires_at,omitempty"`
	FailureCode           string     `json:"failure_code,omitempty"`
	RetryCount            int        `json:"retry_count"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
