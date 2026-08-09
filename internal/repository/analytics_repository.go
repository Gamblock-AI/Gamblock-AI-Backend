package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

// analyticsEventTypes are the privacy-preserving protection events that feed
// role dashboards. Values are aggregate counts only.
var analyticsEventTypes = map[string]bool{
	"block_count_sync": true, "intervention_shown": true,
	"tamper_detected": true, "permission_revoked": true,
}

// partnerAnalyticsScope resolves the live members of a partner's groups,
// optionally narrowed to a single group owned by that partner, and returns the
// member IDs that consent to sharing protection activity.
func (r *Repository) partnerAnalyticsScope(ctx context.Context, partnerID, groupID string) (memberIDs, sharedIDs []string, memberCount int, err error) {
	groups, err := r.ListAccountabilityGroups(ctx, partnerID)
	if err != nil {
		return nil, nil, 0, err
	}
	all := map[string]struct{}{}
	shared := map[string]struct{}{}
	found := groupID == ""
	for _, group := range groups {
		if groupID != "" {
			if group.ID == groupID {
				found = true
			} else {
				continue
			}
		}
		members, listErr := r.ListMembershipsForGroup(ctx, group.ID)
		if listErr != nil {
			return nil, nil, 0, listErr
		}
		for _, member := range members {
			if !liveMembershipStatuses[member.Status] {
				continue
			}
			all[member.StudentID] = struct{}{}
			if member.Sharing.ProtectionActivity {
				shared[member.StudentID] = struct{}{}
			}
		}
	}
	if !found {
		return nil, nil, 0, fmt.Errorf("group is not owned by the partner")
	}
	memberCount = len(all)
	for id := range all {
		memberIDs = append(memberIDs, id)
	}
	for id := range shared {
		sharedIDs = append(sharedIDs, id)
	}
	sort.Strings(memberIDs)
	sort.Strings(sharedIDs)
	return memberIDs, sharedIDs, memberCount, nil
}

// PartnerAnalytics returns group-scoped analytics for a partner. Only members
// who consented to sharing protection activity contribute to the aggregated
// counts; raw browsing data is never represented.
func (r *Repository) PartnerAnalytics(ctx context.Context, partnerID, groupID string, days int, now time.Time) (model.AnalyticsSummary, error) {
	_, sharedIDs, memberCount, err := r.partnerAnalyticsScope(ctx, partnerID, groupID)
	if err != nil {
		return model.AnalyticsSummary{}, err
	}
	if len(sharedIDs) == 0 {
		return emptyAnalytics(days, memberCount, 0), nil
	}
	events, err := r.aggregateEventsForUsers(ctx, sharedIDs, days, now)
	if err != nil {
		return model.AnalyticsSummary{}, err
	}
	summary := buildAnalyticsSummary(events, days, now, r.db != nil)
	summary.MemberCount = memberCount
	summary.SharedMemberCount = len(sharedIDs)
	return summary, nil
}

// PlatformAnalytics returns platform-wide analytics across all devices. It is
// served only to verified admin roles.
func (r *Repository) PlatformAnalytics(ctx context.Context, days int, now time.Time) (model.AnalyticsSummary, error) {
	events, err := r.allAggregateEvents(ctx, days, now)
	if err != nil {
		return model.AnalyticsSummary{}, err
	}
	summary := buildAnalyticsSummary(events, days, now, r.db != nil)
	summary.ProtectedUsers = r.countProtectedUsers(ctx)
	return summary, nil
}

