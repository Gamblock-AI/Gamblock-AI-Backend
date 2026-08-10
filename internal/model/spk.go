package model

import "time"

// InterventionRecord persists one SPK daily recommendation so the
// effectiveness feedback loop can feed history back into the engine. It
// carries only SPK decision metadata and the optional LLM-personalized message;
// it never stores browsing content or blocked-event timestamps.
type InterventionRecord struct {
	ID                      string     `json:"id"`
	UserID                  string     `json:"-"`
	InterventionKey         string     `json:"intervention_key"`
	ResponseType            string     `json:"response_type"`
	SupportLevel            string     `json:"support_level"`
	EngagementLevel         string     `json:"engagement_level"`
	ReadinessLevel          string     `json:"readiness_level,omitempty"`
	Status                  string     `json:"status"`
	RecommendedAt           time.Time  `json:"recommended_at"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
	EffectivenessStatus     string     `json:"effectiveness_status"`
	PersonalizedMessage     string     `json:"personalized_message,omitempty"`
	PersonalizedExplanation string     `json:"personalized_explanation,omitempty"`
	LLMUsed                 bool       `json:"llm_used"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// BlockedEvent stores a system-generated blocked-event timestamp (when a block
// fired) for SPK risky-hour pattern detection. It carries no URL, domain, DOM,
// or other browsing content.
type BlockedEvent struct {
	ID         string    `json:"id"`
	UserID     string    `json:"-"`
	DeviceID   string    `json:"device_id,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// SpkPreference holds the per-student SPK privacy set. Each boolean gates
// which data the daily recommendation may use (usage, never storage):
// SpkRecommendationEnabled is the master switch, SpkUseProtection gates
// aggregate block counts + blocked-event timestamps, SpkUseRecovery gates
// streak/mission/learning activity, SpkUsePersonal gates the self-reported
// intention context, and LLMPersonalizationEnabled gates the DeepSeek LLM.
type SpkPreference struct {
	ID                         string    `json:"id"`
	UserID                     string    `json:"user_id"`
	SpkRecommendationEnabled   bool      `json:"spk_recommendation_enabled"`
	SpkUseProtection           bool      `json:"spk_use_protection"`
	SpkUseRecovery             bool      `json:"spk_use_recovery"`
	SpkUsePersonal             bool      `json:"spk_use_personal"`
	LLMPersonalizationEnabled  bool      `json:"llm_personalization_enabled"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

// SpkFeature maps one SPK intervention key to an actual product surface. The
// module stays copy-free: `feature_id`/`category`/`route` are machine-readable
// descriptors that the client renders with its own localized labels.
type SpkFeature struct {
	InterventionKey string `json:"intervention_key"`
	ResponseType    string `json:"response_type"`
	FeatureID       string `json:"feature_id"`
	Category        string `json:"category"`
	Route           string `json:"route,omitempty"`
	Action          string `json:"action,omitempty"`
	Load            int    `json:"load"`
}

// SpkDataState classifies how much scoring data was available for the
// recommendation so clients can adjust their messaging.
type SpkDataState string

const (
	SpkDataSufficient   SpkDataState = "sufficient"
	SpkDataPartial      SpkDataState = "partial"
	SpkDataInsufficient SpkDataState = "insufficient"
)

// SpkReason exposes the decision rationale in a structured, localizable form.
// Copy lives on the client; the backend only emits machine-readable keys,
// levels, and the contributing score factors.
type SpkReason struct {
	Code            string           `json:"code"`
	SupportLevel    string           `json:"support_level"`
	EngagementLevel string           `json:"engagement_level"`
	SupportScore    float64          `json:"support_score"`
	Factors         []SpkReasonFactor `json:"factors"`
}

// SpkReasonFactor is one available score component. Score follows the engine
// convention: 0 healthy, 1 moderate, 2 risk. Factors are ordered by weight.
type SpkReasonFactor struct {
	Key    string  `json:"key"`
	Score  int     `json:"score"`
	Weight float64 `json:"weight_percent"`
}

// SpkDataGap is one actionable piece of missing data the student can add to
// sharpen the recommendation. Route is empty when the action has no dedicated
// page (the app already prompts it).
type SpkDataGap struct {
	Key    string `json:"key"`
	Action string `json:"action"`
	Route  string `json:"route,omitempty"`
}

// SpkRecommendation is the dashboard-facing output of the SPK engine plus the
// feature mapping, data-sufficiency summary, and optional LLM personalization.
type SpkRecommendation struct {
	RecommendationID        string           `json:"recommendation_id"`
	RecommendedAt           time.Time        `json:"recommended_at"`
	RecommendationEnabled   bool             `json:"recommendation_enabled"`
	Feature                 SpkFeature       `json:"feature"`
	SupportLevel            string           `json:"support_level"`
	SupportScore            float64          `json:"support_score"`
	EngagementLevel         string           `json:"engagement_level"`
	InterventionNeeded      bool             `json:"intervention_needed"`
	ReasonCode              string           `json:"reason_code"`
	Reason                  SpkReason        `json:"reason"`
	TimeTrigger             *SpkTimeTrigger  `json:"time_trigger"`
	EffectivenessUsed       bool             `json:"effectiveness_history_used"`
	TriggeredRules          []string         `json:"triggered_rules,omitempty"`
	DataState               SpkDataState     `json:"data_state"`
	DataGaps                []SpkDataGap     `json:"data_gaps,omitempty"`
	AvailableWeightPercent  float64          `json:"available_weight_percent"`
	UnavailableFields       []string         `json:"unavailable_fields,omitempty"`
	PersonalizedMessage     string           `json:"personalized_message,omitempty"`
	PersonalizedExplanation string           `json:"personalized_explanation,omitempty"`
	LLMUsed                 bool             `json:"llm_used"`
}

// SpkTimeTrigger mirrors the engine's risky-hour window result.
type SpkTimeTrigger struct {
	HasTimePattern bool   `json:"has_time_pattern"`
	PatternStart   string `json:"pattern_start,omitempty"`
	PatternEnd     string `json:"pattern_end,omitempty"`
	TriggerStart   string `json:"trigger_start,omitempty"`
	TriggerEnd     string `json:"trigger_end,omitempty"`
}
