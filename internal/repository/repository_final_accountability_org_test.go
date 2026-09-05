package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestRepositoryFinalAccountabilityOrg_ExitFilteringAndOverdue(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	st := store.New()
	st.Users = []model.User{{ID: "final-student", Email: "final-student@example.test", DisplayName: "Final Student"}}
	st.AccountabilityGroups = []model.AccountabilityGroup{
		{ID: "final-group", OwnerPartnerID: "final-partner", Name: "Final Group", Status: "active", JoinCodeHash: "final-code", CreatedAt: now, UpdatedAt: now},
		{ID: "final-inactive", OwnerPartnerID: "final-partner", Name: "Inactive Group", Status: "archived", JoinCodeHash: "inactive-code", CreatedAt: now, UpdatedAt: now},
	}
	st.AccountabilityMemberships = []model.AccountabilityMembership{
		{ID: "final-live", GroupID: "final-group", StudentID: "final-student", Status: "active", JoinedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "final-support", GroupID: "final-group", StudentID: "final-support-student", Status: "support_review", JoinedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "final-terminal", GroupID: "final-group", StudentID: "final-terminal-student", Status: "left", JoinedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "final-other-group", GroupID: "other-group", StudentID: "final-other-student", Status: "active", JoinedAt: now, CreatedAt: now, UpdatedAt: now},
	}
	st.MembershipExitRequests = []model.MembershipExitRequest{
		{ID: "final-overdue", MembershipID: "final-live", RequestedBy: "final-student", Kind: "normal", Status: "pending", ReviewDueAt: repositoryFinalTimePtr(now.Add(-time.Minute)), CreatedAt: now.Add(-4 * time.Hour), UpdatedAt: now.Add(-4 * time.Hour)},
		{ID: "final-future", MembershipID: "final-support", RequestedBy: "future-student", Kind: "normal", Status: "pending", ReviewDueAt: repositoryFinalTimePtr(now.Add(time.Hour)), CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: "final-unsafe", MembershipID: "final-terminal", RequestedBy: "terminal-student", Kind: "unsafe", Status: "pending", ReviewDueAt: repositoryFinalTimePtr(now.Add(-time.Hour)), CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "final-no-due", MembershipID: "final-support", RequestedBy: "support-student", Kind: "normal", Status: "pending", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
	}
	repo := New(nil, st)

	groups, err := repo.ListAccountabilityGroups(ctx, "final-partner")
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, 2, groups[0].MemberCount, "active and support_review memberships are live; left is terminal")
	assert.EqualError(t, func() error { _, err := repo.AccountabilityGroupByCodeHash(ctx, "inactive-code"); return err }(), "join code is invalid")
	assert.EqualError(t, repo.RotateAccountabilityGroupCode(ctx, "final-inactive", "final-partner", "new", "N", "E", now), "partner is not authorized for this group")
	assert.EqualError(t, repo.RotateAccountabilityGroupCode(ctx, "missing-final-group", "final-partner", "new", "N", "E", now), "partner is not authorized for this group")

	exits, err := repo.ListExitRequests(ctx, []string{"final-support", "final-live", "not-present"})
	require.NoError(t, err)
	require.Len(t, exits, 3)
	assert.Equal(t, "final-no-due", exits[0].ID)
	assert.Equal(t, "final-future", exits[1].ID)
	assert.Equal(t, "final-overdue", exits[2].ID)
	empty, err := repo.ListExitRequests(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, empty)
	assert.Empty(t, empty)

	require.NoError(t, repo.EscalateOverdueExitRequests(ctx, now))
	assert.Equal(t, "auto_reviewed", repositoryFinalExitStatus(st, "final-overdue"))
	assert.Equal(t, "support_review", repositoryFinalMembershipStatus(st, "final-live"))
	assert.Equal(t, "pending", repositoryFinalExitStatus(st, "final-future"))
	assert.Equal(t, "pending", repositoryFinalExitStatus(st, "final-unsafe"))
	assert.Nil(t, repositoryFinalActiveMembership(t, repo, "final-terminal"))
	active, err := repo.ActiveMembershipForStudent(ctx, "final-student")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, "support_review", active.Status)

	rotatedAt := now.Add(time.Minute)
	require.NoError(t, repo.RotateAccountabilityGroupCode(ctx, "final-group", "final-partner", "rotated-code", "RC", "encrypted", rotatedAt))
	rotated, err := repo.AccountabilityGroupByCodeHash(ctx, "rotated-code")
	require.NoError(t, err)
	assert.Equal(t, "RC", rotated.JoinCodeHint)
}

