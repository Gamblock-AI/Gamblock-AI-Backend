package seedscale

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
)

type LocalAccount struct {
	ID          string
	Email       string
	DisplayName string
	Phone       string
	Role        user.Role
	AvatarURL   string
}

var CoreLocalAccounts = []LocalAccount{
	{
		ID:          "usr_gading",
		Email:       "gading@gmail.com",
		DisplayName: "Gading",
		Phone:       "+62895363116378",
		Role:        user.RoleUser,
		AvatarURL:   "avatar/gading.webp",
	},
	{
		ID:          "usr_dery",
		Email:       "dery@gmail.com",
		DisplayName: "Dery",
		Phone:       "+6282377341268",
		Role:        user.RoleUser,
		AvatarURL:   "avatar/dery.webp",
	},
	{
		ID:          "usr_suci",
		Email:       "suci@gmail.com",
		DisplayName: "Suci",
		Phone:       "+6282385822192",
		Role:        user.RolePartner,
		AvatarURL:   "avatar/suci.webp",
	},
	{
		ID:          "usr_nasywa",
		Email:       "nasywa@gmail.com",
		DisplayName: "Nasywa",
		Phone:       "+6282328514811",
		Role:        user.RoleAdmin,
		AvatarURL:   "avatar/nasywa.webp",
	},
}

// SeedLocalAccounts installs only the core accounts for local zero-data testing.
// It is completely isolated from production/staging seeder assertions so local development
// can modify test accounts without risking production or staging constraints.
func SeedLocalAccounts(ctx context.Context, client *ent.Client) error {
	passwordHash, err := authn.HashPassword("password")
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	now := time.Now().UTC()
	for _, acc := range CoreLocalAccounts {
		existing, err := client.User.Get(ctx, acc.ID)
		if err == nil && existing != nil {
			// Account already exists, keep or update if necessary
			continue
		} else if err != nil && !ent.IsNotFound(err) {
			return fmt.Errorf("check local user %s: %w", acc.ID, err)
		}

		_, err = client.User.Create().
			SetID(acc.ID).
			SetEmail(acc.Email).
			SetDisplayName(acc.DisplayName).
			SetRole(acc.Role).
			SetPasswordHash(passwordHash).
			SetEmailVerifiedAt(now).
			SetPhoneE164(acc.Phone).
			SetPhoneVerifiedAt(now).
			SetAvatarURL(acc.AvatarURL).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("seed local user %s: %w", acc.ID, err)
		}
	}
	return nil
}
