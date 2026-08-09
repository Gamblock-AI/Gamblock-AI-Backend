package spk

// EffectivenessInput carries one past intervention and the blocked-attempt
// counts observed in the days before and after it. BeforeBlocked and
// AfterBlocked hold one entry per day, aligned with EffectivenessWindowDays.
type EffectivenessInput struct {
	InterventionKey InterventionKey
	Completed       bool
	BeforeBlocked   []int
	AfterBlocked    []int
}

// EvaluateEffectiveness classifies an intervention using a
// EffectivenessWindowDays-day observation window on each side:
//
//   - not completed                         -> NOT_EVALUATED
//   - either window incomplete              -> UNCLEAR
//   - complete BEFORE window totals zero    -> UNCLEAR
//   - blocked attempts fell by              -> EFFECTIVE
//     EffectivenessDecreasePercent or more
//   - otherwise                             -> LESS_EFFECTIVE
//
// When a side holds more entries than the configured window, exactly the first
// EffectivenessWindowDays entries are used.
func (c Config) EvaluateEffectiveness(input EffectivenessInput) EffectivenessStatus {
	if !input.Completed {
		return EffectivenessNotEvaluated
	}
	if len(input.BeforeBlocked) < c.EffectivenessWindowDays || len(input.AfterBlocked) < c.EffectivenessWindowDays {
		return EffectivenessUnclear
	}
	before := sumFirstN(input.BeforeBlocked, c.EffectivenessWindowDays)
	after := sumFirstN(input.AfterBlocked, c.EffectivenessWindowDays)
	if before == 0 {
		return EffectivenessUnclear
	}
	if float64(before-after)/float64(before) >= c.EffectivenessDecreasePercent/100 {
		return EffectivenessEffective
	}
	return EffectivenessLessEffective
}

func sumFirstN(values []int, n int) int {
	total := 0
	if n > len(values) {
		n = len(values)
	}
	for i := 0; i < n; i++ {
		total += values[i]
	}
	return total
}
