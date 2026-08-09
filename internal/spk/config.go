package spk

import "sort"

// InterventionMeta describes one knowledge-base intervention. Load ranks how
// much commitment an intervention asks of the user (lower is lighter); it is
// used to keep selections consistent with the user's engagement level.
type InterventionMeta struct {
	Key                 InterventionKey
	ResponseType        ResponseType
	Load                int
	NeedsAccountability bool
}

// LevelPair keys the baseline support x engagement rule table.
type LevelPair struct {
	Support    SupportLevel
	Engagement EngagementLevel
}

// Config holds every weight, threshold, and rule of the engine. Zero fields
// fall back to DefaultConfig values; start from DefaultConfig() and override
// only what you need.
type Config struct {
	WeightBlockedToday       float64
	WeightBlockedPersistence float64
	WeightRecoveryStreak     float64
	WeightDailyMissions      float64
	WeightLearning           float64
	WeightReadiness          float64

	BlockedTodayScore0Max int
	BlockedTodayScore1Max int
	BlockedDaysScore0Max  int
	BlockedDaysScore1Max  int
	StreakScore0Min       int
	StreakScore1Min       int
	MissionHighRatio      float64
	MissionMediumRatio    float64
	LearningScore0Min     int
	LearningScore1Min     int
	DefaultMissionTotal   int

	SupportLowMax    float64
	SupportMediumMax float64

	EngagementRawMediumMin        int
	EngagementRawLowMin           int
	EngagementNormalizedMediumMin float64
	EngagementNormalizedLowMin    float64

	TimePatternWindowMinutes int
	TimePatternMinEvents     int
	TimeTriggerLeadMinutes   int
	TimePatternWindowDays    int

	EffectivenessWindowDays      int
	EffectivenessDecreasePercent float64

	KnowledgeBase map[InterventionKey]InterventionMeta
	Baseline      map[LevelPair][]InterventionKey
	Fallback      InterventionKey

	LowEngagementMaxLoad    int
	MediumEngagementMaxLoad int

	ReadinessLowLightKeys   []InterventionKey
	ReadinessLowFocusKey    InterventionKey
	ReadinessLowLightDelta  float64
	ReadinessLowFocusDelta  float64
	ReadinessHighFocusKeys  []InterventionKey
	ReadinessHighFocusDelta float64

	HistoryEffectiveDelta     float64
	HistoryLessEffectiveDelta float64
}

// DefaultConfig returns the specification baseline: weights sum to 100 and
// every threshold matches the documented scoring, engagement, time-pattern,
// and effectiveness rules.
func DefaultConfig() Config {
	return Config{
		WeightBlockedToday:       20,
		WeightBlockedPersistence: 20,
		WeightRecoveryStreak:     20,
		WeightDailyMissions:      15,
		WeightLearning:           10,
		WeightReadiness:          15,

		BlockedTodayScore0Max: 1,
		BlockedTodayScore1Max: 4,
		BlockedDaysScore0Max:  1,
		BlockedDaysScore1Max:  3,
		StreakScore0Min:       7,
		StreakScore1Min:       3,
		MissionHighRatio:      0.8,
		MissionMediumRatio:    0.4,
		LearningScore0Min:     3,
		LearningScore1Min:     1,
		DefaultMissionTotal:   5,

		SupportLowMax:    39,
		SupportMediumMax: 69,

		EngagementRawMediumMin:        2,
		EngagementRawLowMin:           4,
		EngagementNormalizedMediumMin: 1,
		EngagementNormalizedLowMin:    3,

		TimePatternWindowMinutes: 120,
		TimePatternMinEvents:     3,
		TimeTriggerLeadMinutes:   30,
		TimePatternWindowDays:    7,

		EffectivenessWindowDays:      2,
		EffectivenessDecreasePercent: 30,

		KnowledgeBase: defaultKnowledgeBase(),
		Baseline:      defaultBaseline(),
		Fallback:      InterventionShortReflection,

		LowEngagementMaxLoad:    2,
		MediumEngagementMaxLoad: 4,

		ReadinessLowLightKeys: []InterventionKey{
			InterventionShortGrounding,
			InterventionAlternativeActivity,
			InterventionLightEducation,
		},
		ReadinessLowFocusKey:    InterventionShortRecoveryPractice,
		ReadinessLowLightDelta:  5,
		ReadinessLowFocusDelta:  3,
		ReadinessHighFocusKeys:  []InterventionKey{InterventionShortRecoveryPractice},
		ReadinessHighFocusDelta: 1,

		HistoryEffectiveDelta:     5,
		HistoryLessEffectiveDelta: 5,
	}
}

func defaultKnowledgeBase() map[InterventionKey]InterventionMeta {
	return map[InterventionKey]InterventionMeta{
		InterventionNoIntervention: {
			Key: InterventionNoIntervention, ResponseType: ResponseTypeAppreciation, Load: 0,
		},
		InterventionLightEducation: {
			Key: InterventionLightEducation, ResponseType: ResponseTypeEducation, Load: 2,
		},
		InterventionShortRecoveryPractice: {
			Key: InterventionShortRecoveryPractice, ResponseType: ResponseTypeRecoveryPractice, Load: 4,
		},
		InterventionShortGrounding: {
			Key: InterventionShortGrounding, ResponseType: ResponseTypeGrounding, Load: 1,
		},
		InterventionShortReflection: {
			Key: InterventionShortReflection, ResponseType: ResponseTypeReflection, Load: 3,
		},
		InterventionAlternativeActivity: {
			Key: InterventionAlternativeActivity, ResponseType: ResponseTypeActivity, Load: 1,
		},
		InterventionAccountabilityOption: {
			Key: InterventionAccountabilityOption, ResponseType: ResponseTypeAccountability, Load: 3,
			NeedsAccountability: true,
		},
	}
}

