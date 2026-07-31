package model

import "time"

type DailyMission struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Date   string `json:"date"`
	// Legacy flags are retained only to read existing in-memory/demo data and
	// older database rows. New mission state lives in TaskRecords.
	Mission1           bool                `json:"-"`
	Mission2           bool                `json:"-"`
	Mission3           bool                `json:"-"`
	Mission4           bool                `json:"-"`
	Mission5           bool                `json:"-"`
	Mission6           bool                `json:"-"`
	Tasks              []DailyMissionTask  `json:"tasks"`
	TaskRecords        []MissionRecord     `json:"-"`
	Experience         ExperienceProgress  `json:"experience"`
	CompletedCount     int                 `json:"completed_count"`
	ResolvedCount      int                 `json:"resolved_count"`
	TotalCount         int                 `json:"total_count"`
	Adjustment         *MissionAdjustment  `json:"-"`
	ReplacementOptions []int               `json:"-"`
	AdjustmentHistory  []MissionAdjustment `json:"-"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type DailyMissionTask struct {
	ID              string `json:"id"`
	Number          int    `json:"number,omitempty"`
	Key             string `json:"key"`
	Role            string `json:"role,omitempty"`
	Source          string `json:"source"`
	SystemKey       string `json:"system_key,omitempty"`
	Title           string `json:"title,omitempty"`
	Completed       bool   `json:"completed"`
	Claimable       bool   `json:"claimable"`
	Status          string `json:"status"`
	ClaimMode       string `json:"claim_mode"`
	VerificationKey string `json:"verification_key,omitempty"`
	EXPReward       int    `json:"exp_reward"`
	ReplacedFrom    int    `json:"replaced_from,omitempty"`
}

type MissionRecord struct {
	ID               string
	Key              string
	Source           string
	TitleEncrypted   string
	Status           string
	Reward           int
	AdjustmentReason string
	CompletedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// MissionAdjustment remains a read-only compatibility type for historical
// records created before the skip action was retired.
type MissionAdjustment struct {
	OriginalNumber    int       `json:"original_number"`
	Action            string    `json:"action"`
	Reason            string    `json:"reason"`
	ReplacementNumber int       `json:"replacement_number,omitempty"`
	AdjustedAt        time.Time `json:"adjusted_at"`
}

func (mission DailyMission) CompletedTaskCount() int {
	if len(mission.TaskRecords) > 0 {
		count := 0
		for _, task := range mission.TaskRecords {
			if task.Status == "completed" {
				count++
			}
		}
		return count
	}
	count := 0
	for _, completed := range []bool{mission.Mission1, mission.Mission2, mission.Mission3, mission.Mission4, mission.Mission5, mission.Mission6} {
		if completed {
			count++
		}
	}
	return count
}

// SystemCompletedTaskCount intentionally excludes self-attested custom
// missions from partner-facing aggregates.
func (mission DailyMission) SystemCompletedTaskCount() int {
	if len(mission.TaskRecords) == 0 {
		count := 0
		for _, completed := range []bool{mission.Mission1, mission.Mission2, mission.Mission3, mission.Mission4, mission.Mission5, mission.Mission6} {
			if completed {
				count++
			}
		}
		return count
	}

	count := 0
	recorded := make(map[string]struct{}, len(mission.TaskRecords))
	for _, task := range mission.TaskRecords {
		if task.Source != "system" {
			continue
		}
		recorded[task.Key] = struct{}{}
		if task.Status == "completed" {
			count++
		}
	}
	for key, completed := range map[string]bool{
		"mission_1": mission.Mission1, "mission_2": mission.Mission2,
		"mission_3": mission.Mission3, "mission_4": mission.Mission4,
		"mission_5": mission.Mission5, "mission_6": mission.Mission6,
	} {
		if completed {
			if _, exists := recorded[key]; !exists {
				count++
			}
		}
	}
	return count
}

type ExperienceProgress struct {
	TotalEXP      int      `json:"total_exp"`
	Level         int      `json:"level"`
	LevelProgress int      `json:"level_progress"`
	LevelTarget   int      `json:"level_target"`
	NewlyUnlocked []string `json:"newly_unlocked,omitempty"`
}
