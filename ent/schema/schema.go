package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

func idField() ent.Field {
	return field.String("id").DefaultFunc(func() string { return uuid.NewString() }).Immutable()
}
func createdAt() ent.Field { return field.Time("created_at").Default(time.Now).Immutable() }
func updatedAt() ent.Field { return field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now) }

type User struct{ ent.Schema }

func (User) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("email").Unique(), field.String("display_name"), field.String("password_hash").Optional().Nillable().Sensitive(), field.String("avatar_url").Optional().Nillable(), field.String("google_subject").Optional().Nillable().Unique(), field.Enum("role").Values("user", "partner", "organization_owner", "organization_admin", "content_admin", "model_release_operator", "support_operator", "research_evaluator", "platform_admin").Default("user"), field.Time("disabled_at").Optional().Nillable(), createdAt(), updatedAt()}
}

type RefreshToken struct{ ent.Schema }

func (RefreshToken) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("token_hash"), field.String("device_id").Optional().Nillable(), field.Time("expires_at"), field.Time("revoked_at").Optional().Nillable(), createdAt()}
}

type Device struct{ ent.Schema }

func (Device) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("client_instance_id").Optional().Nillable(), field.Enum("platform").Values("android", "windows", "linux", "macos", "web"), field.String("label"), field.String("app_version").Default(""), field.String("os_version").Default(""), field.String("model_version").Optional().Nillable(), field.String("ruleset_version").Optional().Nillable(), field.Enum("protection_status").Values("active", "degraded", "paused", "inactive").Default("inactive"), field.Time("last_seen_at").Optional().Nillable(), createdAt(), updatedAt()}
}

func (Device) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "client_instance_id").Unique()}
}

type PartnerLink struct{ ent.Schema }

func (PartnerLink) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("partner_user_id").Optional().Nillable(), field.String("partner_email"), field.String("partner_phone").Optional().Nillable(), field.Enum("status").Values("invited", "active", "revoked", "expired").Default("invited"), field.String("invite_token_hash").Optional().Nillable(), field.Time("accepted_at").Optional().Nillable(), field.Time("revoked_at").Optional().Nillable(), createdAt(), updatedAt()}
}

type ApprovalRequest struct{ ent.Schema }

func (ApprovalRequest) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("device_id").Optional().Nillable(), field.String("partner_link_id"), field.String("quick_token_hash").Optional().Nillable().Unique().Sensitive(), field.Enum("action").Values("disable_protection", "remove_partner", "uninstall_detected", "reset_settings", "pause_protection", "emergency_access"), field.Enum("status").Values("pending", "approved", "denied", "expired", "cancelled").Default("pending"), field.String("reason").Optional().Nillable(), field.Int("requested_duration_minutes").Optional().Nillable(), field.Time("expires_at"), field.String("resolved_by").Optional().Nillable(), field.Time("resolved_at").Optional().Nillable(), field.Time("applied_at").Optional().Nillable(), field.Time("grant_expires_at").Optional().Nillable(), createdAt(), updatedAt()}
}

type NotificationDelivery struct{ ent.Schema }

func (NotificationDelivery) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("approval_request_id").Optional().Nillable(), field.String("support_case_id").Optional().Nillable(), field.Enum("channel").Values("email", "whatsapp", "in_app"), field.String("recipient"), field.Enum("status").Values("queued", "sent", "failed").Default("queued"), field.String("provider_message_id").Optional().Nillable(), field.String("error_code").Optional().Nillable(), field.Int("attempt_count").Default(0), createdAt(), updatedAt()}
}

type PsychoeducationModule struct{ ent.Schema }

func (PsychoeducationModule) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("slug").Unique(), field.String("title"), field.String("summary"), field.Text("body_markdown"), field.Int("estimated_minutes"), field.Enum("status").Values("draft", "published", "archived").Default("draft"), createdAt(), updatedAt()}
}

type ModelRelease struct{ ent.Schema }

func (ModelRelease) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("version").Unique(), field.Enum("platform").Values("android", "windows", "linux", "macos", "web", "all").Default("all"), field.String("artifact_path"), field.String("sha256"), field.Float("threshold").Default(0), field.String("contract_version").Default("v1"), field.Enum("status").Values("draft", "validated", "staged", "published", "paused", "rolled_back", "retired").Default("draft"), field.JSON("metrics_json", map[string]any{}).Optional(), field.Time("published_at").Optional().Nillable(), createdAt(), updatedAt()}
}

type RulesetRelease struct{ ent.Schema }

func (RulesetRelease) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("version").Unique(), field.String("artifact_path"), field.String("sha256"), field.Enum("status").Values("draft", "validated", "staged", "published", "paused", "rolled_back", "retired").Default("draft"), field.JSON("rules_json", map[string]any{}).Optional(), field.Time("published_at").Optional().Nillable(), createdAt(), updatedAt()}
}

type NetworkRulesetRelease struct{ ent.Schema }

func (NetworkRulesetRelease) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("version").Unique(), field.String("artifact_path"), field.String("sha256"), field.Enum("status").Values("draft", "validated", "published", "retired").Default("draft"), field.JSON("rules_json", map[string]any{}).Optional(), field.String("created_by").Optional().Nillable(), field.Time("published_at").Optional().Nillable(), createdAt()}
}

type AggregateEvent struct{ ent.Schema }

func (AggregateEvent) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("device_id").Optional().Nillable(), field.String("idempotency_key").Optional().Unique(), field.Enum("event_type").Values("intervention_shown", "block_count_sync", "tamper_detected", "permission_revoked", "model_updated", "ruleset_updated"), field.Time("event_date"), field.Int("count"), field.JSON("metadata_json", map[string]any{}).Optional(), createdAt()}
}