func defaultBaseline() map[LevelPair][]InterventionKey {
	return map[LevelPair][]InterventionKey{
		{Support: SupportLow, Engagement: EngagementHigh}: {
			InterventionNoIntervention,
		},
		{Support: SupportLow, Engagement: EngagementMedium}: {
			InterventionLightEducation,
		},
		{Support: SupportLow, Engagement: EngagementLow}: {
			InterventionAlternativeActivity,
		},
		{Support: SupportMedium, Engagement: EngagementHigh}: {
			InterventionShortRecoveryPractice,
		},
		{Support: SupportMedium, Engagement: EngagementMedium}: {
			InterventionShortRecoveryPractice,
		},
		{Support: SupportMedium, Engagement: EngagementLow}: {
			InterventionShortGrounding,
		},
		{Support: SupportHigh, Engagement: EngagementHigh}: {
			InterventionShortRecoveryPractice,
			InterventionAlternativeActivity,
			InterventionShortGrounding,
		},
		{Support: SupportHigh, Engagement: EngagementMedium}: {
			InterventionShortGrounding,
			InterventionAlternativeActivity,
			InterventionShortRecoveryPractice,
		},
		{Support: SupportHigh, Engagement: EngagementLow}: {
			InterventionShortGrounding,
			InterventionAlternativeActivity,
			InterventionLightEducation,
		},
	}
}

// completed fills only the collection fields that are required for the engine
// to run. Scalar weights and thresholds are honored as provided, so setting a
// weight to zero intentionally disables that factor.
func (c Config) completed() Config {
	base := DefaultConfig()
	if c.KnowledgeBase == nil {
		c.KnowledgeBase = base.KnowledgeBase
	}
	if c.Baseline == nil {
		c.Baseline = base.Baseline
	}
	if c.Fallback == "" {
		c.Fallback = base.Fallback
	}
	if len(c.ReadinessLowLightKeys) == 0 {
		c.ReadinessLowLightKeys = base.ReadinessLowLightKeys
	}
	if c.ReadinessLowFocusKey == "" {
		c.ReadinessLowFocusKey = base.ReadinessLowFocusKey
	}
	if len(c.ReadinessHighFocusKeys) == 0 {
		c.ReadinessHighFocusKeys = base.ReadinessHighFocusKeys
	}
	return c
}

func (c Config) supportLevel(score float64) SupportLevel {
	switch {
	case score <= c.SupportLowMax:
		return SupportLow
	case score <= c.SupportMediumMax:
		return SupportMedium
	default:
		return SupportHigh
	}
}

func (c Config) engagementAllows(load int, engagement EngagementLevel) bool {
	switch engagement {
	case EngagementLow:
		return load <= c.LowEngagementMaxLoad
	case EngagementMedium:
		return load <= c.MediumEngagementMaxLoad
	default:
		return true
	}
}

func accountabilityOn(cond Condition) bool {
	return cond.AccountabilityEnabled != nil && *cond.AccountabilityEnabled
}

func isValidReadiness(readiness ChangeReadiness) bool {
	switch readiness {
	case ReadinessReadyFirm, ReadinessReadyHigh, ReadinessReadyMedium, ReadinessReadyLow:
		return true
	default:
		return false
	}
}

// resolveFallback picks an intervention when the candidate pool is empty. It
// never bypasses the engagement load guardrail: the configured Fallback is used
// only when valid and engagement-compatible, otherwise the configured light
// candidates are tried, then the lightest compatible knowledge-base entry.
func (c Config) resolveFallback(cond Condition, engagement EngagementLevel) (InterventionKey, InterventionMeta) {
	if meta, ok := c.KnowledgeBase[c.Fallback]; ok {
		if (!meta.NeedsAccountability || accountabilityOn(cond)) && c.engagementAllows(meta.Load, engagement) {
			return c.Fallback, meta
		}
	}
	for _, key := range c.ReadinessLowLightKeys {
		meta, ok := c.KnowledgeBase[key]
		if !ok {
			continue
		}
		if meta.NeedsAccountability && !accountabilityOn(cond) {
			continue
		}
		if !c.engagementAllows(meta.Load, engagement) {
			continue
		}
		return key, meta
	}
	keys := make([]InterventionKey, 0, len(c.KnowledgeBase))
	for key := range c.KnowledgeBase {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := c.KnowledgeBase[keys[i]]
		right := c.KnowledgeBase[keys[j]]
		if left.Load != right.Load {
			return left.Load < right.Load
		}
		return keys[i] < keys[j]
	})
	for _, key := range keys {
		meta := c.KnowledgeBase[key]
		if meta.NeedsAccountability && !accountabilityOn(cond) {
			continue
		}
		if !c.engagementAllows(meta.Load, engagement) {
			continue
		}
		return key, meta
	}
	return InterventionNoIntervention, InterventionMeta{
		Key: InterventionNoIntervention, ResponseType: ResponseTypeAppreciation, Load: 0,
	}
}
