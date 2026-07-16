package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) UserByEmail(ctx context.Context, email string) (model.User, bool) {
	if r.db == nil {
		u, ok := r.store.UserByEmail(email)
		if !ok {
			return model.User{}, false
		}
		return u, true
	}
	row, err := r.db.User.Query().Where(entuser.EmailEqualFold(email)).Only(ctx)
	if err != nil {
		return model.User{}, false
	}
	return userFromEnt(row), true
}

func (r *Repository) UserByID(ctx context.Context, id string) (model.User, bool) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, u := range snapshot.Users {
			if u.ID == id {
				return u, true
			}
		}
		return model.User{}, false
	}
	row, err := r.db.User.Query().Where(entuser.IDEQ(id)).Only(ctx)
	if err != nil {
		return model.User{}, false
	}
	return userFromEnt(row), true
}

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

func (r *Repository) GetUserByGoogleSubject(ctx context.Context, subject string) (model.User, error) {
	if r.db == nil {
		for _, user := range r.store.Snapshot().Users {
			if user.GoogleSubject == subject {
				return user, nil
			}
		}
		return model.User{}, fmt.Errorf("user not found")
	}
	row, err := r.db.User.Query().Where(entuser.GoogleSubjectEQ(subject)).Only(ctx)
	if err != nil {
		return model.User{}, err
	}
	return userFromEnt(row), nil
}

func (r *Repository) CreateUserGoogle(ctx context.Context, id, email, name string, avatarURL *string, subject string) (model.User, error) {
	if r.db == nil {
		if _, exists := r.store.UserByEmail(email); exists {
			return model.User{}, fmt.Errorf("email already exists")
		}
		now := time.Now().UTC()
		u := model.User{ID: id, Email: email, DisplayName: name, Role: "user", GoogleSubject: subject, CreatedAt: now, UpdatedAt: now}
		r.store.Lock()
		r.store.Users = append(r.store.Users, u)
		r.store.Unlock()
		return u, nil
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

func (r *Repository) UpdateUserGoogle(ctx context.Context, id, name string, avatarURL *string, subject string) (model.User, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			if r.store.Users[index].ID == id {
				r.store.Users[index].DisplayName = name
				r.store.Users[index].GoogleSubject = subject
				r.store.Users[index].UpdatedAt = time.Now().UTC()
				return r.store.Users[index], nil
			}
		}
		return model.User{}, fmt.Errorf("user not found")
	}
	updater := r.db.User.UpdateOneID(id).
		SetGoogleSubject(subject).
		SetDisplayName(name)
	if avatarURL != nil {
		updater.SetAvatarURL(*avatarURL)
	}
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
				return r.store.Users[index], nil
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

func userFromEnt(row *ent.User) model.User {
	return model.User{
		ID:            row.ID,
		Email:         row.Email,
		DisplayName:   row.DisplayName,
		Role:          row.Role.String(),
		PasswordHash:  value(row.PasswordHash),
		GoogleSubject: value(row.GoogleSubject),
		DisabledAt:    row.DisabledAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
