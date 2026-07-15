// Package store is the in-memory backing store for the Gamblock-AI backend.
//
// It intentionally holds model.* domain types only — there is no parallel set
// of store-owned structs. The types below are aliases of the domain types in
// internal/model, so store.X and model.X are the SAME type. This keeps a single
// source of truth for domain shapes (clean architecture: the in-memory store is
// a data-access implementation detail, not a separate domain).
package store

import (
	"strings"
	"sync"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// Domain type aliases (single source of truth: internal/model).
type (
	User             = model.User
	Device           = model.Device
	Partner          = model.Partner
	ApprovalRequest  = model.ApprovalRequest
	EducationModule  = model.EducationModule
	SupportCase      = model.SupportCase
	DataRequest      = model.DataRequest
	Organization     = model.Organization
	Release          = model.Release
	AuditEvent       = model.AuditEvent
	NotificationItem = model.NotificationItem
	JournalEntry     = model.JournalEntry
	DailyMission     = model.DailyMission
	Intention        = model.Intention
	CheckIn          = model.CheckIn
)

// Store is a concurrency-safe in-memory backing store. It is used as a
// privacy-safe fallback when PostgreSQL is unavailable and as a cache that is
// refreshed from ent by Repository.RefreshStore.
type Store struct {
	mu sync.RWMutex

	Users              []User             `json:"users"`
	Devices            []Device           `json:"devices"`
	Partners           []Partner          `json:"partners"`
	Approvals          []ApprovalRequest  `json:"approvals"`
	Modules            []EducationModule  `json:"modules"`
	SupportCases       []SupportCase      `json:"support_cases"`
	DataRequests       []DataRequest      `json:"data_requests"`
	Organizations      []Organization     `json:"organizations"`
	ModelReleases      []Release          `json:"model_releases"`
	RulesetReleases    []Release          `json:"ruleset_releases"`
	NetworkRulesets    []Release          `json:"network_rulesets"`
	AuditEvents        []AuditEvent       `json:"audit_events"`
	NotificationEvents []NotificationItem `json:"notification_events"`
	JournalEntries     []JournalEntry     `json:"journal_entries"`
	Missions           []DailyMission     `json:"missions"`
	Intentions         []Intention        `json:"intentions"`
	CheckIns           []CheckIn          `json:"check_ins"`
}

func NewSeeded() *Store {
	now := time.Now().UTC()
	return &Store{
		Users: []User{
			{ID: "usr_gading", Email: "gading@gmail.com", DisplayName: "Gading", Role: "user", CreatedAt: now, UpdatedAt: now},
			{ID: "usr_dery", Email: "dery@gmail.com", DisplayName: "Dery", Role: "user", CreatedAt: now, UpdatedAt: now},
			{ID: "usr_suci", Email: "suci@gmail.com", DisplayName: "Suci", Role: "partner", CreatedAt: now, UpdatedAt: now},
			{ID: "usr_nasywa", Email: "nasywa@gmail.com", DisplayName: "Nasywa", Role: "platform_admin", CreatedAt: now, UpdatedAt: now},
		},
		Devices: []Device{
			{ID: "dev_android", UserID: "usr_gading", Platform: "android", Label: "Gading Android", AppVersion: "1.0.0", OSVersion: "Android 15", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "active", LastSeenAt: now.Add(-2 * time.Minute), CreatedAt: now, UpdatedAt: now},
			{ID: "dev_windows", UserID: "usr_gading", Platform: "windows", Label: "Gading Windows", AppVersion: "1.0.0", OSVersion: "Windows 11", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "degraded", LastSeenAt: now.Add(-38 * time.Minute), CreatedAt: now, UpdatedAt: now},
		},
		Partners: []Partner{{ID: "pl_active", Name: "Suci", Contact: "suci@gmail.com | +62 812-0000-0000", Status: "Active partner", PartnerEmail: "suci@gmail.com", CreatedAt: now, UpdatedAt: now}},
		Approvals: []ApprovalRequest{
			{ID: "APR-2401", Action: "Pause protection for 15 minutes", ExpiresIn: "Expires in 23 minutes", Status: "Pending partner approval", Reason: "Troubleshooting app setup", RequestedDurationMinutes: 15, CreatedAt: now.Add(-7 * time.Minute), UpdatedAt: now.Add(-7 * time.Minute)},
			{ID: "APR-2398", Action: "Permission revoked detected", ExpiresIn: "Reviewed yesterday", Status: "Partner notified", Reason: "Accessibility service disabled", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour)},
		},
		Modules: []EducationModule{
			{ID: "mod_pause", Slug: "pause-before-impulse", Title: "Pause before impulse", Summary: "A short exercise to identify triggers and choose one safer action.", BodyMarkdown: "## Pause\n\nName the impulse, breathe for ten seconds, and choose one safe next action.", EstimatedMinutes: 4, Progress: 0.7, Status: "published", CreatedAt: now, UpdatedAt: now},
			{ID: "mod_finance", Slug: "financial-reality-check", Title: "Financial reality check", Summary: "A simple reflection on losses, debt risk, and recovery support.", BodyMarkdown: "## Reality check\n\nWrite down the amount at risk and contact your accountability partner.", EstimatedMinutes: 6, Progress: 0.35, Status: "published", CreatedAt: now, UpdatedAt: now},
			{ID: "mod_support", Slug: "ask-for-support", Title: "Ask for support", Summary: "How to talk with your partner without shame or blame.", BodyMarkdown: "## Ask\n\nUse a short, concrete message and state the help you need now.", EstimatedMinutes: 5, Progress: 0, Status: "published", CreatedAt: now, UpdatedAt: now},
		},
		SupportCases: []SupportCase{
			{ID: "CASE-1087", Title: "Setup and permissions", Type: "device_recovery", Status: "waiting_user", Priority: "normal", Owner: "Gading", CreatedAt: now, UpdatedAt: now},
			{ID: "CASE-1084", Title: "Partner approval issue", Type: "stuck_approval", Status: "open", Priority: "high", Owner: "Suci", CreatedAt: now, UpdatedAt: now},
		},
		DataRequests: []DataRequest{
			{ID: "DR-1042", Title: "Export account data", Type: "export", Status: "ready", CreatedAt: now, UpdatedAt: now},
			{ID: "DR-1035", Title: "Delete archived support notes", Type: "delete", Status: "review", CreatedAt: now, UpdatedAt: now},
		},
		Organizations:   []Organization{{ID: "org_community", Name: "Gamblock Community Pilot", Slug: "community-pilot", Status: "active", Members: 128, CreatedAt: now, UpdatedAt: now}},
		ModelReleases:   []Release{{ID: "rel_model_031", Version: "artifact-v0.3.1", Platform: "all", SHA256: "c3a12b939f2923c21d3e729a514610a3989cab321c895c9d2f63ac8eb8a0199c", Status: "published", DownloadURL: "/v1/releases/model/artifact-v0.3.1/download", Metrics: map[string]any{"false_positive_reviewed": true, "latency_ms_p95": 42}, PublishedAtText: "Published 2 days ago", CreatedAt: now, UpdatedAt: now}},
		RulesetReleases: []Release{{ID: "rel_rules_202605", Version: "ruleset-2026.05.1", Platform: "all", SHA256: "c9a31a473ca232c060d49d431e6a1029670df9c4888f32dbc4d06554a41bf586", Status: "published", DownloadURL: "/v1/releases/ruleset/ruleset-2026.05.1/download", Metrics: map[string]any{"rules": 42}, PublishedAtText: "Published today", CreatedAt: now, UpdatedAt: now}},
		NetworkRulesets: []Release{{ID: "rel_net_12", Version: "global-risk-category-v12", Platform: "all", SHA256: "e6091a4405a8db35789ea9197f2b44d46d16d425ee8b805c35c8f4b7f5d76127", Status: "validated", DownloadURL: "/v1/releases/network-rulesets/global-risk-category-v12/download", Metrics: map[string]any{"domains": 0, "privacy": "metadata_only"}, PublishedAtText: "Validated today", CreatedAt: now, UpdatedAt: now}},
		AuditEvents: []AuditEvent{
			{ID: "audit_1", Actor: "nasywa@gmail.com", Action: "Published content module", Target: "pause-before-impulse", CreatedAt: now, UpdatedAt: now},
			{ID: "audit_2", Actor: "nasywa@gmail.com", Action: "Staged model artifact", Target: "artifact-v0.4.0-rc1", CreatedAt: now, UpdatedAt: now},
		},
		NotificationEvents: []NotificationItem{{ID: "ntf_1", Channel: "email", Recipient: "suci@gmail.com", Status: "sent", Reason: "approval_request", CreatedAt: now, UpdatedAt: now}},
		JournalEntries: []JournalEntry{
			{ID: "ref_1", UserID: "usr_gading", Text: "Hari ini saya berhasil menahan diri dari godaan untuk membuka aplikasi berkat Pattern Interrupt.", Mood: "😊 Baik", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour)},
			{ID: "ref_2", UserID: "usr_gading", Text: "Merasa cemas di sore hari karena bosan, tapi berhasil mengalihkan perhatian dengan jalan kaki dan membaca modul kesadaran.", Mood: "😟 Cemas", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)},
		},
		Missions: []DailyMission{
			{ID: "mis_001", UserID: "usr_gading", Date: now.Format("2006-01-02"), Mission1: true, Mission2: true, Mission3: false, Mission4: false, Mission5: false, CreatedAt: now, UpdatedAt: now},
		},
		Intentions: []Intention{
			{ID: "int_1", UserID: "usr_gading", Text: "Saya ingin menyelesaikan kuliah dengan pikiran yang lebih tenang.", Status: "active", CreatedAt: now, UpdatedAt: now},
		},
		CheckIns: []CheckIn{
			{ID: "chk_1", UserID: "usr_gading", Mood: 4, Urge: 2, Context: "Merasa cukup tenang pagi ini.", CreatedAt: now},
		},
	}
}

