package model

import "time"

type Partner struct {
	ID               string    `json:"id"`
	UserID           string    `json:"-"`
	PartnerUserID    string    `json:"-"`
	InviteTokenHash  string    `json:"-"`
	RelationshipRole string    `json:"relationship_role"`
	Name             string    `json:"name"`
	Contact          string    `json:"contact"`
	Status           string    `json:"status"`
	PartnerEmail     string    `json:"partner_email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
