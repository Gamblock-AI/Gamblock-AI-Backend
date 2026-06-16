package repository

import (
	"context"
	"fmt"
	"time"

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
		return model.User{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}, true
	}
	row, err := r.db.User.Query().Where(entuser.EmailEqualFold(email)).Only(ctx)
	if err != nil {
		return model.User{}, false
	}
	return model.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Role: row.Role.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, true
}

func (r *Repository) UserByID(ctx context.Context, id string) (model.User, bool) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, u := range snapshot.Users {
			if u.ID == id {
				return model.User{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}, true
			}
		}
		return model.User{}, false
	}
	row, err := r.db.User.Query().Where(entuser.IDEQ(id)).Only(ctx)
	if err != nil {
		return model.User{}, false
	}
	return model.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Role: row.Role.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, true
}

func (r *Repository) CreateUser(ctx context.Context, id, email, name string) (model.User, error) {
	if r.db == nil {
		newUser := store.User{
			ID:          id,
			Email:       email,
			DisplayName: name,
			Role:        "user",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		r.store.Lock()
		r.store.Users = append(r.store.Users, newUser)
		r.store.Unlock()
		return model.User{ID: newUser.ID, Email: newUser.Email, DisplayName: newUser.DisplayName, Role: newUser.Role, CreatedAt: newUser.CreatedAt, UpdatedAt: newUser.UpdatedAt}, nil
	}
	row, err := r.db.User.Create().
		SetID(id).
		SetEmail(email).
		SetDisplayName(name).
		SetRole(entuser.RoleUser).
		Save(ctx)
	if err != nil {
		return model.User{}, err
	}
	r.RefreshStore(ctx)
	return model.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Role: row.Role.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) GetUserByGoogleSubject(ctx context.Context, subject string) (model.User, error) {
	if r.db == nil {
		return model.User{}, fmt.Errorf("user not found")
	}
	row, err := r.db.User.Query().Where(entuser.GoogleSubjectEQ(subject)).Only(ctx)
	if err != nil {
		return model.User{}, err
	}
	return model.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Role: row.Role.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) CreateUserGoogle(ctx context.Context, id, email, name string, avatarURL *string, subject string) (model.User, error) {
	if r.db == nil {
		u := r.store.DefaultUser()
		return model.User{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}, nil
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
	return model.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Role: row.Role.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func (r *Repository) UpdateUserGoogle(ctx context.Context, id, name string, avatarURL *string, subject string) (model.User, error) {
	if r.db == nil {
		u := r.store.DefaultUser()
		return model.User{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Role: u.Role, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}, nil
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
	return model.User{ID: row.ID, Email: row.Email, DisplayName: row.DisplayName, Role: row.Role.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}
