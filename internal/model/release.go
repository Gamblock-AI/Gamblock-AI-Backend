package model

import "time"

type Release struct {
	ID              string         `json:"id"`
	Version         string         `json:"version"`
	Platform        string         `json:"platform"`
	SHA256          string         `json:"sha256"`
	Status          string         `json:"status"`
	DownloadURL     string         `json:"download_url"`
	Metrics         map[string]any `json:"metrics"`
	PublishedAtText string         `json:"published_at_text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
