package repository

import (
	"context"
	"fmt"

	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) UserByEmail(ctx context.Context, email string) (model.User, bool) {
	if r.db == nil {
		user, ok := r.store.UserByEmail(email)
		if !ok {
			return model.User{}, false
		}
		return userForResponse(user), true
	}
	row, err := r.db.User.Query().Where(entuser.EmailEqualFold(email)).Only(ctx)
	if err != nil {
		return model.User{}, false
	}
	return userFromEnt(row), true
}

func (r *Repository) UserByID(ctx context.Context, id string) (model.User, bool) {
	if r.db == nil {
		for _, user := range r.store.Snapshot().Users {
			if user.ID == id {
				return userForResponse(user), true
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

func (r *Repository) GetUserByGoogleSubject(ctx context.Context, subject string) (model.User, error) {
	if r.db == nil {
		for _, user := range r.store.Snapshot().Users {
			if user.GoogleSubject == subject {
				return userForResponse(user), nil
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
