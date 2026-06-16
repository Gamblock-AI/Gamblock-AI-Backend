package model

import "time"

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	GroupCode string    `json:"group_code"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	Members   int       `json:"members"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrganizationMember struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	UserID         string     `json:"user_id"`
	UserName       string     `json:"user_name"`
	UserEmail      string     `json:"user_email"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	JoinedAt       *time.Time `json:"joined_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type OrganizationAnalytics struct {
	TotalMembers       int     `json:"total_members"`
	ActiveDevices      int     `json:"active_devices"`
	AvgMoodScore       float64 `json:"avg_mood_score"`
	TotalBlocks        int     `json:"total_blocks"`
	CompletedMissions  int     `json:"completed_missions"`
	PendingApprovals   int     `json:"pending_approvals"`
	WeeklyBlockTrend   []int   `json:"weekly_block_trend"`
}