func TestRepositoryFinalAccountabilityOrg_MembershipContactAndTerminalErrors(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, time.September, 5, 13, 0, 0, 0, time.UTC)
	st := store.New()
	st.Users = []model.User{{ID: "final-student", Email: "student@example.test", DisplayName: "Student"}}
	st.AccountabilityGroups = []model.AccountabilityGroup{{ID: "final-contact-group", OwnerPartnerID: "final-partner", Status: "active"}}
	st.AccountabilityMemberships = []model.AccountabilityMembership{{ID: "final-contact-membership", GroupID: "final-contact-group", StudentID: "final-student", Status: "active", JoinedAt: now}}
	repo := New(nil, st)

	created, err := repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{
		ID: "final-new-membership", GroupID: "final-contact-group", StudentID: "new-student", Status: "active", JoinedAt: now,
	})
	require.NoError(t, err)
	assert.Equal(t, "final-new-membership", created.ID)
	updated, err := repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{
		ID: "ignored-id", GroupID: "final-contact-group", StudentID: "new-student", Status: "left", JoinedAt: now,
	})
	require.NoError(t, err)
	assert.Equal(t, "final-new-membership", updated.ID)
	assert.EqualError(t, func() error {
		_, err := repo.UpdateMembershipSharing(ctx, "final-new-membership", "new-student", model.SharingPreferences{})
		return err
	}(), "student is not authorized for this membership")
	assert.EqualError(t, func() error {
		_, err := repo.UpdateMembershipSharing(ctx, "missing-membership", "new-student", model.SharingPreferences{})
		return err
	}(), "student is not authorized for this membership")
	require.NoError(t, repo.SetMembershipStatus(ctx, "final-contact-membership", "leave_pending", repositoryFinalTimePtr(now)))
	assert.Equal(t, "leave_pending", repositoryFinalMembershipStatus(st, "final-contact-membership"))

	contacts := []model.PartnerContactRequest{
		{ID: "final-contact-ack", MembershipID: "final-contact-membership", StudentID: "final-student", PartnerID: "final-partner", Category: "accountability", Status: "pending", CreatedAt: now},
		{ID: "final-contact-close", MembershipID: "final-contact-membership", StudentID: "final-student", PartnerID: "final-partner", Category: "other", Status: "pending", CreatedAt: now.Add(-time.Hour)},
		{ID: "final-contact-cancel", MembershipID: "final-contact-membership", StudentID: "final-student", PartnerID: "final-partner", Category: "check_in", Status: "pending", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "final-contact-escalate", MembershipID: "final-contact-membership", StudentID: "final-student", PartnerID: "final-partner", Category: "practical_help", Status: "pending", CreatedAt: time.Now().UTC().Add(-25 * time.Hour)},
	}
	for _, contact := range contacts {
		_, err := repo.CreatePartnerContactRequest(ctx, contact)
		require.NoError(t, err)
	}

	studentContacts, err := repo.ListPartnerContactRequests(ctx, "final-student", "user")
	require.NoError(t, err)
	assert.Len(t, studentContacts, 4)
	partnerContacts, err := repo.ListPartnerContactRequests(ctx, "final-partner", "partner")
	require.NoError(t, err)
	assert.Len(t, partnerContacts, 4)
	emptyContacts, err := repo.ListPartnerContactRequests(ctx, "unrelated-final-user", "user")
	require.NoError(t, err)
	assert.Empty(t, emptyContacts)

	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-ack", "unrelated-final-user", "closed"), "actor is not authorized for contact request")
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-ack", "final-student", "acknowledged"), "only the partner can acknowledge")
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-ack", "final-partner", "unsupported"), "invalid contact request transition")
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-ack", "final-partner", "acknowledged"))
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-close", "final-partner", "closed"))
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-cancel", "final-student", "cancelled"))
	assert.EqualError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-escalate", "final-partner", "escalated"), "escalation is available to the student after 24 hours")
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "final-contact-escalate", "final-student", "escalated"))
	assert.Equal(t, "acknowledged", repositoryFinalContactStatus(st, "final-contact-ack"))
	assert.Equal(t, "closed", repositoryFinalContactStatus(st, "final-contact-close"))
	assert.Equal(t, "cancelled", repositoryFinalContactStatus(st, "final-contact-cancel"))
	assert.NotNil(t, repositoryFinalContact(st, "final-contact-escalate").EscalatedAt)
}

