package spk

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(value int) *int    { return &value }
func boolPtr(value bool) *bool { return &value }

func eventAt(day, hour, minute int) time.Time {
	return time.Date(2026, time.August, day, hour, minute, 0, 0, time.UTC)
}

func TestSpk_BlockedZeroLongStreakActiveMissions_NoIntervention(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyHigh
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(0),
		BlockedActiveDays7d:    intPtr(0),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &readiness,
	})

	assert.Equal(t, SupportLow, result.SupportLevel)
	assert.Equal(t, EngagementHigh, result.EngagementLevel)
	assert.Equal(t, InterventionNoIntervention, result.InterventionKey)
	assert.Equal(t, ResponseTypeAppreciation, result.ResponseType)
	assert.False(t, result.InterventionNeeded)
	assert.Equal(t, ReasonNoIntervention, result.ReasonCode)
}

func TestSpk_BlockedHighLowEngagement_LightShortIntervention(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(6),
		BlockedActiveDays7d:    intPtr(6),
		RecoveryStreakDays:     intPtr(0),
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(0),
		ChangeReadiness:        &readiness,
	})

	assert.Equal(t, SupportHigh, result.SupportLevel)
	assert.Equal(t, EngagementLow, result.EngagementLevel)
	assert.Equal(t, InterventionShortGrounding, result.InterventionKey)
	assert.Equal(t, ResponseTypeGrounding, result.ResponseType)
	assert.True(t, result.InterventionNeeded)
	assert.Equal(t, ReasonBaseline, result.ReasonCode)
}

func TestSpk_BlockedHighEngagementHigh_AlternativeStrategyNotMoreMissions(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	history := []InterventionRecord{
		{
			InterventionKey:       InterventionAlternativeActivity,
			Timestamp:             eventAt(1, 12, 0),
			Completed:             true,
			SupportLevelAtTime:    SupportHigh,
			EngagementLevelAtTime: EngagementHigh,
			ReadinessLevelAtTime:  ReadinessReadyMedium,
			EffectivenessStatus:   EffectivenessEffective,
		},
	}
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:  intPtr(6),
		BlockedActiveDays7d:   intPtr(6),
		LearningActivities7d:  intPtr(3),
		ChangeReadiness:       &readiness,
		PreviousInterventions: history,
	})

	assert.Equal(t, SupportHigh, result.SupportLevel)
	assert.Equal(t, EngagementHigh, result.EngagementLevel)
	assert.Equal(t, InterventionAlternativeActivity, result.InterventionKey)
	assert.True(t, result.InterventionNeeded)
	assert.True(t, result.EffectivenessHistoryUsed)
	assert.Equal(t, ReasonHistoryEffective, result.ReasonCode)
}

func TestSpk_BlockedMediumEngagementHigh_RecoveryPractice(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(5),
		BlockedActiveDays7d:    intPtr(5),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &readiness,
	})

	assert.Equal(t, SupportMedium, result.SupportLevel)
	assert.Equal(t, EngagementHigh, result.EngagementLevel)
	assert.Equal(t, InterventionShortRecoveryPractice, result.InterventionKey)
	assert.Equal(t, ReasonBaseline, result.ReasonCode)
}

func TestSpk_ReadinessLow_LighterIntervention(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyLow
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(5),
		BlockedActiveDays7d:    intPtr(5),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &readiness,
	})

	assert.Equal(t, SupportMedium, result.SupportLevel)
	assert.Equal(t, EngagementHigh, result.EngagementLevel)
	assert.Equal(t, InterventionShortGrounding, result.InterventionKey)
	assert.Equal(t, ReasonReadinessLow, result.ReasonCode)
	load := DefaultConfig().KnowledgeBase[result.InterventionKey].Load
	practiceLoad := DefaultConfig().KnowledgeBase[InterventionShortRecoveryPractice].Load
	assert.Less(t, load, practiceLoad, "readiness modifier must pick a lighter intervention")
}

