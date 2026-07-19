package model

import "time"

// RichTextDocument stores the allowlisted TipTap/ProseMirror JSON tree. The
// backend validates nodes and marks before a document can be published; raw
// HTML is never part of the education contract.
type RichTextDocument map[string]any

type EducationSource struct {
	Title       string    `json:"title"`
	Publisher   string    `json:"publisher"`
	URL         string    `json:"url"`
	PublishedAt string    `json:"published_at,omitempty"`
	AccessedAt  time.Time `json:"accessed_at"`
}

type EducationChoice struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type EducationKnowledgeCheck struct {
	ID              string            `json:"id"`
	Question        string            `json:"question"`
	Choices         []EducationChoice `json:"choices"`
	CorrectChoiceID string            `json:"correct_choice_id,omitempty"`
	Explanation     string            `json:"explanation,omitempty"`
	Required        bool              `json:"required"`
}

type EducationSectionTranslation struct {
	Title          string                   `json:"title"`
	Content        RichTextDocument         `json:"content"`
	KnowledgeCheck *EducationKnowledgeCheck `json:"knowledge_check,omitempty"`
}

type EducationSection struct {
	ID           string                                 `json:"id"`
	SortOrder    int                                    `json:"sort_order"`
	Required     bool                                   `json:"required"`
	Translations map[string]EducationSectionTranslation `json:"translations"`
}

type EducationTranslation struct {
	Title             string `json:"title"`
	Summary           string `json:"summary"`
	LearningObjective string `json:"learning_objective"`
	Disclaimer        string `json:"disclaimer"`
	ReviewerRole      string `json:"reviewer_role,omitempty"`
}

type EducationThumbnail struct {
	MediaID   string            `json:"media_id"`
	SortOrder int               `json:"sort_order"`
	AltText   map[string]string `json:"alt_text"`
}

// EducationDocument is a revision-safe bilingual module snapshot. Section and
// check identifiers are shared across locales so progress survives language
// changes.
type EducationDocument struct {
	Audience         string                          `json:"audience"`
	ExperienceType   string                          `json:"experience_type"`
	Category         string                          `json:"category"`
	EstimatedMinutes int                             `json:"estimated_minutes"`
	ReviewerName     string                          `json:"reviewer_name"`
	ReviewerRole     string                          `json:"reviewer_role"`
	ReviewedAt       string                          `json:"reviewed_at"`
	Translations     map[string]EducationTranslation `json:"translations"`
	Sections         []EducationSection              `json:"sections"`
	Thumbnails       []EducationThumbnail            `json:"thumbnails"`
	Sources          []EducationSource               `json:"sources"`
}

type EducationMedia struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Purpose      string    `json:"purpose"`
	MediaType    string    `json:"media_type"`
	MIMEType     string    `json:"mime_type"`
	StorageKey   string    `json:"storage_key,omitempty"`
	ExternalURL  string    `json:"external_url,omitempty"`
	OriginalName string    `json:"original_name,omitempty"`
	SizeBytes    int64     `json:"size_bytes"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	SHA256       string    `json:"sha256,omitempty"`
	Status       string    `json:"status"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EducationProgress struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	ModuleID            string     `json:"module_id"`
	Revision            int        `json:"revision"`
	CompletedSectionIDs []string   `json:"completed_section_ids"`
	OpenedMediaIDs      []string   `json:"opened_media_ids"`
	CorrectCheckIDs     []string   `json:"correct_check_ids"`
	ProgressPercent     int        `json:"progress_percent"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type EducationModule struct {
	ID                string             `json:"id"`
	Slug              string             `json:"slug"`
	Title             string             `json:"title"`
	Summary           string             `json:"summary"`
	BodyMarkdown      string             `json:"body_markdown,omitempty"`
	EstimatedMinutes  int                `json:"estimated_minutes"`
	Progress          float64            `json:"progress"`
	ProgressPercent   int                `json:"progress_percent"`
	Status            string             `json:"status"`
	DraftDocument     EducationDocument  `json:"draft_document"`
	PublishedDocument *EducationDocument `json:"published_document,omitempty"`
	DraftRevision     int                `json:"draft_revision"`
	PublishedRevision int                `json:"published_revision"`
	PublishedAt       *time.Time         `json:"published_at,omitempty"`
	ArchivedAt        *time.Time         `json:"archived_at,omitempty"`
	CreatedBy         string             `json:"created_by,omitempty"`
	UpdatedBy         string             `json:"updated_by,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type LocalizedEducationSection struct {
	ID             string                   `json:"id"`
	SortOrder      int                      `json:"sort_order"`
	Required       bool                     `json:"required"`
	Title          string                   `json:"title"`
	Content        RichTextDocument         `json:"content"`
	KnowledgeCheck *EducationKnowledgeCheck `json:"knowledge_check,omitempty"`
}

type LocalizedEducationModule struct {
	ID                string                      `json:"id"`
	Slug              string                      `json:"slug"`
	Locale            string                      `json:"locale"`
	Title             string                      `json:"title"`
	Summary           string                      `json:"summary"`
	LearningObjective string                      `json:"learning_objective"`
	Disclaimer        string                      `json:"disclaimer"`
	Category          string                      `json:"category"`
	Audience          string                      `json:"audience"`
	ExperienceType    string                      `json:"experience_type"`
	EstimatedMinutes  int                         `json:"estimated_minutes"`
	ReviewerName      string                      `json:"reviewer_name"`
	ReviewerRole      string                      `json:"reviewer_role"`
	ReviewedAt        string                      `json:"reviewed_at"`
	Revision          int                         `json:"revision"`
	Thumbnails        []EducationThumbnail        `json:"thumbnails"`
	ThumbnailURLs     map[string]string           `json:"thumbnail_urls"`
	MediaURLs         map[string]string           `json:"media_urls"`
	Sources           []EducationSource           `json:"sources"`
	Sections          []LocalizedEducationSection `json:"sections"`
	Progress          EducationProgress           `json:"progress"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

type EducationProgressInput struct {
	CompletedSectionIDs []string `json:"completed_section_ids"`
	OpenedMediaIDs      []string `json:"opened_media_ids"`
}

type EducationCheckAnswer struct {
	ChoiceID string `json:"choice_id"`
}

type EducationCheckResult struct {
	Correct     bool              `json:"correct"`
	Explanation string            `json:"explanation"`
	Progress    EducationProgress `json:"progress"`
}
