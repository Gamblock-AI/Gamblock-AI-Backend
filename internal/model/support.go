package model

import "time"

type SupportCase struct {
	ID             string           `json:"id"`
	UserID         string           `json:"user_id,omitempty"`
	UserName       string           `json:"user_name,omitempty"`
	UserAvatarURL  *string          `json:"user_avatar_url,omitempty"`
	Title          string           `json:"title"`
	Type           string           `json:"type"`
	Status         string           `json:"status"`
	Priority       string           `json:"priority"`
	Owner          string           `json:"owner,omitempty"`
	Impact         string           `json:"impact"`
	Messages       []SupportMessage `json:"messages,omitempty"`
	UnreadCount    int              `json:"unread_count"`
	ResolvedAt     *time.Time       `json:"resolved_at,omitempty"`
	ClosedAt       *time.Time       `json:"closed_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type SupportMessage struct {
	ID              string     `json:"id"`
	SupportCaseID   string     `json:"support_case_id"`
	AuthorID        string     `json:"-"`
	AuthorRole      string     `json:"author_role"`
	AuthorName      string     `json:"author_name,omitempty"`
	AuthorAvatarURL *string    `json:"author_avatar_url,omitempty"`
	Content         string     `json:"content"`
	ReadAt          *time.Time `json:"read_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}
