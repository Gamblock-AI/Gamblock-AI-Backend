package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetMissionByDate(ctx context.Context, userID, date string) (model.DailyMission, error) {
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
	r.store.Lock()
	defer r.store.Unlock()
	for i, m := range r.store.Missions {
		if m.UserID == userID && m.Date == date {
			switch missionNum {
			case 1: r.store.Missions[i].Mission1 = completed
			case 2: r.store.Missions[i].Mission2 = completed
			case 3: r.store.Missions[i].Mission3 = completed
			case 4: r.store.Missions[i].Mission4 = completed
			case 5: r.store.Missions[i].Mission5 = completed
			}
			r.store.Missions[i].UpdatedAt = now
			e := r.store.Missions[i]
			return toDailyMission(e), nil
		}
	}
	entry := store.DailyMission{ID: "mis_" + uuid.NewString()[:8], UserID: userID, Date: date, CreatedAt: now, UpdatedAt: now}
	switch missionNum {
	case 1: entry.Mission1 = completed
	case 2: entry.Mission2 = completed
	case 3: entry.Mission3 = completed
	case 4: entry.Mission4 = completed
	case 5: entry.Mission5 = completed
	}
	r.store.Missions = append(r.store.Missions, entry)
	return toDailyMission(entry), nil
}

func toDailyMission(m store.DailyMission) model.DailyMission {
	return model.DailyMission{
		ID: m.ID, UserID: m.UserID, Date: m.Date,
		Mission1: m.Mission1, Mission2: m.Mission2, Mission3: m.Mission3,
		Mission4: m.Mission4, Mission5: m.Mission5,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
