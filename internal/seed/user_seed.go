package seed

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
)

//go:embed assets/avatars/*.webp
var userAvatarAssets embed.FS

func SeedUsers(ctx context.Context, client *ent.Client) error {
	passwordHash, err := authn.HashPassword("password")
	if err != nil {
		return err
	}
	avatarRoot := "./var/media/avatars"
	if err := os.MkdirAll(avatarRoot, 0o750); err != nil {
		return err
	}

	users := []struct {
		id, email, name, phone, avatarFile string
		role                               user.Role
	}{
		{"usr_gading", "gading@gmail.com", "Gading", "+62895363116378", "gading.webp", user.RoleUser},
		{"usr_dery", "dery@gmail.com", "Dery", "+6282377341268", "dery.webp", user.RoleUser},
		{"usr_suci", "suci@gmail.com", "Suci", "+6282385822192", "suci.webp", user.RolePartner},
		{"usr_nasywa", "nasywa@gmail.com", "Nasywa", "+6282328514811", "nasywa.webp", user.RoleAdmin},
	}
	for _, item := range users {
		storageKey := "avatar/" + item.avatarFile
		if data, err := userAvatarAssets.ReadFile("assets/avatars/" + item.avatarFile); err == nil {
			target := filepath.Join(avatarRoot, item.avatarFile)
			_ = os.WriteFile(target, data, 0o640)
		}

		existing, err := client.User.Get(ctx, item.id)
		switch {
		case err == nil:
			// Retrofit avatar for existing seeded users if missing or not set.
			if existing.AvatarURL == nil || *existing.AvatarURL == "" {
				_ = client.User.UpdateOneID(item.id).SetAvatarURL(storageKey).Exec(ctx)
			}
			continue
		case !ent.IsNotFound(err):
			return err
		}

		create := client.User.Create().
			SetID(item.id).
			SetEmail(item.email).
			SetDisplayName(item.name).
			SetRole(item.role).
			SetPasswordHash(passwordHash).
			SetEmailVerifiedAt(time.Now().UTC()).
			SetPhoneE164(item.phone).
			SetPhoneVerifiedAt(time.Now().UTC()).
			SetAvatarURL(storageKey)
		if _, err := create.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
