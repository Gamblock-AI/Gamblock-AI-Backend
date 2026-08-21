package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

func (r *Repository) ListAdminAccountsPaginated(ctx context.Context, query model.PaginationQuery) (model.PaginatedList[model.AdminAccount], error) {
	page, limit, offset := query.Normalize(10)
	role := strings.TrimSpace(query.Role)
	search := strings.ToLower(strings.TrimSpace(query.Query))

	if r.db == nil {
		users := r.store.Snapshot().Users
		var filtered []model.AdminAccount
		for _, user := range users {
			if !model.IsAccountRole(user.Role) {
				continue
			}
			if role != "" && user.Role != role {
				continue
			}
			if search != "" {
				nameMatch := strings.Contains(strings.ToLower(user.DisplayName), search)
				emailMatch := strings.Contains(strings.ToLower(user.Email), search)
				if !nameMatch && !emailMatch {
					continue
				}
			}
			filtered = append(filtered, adminAccountFromUser(user))
		}
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Email < filtered[j].Email })
		total := len(filtered)
		start := offset
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		paged := filtered[start:end]
		return model.NewPaginatedList(paged, total, page, limit), nil
	}

	qb := r.db.User.Query().Where(entuser.RoleIn(entuser.RoleUser, entuser.RolePartner, entuser.RoleAdmin))
	if role != "" {
		qb = qb.Where(entuser.RoleEQ(entuser.Role(role)))
	}
	if search != "" {
		qb = qb.Where(entuser.Or(
			entuser.DisplayNameContainsFold(search),
			entuser.EmailContainsFold(search),
		))
	}

	total, err := qb.Count(ctx)
	if err != nil {
		return model.PaginatedList[model.AdminAccount]{}, err
	}

	rows, err := qb.Order(ent.Asc(entuser.FieldEmail)).Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return model.PaginatedList[model.AdminAccount]{}, err
	}

	items := make([]model.AdminAccount, 0, len(rows))
	for _, row := range rows {
		items = append(items, adminAccountFromUser(userFromEnt(row)))
	}
	return model.NewPaginatedList(items, total, page, limit), nil
}

func (r *Repository) ListActiveAdminPhones(ctx context.Context) ([]string, error) {
	if r.db == nil {
		users := r.store.Snapshot().Users
		var phones []string
		for _, user := range users {
			if user.Role == "admin" && user.DisabledAt == nil && strings.TrimSpace(user.PhoneE164) != "" {
				phones = append(phones, user.PhoneE164)
			}
		}
		return phones, nil
	}
	rows, err := r.db.User.Query().
		Where(
			entuser.RoleEQ(entuser.RoleAdmin),
			entuser.DisabledAtIsNil(),
			entuser.PhoneE164NotNil(),
			entuser.PhoneE164NEQ(""),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	phones := make([]string, 0, len(rows))
	for _, row := range rows {
		phone := strings.TrimSpace(value(row.PhoneE164))
		if phone != "" {
			phones = append(phones, phone)
		}
	}
	return phones, nil
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
