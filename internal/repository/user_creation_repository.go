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
	return r.CreateUserWithPassword(ctx, id, email, name, "", "user")
}

func (r *Repository) CreateUserWithPassword(ctx context.Context, id, email, name, passwordHash, role string) (model.User, error) {
	return r.CreateProvisionedUser(ctx, id, email, name, passwordHash, role, false)
}

func (r *Repository) CreateProvisionedUser(ctx context.Context, id, email, name, passwordHash, role string, mustChangePassword bool) (model.User, error) {
	if r.db == nil {
		if _, exists := r.store.UserByEmail(email); exists {
			return model.User{}, fmt.Errorf("email already exists")
		}
		newUser := store.User{
			ID:                 id,
			Email:              email,
			DisplayName:        name,
			Role:               role,
			PasswordHash:       passwordHash,
			MustChangePassword: mustChangePassword,
			CreatedAt:          time.Now().UTC(),
			UpdatedAt:          time.Now().UTC(),
		}
		r.store.Lock()
		r.store.Users = append(r.store.Users, newUser)
		r.store.Unlock()
		return userForResponse(newUser), nil
	}
	creator := r.db.User.Create().
		SetID(id).
		SetEmail(email).
		SetDisplayName(name).
		SetRole(entuser.Role(role)).
		SetMustChangePassword(mustChangePassword)
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

func (r *Repository) CreateUserGoogle(ctx context.Context, id, email, name string, _ *string, subject, role string) (model.User, error) {
	if r.db == nil {
		if _, exists := r.store.UserByEmail(email); exists {
			return model.User{}, fmt.Errorf("email already exists")
		}
		now := time.Now().UTC()
		user := model.User{ID: id, Email: email, DisplayName: name, Role: role, GoogleSubject: subject, EmailVerifiedAt: &now, CreatedAt: now, UpdatedAt: now}
		r.store.Lock()
		r.store.Users = append(r.store.Users, user)
		r.store.Unlock()
		return userForResponse(user), nil
	}
	creator := r.db.User.Create().
		SetID(id).
		SetEmail(email).
		SetDisplayName(name).
		SetGoogleSubject(subject).
		SetRole(entuser.Role(role)).
		SetEmailVerifiedAt(time.Now().UTC())
	row, err := creator.Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return userFromEnt(row), nil
}