func TestSpk_HistoryEffective_SimilarConditionPrioritized(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	history := []InterventionRecord{
		{
			InterventionKey:       InterventionLightEducation,
			Timestamp:             eventAt(1, 12, 0),
			Completed:             true,
			SupportLevelAtTime:    SupportHigh,
			EngagementLevelAtTime: EngagementLow,
			ReadinessLevelAtTime:  ReadinessReadyMedium,
			EffectivenessStatus:   EffectivenessEffective,
		},
		{
			// Different readiness triple must not influence the decision.
			InterventionKey:       InterventionShortRecoveryPractice,
			Timestamp:             eventAt(2, 12, 0),
			Completed:             true,
			SupportLevelAtTime:    SupportHigh,
			EngagementLevelAtTime: EngagementLow,
			ReadinessLevelAtTime:  ReadinessReadyLow,
			EffectivenessStatus:   EffectivenessEffective,
		},
	}
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(6),
		BlockedActiveDays7d:    intPtr(6),
		RecoveryStreakDays:     intPtr(0),
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(0),
		ChangeReadiness:        &readiness,
		PreviousInterventions:  history,
	})

	assert.Equal(t, InterventionLightEducation, result.InterventionKey)
	assert.True(t, result.EffectivenessHistoryUsed)
	assert.Equal(t, ReasonHistoryEffective, result.ReasonCode)
	require.Contains(t, result.TriggeredRules, "history_applied_EFFECTIVE=LIGHT_EDUCATION")
}

func TestSpk_HistoryLessEffective_NotChosenFirst(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	history := []InterventionRecord{
		{
			InterventionKey:       InterventionShortGrounding,
			Timestamp:             eventAt(1, 12, 0),
			Completed:             true,
			SupportLevelAtTime:    SupportHigh,
			EngagementLevelAtTime: EngagementLow,
			ReadinessLevelAtTime:  ReadinessReadyMedium,
			EffectivenessStatus:   EffectivenessLessEffective,
		},
	}
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(6),
		BlockedActiveDays7d:    intPtr(6),
		RecoveryStreakDays:     intPtr(0),
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(0),
		ChangeReadiness:        &readiness,
		PreviousInterventions:  history,
	})

	assert.Equal(t, InterventionAlternativeActivity, result.InterventionKey)
	assert.Equal(t, ReasonHistoryLessEffective, result.ReasonCode)
	assert.True(t, result.EffectivenessHistoryUsed)
	require.Contains(t, result.TriggeredRules, "history_applied_LESS_EFFECTIVE=SHORT_GROUNDING")
}

func TestSpk_EngagementMediumMapping(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(3),
		BlockedActiveDays7d:    intPtr(3),
		RecoveryStreakDays:     intPtr(5),
		DailyMissionsCompleted: intPtr(2),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &readiness,
	})

	assert.Equal(t, 2, result.ScoreBreakdown.EngagementRaw)
	assert.Equal(t, EngagementMedium, result.EngagementLevel)
}

func TestSpk_AccountabilityOptionGatedByEnablement(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Baseline[LevelPair{Support: SupportMedium, Engagement: EngagementHigh}] = []InterventionKey{
		InterventionAccountabilityOption,
		InterventionShortRecoveryPractice,
	}
	engine := NewEngine(cfg)
	readiness := ReadinessReadyMedium
	base := Condition{
		BlockedAttemptsToday:   intPtr(5),
		BlockedActiveDays7d:    intPtr(5),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &readiness,
	}

	withoutAccountability := base
	withoutAccountability.AccountabilityEnabled = boolPtr(false)
	disabled := engine.Evaluate(withoutAccountability)
	assert.Equal(t, InterventionShortRecoveryPractice, disabled.InterventionKey)

	withAccountability := base
	withAccountability.AccountabilityEnabled = boolPtr(true)
	enabled := engine.Evaluate(withAccountability)
	assert.Equal(t, InterventionAccountabilityOption, enabled.InterventionKey)
}

func TestSpk_Effectiveness_NotCompleted(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		InterventionKey: InterventionShortRecoveryPractice,
		Completed:       false,
		BeforeBlocked:   []int{5, 5},
		AfterBlocked:    []int{1, 1},
	})
	assert.Equal(t, EffectivenessNotEvaluated, status)
}

func TestSpk_Effectiveness_IncompleteAfterWindow(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		InterventionKey: InterventionShortRecoveryPractice,
		Completed:       true,
		BeforeBlocked:   []int{5, 5},
		AfterBlocked:    []int{2},
	})
	assert.Equal(t, EffectivenessUnclear, status)
}

