package seed

import (
	"context"
	"time"

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
		{"usr_nasywa", "nasywa@gmail.com", "Nasywa", user.RoleAdmin},
		{"usr_demo_student", "student@gmail.com", "Demo Student", user.RoleUser},
		{"usr_demo_partner", "partner@gmail.com", "Demo Partner", user.RolePartner},
	}
	for _, item := range users {
		// Skip users that already exist so the seeder is re-runnable.
		if _, err := client.User.Get(ctx, item.id); err == nil {
			continue
		} else if !ent.IsNotFound(err) {
			return err
		}
		create := client.User.Create().SetID(item.id).SetEmail(item.email).SetDisplayName(item.name).SetRole(item.role).SetPasswordHash(passwordHash).SetEmailVerifiedAt(time.Now().UTC())
		if item.role == user.RolePartner {
			create.SetPhoneE164("+6281200000000").SetPhoneVerifiedAt(time.Now().UTC())
		}
		if _, err := create.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