func (s *Store) UserByEmail(email string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.Users {
		if strings.EqualFold(user.Email, email) {
			return user, true
		}
	}
	return User{}, false
}

func (s *Store) DefaultUser() User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Users[0]
}

func (s *Store) Snapshot() Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Store{
		Users:              append([]User(nil), s.Users...),
		Devices:            append([]Device(nil), s.Devices...),
		Partners:           append([]Partner(nil), s.Partners...),
		Approvals:          append([]ApprovalRequest(nil), s.Approvals...),
		Modules:            append([]EducationModule(nil), s.Modules...),
		SupportCases:       append([]SupportCase(nil), s.SupportCases...),
		DataRequests:       append([]DataRequest(nil), s.DataRequests...),
		Organizations:      append([]Organization(nil), s.Organizations...),
		ModelReleases:      append([]Release(nil), s.ModelReleases...),
		RulesetReleases:    append([]Release(nil), s.RulesetReleases...),
		NetworkRulesets:    append([]Release(nil), s.NetworkRulesets...),
		AuditEvents:        append([]AuditEvent(nil), s.AuditEvents...),
		NotificationEvents: append([]NotificationItem(nil), s.NotificationEvents...),
		JournalEntries:     append([]JournalEntry(nil), s.JournalEntries...),
		Missions:           append([]DailyMission(nil), s.Missions...),
		Intentions:         append([]Intention(nil), s.Intentions...),
		CheckIns:           append([]CheckIn(nil), s.CheckIns...),
	}
}