func TestSpk_Effectiveness_EffectiveWhenReductionReached(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		Completed:     true,
		BeforeBlocked: []int{5, 5},
		AfterBlocked:  []int{3, 4},
	})
	assert.Equal(t, EffectivenessEffective, status)
}

func TestSpk_Effectiveness_LessEffectiveBelowThreshold(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		Completed:     true,
		BeforeBlocked: []int{5, 5},
		AfterBlocked:  []int{4, 4},
	})
	assert.Equal(t, EffectivenessLessEffective, status)
}

func TestSpk_PartialData_NormalizedScoreValidOutput(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyHigh
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday: intPtr(5),
		BlockedActiveDays7d:  intPtr(2),
		ChangeReadiness:      &readiness,
	})

	assert.Equal(t, SupportMedium, result.SupportLevel)
	assert.InDelta(t, 30.0/55.0*100, result.SupportScore, 0.01)
	assert.Equal(t, EngagementMedium, result.EngagementLevel)
	assert.Equal(t, InterventionShortRecoveryPractice, result.InterventionKey)
	require.Contains(t, result.UnavailableFields, "recovery_streak_days")
	require.Contains(t, result.UnavailableFields, "daily_missions_completed")
	require.Contains(t, result.TriggeredRules, "insufficient_data_normalized")
	assert.Equal(t, float64(55), result.ScoreBreakdown.AvailableWeight)
}

func TestSpk_NoScoringData_StillValid(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	result := engine.Evaluate(Condition{AccountabilityEnabled: boolPtr(false)})

	assert.Equal(t, SupportLow, result.SupportLevel)
	assert.Equal(t, EngagementMedium, result.EngagementLevel)
	assert.Equal(t, InterventionLightEducation, result.InterventionKey)
	assert.True(t, result.InterventionNeeded)
	assert.NotEmpty(t, result.UnavailableFields)
}

func TestSpk_TimePattern_ClusterWithinTwoHours(t *testing.T) {
	cfg := DefaultConfig()
	now := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	trigger := cfg.detectTimePattern([]time.Time{
		eventAt(1, 22, 30),
		eventAt(2, 22, 45),
		eventAt(3, 23, 15),
	}, now)

	require.NotNil(t, trigger)
	assert.True(t, trigger.HasTimePattern)
	assert.Equal(t, "22:30", trigger.PatternStart)
	assert.Equal(t, "00:30", trigger.PatternEnd)
	assert.Equal(t, "22:00", trigger.TriggerStart)
	assert.Equal(t, "00:30", trigger.TriggerEnd)
}

func TestSpk_TimePattern_EventTimesInsideEngine(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	now := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	readiness := ReadinessReadyHigh
	result := engine.EvaluateAt(Condition{
		BlockedAttemptsToday: intPtr(6),
		BlockedActiveDays7d:  intPtr(5),
		BlockedEventTimes: []time.Time{
			eventAt(1, 22, 30),
			eventAt(2, 22, 45),
			eventAt(3, 23, 15),
		},
		ChangeReadiness: &readiness,
	}, now)

	require.NotNil(t, result.TimeTrigger)
	assert.True(t, result.TimeTrigger.HasTimePattern)
	assert.Equal(t, "22:00", result.TimeTrigger.TriggerStart)
	assert.Contains(t, result.TriggeredRules, "time_pattern_detected")
}

func TestSpk_TimePattern_NotEnoughEvents(t *testing.T) {
	cfg := DefaultConfig()
	now := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	trigger := cfg.detectTimePattern([]time.Time{
		eventAt(1, 22, 30),
		eventAt(2, 22, 45),
	}, now)
	assert.Nil(t, trigger)

	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyHigh
	result := engine.EvaluateAt(Condition{
		BlockedAttemptsToday: intPtr(0),
		BlockedActiveDays7d:  intPtr(0),
		BlockedEventTimes: []time.Time{
			eventAt(1, 8, 0),
			eventAt(2, 15, 0),
			eventAt(3, 22, 0),
		},
		ChangeReadiness: &readiness,
	}, now)
	assert.Nil(t, result.TimeTrigger)
}

