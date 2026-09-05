package repository

import (
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountabilityRepository_InMemoryGroupsMembershipsAndAggregation(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	groups, err := repo.ListAccountabilityGroups(ctx, "usr_suci")
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 2, groups[0].MemberCount)
	group, err := repo.AccountabilityGroupByID(ctx, "grp_demo")
	require.NoError(t, err)
	assert.Equal(t, "Kelas Informatika C", group.Name)
	_, err = repo.AccountabilityGroupByID(ctx, "missing-group")
	require.EqualError(t, err, "accountability group not found")
	group, err = repo.AccountabilityGroupByCodeHash(ctx, "cf555032cd87549c8369da3e5148f4fdcc6833a78c2f905b9944d2fa4cc04c45")
	require.NoError(t, err)
	assert.Equal(t, "grp_demo", group.ID)
	_, err = repo.AccountabilityGroupByCodeHash(ctx, "bad-code")
	require.EqualError(t, err, "join code is invalid")

	require.NoError(t, repo.RotateAccountabilityGroupCode(ctx, "grp_demo", "usr_suci", "new-hash", "NEW", "encrypted", now))
	rotated, err := repo.AccountabilityGroupByCodeHash(ctx, "new-hash")
	require.NoError(t, err)
	assert.Equal(t, "NEW", rotated.JoinCodeHint)
	assert.EqualError(t, repo.RotateAccountabilityGroupCode(ctx, "grp_demo", "usr_gading", "bad", "", "", now), "partner is not authorized for this group")

	membership, err := repo.ActiveMembershipForStudent(ctx, "usr_gading")
	require.NoError(t, err)
	require.NotNil(t, membership)
	assert.Equal(t, "Gading", membership.StudentName)
	assert.Equal(t, "gading@gmail.com", membership.StudentMail)
	membershipByID, err := repo.MembershipByID(ctx, "mbr_active")
	require.NoError(t, err)
	assert.Equal(t, "usr_gading", membershipByID.StudentID)
	_, err = repo.ActiveMembershipForStudent(ctx, "missing-student")
	require.NoError(t, err)
	assert.Nil(t, membershipForStudent(t, repo, "missing-student"))
	_, err = repo.MembershipByID(ctx, "missing-membership")
	require.EqualError(t, err, "membership not found")

	updated, err := repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{ID: "ignored", GroupID: "grp_demo", StudentID: "usr_gading", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: false}, JoinedAt: now})
	require.NoError(t, err)
	assert.Equal(t, "mbr_active", updated.ID)
	assert.False(t, updated.Sharing.ProtectionActivity)
	created, err := repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{ID: "mbr_new", GroupID: "grp_demo", StudentID: "usr_dery", Status: "active", JoinedAt: now})
	require.NoError(t, err)
	assert.Equal(t, "mbr_new", created.ID)
	assert.Equal(t, "Dery", created.StudentName)
	updated, err = repo.UpdateMembershipSharing(ctx, "mbr_active", "usr_gading", model.SharingPreferences{ProtectionHealth: true})
	require.NoError(t, err)
	assert.True(t, updated.Sharing.ProtectionHealth)
	_, err = repo.UpdateMembershipSharing(ctx, "mbr_active", "usr_dery", model.SharingPreferences{})
	require.EqualError(t, err, "student is not authorized for this membership")
	require.NoError(t, repo.SetMembershipStatus(ctx, "mbr_new", "left", nil))
	assert.EqualError(t, repo.SetMembershipStatus(ctx, "missing-membership", "left", nil), "membership not found")

	list, err := repo.ListMembershipsForGroup(ctx, "grp_demo")
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// Exercise the privacy-gated aggregate projection independently from the
	// demo fixture so each sharing category has an observable result.
	aggregateStore := &store.Store{
		Users:             []model.User{{ID: "student", Email: "student@example.com", DisplayName: "Student"}},
		Devices:           []model.Device{{UserID: "student", ProtectionStatus: "active", LastSeenAt: now.Add(-2 * 24 * time.Hour)}},
		AggregateEvents:   []model.AggregateEvent{{UserID: "student", EventType: "block_count_sync", EventDate: now, Count: 4}},
		CheckIns:          []model.CheckIn{{UserID: "student", CreatedAt: now}},
		Missions:          []model.DailyMission{{UserID: "student", TaskRecords: []model.MissionRecord{{Source: "system", Status: "completed"}}}},
		EducationProgress: []model.EducationProgress{{UserID: "student", ProgressPercent: 50}},
	}
	item := model.AccountabilityMembership{StudentID: "student", Sharing: model.SharingPreferences{ProtectionHealth: true, ProtectionActivity: true, RecoveryEngagement: true, EducationProgress: true}}
	summary := aggregateForMembershipAt(aggregateStore, item, now)
	assert.Equal(t, "ready", summary.ProtectionStatus)
	assert.Equal(t, 1, summary.ActiveDeviceCount)
	assert.Equal(t, "1-3d", summary.LastHeartbeatBucket)
	assert.Equal(t, 4, summary.WeeklyBlockCount)
	assert.Equal(t, 1, summary.CheckInDays)
	assert.Equal(t, 1, summary.MissionCompleted)
	assert.Equal(t, "in_progress", summary.EducationProgressBand)
	assert.Equal(t, "never", heartbeatBucket(now, time.Time{}))
	assert.Equal(t, "today", heartbeatBucket(now, now.Add(-time.Hour)))
	assert.Equal(t, "4-7d", heartbeatBucket(now, now.Add(-4*24*time.Hour)))
	assert.Equal(t, "older", heartbeatBucket(now, now.Add(-8*24*time.Hour)))
}

