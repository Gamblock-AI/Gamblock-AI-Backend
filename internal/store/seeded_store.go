package store

import (
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
)

// NewSeeded is explicit test/demo data. Production starts with New instead.
func NewSeeded() *Store {
	now := time.Now().UTC()
	verifiedAt := now
	jakartaDate := now.In(time.FixedZone("Asia/Jakarta", 7*60*60)).Format("2006-01-02")
	aggregateDate := func(daysAgo int) time.Time {
		value := now.In(time.FixedZone("Asia/Jakarta", 7*60*60)).AddDate(0, 0, -daysAgo)
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	}
	demoPasswordHash, _ := authn.HashPassword("password")
	return &Store{
		Users: []User{
			{ID: "usr_gading", Email: "gading@gmail.com", DisplayName: "Gading", Role: "user", EmailVerifiedAt: &verifiedAt, PhoneE164: "+62895363116378", PhoneVerifiedAt: &verifiedAt, PasswordHash: demoPasswordHash, ExperiencePoints: 20, CreatedAt: now, UpdatedAt: now},
			{ID: "usr_dery", Email: "dery@gmail.com", DisplayName: "Dery", Role: "user", EmailVerifiedAt: &verifiedAt, PhoneE164: "+6282377341268", PhoneVerifiedAt: &verifiedAt, PasswordHash: demoPasswordHash, CreatedAt: now, UpdatedAt: now},
			{ID: "usr_suci", Email: "suci@gmail.com", DisplayName: "Suci", Role: "partner", EmailVerifiedAt: &verifiedAt, PhoneE164: "+6282385822192", PhoneVerifiedAt: &verifiedAt, PasswordHash: demoPasswordHash, CreatedAt: now, UpdatedAt: now},
			{ID: "usr_nasywa", Email: "nasywa@gmail.com", DisplayName: "Nasywa", Role: "admin", EmailVerifiedAt: &verifiedAt, PhoneE164: "+6282328514811", PhoneVerifiedAt: &verifiedAt, PasswordHash: demoPasswordHash, CreatedAt: now, UpdatedAt: now},
		},
		Devices: []Device{
			{ID: "dev_android", UserID: "usr_gading", Platform: "android", Label: "Gading Android", AppVersion: "1.0.0", OSVersion: "Android 15", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "active", LastSeenAt: now.Add(-2 * time.Minute), CreatedAt: now, UpdatedAt: now},
			{ID: "dev_windows", UserID: "usr_gading", Platform: "windows", Label: "Gading Windows", AppVersion: "1.0.0", OSVersion: "Windows 11", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "degraded", LastSeenAt: now.Add(-38 * time.Minute), CreatedAt: now, UpdatedAt: now},
			{ID: "dev_dery_android", UserID: "usr_dery", Platform: "android", Label: "Dery Android", AppVersion: "1.0.0", OSVersion: "Android 14", ModelVersion: "artifact-v0.3.1", RulesetVersion: "ruleset-2026.05.1", ProtectionStatus: "active", LastSeenAt: now.Add(-4 * time.Minute), CreatedAt: now, UpdatedAt: now},
		},
		Partners: []Partner{{ID: "pl_active", UserID: "usr_gading", PartnerUserID: "usr_suci", Name: "Suci", Contact: "suci@gmail.com | +62 812-0000-0000", Status: "active", PartnerEmail: "suci@gmail.com", CreatedAt: now, UpdatedAt: now}},
		AccountabilityGroups: []AccountabilityGroup{{
			ID: "grp_demo", OwnerPartnerID: "usr_suci", OwnerName: "Suci",
			Name: "Kelas Informatika C", Description: "Kelas pendampingan mahasiswa Informatika yang berfokus pada dukungan dan keputusan proteksi.",
			JoinCode: "GAMBLOCK42", JoinCodeHash: "cf555032cd87549c8369da3e5148f4fdcc6833a78c2f905b9944d2fa4cc04c45",
			JoinCodeHint: "CK42", Status: "active", MemberCount: 1, CodeRotatedAt: now, CreatedAt: now, UpdatedAt: now,
		}},
		AccountabilityMemberships: []AccountabilityMembership{{
			ID: "mbr_active", GroupID: "grp_demo", StudentID: "usr_gading", StudentName: "Gading", StudentMail: "gading@gmail.com",
			Status: "active", Sharing: SharingPreferences{ProtectionHealth: true, ProtectionActivity: true, RecoveryEngagement: true, EducationProgress: true},
			JoinedAt: now, CreatedAt: now, UpdatedAt: now,
		}},
		AggregateEvents: []AggregateEvent{
			{ID: "agg_seed_gading_android_0", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "seed:gading:android:0:block", EventType: "block_count_sync", EventDate: aggregateDate(0), Count: 3, CreatedAt: now},
			{ID: "agg_seed_gading_windows_0", UserID: "usr_gading", DeviceID: "dev_windows", IdempotencyKey: "seed:gading:windows:0:block", EventType: "block_count_sync", EventDate: aggregateDate(0), Count: 1, CreatedAt: now},
			{ID: "agg_seed_gading_android_1", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "seed:gading:android:1:block", EventType: "block_count_sync", EventDate: aggregateDate(1), Count: 2, CreatedAt: now},
			{ID: "agg_seed_gading_windows_2", UserID: "usr_gading", DeviceID: "dev_windows", IdempotencyKey: "seed:gading:windows:2:block", EventType: "block_count_sync", EventDate: aggregateDate(2), Count: 2, CreatedAt: now},
			{ID: "agg_seed_gading_android_3", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "seed:gading:android:3:block", EventType: "block_count_sync", EventDate: aggregateDate(3), Count: 1, CreatedAt: now},
			{ID: "agg_seed_dery_android_0", UserID: "usr_dery", DeviceID: "dev_dery_android", IdempotencyKey: "seed:dery:android:0:block", EventType: "block_count_sync", EventDate: aggregateDate(0), Count: 2, CreatedAt: now},
			{ID: "agg_seed_dery_android_4", UserID: "usr_dery", DeviceID: "dev_dery_android", IdempotencyKey: "seed:dery:android:4:block", EventType: "block_count_sync", EventDate: aggregateDate(4), Count: 1, CreatedAt: now},
		},
		Approvals: []ApprovalRequest{
			{ID: "APR-2401", UserID: "usr_gading", DeviceID: "dev_android", MembershipID: "mbr_active", Action: "pause_protection", ExpiresIn: "Expires in 23 minutes", Status: "pending", Reason: "Troubleshooting app setup", RequestedDurationMinutes: 15, CreatedAt: now.Add(-7 * time.Minute), UpdatedAt: now.Add(-7 * time.Minute), ExpiresAt: now.Add(23 * time.Minute)},
			{ID: "APR-2398", UserID: "usr_gading", DeviceID: "dev_android", MembershipID: "mbr_active", Action: "uninstall_detected", ExpiresIn: "Reviewed yesterday", Status: "approved", Reason: "Accessibility service disabled", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour), ExpiresAt: now.Add(-23 * time.Hour)},
		},
		Modules:        seed.DemoEducationModules(now),
		EducationMedia: seed.DemoEducationMedia(now),
		SupportCases: []SupportCase{
			{ID: "CASE-1087", UserID: "usr_gading", Title: "Setup and permissions", Type: "device_recovery", Status: "waiting_user", Priority: "normal", Owner: "Gading", CreatedAt: now, UpdatedAt: now},
			{ID: "CASE-1084", UserID: "usr_dery", Title: "Cannot finish app setup", Type: "device_recovery", Status: "waiting_user", Priority: "normal", Owner: "Dery", CreatedAt: now, UpdatedAt: now},
		},
		DataRequests: []DataRequest{
			{ID: "DR-1042", UserID: "usr_gading", Title: "Export account data", Type: "export", Status: "completed", FailureCode: "result_unavailable", CreatedAt: now, UpdatedAt: now},
			{ID: "DR-1035", UserID: "usr_dery", Title: "Delete archived support notes", Type: "delete", Status: "processing", CreatedAt: now, UpdatedAt: now},
		},
		AuditEvents: []AuditEvent{
			{ID: "audit_1", Actor: "nasywa@gmail.com", Action: "education_module_published", TargetType: "education_module", Target: "mod_impulse_cycle", CreatedAt: now, UpdatedAt: now},
		},
		NotificationEvents: []NotificationItem{{ID: "ntf_1", Channel: "whatsapp", Recipient: "+6282385822192", Status: "sent", Reason: "approval_request", CreatedAt: now, UpdatedAt: now}},
		JournalEntries: []JournalEntry{
			{ID: "ref_1", UserID: "usr_gading", Text: "Hari ini saya berhasil menahan diri dari godaan untuk membuka aplikasi berkat Pattern Interrupt.", Mood: "😊 Baik", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour)},
			{ID: "ref_2", UserID: "usr_gading", Text: "Merasa cemas di sore hari karena bosan, tapi berhasil mengalihkan perhatian dengan jalan kaki dan membaca modul kesadaran.", Mood: "😟 Cemas", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)},
		},
		Missions: []DailyMission{{ID: "mis_001", UserID: "usr_gading", Date: jakartaDate, Mission1: true, Mission2: true, Mission3: false, Mission4: false, Mission5: false, CreatedAt: now, UpdatedAt: now}},
		RecoveryPracticeSessions: []RecoveryPracticeSession{
			{ID: "practice_demo_001", UserID: "usr_gading", PracticeKind: "grounding_54321", DurationSeconds: 120, Feedback: "lighter", CompletedAt: now.Add(-30 * time.Minute), CreatedAt: now.Add(-30 * time.Minute)},
		},
		LearningProgress: []LearningProgress{
			{UserID: "usr_gading", ItemID: "learn_know_your_urges", State: "completed", CreatedAt: now.Add(-72 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour)},
			{UserID: "usr_gading", ItemID: "learn_pause_technique", State: "started", CreatedAt: now.Add(-4 * 24 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)},
			{UserID: "usr_gading", ItemID: "learn_financial_guardrails", State: "started", CreatedAt: now.Add(-6 * 24 * time.Hour), UpdatedAt: now.Add(-3 * 24 * time.Hour)},
		},
		EducationProgress: []EducationProgress{
			{ID: "eduprog_impulse", UserID: "usr_gading", ModuleID: "mod_impulse_cycle", Revision: 1, CompletedSectionIDs: []string{"cycle-map", "choice-point"}, ProgressPercent: 66, CreatedAt: now.Add(-3 * 24 * time.Hour), UpdatedAt: now.Add(-2 * 24 * time.Hour)},
		},
	}
}