func TestSpk_ResultJSONContract(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyHigh
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(0),
		BlockedActiveDays7d:    intPtr(0),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &readiness,
	})

	raw, err := json.Marshal(result)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, key := range []string{
		"support_score", "support_level", "engagement_level",
		"intervention_needed", "intervention_key", "response_type",
		"reason_code", "time_trigger", "effectiveness_history_used",
		"score_breakdown", "triggered_rules", "unavailable_fields",
	} {
		_, ok := decoded[key]
		assert.True(t, ok, "missing json key %q", key)
	}
	assert.Nil(t, decoded["time_trigger"])
}

func TestSpk_ConfigurableWeightsAndThresholds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WeightBlockedToday = 100
	cfg.WeightBlockedPersistence = 0
	cfg.WeightRecoveryStreak = 0
	cfg.WeightDailyMissions = 0
	cfg.WeightLearning = 0
	cfg.WeightReadiness = 0
	engine := NewEngine(cfg)
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday: intPtr(6),
		BlockedActiveDays7d:  intPtr(6),
	})

	assert.Equal(t, SupportHigh, result.SupportLevel)
	assert.Equal(t, float64(100), result.ScoreBreakdown.AvailableWeight)
	assert.InDelta(t, 100.0, result.SupportScore, 0.01)

	condition := Condition{
		BlockedAttemptsToday:   intPtr(2),
		BlockedActiveDays7d:    intPtr(0),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
	}

	thresholdCfg := DefaultConfig()
	thresholdCfg.BlockedTodayScore0Max = 0
	thresholdCfg.BlockedTodayScore1Max = 1
	thresholdEngine := NewEngine(thresholdCfg)
	thresholdResult := thresholdEngine.Evaluate(condition)
	defaultResult := NewEngine(DefaultConfig()).Evaluate(condition)

	thresholdComponent := findComponent(thresholdResult.ScoreBreakdown.Components, KeyBlockedToday)
	defaultComponent := findComponent(defaultResult.ScoreBreakdown.Components, KeyBlockedToday)
	require.NotNil(t, thresholdComponent)
	require.NotNil(t, defaultComponent)
	assert.Equal(t, 2, thresholdComponent.Score)
	assert.Equal(t, 1, defaultComponent.Score)
}

func TestSpk_Engagement_NoEngagementDataNeutralMedium(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday: intPtr(5),
		BlockedActiveDays7d:  intPtr(5),
		ChangeReadiness:      &readiness,
	})

	assert.Equal(t, EngagementMedium, result.EngagementLevel)
	assert.Equal(t, 0, result.ScoreBreakdown.EngagementRaw)
}

func TestSpk_Engagement_PartialDataNormalized(t *testing.T) {
	engine := NewEngine(DefaultConfig())

	low := engine.Evaluate(Condition{
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
	})
	assert.Equal(t, EngagementLow, low.EngagementLevel)
	assert.InDelta(t, 6.0, low.ScoreBreakdown.EngagementNormalizedRaw, 0.001)

	medium := engine.Evaluate(Condition{
		RecoveryStreakDays: intPtr(5),
	})
	assert.Equal(t, EngagementMedium, medium.EngagementLevel)
	assert.InDelta(t, 3.0, medium.ScoreBreakdown.EngagementNormalizedRaw, 0.001)

	high := engine.Evaluate(Condition{
		LearningActivities7d: intPtr(3),
	})
	assert.Equal(t, EngagementHigh, high.EngagementLevel)
	assert.InDelta(t, 0.0, high.ScoreBreakdown.EngagementNormalizedRaw, 0.001)
}

func TestSpk_Effectiveness_IncompleteBeforeWindow(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		Completed:     true,
		BeforeBlocked: []int{5},
		AfterBlocked:  []int{3, 3},
	})
	assert.Equal(t, EffectivenessUnclear, status)
}

func TestSpk_Effectiveness_ZeroBeforeTotalUnclear(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		Completed:     true,
		BeforeBlocked: []int{0, 0},
		AfterBlocked:  []int{1, 1},
	})
	assert.Equal(t, EffectivenessUnclear, status)
}

