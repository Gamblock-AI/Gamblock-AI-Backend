package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reflection"
	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetReflections(ctx context.Context, userID string) ([]model.JournalEntry, error) {
	if r.db != nil {
		rows, err := r.db.Reflection.Query().
			Where(reflection.UserID(userID)).
			Order(ent.Desc(reflection.FieldCreatedAt)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		list := make([]model.JournalEntry, 0, len(rows))
		for _, item := range rows {
			list = append(list, model.JournalEntry{
				ID: item.ID, UserID: item.UserID, Text: item.ContentEncrypted,
				Mood: value(item.PromptKey), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
			})
		}
		return list, nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	var list []model.JournalEntry
	for _, e := range r.store.JournalEntries {
		if e.UserID == userID {
			list = append(list, model.JournalEntry{
				ID:        e.ID,
				UserID:    e.UserID,
				Text:      e.Text,
				Mood:      e.Mood,
				CreatedAt: e.CreatedAt,
				UpdatedAt: e.UpdatedAt,
			})
		}
	}
	return list, nil
}

func (r *Repository) CreateReflection(ctx context.Context, userID, text, mood string) (model.JournalEntry, error) {
	now := time.Now().UTC()
	if r.db != nil {
		item, err := r.db.Reflection.Create().
			SetID("ref_" + uuid.NewString()[:8]).
			SetUserID(userID).
			SetContentEncrypted(text).
			SetNillablePromptKey(optional(mood)).
			Save(ctx)
		if err != nil {
			return model.JournalEntry{}, err
		}
		r.RefreshStore(ctx)
		return model.JournalEntry{
			ID: item.ID, UserID: item.UserID, Text: item.ContentEncrypted,
			Mood: value(item.PromptKey), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		}, nil
	}
	newEntry := store.JournalEntry{
		ID:        "ref_" + uuid.NewString()[:8],
		UserID:    userID,
		Text:      text,
		Mood:      mood,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.store.Lock()
	r.store.JournalEntries = append(r.store.JournalEntries, newEntry)
	r.store.Unlock()
	return model.JournalEntry{
		ID:        newEntry.ID,
		UserID:    newEntry.UserID,
		Text:      newEntry.Text,
		Mood:      newEntry.Mood,
		CreatedAt: newEntry.CreatedAt,
		UpdatedAt: newEntry.UpdatedAt,
	}, nil
}
