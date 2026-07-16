package repository

import (
	"context"
	"fmt"
	"time"

	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) CreateUser(ctx context.Context, id, email, name string) (model.User, error) {
	return r.CreateUserWithPassword(ctx, id, email, name, "")
}

func (r *Repository) CreateUserWithPassword(ctx context.Context, id, email, name, passwordHash string) (model.User, error) {
	if r.db == nil {
		if _, exists := r.store.UserByEmail(email); exists {
			return model.User{}, fmt.Errorf("email already exists")
		}
		newUser := store.User{
			ID:           id,
			Email:        email,
			DisplayName:  name,
			Role:         "user",
			PasswordHash: passwordHash,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}
		r.store.Lock()
		r.store.Users = append(r.store.Users, newUser)
		r.store.Unlock()
		return newUser, nil
	}
	creator := r.db.User.Create().
		SetID(id).
		SetEmail(email).
		SetDisplayName(name).
		SetRole(entuser.RoleUser)
	if passwordHash != "" {
		creator.SetPasswordHash(passwordHash)
	}
	row, err := creator.Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}

func (r *Repository) CreateUserGoogle(ctx context.Context, id, email, name string, avatarURL *string, subject string) (model.User, error) {
	if r.db == nil {
		if _, exists := r.store.UserByEmail(email); exists {
			return model.User{}, fmt.Errorf("email already exists")
		}
		now := time.Now().UTC()
		user := model.User{ID: id, Email: email, DisplayName: name, Role: "user", GoogleSubject: subject, CreatedAt: now, UpdatedAt: now}
		r.store.Lock()
		r.store.Users = append(r.store.Users, user)
		r.store.Unlock()
		return user, nil
	}
	creator := r.db.User.Create().
		SetID(id).
		SetEmail(email).
		SetDisplayName(name).
		SetGoogleSubject(subject).
		SetRole(entuser.RoleUser)
	if avatarURL != nil {
		creator.SetAvatarURL(*avatarURL)
	}
	row, err := creator.Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}