func membershipForStudent(t *testing.T, repo *Repository, studentID string) *model.AccountabilityMembership {
	t.Helper()
	item, err := repo.ActiveMembershipForStudent(t.Context(), studentID)
	require.NoError(t, err)
	return item
}

func TestAccountabilityRepository_InMemoryExitAndContactTransitions(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	st.AccountabilityMemberships = append(st.AccountabilityMemberships,
		model.AccountabilityMembership{ID: "mbr_approve", GroupID: "grp_demo", StudentID: "usr_dery", Status: "leave_pending", JoinedAt: now},
		model.AccountabilityMembership{ID: "mbr_deny", GroupID: "grp_demo", StudentID: "usr_dery", Status: "leave_pending", JoinedAt: now},
		model.AccountabilityMembership{ID: "mbr_cancel", GroupID: "grp_demo", StudentID: "usr_dery", Status: "leave_pending", JoinedAt: now},
		model.AccountabilityMembership{ID: "mbr_escalate", GroupID: "grp_demo", StudentID: "usr_dery", Status: "leave_pending", JoinedAt: now},
	)
	st.MembershipExitRequests = append(st.MembershipExitRequests,
		model.MembershipExitRequest{ID: "exit_approve", MembershipID: "mbr_approve", RequestedBy: "usr_dery", Kind: "normal", Status: "pending", CreatedAt: now},
		model.MembershipExitRequest{ID: "exit_deny", MembershipID: "mbr_deny", RequestedBy: "usr_dery", Kind: "normal", Status: "pending", CreatedAt: now},
		model.MembershipExitRequest{ID: "exit_cancel", MembershipID: "mbr_cancel", RequestedBy: "usr_dery", Kind: "normal", Status: "pending", CreatedAt: now},
		model.MembershipExitRequest{ID: "exit_escalate", MembershipID: "mbr_escalate", RequestedBy: "usr_dery", Kind: "normal", Status: "pending", ReviewDueAt: timePtr(now.Add(-time.Hour)), CreatedAt: now.Add(-25 * time.Hour)},
	)

	exits, err := repo.ListExitRequests(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, exits)
	exits, err = repo.ListExitRequests(ctx, []string{"mbr_approve", "mbr_deny"})
	require.NoError(t, err)
	assert.Len(t, exits, 2)

	assert.EqualError(t, repo.ResolveMembershipExitRequest(ctx, "exit_approve", "usr_dery", "approved"), "partner is not authorized for this request")
	assert.EqualError(t, repo.ResolveMembershipExitRequest(ctx, "exit_approve", "usr_suci", "invalid"), "invalid exit decision")
	require.NoError(t, repo.ResolveMembershipExitRequest(ctx, "exit_approve", "usr_suci", "approved"))
	assert.Equal(t, "left", membershipStatus(st, "mbr_approve"))
	assert.Equal(t, "approved", exitStatus(st, "exit_approve"))

	require.NoError(t, repo.ResolveMembershipExitRequest(ctx, "exit_deny", "usr_suci", "denied"))
	assert.Equal(t, "active", membershipStatus(st, "mbr_deny"))
	assert.Equal(t, "denied", exitStatus(st, "exit_deny"))
	assert.EqualError(t, repo.ResolveMembershipExitRequest(ctx, "exit_deny", "usr_suci", "approved"), "partner is not authorized for this request")

	require.NoError(t, repo.CancelMembershipExitRequest(ctx, "exit_cancel", "usr_dery"))
	assert.Equal(t, "active", membershipStatus(st, "mbr_cancel"))
	assert.Equal(t, "cancelled", exitStatus(st, "exit_cancel"))
	assert.EqualError(t, repo.CancelMembershipExitRequest(ctx, "exit_cancel", "usr_dery"), "pending normal exit request not found for student")
	require.NoError(t, repo.CancelPendingNormalExitRequestsForMembership(ctx, "mbr_deny", "usr_suci"))

	require.NoError(t, repo.EscalateOverdueExitRequests(ctx, now))
	assert.Equal(t, "support_review", membershipStatus(st, "mbr_escalate"))
	assert.Equal(t, "auto_reviewed", exitStatus(st, "exit_escalate"))

	contact := model.PartnerContactRequest{ID: "contact_cov", MembershipID: "mbr_active", StudentID: "usr_gading", PartnerID: "usr_suci", Category: "support", Message: "Need help", Status: "pending", CreatedAt: now}
	_, err = repo.CreatePartnerContactRequest(ctx, contact)
	require.NoError(t, err)
	studentRequests, err := repo.ListPartnerContactRequests(ctx, "usr_gading", "user")
	require.NoError(t, err)
	require.Len(t, studentRequests, 1)
	assert.Equal(t, "Gading", studentRequests[0].StudentName)
	partnerRequests, err := repo.ListPartnerContactRequests(ctx, "usr_suci", "partner")
	require.NoError(t, err)
	assert.Len(t, partnerRequests, 1)
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "contact_cov", "usr_dery", "closed"), "actor is not authorized for contact request")
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "contact_cov", "usr_gading", "acknowledged"), "only the partner can acknowledge")
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "contact_cov", "usr_suci", "invalid"), "invalid contact request transition")
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "contact_cov", "usr_suci", "acknowledged"))
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "contact_cov", "usr_gading", "closed"))
	assert.Equal(t, "closed", contactStatus(st, "contact_cov"))

	oldContact := contact
	oldContact.ID = "contact_escalate"
	oldContact.Status = "pending"
	oldContact.CreatedAt = now.Add(-25 * time.Hour)
	st.PartnerContactRequests = append(st.PartnerContactRequests, oldContact)
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "contact_escalate", "usr_suci", "escalated"), "escalation is available to the student after 24 hours")
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "contact_escalate", "usr_gading", "escalated"))
	assert.NotNil(t, contactValue(st, "contact_escalate").EscalatedAt)
}

