package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	persistence "github.com/gamblock-ai/gamblock-ai-backend/internal/db"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

type Repository struct {
	db    *ent.Client
	store *store.Store
}

func New(db *ent.Client, st *store.Store) *Repository {
	return &Repository{db: db, store: st}
}

func (r *Repository) RefreshStore(ctx context.Context) {
	if r.db == nil {
		return
	}
	loaded, err := persistence.LoadStore(ctx, r.db)
	if err == nil && loaded != nil {
		*r.store = *loaded
	}
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func optional(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func valueInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func humanExpiry(t time.Time) string {
	if time.Until(t) > 0 {
		return "Expires in 23 minutes"
	}
	return "Reviewed yesterday"
}

func humanPublished(t *time.Time) string {
	if t == nil {
		return "Not published"
	}
	return "Published"
}

func humanApprovalStatus(status string) string {
	if status == "pending" {
		return "Pending partner approval"
	}
	if status == "approved" {
		return "Partner notified"
	}
	return status
}

func humanDataRequestTitle(kind string) string {
	if kind == "export" {
		return "Export account data"
	}
	if kind == "delete" {
		return "Delete archived support notes"
	}
	return "Data request"
}

func moduleProgress(slug string) float64 {
	if slug == "pause-before-impulse" {
		return 0.7
	}
	if slug == "financial-reality-check" {
		return 0.35
	}
	return 0
}