func TestRepositoryFinalAccountabilityOrg_OrganizationMetricsAndPartnerFiltering(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, time.September, 5, 14, 0, 0, 0, time.UTC)
	st := store.New()
	st.Users = []model.User{
		{ID: "final-owner", Email: "owner@example.test", DisplayName: "Owner"},
		{ID: "final-partner-user", Email: "partner@example.test", DisplayName: "Partner"},
		{ID: "final-other-user", Email: "other@example.test", DisplayName: "Other"},
	}
	st.Partners = []model.Partner{
		{ID: "final-active", UserID: "final-owner", PartnerUserID: "final-partner-user", Name: "Active", Contact: "owner@example.test | +628111", PartnerEmail: "partner@example.test", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "final-invited", UserID: "final-owner", PartnerEmail: "partner@example.test", InviteTokenHash: "final-valid-token", Status: "invited", CreatedAt: now, UpdatedAt: now},
		{ID: "final-expired", UserID: "final-owner", PartnerEmail: "partner@example.test", InviteTokenHash: "final-expired-token", Status: "invited", CreatedAt: now.Add(-8 * 24 * time.Hour), UpdatedAt: now},
		{ID: "final-no-separator", UserID: "final-owner", PartnerEmail: "partner@example.test", Contact: "email-only", Status: "active", CreatedAt: now, UpdatedAt: now},
	}
	repo := New(nil, st)

	active, items, err := repo.GetPartners(ctx, "final-owner")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Len(t, items, 4)
	assert.Equal(t, "owner", active.RelationshipRole)
	partnerView, partnerItems, err := repo.GetPartners(ctx, "final-partner-user")
	require.NoError(t, err)
	require.NotNil(t, partnerView)
	assert.Equal(t, "partner", partnerView.RelationshipRole)
	assert.Len(t, partnerItems, 1)
	emptyActive, emptyItems, err := repo.GetPartners(ctx, "final-other-user")
	require.NoError(t, err)
	assert.Nil(t, emptyActive)
	assert.Empty(t, emptyItems)

	assert.True(t, repo.IsActivePartnerLinkOwnedBy(ctx, "final-active", "final-owner"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "final-active", "final-partner-user"))
	assert.Equal(t, "+628111", repo.GetActivePartnerPhone(ctx, "final-active", "final-owner"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "final-no-separator", "final-owner"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "missing-final-link", "final-owner"))
	assert.Equal(t, "final-invited", repositoryFinalMustPartnerToken(t, repo, ctx, "final-valid-token"))
	assert.EqualError(t, func() error { _, err := repo.GetPartnerLinkByToken(ctx, "final-expired-token"); return err }(), "invitation not found")

	created, err := repo.CreateOrganization(ctx, "final-org", "Final Org", "final-org", "FINAL", "final-owner")
	require.NoError(t, err)
	assert.Equal(t, "FINAL", created.GroupCode)
	org, err := repo.GetOrganizationByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Name, org.Name)
	byOwner, err := repo.GetOrganizationByUserID(ctx, "final-owner")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byOwner.ID)
	byCode, err := repo.GetOrganizationByGroupCode(ctx, "FINAL")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byCode.ID)
	assert.EqualError(t, func() error { _, err := repo.GetOrganizationByID(ctx, "missing-final-org"); return err }(), "organisasi tidak ditemukan")
	assert.EqualError(t, func() error { _, err := repo.GetOrganizationByUserID(ctx, "missing-final-user"); return err }(), "tidak ada grup")
	assert.EqualError(t, func() error { _, err := repo.GetOrganizationByGroupCode(ctx, "missing-final-code"); return err }(), "kode grup tidak valid")
	assert.Equal(t, "short", idToGroupCode("short"))
	assert.Equal(t, "123456", idToGroupCode("org123456"))

	assert.NoError(t, repo.CreateOrganizationMember(ctx, "final-member", created.ID, "final-owner", "member", "active", nil))
	members, err := repo.ListOrganizationMembers(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, members, 3)
	assert.EqualError(t, func() error { _, err := repo.GetOrganizationMember(ctx, created.ID, "final-owner"); return err }(), "not found")
	assert.NoError(t, repo.RemoveOrganizationMember(ctx, created.ID, "final-owner"))
	count, err := repo.CountPendingApprovalsForOrg(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	summary, err := repo.GetMemberProgressSummary(ctx, "final-owner")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.ActiveDevices)
	assert.Equal(t, 5+len("final-owner")%20, summary.BlockedAttempts)
}

