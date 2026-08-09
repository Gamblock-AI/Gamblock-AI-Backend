package spk

type engagementEvaluation struct {
	level            EngagementLevel
	raw              int
	factorsAvailable int
	normalizedRaw    float64
}

// evaluateEngagement derives the engagement level from the streak, mission,
// and learning scores.
//
//   - all three factors available: raw sum maps 0-1 HIGH, 2-3 MEDIUM, 4-6 LOW
//   - partially available: normalized (raw / (available*2)) * 6 maps 0-1 HIGH,
//     >1-3 MEDIUM, >3-6 LOW
//   - nothing available: neutral MEDIUM, so missing data is never treated as
//     good or bad engagement
func (c Config) evaluateEngagement(support supportEvaluation) engagementEvaluation {
	raw := 0
	available := 0
	for _, result := range []*scoreResult{support.streak, support.mission, support.learning} {
		if result != nil {
			raw += result.score
			available++
		}
	}
	normalized := 0.0
	if available > 0 {
		normalized = (float64(raw) / (float64(available) * 2.0)) * 6.0
	}

	level := EngagementMedium
	switch {
	case available == 0:
		// Neutral fallback for entirely missing engagement data.
	case available == 3:
		switch {
		case raw >= c.EngagementRawLowMin:
			level = EngagementLow
		case raw >= c.EngagementRawMediumMin:
			level = EngagementMedium
		default:
			level = EngagementHigh
		}
	default:
		switch {
		case normalized > c.EngagementNormalizedLowMin:
			level = EngagementLow
		case normalized > c.EngagementNormalizedMediumMin:
			level = EngagementMedium
		default:
			level = EngagementHigh
		}
	}
	return engagementEvaluation{
		level:            level,
		raw:              raw,
		factorsAvailable: available,
		normalizedRaw:    normalized,
	}
}
