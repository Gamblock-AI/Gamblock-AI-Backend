package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func Open(databaseURL string) (*ent.Client, func() error, error) {
	if databaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL is empty")
	}
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, nil, err
	}
	driver := entsql.OpenDB(dialect.Postgres, sqlDB)
	return ent.NewClient(ent.Driver(driver)), sqlDB.Close, nil
}

func Migrate(ctx context.Context, client *ent.Client) error {
	return client.Schema.Create(ctx)
}

func Seed(ctx context.Context, client *ent.Client) error {
	return seed.Seed(ctx, client)
}

func LoadStore(ctx context.Context, client *ent.Client) (*store.Store, error) {
	seedData := store.NewSeeded()
	users, err := client.User.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return seedData, nil
	}
	out := &store.Store{}
	for _, item := range users {
		out.Users = append(out.Users, store.User{
			ID:          item.ID,
			Email:       item.Email,
			DisplayName: item.DisplayName,
			Role:        item.Role.String(),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	devices, err := client.Device.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range devices {
		lastSeen := time.Time{}
		if item.LastSeenAt != nil {
			lastSeen = *item.LastSeenAt
		}
		out.Devices = append(out.Devices, store.Device{
			ID:               item.ID,
			UserID:           item.UserID,
			Platform:         item.Platform.String(),
			Label:            item.Label,
			AppVersion:       item.AppVersion,
			OSVersion:        item.OsVersion,
			ModelVersion:     value(item.ModelVersion),
			RulesetVersion:   value(item.RulesetVersion),
			ProtectionStatus: item.ProtectionStatus.String(),
			LastSeenAt:       lastSeen,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	partners, err := client.PartnerLink.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range partners {
		out.Partners = append(out.Partners, store.Partner{
			ID:           item.ID,
			Name:         "Suci Maisaa",
			Contact:      item.PartnerEmail + " | " + value(item.PartnerPhone),
			Status:       "Active partner",
			PartnerEmail: item.PartnerEmail,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}
	approvals, err := client.ApprovalRequest.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range approvals {
		duration := 0
		if item.RequestedDurationMinutes != nil {
			duration = *item.RequestedDurationMinutes
		}
		out.Approvals = append(out.Approvals, store.ApprovalRequest{
			ID:                       item.ID,
			Action:                   humanApprovalAction(item.Action.String(), duration),
			ExpiresIn:                humanExpiry(item.ExpiresAt),
			Status:                   humanApprovalStatus(item.Status.String()),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: duration,
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
		})
	}
	modules, err := client.PsychoeducationModule.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range modules {
		out.Modules = append(out.Modules, store.EducationModule{
			ID:               item.ID,
			Slug:             item.Slug,
			Title:            item.Title,
			Summary:          item.Summary,
			BodyMarkdown:     item.BodyMarkdown,
			EstimatedMinutes: item.EstimatedMinutes,
			Progress:         moduleProgress(item.Slug),
			Status:           item.Status.String(),
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	modelReleases, err := client.ModelRelease.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range modelReleases {
		out.ModelReleases = append(out.ModelReleases, store.Release{
			ID:              item.ID,
			Version:         item.Version,
			Platform:        item.Platform.String(),
			SHA256:          item.Sha256,
			Status:          item.Status.String(),
			DownloadURL:     "/v1/releases/model/" + item.Version + "/download",
			Metrics:         item.MetricsJSON,
			PublishedAtText: humanPublished(item.PublishedAt),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	rulesets, err := client.RulesetRelease.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range rulesets {
		out.RulesetReleases = append(out.RulesetReleases, store.Release{
			ID:              item.ID,
			Version:         item.Version,
			Platform:        "all",
			SHA256:          item.Sha256,
			Status:          item.Status.String(),
			DownloadURL:     "/v1/releases/ruleset/" + item.Version + "/download",
			Metrics:         item.RulesJSON,
			PublishedAtText: humanPublished(item.PublishedAt),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	networkRules, err := client.NetworkRulesetRelease.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range networkRules {
		out.NetworkRulesets = append(out.NetworkRulesets, store.Release{
			ID:              item.ID,
			Version:         item.Version,
			Platform:        "all",
			SHA256:          item.Sha256,
			Status:          item.Status.String(),
			DownloadURL:     "/v1/releases/network-rulesets/" + item.Version + "/download",
			Metrics:         item.RulesJSON,
			PublishedAtText: humanPublished(item.PublishedAt),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       time.Time{},
		})
	}
	orgs, err := client.Organization.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range orgs {
		out.Organizations = append(out.Organizations, store.Organization{
			ID:        item.ID,
			Name:      item.Name,
			Slug:      item.Slug,
			Status:    item.Status.String(),
			Members:   128,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	supportCases, err := client.SupportCase.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range supportCases {
		out.SupportCases = append(out.SupportCases, store.SupportCase{
			ID:        item.ID,
			Title:     item.Summary,
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			Priority:  item.Priority.String(),
			Owner:     "Alfian",
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	dataRequests, err := client.DataRequest.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range dataRequests {
		out.DataRequests = append(out.DataRequests, store.DataRequest{
			ID:        item.ID,
			Title:     humanDataRequestTitle(item.Type.String()),
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			CreatedAt: item.RequestedAt,
			UpdatedAt: time.Time{},
		})
	}
	audits, err := client.AuditLog.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range audits {
		out.AuditEvents = append(out.AuditEvents, store.AuditEvent{
			ID:        item.ID,
			Actor:     item.ActorEmail,
			Action:    item.Action,
			Target:    item.TargetID,
			CreatedAt: item.CreatedAt,
			UpdatedAt: time.Time{},
		})
	}
	notifications, err := client.NotificationDelivery.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range notifications {
		out.NotificationEvents = append(out.NotificationEvents, store.NotificationItem{
			ID:        item.ID,
			Channel:   item.Channel.String(),
			Recipient: item.Recipient,
			Status:    item.Status.String(),
			Reason:    value(item.ApprovalRequestID),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	ensureDefaults(out, seedData)
	return out, nil
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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

func humanApprovalAction(action string, duration int) string {
	if action == "pause_protection" {
		return fmt.Sprintf("Pause protection for %d minutes", duration)
	}
	if action == "uninstall_detected" {
		return "Permission revoked detected"
	}
	return action
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

func ensureDefaults(out *store.Store, seed *store.Store) {
	if len(out.ModelReleases) == 0 {
		out.ModelReleases = seed.ModelReleases
	}
	if len(out.RulesetReleases) == 0 {
		out.RulesetReleases = seed.RulesetReleases
	}
	if len(out.NetworkRulesets) == 0 {
		out.NetworkRulesets = seed.NetworkRulesets
	}
	if len(out.Modules) == 0 {
		out.Modules = seed.Modules
	}
}
