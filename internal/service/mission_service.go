package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

const experiencePerLevel = 100

var jakartaLocation = time.FixedZone("Asia/Jakarta", 7*60*60)
var ErrMissionNotClaimable = errors.New("mission requirements are not verified")

type MissionService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

func NewMissionService(repo *repository.Repository, logger *zap.Logger) *MissionService {
	return &MissionService{repo: repo, logger: logger}
}

func (s *MissionService) GetToday(ctx context.Context, userID string) (model.DailyMission, error) {
	now := time.Now().In(jakartaLocation)
	date, dayStartUTC, dayEndUTC := jakartaDay(now)
	mission, points, err := s.repo.GetMissionByDate(ctx, userID, date, dayStartUTC, dayEndUTC)
	if err != nil {
		return model.DailyMission{}, err
	}
	eligibility, err := s.missionEligibility(ctx, userID, assignedMissions(now), dayStartUTC, dayEndUTC)
	if err != nil {
		return model.DailyMission{}, err
	}
	return decorateDailyMission(mission, points, now, eligibility), nil
}

func (s *MissionService) UpdateMission(
	ctx context.Context,
	userID string,
	missionNum int,
	completed bool,
) (model.DailyMission, error) {
	if !completed {
		return model.DailyMission{}, ErrMissionNotClaimable
	}
	return s.ClaimMission(ctx, userID, missionNum)
}

func (s *MissionService) ClaimMission(
	ctx context.Context,
	userID string,
	missionNum int,
) (model.DailyMission, error) {
	now := time.Now().In(jakartaLocation)
	assigned := assignedMissions(now)
	if !isAssignedMission(missionNum, assigned) {
		return model.DailyMission{}, fmt.Errorf("mission %d is not assigned today", missionNum)
	}
	date, dayStartUTC, dayEndUTC := jakartaDay(now)
	current, points, err := s.repo.GetMissionByDate(ctx, userID, date, dayStartUTC, dayEndUTC)
	if err != nil {
		return model.DailyMission{}, err
	}
	eligibility, err := s.missionEligibility(ctx, userID, assigned, dayStartUTC, dayEndUTC)
	if err != nil {
		return model.DailyMission{}, err
	}
	if missionFlag(current, missionNum) {
		return decorateDailyMission(current, points, now, eligibility), nil
	}
	if !eligibility[missionNum] {
		return model.DailyMission{}, ErrMissionNotClaimable
	}
	mission, points, err := s.repo.UpsertMission(
		ctx,
		userID,
		date,
		dayStartUTC,
		dayEndUTC,
		missionNum,
		true,
		missionReward(missionNum),
	)
	if err != nil {
		return model.DailyMission{}, err
	}
	return decorateDailyMission(mission, points, now, eligibility), nil
}

func (s *MissionService) missionEligibility(
	ctx context.Context,
	userID string,
	assigned [3]int,
	dayStartUTC, dayEndUTC time.Time,
) (map[int]bool, error) {
	eligibility := make(map[int]bool, len(assigned))
	for _, number := range assigned {
		claimable, err := s.repo.IsMissionClaimable(ctx, userID, number, dayStartUTC, dayEndUTC)
		if err != nil {
			return nil, err
		}
		eligibility[number] = claimable
	}
	return eligibility, nil
}

func jakartaDay(now time.Time) (string, time.Time, time.Time) {
	local := now.In(jakartaLocation)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, jakartaLocation)
	return start.Format("2006-01-02"), start.UTC(), start.AddDate(0, 0, 1).UTC()
}

func assignedMissions(now time.Time) [3]int {
	local := now.In(jakartaLocation)
	calendarDay := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	rotationStart := int(calendarDay.Unix()/int64(24*time.Hour/time.Second)) % 5
	if rotationStart < 0 {
		rotationStart += 5
	}
	return [3]int{
		rotationStart + 1,
		(rotationStart+1)%5 + 1,
		(rotationStart+2)%5 + 1,
	}
}

func isAssignedMission(number int, assigned [3]int) bool {
	for _, candidate := range assigned {
		if candidate == number {
			return true
		}
	}
	return false
}

func missionReward(number int) int {
	switch number {
	case 3:
		return 20
	case 5:
		return 30
	default:
		return 10
	}
}

func decorateDailyMission(
	mission model.DailyMission,
	points int,
	now time.Time,
	eligibility map[int]bool,
) model.DailyMission {
	assigned := assignedMissions(now)
	mission.Tasks = make([]model.DailyMissionTask, 0, len(assigned))
	mission.CompletedCount = 0
	mission.TotalCount = len(assigned)
	for index, number := range assigned {
		role := "bonus"
		if index == 0 {
			role = "primary"
		}
		completed := missionFlag(mission, number)
		claimable := eligibility[number] && !completed
		status := "locked"
		if completed {
			mission.CompletedCount++
			status = "claimed"
		} else if claimable {
			status = "claimable"
		}
		mission.Tasks = append(mission.Tasks, model.DailyMissionTask{
			Number:          number,
			Key:             fmt.Sprintf("mission_%d", number),
			Role:            role,
			Completed:       completed,
			Claimable:       claimable,
			Status:          status,
			VerificationKey: missionVerificationKey(number),
			EXPReward:       missionReward(number),
		})
	}
	mission.Experience = experienceProgress(points)
	return mission
}

func missionVerificationKey(number int) string {
	switch number {
	case 1:
		return "active_protection_today"
	case 2:
		return "daily_check_in"
	case 3:
		return "education_section_today"
	case 4:
		return "active_partner"
	case 5:
		return "education_module_today"
	default:
		return "unknown"
	}
}

func experienceProgress(points int) model.ExperienceProgress {
	points = max(0, points)
	return model.ExperienceProgress{
		TotalEXP:      points,
		Level:         points/experiencePerLevel + 1,
		LevelProgress: points % experiencePerLevel,
		LevelTarget:   experiencePerLevel,
	}
}

func missionFlag(mission model.DailyMission, number int) bool {
	switch number {
	case 1:
		return mission.Mission1
	case 2:
		return mission.Mission2
	case 3:
		return mission.Mission3
	case 4:
		return mission.Mission4
	case 5:
		return mission.Mission5
	default:
		return false
	}
}
