package spk

import (
	"fmt"
	"reflect"
	"time"
)

// Engine is the SPK orchestrator. It is safe for concurrent use because it
// keeps only an immutable Config.
type Engine struct {
	config Config
}

// NewEngine builds an engine from the given config. An empty config produces
// the DefaultConfig baseline. For custom configs start from DefaultConfig() and
// override only what you need; scalar weights and thresholds are honored as
// provided, so a zero weight intentionally disables that factor.
func NewEngine(config Config) *Engine {
	if reflect.DeepEqual(config, Config{}) {
		return &Engine{config: DefaultConfig()}
	}
	return &Engine{config: config.completed()}
}

// Config exposes the effective (normalized) configuration.
func (e *Engine) Config() Config {
	return e.config
}

// Evaluate runs the full decision pipeline and returns the structured result.
// The reference time is time.Now().UTC(); use EvaluateAt for deterministic
// time-pattern evaluation.
func (e *Engine) Evaluate(cond Condition) Result {
	return e.EvaluateAt(cond, time.Now().UTC())
}

// EvaluateAt runs the full decision pipeline against a fixed reference time so
// the 7-day time-pattern window is deterministic and testable.
func (e *Engine) EvaluateAt(cond Condition, now time.Time) Result {
	cfg := e.config

	support := cfg.evaluateSupport(cond)
	engagement := cfg.evaluateEngagement(support)
	trigger := cfg.detectTimePattern(cond.BlockedEventTimes, now)
	selection := cfg.selectIntervention(cond, support.level, engagement.level, cond.ChangeReadiness)

	readinessValue := ChangeReadiness("")
	if cond.ChangeReadiness != nil {
		readinessValue = *cond.ChangeReadiness
	}
	similar := cfg.similarHistory(cond, support.level, engagement.level, readinessValue)

	return Result{
		SupportScore:             support.score,
		SupportLevel:             support.level,
		EngagementLevel:          engagement.level,
		InterventionNeeded:       selection.needed,
		InterventionKey:          selection.key,
		ResponseType:             selection.responseType,
		ReasonCode:               selection.reasonCode,
		TimeTrigger:              trigger,
		EffectivenessHistoryUsed: selection.historyUsed,
		ScoreBreakdown: ScoreBreakdown{
			Components:              support.components,
			AvailableWeight:         support.availableWeight,
			TotalWeight:             support.totalWeight,
			RawAvailableScore:       rawFromComponents(support.components),
			EngagementRaw:           engagement.raw,
			EngagementNormalizedRaw: engagement.normalizedRaw,
		},
		TriggeredRules:    cfg.buildRules(support, engagement, trigger, cond, selection, similar),
		UnavailableFields: cond.unavailableFields(),
		Context:           cond.context(),
	}
}

func rawFromComponents(components []ScoreComponent) float64 {
	total := 0.0
	for _, component := range components {
		total += component.Contribution
	}
	return total
}

func (c Config) buildRules(support supportEvaluation, engagement engagementEvaluation, trigger *TimeTrigger, cond Condition, selection selection, similar []InterventionRecord) []string {
	rules := []string{
		fmt.Sprintf("baseline_%s_%s", support.level, engagement.level),
		fmt.Sprintf("support_level=%s", support.level),
		fmt.Sprintf("engagement_level=%s", engagement.level),
	}
	for _, component := range support.components {
		if component.Available {
			rules = append(rules, fmt.Sprintf("%s_score=%d", component.Key, component.Score))
		}
	}
	if support.availableWeight < support.totalWeight {
		rules = append(rules, "insufficient_data_normalized")
	}
	for _, key := range cond.invalidInputs() {
		rules = append(rules, "invalid_input="+key)
	}
	if trigger != nil {
		rules = append(rules, "time_pattern_detected")
	}
	if cond.ChangeReadiness != nil && isValidReadiness(*cond.ChangeReadiness) {
		rules = append(rules, fmt.Sprintf("readiness=%s", *cond.ChangeReadiness))
		switch *cond.ChangeReadiness {
		case ReadinessReadyLow:
			rules = append(rules, "readiness_low_light_priority")
		case ReadinessReadyHigh, ReadinessReadyFirm:
			rules = append(rules, "readiness_high_focus_priority")
		}
	}
	applied := map[string]bool{}
	for _, record := range selection.appliedHistory {
		applied[historyRuleKey(record)] = true
	}
	for _, record := range similar {
		if applied[historyRuleKey(record)] {
			rules = append(rules, fmt.Sprintf("history_applied_%s=%s", record.EffectivenessStatus, record.InterventionKey))
		} else {
			rules = append(rules, fmt.Sprintf("history_found_%s=%s", record.EffectivenessStatus, record.InterventionKey))
		}
	}
	if selection.key == InterventionAccountabilityOption {
		rules = append(rules, "accountability_option_selected")
	}
	if len(rules) == 0 {
		rules = append(rules, "no_rule_fired")
	}
	return rules
}

