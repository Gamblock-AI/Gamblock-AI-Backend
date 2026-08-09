package spk

import (
	"fmt"
	"sort"
	"time"
)

// detectTimePattern finds a 2-hour clock window that contains at least
// TimePatternMinEvents blocked events from the last TimePatternWindowDays days
// relative to now. When found it returns the pattern and a trigger window that
// starts TimeTriggerLeadMinutes before the pattern start. Clock times wrap at
// midnight; no assumption is made about which part of the day is riskier.
func (c Config) detectTimePattern(events []time.Time, now time.Time) *TimeTrigger {
	cutoff := now.AddDate(0, 0, -c.TimePatternWindowDays)
	filtered := make([]time.Time, 0, len(events))
	for _, event := range events {
		if event.After(cutoff) && !event.After(now) {
			filtered = append(filtered, event)
		}
	}
	if len(filtered) < c.TimePatternMinEvents {
		return nil
	}
	minutes := make([]int, 0, len(filtered))
	for _, event := range filtered {
		minutes = append(minutes, event.Hour()*60+event.Minute())
	}
	sort.Ints(minutes)

	n := len(minutes)
	extended := make([]int, 0, 2*n)
	for _, minute := range minutes {
		extended = append(extended, minute)
	}
	for _, minute := range minutes {
		extended = append(extended, minute+24*60)
	}

	bestStart := 0
	bestCount := 0
	for i := 0; i < n; i++ {
		start := extended[i]
		count := 0
		for j := i; j < 2*n && extended[j] <= start+c.TimePatternWindowMinutes; j++ {
			count++
		}
		if count > bestCount {
			bestCount = count
			bestStart = start
		}
	}
	if bestCount < c.TimePatternMinEvents {
		return nil
	}

	patternStart := bestStart % (24 * 60)
	patternEnd := (bestStart + c.TimePatternWindowMinutes) % (24 * 60)
	triggerStart := (patternStart - c.TimeTriggerLeadMinutes + 24*60) % (24 * 60)
	return &TimeTrigger{
		HasTimePattern: true,
		PatternStart:   minutesToString(patternStart),
		PatternEnd:     minutesToString(patternEnd),
		TriggerStart:   minutesToString(triggerStart),
		TriggerEnd:     minutesToString(patternEnd),
	}
}

func minutesToString(minutes int) string {
	minute := minutes % (24 * 60)
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}
