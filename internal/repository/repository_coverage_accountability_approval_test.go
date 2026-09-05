package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func TestRepositoryCoverageAccountabilityApproval_GroupMembershipExitContactStates(t *testing.T) {
	ctx := t.Context()
	joinedAt := time.Date(2026, time.January, 10, 8, 0, 0, 0, time.UTC)
	student := model.User{
		ID:          "cov-accountability-student",
		Email:       "student@example.test",
		DisplayName: "Coverage Student",
	}
	st := &store.Store{Users: []model.User{student}}
	repo := New(nil, st)

	group := model.AccountabilityGroup{
		ID:                "cov-accountability-group",
		OwnerPartnerID:    "cov-accountability-partner",
		Name:              "Coverage group",
		Description:       "State transition coverage",
		JoinCodeHash:      "cov-code-hash",
		JoinCodeHint:      "COV-123",
		JoinCodeEncrypted: "encrypted-code",
		Status:            "active",
		CodeRotatedAt:     joinedAt,
	}
	createdGroup, err := repo.CreateAccountabilityGroup(ctx, group)
	require.NoError(t, err)
	assert.Equal(t, group.ID, createdGroup.ID)

	groups, err := repo.ListAccountabilityGroups(ctx, group.OwnerPartnerID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 0, groups[0].MemberCount)

	gotGroup, err := repo.AccountabilityGroupByID(ctx, group.ID)
	require.NoError(t, err)
	assert.Equal(t, group.Name, gotGroup.Name)
	_, err = repo.AccountabilityGroupByID(ctx, "cov-missing-group")
	assert.EqualError(t, err, "accountability group not found")
	gotGroup, err = repo.AccountabilityGroupByCodeHash(ctx, group.JoinCodeHash)
	require.NoError(t, err)
	assert.Equal(t, group.ID, gotGroup.ID)
	_, err = repo.AccountabilityGroupByCodeHash(ctx, "cov-invalid-code")
	assert.EqualError(t, err, "join code is invalid")

	assert.EqualError(t, repo.RotateAccountabilityGroupCode(ctx, group.ID, "cov-wrong-partner", "x", "X", "Y", joinedAt), "partner is not authorized for this group")
	rotatedAt := joinedAt.Add(time.Hour)
	require.NoError(t, repo.RotateAccountabilityGroupCode(ctx, group.ID, group.OwnerPartnerID, "cov-rotated-hash", "ROTATED", "rotated-encrypted", rotatedAt))
	rotated, err := repo.AccountabilityGroupByCodeHash(ctx, "cov-rotated-hash")
	require.NoError(t, err)
	assert.Equal(t, "ROTATED", rotated.JoinCodeHint)

	membership := model.AccountabilityMembership{
		ID:        "cov-accountability-membership",
		GroupID:   group.ID,
		StudentID: student.ID,
		Status:    "active",
		Sharing:   model.SharingPreferences{ProtectionHealth: true},
		JoinedAt:  joinedAt,
	}
	saved, err := repo.SaveAccountabilityMembership(ctx, membership)
	require.NoError(t, err)
	assert.Equal(t, membership.ID, saved.ID)
	assert.Equal(t, student.DisplayName, saved.StudentName)
	assert.Equal(t, student.Email, saved.StudentMail)

	updated, err := repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{
		ID:        "ignored-id",
		GroupID:   group.ID,
		StudentID: student.ID,
		Status:    "support_review",
		Sharing:   model.SharingPreferences{ProtectionActivity: true},
		JoinedAt:  joinedAt.Add(time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, membership.ID, updated.ID)
	assert.Equal(t, "support_review", updated.Status)
	assert.True(t, updated.Sharing.ProtectionActivity)

	active, err := repo.ActiveMembershipForStudent(ctx, student.ID)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, membership.ID, active.ID)
	listed, err := repo.ListMembershipsForGroup(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, 1, func() int {
		groups, listErr := repo.ListAccountabilityGroups(ctx, group.OwnerPartnerID)
		if listErr != nil || len(groups) != 1 {
			return 0
		}
		return groups[0].MemberCount
	}())

	shared, err := repo.UpdateMembershipSharing(ctx, membership.ID, student.ID, model.SharingPreferences{EducationProgress: true})
	require.NoError(t, err)
	assert.True(t, shared.Sharing.EducationProgress)
	_, err = repo.UpdateMembershipSharing(ctx, membership.ID, "cov-other-student", model.SharingPreferences{})
	assert.EqualError(t, err, "student is not authorized for this membership")

	endedAt := joinedAt.Add(24 * time.Hour)
	require.NoError(t, repo.SetMembershipStatus(ctx, membership.ID, "left", &endedAt))
	left, err := repo.MembershipByID(ctx, membership.ID)
	require.NoError(t, err)
	assert.Equal(t, "left", left.Status)
	assert.Equal(t, endedAt, *left.EndedAt)
	assert.Nil(t, func() *model.AccountabilityMembership {
		item, lookupErr := repo.ActiveMembershipForStudent(ctx, student.ID)
		if lookupErr != nil {
			return nil
		}
		return item
	}())
	_, err = repo.MembershipByID(ctx, "cov-missing-membership")
	assert.EqualError(t, err, "membership not found")
	assert.EqualError(t, repo.SetMembershipStatus(ctx, "cov-missing-membership", "left", nil), "membership not found")

	reviewDue := joinedAt.Add(48 * time.Hour)
	exit, err := repo.CreateMembershipExitRequest(ctx, model.MembershipExitRequest{
		ID:           "cov-accountability-exit",
		MembershipID: membership.ID,
		RequestedBy:  student.ID,
		Kind:         "normal",
		Status:       "pending",
		Reason:       "coverage",
		ReviewDueAt:  &reviewDue,
		CreatedAt:    joinedAt.Add(2 * time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, reviewDue, *exit.ReviewDueAt)
	emptyExits, err := repo.ListExitRequests(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, emptyExits)
	exits, err := repo.ListExitRequests(ctx, []string{membership.ID})
	require.NoError(t, err)
	require.Len(t, exits, 1)
	assert.Equal(t, exit.ID, exits[0].ID)

	contact, err := repo.CreatePartnerContactRequest(ctx, model.PartnerContactRequest{
		ID:           "cov-accountability-contact",
		MembershipID: membership.ID,
		StudentID:    student.ID,
		PartnerID:    group.OwnerPartnerID,
		Category:     "support",
		Message:      "Please contact me",
		Status:       "pending",
		CreatedAt:    joinedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, "support", contact.Category)
	studentContacts, err := repo.ListPartnerContactRequests(ctx, student.ID, "user")
	require.NoError(t, err)
	require.Len(t, studentContacts, 1)
	assert.Equal(t, student.DisplayName, studentContacts[0].StudentName)
	partnerContacts, err := repo.ListPartnerContactRequests(ctx, group.OwnerPartnerID, "partner")
	require.NoError(t, err)
	assert.Len(t, partnerContacts, 1)
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, contact.ID, student.ID, "acknowledged"), "only the partner can acknowledge")
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, contact.ID, student.ID, "cancelled"))
	assert.NotNil(t, repositoryCoverageAccountabilityApprovalContact(st, contact.ID).ClosedAt)
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, contact.ID, student.ID, "unsupported"), "invalid contact request transition")

	// A group with only terminal memberships can be deleted, including the
	// no-membership-ID branch of the in-memory cleanup path.
	emptyGroup := model.AccountabilityGroup{ID: "cov-empty-group", OwnerPartnerID: group.OwnerPartnerID, Name: "Empty", Status: "active"}
	_, err = repo.CreateAccountabilityGroup(ctx, emptyGroup)
	require.NoError(t, err)
	require.NoError(t, repo.DeleteAccountabilityGroup(ctx, emptyGroup.ID, group.OwnerPartnerID))
	_, err = repo.AccountabilityGroupByID(ctx, emptyGroup.ID)
	assert.EqualError(t, err, "accountability group not found")
}

