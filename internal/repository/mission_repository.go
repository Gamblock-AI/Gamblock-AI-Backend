package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/checkin"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/dailymission"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationprogress"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/google/uuid"
)

var ErrCustomMissionNotFound = errors.New("custom mission not found")
var ErrCustomMissionResolved = errors.New("custom mission is already resolved")

func (r *Repository) IsMissionClaimable(
	ctx context.Context,
	userID string,
	missionNum int,
	dayStartUTC, dayEndUTC time.Time,
) (bool, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		switch missionNum {
		case 1:
			for _, item := range snapshot.Devices {
				if item.UserID == userID && item.ProtectionStatus == "active" &&
					!item.LastSeenAt.Before(dayStartUTC) && item.LastSeenAt.Before(dayEndUTC) {
					return true, nil
				}
			}
		case 2:
			for _, item := range snapshot.CheckIns {
				if item.UserID == userID && !item.CreatedAt.Before(dayStartUTC) && item.CreatedAt.Before(dayEndUTC) {
					return true, nil
				}
			}
		case 3:
			for _, item := range snapshot.EducationProgress {
				if item.UserID == userID && len(item.CompletedSectionIDs) > 0 &&
					!item.UpdatedAt.Before(dayStartUTC) && item.UpdatedAt.Before(dayEndUTC) {
					return true, nil
				}
			}
		case 4:
			for _, item := range snapshot.Partners {
				if item.UserID == userID && item.Status == "active" {
					return true, nil
				}
			}
		case 5:
			for _, item := range snapshot.EducationProgress {
				if item.UserID == userID && item.CompletedAt != nil &&
					!item.CompletedAt.Before(dayStartUTC) && item.CompletedAt.Before(dayEndUTC) {
					return true, nil
				}
			}
		}
		return false, nil
	}

	switch missionNum {
	case 1:
		return r.db.Device.Query().Where(
			device.UserIDEQ(userID),
			device.ProtectionStatusEQ(device.ProtectionStatusActive),
			device.LastSeenAtGTE(dayStartUTC),
			device.LastSeenAtLT(dayEndUTC),
		).Exist(ctx)
	case 2:
		return r.db.CheckIn.Query().Where(
			checkin.UserIDEQ(userID),
			checkin.CreatedAtGTE(dayStartUTC),
			checkin.CreatedAtLT(dayEndUTC),
		).Exist(ctx)
	case 3:
		rows, err := r.db.PsychoeducationProgress.Query().Where(
			psychoeducationprogress.UserIDEQ(userID),
			psychoeducationprogress.UpdatedAtGTE(dayStartUTC),
			psychoeducationprogress.UpdatedAtLT(dayEndUTC),
		).All(ctx)
		if err != nil {
			return false, err
		}
		for _, item := range rows {
			if len(item.CompletedSectionIds) > 0 {
				return true, nil
			}
		}
		return false, nil
	case 4:
		return r.db.PartnerLink.Query().Where(
			partnerlink.UserIDEQ(userID),
			partnerlink.StatusEQ(partnerlink.StatusActive),
		).Exist(ctx)
	case 5:
		return r.db.PsychoeducationProgress.Query().Where(
			psychoeducationprogress.UserIDEQ(userID),
			psychoeducationprogress.CompletedAtGTE(dayStartUTC),
			psychoeducationprogress.CompletedAtLT(dayEndUTC),
		).Exist(ctx)
	default:
		return false, fmt.Errorf("unsupported mission %d", missionNum)
	}
}

