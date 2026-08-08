package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entreminder "github.com/gamblock-ai/gamblock-ai-backend/ent/reminderpreference"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/google/uuid"
)

const defaultReminderTime = "19:00"
const defaultReminderTimezone = "Asia/Jakarta"
const defaultReminderLocale = "id"

// DefaultReminderPreference returns the fallback opt-in preference shape when no
// row exists yet. It is intentionally not persisted here; callers decide when
// to upsert.
func DefaultReminderPreference(userID string) model.ReminderPreference {
	return model.ReminderPreference{
		ID:        "rem_" + uuid.NewString(),
		UserID:    userID,
		Enabled:   false,
		LocalTime: defaultReminderTime,
		Timezone:  defaultReminderTimezone,
		Locale:    defaultReminderLocale,
	}
}

func (r *Repository) ReminderPreference(ctx context.Context, userID string) (model.ReminderPreference, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		for _, pref := range r.store.ReminderPreferences {
			if pref.UserID == userID {
				return pref, nil
			}
		}
		return DefaultReminderPreference(userID), nil
	}
	row, err := r.db.ReminderPreference.Query().Where(entreminder.UserIDEQ(userID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return DefaultReminderPreference(userID), nil
		}
		return model.ReminderPreference{}, err
	}
	return reminderPreferenceFromEnt(row), nil
}

func (r *Repository) UpsertReminderPreference(ctx context.Context, userID string, enabled bool, localTime, timezone, locale string) (model.ReminderPreference, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.ReminderPreferences {
			if r.store.ReminderPreferences[index].UserID == userID {
				r.store.ReminderPreferences[index].Enabled = enabled
				r.store.ReminderPreferences[index].LocalTime = localTime
				r.store.ReminderPreferences[index].Timezone = timezone
				r.store.ReminderPreferences[index].Locale = locale
				r.store.ReminderPreferences[index].UpdatedAt = time.Now().UTC()
				return r.store.ReminderPreferences[index], nil
			}
		}
		pref := DefaultReminderPreference(userID)
		pref.Enabled = enabled
		pref.LocalTime = localTime
		pref.Timezone = timezone
		pref.Locale = locale
		pref.CreatedAt = time.Now().UTC()
		pref.UpdatedAt = pref.CreatedAt
		r.store.ReminderPreferences = append(r.store.ReminderPreferences, pref)
		return pref, nil
	}
	existing, err := r.db.ReminderPreference.Query().Where(entreminder.UserIDEQ(userID)).Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return model.ReminderPreference{}, err
	}
	var row *ent.ReminderPreference
	if err == nil {
		row, err = r.db.ReminderPreference.UpdateOneID(existing.ID).
			SetEnabled(enabled).
			SetLocalTime(localTime).
			SetTimezone(timezone).
			SetLocale(locale).
			Save(ctx)
	} else {
		row, err = r.db.ReminderPreference.Create().
			SetID("rem_" + uuid.NewString()).
			SetUserID(userID).
			SetEnabled(enabled).
			SetLocalTime(localTime).
			SetTimezone(timezone).
			SetLocale(locale).
			Save(ctx)
	}
	if err != nil {
		return model.ReminderPreference{}, err
	}
	r.RefreshStore(ctx)
	return reminderPreferenceFromEnt(row), nil
}

func (r *Repository) MarkReminderFired(ctx context.Context, userID string, firedAt time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.ReminderPreferences {
			if r.store.ReminderPreferences[index].UserID == userID {
				r.store.ReminderPreferences[index].LastFiredAt = &firedAt
				return nil
			}
		}
		return nil
	}
	if _, err := r.db.ReminderPreference.Update().
		Where(entreminder.UserIDEQ(userID)).
		SetLastFiredAt(firedAt).
		Save(ctx); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

// EnabledReminderPreferences lists every opted-in preference with its push
// subscriptions attached, used by the daily scheduler.
func (r *Repository) EnabledReminderPreferences(ctx context.Context) ([]store.ReminderPreference, error) {
	if r.db == nil {
		r.store.RLock()
		defer r.store.RUnlock()
		var out []store.ReminderPreference
		for _, pref := range r.store.ReminderPreferences {
			if pref.Enabled {
				out = append(out, pref)
			}
		}
		return out, nil
	}
	rows, err := r.db.ReminderPreference.Query().Where(entreminder.Enabled(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.ReminderPreference, 0, len(rows))
	for _, row := range rows {
		out = append(out, reminderPreferenceFromEnt(row))
	}
	return out, nil
}

func reminderPreferenceFromEnt(row *ent.ReminderPreference) model.ReminderPreference {
	return model.ReminderPreference{
		ID:          row.ID,
		UserID:      row.UserID,
		Enabled:     row.Enabled,
		LocalTime:   row.LocalTime,
		Timezone:    row.Timezone,
		Locale:      row.Locale,
		LastFiredAt: row.LastFiredAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