func TestRepositoryCoverageAccountabilityApproval_ApprovalVisibilityAndTokens(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC()
	partnerID := "cov-approval-partner"
	studentID := "cov-approval-student"
	groupID := "cov-approval-group"
	membershipID := "cov-approval-membership"
	st := &store.Store{
		AccountabilityGroups: []model.AccountabilityGroup{{ID: groupID, OwnerPartnerID: partnerID, Status: "active"}},
		AccountabilityMemberships: []model.AccountabilityMembership{{
			ID: membershipID, GroupID: groupID, StudentID: studentID, Status: "active",
		}},
		Approvals: []model.ApprovalRequest{
			{ID: "cov-approval-own", UserID: studentID, MembershipID: "other-membership", Action: "pause_protection", Status: "pending", RequestedDurationMinutes: 15, ExpiresAt: now.Add(time.Hour)},
			{ID: "cov-approval-partner", UserID: "other-student", MembershipID: membershipID, Action: "pause_protection", Status: "pending", RequestedDurationMinutes: 30, ExpiresAt: now.Add(time.Hour)},
			{ID: "cov-approval-expired", UserID: studentID, MembershipID: "other-membership", Action: "pause_protection", Status: "pending", ExpiresAt: now.Add(-time.Minute)},
			{ID: "cov-approval-hidden", UserID: "other-student", MembershipID: "other-membership", Action: "pause_protection", Status: "pending", ExpiresAt: now.Add(time.Hour)},
		},
	}
	repo := New(nil, st)

	partnerVisible, err := repo.GetApprovalRequests(ctx, partnerID)
	require.NoError(t, err)
	require.Len(t, partnerVisible, 1)
	assert.Equal(t, "cov-approval-partner", partnerVisible[0].ID)
	assert.Equal(t, "Pause protection for 30 minutes", partnerVisible[0].ActionLabel)

	studentVisible, err := repo.GetApprovalRequests(ctx, studentID)
	require.NoError(t, err)
	require.Len(t, studentVisible, 2)
	statuses := map[string]string{}
	for _, item := range studentVisible {
		statuses[item.ID] = item.Status
	}
	assert.Equal(t, "pending", statuses["cov-approval-own"])
	assert.Equal(t, "expired", statuses["cov-approval-expired"])

	assert.NoError(t, repo.CreateApprovalRequest(ctx, "cov-noop-create", studentID, "cov-device", "cov-link", "pause_protection", "reason", 15, now.Add(time.Hour)))
	assert.EqualError(t, repo.UpdateApprovalRequest(ctx, "cov-approval-expired", "approved", partnerID), "pending approval request not found")
	st.Approvals = append(st.Approvals, model.ApprovalRequest{
		ID: "cov-approval-update", UserID: studentID, MembershipID: membershipID, Status: "pending", ExpiresAt: now.Add(time.Hour), Action: "pause_protection",
	})
	require.NoError(t, repo.UpdateApprovalRequest(ctx, "cov-approval-update", "approved", partnerID))
	assert.Equal(t, "approved", repositoryCoverageAccountabilityApprovalRequest(st, "cov-approval-update").Status)

	_, err = repo.CreateApprovalRequestWithToken(ctx, "cov-approval-token", studentID, "cov-device", membershipID, "pause_protection", "token reason", 60, now.Add(time.Hour), "coverage-approval-token")
	require.NoError(t, err)
	repo.UpdateQuickTokenState("coverage-approval-token", store.ApprovalRequest{ID: "cov-approval-token", Status: "approved", Action: "pause_protection"})
	quick, err := repo.GetApprovalByQuickToken(ctx, "coverage-approval-token")
	require.NoError(t, err)
	assert.Equal(t, "approved", quick.Status)
	_, err = repo.GetApprovalByQuickToken(ctx, "coverage-approval-token-missing")
	assert.EqualError(t, err, "token not found")

	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "cov-approval-expired", partnerID, "approved", ""), "pending approval request not found")
	assert.EqualError(t, repo.ResolveApprovalAsPartner(ctx, "cov-approval-token", "cov-wrong-partner", "approved", ""), "pending approval request not found")
	assert.NoError(t, repo.ResolveApprovalAsPartner(ctx, "cov-approval-token", partnerID, "approved", "Approved"))
	assert.Equal(t, "approved", repositoryCoverageAccountabilityApprovalRequest(st, "cov-approval-token").Status)

	st.Approvals = append(st.Approvals,
		model.ApprovalRequest{ID: "cov-approval-unsupported", UserID: studentID, DeviceID: "cov-device", Action: "unsupported", Status: "approved", ResolvedAt: &now, ExpiresAt: now.Add(time.Hour)},
		model.ApprovalRequest{ID: "cov-approval-uninstall", UserID: studentID, DeviceID: "cov-device", Action: "uninstall_detected", Status: "approved", RequestedDurationMinutes: 0, ResolvedAt: &now, ExpiresAt: now.Add(time.Hour)},
	)
	_, err = repo.ApplyApprovedRequest(ctx, "cov-approval-unsupported", studentID, "cov-device", now, "unsupported-jti")
	assert.EqualError(t, err, "approval action cannot be applied by a protection client")
	uninstallGrant, err := repo.ApplyApprovedRequest(ctx, "cov-approval-uninstall", studentID, "cov-device", now, "uninstall-jti")
	require.NoError(t, err)
	assert.Equal(t, "uninstall_detected", uninstallGrant.Action)
}

