package model

import "time"

type EducationModule struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	Summary          string    `json:"summary"`
	BodyMarkdown     string    `json:"body_markdown"`
	EstimatedMinutes int       `json:"estimated_minutes"`
	Progress         float64   `json:"progress"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
