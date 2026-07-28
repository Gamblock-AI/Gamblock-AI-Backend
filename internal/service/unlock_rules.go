package service

import "github.com/gamblock-ai/gamblock-ai-backend/internal/model"

// Recovery-space progression rules, version 2.
//
// Every rule is deterministic, additive, and its criterion is shown verbatim
// in the clients (no mystery boxes). Version 1 rules are preserved unchanged;
// unlock state is always the union of stored and computed items, so accounts
// keep everything they ever unlocked when rules evolve.
const recoveryUnlockRuleVersion = 2

type decorUnlockRule struct {
	Item     string
	Unlocked func(evidence model.RecoveryUnlockEvidence) bool
}

func evidenceLevel(evidence model.RecoveryUnlockEvidence) int {
	return max(0, evidence.ExperiencePoints)/experiencePerLevel + 1
}

var decorUnlockRules = []decorUnlockRule{
	// Version 1 rules (unchanged).
	{"plant", func(e model.RecoveryUnlockEvidence) bool { return e.PracticeKinds["grounding_54321"] }},
	{"curtain", func(e model.RecoveryUnlockEvidence) bool { return e.PracticeKinds["urge_surfing"] }},
	{"desk_lamp", func(e model.RecoveryUnlockEvidence) bool { return e.PracticeKinds["focus_sprint"] }},
	{"note_board", func(e model.RecoveryUnlockEvidence) bool { return e.HasFocusJournal }},
	{"wall_art", func(e model.RecoveryUnlockEvidence) bool { return e.HasWeeklyReview }},
	{"gami_figure", func(e model.RecoveryUnlockEvidence) bool { return e.ActiveDays >= 5 }},
	// Version 2: tiered practice/participation rewards.
	{"cushion", func(e model.RecoveryUnlockEvidence) bool { return e.TotalPractices >= 5 }},
	{"zen_tray", func(e model.RecoveryUnlockEvidence) bool { return e.TotalPractices >= 15 }},
	{"fountain_mini", func(e model.RecoveryUnlockEvidence) bool { return e.TotalPractices >= 30 }},
	{"photo_frame", func(e model.RecoveryUnlockEvidence) bool { return e.ActiveDays >= 10 }},
	{"wall_clock", func(e model.RecoveryUnlockEvidence) bool { return e.ActiveDays >= 25 }},
	{"calendar_wall", func(e model.RecoveryUnlockEvidence) bool { return e.WeeklyReviews >= 3 }},
	{"desk_organizer", func(e model.RecoveryUnlockEvidence) bool { return e.MissionsClaimed >= 10 }},
	// Version 2: level-gated rewards.
	{"poster_calm", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 2 }},
	{"mug_warm", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 3 }},
	{"rug_soft", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 4 }},
	{"bookshelf_mini", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 6 }},
	{"string_lights", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 8 }},
	{"radio_lofi", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 12 }},
	{"aquarium_mini", func(e model.RecoveryUnlockEvidence) bool { return evidenceLevel(e) >= 16 }},
}

// levelDecorUnlocks powers the "newly unlocked" list in the mission-claim
// response so clients can show a calm level-up moment without another fetch.
var levelDecorUnlocks = map[int][]string{
	2:  {"poster_calm"},
	3:  {"mug_warm"},
	4:  {"rug_soft"},
	6:  {"bookshelf_mini"},
	8:  {"string_lights"},
	12: {"radio_lofi"},
	16: {"aquarium_mini"},
}

// decorSlots lists the placement slots allowed per item; the first entry is
// the default used when a placement value is `true`.
var decorSlots = map[string][]string{
	"plant":          {"floor_left", "window_sill", "shelf"},
	"curtain":        {"window_sill"},
	"desk_lamp":      {"desk", "shelf"},
	"note_board":     {"wall_left", "wall_center", "wall_right"},
	"wall_art":       {"wall_center", "wall_left", "wall_right"},
	"gami_figure":    {"shelf", "desk", "window_sill"},
	"cushion":        {"floor_center", "floor_left", "floor_right"},
	"zen_tray":       {"desk", "shelf"},
	"fountain_mini":  {"shelf", "window_sill"},
	"photo_frame":    {"desk", "shelf", "wall_right"},
	"wall_clock":     {"wall_right", "wall_left"},
	"calendar_wall":  {"wall_left", "wall_right"},
	"desk_organizer": {"desk"},
	"poster_calm":    {"wall_right", "wall_left", "wall_center"},
	"mug_warm":       {"desk", "shelf"},
	"rug_soft":       {"floor_center"},
	"bookshelf_mini": {"floor_right", "floor_left"},
	"string_lights":  {"wall_center", "window_sill"},
	"radio_lofi":     {"shelf", "desk"},
	"aquarium_mini":  {"shelf", "desk"},
}

func unlockedThemes(evidence model.RecoveryUnlockEvidence) []string {
	themes := []string{"dorm_room"}
	if evidenceLevel(evidence) >= 18 {
		themes = append(themes, "sunrise_study")
	}
	return themes
}

func newlyUnlockedDecor(levelBefore, levelAfter int) []string {
	if levelAfter <= levelBefore {
		return nil
	}
	var items []string
	for level := levelBefore + 1; level <= levelAfter; level++ {
		items = append(items, levelDecorUnlocks[level]...)
	}
	return items
}