func TestSpk_Effectiveness_ExactWindowDaysTrimmed(t *testing.T) {
	cfg := DefaultConfig()
	status := cfg.EvaluateEffectiveness(EffectivenessInput{
		Completed:     true,
		BeforeBlocked: []int{5, 5, 20, 20},
		AfterBlocked:  []int{8, 8},
	})
	assert.Equal(t, EffectivenessLessEffective, status)
}

func TestSpk_History_UnclearDoesNotEnableHistoryUsed(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	history := []InterventionRecord{
		{
			InterventionKey:       InterventionLightEducation,
			Timestamp:             eventAt(1, 12, 0),
			Completed:             true,
			SupportLevelAtTime:    SupportHigh,
			EngagementLevelAtTime: EngagementLow,
			ReadinessLevelAtTime:  ReadinessReadyMedium,
			EffectivenessStatus:   EffectivenessUnclear,
		},
	}
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(6),
		BlockedActiveDays7d:    intPtr(6),
		RecoveryStreakDays:     intPtr(0),
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(0),
		ChangeReadiness:        &readiness,
		PreviousInterventions:  history,
	})

	assert.Equal(t, InterventionShortGrounding, result.InterventionKey)
	assert.False(t, result.EffectivenessHistoryUsed)
	assert.Equal(t, ReasonBaseline, result.ReasonCode)
	require.Contains(t, result.TriggeredRules, "history_found_UNCLEAR=LIGHT_EDUCATION")
	assert.NotContains(t, result.TriggeredRules, "history_applied_")
}

func TestSpk_History_NotEvaluatedDoesNotEnableHistoryUsed(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	readiness := ReadinessReadyMedium
	history := []InterventionRecord{
		{
			InterventionKey:       InterventionLightEducation,
			Timestamp:             eventAt(1, 12, 0),
			Completed:             true,
			SupportLevelAtTime:    SupportHigh,
			EngagementLevelAtTime: EngagementLow,
			ReadinessLevelAtTime:  ReadinessReadyMedium,
			EffectivenessStatus:   EffectivenessNotEvaluated,
		},
	}
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(6),
		BlockedActiveDays7d:    intPtr(6),
		RecoveryStreakDays:     intPtr(0),
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(0),
		ChangeReadiness:        &readiness,
		PreviousInterventions:  history,
	})

	assert.False(t, result.EffectivenessHistoryUsed)
	require.Contains(t, result.TriggeredRules, "history_found_NOT_EVALUATED=LIGHT_EDUCATION")
	assert.NotContains(t, result.TriggeredRules, "history_applied_")
}

func TestSpk_TimePattern_OutsideSevenDaysExcluded(t *testing.T) {
	cfg := DefaultConfig()
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	events := []time.Time{
		eventAt(1, 22, 30),
		eventAt(4, 22, 45),
		eventAt(5, 23, 15),
	}
	trigger := cfg.detectTimePattern(events, now)
	assert.Nil(t, trigger)

	engine := NewEngine(DefaultConfig())
	result := engine.EvaluateAt(Condition{
		BlockedAttemptsToday: intPtr(6),
		BlockedActiveDays7d:  intPtr(5),
		BlockedEventTimes:    events,
	}, now)
	assert.Nil(t, result.TimeTrigger)
}

func TestSpk_Fallback_EngagementGuardrail(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Fallback = InterventionShortRecoveryPractice
	cfg.Baseline[LevelPair{Support: SupportHigh, Engagement: EngagementLow}] = []InterventionKey{
		InterventionAccountabilityOption,
	}
	engine := NewEngine(cfg)
	readiness := ReadinessReadyMedium
	condition := Condition{
		BlockedAttemptsToday:   intPtr(6),
		BlockedActiveDays7d:    intPtr(6),
		RecoveryStreakDays:     intPtr(0),
		DailyMissionsCompleted: intPtr(0),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(0),
		ChangeReadiness:        &readiness,
		AccountabilityEnabled:  boolPtr(false),
	}

	result := engine.Evaluate(condition)
	assert.Equal(t, InterventionShortGrounding, result.InterventionKey)
	assert.Equal(t, ReasonFallback, result.ReasonCode)
	assert.True(t, result.InterventionNeeded)
	assert.LessOrEqual(t, DefaultConfig().KnowledgeBase[result.InterventionKey].Load, DefaultConfig().LowEngagementMaxLoad)

	compatibleCfg := DefaultConfig()
	compatibleCfg.Baseline[LevelPair{Support: SupportHigh, Engagement: EngagementLow}] = []InterventionKey{
		InterventionAccountabilityOption,
	}
	compatibleCfg.Fallback = InterventionLightEducation
	compatibleResult := NewEngine(compatibleCfg).Evaluate(condition)
	assert.Equal(t, InterventionLightEducation, compatibleResult.InterventionKey)
}

