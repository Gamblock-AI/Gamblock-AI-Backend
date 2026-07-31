package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	appcrypto "github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

const experiencePerLevel = 100
const dailyMissionReward = 10
const dailyMissionSlots = 5
const customMissionTitleMaxLength = 160

var jakartaLocation = time.FixedZone("Asia/Jakarta", 7*60*60)

var ErrMissionNotClaimable = errors.New("mission requirements are not verified")
var ErrCustomMissionLimit = errors.New("custom mission limit reached")
var ErrCustomMissionInvalid = errors.New("custom mission is invalid")
var ErrCustomMissionNotEditable = errors.New("custom mission is not editable")

type systemMission struct {
	Number int
	Key    string
}

// Every default task can be completed without a partner relationship so the
// self-directed rehabilitation path has the same daily opportunity as a
// partner-connected student.
var defaultSystemMissions = []systemMission{
	{Number: 1, Key: "active_protection_today"},
	{Number: 2, Key: "daily_check_in"},
	{Number: 3, Key: "education_section_today"},
	{Number: 5, Key: "education_module_today"},
	{Number: 6, Key: "recovery_practice_today"},
}

type MissionService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

// NewMissionService is retained for standalone unit tests that do not create
// custom missions. Runtime wiring uses NewMissionServiceWithConfig so custom
// titles always use the configured AES-256-GCM key.
func NewMissionService(repo *repository.Repository, logger *zap.Logger) *MissionService {
	return NewMissionServiceWithConfig(repo, config.Config{}, logger)
}

