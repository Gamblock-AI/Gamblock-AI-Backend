package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func adminAccountFromUser(user model.User) model.AdminAccount {
	return model.AdminAccount{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Role: user.Role,
		EmailVerifiedAt: user.EmailVerifiedAt, DisabledAt: user.DisabledAt,
		MustChangePassword: user.MustChangePassword, CreatedAt: user.CreatedAt}
}

func (r *Repository) ListAdminAccounts(ctx context.Context) ([]model.AdminAccount, error) {
	if r.db == nil {
		users := r.store.Snapshot().Users
		items := make([]model.AdminAccount, 0, len(users))
		for _, user := range users {
			if model.IsAccountRole(user.Role) {
				items = append(items, adminAccountFromUser(user))
			}
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Email < items[j].Email })
		return items, nil
	}
	rows, err := r.db.User.Query().Where(entuser.RoleIn(entuser.RoleUser, entuser.RolePartner, entuser.RoleAdmin)).
		Order(ent.Asc(entuser.FieldEmail)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.AdminAccount, 0, len(rows))
	for _, row := range rows {
		items = append(items, adminAccountFromUser(userFromEnt(row)))
	}
	return items, nil
}

func (r *Repository) SetAccountDisabled(ctx context.Context, id string, disabled bool, now time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Users {
			if r.store.Users[index].ID != id {
				continue
			}
			if disabled {
				r.store.Users[index].DisabledAt = &now
			} else {
				r.store.Users[index].DisabledAt = nil
			}
			r.store.Users[index].UpdatedAt = now
			return nil
		}
		return fmt.Errorf("account not found")
	}
	update := r.db.User.UpdateOneID(id)
	if disabled {
		update.SetDisabledAt(now)
	} else {
		update.ClearDisabledAt()
	}
	if _, err := update.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("account not found")
		}
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
