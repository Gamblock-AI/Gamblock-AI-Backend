package repository

import (
	"context"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryFinalApprovalEmergency_VisibilityAndTransitions(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	partnerID := "final-approval-partner"
	studentID := "final-approval-student"
	groupID := "final-approval-group"
	membershipID := "final-approval-membership"

	st := store.New()
	st.AccountabilityGroups = []model.AccountabilityGroup{{
		ID: groupID, OwnerPartnerID: partnerID, Name: "Final group", Status: "active",
	}}
	st.AccountabilityMemberships = []model.AccountabilityMembership{{
		ID: membershipID, GroupID: groupID, StudentID: studentID, Status: "active",
	}}
	st.Approvals = []model.ApprovalRequest{
		{
			ID: "final-approval-student-request", UserID: studentID, MembershipID: "final-other-membership",
			Action: "pause_protection", RequestedDurationMinutes: 15, Status: "pending",
			Reason: "student request", ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-4 * time.Minute), UpdatedAt: now,
		},
		{
			ID: "final-approval-partner-request", UserID: "final-other-student", MembershipID: membershipID,
			Action: "pause_protection", RequestedDurationMinutes: 30, Status: "pending",
			Reason: "partner-visible request", ExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-3 * time.Minute), UpdatedAt: now,
		},
		{
			ID: "final-approval-expired-request", UserID: studentID, MembershipID: "final-other-membership",
			Action: "pause_protection", Status: "pending", ExpiresAt: now.Add(-time.Minute), CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now,
		},
		{
			ID: "final-approval-hidden-request", UserID: "final-hidden-student", MembershipID: "final-other-membership",
			Action: "pause_protection", Status: "pending", ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		},
	}
	repo := New(nil, st)

	partnerVisible, err := repo.GetApprovalRequests(ctx, partnerID)
	require.NoError(t, err)
	require.Len(t, partnerVisible, 1)
	assert.Equal(t, "final-approval-partner-request", partnerVisible[0].ID)
	assert.Equal(t, "Pause protection for 30 minutes", partnerVisible[0].ActionLabel)

	studentVisible, err := repo.GetApprovalRequests(ctx, studentID)
	require.NoError(t, err)
	require.Len(t, studentVisible, 2)
	studentStatuses := map[string]string{}
	for _, item := range studentVisible {
		studentStatuses[item.ID] = item.Status
	}
	assert.Equal(t, "pending", studentStatuses["final-approval-student-request"])
	assert.Equal(t, "expired", studentStatuses["final-approval-expired-request"])

	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "final-approval-partner-request", partnerID, "cancelled", ""), "invalid approval status")
	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "final-approval-hidden-request", "final-wrong-partner", "approved", ""), "pending approval request not found")
	assert.NoError(t, repo.ResolveApprovalAsPartner(ctx, "final-approval-partner-request", partnerID, "approved", "Take a pause"))
	assert.Equal(t, "approved", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-partner-request").Status)
	assert.Equal(t, "Take a pause", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-partner-request").SupportiveResponse)
	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "final-approval-partner-request", partnerID, "denied", "late"), "pending approval request not found")

	assert.NoError(t, repo.CreateApprovalRequest(ctx, "final-approval-noop-create", studentID, "final-device", "final-link", "pause_protection", "reason", 15, now.Add(time.Hour)))
	st.Approvals = append(st.Approvals, model.ApprovalRequest{
		ID: "final-approval-update", UserID: studentID, MembershipID: membershipID,
		Action: "pause_protection", Status: "pending", ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	})
	assert.NoError(t, repo.UpdateApprovalRequest(ctx, "final-approval-update", "denied", partnerID))
	assert.Equal(t, "denied", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-update").Status)
	assert.EqualError(t, repo.UpdateApprovalRequest(ctx, "final-approval-update", "approved", partnerID), "pending approval request not found")
	assert.EqualError(t, repo.UpdateApprovalRequest(ctx, "final-approval-missing", "approved", partnerID), "pending approval request not found")

	st.Approvals = append(st.Approvals, model.ApprovalRequest{
		ID: "final-approval-cancel", UserID: studentID, MembershipID: membershipID,
		Action: "pause_protection", Status: "pending", ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	})
	assert.EqualError(t, repo.CancelApprovalRequest(ctx, "final-approval-cancel", "final-other-student"), "pending approval request not found")
	assert.NoError(t, repo.CancelApprovalRequest(ctx, "final-approval-cancel", studentID))
	assert.Equal(t, "cancelled", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-cancel").Status)
	assert.EqualError(t, repo.CancelApprovalRequest(ctx, "final-approval-cancel", studentID), "pending approval request not found")

	st.Approvals = append(st.Approvals,
		model.ApprovalRequest{ID: "final-approval-membership-pending", UserID: studentID, MembershipID: membershipID, Status: "pending", ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "final-approval-membership-approved", UserID: studentID, MembershipID: membershipID, Status: "approved", ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "final-approval-other-membership", UserID: studentID, MembershipID: "final-unrelated-membership", Status: "pending", ExpiresAt: now.Add(time.Hour)},
	)
	assert.NoError(t, repo.CancelPendingApprovalsForMembership(ctx, membershipID, partnerID))
	assert.Equal(t, "cancelled", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-membership-pending").Status)
	assert.Equal(t, "approved", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-membership-approved").Status)
	assert.Equal(t, "pending", repositoryFinalApprovalEmergencyFindApproval(st, "final-approval-other-membership").Status)
}

func TestRepositoryFinalApprovalEmergency_TokenStateAndGrantDurations(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	repo := New(nil, store.New())

	request, err := repo.CreateApprovalRequestWithToken(ctx, "final-token-request", "final-token-user", "final-token-device", "final-token-membership", "pause_protection", "token reason", 15, now.Add(time.Hour), "final-token-hash")
	require.NoError(t, err)
	assert.Equal(t, "pending", request.Status)
	assert.Equal(t, "Pause protection for 15 minutes", request.ActionLabel)

	quick, err := repo.GetApprovalByQuickToken(ctx, "final-token-hash")
	require.NoError(t, err)
	assert.Equal(t, "final-token-request", quick.ID)
	assert.Equal(t, "pending", quick.Status)
	repo.UpdateQuickTokenState("final-token-hash", store.ApprovalRequest{ID: request.ID, Status: "approved", Action: "pause_protection", GrantJTI: "final-token-grant"})
	quick, err = repo.GetApprovalByQuickToken(ctx, "final-token-hash")
	require.NoError(t, err)
	assert.Equal(t, "approved", quick.Status)
	assert.Equal(t, "final-token-grant", quick.GrantJTI)
	assert.EqualError(t, func() error {
		_, lookupErr := repo.GetApprovalByQuickToken(ctx, "final-token-missing")
		return lookupErr
	}(), "token not found")

	for _, duration := range []int{15, 30, 60, 120} {
		id := "final-grant-duration-" + time.Duration(duration).String()
		st := repo.store
		st.Lock()
		resolvedAt := now
		st.Approvals = append(st.Approvals, model.ApprovalRequest{
			ID: id, UserID: "final-grant-user", DeviceID: id, Action: "pause_protection",
			RequestedDurationMinutes: duration, Status: "approved", ResolvedAt: &resolvedAt,
			ExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
		})
		st.Unlock()

		grant, grantErr := repo.ApplyApprovedRequest(ctx, id, "final-grant-user", id, now, "jti-"+id)
		require.NoError(t, grantErr)
		assert.Equal(t, time.Duration(duration)*time.Minute, grant.GrantExpiresAt.Sub(now))
	}

	resolvedAt := now
	st := repo.store
	st.Lock()
	st.Approvals = append(st.Approvals,
		model.ApprovalRequest{ID: "final-grant-uninstall", UserID: "final-grant-user", DeviceID: "final-uninstall-device", Action: "uninstall_detected", Status: "approved", ResolvedAt: &resolvedAt, ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "final-grant-invalid-action", UserID: "final-grant-user", DeviceID: "final-invalid-device", Action: "unsupported", Status: "approved", ResolvedAt: &resolvedAt, ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "final-grant-invalid-duration", UserID: "final-grant-user", DeviceID: "final-invalid-duration-device", Action: "pause_protection", RequestedDurationMinutes: 10, Status: "approved", ResolvedAt: &resolvedAt, ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "final-grant-pending", UserID: "final-grant-user", DeviceID: "final-pending-device", Action: "pause_protection", RequestedDurationMinutes: 15, Status: "pending", ResolvedAt: &resolvedAt, ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "final-grant-expired-window", UserID: "final-grant-user", DeviceID: "final-expired-device", Action: "pause_protection", RequestedDurationMinutes: 15, Status: "approved", ResolvedAt: repositoryFinalApprovalEmergencyTimePtr(now.Add(-31 * time.Minute)), ExpiresAt: now.Add(time.Hour)},
	)
	st.Unlock()

	uninstall, err := repo.ApplyApprovedRequest(ctx, "final-grant-uninstall", "final-grant-user", "final-uninstall-device", now, "jti-uninstall")
	require.NoError(t, err)
	assert.Equal(t, 10*time.Minute, uninstall.GrantExpiresAt.Sub(now))
	assert.EqualError(t, func() error {
		_, applyErr := repo.ApplyApprovedRequest(ctx, "final-grant-invalid-action", "final-grant-user", "final-invalid-device", now, "jti-invalid")
		return applyErr
	}(), "approval action cannot be applied by a protection client")
	assert.EqualError(t, func() error {
		_, applyErr := repo.ApplyApprovedRequest(ctx, "final-grant-invalid-duration", "final-grant-user", "final-invalid-duration-device", now, "jti-invalid-duration")
		return applyErr
	}(), "pause duration must be 15, 30, 60, or 120 minutes")
	assert.EqualError(t, func() error {
		_, applyErr := repo.ApplyApprovedRequest(ctx, "final-grant-pending", "final-grant-user", "final-pending-device", now, "jti-pending")
		return applyErr
	}(), "approval request is not approved")
	assert.EqualError(t, func() error {
		_, applyErr := repo.ApplyApprovedRequest(ctx, "final-grant-expired-window", "final-grant-user", "final-expired-device", now, "jti-expired")
		return applyErr
	}(), "approval apply window expired")
	assert.EqualError(t, func() error {
		_, applyErr := repo.ApplyApprovedRequest(ctx, "final-grant-uninstall", "final-other-user", "final-uninstall-device", now, "jti-owner")
		return applyErr
	}(), "approval request not found")
}

func TestRepositoryFinalApprovalEmergency_EmergencyVisibilityExpiryAndPagination(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.June, 20, 12, 0, 0, 0, time.UTC)
	repo := New(nil, store.New())
	keyExpired := now.Add(-time.Minute)

	requests := []model.EmergencyKeyRequest{
		{ID: "final-emergency-pending", RequestedBy: "final-emergency-user", DeviceID: "final-device-a", Status: "pending", RequestExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now},
		{ID: "final-emergency-reviewed", RequestedBy: "final-emergency-user", DeviceID: "final-device-b", Status: "reviewed", RequestExpiresAt: now.Add(2 * time.Hour), CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now},
		{ID: "final-emergency-expired", RequestedBy: "final-other-user", DeviceID: "final-device-c", Status: "pending", RequestExpiresAt: now.Add(-time.Minute), CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now},
		{ID: "final-emergency-approved-expired", RequestedBy: "final-emergency-user", DeviceID: "final-device-a", Status: "approved", RequestExpiresAt: now.Add(time.Hour), KeyExpiresAt: &keyExpired, KeyHash: "final-expired-key", CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
		{ID: "final-emergency-used", RequestedBy: "final-history-user", DeviceID: "final-history-device", Status: "used", RequestExpiresAt: now.Add(time.Hour), CreatedAt: now.Add(-30 * time.Minute), UpdatedAt: now},
	}
	for _, request := range requests {
		_, err := repo.CreateEmergencyKeyRequest(ctx, request)
		require.NoError(t, err)
	}

	current, err := repo.GetCurrentEmergencyKeyRequest(ctx, "final-emergency-user", "final-device-a", now)
	require.NoError(t, err)
	assert.Equal(t, "final-emergency-approved-expired", current.ID)
	assert.Equal(t, "expired", current.Status)

	pending, err := repo.GetPendingEmergencyKeyRequests(ctx, now)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.Equal(t, "final-emergency-pending", pending[0].ID)
	assert.Equal(t, "final-emergency-reviewed", pending[1].ID)

	filtered, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Query: "FINAL-DEVICE-B", Limit: 1})
	require.NoError(t, err)
	require.Len(t, filtered.Items, 1)
	assert.Equal(t, "final-emergency-reviewed", filtered.Items[0].ID)
	assert.Equal(t, 1, filtered.TotalCount)

	statusFiltered, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Status: "reviewed", Limit: 10})
	require.NoError(t, err)
	require.Len(t, statusFiltered.Items, 1)
	assert.Equal(t, "final-emergency-reviewed", statusFiltered.Items[0].ID)

	history, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Bucket: "history", Limit: 10})
	require.NoError(t, err)
	require.Len(t, history.Items, 3)
	assert.Equal(t, 3, history.TotalCount)

	offsetPage, err := repo.GetPendingEmergencyKeyRequestsPaginated(ctx, now, model.PaginationQuery{Page: 99, Limit: 2})
	require.NoError(t, err)
	assert.Empty(t, offsetPage.Items)

	_, err = repo.GetCurrentEmergencyKeyRequest(ctx, "final-missing-user", "final-missing-device", now)
	assert.EqualError(t, err, "emergency request not found")
}