func membershipStatus(st *store.Store, id string) string {
	for _, item := range st.Snapshot().AccountabilityMemberships {
		if item.ID == id {
			return item.Status
		}
	}
	return ""
}

func exitStatus(st *store.Store, id string) string {
	for _, item := range st.Snapshot().MembershipExitRequests {
		if item.ID == id {
			return item.Status
		}
	}
	return ""
}

func contactValue(st *store.Store, id string) model.PartnerContactRequest {
	for _, item := range st.Snapshot().PartnerContactRequests {
		if item.ID == id {
			return item
		}
	}
	return model.PartnerContactRequest{}
}

func contactStatus(st *store.Store, id string) string {
	return contactValue(st, id).Status
}

func TestAccountabilityRepository_InMemoryDeleteAndPartnerQueries(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()

	assert.True(t, repo.IsActivePartnerLinkOwnedBy(ctx, "pl_active", "usr_gading"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "pl_active", "usr_suci"))
	assert.Equal(t, "+62 812-0000-0000", repo.GetActivePartnerPhone(ctx, "pl_active", "usr_gading"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "pl_active", "usr_suci"))

	group := model.AccountabilityGroup{ID: "grp_delete", OwnerPartnerID: "usr_suci", Name: "Delete me", Status: "active"}
	require.NoError(t, func() error { _, err := repo.CreateAccountabilityGroup(ctx, group); return err }())
	st.AccountabilityMemberships = append(st.AccountabilityMemberships, model.AccountabilityMembership{ID: "mbr_deleted", GroupID: group.ID, StudentID: "usr_dery", Status: "left"})
	st.MembershipExitRequests = append(st.MembershipExitRequests, model.MembershipExitRequest{ID: "exit_deleted", MembershipID: "mbr_deleted"})
	st.PartnerContactRequests = append(st.PartnerContactRequests, model.PartnerContactRequest{ID: "contact_deleted", MembershipID: "mbr_deleted"})
	st.Approvals = append(st.Approvals, model.ApprovalRequest{ID: "approval_deleted", MembershipID: "mbr_deleted"})
	require.NoError(t, repo.DeleteAccountabilityGroup(ctx, group.ID, "usr_suci"))
	assert.EqualError(t, repo.DeleteAccountabilityGroup(ctx, group.ID, "usr_suci"), "partner is not authorized for this group")
	assert.EqualError(t, repo.DeleteAccountabilityGroup(ctx, "grp_demo", "usr_suci"), "group still has active members")
	assert.NotContains(t, st.Snapshot().AccountabilityGroups, group)

	_, err := repo.GetPartnerLinkByToken(ctx, "missing-token")
	assert.EqualError(t, err, "invitation not found")
	assert.NoError(t, repo.RevokePartner(ctx, "pl_active", "usr_gading"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "pl_active", "usr_gading"))
}