func NewMissionServiceWithConfig(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *MissionService {
	return &MissionService{repo: repo, cfg: cfg, logger: logger}
}

func (s *MissionService) GetToday(ctx context.Context, userID string) (model.DailyMission, error) {
	date, start, end := jakartaDay(time.Now())
	mission, points, err := s.repo.GetMissionByDate(ctx, userID, date, start, end)
	if err != nil {
		return model.DailyMission{}, err
	}
	return s.decorate(ctx, userID, mission, points, start, end)
}

func (s *MissionService) CreateCustomMission(ctx context.Context, userID, title string) (model.DailyMission, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > customMissionTitleMaxLength {
		return model.DailyMission{}, ErrCustomMissionInvalid
	}
	date, start, end := jakartaDay(time.Now())
	current, points, err := s.repo.GetMissionByDate(ctx, userID, date, start, end)
	if err != nil {
		return model.DailyMission{}, err
	}
	if !canAddCustomMission(current) {
		return model.DailyMission{}, ErrCustomMissionLimit
	}
	encrypted, err := s.encryptTitle(title)
	if err != nil {
		return model.DailyMission{}, err
	}
	mission, points, err := s.repo.CreateCustomMission(ctx, userID, date, start, end, encrypted, dailyMissionReward)
	if err != nil {
		return model.DailyMission{}, err
	}
	return s.decorate(ctx, userID, mission, points, start, end)
}

func (s *MissionService) UpdateCustomMission(ctx context.Context, userID, id, title string) (model.DailyMission, error) {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > customMissionTitleMaxLength {
		return model.DailyMission{}, ErrCustomMissionInvalid
	}
	date, start, end := jakartaDay(time.Now())
	encrypted, err := s.encryptTitle(title)
	if err != nil {
		return model.DailyMission{}, err
	}
	mission, points, err := s.repo.UpdateCustomMission(ctx, userID, date, id, encrypted)
	if err != nil {
		return model.DailyMission{}, normalizeCustomMissionError(err)
	}
	return s.decorate(ctx, userID, mission, points, start, end)
}

func (s *MissionService) DeleteCustomMission(ctx context.Context, userID, id string) (model.DailyMission, error) {
	date, start, end := jakartaDay(time.Now())
	mission, points, err := s.repo.DeleteCustomMission(ctx, userID, date, id)
	if err != nil {
		return model.DailyMission{}, normalizeCustomMissionError(err)
	}
	return s.decorate(ctx, userID, mission, points, start, end)
}

func (s *MissionService) ClaimMissionByID(ctx context.Context, userID, id string) (model.DailyMission, error) {
	date, start, end := jakartaDay(time.Now())
	current, points, err := s.repo.GetMissionByDate(ctx, userID, date, start, end)
	if err != nil {
		return model.DailyMission{}, err
	}
	decorated, err := s.decorate(ctx, userID, current, points, start, end)
	if err != nil {
		return model.DailyMission{}, err
	}
	task := findMissionTask(decorated.Tasks, id)
	if task == nil {
		return model.DailyMission{}, ErrMissionNotClaimable
	}
	if task.Completed {
		return decorated, nil
	}
	if !task.Claimable {
		return model.DailyMission{}, ErrMissionNotClaimable
	}
	levelBefore := decorated.Experience.Level
	if task.Source == "custom" {
		current, points, err = s.repo.CompleteCustomMission(ctx, userID, date, task.ID, dailyMissionReward)
	} else {
		current, points, err = s.repo.UpsertMission(ctx, userID, date, start, end, task.Number, true, dailyMissionReward)
	}
	if err != nil {
		return model.DailyMission{}, err
	}
	updated, err := s.decorate(ctx, userID, current, points, start, end)
	if err != nil {
		return model.DailyMission{}, err
	}
	updated.Experience.NewlyUnlocked = newlyUnlockedDecor(levelBefore, updated.Experience.Level)
	return updated, nil
}

// Legacy number-based methods keep older in-repository callers compiling. New
// HTTP clients use task ids so custom and system tasks share one contract.
func (s *MissionService) UpdateMission(ctx context.Context, userID string, missionNum int, completed bool) (model.DailyMission, error) {
	if !completed {
		return model.DailyMission{}, ErrMissionNotClaimable
	}
	return s.ClaimMission(ctx, userID, missionNum)
}

func (s *MissionService) ClaimMission(ctx context.Context, userID string, missionNum int) (model.DailyMission, error) {
	return s.ClaimMissionByID(ctx, userID, systemMissionID(missionNum))
}

func (s *MissionService) decorate(ctx context.Context, userID string, mission model.DailyMission, points int, start, end time.Time) (model.DailyMission, error) {
	records := missionRecords(mission)
	custom := make([]model.MissionRecord, 0, len(records))
	systemByNumber := make(map[int]model.MissionRecord, len(defaultSystemMissions))
	for _, record := range records {
		if record.Source == "custom" {
			custom = append(custom, record)
			continue
		}
		if number := missionNumberFromKey(record.Key); number != 0 {
			systemByNumber[number] = record
		}
	}

	mission.Tasks = make([]model.DailyMissionTask, 0, dailyMissionSlots)
	mission.CompletedCount = 0
	mission.ResolvedCount = 0
	for _, record := range custom {
		title, err := s.decryptTitle(record.TitleEncrypted)
		if err != nil {
			return model.DailyMission{}, err
		}
		task := taskFromCustomRecord(record, title)
		mission.Tasks = append(mission.Tasks, task)
		countMissionTask(&mission, task)
	}

	remainingSystemSlots := dailyMissionSlots - len(custom)
	for _, definition := range defaultSystemMissions {
		if record, exists := systemByNumber[definition.Number]; exists {
			task := taskFromSystemRecord(definition, record)
			mission.Tasks = append(mission.Tasks, task)
			countMissionTask(&mission, task)
			remainingSystemSlots--
		}
	}
	for _, definition := range defaultSystemMissions {
		if remainingSystemSlots == 0 {
			break
		}
		if _, exists := systemByNumber[definition.Number]; exists {
			continue
		}
		claimable, err := s.repo.IsMissionClaimable(ctx, userID, definition.Number, start, end)
		if err != nil {
			return model.DailyMission{}, err
		}
		task := model.DailyMissionTask{
			ID: systemMissionID(definition.Number), Number: definition.Number,
			Key: fmt.Sprintf("mission_%d", definition.Number), Source: "system",
			SystemKey: definition.Key, Completed: false, Claimable: claimable,
			Status: statusForClaimable(claimable), ClaimMode: "verified",
			VerificationKey: definition.Key, EXPReward: dailyMissionReward,
		}
		mission.Tasks = append(mission.Tasks, task)
		remainingSystemSlots--
	}
	mission.TotalCount = len(mission.Tasks)
	mission.Experience = experienceProgress(points)
	return mission, nil
}

func canAddCustomMission(mission model.DailyMission) bool {
	customCount, resolvedSystemCount := 0, 0
	for _, record := range missionRecords(mission) {
		if record.Source == "custom" {
			customCount++
			continue
		}
		if missionNumberFromKey(record.Key) != 0 {
			resolvedSystemCount++
		}
	}
	return customCount+resolvedSystemCount < dailyMissionSlots
}

func missionRecords(mission model.DailyMission) []model.MissionRecord {
	records := append([]model.MissionRecord(nil), mission.TaskRecords...)
	systemRecord := make(map[int]struct{}, len(records))
	for _, record := range records {
		if record.Source == "system" {
			systemRecord[missionNumberFromKey(record.Key)] = struct{}{}
		}
	}
	for number, completed := range map[int]bool{1: mission.Mission1, 2: mission.Mission2, 3: mission.Mission3, 4: mission.Mission4, 5: mission.Mission5, 6: mission.Mission6} {
		if !completed {
			continue
		}
		if _, exists := systemRecord[number]; exists {
			continue
		}
		records = append(records, model.MissionRecord{ID: systemMissionID(number), Key: fmt.Sprintf("mission_%d", number), Source: "system", Status: "completed", Reward: dailyMissionReward})
		systemRecord[number] = struct{}{}
	}
	for _, adjustment := range mission.AdjustmentHistory {
		if adjustment.Action != "skip" || adjustment.OriginalNumber == 0 {
			continue
		}
		if _, exists := systemRecord[adjustment.OriginalNumber]; exists {
			continue
		}
		records = append(records, model.MissionRecord{
			ID:               systemMissionID(adjustment.OriginalNumber),
			Key:              fmt.Sprintf("mission_%d", adjustment.OriginalNumber),
			Source:           "system",
			Status:           "skipped",
			Reward:           dailyMissionReward,
			AdjustmentReason: adjustment.Reason,
			UpdatedAt:        adjustment.AdjustedAt,
		})
		systemRecord[adjustment.OriginalNumber] = struct{}{}
	}
	return records
}

func taskFromCustomRecord(record model.MissionRecord, title string) model.DailyMissionTask {
	task := model.DailyMissionTask{ID: record.ID, Key: record.Key, Source: "custom", Title: title, Status: record.Status, Completed: record.Status == "completed", ClaimMode: "self_attested", EXPReward: dailyMissionReward}
	task.Claimable = record.Status == "pending"
	return task
}

func taskFromSystemRecord(definition systemMission, record model.MissionRecord) model.DailyMissionTask {
	reward := record.Reward
	if reward == 0 {
		reward = dailyMissionReward
	}
	return model.DailyMissionTask{ID: record.ID, Number: definition.Number, Key: record.Key, Source: "system", SystemKey: definition.Key, Completed: record.Status == "completed", Claimable: false, Status: record.Status, ClaimMode: "verified", VerificationKey: definition.Key, EXPReward: reward}
}

func countMissionTask(mission *model.DailyMission, task model.DailyMissionTask) {
	if task.Completed {
		mission.CompletedCount++
		mission.ResolvedCount++
		return
	}
	if task.Status == "skipped" {
		mission.ResolvedCount++
	}
}

func findMissionTask(tasks []model.DailyMissionTask, id string) *model.DailyMissionTask {
	for index := range tasks {
		if tasks[index].ID == id {
			return &tasks[index]
		}
	}
	return nil
}

func systemMissionID(number int) string { return fmt.Sprintf("system:%d", number) }

func missionNumberFromKey(key string) int {
	for _, definition := range defaultSystemMissions {
		if key == fmt.Sprintf("mission_%d", definition.Number) {
			return definition.Number
		}
	}
	return 0
}

func statusForClaimable(claimable bool) string {
	if claimable {
		return "claimable"
	}
	return "locked"
}

func (s *MissionService) encryptTitle(title string) (string, error) {
	if s.cfg.JournalEncryptionKey == "" {
		return "", fmt.Errorf("custom mission encryption is not configured")
	}
	return appcrypto.Encrypt(title, s.cfg.JournalEncryptionKey)
}

func (s *MissionService) decryptTitle(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", fmt.Errorf("custom mission title is unavailable")
	}
	if s.cfg.JournalEncryptionKey == "" {
		return "", fmt.Errorf("custom mission encryption is not configured")
	}
	return appcrypto.Decrypt(ciphertext, s.cfg.JournalEncryptionKey)
}

func normalizeCustomMissionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrCustomMissionNotFound) || errors.Is(err, repository.ErrCustomMissionResolved) {
		return fmt.Errorf("%w: %v", ErrCustomMissionNotEditable, err)
	}
	return err
}

func jakartaDay(now time.Time) (string, time.Time, time.Time) {
	local := now.In(jakartaLocation)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, jakartaLocation)
	return start.Format("2006-01-02"), start.UTC(), start.AddDate(0, 0, 1).UTC()
}

func experienceProgress(points int) model.ExperienceProgress {
	points = max(0, points)
	return model.ExperienceProgress{TotalEXP: points, Level: points/experiencePerLevel + 1, LevelProgress: points % experiencePerLevel, LevelTarget: experiencePerLevel}
}