func (r *Repository) GetMissionByDate(
	ctx context.Context,
	userID, date string,
	dayStartUTC, dayEndUTC time.Time,
) (model.DailyMission, int, error) {
	if r.db != nil {
		rows, err := r.db.DailyMission.Query().Where(
			dailymission.UserID(userID),
			dailymission.Or(
				dailymission.MissionDate(date),
				dailymission.And(
					dailymission.MissionDateIsNil(),
					dailymission.CreatedAtGTE(dayStartUTC),
					dailymission.CreatedAtLT(dayEndUTC),
				),
			),
		).Order(ent.Asc(dailymission.FieldCreatedAt)).All(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		points, err := r.userExperience(ctx, userID)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		return missionFromRows(rows, userID, date), points, nil
	}

	r.store.RLock()
	defer r.store.RUnlock()
	points := 0
	for _, user := range r.store.Users {
		if user.ID == userID {
			points = user.ExperiencePoints
			break
		}
	}
	for _, mission := range r.store.Missions {
		if mission.UserID == userID && mission.Date == date {
			return toDailyMission(mission), points, nil
		}
	}
	return model.DailyMission{ID: "day_" + date, UserID: userID, Date: date}, points, nil
}

func (r *Repository) UpsertMission(
	ctx context.Context,
	userID, date string,
	dayStartUTC, dayEndUTC time.Time,
	missionNum int,
	completed bool,
	reward int,
) (model.DailyMission, int, error) {
	if r.db != nil {
		var lastErr error
		for attempt := 0; attempt < 2; attempt++ {
			mission, points, err := r.upsertMissionDB(
				ctx, userID, date, dayStartUTC, dayEndUTC, missionNum, completed, reward,
			)
			if err == nil {
				r.RefreshStore(ctx)
				return mission, points, nil
			}
			lastErr = err
			if !ent.IsConstraintError(err) {
				break
			}
		}
		return model.DailyMission{}, 0, lastErr
	}
	return r.upsertMissionInMemory(userID, date, missionNum, completed, reward)
}

func (r *Repository) upsertMissionDB(
	ctx context.Context,
	userID, date string,
	dayStartUTC, dayEndUTC time.Time,
	missionNum int,
	completed bool,
	reward int,
) (model.DailyMission, int, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.DailyMission{}, 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	key := fmt.Sprintf("mission_%d", missionNum)
	row, queryErr := tx.DailyMission.Query().Where(
		dailymission.UserID(userID),
		dailymission.MissionKey(key),
		dailymission.Or(
			dailymission.MissionDate(date),
			dailymission.And(
				dailymission.MissionDateIsNil(),
				dailymission.CreatedAtGTE(dayStartUTC),
				dailymission.CreatedAtLT(dayEndUTC),
			),
		),
	).First(ctx)
	if queryErr != nil && !ent.IsNotFound(queryErr) {
		return model.DailyMission{}, 0, queryErr
	}

	if ent.IsNotFound(queryErr) {
		status := dailymission.StatusPending
		effectiveReward := reward
		if completed {
			status = dailymission.StatusCompleted
			available, availableErr := availableDailyExperience(ctx, tx.Client(), userID, date)
			if availableErr != nil {
				return model.DailyMission{}, 0, availableErr
			}
			if available < reward {
				effectiveReward = 0
			}
		}
		creator := tx.DailyMission.Create().
			SetID("mis_" + uuid.NewString()[:8]).
			SetUserID(userID).
			SetMissionDate(date).
			SetMissionKey(key).
			SetStatus(status).
			SetExpReward(effectiveReward)
		if completed {
			creator.SetCompletedAt(time.Now().UTC())
		}
		if _, err = creator.Save(ctx); err != nil {
			return model.DailyMission{}, 0, err
		}
		if completed {
			if _, err = tx.User.Update().
				Where(entuser.IDEQ(userID)).
				AddExperiencePoints(effectiveReward).
				Save(ctx); err != nil {
				return model.DailyMission{}, 0, err
			}
		}
	} else {
		if row.Status == dailymission.StatusSkipped {
			return model.DailyMission{}, 0, fmt.Errorf("mission %d is already adjusted", missionNum)
		}
		desiredStatus := dailymission.StatusPending
		if completed {
			desiredStatus = dailymission.StatusCompleted
		}
		effectiveReward := row.ExpReward
		if effectiveReward == 0 && row.Status != dailymission.StatusCompleted {
			effectiveReward = reward
			if completed {
				available, availableErr := availableDailyExperience(ctx, tx.Client(), userID, date)
				if availableErr != nil {
					return model.DailyMission{}, 0, availableErr
				}
				if available < effectiveReward {
					effectiveReward = 0
				}
			}
		}
		updater := tx.DailyMission.Update().Where(
			dailymission.IDEQ(row.ID),
			dailymission.StatusNEQ(desiredStatus),
		).SetStatus(desiredStatus).SetMissionDate(date).SetExpReward(effectiveReward)
		if completed {
			updater.SetCompletedAt(time.Now().UTC())
		} else {
			updater.ClearCompletedAt()
		}
		changed, updateErr := updater.Save(ctx)
		if updateErr != nil {
			return model.DailyMission{}, 0, updateErr
		}
		if changed == 1 && effectiveReward > 0 {
			userUpdater := tx.User.Update().Where(entuser.IDEQ(userID))
			if completed {
				userUpdater.AddExperiencePoints(effectiveReward)
			} else {
				userUpdater.Where(entuser.ExperiencePointsGTE(effectiveReward)).
					AddExperiencePoints(-effectiveReward)
			}
			if _, err = userUpdater.Save(ctx); err != nil {
				return model.DailyMission{}, 0, err
			}
		}
	}

	rows, err := tx.DailyMission.Query().Where(
		dailymission.UserID(userID),
		dailymission.MissionDate(date),
	).Order(ent.Asc(dailymission.FieldCreatedAt)).All(ctx)
	if err != nil {
		return model.DailyMission{}, 0, err
	}
	user, err := tx.User.Query().Where(entuser.IDEQ(userID)).Only(ctx)
	if err != nil {
		return model.DailyMission{}, 0, err
	}
	mission := missionFromRows(rows, userID, date)
	points := user.ExperiencePoints
	if err = tx.Commit(); err != nil {
		return model.DailyMission{}, 0, err
	}
	committed = true
	return mission, points, nil
}

func (r *Repository) upsertMissionInMemory(
	userID, date string,
	missionNum int,
	completed bool,
	reward int,
) (model.DailyMission, int, error) {
	now := time.Now().UTC()
	r.store.Lock()
	defer r.store.Unlock()

	userIndex := -1
	for i := range r.store.Users {
		if r.store.Users[i].ID == userID {
			userIndex = i
			break
		}
	}
	for i := range r.store.Missions {
		mission := &r.store.Missions[i]
		if mission.UserID != userID || mission.Date != date {
			continue
		}
		wasCompleted := missionCompleted(*mission, missionNum)
		if wasCompleted != completed {
			setMissionFlag(mission, missionNum, completed)
			mission.UpdatedAt = now
			if userIndex >= 0 {
				if completed {
					r.store.Users[userIndex].ExperiencePoints += reward
				} else {
					r.store.Users[userIndex].ExperiencePoints = max(
						0,
						r.store.Users[userIndex].ExperiencePoints-reward,
					)
				}
			}
		}
		return toDailyMission(*mission), userExperienceFromStore(r.store, userIndex), nil
	}

	entry := store.DailyMission{
		ID: "mis_" + uuid.NewString()[:8], UserID: userID, Date: date,
		CreatedAt: now, UpdatedAt: now,
	}
	setMissionFlag(&entry, missionNum, completed)
	if completed && userIndex >= 0 {
		r.store.Users[userIndex].ExperiencePoints += reward
	}
	r.store.Missions = append(r.store.Missions, entry)
	return toDailyMission(entry), userExperienceFromStore(r.store, userIndex), nil
}

func (r *Repository) AddUserExperiencePoints(
	ctx context.Context,
	userID string,
	delta int,
) (int, error) {
	if r.db != nil {
		if _, err := r.db.User.Update().
			Where(entuser.IDEQ(userID)).
			AddExperiencePoints(delta).
			Save(ctx); err != nil {
			return 0, err
		}
		return r.userExperience(ctx, userID)
	}

	r.store.Lock()
	defer r.store.Unlock()
	for index := range r.store.Users {
		if r.store.Users[index].ID == userID {
			r.store.Users[index].ExperiencePoints = max(
				0,
				r.store.Users[index].ExperiencePoints+delta,
			)
			return r.store.Users[index].ExperiencePoints, nil
		}
	}
	return 0, fmt.Errorf("user %s not found", userID)
}

func (r *Repository) userExperience(ctx context.Context, userID string) (int, error) {
	user, err := r.db.User.Query().Where(entuser.IDEQ(userID)).Only(ctx)
	if err != nil {
		return 0, err
	}
	return user.ExperiencePoints, nil
}

func userExperienceFromStore(st *store.Store, index int) int {
	if index < 0 {
		return 0
	}
	return st.Users[index].ExperiencePoints
}

func missionFromRows(rows []*ent.DailyMission, userID, date string) model.DailyMission {
	mission := model.DailyMission{ID: "day_" + date, UserID: userID, Date: date}
	for index, item := range rows {
		if index == 0 || item.CreatedAt.Before(mission.CreatedAt) {
			mission.CreatedAt = item.CreatedAt
		}
		if item.UpdatedAt.After(mission.UpdatedAt) {
			mission.UpdatedAt = item.UpdatedAt
		}
		mission.TaskRecords = append(mission.TaskRecords, model.MissionRecord{
			ID:               item.ID,
			Key:              item.MissionKey,
			Source:           string(item.Source),
			TitleEncrypted:   value(item.TitleEncrypted),
			Status:           string(item.Status),
			Reward:           item.ExpReward,
			AdjustmentReason: valueEnum(item.AdjustmentReason),
			CompletedAt:      item.CompletedAt,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
		setMissionFlag(&mission, missionNumber(item.MissionKey), item.Status == dailymission.StatusCompleted)
		if item.Status == dailymission.StatusSkipped && item.AdjustmentReason != nil {
			adjustment := model.MissionAdjustment{
				OriginalNumber: missionNumber(item.MissionKey),
				Action:         "skip",
				Reason:         string(*item.AdjustmentReason),
				AdjustedAt:     item.UpdatedAt,
			}
			if item.ReplacementKey != nil {
				adjustment.Action = "replace"
				adjustment.ReplacementNumber = missionNumber(*item.ReplacementKey)
			}
			mission.AdjustmentHistory = append(mission.AdjustmentHistory, adjustment)
		}
	}
	if len(mission.AdjustmentHistory) > 0 {
		mission.Adjustment = &mission.AdjustmentHistory[len(mission.AdjustmentHistory)-1]
	}
	return mission
}

func missionNumber(key string) int {
	number, _ := strconv.Atoi(strings.TrimPrefix(key, "mission_"))
	return number
}

func missionCompleted(mission model.DailyMission, number int) bool {
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
	case 6:
		return mission.Mission6
	default:
		return false
	}
}

func setMissionFlag(mission *model.DailyMission, number int, completed bool) {
	switch number {
	case 1:
		mission.Mission1 = completed
	case 2:
		mission.Mission2 = completed
	case 3:
		mission.Mission3 = completed
	case 4:
		mission.Mission4 = completed
	case 5:
		mission.Mission5 = completed
	case 6:
		mission.Mission6 = completed
	}
}

func toDailyMission(mission store.DailyMission) model.DailyMission {
	result := model.DailyMission{
		ID: mission.ID, UserID: mission.UserID, Date: mission.Date,
		Mission1: mission.Mission1, Mission2: mission.Mission2, Mission3: mission.Mission3,
		Mission4: mission.Mission4, Mission5: mission.Mission5, Mission6: mission.Mission6,
		TaskRecords:       append([]model.MissionRecord(nil), mission.TaskRecords...),
		AdjustmentHistory: append([]model.MissionAdjustment(nil), mission.AdjustmentHistory...),
		CreatedAt:         mission.CreatedAt, UpdatedAt: mission.UpdatedAt,
	}
	if len(result.AdjustmentHistory) > 0 {
		result.Adjustment = &result.AdjustmentHistory[len(result.AdjustmentHistory)-1]
	}
	return result
}

func valueEnum[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func (r *Repository) CreateCustomMission(
	ctx context.Context,
	userID, date string,
	dayStartUTC, dayEndUTC time.Time,
	titleEncrypted string,
	reward int,
) (model.DailyMission, int, error) {
	if r.db != nil {
		tx, err := r.db.Tx(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		id := "mis_" + uuid.NewString()[:8]
		now := time.Now().UTC()
		if _, err = tx.DailyMission.Create().
			SetID(id).
			SetUserID(userID).
			SetMissionDate(date).
			SetMissionKey("custom_" + id).
			SetSource(dailymission.SourceCustom).
			SetTitleEncrypted(titleEncrypted).
			SetStatus(dailymission.StatusPending).
			SetExpReward(reward).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx); err != nil {
			return model.DailyMission{}, 0, err
		}
		rows, err := tx.DailyMission.Query().Where(dailymission.UserID(userID), dailymission.MissionDate(date)).Order(ent.Asc(dailymission.FieldCreatedAt)).All(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		user, err := tx.User.Query().Where(entuser.IDEQ(userID)).Only(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		if err = tx.Commit(); err != nil {
			return model.DailyMission{}, 0, err
		}
		committed = true
		r.RefreshStore(ctx)
		return missionFromRows(rows, userID, date), user.ExperiencePoints, nil
	}

	now := time.Now().UTC()
	r.store.Lock()
	defer r.store.Unlock()
	userIndex := missionUserIndex(r.store, userID)
	mission := missionForDay(r.store, userID, date, now)
	id := "mis_" + uuid.NewString()[:8]
	mission.TaskRecords = append(mission.TaskRecords, model.MissionRecord{ID: id, Key: "custom_" + id, Source: "custom", TitleEncrypted: titleEncrypted, Status: "pending", Reward: reward, CreatedAt: now, UpdatedAt: now})
	mission.UpdatedAt = now
	return toDailyMission(*mission), userExperienceFromStore(r.store, userIndex), nil
}

func (r *Repository) UpdateCustomMission(
	ctx context.Context,
	userID, date, id, titleEncrypted string,
) (model.DailyMission, int, error) {
	if r.db != nil {
		row, err := r.db.DailyMission.Query().Where(dailymission.IDEQ(id), dailymission.UserIDEQ(userID), dailymission.MissionDateEQ(date), dailymission.SourceEQ(dailymission.SourceCustom)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return model.DailyMission{}, 0, ErrCustomMissionNotFound
			}
			return model.DailyMission{}, 0, err
		}
		if row.Status != dailymission.StatusPending {
			return model.DailyMission{}, 0, ErrCustomMissionResolved
		}
		if _, err = row.Update().SetTitleEncrypted(titleEncrypted).SetUpdatedAt(time.Now().UTC()).Save(ctx); err != nil {
			return model.DailyMission{}, 0, err
		}
		r.RefreshStore(ctx)
		return r.GetMissionByDate(ctx, userID, date, time.Time{}, time.Time{})
	}
	r.store.Lock()
	defer r.store.Unlock()
	userIndex := missionUserIndex(r.store, userID)
	for _, mission := range r.store.Missions {
		if mission.UserID != userID || mission.Date != date {
			continue
		}
		for index := range mission.TaskRecords {
			record := &mission.TaskRecords[index]
			if record.ID == id && record.Source == "custom" {
				if record.Status != "pending" {
					return model.DailyMission{}, 0, ErrCustomMissionResolved
				}
				record.TitleEncrypted = titleEncrypted
				record.UpdatedAt = time.Now().UTC()
				return toDailyMission(mission), userExperienceFromStore(r.store, userIndex), nil
			}
		}
	}
	return model.DailyMission{}, 0, ErrCustomMissionNotFound
}

func (r *Repository) DeleteCustomMission(ctx context.Context, userID, date, id string) (model.DailyMission, int, error) {
	if r.db != nil {
		row, err := r.db.DailyMission.Query().Where(dailymission.IDEQ(id), dailymission.UserIDEQ(userID), dailymission.MissionDateEQ(date), dailymission.SourceEQ(dailymission.SourceCustom)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return model.DailyMission{}, 0, ErrCustomMissionNotFound
			}
			return model.DailyMission{}, 0, err
		}
		if row.Status != dailymission.StatusPending {
			return model.DailyMission{}, 0, ErrCustomMissionResolved
		}
		if err = r.db.DailyMission.DeleteOneID(row.ID).Exec(ctx); err != nil {
			return model.DailyMission{}, 0, err
		}
		r.RefreshStore(ctx)
		return r.GetMissionByDate(ctx, userID, date, time.Time{}, time.Time{})
	}
	r.store.Lock()
	defer r.store.Unlock()
	userIndex := missionUserIndex(r.store, userID)
	for missionIndex := range r.store.Missions {
		mission := &r.store.Missions[missionIndex]
		if mission.UserID != userID || mission.Date != date {
			continue
		}
		for index := range mission.TaskRecords {
			record := mission.TaskRecords[index]
			if record.ID == id && record.Source == "custom" {
				if record.Status != "pending" {
					return model.DailyMission{}, 0, ErrCustomMissionResolved
				}
				mission.TaskRecords = append(mission.TaskRecords[:index], mission.TaskRecords[index+1:]...)
				mission.UpdatedAt = time.Now().UTC()
				return toDailyMission(*mission), userExperienceFromStore(r.store, userIndex), nil
			}
		}
	}
	return model.DailyMission{}, 0, ErrCustomMissionNotFound
}

func (r *Repository) CompleteCustomMission(ctx context.Context, userID, date, id string, reward int) (model.DailyMission, int, error) {
	if r.db != nil {
		tx, err := r.db.Tx(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()
		row, err := tx.DailyMission.Query().Where(dailymission.IDEQ(id), dailymission.UserIDEQ(userID), dailymission.MissionDateEQ(date), dailymission.SourceEQ(dailymission.SourceCustom)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return model.DailyMission{}, 0, ErrCustomMissionNotFound
			}
			return model.DailyMission{}, 0, err
		}
		if row.Status == dailymission.StatusSkipped {
			return model.DailyMission{}, 0, ErrCustomMissionResolved
		}
		if row.Status == dailymission.StatusPending {
			available, availableErr := availableDailyExperience(ctx, tx.Client(), userID, date)
			if availableErr != nil {
				return model.DailyMission{}, 0, availableErr
			}
			effectiveReward := reward
			if available < reward {
				effectiveReward = 0
			}
			changed, err := tx.DailyMission.Update().Where(dailymission.IDEQ(id), dailymission.StatusEQ(dailymission.StatusPending)).SetStatus(dailymission.StatusCompleted).SetExpReward(effectiveReward).SetCompletedAt(time.Now().UTC()).SetUpdatedAt(time.Now().UTC()).Save(ctx)
			if err != nil {
				return model.DailyMission{}, 0, err
			}
			if changed != 1 {
				return model.DailyMission{}, 0, fmt.Errorf("custom mission claim conflicted")
			}
			if _, err = tx.User.Update().Where(entuser.IDEQ(userID)).AddExperiencePoints(effectiveReward).Save(ctx); err != nil {
				return model.DailyMission{}, 0, err
			}
		}
		rows, err := tx.DailyMission.Query().Where(dailymission.UserID(userID), dailymission.MissionDate(date)).Order(ent.Asc(dailymission.FieldCreatedAt)).All(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		user, err := tx.User.Query().Where(entuser.IDEQ(userID)).Only(ctx)
		if err != nil {
			return model.DailyMission{}, 0, err
		}
		if err = tx.Commit(); err != nil {
			return model.DailyMission{}, 0, err
		}
		committed = true
		r.RefreshStore(ctx)
		return missionFromRows(rows, userID, date), user.ExperiencePoints, nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	userIndex := missionUserIndex(r.store, userID)
	for missionIndex := range r.store.Missions {
		mission := &r.store.Missions[missionIndex]
		if mission.UserID != userID || mission.Date != date {
			continue
		}
		for index := range mission.TaskRecords {
			record := &mission.TaskRecords[index]
			if record.ID == id && record.Source == "custom" {
				if record.Status == "skipped" {
					return model.DailyMission{}, 0, ErrCustomMissionResolved
				}
				if record.Status == "pending" {
					now := time.Now().UTC()
					record.Status = "completed"
					record.Reward = reward
					record.CompletedAt = &now
					record.UpdatedAt = now
					mission.UpdatedAt = now
					if userIndex >= 0 {
						r.store.Users[userIndex].ExperiencePoints += reward
					}
				}
				return toDailyMission(*mission), userExperienceFromStore(r.store, userIndex), nil
			}
		}
	}
	return model.DailyMission{}, 0, ErrCustomMissionNotFound
}

func missionUserIndex(st *store.Store, userID string) int {
	for index := range st.Users {
		if st.Users[index].ID == userID {
			return index
		}
	}
	return -1
}

func missionForDay(st *store.Store, userID, date string, now time.Time) *store.DailyMission {
	for index := range st.Missions {
		if st.Missions[index].UserID == userID && st.Missions[index].Date == date {
			return &st.Missions[index]
		}
	}
	st.Missions = append(st.Missions, store.DailyMission{ID: "day_" + date, UserID: userID, Date: date, CreatedAt: now, UpdatedAt: now})
	return &st.Missions[len(st.Missions)-1]
}
