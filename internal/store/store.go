// Package store is the in-memory backing store for the Gamblock-AI backend.
//
// It intentionally holds model.* domain types only — there is no parallel set
// of store-owned structs. The types below are aliases of the domain types in
// internal/model, so store.X and model.X are the SAME type. This keeps a single
// source of truth for domain shapes (clean architecture: the in-memory store is
// a data-access implementation detail, not a separate domain).
package store

import (
	"sync"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// Domain type aliases (single source of truth: internal/model).
type (
	User                     = model.User
	ContactVerification      = model.ContactVerification
	Device                   = model.Device
	Partner                  = model.Partner
	AccountabilityGroup      = model.AccountabilityGroup
	SharingPreferences       = model.SharingPreferences
	AccountabilityMembership = model.AccountabilityMembership
	MembershipExitRequest    = model.MembershipExitRequest
	PartnerContactRequest    = model.PartnerContactRequest
	ApprovalRequest          = model.ApprovalRequest
	EducationModule          = model.EducationModule
	EducationMedia           = model.EducationMedia
	EducationProgress        = model.EducationProgress
	EducationRevision        = model.EducationRevision
	SupportCase              = model.SupportCase
	SupportMessage           = model.SupportMessage
	DataRequest              = model.DataRequest
	Organization             = model.Organization
	Release                  = model.Release
	AuditEvent               = model.AuditEvent
	NotificationItem         = model.NotificationItem
	JournalEntry             = model.JournalEntry
	DailyMission             = model.DailyMission
	Intention                = model.Intention
	CheckIn                  = model.CheckIn
	RecoveryRecord           = model.RecoveryRecord
	RecoveryPracticeSession  = model.RecoveryPracticeSession
	RecoverySpace            = model.RecoverySpace
	AggregateEvent           = model.AggregateEvent
	EmergencyKeyRequest      = model.EmergencyKeyRequest
	SiteSocialLink           = model.SiteSocialLink
	OperatorInvitation       = model.OperatorInvitation
	ReleaseRollout           = model.ReleaseRollout
)

// Store is a concurrency-safe in-memory backing store. It is used as a
// privacy-safe fallback when PostgreSQL is unavailable and as a cache that is
// refreshed from ent by Repository.RefreshStore.
type Store struct {
	mu sync.RWMutex

	Users                     []User                     `json:"users"`
	ContactVerifications      []ContactVerification      `json:"contact_verifications"`
	Devices                   []Device                   `json:"devices"`
	Partners                  []Partner                  `json:"partners"`
	AccountabilityGroups      []AccountabilityGroup      `json:"accountability_groups"`
	AccountabilityMemberships []AccountabilityMembership `json:"accountability_memberships"`
	MembershipExitRequests    []MembershipExitRequest    `json:"membership_exit_requests"`
	PartnerContactRequests    []PartnerContactRequest    `json:"partner_contact_requests"`
	Approvals                 []ApprovalRequest          `json:"approvals"`
	Modules                   []EducationModule          `json:"modules"`
	EducationMedia            []EducationMedia           `json:"education_media"`
	EducationProgress         []EducationProgress        `json:"education_progress"`
	EducationRevisions        []EducationRevision        `json:"education_revisions"`
	SupportCases              []SupportCase              `json:"support_cases"`
	SupportMessages           []SupportMessage           `json:"support_messages"`
	DataRequests              []DataRequest              `json:"data_requests"`
	Organizations             []Organization             `json:"organizations"`
	ModelReleases             []Release                  `json:"model_releases"`
	RulesetReleases           []Release                  `json:"ruleset_releases"`
	NetworkRulesets           []Release                  `json:"network_rulesets"`
	AuditEvents               []AuditEvent               `json:"audit_events"`
	NotificationEvents        []NotificationItem         `json:"notification_events"`
	JournalEntries            []JournalEntry             `json:"journal_entries"`
	Missions                  []DailyMission             `json:"missions"`
	Intentions                []Intention                `json:"intentions"`
	CheckIns                  []CheckIn                  `json:"check_ins"`
	RecoveryRecords           []RecoveryRecord           `json:"recovery_records"`
	RecoveryPracticeSessions  []RecoveryPracticeSession  `json:"recovery_practice_sessions"`
	RecoverySpaces            []RecoverySpace            `json:"recovery_spaces"`
	AggregateEvents           []AggregateEvent           `json:"aggregate_events"`
	EmergencyKeyRequests      []EmergencyKeyRequest      `json:"emergency_key_requests"`
	SiteSocialLinks           []SiteSocialLink           `json:"site_social_links"`
	OperatorInvitations       []OperatorInvitation       `json:"operator_invitations"`
	ReleaseRollouts           []ReleaseRollout           `json:"release_rollouts"`
}

func New() *Store {
	return &Store{}
}
