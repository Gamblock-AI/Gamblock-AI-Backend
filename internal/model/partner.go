package model

import "time"

type Partner struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Contact      string    `json:"contact"`
	Status       string    `json:"status"`
	PartnerEmail string    `json:"partner_email"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
