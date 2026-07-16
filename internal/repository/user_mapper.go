package repository

import (
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func userFromEnt(row *ent.User) model.User {
	return model.User{
		ID:            row.ID,
		Email:         row.Email,
		DisplayName:   row.DisplayName,
		Role:          row.Role.String(),
		PasswordHash:  value(row.PasswordHash),
		GoogleSubject: value(row.GoogleSubject),
		DisabledAt:    row.DisabledAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