func historyRuleKey(record InterventionRecord) string {
	return string(record.EffectivenessStatus) + "|" + string(record.InterventionKey)
}

func (cond Condition) unavailableFields() []string {
	var out []string
	if cond.BlockedAttemptsToday == nil || *cond.BlockedAttemptsToday < 0 {
		out = append(out, KeyBlockedToday)
	}
	if cond.BlockedActiveDays7d == nil || *cond.BlockedActiveDays7d < 0 {
		out = append(out, KeyBlockedPersistence)
	}
	if cond.BlockedEventTimes == nil {
		out = append(out, "blocked_event_times")
	}
	if cond.RecoveryStreakDays == nil || *cond.RecoveryStreakDays < 0 {
		out = append(out, KeyRecoveryStreak)
	}
	if cond.DailyMissionsCompleted == nil || *cond.DailyMissionsCompleted < 0 {
		out = append(out, KeyDailyMissions)
	}
	if cond.DailyMissionsTotal == nil || *cond.DailyMissionsTotal < 0 {
		out = append(out, "daily_missions_total")
	}
	if cond.LearningActivities7d == nil || *cond.LearningActivities7d < 0 {
		out = append(out, KeyLearning)
	}
	if cond.ChangeReadiness == nil || !isValidReadiness(*cond.ChangeReadiness) {
		out = append(out, KeyReadiness)
	}
	if cond.EducationImpact == nil {
		out = append(out, "education_impact")
	}
	if cond.FinancialImpact == nil {
		out = append(out, "financial_impact")
	}
	if cond.GamblingScreenTime == nil {
		out = append(out, "gambling_screen_time")
	}
	if cond.PreviousQuitAttempts == nil {
		out = append(out, "previous_quit_attempts")
	}
	if cond.PersonalIntention == nil {
		out = append(out, "personal_intention")
	}
	if cond.AccountabilityEnabled == nil {
		out = append(out, "accountability_enabled")
	}
	return out
}

func (cond Condition) invalidInputs() []string {
	var out []string
	if cond.BlockedAttemptsToday != nil && *cond.BlockedAttemptsToday < 0 {
		out = append(out, KeyBlockedToday)
	}
	if cond.BlockedActiveDays7d != nil && *cond.BlockedActiveDays7d < 0 {
		out = append(out, KeyBlockedPersistence)
	}
	if cond.RecoveryStreakDays != nil && *cond.RecoveryStreakDays < 0 {
		out = append(out, KeyRecoveryStreak)
	}
	if cond.DailyMissionsCompleted != nil && *cond.DailyMissionsCompleted < 0 {
		out = append(out, KeyDailyMissions)
	}
	if cond.DailyMissionsTotal != nil && *cond.DailyMissionsTotal < 0 {
		out = append(out, "daily_missions_total")
	}
	if cond.LearningActivities7d != nil && *cond.LearningActivities7d < 0 {
		out = append(out, KeyLearning)
	}
	if cond.ChangeReadiness != nil && !isValidReadiness(*cond.ChangeReadiness) {
		out = append(out, KeyReadiness)
	}
	return out
}

func (cond Condition) context() ConditionContext {
	return ConditionContext{
		EducationImpact:      cond.EducationImpact,
		FinancialImpact:      cond.FinancialImpact,
		GamblingScreenTime:   cond.GamblingScreenTime,
		PreviousQuitAttempts: cond.PreviousQuitAttempts,
		PersonalIntention:    cond.PersonalIntention,
	}
}
