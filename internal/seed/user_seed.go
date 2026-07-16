package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
)

func SeedUsers(ctx context.Context, client *ent.Client) error {
	passwordHash, err := authn.HashPassword("password")
	if err != nil {
		return err
	}
	users := []struct {
		id, email, name string
		role            user.Role
	}{
		{"usr_gading", "gading@gmail.com", "Gading", user.RoleUser},
		{"usr_dery", "dery@gmail.com", "Dery", user.RoleUser},
		{"usr_suci", "suci@gmail.com", "Suci", user.RolePartner},
		{"usr_nasywa", "nasywa@gmail.com", "Nasywa", user.RolePlatformAdmin},
	}
	for _, item := range users {
		if _, err := client.User.Create().SetID(item.id).SetEmail(item.email).SetDisplayName(item.name).SetRole(item.role).SetPasswordHash(passwordHash).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