func TestRepositoryCoverageAccountabilityApproval_InvitationAndStandaloneErrors(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC()
	st := &store.Store{Users: []model.User{
		{ID: "cov-invitation-owner", Email: "owner@example.test", DisplayName: "Owner"},
		{ID: "cov-invitation-partner", Email: "PARTNER@example.test", DisplayName: "Partner"},
		{ID: "cov-invitation-other", Email: "other@example.test", DisplayName: "Other"},
	}}
	repo := New(nil, st)

	created, err := repo.CreatePartnerInvitation(ctx, "cov-invitation-link", "cov-invitation-owner", "partner@example.test", nil, "coverage-invite-token")
	require.NoError(t, err)
	assert.Equal(t, "invited", created.Status)
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "cov-invitation-link", "cov-invitation-owner"), "invitation not found")
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "cov-invitation-link", "cov-invitation-other"), "invitation not found")
	require.NoError(t, repo.AcceptPartnerInvitation(ctx, "cov-invitation-link", "cov-invitation-partner"))
	assert.Equal(t, "active", repositoryCoverageAccountabilityApprovalPartner(st, "cov-invitation-link").Status)
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "cov-invitation-link", "cov-invitation-partner"), "invitation not found")
	require.NoError(t, repo.RevokePartner(ctx, "cov-invitation-link", "cov-invitation-partner"))
	assert.Equal(t, "revoked", repositoryCoverageAccountabilityApprovalPartner(st, "cov-invitation-link").Status)
	assert.EqualError(t, repo.RevokePartner(ctx, "cov-missing-link", "cov-invitation-owner"), "partner link not found")

	for _, claims := range [][4]string{
		{"", "user", "device", "jti"},
		{"request", "", "device", "jti"},
		{"request", "user", "", "jti"},
		{"request", "user", "device", ""},
	} {
		_, err = repo.IssueStandaloneRemovalGrant(ctx, claims[0], claims[1], claims[2], claims[3], now)
		assert.EqualError(t, err, "standalone removal grant claims are incomplete")
	}

	first, err := repo.IssueStandaloneRemovalGrant(ctx, "cov-standalone-one", "cov-standalone-user", "cov-device-one", "cov-standalone-jti-one", now)
	require.NoError(t, err)
	assert.Equal(t, 10*time.Minute, first.GrantExpiresAt.Sub(first.GrantStartsAt))
	second, err := repo.IssueStandaloneRemovalGrant(ctx, "cov-standalone-two", "cov-standalone-user", "cov-device-two", "cov-standalone-jti-two", now)
	require.NoError(t, err)
	assert.NotEqual(t, first.RequestID, second.RequestID)
	_, err = repo.IssueStandaloneRemovalGrant(ctx, "cov-standalone-three", "cov-standalone-user", "cov-device-one", "cov-standalone-jti-three", now)
	assert.EqualError(t, err, "a standalone removal grant is already active")
}

func repositoryCoverageAccountabilityApprovalRequest(st *store.Store, id string) model.ApprovalRequest {
	for _, item := range st.Snapshot().Approvals {
		if item.ID == id {
			return item
		}
	}
	return model.ApprovalRequest{}
}

func repositoryCoverageAccountabilityApprovalContact(st *store.Store, id string) model.PartnerContactRequest {
	for _, item := range st.Snapshot().PartnerContactRequests {
		if item.ID == id {
			return item
		}
	}
	return model.PartnerContactRequest{}
}

func repositoryCoverageAccountabilityApprovalPartner(st *store.Store, id string) model.Partner {
	for _, item := range st.Snapshot().Partners {
		if item.ID == id {
			return item
		}
	}
	return model.Partner{}
}