func TestRepositoryFinalAccountabilityOrg_PartnerInvitationErrorBranches(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC()
	st := store.New()
	st.Users = []model.User{
		{ID: "final-invite-owner", Email: "owner@example.test"},
		{ID: "final-invitee", Email: "Invitee@Example.Test"},
		{ID: "final-wrong-email", Email: "wrong@example.test"},
	}
	st.Partners = []model.Partner{
		{ID: "final-valid-invite", UserID: "final-invite-owner", PartnerEmail: "invitee@example.test", Status: "invited", CreatedAt: now, UpdatedAt: now},
		{ID: "final-wrong-invite", UserID: "final-invite-owner", PartnerEmail: "wrong@example.test", Status: "invited", CreatedAt: now, UpdatedAt: now},
		{ID: "final-active-invite", UserID: "final-invite-owner", PartnerEmail: "invitee@example.test", Status: "active", CreatedAt: now, UpdatedAt: now},
	}
	repo := New(nil, st)

	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "missing-final-invite", "final-invitee"), "invitation not found")
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "final-valid-invite", "final-invite-owner"), "invitation not found")
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "final-wrong-invite", "final-invitee"), "invitation not found")
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "final-valid-invite", "final-wrong-email"), "invitation not found")
	require.NoError(t, repo.AcceptPartnerInvitation(ctx, "final-valid-invite", "final-invitee"))
	assert.Equal(t, "active", repositoryFinalPartnerStatus(st, "final-valid-invite"))
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "final-valid-invite", "final-invitee"), "invitation not found")

	assert.EqualError(t, repo.RevokePartner(ctx, "missing-final-link", "final-invite-owner"), "partner link not found")
	require.NoError(t, repo.RevokePartner(ctx, "final-valid-invite", "final-invitee"))
	assert.Equal(t, "revoked", repositoryFinalPartnerStatus(st, "final-valid-invite"))
	assert.EqualError(t, repo.RevokePartner(ctx, "final-active-invite", "unrelated-final-user"), "partner link not found")
}

