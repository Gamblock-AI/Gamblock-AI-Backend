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
	User                = model.User
	Device              = model.Device
	Partner             = model.Partner
	ApprovalRequest     = model.ApprovalRequest
	EducationModule     = model.EducationModule
	EducationMedia      = model.EducationMedia
	EducationProgress   = model.EducationProgress
	SupportCase         = model.SupportCase
	DataRequest         = model.DataRequest
	Organization        = model.Organization
	Release             = model.Release
	AuditEvent          = model.AuditEvent
	NotificationItem    = model.NotificationItem
	JournalEntry        = model.JournalEntry
	DailyMission        = model.DailyMission
	Intention           = model.Intention
	CheckIn             = model.CheckIn
	AggregateEvent      = model.AggregateEvent
	EmergencyKeyRequest = model.EmergencyKeyRequest
)

// Store is a concurrency-safe in-memory backing store. It is used as a
// privacy-safe fallback when PostgreSQL is unavailable and as a cache that is
// refreshed from ent by Repository.RefreshStore.
type Store struct {
	mu sync.RWMutex

	Users                []User                `json:"users"`
	Devices              []Device              `json:"devices"`
	Partners             []Partner             `json:"partners"`
	Approvals            []ApprovalRequest     `json:"approvals"`
	Modules              []EducationModule     `json:"modules"`
	EducationMedia       []EducationMedia      `json:"education_media"`
	EducationProgress    []EducationProgress   `json:"education_progress"`
	SupportCases         []SupportCase         `json:"support_cases"`
	DataRequests         []DataRequest         `json:"data_requests"`
	Organizations        []Organization        `json:"organizations"`
	ModelReleases        []Release             `json:"model_releases"`
	RulesetReleases      []Release             `json:"ruleset_releases"`
	NetworkRulesets      []Release             `json:"network_rulesets"`
	AuditEvents          []AuditEvent          `json:"audit_events"`
	NotificationEvents   []NotificationItem    `json:"notification_events"`
	JournalEntries       []JournalEntry        `json:"journal_entries"`
	Missions             []DailyMission        `json:"missions"`
	Intentions           []Intention           `json:"intentions"`
	CheckIns             []CheckIn             `json:"check_ins"`
	AggregateEvents      []AggregateEvent      `json:"aggregate_events"`
	EmergencyKeyRequests []EmergencyKeyRequest `json:"emergency_key_requests"`
}

func New() *Store {
	return &Store{}
}