func TestSpk_InvalidInput_NegativeCountsNeutral(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(-3),
		BlockedActiveDays7d:    intPtr(-1),
		RecoveryStreakDays:     intPtr(-5),
		DailyMissionsCompleted: intPtr(-2),
		DailyMissionsTotal:     intPtr(-1),
		LearningActivities7d:   intPtr(-4),
	})

	assert.Equal(t, SupportLow, result.SupportLevel)
	assert.Equal(t, EngagementMedium, result.EngagementLevel)
	assert.Equal(t, InterventionLightEducation, result.InterventionKey)
	assert.True(t, result.InterventionNeeded)
	require.Contains(t, result.UnavailableFields, "blocked_attempts_today")
	require.Contains(t, result.UnavailableFields, "recovery_streak_days")
	require.Contains(t, result.UnavailableFields, "daily_missions_completed")
	require.Contains(t, result.TriggeredRules, "invalid_input=blocked_attempts_today")
	require.Contains(t, result.TriggeredRules, "insufficient_data_normalized")
}

func TestSpk_InvalidReadiness_NotReadyLow(t *testing.T) {
	engine := NewEngine(DefaultConfig())
	invalid := ChangeReadiness("BINGO")
	result := engine.Evaluate(Condition{
		BlockedAttemptsToday:   intPtr(5),
		BlockedActiveDays7d:    intPtr(5),
		RecoveryStreakDays:     intPtr(8),
		DailyMissionsCompleted: intPtr(5),
		DailyMissionsTotal:     intPtr(5),
		LearningActivities7d:   intPtr(3),
		ChangeReadiness:        &invalid,
	})

	assert.Equal(t, SupportMedium, result.SupportLevel)
	assert.Equal(t, EngagementHigh, result.EngagementLevel)
	assert.Equal(t, InterventionShortRecoveryPractice, result.InterventionKey)
	require.Contains(t, result.UnavailableFields, "change_readiness")
	readinessComponent := findComponent(result.ScoreBreakdown.Components, KeyReadiness)
	require.NotNil(t, readinessComponent)
	assert.False(t, readinessComponent.Available)
	assert.NotContains(t, result.TriggeredRules, "readiness_low_light_priority")
}

func TestSpk_KnowledgeBase_RenamedKeysPreserveBehavior(t *testing.T) {
	kb := DefaultConfig().KnowledgeBase
	baseline := DefaultConfig().Baseline

	assert.Equal(t, "SHORT_GROUNDING", string(InterventionShortGrounding))
	assert.Equal(t, 1, kb[InterventionShortGrounding].Load)
	assert.Equal(t, ResponseTypeGrounding, kb[InterventionShortGrounding].ResponseType)

	assert.Equal(t, "SHORT_RECOVERY_PRACTICE", string(InterventionShortRecoveryPractice))
	assert.Equal(t, 4, kb[InterventionShortRecoveryPractice].Load)
	assert.Equal(t, ResponseTypeRecoveryPractice, kb[InterventionShortRecoveryPractice].ResponseType)

	assert.Equal(t, []InterventionKey{InterventionShortRecoveryPractice},
		baseline[LevelPair{Support: SupportMedium, Engagement: EngagementHigh}])
	assert.Equal(t, []InterventionKey{InterventionShortGrounding},
		baseline[LevelPair{Support: SupportMedium, Engagement: EngagementLow}])
	assert.Equal(t, []InterventionKey{InterventionShortRecoveryPractice, InterventionAlternativeActivity, InterventionShortGrounding},
		baseline[LevelPair{Support: SupportHigh, Engagement: EngagementHigh}])
}

func findComponent(components []ScoreComponent, key string) *ScoreComponent {
	for i := range components {
		if components[i].Key == key {
			return &components[i]
		}
	}
	return nil
}
