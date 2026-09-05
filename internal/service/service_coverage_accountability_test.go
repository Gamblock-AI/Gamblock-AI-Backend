package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAccountabilityGroupJoinLeaveContactAndPagination(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	cfg := testCfg()
	svc := NewAccountabilityGroupService(repo, cfg)
	_, err := svc.CreateGroup(ctx, "usr_gading", "bad", "")
	require.EqualError(t, err, "only a partner can create an accountability group")
	_, err = svc.CreateGroup(ctx, "usr_suci", "", "")
	require.EqualError(t, err, "group name or description is invalid")
	group, err := svc.CreateGroup(ctx, "usr_suci", "Coverage Group", "A test group")
	require.NoError(t, err)
	assert.Len(t, group.JoinCode, 10)

	_, err = svc.PreviewJoin(ctx, "usr_nasywa", group.JoinCode)
	require.EqualError(t, err, "only a student can join an accountability group")
	_, err = svc.PreviewJoin(ctx, "usr_dery", "short")
	require.EqualError(t, err, "join code is invalid")
	preview, err := svc.PreviewJoin(ctx, "usr_dery", group.JoinCode)
	require.NoError(t, err)
	assert.Empty(t, preview.JoinCodeHash)

	_, err = svc.Join(ctx, "usr_dery", group.JoinCode, false)
	require.EqualError(t, err, "group confirmation is required")
	joined, err := svc.Join(ctx, "usr_dery", group.JoinCode, true)
	require.NoError(t, err)
	assert.Equal(t, group.ID, joined.GroupID)
	updated, err := svc.UpdateSharing(ctx, "usr_dery", joined.ID, model.SharingPreferences{ProtectionActivity: false})
	require.NoError(t, err)
	assert.False(t, updated.Sharing.ProtectionActivity)

	exit, err := svc.RequestLeave(ctx, "usr_dery", joined.ID, "normal", "need a break")
	require.NoError(t, err)
	assert.Equal(t, "pending", exit.Status)
	require.NoError(t, svc.CancelLeave(ctx, "usr_dery", exit.ID))
	exit, err = svc.RequestLeave(ctx, "usr_dery", joined.ID, "unsafe", "safety concern")
	require.NoError(t, err)
	assert.Equal(t, "unsafe", exit.Kind)
	_, err = svc.RequestLeave(ctx, "usr_dery", joined.ID, "invalid", "")
	require.EqualError(t, err, "student is not authorized for this membership")

	contact, err := svc.CreateContactRequest(ctx, "usr_gading", "mbr_active", "check_in", "  Can we talk? ")
	require.NoError(t, err)
	assert.NotEmpty(t, contact.ID)
	require.NoError(t, svc.TransitionContactRequest(ctx, "usr_suci", contact.ID, "acknowledged"))
	workspace, err := svc.Workspace(ctx, "usr_suci")
	require.NoError(t, err)
	assert.NotEmpty(t, workspace.Groups)
	assert.NotEmpty(t, workspace.Members)
	partnerSummary, err := svc.Summary(ctx, "usr_suci")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, partnerSummary.ActiveGroups, 1)
	studentSummary, err := svc.Summary(ctx, "usr_gading")
	require.NoError(t, err)
	assert.NotNil(t, studentSummary.Membership)
	members, err := svc.MembersPaginated(ctx, "usr_suci", model.PaginationQuery{Query: "gading"})
	require.NoError(t, err)
	assert.NotEmpty(t, members.Items)
	analytics, err := svc.AnalyticsMembersPaginated(ctx, "usr_suci", model.PaginationQuery{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, analytics.TotalMembers, 1)
	groups, err := svc.GroupsPaginated(ctx, "usr_suci", model.PaginationQuery{Status: "active", Query: "coverage"})
	require.NoError(t, err)
	assert.NotEmpty(t, groups.Items)
	exits, err := svc.ExitRequestsPaginated(ctx, "usr_dery", model.PaginationQuery{})
	require.NoError(t, err)
	assert.NotNil(t, exits.Items)
	contacts, err := svc.ContactRequestsPaginated(ctx, "usr_suci", model.PaginationQuery{Bucket: "incoming"})
	require.NoError(t, err)
	assert.NotNil(t, contacts.Items)

	flagged, err := svc.FlaggedMembers(ctx, "usr_suci", model.PaginationQuery{Limit: 3})
	require.NoError(t, err)
	assert.NotNil(t, flagged.Items)
	assert.True(t, liveMembershipStatus("safety_suspended"))
	assert.False(t, liveMembershipStatus("removed"))
	assert.ElementsMatch(t, []string{"status", "protection", "inactive", "noCheckIn"}, computeMonitorFlags(model.AccountabilityMembership{Status: "leave_pending", Aggregate: model.MemberAggregateSummary{ProtectionStatus: "attention", LastHeartbeatBucket: "never"}}))
	assert.Equal(t, 0, minMonitorSeverity([]string{"status", "protection"}))
	assert.Equal(t, 4, minMonitorSeverity([]string{"unknown"}))

	require.NoError(t, svc.RemoveMember(ctx, "usr_suci", joined.ID, "cleanup"))
}

