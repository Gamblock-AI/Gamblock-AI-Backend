package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/dailymission"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/google/uuid"
)

func (r *Repository) GetMissionByDate(ctx context.Context, userID, date string) (model.DailyMission, error) {
	if r.db != nil {
		start, err := time.Parse("2006-01-02", date)
		if err != nil {
			return model.DailyMission{}, err
		}
		rows, err := r.db.DailyMission.Query().Where(
			dailymission.UserID(userID),
			dailymission.CreatedAtGTE(start),
			dailymission.CreatedAtLT(start.Add(24*time.Hour)),
		).All(ctx)
		if err != nil {
			return model.DailyMission{}, err
		}
		if len(rows) == 0 {
			return model.DailyMission{}, fmt.Errorf("not found")
		}
		mission := model.DailyMission{ID: "day_" + date, UserID: userID, Date: date, CreatedAt: rows[0].CreatedAt, UpdatedAt: rows[0].CreatedAt}
		for _, item := range rows {
			setMissionFlag(&mission, missionNumber(item.MissionKey), item.Status == dailymission.StatusCompleted)
			if item.CreatedAt.After(mission.UpdatedAt) {
				mission.UpdatedAt = item.CreatedAt
			}
		}
		return mission, nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	for _, m := range r.store.Missions {
		if m.UserID == userID && m.Date == date {
			return model.DailyMission{
				ID: m.ID, UserID: m.UserID, Date: m.Date,
				Mission1: m.Mission1, Mission2: m.Mission2, Mission3: m.Mission3,
				Mission4: m.Mission4, Mission5: m.Mission5,
				CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
			}, nil
		}
	}
	return model.DailyMission{}, fmt.Errorf("not found")
}

func (r *Repository) UpsertMission(ctx context.Context, userID, date string, missionNum int, completed bool) (model.DailyMission, error) {
	now := time.Now().UTC()
	if r.db != nil {
		start, err := time.Parse("2006-01-02", date)
		if err != nil {
			return model.DailyMission{}, err
		}
		key := fmt.Sprintf("mission_%d", missionNum)
		status := dailymission.StatusPending
		var completedAt *time.Time
		if completed {
			status = dailymission.StatusCompleted
			completedAt = &now
		}
		item, queryErr := r.db.DailyMission.Query().Where(
			dailymission.UserID(userID),
			dailymission.MissionKey(key),
			dailymission.CreatedAtGTE(start),
			dailymission.CreatedAtLT(start.Add(24*time.Hour)),
		).First(ctx)
		if queryErr == nil {
			_, err = item.Update().SetStatus(status).SetNillableCompletedAt(completedAt).Save(ctx)
		} else {
			creator := r.db.DailyMission.Create().
				SetID("mis_" + uuid.NewString()[:8]).
				SetUserID(userID).
				SetMissionKey(key).
				SetStatus(status)
			if completedAt != nil {
				creator.SetCompletedAt(*completedAt)
			}
			_, err = creator.Save(ctx)
		}
		if err != nil {
			return model.DailyMission{}, err
		}
		r.RefreshStore(ctx)
		return r.GetMissionByDate(ctx, userID, date)
	}
	r.store.Lock()
	defer r.store.Unlock()
	for i, m := range r.store.Missions {
		if m.UserID == userID && m.Date == date {
			switch missionNum {
			case 1:
				r.store.Missions[i].Mission1 = completed
			case 2:
				r.store.Missions[i].Mission2 = completed
			case 3:
				r.store.Missions[i].Mission3 = completed
			case 4:
				r.store.Missions[i].Mission4 = completed
			case 5:
				r.store.Missions[i].Mission5 = completed
			}
			r.store.Missions[i].UpdatedAt = now
			e := r.store.Missions[i]
			return toDailyMission(e), nil
		}
	}
	entry := store.DailyMission{ID: "mis_" + uuid.NewString()[:8], UserID: userID, Date: date, CreatedAt: now, UpdatedAt: now}
	switch missionNum {
	case 1:
		entry.Mission1 = completed
	case 2:
		entry.Mission2 = completed
	case 3:
		entry.Mission3 = completed
	case 4:
		entry.Mission4 = completed
	case 5:
		entry.Mission5 = completed
	}
	r.store.Missions = append(r.store.Missions, entry)
	return toDailyMission(entry), nil
}

func missionNumber(key string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(key, "mission_"))
	return n
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
	}
}

func toDailyMission(m store.DailyMission) model.DailyMission {
	return model.DailyMission{
		ID: m.ID, UserID: m.UserID, Date: m.Date,
		Mission1: m.Mission1, Mission2: m.Mission2, Mission3: m.Mission3,
		Mission4: m.Mission4, Mission5: m.Mission5,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
