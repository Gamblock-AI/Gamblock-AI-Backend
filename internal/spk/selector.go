package spk

import "sort"

type selection struct {
	key            InterventionKey
	responseType   ResponseType
	reasonCode     ReasonCode
	needed         bool
	historyUsed    bool
	appliedHistory []InterventionRecord
}

// selectIntervention applies the decision priority chain:
//
//  1. fit with the current engagement level
//  2. effective-intervention history on similar conditions
//  3. support level (baseline rule)
//  4. readiness modifier
//  5. default knowledge-base mapping
//
// Engagement compatibility is enforced before any history or readiness boost,
// so an intervention is never picked just because it worked before when it is
// too heavy for the user's current engagement.
func (c Config) selectIntervention(cond Condition, support SupportLevel, engagement EngagementLevel, readiness *ChangeReadiness) selection {
	readinessValue := ChangeReadiness("")
	if readiness != nil {
		readinessValue = *readiness
	}

	pool, baseOrder := c.candidatePool(cond, support, engagement, readinessValue)
	if len(pool) == 0 {
		key, meta := c.resolveFallback(cond, engagement)
		return selection{
			key:          key,
			responseType: meta.ResponseType,
			reasonCode:   ReasonFallback,
			needed:       key != InterventionNoIntervention,
		}
	}
	baselineFirst := pool[0]

	type ranked struct {
		key      InterventionKey
		base     int
		priority float64
	}
	items := make([]ranked, 0, len(pool))
	for i, key := range pool {
		items = append(items, ranked{key: key, base: baseOrder[i], priority: float64(baseOrder[i])})
	}

	switch readinessValue {
	case ReadinessReadyLow:
		for i := range items {
			if containsKey(c.ReadinessLowLightKeys, items[i].key) {
				items[i].priority -= c.ReadinessLowLightDelta
			}
			if items[i].key == c.ReadinessLowFocusKey {
				items[i].priority += c.ReadinessLowFocusDelta
			}
		}
	case ReadinessReadyHigh, ReadinessReadyFirm:
		if engagement == EngagementHigh {
			for i := range items {
				if containsKey(c.ReadinessHighFocusKeys, items[i].key) {
					items[i].priority -= c.ReadinessHighFocusDelta
				}
			}
		}
	}

	similar := c.similarHistory(cond, support, engagement, readinessValue)
	var appliedHistory []InterventionRecord
	for _, record := range similar {
		switch record.EffectivenessStatus {
		case EffectivenessEffective, EffectivenessLessEffective:
		default:
			continue
		}
		applied := false
		for i := range items {
			if items[i].key != record.InterventionKey {
				continue
			}
			applied = true
			if record.EffectivenessStatus == EffectivenessEffective {
				items[i].priority -= c.HistoryEffectiveDelta
			} else {
				items[i].priority += c.HistoryLessEffectiveDelta
			}
		}
		if applied {
			appliedHistory = append(appliedHistory, record)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].base < items[j].base
	})

	chosen := items[0].key
	meta := c.KnowledgeBase[chosen]
	return selection{
		key:            chosen,
		responseType:   meta.ResponseType,
		reasonCode:     c.reasonFor(chosen, baselineFirst, readinessValue, similar),
		needed:         chosen != InterventionNoIntervention,
		historyUsed:    len(appliedHistory) > 0,
		appliedHistory: appliedHistory,
	}
}

// candidatePool builds the engagement-compatible candidate list. Baseline
// candidates keep the knowledge-base order; for READY_LOW the light
// alternatives are appended so they can be promoted by the readiness modifier.
func (c Config) candidatePool(cond Condition, support SupportLevel, engagement EngagementLevel, readiness ChangeReadiness) ([]InterventionKey, []int) {
	baseline := c.Baseline[LevelPair{Support: support, Engagement: engagement}]
	if len(baseline) == 0 {
		baseline = []InterventionKey{c.Fallback}
	}

	seen := map[InterventionKey]bool{}
	pool := make([]InterventionKey, 0, len(baseline)+3)
	baseOrder := make([]int, 0, len(baseline)+3)
	add := func(key InterventionKey) {
		if seen[key] {
			return
		}
		meta, ok := c.KnowledgeBase[key]
		if !ok {
			return
		}
		if meta.NeedsAccountability && !accountabilityOn(cond) {
			return
		}
		if !c.engagementAllows(meta.Load, engagement) {
			return
		}
		seen[key] = true
		baseOrder = append(baseOrder, len(pool))
		pool = append(pool, key)
	}

	for _, key := range baseline {
		add(key)
	}
	if readiness == ReadinessReadyLow {
		for _, key := range c.ReadinessLowLightKeys {
			add(key)
		}
	}
	return pool, baseOrder
}

// similarHistory returns past interventions recorded under the same
// support/engagement/readiness triple.
func (c Config) similarHistory(cond Condition, support SupportLevel, engagement EngagementLevel, readiness ChangeReadiness) []InterventionRecord {
	var similar []InterventionRecord
	for _, record := range cond.PreviousInterventions {
		if record.SupportLevelAtTime == support &&
			record.EngagementLevelAtTime == engagement &&
			record.ReadinessLevelAtTime == readiness {
			similar = append(similar, record)
		}
	}
	return similar
}

func (c Config) reasonFor(chosen, baselineFirst InterventionKey, readiness ChangeReadiness, similar []InterventionRecord) ReasonCode {
	if chosen == InterventionNoIntervention {
		return ReasonNoIntervention
	}
	for _, record := range similar {
		if record.InterventionKey == chosen && record.EffectivenessStatus == EffectivenessEffective {
			return ReasonHistoryEffective
		}
	}
	if baselineFirst != chosen {
		for _, record := range similar {
			if record.InterventionKey == baselineFirst && record.EffectivenessStatus == EffectivenessLessEffective {
				return ReasonHistoryLessEffective
			}
		}
		if readiness == ReadinessReadyLow {
			return ReasonReadinessLow
		}
		if readiness == ReadinessReadyHigh || readiness == ReadinessReadyFirm {
			return ReasonReadinessHigh
		}
	}
	return ReasonBaseline
}

func containsKey(keys []InterventionKey, target InterventionKey) bool {
	for _, key := range keys {
		if key == target {
			return true
		}
	}
	return false
}
