package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) UpdateUserGoogle(ctx context.Context, id, name string, avatarURL *string, subject string) (model.User, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			if r.store.Users[index].ID == id {
				r.store.Users[index].DisplayName = name
				r.store.Users[index].GoogleSubject = subject
				r.store.Users[index].UpdatedAt = time.Now().UTC()
				return userForResponse(r.store.Users[index]), nil
			}
		}
		return model.User{}, fmt.Errorf("user not found")
	}
	updater := r.db.User.UpdateOneID(id).
		SetGoogleSubject(subject).
		SetDisplayName(name)
	// Provider-hosted Google photos are intentionally not retained as account
	// avatars. Only images uploaded through the authenticated avatar flow are
	// exposed by this service.
	_ = avatarURL
	row, err := updater.Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}

func (r *Repository) UpdateUserDisplayName(ctx context.Context, id, displayName string) (model.User, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			if r.store.Users[index].ID == id {
				r.store.Users[index].DisplayName = displayName
				r.store.Users[index].UpdatedAt = time.Now().UTC()
				return userForResponse(r.store.Users[index]), nil
			}
		}
		return model.User{}, fmt.Errorf("user not found")
	}
	row, err := r.db.User.UpdateOneID(id).SetDisplayName(displayName).Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}

func (r *Repository) UpdateUserAvatar(ctx context.Context, id string, storageKey *string) (model.User, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			if r.store.Users[index].ID == id {
				r.store.Users[index].AvatarURL = storageKey
				r.store.Users[index].UpdatedAt = time.Now().UTC()
				return userForResponse(r.store.Users[index]), nil
			}
		}
		return model.User{}, fmt.Errorf("user not found")
	}
	updater := r.db.User.UpdateOneID(id)
	if storageKey == nil {
		updater.ClearAvatarURL()
	} else {
		updater.SetAvatarURL(*storageKey)
	}
	row, err := updater.Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}

func (r *Repository) UserAvatarStorageKey(ctx context.Context, id string) (string, bool) {
	if r.db == nil {
		for _, user := range r.store.Snapshot().Users {
			if user.ID == id && user.AvatarURL != nil && strings.HasPrefix(*user.AvatarURL, "avatar/") {
				return *user.AvatarURL, true
			}
		}
		return "", false
	}
	row, err := r.db.User.Get(ctx, id)
	if err != nil || row.AvatarURL == nil || !strings.HasPrefix(*row.AvatarURL, "avatar/") {
		return "", false
	}
	return *row.AvatarURL, true
}

func (r *Repository) UpdateUserPasswordHash(ctx context.Context, id, passwordHash string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			if r.store.Users[index].ID == id {
				r.store.Users[index].PasswordHash = passwordHash
				r.store.Users[index].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("user not found")
	}
	if _, err := r.db.User.UpdateOneID(id).SetPasswordHash(passwordHash).Save(ctx); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
