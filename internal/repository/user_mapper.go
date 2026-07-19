package repository

import (
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func avatarRoute(userID string, storageKey *string) *string {
	if storageKey == nil || !strings.HasPrefix(*storageKey, "avatar/") {
		return nil
	}
	route := "/v1/users/" + userID + "/avatar"
	return &route
}

func userForResponse(user model.User) model.User {
	user.AvatarURL = avatarRoute(user.ID, user.AvatarURL)
	return user
}

func userFromEnt(row *ent.User) model.User {
	return userForResponse(model.User{
		ID:                 row.ID,
		Email:              row.Email,
		DisplayName:        row.DisplayName,
		AvatarURL:          row.AvatarURL,
		Role:               row.Role.String(),
		MustChangePassword: row.MustChangePassword,
		EmailVerifiedAt:    row.EmailVerifiedAt,
		PhoneE164:          value(row.PhoneE164),
		PhoneVerifiedAt:    row.PhoneVerifiedAt,
		ExperiencePoints:   row.ExperiencePoints,
		PasswordHash:       value(row.PasswordHash),
		GoogleSubject:      value(row.GoogleSubject),
		DisabledAt:         row.DisabledAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	})
}
