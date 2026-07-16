package model

import "time"

type Release struct {
	ID              string         `json:"id"`
	Version         string         `json:"version"`
	Platform        string         `json:"platform"`
	SHA256          string         `json:"sha256"`
	ArtifactPath    string         `json:"-"`
	ContractVersion string         `json:"contract_version,omitempty"`
	Threshold       float64        `json:"threshold,omitempty"`
	Status          string         `json:"status"`
	DownloadURL     string         `json:"download_url"`
	Metrics         map[string]any `json:"metrics"`
	PublishedAtText string         `json:"published_at_text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
