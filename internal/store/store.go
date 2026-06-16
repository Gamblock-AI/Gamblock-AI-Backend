package store

import (
	"strings"
	"sync"
	"time"
)

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
	Missions          []DailyMission    `json:"missions"`
}

type JournalEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Text      string    `json:"text"`
	Mood      string    `json:"mood"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Device struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Platform         string    `json:"platform"`
	Label            string    `json:"label"`
	AppVersion       string    `json:"app_version"`
	OSVersion        string    `json:"os_version"`
	ModelVersion     string    `json:"model_version"`
	RulesetVersion   string    `json:"ruleset_version"`
	ProtectionStatus string    `json:"protection_status"`
	LastSeenAt       time.Time `json:"last_seen_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Partner struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Contact      string    `json:"contact"`
	Status       string    `json:"status"`
	PartnerEmail string    `json:"partner_email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ApprovalRequest struct {
	ID                       string    `json:"id"`
	Action                   string    `json:"action"`
	ExpiresIn                string    `json:"expires_in"`
	Status                   string    `json:"status"`
	Reason                   string    `json:"reason"`
	RequestedDurationMinutes int       `json:"requested_duration_minutes"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type EducationModule struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	Summary          string    `json:"summary"`
	BodyMarkdown     string    `json:"body_markdown"`
	EstimatedMinutes int       `json:"estimated_minutes"`
	Progress         float64   `json:"progress"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type SupportCase struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DataRequest struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	Members   int       `json:"members"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Release struct {
	ID              string         `json:"id"`
	Version         string         `json:"version"`
	Platform        string         `json:"platform"`
	SHA256          string         `json:"sha256"`
	Status          string         `json:"status"`
	DownloadURL     string         `json:"download_url"`
	Metrics         map[string]any `json:"metrics"`
	PublishedAtText string         `json:"published_at_text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type AuditEvent struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NotificationItem struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

type DailyMission struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Date      string    `json:"date"`
	Mission1  bool      `json:"mission_1"`
	Mission2  bool      `json:"mission_2"`
	Mission3  bool      `json:"mission_3"`
	Mission4  bool      `json:"mission_4"`
	Mission5  bool      `json:"mission_5"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