func (r *Repository) aggregateEventsForUsers(ctx context.Context, userIDs []string, days int, now time.Time) ([]model.AggregateEvent, error) {
	if len(userIDs) == 0 {
		return []model.AggregateEvent{}, nil
	}
	start := startOfDay(now.UTC()).AddDate(0, 0, -(days - 1))
	if r.db == nil {
		allowed := make(map[string]struct{}, len(userIDs))
		for _, id := range userIDs {
			allowed[id] = struct{}{}
		}
		events := make([]model.AggregateEvent, 0)
		for _, event := range r.store.Snapshot().AggregateEvents {
			if _, ok := allowed[event.UserID]; ok && analyticsEventTypes[event.EventType] && !event.EventDate.Before(start) {
				events = append(events, event)
			}
		}
		return events, nil
	}
	rows, err := r.db.AggregateEvent.Query().Where(
		aggregateevent.UserIDIn(userIDs...),
		aggregateevent.EventDateGTE(start),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	events := make([]model.AggregateEvent, 0, len(rows))
	for _, item := range rows {
		if !analyticsEventTypes[item.EventType.String()] {
			continue
		}
		events = append(events, model.AggregateEvent{
			EventType: item.EventType.String(), EventDate: item.EventDate,
			Count: item.Count, MetadataJSON: item.MetadataJSON,
		})
	}
	return events, nil
}

func (r *Repository) allAggregateEvents(ctx context.Context, days int, now time.Time) ([]model.AggregateEvent, error) {
	start := startOfDay(now.UTC()).AddDate(0, 0, -(days - 1))
	if r.db == nil {
		events := make([]model.AggregateEvent, 0)
		for _, event := range r.store.Snapshot().AggregateEvents {
			if analyticsEventTypes[event.EventType] && !event.EventDate.Before(start) {
				events = append(events, event)
			}
		}
		return events, nil
	}
	rows, err := r.db.AggregateEvent.Query().Where(
		aggregateevent.EventDateGTE(start),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	events := make([]model.AggregateEvent, 0, len(rows))
	for _, item := range rows {
		if !analyticsEventTypes[item.EventType.String()] {
			continue
		}
		events = append(events, model.AggregateEvent{
			EventType: item.EventType.String(), EventDate: item.EventDate,
			Count: item.Count, MetadataJSON: item.MetadataJSON,
		})
	}
	return events, nil
}

func (r *Repository) countProtectedUsers(ctx context.Context) int {
	if r.db == nil {
		protected := map[string]struct{}{}
		for _, item := range r.store.Snapshot().Devices {
			if item.ProtectionStatus == "active" {
				protected[item.UserID] = struct{}{}
			}
		}
		return len(protected)
	}
	rows, err := r.db.Device.Query().Where(device.ProtectionStatusEQ(device.ProtectionStatusActive)).All(ctx)
	if err != nil {
		return 0
	}
	protected := map[string]struct{}{}
	for _, item := range rows {
		protected[item.UserID] = struct{}{}
	}
	return len(protected)
}

func emptyAnalytics(days, memberCount, sharedMemberCount int) model.AnalyticsSummary {
	summary := buildAnalyticsSummary(nil, days, time.Now().UTC(), false)
	summary.MemberCount = memberCount
	summary.SharedMemberCount = sharedMemberCount
	return summary
}

func buildAnalyticsSummary(events []model.AggregateEvent, days int, now time.Time, synced bool) model.AnalyticsSummary {
	start := startOfDay(now.UTC()).AddDate(0, 0, -(days - 1))
	daily := make([]model.AnalyticsDay, days)
	byDate := make(map[string]int, days)
	for index := range daily {
		date := start.AddDate(0, 0, index).Format("2006-01-02")
		daily[index].Date = date
		byDate[date] = index
	}
	hourly := make([]model.AnalyticsHour, 24)
	for hour := 0; hour < 24; hour++ {
		hourly[hour].Hour = hour
	}
	totals := model.AnalyticsTotals{}
	dataState := "local_only"
	if synced {
		dataState = "synced"
	}
	for _, event := range events {
		index, ok := byDate[event.EventDate.UTC().Format("2006-01-02")]
		if !ok {
			continue
		}
		switch event.EventType {
		case "block_count_sync":
			daily[index].Blocked += event.Count
			totals.Blocked += event.Count
		case "intervention_shown":
			daily[index].Interventions += event.Count
			totals.Interventions += event.Count
		case "tamper_detected":
			daily[index].TamperEvents += event.Count
			totals.TamperEvents += event.Count
		case "permission_revoked":
			daily[index].PermissionRevoked += event.Count
			totals.PermissionRevoked += event.Count
		}
		if event.EventType == "block_count_sync" {
			addHourlyHistogram(hourly, event.MetadataJSON)
		}
	}
	if totals == (model.AnalyticsTotals{}) && hourlyTotal(hourly) == 0 {
		dataState = "empty"
	}
	return model.AnalyticsSummary{
		PeriodDays: days, Totals: totals, Daily: daily, Hourly: hourly,
		DataState: dataState,
	}
}

func addHourlyHistogram(hourly []model.AnalyticsHour, metadataJSON map[string]any) {
	raw, ok := metadataJSON["hourly"]
	if !ok {
		return
	}
	values, ok := raw.([]any)
	if !ok || len(values) != 24 {
		return
	}
	for hour, value := range values {
		count, ok := value.(float64)
		if !ok || count < 0 {
			continue
		}
		hourly[hour].Count += int(count)
	}
}

func hourlyTotal(hourly []model.AnalyticsHour) int {
	total := 0
	for _, slot := range hourly {
		total += slot.Count
	}
	return total
}
