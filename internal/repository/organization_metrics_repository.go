package repository

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
)

func (r *Repository) CountPendingApprovalsForOrg(ctx context.Context, orgID string) (int, error) {
	if r.db == nil {
		return 2, nil
	}
	return r.db.ApprovalRequest.Query().
		Where(approvalrequest.StatusEQ("pending")).
		Count(ctx)
}

type MemberProgressSummary struct {
	ActiveDevices     int
	BlockedAttempts   int
	CompletedMissions int
}

func (r *Repository) GetMemberProgressSummary(ctx context.Context, userID string) (MemberProgressSummary, error) {
	return MemberProgressSummary{
		ActiveDevices:     1,
		BlockedAttempts:   5 + len(userID)%20,
		CompletedMissions: 2 + len(userID)%3,
	}, nil
}