func (s *Store) Lock() {
	s.mu.Lock()
}

func (s *Store) Unlock() {
	s.mu.Unlock()
}

func (s *Store) RLock() {
	s.mu.RLock()
}

func (s *Store) RUnlock() {
	s.mu.RUnlock()
}

type tokenMapping map[string]ApprovalRequest

func (s *Store) tokenToRequest() tokenMapping {
	return nil
}

var globalTokenMap = make(map[string]ApprovalRequest)
var globalTokenMu sync.RWMutex

func (s *Store) SetTokenMapping(tokenHash string, req ApprovalRequest) {
	globalTokenMu.Lock()
	defer globalTokenMu.Unlock()
	globalTokenMap[tokenHash] = req
}

func (s *Store) GetTokenMapping(tokenHash string) (ApprovalRequest, bool) {
	globalTokenMu.RLock()
	defer globalTokenMu.RUnlock()
	req, ok := globalTokenMap[tokenHash]
	return req, ok
}


// RefreshTokenRecord is the in-memory backing for refresh tokens (used when the
// DB is unavailable). Mirrors the ent RefreshToken fields used by the auth flow.
type RefreshTokenRecord struct {
	ID        string
	UserID    string
	TokenHash string
	DeviceID  *string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

var globalRefreshTokens = make(map[string]RefreshTokenRecord) // keyed by TokenHash
var globalRefreshMu sync.RWMutex

func (s *Store) SaveRefreshToken(rec RefreshTokenRecord) {
	globalRefreshMu.Lock()
	defer globalRefreshMu.Unlock()
	globalRefreshTokens[rec.TokenHash] = rec
}

func (s *Store) GetRefreshToken(tokenHash string) (RefreshTokenRecord, bool) {
	globalRefreshMu.RLock()
	defer globalRefreshMu.RUnlock()
	rec, ok := globalRefreshTokens[tokenHash]
	return rec, ok
}

func (s *Store) RevokeRefreshToken(tokenHash string) bool {
	globalRefreshMu.Lock()
	defer globalRefreshMu.Unlock()
	rec, ok := globalRefreshTokens[tokenHash]
	if !ok {
		return false
	}
	now := time.Now().UTC()
	rec.RevokedAt = &now
	globalRefreshTokens[tokenHash] = rec
	return true
}

func (s *Store) RevokeRefreshTokenByID(id string) bool {
	globalRefreshMu.Lock()
	defer globalRefreshMu.Unlock()
	for hash, rec := range globalRefreshTokens {
		if rec.ID == id {
			now := time.Now().UTC()
			rec.RevokedAt = &now
			globalRefreshTokens[hash] = rec
			return true
		}
	}
	return false
}
