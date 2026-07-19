package model

import "time"

type SiteSocialLink struct {
	ID        string    `json:"id"`
	Platform  string    `json:"platform"`
	Label     string    `json:"label"`
	URL       *string   `json:"url"`
	Enabled   bool      `json:"enabled"`
	SortOrder int       `json:"sort_order"`
	UpdatedBy string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OperatorInvitation struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	Status     string     `json:"status"`
	InvitedBy  string     `json:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	TokenHash  string     `json:"-"`
}

type AdminAccount struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	DisplayName        string     `json:"display_name"`
	Role               string     `json:"role"`
	EmailVerifiedAt    *time.Time `json:"email_verified_at,omitempty"`
	DisabledAt         *time.Time `json:"disabled_at,omitempty"`
	MustChangePassword bool       `json:"must_change_password"`
	CreatedAt          time.Time  `json:"created_at"`
}

type EducationRevision struct {
	ID        string            `json:"id"`
	ModuleID  string            `json:"module_id"`
	Revision  int               `json:"revision"`
	Document  EducationDocument `json:"document"`
	Slug      string            `json:"slug"`
	Kind      string            `json:"kind"`
	CreatedBy string            `json:"created_by"`
	CreatedAt time.Time         `json:"created_at"`
}

type ReleaseRollout struct {
	ID                   string    `json:"id"`
	Kind                 string    `json:"kind"`
	ReleaseID            string    `json:"release_id"`
	ReleaseVersion       string    `json:"release_version"`
	Status               string    `json:"status"`
	Platform             string    `json:"platform"`
	Percentage           int       `json:"percentage"`
	AppVersionConstraint string    `json:"app_version_constraint,omitempty"`
	CreatedBy            string    `json:"created_by"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type AdminOverview struct {
	Role               string `json:"role"`
	DraftContent       int    `json:"draft_content,omitempty"`
	ReviewContent      int    `json:"review_content,omitempty"`
	OpenSupport        int    `json:"open_support,omitempty"`
	UnassignedSupport  int    `json:"unassigned_support,omitempty"`
	FailedDataRequests int    `json:"failed_data_requests,omitempty"`
	ValidatedReleases  int    `json:"validated_releases,omitempty"`
	ActiveRollouts     int    `json:"active_rollouts,omitempty"`
	PendingEmergency   int    `json:"pending_emergency,omitempty"`
	ActiveOperators    int    `json:"active_operators,omitempty"`
	VisibleSocialLinks int    `json:"visible_social_links,omitempty"`
}
