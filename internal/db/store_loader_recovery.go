package db

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoverypracticesession"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func loadRecoveryStore(ctx context.Context, client *ent.Client, out *store.Store) {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	if reflections, err := client.Reflection.Query().All(ctx); err == nil {
		for _, item := range reflections {
			out.JournalEntries = append(out.JournalEntries, store.JournalEntry{
				ID:        item.ID,
				UserID:    item.UserID,
				Text:      item.ContentEncrypted, // Store only the encrypted payload.
				Mood:      value(item.PromptKey),
				Status:    item.Status.String(),
				IsFocus:   item.IsFocus,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			})
		}
	}

	if missions, err := client.DailyMission.Query().All(ctx); err == nil {
		byDay := make(map[string]*store.DailyMission)
		for _, item := range missions {
			date := item.CreatedAt.In(jakarta).Format("2006-01-02")
			if item.MissionDate != nil {
				date = *item.MissionDate
			}
			key := item.UserID + ":" + date
			day, ok := byDay[key]
			if !ok {
				day = &store.DailyMission{
					ID:        "day_" + date,
					UserID:    item.UserID,
					Date:      date,
					CreatedAt: item.CreatedAt,
					UpdatedAt: item.UpdatedAt,
				}
				byDay[key] = day
			}
			setMissionCompleted(day, missionKeyNumber(item.MissionKey), item.Status.String() == "completed")
			if item.UpdatedAt.After(day.UpdatedAt) {
				day.UpdatedAt = item.UpdatedAt
			}
		}
		for _, day := range byDay {
			out.Missions = append(out.Missions, *day)
		}
	}

	if intentions, err := client.Intention.Query().All(ctx); err == nil {
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

	if checkIns, err := client.CheckIn.Query().All(ctx); err == nil {
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

	if records, err := client.RecoveryRecord.Query().All(ctx); err == nil {
		for _, item := range records {
			out.RecoveryRecords = append(out.RecoveryRecords, store.RecoveryRecord{
				ID: item.ID, UserID: item.UserID, Kind: item.Kind.String(),
				RecordDate: item.RecordDate, Metadata: item.MetadataJSON,
				Content: value(item.ContentEncrypted), Status: item.Status.String(),
				CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
			})
		}
	}

	if sessions, err := client.RecoveryPracticeSession.Query().All(ctx); err == nil {
		for _, item := range sessions {
			out.RecoveryPracticeSessions = append(out.RecoveryPracticeSessions, store.RecoveryPracticeSession{
				ID: item.ID, UserID: item.UserID, PracticeKind: item.PracticeKind.String(),
				DurationSeconds: item.DurationSeconds, Feedback: recoveryFeedback(item.Feedback),
				CompletedAt: item.CompletedAt, CreatedAt: item.CreatedAt,
			})
		}
	}

	if spaces, err := client.RecoverySpace.Query().All(ctx); err == nil {
		for _, item := range spaces {
			out.RecoverySpaces = append(out.RecoverySpaces, store.RecoverySpace{
				ID: item.ID, UserID: item.UserID, Theme: item.Theme.String(),
				UnlockedItems: item.UnlockedItemsJSON, PlacedItems: item.PlacedItemsJSON,
				UnlockRuleVersion: item.UnlockRuleVersion, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
			})
		}
	}

	if aggregates, err := client.AggregateEvent.Query().All(ctx); err == nil {
		for _, item := range aggregates {
			out.AggregateEvents = append(out.AggregateEvents, store.AggregateEvent{
				ID:             item.ID,
				UserID:         item.UserID,
				DeviceID:       value(item.DeviceID),
				IdempotencyKey: item.IdempotencyKey,
				EventType:      item.EventType.String(),
				EventDate:      item.EventDate,
				Count:          item.Count,
				CreatedAt:      item.CreatedAt,
			})
		}
	}

	if requests, err := client.EmergencyKeyRequest.Query().All(ctx); err == nil {
		for _, item := range requests {
			out.EmergencyKeyRequests = append(out.EmergencyKeyRequests, store.EmergencyKeyRequest{
				ID:               item.ID,
				RequestedBy:      item.RequestedBy,
				DeviceID:         value(item.DeviceID),
				ReviewedBy:       value(item.ReviewedBy),
				ApprovedBy:       value(item.ApprovedBy),
				Status:           item.Status.String(),
				RequestExpiresAt: item.RequestExpiresAt,
				KeyExpiresAt:     item.KeyExpiresAt,
				ReviewedAt:       item.ReviewedAt,
				ApprovedAt:       item.ApprovedAt,
				UsedAt:           item.UsedAt,
				CreatedAt:        item.CreatedAt,
				UpdatedAt:        item.UpdatedAt,
				KeyHash:          value(item.KeyHash),
			})
		}
	}
}

func recoveryFeedback(value *recoverypracticesession.Feedback) string {
	if value == nil {
		return ""
	}
	return value.String()
}
