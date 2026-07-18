package store

import (
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
)

// NewSeeded is explicit test/demo data. Production starts with New instead.
func NewSeeded() *Store {
	now := time.Now().UTC()
	jakartaDate := now.In(time.FixedZone("Asia/Jakarta", 7*60*60)).Format("2006-01-02")
	demoPasswordHash, _ := authn.HashPassword("password")
	return &Store{
		Users: []User{
			{ID: "usr_gading", Email: "gading@gmail.com", DisplayName: "Gading", Role: "user", PasswordHash: demoPasswordHash, ExperiencePoints: 20, CreatedAt: now, UpdatedAt: now},
			{ID: "usr_dery", Email: "dery@gmail.com", DisplayName: "Dery", Role: "user", PasswordHash: demoPasswordHash, CreatedAt: now, UpdatedAt: now},
			{ID: "usr_suci", Email: "suci@gmail.com", DisplayName: "Suci", Role: "partner", PasswordHash: demoPasswordHash, CreatedAt: now, UpdatedAt: now},
			{ID: "usr_nasywa", Email: "nasywa@gmail.com", DisplayName: "Nasywa", Role: "platform_admin", PasswordHash: demoPasswordHash, CreatedAt: now, UpdatedAt: now},
		},
		Devices: []Device{
			{ID: "dev_android", UserID: "usr_gading", Platform: "android", Label: "Gading Android", AppVersion: "1.0.0", OSVersion: "Android 15", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "active", LastSeenAt: now.Add(-2 * time.Minute), CreatedAt: now, UpdatedAt: now},
			{ID: "dev_windows", UserID: "usr_gading", Platform: "windows", Label: "Gading Windows", AppVersion: "1.0.0", OSVersion: "Windows 11", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "degraded", LastSeenAt: now.Add(-38 * time.Minute), CreatedAt: now, UpdatedAt: now},
		},
		Partners: []Partner{{ID: "pl_active", UserID: "usr_gading", PartnerUserID: "usr_suci", Name: "Suci", Contact: "suci@gmail.com | +62 812-0000-0000", Status: "active", PartnerEmail: "suci@gmail.com", CreatedAt: now, UpdatedAt: now}},
		Approvals: []ApprovalRequest{
			{ID: "APR-2401", UserID: "usr_gading", DeviceID: "dev_android", PartnerLinkID: "pl_active", Action: "pause_protection", ExpiresIn: "Expires in 23 minutes", Status: "pending", Reason: "Troubleshooting app setup", RequestedDurationMinutes: 15, CreatedAt: now.Add(-7 * time.Minute), UpdatedAt: now.Add(-7 * time.Minute), ExpiresAt: now.Add(23 * time.Minute)},
			{ID: "APR-2398", UserID: "usr_gading", DeviceID: "dev_android", PartnerLinkID: "pl_active", Action: "uninstall_detected", ExpiresIn: "Reviewed yesterday", Status: "approved", Reason: "Accessibility service disabled", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour), ExpiresAt: now.Add(-23 * time.Hour)},
		},
		Modules:        seed.DemoEducationModules(now),
		EducationMedia: seed.DemoEducationMedia(now),
		SupportCases: []SupportCase{
			{ID: "CASE-1087", UserID: "usr_gading", Title: "Setup and permissions", Type: "device_recovery", Status: "waiting_user", Priority: "normal", Owner: "Gading", CreatedAt: now, UpdatedAt: now},
			{ID: "CASE-1084", UserID: "usr_dery", Title: "Partner approval issue", Type: "stuck_approval", Status: "open", Priority: "high", Owner: "Suci", CreatedAt: now, UpdatedAt: now},
		},
		DataRequests: []DataRequest{
			{ID: "DR-1042", UserID: "usr_gading", Title: "Export account data", Type: "export", Status: "completed", CreatedAt: now, UpdatedAt: now},
			{ID: "DR-1035", UserID: "usr_dery", Title: "Delete archived support notes", Type: "delete", Status: "processing", CreatedAt: now, UpdatedAt: now},
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
		Missions:   []DailyMission{{ID: "mis_001", UserID: "usr_gading", Date: jakartaDate, Mission1: true, Mission2: true, Mission3: false, Mission4: false, Mission5: false, CreatedAt: now, UpdatedAt: now}},
		Intentions: []Intention{{ID: "int_1", UserID: "usr_gading", Text: "Saya ingin menyelesaikan kuliah dengan pikiran yang lebih tenang.", Status: "active", CreatedAt: now, UpdatedAt: now}},
		CheckIns:   []CheckIn{{ID: "chk_1", UserID: "usr_gading", Mood: 4, Urge: 2, Context: "Merasa cukup tenang pagi ini.", CreatedAt: now}},
	}
}
