package repository

import (
	"context"
	"fmt"
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
	if err != nil || loaded == nil {
		return
	}
	// Copy fields individually to avoid copying the Store's mutex by value
	// (vet: copylocks). The mutex is not part of the data we want to replace.
	r.store.Lock()
	defer r.store.Unlock()
	r.store.Users = loaded.Users
	r.store.Devices = loaded.Devices
	r.store.Partners = loaded.Partners
	r.store.Approvals = loaded.Approvals
	r.store.Modules = loaded.Modules
	r.store.EducationMedia = loaded.EducationMedia
	r.store.EducationProgress = loaded.EducationProgress
	r.store.SupportCases = loaded.SupportCases
	r.store.DataRequests = loaded.DataRequests
	r.store.Organizations = loaded.Organizations
	r.store.ModelReleases = loaded.ModelReleases
	r.store.RulesetReleases = loaded.RulesetReleases
	r.store.NetworkRulesets = loaded.NetworkRulesets
	r.store.AuditEvents = loaded.AuditEvents
	r.store.NotificationEvents = loaded.NotificationEvents
	r.store.JournalEntries = loaded.JournalEntries
	r.store.Missions = loaded.Missions
	r.store.Intentions = loaded.Intentions
	r.store.CheckIns = loaded.CheckIns
	r.store.AggregateEvents = loaded.AggregateEvents
	r.store.EmergencyKeyRequests = loaded.EmergencyKeyRequests
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
		return "Approved"
	}
	if status == "denied" {
		return "Denied"
	}
	if status == "expired" {
		return "Expired"
	}
	if status == "cancelled" {
		return "Cancelled"
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

func humanApprovalAction(action string, duration int) string {
	if action == "pause_protection" {
		return fmt.Sprintf("Pause protection for %d minutes", duration)
	}
	if action == "uninstall_detected" {
		return "Permission revoked detected"
	}
	return action
}
