package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

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
	switch status {
	case "pending":
		return "Pending partner approval"
	case "approved":
		return "Approved"
	case "denied":
		return "Denied"
	case "expired":
		return "Expired"
	case "cancelled":
		return "Cancelled"
	default:
		return status
	}
}

func humanApprovalAction(action string, duration int) string {
	switch action {
	case "pause_protection":
		return fmt.Sprintf("Pause protection for %d minutes", duration)
	case "uninstall_detected":
		return "Permission revoked detected"
	default:
		return action
	}
}

func humanDataRequestTitle(kind string) string {
	switch kind {
	case "export":
		return "Export account data"
	case "delete":
		return "Delete archived support notes"
	default:
		return "Data request"
	}
}
