package model

import "time"

type DailyMission struct {
	ID                 string              `json:"id"`
	UserID             string              `json:"user_id"`
	Date               string              `json:"date"`
	Mission1           bool                `json:"mission_1"`
	Mission2           bool                `json:"mission_2"`
	Mission3           bool                `json:"mission_3"`
	Mission4           bool                `json:"mission_4"`
	Mission5           bool                `json:"mission_5"`
	Tasks              []DailyMissionTask  `json:"tasks"`
	Experience         ExperienceProgress  `json:"experience"`
	CompletedCount     int                 `json:"completed_count"`
	ResolvedCount      int                 `json:"resolved_count"`
	TotalCount         int                 `json:"total_count"`
	Adjustment         *MissionAdjustment  `json:"adjustment,omitempty"`
	ReplacementOptions []int               `json:"replacement_options"`
	AdjustmentHistory  []MissionAdjustment `json:"-"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type DailyMissionTask struct {
	Number          int    `json:"number"`
	Key             string `json:"key"`
	Role            string `json:"role"`
	Completed       bool   `json:"completed"`
	Claimable       bool   `json:"claimable"`
	Status          string `json:"status"`
	VerificationKey string `json:"verification_key"`
	EXPReward       int    `json:"exp_reward"`
	ReplacedFrom    int    `json:"replaced_from,omitempty"`
}

type MissionAdjustment struct {
	OriginalNumber    int       `json:"original_number"`
	Action            string    `json:"action"`
	Reason            string    `json:"reason"`
	ReplacementNumber int       `json:"replacement_number,omitempty"`
	AdjustedAt        time.Time `json:"adjusted_at"`
}

type ExperienceProgress struct {
	TotalEXP      int `json:"total_exp"`
	Level         int `json:"level"`
	LevelProgress int `json:"level_progress"`
	LevelTarget   int `json:"level_target"`
}
