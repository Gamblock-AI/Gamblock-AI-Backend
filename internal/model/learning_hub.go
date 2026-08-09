package model

import "time"

type Institution struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LearningCluster struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type AcademicProgram struct {
	ID                 string `json:"id"`
	InstitutionID      string `json:"institution_id"`
	Slug               string `json:"slug"`
	Name               string `json:"name"`
	Degree             string `json:"degree"`
	PrimaryClusterSlug string `json:"primary_cluster_slug"`
	SortOrder          int    `json:"sort_order"`
}

type LearningProgress struct {
	UserID              string     `json:"-"`
	ItemID              string     `json:"item_id"`
	State               string     `json:"state"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	ReflectionEncrypted string     `json:"-"`
	OutcomeEncrypted    string     `json:"-"`
}

type LearningItem struct {
	ID                  string            `json:"id"`
	Slug                string            `json:"slug"`
	Kind                string            `json:"kind"`
	Title               string            `json:"title"`
	Summary             string            `json:"summary"`
	Provider            string            `json:"provider,omitempty"`
	ProviderDescription string            `json:"provider_description,omitempty"`
	URL                 string            `json:"url,omitempty"`
	ProviderLogoURL     string            `json:"provider_logo_url,omitempty"`
	ThumbnailURL    string            `json:"thumbnail_url,omitempty"`
	Cost            string            `json:"cost,omitempty"`
	Certificate     string            `json:"certificate,omitempty"`
	Language        []string          `json:"language,omitempty"`
	Difficulty      string            `json:"difficulty,omitempty"`
	DurationMinutes int               `json:"duration_minutes,omitempty"`
	Outcomes        []string          `json:"outcomes,omitempty"`
	Prerequisites   string            `json:"prerequisites,omitempty"`
	Clusters        []string          `json:"clusters,omitempty"`
	Programs        []string          `json:"programs,omitempty"`
	CareerSnapshot  string            `json:"career_snapshot,omitempty"`
	ReviewedAt      string            `json:"reviewed_at,omitempty"`
	ExpiresAt       *time.Time        `json:"expires_at,omitempty"`
	Steps           []string          `json:"steps,omitempty"`
	Projects        []string          `json:"projects,omitempty"`
	Progress        *LearningProgress `json:"progress,omitempty"`
}

// LearningItemDraft is the editable CMS payload. The structured document
// contains supporting metadata (source, outcomes, audience, and review data)
// while titles/summaries remain first-class bilingual fields for catalog
// queries.
type LearningItemDraft struct {
	Slug      string         `json:"slug"`
	Kind      string         `json:"kind"`
	TitleID   string         `json:"title_id"`
	TitleEN   string         `json:"title_en"`
	SummaryID string         `json:"summary_id"`
	SummaryEN string         `json:"summary_en"`
	Document  map[string]any `json:"document"`
}

type AdminLearningItem struct {
	LearningItem
	TitleID           string         `json:"title_id"`
	TitleEN           string         `json:"title_en"`
	SummaryID         string         `json:"summary_id"`
	SummaryEN         string         `json:"summary_en"`
	Status            string         `json:"status"`
	DraftRevision     int            `json:"draft_revision"`
	PublishedRevision int            `json:"published_revision"`
	DraftDocument     map[string]any `json:"draft_document"`
	PublishedDocument map[string]any `json:"published_document,omitempty"`
	PublishedAt       *time.Time     `json:"published_at,omitempty"`
	ArchivedAt        *time.Time     `json:"archived_at,omitempty"`
	CreatedBy         string         `json:"created_by,omitempty"`
	UpdatedBy         string         `json:"updated_by,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type LearningRevision struct {
	ID        string         `json:"id"`
	ItemID    string         `json:"item_id"`
	Revision  int            `json:"revision"`
	Document  map[string]any `json:"document"`
	Kind      string         `json:"kind"`
	CreatedBy string         `json:"created_by"`
	CreatedAt time.Time      `json:"created_at"`
}

type AdminLearningCluster struct {
	LearningCluster
	TitleID       string `json:"title_id"`
	TitleEN       string `json:"title_en"`
	DescriptionID string `json:"description_id"`
	DescriptionEN string `json:"description_en"`
	Active        bool   `json:"active"`
}

type AdminAcademicProgram struct {
	AcademicProgram
	Active bool `json:"active"`
}

type LearningHubTaxonomy struct {
	Institution Institution            `json:"institution"`
	Clusters    []AdminLearningCluster `json:"clusters"`
	Programs    []AdminAcademicProgram `json:"programs"`
}

type LearningClusterInput struct {
	Slug          string `json:"slug"`
	TitleID       string `json:"title_id"`
	TitleEN       string `json:"title_en"`
	DescriptionID string `json:"description_id"`
	DescriptionEN string `json:"description_en"`
	SortOrder     int    `json:"sort_order"`
	Active        *bool  `json:"active,omitempty"`
}

type AcademicProgramInput struct {
	Slug               string `json:"slug"`
	Name               string `json:"name"`
	Degree             string `json:"degree"`
	PrimaryClusterSlug string `json:"primary_cluster_slug"`
	SortOrder          int    `json:"sort_order"`
	Active             *bool  `json:"active,omitempty"`
}

type LearningCatalog struct {
	Clusters   []LearningCluster  `json:"clusters"`
	Programs   []AcademicProgram  `json:"programs"`
	Items      []LearningItem     `json:"items"`
	Progress   []LearningProgress `json:"progress"`
	Experience ExperienceProgress `json:"experience"`
}

type LearningStateInput struct {
	State string `json:"state"`
}

type LearningCheckpointInput struct {
	Reflection string `json:"reflection"`
	Outcome    string `json:"outcome"`
}

type LearningCheckpointResult struct {
	Progress   LearningProgress   `json:"progress"`
	EXPGranted bool               `json:"exp_granted"`
	CapReached bool               `json:"cap_reached"`
	Experience ExperienceProgress `json:"experience"`
}

type ExperienceGrant struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	SourceKind     string    `json:"source_kind"`
	SourceID       string    `json:"source_id"`
	GrantDate      string    `json:"grant_date"`
	Amount         int       `json:"amount"`
	IdempotencyKey string    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}
