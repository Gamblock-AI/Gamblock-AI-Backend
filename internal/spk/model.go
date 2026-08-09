// Package spk implements the Gamblock-AI rule-based recovery decision support
// module (Sistem Pendukung Keputusan). It computes a support score and an
// engagement level from a user's recovery condition, detects time patterns in
// blocked events, selects a knowledge-base intervention, and evaluates the
// effectiveness of past interventions.
//
// The module is intentionally isolated: it owns no routes, handlers, storage,
// or AI-blocking logic, and it never produces user-facing copy, sends
// notifications, or contacts an accountability partner. Every weight,
// threshold, and rule is configurable through Config.
package spk

import "time"

// ChangeReadiness is the user's stated readiness to change. It is a scoring
// input, never a diagnosis, and also serves as a similarity dimension for the
// effectiveness feedback loop.
type ChangeReadiness string

const (
	ReadinessReadyFirm   ChangeReadiness = "READY_FIRM"
	ReadinessReadyHigh   ChangeReadiness = "READY_HIGH"
	ReadinessReadyMedium ChangeReadiness = "READY_MEDIUM"
	ReadinessReadyLow    ChangeReadiness = "READY_LOW"
)

// SupportLevel classifies the normalized support score.
type SupportLevel string

const (
	SupportLow    SupportLevel = "LOW"
	SupportMedium SupportLevel = "MEDIUM"
	SupportHigh   SupportLevel = "HIGH"
)

// EngagementLevel classifies the engagement index derived from the streak,
// mission, and learning scores.
type EngagementLevel string

const (
	EngagementHigh   EngagementLevel = "HIGH"
	EngagementMedium EngagementLevel = "MEDIUM"
	EngagementLow    EngagementLevel = "LOW"
)

// EffectivenessStatus is the outcome of an intervention evaluation.
type EffectivenessStatus string

const (
	EffectivenessEffective     EffectivenessStatus = "EFFECTIVE"
	EffectivenessUnclear       EffectivenessStatus = "UNCLEAR"
	EffectivenessLessEffective EffectivenessStatus = "LESS_EFFECTIVE"
	EffectivenessNotEvaluated  EffectivenessStatus = "NOT_EVALUATED"
)

// InterventionKey identifies a knowledge-base intervention. APPRECIATION is a
// response type, never an intervention key.
//
// Keys are abstract categories; the integration layer must map each one to a
// feature that actually exists in the product:
//
//   - NO_INTERVENTION          no extra task; the user keeps doing normal
//     daily missions
//   - LIGHT_EDUCATION          psychoeducation / light educational content
//   - SHORT_RECOVERY_PRACTICE  focused recovery practice / skill activity,
//     mapped by the integration layer to the Recovery Room, Learning Hub
//     skills, or the "recovery practice" daily mission; no new "Focus
//     Challenge" feature may be invented
//   - SHORT_GROUNDING          Grounding Tools
//   - SHORT_REFLECTION         journal/reflection or checkpoint reflection
//   - ALTERNATIVE_ACTIVITY     an activity already available in Learning Hub,
//     Skills, or Recovery Room; it is a category, never a new page/feature
//   - ACCOUNTABILITY_OPTION    recommendation only, requires
//     accountability_enabled; the module never contacts the partner
type InterventionKey string

const (
	InterventionNoIntervention        InterventionKey = "NO_INTERVENTION"
	InterventionLightEducation        InterventionKey = "LIGHT_EDUCATION"
	InterventionShortRecoveryPractice InterventionKey = "SHORT_RECOVERY_PRACTICE"
	InterventionShortGrounding        InterventionKey = "SHORT_GROUNDING"
	InterventionShortReflection       InterventionKey = "SHORT_REFLECTION"
	InterventionAlternativeActivity   InterventionKey = "ALTERNATIVE_ACTIVITY"
	InterventionAccountabilityOption  InterventionKey = "ACCOUNTABILITY_OPTION"
)

// ResponseType is the family of client response (for example the response Gami
// would render). The module produces the response type only; it never writes
// the actual message.
type ResponseType string

const (
	ResponseTypeAppreciation     ResponseType = "APPRECIATION"
	ResponseTypeEducation        ResponseType = "EDUCATION"
	ResponseTypeRecoveryPractice ResponseType = "RECOVERY_PRACTICE"
	ResponseTypeGrounding        ResponseType = "GROUNDING"
	ResponseTypeReflection       ResponseType = "REFLECTION"
	ResponseTypeActivity         ResponseType = "ACTIVITY"
	ResponseTypeAccountability   ResponseType = "ACCOUNTABILITY"
)

// ReasonCode describes the dominant rule behind an intervention choice. These
// are module-internal decision codes, not backend error catalog codes.
type ReasonCode string

