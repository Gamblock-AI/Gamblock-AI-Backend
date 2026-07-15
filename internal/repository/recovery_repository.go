package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetIntention(ctx context.Context, userID string) (model.Intention, bool) {
	r.store.RLock()
	defer r.store.RUnlock()
	for _, intn := range r.store.Intentions {
		if intn.UserID == userID && intn.Status == "active" {
			return intn, true
		}
	}
	return model.Intention{}, false
}

func (r *Repository) SaveIntention(ctx context.Context, userID, text, status string) (model.Intention, error) {
	now := time.Now().UTC()
	r.store.Lock()
	defer r.store.Unlock()

	// Update existing active intention if any, or create a new one.
	for i, intn := range r.store.Intentions {
		if intn.UserID == userID {
			r.store.Intentions[i].Text = text
			r.store.Intentions[i].Status = status
			r.store.Intentions[i].UpdatedAt = now
			return r.store.Intentions[i], nil
		}
	}

	newEntry := store.Intention{
		ID:        "int_" + uuid.NewString()[:8],
		UserID:    userID,
		Text:      text,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.store.Intentions = append(r.store.Intentions, newEntry)
	return newEntry, nil
}

func (r *Repository) GetCheckIns(ctx context.Context, userID string) ([]model.CheckIn, error) {
	r.store.RLock()
	defer r.store.RUnlock()
	var list []model.CheckIn
	for _, chk := range r.store.CheckIns {
		if chk.UserID == userID {
			list = append(list, chk)
		}
	}
	return list, nil
}

func (r *Repository) SaveCheckIn(ctx context.Context, userID string, mood, urge int, contextText string) (model.CheckIn, error) {
	now := time.Now().UTC()
	newEntry := store.CheckIn{
		ID:        "chk_" + uuid.NewString()[:8],
		UserID:    userID,
		Mood:      mood,
		Urge:      urge,
		Context:   contextText,
		CreatedAt: now,
	}
	r.store.Lock()
	r.store.CheckIns = append(r.store.CheckIns, newEntry)
	r.store.Unlock()
	return newEntry, nil
}
