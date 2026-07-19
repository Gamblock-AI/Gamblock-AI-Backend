package repository

import (
	"fmt"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func activePartnerLinkIDs(links []model.Partner, partnerUserID string) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, link := range links {
		if link.PartnerUserID == partnerUserID && link.Status == "active" {
			ids[link.ID] = struct{}{}
		}
	}
	return ids
}

func approvalActionLabel(action string, duration int) string {
	switch action {
	case "pause_protection":
		return fmt.Sprintf("Pause protection for %d minutes", duration)
	case "uninstall_detected":
		return "Allow protected app removal"
	default:
		return action
	}
}

func approvalStatusLabel(status string) string {
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