func TestRepositoryFinalAccountabilityOrg_SQLitePersistenceBranches(t *testing.T) {
	client := repositoryFinalOpenSQLite(t)
	defer client.Close()
	ctx := t.Context()
	repo := New(client, store.New())
	now := time.Date(2026, time.September, 5, 15, 0, 0, 0, time.UTC)

	_, err := repo.CreateUserWithPassword(ctx, "final-db-owner", "db-owner@example.test", "DB Owner", "hash", "partner")
	require.NoError(t, err)
	_, err = repo.CreateUserWithPassword(ctx, "final-db-student", "db-student@example.test", "DB Student", "hash", "user")
	require.NoError(t, err)
	_, err = repo.CreateUserWithPassword(ctx, "final-db-invitee", "db-invitee@example.test", "DB Invitee", "hash", "partner")
	require.NoError(t, err)

	group, err := repo.CreateAccountabilityGroup(ctx, model.AccountabilityGroup{
		ID: "final-db-group", OwnerPartnerID: "final-db-owner", Name: "DB Group", Description: "database", JoinCodeHash: "final-db-code", JoinCodeHint: "DB", JoinCodeEncrypted: "encrypted", CodeRotatedAt: now,
	})
	require.NoError(t, err)
	assert.Equal(t, "final-db-group", group.ID)
	membership, err := repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{
		ID: "final-db-membership", GroupID: group.ID, StudentID: "final-db-student", Status: "active", JoinedAt: now,
	})
	require.NoError(t, err)
	assert.Equal(t, "final-db-student", membership.StudentID)
	groups, err := repo.ListAccountabilityGroups(ctx, "final-db-owner")
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, 1, groups[0].MemberCount)
	byCode, err := repo.AccountabilityGroupByCodeHash(ctx, "final-db-code")
	require.NoError(t, err)
	assert.Equal(t, group.ID, byCode.ID)
	_, err = repo.ActiveMembershipForStudent(ctx, "final-db-student")
	require.NoError(t, err)

	due := now.Add(-time.Hour)
	_, err = repo.CreateMembershipExitRequest(ctx, model.MembershipExitRequest{
		ID: "final-db-exit", MembershipID: membership.ID, RequestedBy: "final-db-student", Kind: "normal", Status: "pending", ReviewDueAt: &due, CreatedAt: now,
	})
	require.NoError(t, err)
	exits, err := repo.ListExitRequests(ctx, []string{membership.ID})
	require.NoError(t, err)
	require.Len(t, exits, 1)
	require.NoError(t, repo.EscalateOverdueExitRequests(ctx, now))
	assert.Equal(t, "auto_reviewed", mustRepositoryFinalDBExit(t, client, "final-db-exit"))

	_, err = repo.CreatePartnerContactRequest(ctx, model.PartnerContactRequest{
		ID: "final-db-contact", MembershipID: membership.ID, StudentID: "final-db-student", PartnerID: "final-db-owner", Category: "other", Message: "encrypted message",
	})
	require.NoError(t, err)
	contacts, err := repo.ListPartnerContactRequests(ctx, "final-db-owner", "partner")
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	require.NoError(t, repo.TransitionPartnerContactRequest(ctx, "final-db-contact", "final-db-owner", "acknowledged"))

	created, err := repo.CreatePartnerInvitation(ctx, "final-db-invitation", "final-db-owner", "db-invitee@example.test", nil, "final-db-token")
	require.NoError(t, err)
	assert.Equal(t, "invited", created.Status)
	assert.Equal(t, created.ID, repositoryFinalMustPartnerToken(t, repo, ctx, "final-db-token"))
	require.NoError(t, repo.AcceptPartnerInvitation(ctx, created.ID, "final-db-invitee"))
	active, items, err := repo.GetPartners(ctx, "final-db-owner")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Len(t, items, 1)
	assert.Equal(t, "final-db-invitation", active.ID)
	assert.True(t, repo.IsActivePartnerLinkOwnedBy(ctx, created.ID, "final-db-owner"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, created.ID, "final-db-owner"))
	require.NoError(t, repo.RevokePartner(ctx, created.ID, "final-db-invitee"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, created.ID, "final-db-owner"))

	organization, err := repo.CreateOrganization(ctx, "final-db-org123456", "DB Org", "db-org", "ignored-code", "final-db-owner")
	require.NoError(t, err)
	joinedAt := now
	require.NoError(t, repo.CreateOrganizationMember(ctx, "final-db-member", organization.ID, "final-db-student", "member", "active", &joinedAt))
	byOwner, err := repo.GetOrganizationByUserID(ctx, "final-db-student")
	require.NoError(t, err)
	assert.Equal(t, organization.ID, byOwner.ID)
	organizationByCode, err := repo.GetOrganizationByGroupCode(ctx, "123456")
	require.NoError(t, err)
	assert.Equal(t, organization.ID, organizationByCode.ID)
	members, err := repo.ListOrganizationMembers(ctx, organization.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "DB Student", members[0].UserName)
	member, err := repo.GetOrganizationMember(ctx, organization.ID, "final-db-student")
	require.NoError(t, err)
	assert.Equal(t, "member", member.Role)
	require.NoError(t, repo.RemoveOrganizationMember(ctx, organization.ID, "final-db-student"))
	_, err = repo.GetOrganizationMember(ctx, organization.ID, "final-db-student")
	assert.Error(t, err)

	require.NoError(t, repo.CreateApprovalRequest(ctx, "final-db-approval", "final-db-student", "device", "link", "pause_protection", "reason", 15, now.Add(time.Hour)))
	pending, err := repo.CountPendingApprovalsForOrg(ctx, organization.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, pending)
}