func TestAccountabilityGroupErrorsAndDeletion(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewAccountabilityGroupService(repo, testCfg())
	group, err := svc.CreateGroup(ctx, "usr_suci", "Delete Me", "")
	require.NoError(t, err)
	joined, err := svc.Join(ctx, "usr_dery", group.JoinCode, true)
	require.NoError(t, err)
	_, err = svc.CreateContactRequest(ctx, "usr_dery", "missing", "check_in", "x")
	require.EqualError(t, err, "an active membership is required")
	_, err = svc.CreateContactRequest(ctx, "usr_dery", group.ID, "check_in", strings.Repeat("x", 1001))
	require.Error(t, err)
	require.Error(t, svc.RemoveMember(ctx, "usr_gading", group.ID, "wrong owner"))
	require.NoError(t, svc.RemoveMember(ctx, "usr_suci", joined.ID, "cleanup"))
	require.NoError(t, svc.DeleteGroup(ctx, "usr_suci", group.ID))
	require.Error(t, svc.DeleteGroup(ctx, "usr_suci", group.ID))
}

func TestAccountabilityServiceInvitationsApprovalsQuickTokensAndStandaloneGrant(t *testing.T) {
	ctx := context.Background()
	repo, st := newRepo(t)
	cfg := testCfg()
	cfg.NotificationMode = "demo"
	whatsapp := NewWhatsAppService(cfg, zap.NewNop())
	svc := NewAccountabilityService(repo, cfg, whatsapp, zap.NewNop())
	_, _, err := svc.CreatePartnerInvitation(ctx, "missing", "partner@example.com", "")
	require.EqualError(t, err, "partner must use a different account")
	_, _, err = svc.CreatePartnerInvitation(ctx, "usr_gading", "", "")
	require.EqualError(t, err, "partner email is required")
	_, _, err = svc.CreatePartnerInvitation(ctx, "usr_gading", "gading@gmail.com", "")
	require.EqualError(t, err, "partner must use a different account")
	partner, inviteURL, err := svc.CreatePartnerInvitation(ctx, "usr_dery", "new-partner@example.com", "+6281200000000")
	require.NoError(t, err)
	assert.Contains(t, inviteURL, "/partner/invitations/")
	assert.Equal(t, "invited", partner.Status)
	err = svc.AcceptInvitation(ctx, "invalid", "usr_suci")
	require.EqualError(t, err, "invalid token or invitation already accepted")

	_, err = svc.GroupAnalytics(ctx, "usr_suci", "", 7)
	require.EqualError(t, err, "analytics period must be 14 or 30 days")
	_, err = svc.GroupAnalytics(ctx, "usr_suci", "", 14)
	require.NoError(t, err)
	page, err := svc.GetApprovalRequestsPaginated(ctx, "usr_gading", model.PaginationQuery{Status: "pending", Query: "APR"})
	require.NoError(t, err)
	assert.NotNil(t, page.Items)
	_, err = svc.CreateApprovalRequest(ctx, "usr_gading", "dev_android", "mbr_active", "pause_protection", "reason", 7)
	require.EqualError(t, err, "pause duration must be 15, 30, 60, or 120 minutes")
	_, err = svc.CreateApprovalRequest(ctx, "usr_gading", "dev_android", "mbr_active", "uninstall_detected", "reason", 15)
	require.EqualError(t, err, "requested duration is only valid for pause protection")
	request, err := svc.CreateApprovalRequest(ctx, "usr_gading", "dev_android", "mbr_active", "uninstall_detected", "reason", 0)
	require.NoError(t, err)
	require.NoError(t, svc.CancelApprovalRequest(ctx, request.ID, "usr_gading"))
	err = svc.ResolveApprovalAsPartner(ctx, request.ID, "approved", "usr_suci", strings.Repeat("x", 501))
	require.EqualError(t, err, "supportive response is too long")

	token := "qapp-test-token"
	now := time.Now().UTC()
	st.Lock()
	st.Approvals = append(st.Approvals, model.ApprovalRequest{ID: "quick-1", UserID: "usr_gading", Action: "pause_protection", Reason: "quick", Status: "pending", RequestedDurationMinutes: 15, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	st.Unlock()
	st.SetTokenMapping(HashRefreshToken(token), store.ApprovalRequest{ID: "quick-1", UserID: "usr_gading", Action: "pause_protection", Reason: "quick", Status: "pending", RequestedDurationMinutes: 15, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	preview, err := svc.VerifyQuickToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, "quick-1", preview["request_id"])
	require.NoError(t, svc.ResolveByToken(ctx, token, "approved"))
	_, err = svc.VerifyQuickToken(ctx, token)
	require.Error(t, err)
	err = svc.ResolveByToken(ctx, "invalid", "approved")
	require.EqualError(t, err, "token tidak valid")

	bindTestGrantKey(t, repo, "usr_dery", "dev_dery_android")
	grant, err := svc.IssueStandaloneRemovalGrant(ctx, "usr_dery", "dev_dery_android")
	require.NoError(t, err)
	assert.Equal(t, "uninstall_detected", grant.Action)
	_, err = svc.IssueStandaloneRemovalGrant(ctx, "usr_gading", "dev_android")
	require.EqualError(t, err, "student has an active accountability partner")
	_, err = svc.IssueStandaloneRemovalGrant(ctx, "usr_dery", "")
	require.EqualError(t, err, "device id is required")
}

func TestAccountabilityPureHelpers(t *testing.T) {
	code, err := generateAccountabilityCode(10)
	require.NoError(t, err)
	assert.Len(t, code, 10)
	assert.NotEmpty(t, generateQuickToken())
}