func TestRepositoryFinalApprovalEmergency_EmergencyTransitionsAndErrors(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.June, 21, 9, 0, 0, 0, time.UTC)
	repo := New(nil, store.New())

	_, err := repo.ReviewEmergencyKeyRequest(ctx, "final-emergency-missing", "final-admin", now)
	assert.EqualError(t, err, "request not found")
	_, err = repo.ApproveEmergencyKeyRequest(ctx, "final-emergency-missing", "final-admin", "missing-key", now, now.Add(time.Hour))
	assert.EqualError(t, err, "request not found")

	request := model.EmergencyKeyRequest{
		ID: "final-emergency-transition", RequestedBy: "final-transition-user", DeviceID: "final-transition-device",
		Status: "pending", RequestExpiresAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	_, err = repo.CreateEmergencyKeyRequest(ctx, request)
	require.NoError(t, err)
	reviewed, err := repo.ReviewEmergencyKeyRequest(ctx, request.ID, "final-reviewer", now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "reviewed", reviewed.Status)
	assert.Equal(t, "final-reviewer", reviewed.ReviewedBy)
	_, err = repo.ReviewEmergencyKeyRequest(ctx, request.ID, "final-reviewer", now.Add(2*time.Minute))
	assert.EqualError(t, err, "request is not pending or has expired")

	keyExpiry := now.Add(2 * time.Hour)
	approved, err := repo.ApproveEmergencyKeyRequest(ctx, request.ID, "final-approver", "final-emergency-key", now.Add(3*time.Minute), keyExpiry)
	require.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)
	assert.Equal(t, "final-reviewer", approved.ReviewedBy)
	assert.Equal(t, "final-approver", approved.ApprovedBy)
	_, err = repo.ApproveEmergencyKeyRequest(ctx, request.ID, "final-approver", "second-key", now.Add(4*time.Minute), keyExpiry)
	assert.EqualError(t, err, "request is not pending or reviewed or has expired")

	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "missing-key", request.DeviceID, now)
	assert.EqualError(t, err, "emergency key not found")
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "final-emergency-key", "final-other-device", now)
	assert.EqualError(t, err, "emergency key is not valid for this device")
	usable, err := repo.GetUsableEmergencyKeyRequest(ctx, "final-emergency-key", request.DeviceID, now)
	require.NoError(t, err)
	assert.Equal(t, "approved", usable.Status)

	assert.EqualError(t, func() error {
		_, useErr := repo.UseEmergencyKey(ctx, "final-emergency-key", request.DeviceID, now, now.Add(9*time.Minute), "grant-too-short")
		return useErr
	}(), "invalid emergency grant metadata")
	assert.EqualError(t, func() error {
		_, useErr := repo.UseEmergencyKey(ctx, "missing-key", request.DeviceID, now, now.Add(10*time.Minute), "grant-missing")
		return useErr
	}(), "emergency key not found")
	assert.EqualError(t, func() error {
		_, useErr := repo.UseEmergencyKey(ctx, "final-emergency-key", "final-other-device", now, now.Add(10*time.Minute), "grant-wrong-device")
		return useErr
	}(), "emergency key is not valid for this device")

	used, err := repo.UseEmergencyKey(ctx, "final-emergency-key", request.DeviceID, now, now.Add(10*time.Minute), "final-emergency-grant")
	require.NoError(t, err)
	assert.Equal(t, "used", used.Status)
	assert.Equal(t, "final-emergency-grant", used.GrantJTI)
	retry, err := repo.GetUsableEmergencyKeyRequest(ctx, "final-emergency-key", request.DeviceID, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "final-emergency-grant", retry.GrantJTI)
	retried, err := repo.UseEmergencyKey(ctx, "final-emergency-key", request.DeviceID, now.Add(time.Minute), now.Add(11*time.Minute), "different-grant")
	require.NoError(t, err)
	assert.Equal(t, "final-emergency-grant", retried.GrantJTI)
	_, err = repo.GetUsableEmergencyKeyRequest(ctx, "final-emergency-key", request.DeviceID, now.Add(11*time.Minute))
	assert.EqualError(t, err, "emergency key is invalid, used, or expired")

	expiredRequest := model.EmergencyKeyRequest{
		ID: "final-emergency-expired-transition", RequestedBy: "final-transition-user", DeviceID: "final-expired-device",
		Status: "pending", RequestExpiresAt: now.Add(-time.Minute), CreatedAt: now, UpdatedAt: now,
	}
	_, err = repo.CreateEmergencyKeyRequest(ctx, expiredRequest)
	require.NoError(t, err)
	_, err = repo.ReviewEmergencyKeyRequest(ctx, expiredRequest.ID, "final-reviewer", now)
	assert.EqualError(t, err, "request is not pending or has expired")
	_, err = repo.ApproveEmergencyKeyRequest(ctx, expiredRequest.ID, "final-approver", "expired-key", now, now.Add(time.Hour))
	assert.EqualError(t, err, "request is not pending or reviewed or has expired")
}

func repositoryFinalApprovalEmergencyFindApproval(st *store.Store, id string) model.ApprovalRequest {
	for _, item := range st.Snapshot().Approvals {
		if item.ID == id {
			return item
		}
	}
	return model.ApprovalRequest{}
}

func repositoryFinalApprovalEmergencyTimePtr(value time.Time) *time.Time {
	return &value
}