const (
	ReasonBaseline             ReasonCode = "spk_baseline_rule"
	ReasonNoIntervention       ReasonCode = "spk_no_intervention_needed"
	ReasonHistoryEffective     ReasonCode = "spk_history_effective"
	ReasonHistoryLessEffective ReasonCode = "spk_history_less_effective"
	ReasonReadinessLow         ReasonCode = "spk_readiness_low_modifier"
	ReasonReadinessHigh        ReasonCode = "spk_readiness_high_modifier"
	ReasonFallback             ReasonCode = "spk_fallback_intervention"
)

// Condition is the input contract of the engine. Numeric and enum scoring
// inputs are pointers so a caller can explicitly mark a field as unavailable
// (nil). Context-only fields are never scored; personal_intention is forwarded
// unchanged for personalization outside the module.
type Condition struct {
	BlockedAttemptsToday   *int
	BlockedActiveDays7d    *int
	BlockedEventTimes      []time.Time
	RecoveryStreakDays     *int
	DailyMissionsCompleted *int
	DailyMissionsTotal     *int
	LearningActivities7d   *int
	ChangeReadiness        *ChangeReadiness
	EducationImpact        *string
	FinancialImpact        *string
	GamblingScreenTime     *string
	PreviousQuitAttempts   *int
	PersonalIntention      *string
	AccountabilityEnabled  *bool
	PreviousInterventions  []InterventionRecord
}

// InterventionRecord is the minimal history contract for one past
// intervention. similarity conditions use the levels recorded at the time.
type InterventionRecord struct {
	InterventionKey       InterventionKey     `json:"intervention_key"`
	Timestamp             time.Time           `json:"timestamp"`
	Completed             bool                `json:"completed"`
	SupportLevelAtTime    SupportLevel        `json:"support_level_at_time"`
	EngagementLevelAtTime EngagementLevel     `json:"engagement_level_at_time"`
	ReadinessLevelAtTime  ChangeReadiness     `json:"readiness_level_at_time"`
	EffectivenessStatus   EffectivenessStatus `json:"effectiveness_status"`
}

// TimeTrigger describes a detected blocked-event time pattern. PatternStart and
// PatternEnd bound the 2-hour window containing the clustered events;
// TriggerStart is TriggerLead minutes before the window start.
type TimeTrigger struct {
	HasTimePattern bool   `json:"has_time_pattern"`
	PatternStart   string `json:"pattern_start,omitempty"`
	PatternEnd     string `json:"pattern_end,omitempty"`
	TriggerStart   string `json:"trigger_start,omitempty"`
	TriggerEnd     string `json:"trigger_end,omitempty"`
}

// ScoreComponent is one factor of the support score. Contribution is
// (score/2.0)*weightPercent and is zero for unavailable factors.
type ScoreComponent struct {
	Key           string  `json:"key"`
	WeightPercent float64 `json:"weight_percent"`
	Score         int     `json:"score"`
	Contribution  float64 `json:"contribution"`
	Available     bool    `json:"available"`
}

// ScoreBreakdown exposes the raw support calculation for auditability.
type ScoreBreakdown struct {
	Components              []ScoreComponent `json:"components"`
	AvailableWeight         float64          `json:"available_weight_percent"`
	TotalWeight             float64          `json:"total_weight_percent"`
	RawAvailableScore       float64          `json:"raw_available_score"`
	EngagementRaw           int              `json:"engagement_raw"`
	EngagementNormalizedRaw float64          `json:"engagement_normalized_raw,omitempty"`
}

// ConditionContext forwards unscored context for personalization outside the
// module. These values never influence scoring.
type ConditionContext struct {
	EducationImpact      *string `json:"education_impact,omitempty"`
	FinancialImpact      *string `json:"financial_impact,omitempty"`
	GamblingScreenTime   *string `json:"gambling_screen_time,omitempty"`
	PreviousQuitAttempts *int    `json:"previous_quit_attempts,omitempty"`
	PersonalIntention    *string `json:"personal_intention,omitempty"`
}

// Result is the structured engine output. No user-facing message is produced.
type Result struct {
	SupportScore             float64          `json:"support_score"`
	SupportLevel             SupportLevel     `json:"support_level"`
	EngagementLevel          EngagementLevel  `json:"engagement_level"`
	InterventionNeeded       bool             `json:"intervention_needed"`
	InterventionKey          InterventionKey  `json:"intervention_key"`
	ResponseType             ResponseType     `json:"response_type"`
	ReasonCode               ReasonCode       `json:"reason_code"`
	TimeTrigger              *TimeTrigger     `json:"time_trigger"`
	EffectivenessHistoryUsed bool             `json:"effectiveness_history_used"`
	ScoreBreakdown           ScoreBreakdown   `json:"score_breakdown"`
	TriggeredRules           []string         `json:"triggered_rules"`
	UnavailableFields        []string         `json:"unavailable_fields"`
	Context                  ConditionContext `json:"context"`
}
