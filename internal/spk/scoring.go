package spk

const (
	KeyBlockedToday       = "blocked_attempts_today"
	KeyBlockedPersistence = "blocked_active_days_7d"
	KeyRecoveryStreak     = "recovery_streak_days"
	KeyDailyMissions      = "daily_missions_completed"
	KeyLearning           = "learning_activities_7d"
	KeyReadiness          = "change_readiness"
)

type scoreResult struct {
	key   string
	score int
}

type supportEvaluation struct {
	score           float64
	level           SupportLevel
	components      []ScoreComponent
	streak          *scoreResult
	mission         *scoreResult
	learning        *scoreResult
	availableWeight float64
	totalWeight     float64
}

func (c Config) scoreBlockedToday(value int) int {
	switch {
	case value <= c.BlockedTodayScore0Max:
		return 0
	case value <= c.BlockedTodayScore1Max:
		return 1
	default:
		return 2
	}
}

func (c Config) scoreBlockedPersistence(value int) int {
	switch {
	case value <= c.BlockedDaysScore0Max:
		return 0
	case value <= c.BlockedDaysScore1Max:
		return 1
	default:
		return 2
	}
}

func (c Config) scoreRecoveryStreak(value int) int {
	switch {
	case value >= c.StreakScore0Min:
		return 0
	case value >= c.StreakScore1Min:
		return 1
	default:
		return 2
	}
}

func (c Config) scoreMissions(completed, total int) int {
	if total <= 0 {
		total = c.DefaultMissionTotal
	}
	ratio := float64(completed) / float64(total)
	switch {
	case ratio >= c.MissionHighRatio:
		return 0
	case ratio >= c.MissionMediumRatio:
		return 1
	default:
		return 2
	}
}

func (c Config) scoreLearning(value int) int {
	switch {
	case value >= c.LearningScore0Min:
		return 0
	case value >= c.LearningScore1Min:
		return 1
	default:
		return 2
	}
}

func (c Config) scoreReadiness(readiness ChangeReadiness) int {
	switch readiness {
	case ReadinessReadyFirm, ReadinessReadyHigh:
		return 0
	case ReadinessReadyMedium:
		return 1
	default:
		return 2
	}
}

// validCount reports whether a count field was provided and is non-negative.
// Negative counts are treated as invalid and therefore unavailable.
func validCount(value *int) bool {
	return value != nil && *value >= 0
}

// evaluateSupport computes the weighted support score. Unavailable factors are
// excluded from the raw score and from the available weight, and the final
// score is re-normalized to 0-100 from the available weight only.
func (c Config) evaluateSupport(cond Condition) supportEvaluation {
	totalWeight := c.WeightBlockedToday +
		c.WeightBlockedPersistence +
		c.WeightRecoveryStreak +
		c.WeightDailyMissions +
		c.WeightLearning +
		c.WeightReadiness

	components := make([]ScoreComponent, 0, 6)
	var raw, available float64
	var streak, mission, learning *scoreResult

	push := func(key string, weight float64, score int, isAvailable bool) {
		contribution := 0.0
		if isAvailable {
			contribution = (float64(score) / 2.0) * weight
			raw += contribution
			available += weight
		}
		components = append(components, ScoreComponent{
			Key:           key,
			WeightPercent: weight,
			Score:         score,
			Contribution:  contribution,
			Available:     isAvailable,
		})
	}

	if validCount(cond.BlockedAttemptsToday) {
		push(KeyBlockedToday, c.WeightBlockedToday, c.scoreBlockedToday(*cond.BlockedAttemptsToday), true)
	} else {
		push(KeyBlockedToday, c.WeightBlockedToday, 0, false)
	}
	if validCount(cond.BlockedActiveDays7d) {
		push(KeyBlockedPersistence, c.WeightBlockedPersistence, c.scoreBlockedPersistence(*cond.BlockedActiveDays7d), true)
	} else {
		push(KeyBlockedPersistence, c.WeightBlockedPersistence, 0, false)
	}
	if validCount(cond.RecoveryStreakDays) {
		scored := c.scoreRecoveryStreak(*cond.RecoveryStreakDays)
		streak = &scoreResult{key: KeyRecoveryStreak, score: scored}
		push(KeyRecoveryStreak, c.WeightRecoveryStreak, scored, true)
	} else {
		push(KeyRecoveryStreak, c.WeightRecoveryStreak, 0, false)
	}
	missionTotal := c.DefaultMissionTotal
	if validCount(cond.DailyMissionsTotal) && *cond.DailyMissionsTotal > 0 {
		missionTotal = *cond.DailyMissionsTotal
	}
	if validCount(cond.DailyMissionsCompleted) {
		scored := c.scoreMissions(*cond.DailyMissionsCompleted, missionTotal)
		mission = &scoreResult{key: KeyDailyMissions, score: scored}
		push(KeyDailyMissions, c.WeightDailyMissions, scored, true)
	} else {
		push(KeyDailyMissions, c.WeightDailyMissions, 0, false)
	}
	if validCount(cond.LearningActivities7d) {
		scored := c.scoreLearning(*cond.LearningActivities7d)
		learning = &scoreResult{key: KeyLearning, score: scored}
		push(KeyLearning, c.WeightLearning, scored, true)
	} else {
		push(KeyLearning, c.WeightLearning, 0, false)
	}
	if cond.ChangeReadiness != nil && isValidReadiness(*cond.ChangeReadiness) {
		push(KeyReadiness, c.WeightReadiness, c.scoreReadiness(*cond.ChangeReadiness), true)
	} else {
		push(KeyReadiness, c.WeightReadiness, 0, false)
	}

	score := 0.0
	if available > 0 {
		score = raw / available * 100
	}
	return supportEvaluation{
		score:           score,
		level:           c.supportLevel(score),
		components:      components,
		streak:          streak,
		mission:         mission,
		learning:        learning,
		availableWeight: available,
		totalWeight:     totalWeight,
	}
}