func repositoryFinalOpenSQLite(t *testing.T) *ent.Client {
	t.Helper()
	databaseName := fmt.Sprintf("repository_final_accountability_org_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", databaseName))
	require.NoError(t, err)
	driver := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(driver))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func repositoryFinalTimePtr(value time.Time) *time.Time { return &value }

func repositoryFinalActiveMembership(t *testing.T, repo *Repository, studentID string) *model.AccountabilityMembership {
	t.Helper()
	item, err := repo.ActiveMembershipForStudent(t.Context(), studentID)
	require.NoError(t, err)
	return item
}

func repositoryFinalMembershipStatus(st *store.Store, membershipID string) string {
	for _, item := range st.Snapshot().AccountabilityMemberships {
		if item.ID == membershipID {
			return item.Status
		}
	}
	return ""
}

func repositoryFinalExitStatus(st *store.Store, requestID string) string {
	for _, item := range st.Snapshot().MembershipExitRequests {
		if item.ID == requestID {
			return item.Status
		}
	}
	return ""
}

func repositoryFinalContact(st *store.Store, requestID string) model.PartnerContactRequest {
	for _, item := range st.Snapshot().PartnerContactRequests {
		if item.ID == requestID {
			return item
		}
	}
	return model.PartnerContactRequest{}
}

func repositoryFinalContactStatus(st *store.Store, requestID string) string {
	return repositoryFinalContact(st, requestID).Status
}

func repositoryFinalPartnerStatus(st *store.Store, partnerID string) string {
	for _, item := range st.Snapshot().Partners {
		if item.ID == partnerID {
			return item.Status
		}
	}
	return ""
}

func repositoryFinalMustPartnerToken(t *testing.T, repo *Repository, ctx context.Context, token string) string {
	t.Helper()
	linkID, err := repo.GetPartnerLinkByToken(ctx, token)
	require.NoError(t, err)
	return linkID
}

func mustRepositoryFinalDBExit(t *testing.T, client *ent.Client, requestID string) string {
	t.Helper()
	row, err := client.MembershipExitRequest.Get(t.Context(), requestID)
	require.NoError(t, err)
	return row.Status.String()
}
