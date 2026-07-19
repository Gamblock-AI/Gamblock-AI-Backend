package repository

import (
	"context"
	"fmt"
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
				Mood: value(item.PromptKey), Status: item.Status.String(), IsFocus: item.IsFocus,
				CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
			})
		}
		return list, nil
	}
	r.store.RLock()
	defer r.store.RUnlock()
	var list []model.JournalEntry
	for _, e := range r.store.JournalEntries {
		if e.UserID == userID {
			status := e.Status
			if status == "" {
				status = "active"
			}
			list = append(list, model.JournalEntry{
				ID:        e.ID,
				UserID:    e.UserID,
				Text:      e.Text,
				Mood:      e.Mood,
				Status:    status,
				IsFocus:   e.IsFocus,
				CreatedAt: e.CreatedAt,
				UpdatedAt: e.UpdatedAt,
			})
		}
	}
	return list, nil
}

func (r *Repository) CreateReflection(ctx context.Context, userID, text, mood string) (model.JournalEntry, error) {
	return r.CreateReflectionEntry(ctx, model.JournalEntry{UserID: userID, Text: text, Mood: mood, Status: "active"})
}

func (r *Repository) CreateReflectionEntry(ctx context.Context, entry model.JournalEntry) (model.JournalEntry, error) {
	now := time.Now().UTC()
	if r.db != nil {
		if entry.IsFocus {
			if _, err := r.db.Reflection.Update().Where(reflection.UserIDEQ(entry.UserID), reflection.IsFocusEQ(true)).SetIsFocus(false).Save(ctx); err != nil {
				return model.JournalEntry{}, err
			}
		}
		item, err := r.db.Reflection.Create().
			SetID("ref_" + uuid.NewString()[:8]).
			SetUserID(entry.UserID).
			SetContentEncrypted(entry.Text).
			SetNillablePromptKey(optional(entry.Mood)).
			SetStatus(reflection.Status(entry.Status)).
			SetIsFocus(entry.IsFocus).
			Save(ctx)
		if err != nil {
			return model.JournalEntry{}, err
		}
		r.RefreshStore(ctx)
		return model.JournalEntry{
			ID: item.ID, UserID: item.UserID, Text: item.ContentEncrypted,
			Mood: value(item.PromptKey), Status: item.Status.String(), IsFocus: item.IsFocus,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		}, nil
	}
	newEntry := store.JournalEntry{
		ID:        "ref_" + uuid.NewString()[:8],
		UserID:    entry.UserID,
		Text:      entry.Text,
		Mood:      entry.Mood,
		Status:    entry.Status,
		IsFocus:   entry.IsFocus,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.store.Lock()
	if entry.IsFocus {
		for index := range r.store.JournalEntries {
			if r.store.JournalEntries[index].UserID == entry.UserID {
				r.store.JournalEntries[index].IsFocus = false
			}
		}
	}
	r.store.JournalEntries = append(r.store.JournalEntries, newEntry)
	r.store.Unlock()
	return model.JournalEntry{
		ID:        newEntry.ID,
		UserID:    newEntry.UserID,
		Text:      newEntry.Text,
		Mood:      newEntry.Mood,
		Status:    newEntry.Status,
		IsFocus:   newEntry.IsFocus,
		CreatedAt: newEntry.CreatedAt,
		UpdatedAt: newEntry.UpdatedAt,
	}, nil
}

func (r *Repository) UpdateReflectionEntry(ctx context.Context, entry model.JournalEntry) (model.JournalEntry, error) {
	if r.db != nil {
		if entry.IsFocus {
			if _, err := r.db.Reflection.Update().Where(reflection.UserIDEQ(entry.UserID), reflection.IDNEQ(entry.ID), reflection.IsFocusEQ(true)).SetIsFocus(false).Save(ctx); err != nil {
				return model.JournalEntry{}, err
			}
		}
		row, err := r.db.Reflection.Query().Where(reflection.IDEQ(entry.ID), reflection.UserIDEQ(entry.UserID)).Only(ctx)
		if err != nil {
			return model.JournalEntry{}, err
		}
		row, err = row.Update().SetContentEncrypted(entry.Text).SetStatus(reflection.Status(entry.Status)).SetIsFocus(entry.IsFocus).Save(ctx)
		if err != nil {
			return model.JournalEntry{}, err
		}
		r.RefreshStore(ctx)
		return model.JournalEntry{ID: row.ID, UserID: row.UserID, Text: row.ContentEncrypted, Mood: value(row.PromptKey), Status: row.Status.String(), IsFocus: row.IsFocus, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
	}
	r.store.Lock()
	defer r.store.Unlock()
	if entry.IsFocus {
		for index := range r.store.JournalEntries {
			if r.store.JournalEntries[index].UserID == entry.UserID {
				r.store.JournalEntries[index].IsFocus = false
			}
		}
	}
	for index := range r.store.JournalEntries {
		current := &r.store.JournalEntries[index]
		if current.ID == entry.ID && current.UserID == entry.UserID {
			current.Text, current.Status, current.IsFocus = entry.Text, entry.Status, entry.IsFocus
			current.UpdatedAt = time.Now().UTC()
			return *current, nil
		}
	}
	return model.JournalEntry{}, fmt.Errorf("reflection not found")
}