type Organization struct{ ent.Schema }

func (Organization) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("name"), field.String("slug").Unique(), field.Enum("status").Values("active", "suspended", "archived").Default("active"), field.JSON("retention_policy_json", map[string]any{}).Optional(), field.String("created_by"), createdAt(), updatedAt()}
}

type OrganizationMember struct{ ent.Schema }

func (OrganizationMember) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("organization_id"), field.String("user_id"), field.Enum("role").Values("owner", "admin", "member", "viewer"), field.Enum("status").Values("invited", "active", "suspended", "left").Default("invited"), field.Time("joined_at").Optional().Nillable(), createdAt()}
}

type OrganizationInvite struct{ ent.Schema }

func (OrganizationInvite) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("organization_id"), field.String("email"), field.Enum("role").Values("admin", "member", "viewer"), field.String("token_hash"), field.Enum("status").Values("pending", "accepted", "expired", "revoked").Default("pending"), field.Time("expires_at"), createdAt()}
}

type OrganizationPolicy struct{ ent.Schema }

func (OrganizationPolicy) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("organization_id"), field.String("key"), field.JSON("value_json", map[string]any{}), field.String("created_by"), updatedAt()}
}

type ReportRollup struct{ ent.Schema }

func (ReportRollup) Fields() []ent.Field {
	return []ent.Field{idField(), field.Enum("scope").Values("user", "partner", "organization", "platform"), field.String("scope_id"), field.Enum("period").Values("daily", "weekly", "monthly"), field.Time("period_start"), field.JSON("metrics_json", map[string]any{}), createdAt()}
}

type SupportCase struct{ ent.Schema }

func (SupportCase) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("organization_id").Optional().Nillable(), field.Enum("type").Values("technical_support", "account_recovery", "partner_abuse", "stuck_approval", "device_recovery", "notification_failure", "organization_dispute", "accountability_guidance", "privacy_request"), field.Enum("status").Values("open", "waiting_user", "waiting_internal", "resolved", "closed").Default("open"), field.Enum("priority").Values("low", "normal", "high", "urgent").Default("normal"), field.String("summary"), createdAt(), updatedAt()}
}

type SupportActionAudit struct{ ent.Schema }

func (SupportActionAudit) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("support_case_id"), field.String("operator_id"), field.String("action"), field.String("reason"), field.JSON("before_json", map[string]any{}).Optional(), field.JSON("after_json", map[string]any{}).Optional(), createdAt()}
}

type EmergencyKeyRequest struct{ ent.Schema }

func (EmergencyKeyRequest) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("requested_by"), field.String("device_id").Optional().Nillable(), field.String("reviewed_by").Optional().Nillable(), field.String("approved_by").Optional().Nillable(), field.Enum("status").Values("pending", "reviewed", "approved", "used", "expired").Default("pending"), field.String("key_hash").Optional().Nillable().Sensitive(), field.Time("request_expires_at"), field.Time("key_expires_at").Optional().Nillable(), field.Time("reviewed_at").Optional().Nillable(), field.Time("approved_at").Optional().Nillable(), field.Time("used_at").Optional().Nillable(), createdAt(), updatedAt()}
}

type DataRequest struct{ ent.Schema }

func (DataRequest) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.Enum("type").Values("export", "delete", "retention_review"), field.Enum("status").Values("pending", "processing", "completed", "rejected", "cancelled").Default("pending"), field.Time("requested_at").Default(time.Now), field.Time("completed_at").Optional().Nillable(), field.String("result_path").Optional().Nillable()}
}

type ModelRollout struct{ ent.Schema }

func (ModelRollout) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("model_release_id").Optional().Nillable(), field.String("ruleset_release_id").Optional().Nillable(), field.String("network_ruleset_release_id").Optional().Nillable(), field.Enum("status").Values("draft", "staged", "active", "paused", "rolled_back", "completed").Default("draft"), field.JSON("cohort_json", map[string]any{}).Optional(), field.String("created_by"), createdAt(), updatedAt()}
}

type ReleaseCohort struct{ ent.Schema }

func (ReleaseCohort) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("rollout_id"), field.Enum("platform").Values("android", "windows", "linux", "macos", "web", "all"), field.Int("percentage"), field.String("organization_id").Optional().Nillable(), field.String("app_version_constraint").Optional().Nillable(), createdAt()}
}

type AuditLog struct{ ent.Schema }

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("actor_id"), field.String("actor_email"), field.String("action"), field.String("target_type"), field.String("target_id"), field.String("reason"), field.JSON("metadata_json", map[string]any{}).Optional(), createdAt()}
}

type ContentProgress struct{ ent.Schema }

func (ContentProgress) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("module_slug"), field.Float("progress").Default(0), field.Time("completed_at").Optional().Nillable(), updatedAt()}
}

type Intention struct{ ent.Schema }

func (Intention) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.Text("intention_text"), field.Enum("status").Values("active", "paused", "archived").Default("active"), createdAt(), updatedAt()}
}

type CheckIn struct{ ent.Schema }

func (CheckIn) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.Int("mood_score").Comment("1-5 scale"), field.Int("urge_score").Comment("1-5 scale"), field.String("context_text").Optional().Nillable(), createdAt()}
}

type DailyMission struct{ ent.Schema }

func (DailyMission) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.String("mission_key"), field.Enum("status").Values("completed", "skipped", "pending").Default("completed"), field.Time("completed_at").Optional().Nillable(), createdAt()}
}

type Reflection struct{ ent.Schema }

func (Reflection) Fields() []ent.Field {
	return []ent.Field{idField(), field.String("user_id"), field.Text("content_encrypted"), field.String("prompt_key").Optional().Nillable(), createdAt(), updatedAt()}
}
