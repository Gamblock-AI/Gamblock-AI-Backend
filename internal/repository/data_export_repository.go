package repository

import (
	"context"
	"fmt"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// BuildUserExportSnapshot reads only account-scoped, privacy-permitted records.
// Raw browsing data cannot be included because it is not accepted or stored.
func (r *Repository) BuildUserExportSnapshot(ctx context.Context, userID string) (map[string]any, error) {
	user, ok := r.UserByID(ctx, userID)
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	snapshot := r.store.Snapshot()
	devices := filterSlice(snapshot.Devices, func(item model.Device) bool { return item.UserID == userID })
	aggregates := filterSlice(snapshot.AggregateEvents, func(item model.AggregateEvent) bool { return item.UserID == userID })
	intentions := filterSlice(snapshot.Intentions, func(item model.Intention) bool { return item.UserID == userID })
	checkIns := filterSlice(snapshot.CheckIns, func(item model.CheckIn) bool { return item.UserID == userID })
	recovery := filterSlice(snapshot.RecoveryRecords, func(item model.RecoveryRecord) bool { return item.UserID == userID })
	practices := filterSlice(snapshot.RecoveryPracticeSessions, func(item model.RecoveryPracticeSession) bool { return item.UserID == userID })
	spaces := filterSlice(snapshot.RecoverySpaces, func(item model.RecoverySpace) bool { return item.UserID == userID })
	reflections := filterSlice(snapshot.JournalEntries, func(item model.JournalEntry) bool { return item.UserID == userID })
	missions := filterSlice(snapshot.Missions, func(item model.DailyMission) bool { return item.UserID == userID })
	education := filterSlice(snapshot.EducationProgress, func(item model.EducationProgress) bool { return item.UserID == userID })
	memberships := filterSlice(snapshot.AccountabilityMemberships, func(item model.AccountabilityMembership) bool { return item.StudentID == userID })
	groups := filterSlice(snapshot.AccountabilityGroups, func(item model.AccountabilityGroup) bool { return item.OwnerPartnerID == userID })
	approvals := filterSlice(snapshot.Approvals, func(item model.ApprovalRequest) bool { return item.UserID == userID || item.ResolvedBy == userID })
	supportCases := filterSlice(snapshot.SupportCases, func(item model.SupportCase) bool { return item.UserID == userID })
	caseIDs := make(map[string]bool, len(supportCases))
	for _, item := range supportCases {
		caseIDs[item.ID] = true
	}
	supportMessages := filterSlice(snapshot.SupportMessages, func(item model.SupportMessage) bool { return caseIDs[item.SupportCaseID] })
	return map[string]any{
		"account":                     user,
		"devices":                     devices,
		"aggregate_protection_events": aggregates,
		"intentions":                  intentions,
		"check_ins":                   checkIns,
		"recovery_records":            recovery,
		"recovery_practice_sessions":  practices,
		"recovery_spaces":             spaces,
		"reflections":                 reflections,
		"missions":                    missions,
		"education_progress":          education,
		"accountability_memberships":  memberships,
		"owned_accountability_groups": groups,
		"approval_requests":           approvals,
		"support_cases":               supportCases,
		"support_messages":            supportMessages,
	}, nil
}

func filterSlice[T any](items []T, keep func(T) bool) []T {
	result := make([]T, 0)
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}
