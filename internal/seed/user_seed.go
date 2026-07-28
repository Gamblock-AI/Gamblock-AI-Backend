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
		id, email, name, phone string
		role                   user.Role
	}{
		{"usr_gading", "gading@gmail.com", "Gading", "+6281200000001", user.RoleUser},
		{"usr_dery", "dery@gmail.com", "Dery", "+6281200000002", user.RoleUser},
		{"usr_suci", "suci@gmail.com", "Suci", "+6281200000000", user.RolePartner},
		{"usr_nasywa", "nasywa@gmail.com", "Nasywa", "+6281200000003", user.RoleAdmin},
		{"usr_demo_student", "student@gmail.com", "Demo Student", "+6281200000004", user.RoleUser},
		{"usr_demo_partner", "partner@gmail.com", "Demo Partner", "+6281200000005", user.RolePartner},
	}
	for _, item := range users {
		// Skip users that already exist so the seeder is re-runnable.
		if _, err := client.User.Get(ctx, item.id); err == nil {
			continue
		} else if !ent.IsNotFound(err) {
			return err
		}
		create := client.User.Create().SetID(item.id).SetEmail(item.email).SetDisplayName(item.name).SetRole(item.role).SetPasswordHash(passwordHash).SetEmailVerifiedAt(time.Now().UTC()).SetPhoneE164(item.phone).SetPhoneVerifiedAt(time.Now().UTC())
		if _, err := create.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
