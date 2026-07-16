package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
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
	users, err := client.User.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return store.New(), nil
	}
	out := &store.Store{}
	for _, item := range users {
		out.Users = append(out.Users, store.User{
			ID:            item.ID,
			Email:         item.Email,
			DisplayName:   item.DisplayName,
			Role:          item.Role.String(),
			PasswordHash:  value(item.PasswordHash),
			GoogleSubject: value(item.GoogleSubject),
			DisabledAt:    item.DisabledAt,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
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
		contact := item.PartnerEmail
		if phone := value(item.PartnerPhone); phone != "" {
			contact += " | " + phone
		}
		out.Partners = append(out.Partners, store.Partner{
			ID:              item.ID,
			UserID:          item.UserID,
			PartnerUserID:   value(item.PartnerUserID),
			InviteTokenHash: value(item.InviteTokenHash),
			Name:            item.PartnerEmail,
			Contact:         contact,
			Status:          item.Status.String(),
			PartnerEmail:    item.PartnerEmail,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
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
			UserID:                   item.UserID,
			DeviceID:                 value(item.DeviceID),
			PartnerLinkID:            item.PartnerLinkID,
			QuickTokenHash:           value(item.QuickTokenHash),
			Action:                   humanApprovalAction(item.Action.String(), duration),
			ExpiresIn:                humanExpiry(item.ExpiresAt),
			Status:                   humanApprovalStatus(item.Status.String()),
			Reason:                   value(item.Reason),
			RequestedDurationMinutes: duration,
			CreatedAt:                item.CreatedAt,
			UpdatedAt:                item.UpdatedAt,
			ExpiresAt:                item.ExpiresAt,
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
			Progress:         0,
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
			ArtifactPath:    item.ArtifactPath,
			ContractVersion: item.ContractVersion,
			Threshold:       item.Threshold,
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
			ArtifactPath:    item.ArtifactPath,
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
			ArtifactPath:    item.ArtifactPath,
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
		members, countErr := client.OrganizationMember.Query().Where(organizationmember.OrganizationIDEQ(item.ID)).Count(ctx)
		if countErr != nil {
			return nil, countErr
		}
		out.Organizations = append(out.Organizations, store.Organization{
			ID:        item.ID,
			Name:      item.Name,
			Slug:      item.Slug,
			Status:    item.Status.String(),
			Members:   members,
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
			UserID:    item.UserID,
			Title:     item.Summary,
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			Priority:  item.Priority.String(),
			Owner:     "",
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
			UserID:    item.UserID,
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
	reflections, err := client.Reflection.Query().All(ctx)
	if err == nil {
		for _, item := range reflections {
			out.JournalEntries = append(out.JournalEntries, store.JournalEntry{
				ID:        item.ID,
				UserID:    item.UserID,
				Text:      item.ContentEncrypted, // Text stores the encrypted content payload in ent model
				Mood:      value(item.PromptKey),
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			})
		}
	}
	missions, err := client.DailyMission.Query().All(ctx)
	if err == nil {
		byDay := make(map[string]*store.DailyMission)
		for _, item := range missions {
			date := item.CreatedAt.UTC().Format("2006-01-02")
			key := item.UserID + ":" + date
			day, ok := byDay[key]
			if !ok {
				day = &store.DailyMission{ID: "day_" + date, UserID: item.UserID, Date: date, CreatedAt: item.CreatedAt, UpdatedAt: item.CreatedAt}
				byDay[key] = day
			}
			setMissionCompleted(day, missionKeyNumber(item.MissionKey), item.Status.String() == "completed")
			if item.CreatedAt.After(day.UpdatedAt) {
				day.UpdatedAt = item.CreatedAt
			}
		}
		for _, day := range byDay {
			out.Missions = append(out.Missions, *day)
		}
	}
	intentions, err := client.Intention.Query().All(ctx)
	if err == nil {
		for _, item := range intentions {
			out.Intentions = append(out.Intentions, store.Intention{
				ID:        item.ID,
				UserID:    item.UserID,
				Text:      item.IntentionText,
				Status:    item.Status.String(),
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			})
		}
	}
	checkIns, err := client.CheckIn.Query().All(ctx)
	if err == nil {
		for _, item := range checkIns {
			out.CheckIns = append(out.CheckIns, store.CheckIn{
				ID:        item.ID,
				UserID:    item.UserID,
				Mood:      item.MoodScore,
				Urge:      item.UrgeScore,
				Context:   value(item.ContextText),
				CreatedAt: item.CreatedAt,
			})
		}
	}
	aggregates, err := client.AggregateEvent.Query().All(ctx)
	if err == nil {
		for _, item := range aggregates {
			out.AggregateEvents = append(out.AggregateEvents, store.AggregateEvent{
				ID: item.ID, UserID: item.UserID, DeviceID: value(item.DeviceID),
				IdempotencyKey: item.IdempotencyKey, EventType: item.EventType.String(),
				EventDate: item.EventDate, Count: item.Count, CreatedAt: item.CreatedAt,
			})
		}
	}
	emergencyRequests, err := client.EmergencyKeyRequest.Query().All(ctx)
	if err == nil {
		for _, item := range emergencyRequests {
			out.EmergencyKeyRequests = append(out.EmergencyKeyRequests, store.EmergencyKeyRequest{
				ID: item.ID, RequestedBy: item.RequestedBy, ApprovedBy: value(item.ApprovedBy),
				Status: item.Status.String(), RequestExpiresAt: item.RequestExpiresAt,
				KeyExpiresAt: item.KeyExpiresAt, ApprovedAt: item.ApprovedAt,
				CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, KeyHash: value(item.KeyHash),
			})
		}
	}
	return out, nil
}

func missionKeyNumber(key string) int {
	number, _ := strconv.Atoi(strings.TrimPrefix(key, "mission_"))
	return number
}

func setMissionCompleted(day *store.DailyMission, number int, completed bool) {
	switch number {
	case 1:
		day.Mission1 = completed
	case 2:
		day.Mission2 = completed
	case 3:
		day.Mission3 = completed
	case 4:
		day.Mission4 = completed
	case 5:
		day.Mission5 = completed
	}
}

func value(v *string) string {
	if v == nil {
		return ""
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
