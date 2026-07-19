package model

import "time"

type AccountabilityGroup struct {
	ID             string    `json:"id"`
	OwnerPartnerID string    `json:"-"`
	OwnerName      string    `json:"owner_name"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	JoinCode       string    `json:"join_code,omitempty"`
	JoinCodeHash   string    `json:"-"`
	JoinCodeHint   string    `json:"join_code_hint"`
	Status         string    `json:"status"`
	MemberCount    int       `json:"member_count"`
	CodeRotatedAt  time.Time `json:"code_rotated_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SharingPreferences struct {
	ProtectionHealth   bool `json:"protection_health"`
	ProtectionActivity bool `json:"protection_activity"`
	RecoveryEngagement bool `json:"recovery_engagement"`
	EducationProgress  bool `json:"education_progress"`
}

type AccountabilityMembership struct {
	ID          string                 `json:"id"`
	GroupID     string                 `json:"group_id"`
	StudentID   string                 `json:"student_id"`
	StudentName string                 `json:"student_name"`
	StudentMail string                 `json:"student_email,omitempty"`
	Status      string                 `json:"status"`
	Sharing     SharingPreferences     `json:"sharing"`
	Aggregate   MemberAggregateSummary `json:"aggregate"`
	JoinedAt    time.Time              `json:"joined_at"`
	EndedAt     *time.Time             `json:"ended_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type MemberAggregateSummary struct {
	ProtectionStatus      string `json:"protection_status,omitempty"`
	ActiveDeviceCount     int    `json:"active_device_count,omitempty"`
	LastHeartbeatBucket   string `json:"last_heartbeat_bucket,omitempty"`
	WeeklyBlockCount      int    `json:"weekly_block_count,omitempty"`
	CheckInDays           int    `json:"check_in_days,omitempty"`
	MissionCompleted      int    `json:"mission_completed,omitempty"`
	EducationProgressBand string `json:"education_progress_band,omitempty"`
}

type MembershipExitRequest struct {
	ID           string     `json:"id"`
	MembershipID string     `json:"membership_id"`
	RequestedBy  string     `json:"requested_by"`
	Kind         string     `json:"kind"`
	Status       string     `json:"status"`
	Reason       string     `json:"reason,omitempty"`
	ReviewDueAt  *time.Time `json:"review_due_at,omitempty"`
	ResolvedBy   string     `json:"resolved_by,omitempty"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PartnerContactRequest struct {
	ID             string     `json:"id"`
	MembershipID   string     `json:"membership_id"`
	StudentID      string     `json:"student_id"`
	StudentName    string     `json:"student_name"`
	PartnerID      string     `json:"partner_id"`
	Category       string     `json:"category"`
	Message        string     `json:"message,omitempty"`
	Status         string     `json:"status"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	EscalatedAt    *time.Time `json:"escalated_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AccountabilityWorkspace struct {
	Role            string                     `json:"role"`
	Groups          []AccountabilityGroup      `json:"groups"`
	Membership      *AccountabilityMembership  `json:"membership,omitempty"`
	Members         []AccountabilityMembership `json:"members"`
	ExitRequests    []MembershipExitRequest    `json:"exit_requests"`
	ContactRequests []PartnerContactRequest    `json:"contact_requests"`
	PendingActions  int                        `json:"pending_actions"`
}
