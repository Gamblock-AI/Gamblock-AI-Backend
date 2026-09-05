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

func TestRepositoryCoverageUserOrgSupport_UserCreationAndUpdates(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()

	withPhone, err := repo.CreateUserWithPasswordAndPhone(
		ctx, "usr_user_org_support_phone", "Phone@Example.test", "Phone User", "+628111222333",
		"hashed-phone", "partner",
	)
	require.NoError(t, err)
	assert.Equal(t, "partner", withPhone.Role)
	assert.Equal(t, "+628111222333", withPhone.PhoneE164)
	snapshot := st.Snapshot()
	assert.Equal(t, "hashed-phone", snapshot.Users[len(snapshot.Users)-1].PasswordHash)

	withoutPhone, err := repo.CreateProvisionedUser(
		ctx, "usr_user_org_support_no_phone", "NoPhone@Example.test", "No Phone User",
		"hashed-no-phone", "user", true,
	)
	require.NoError(t, err)
	assert.True(t, withoutPhone.MustChangePassword)
	assert.Empty(t, withoutPhone.PhoneE164)

	got, ok := repo.UserByEmail(ctx, "PHONE@EXAMPLE.TEST")
	require.True(t, ok)
	assert.Equal(t, withPhone.ID, got.ID)
	_, ok = repo.UserByEmail(ctx, "missing-user-org-support@example.test")
	assert.False(t, ok)

	updated, err := repo.UpdateUserDisplayName(ctx, withPhone.ID, "Renamed Phone User")
	require.NoError(t, err)
	assert.Equal(t, "Renamed Phone User", updated.DisplayName)
	require.NoError(t, repo.UpdateUserPasswordHash(ctx, withPhone.ID, "hashed-updated"))

	avatarKey := "avatar/user-org-support.webp"
	_, err = repo.UpdateUserAvatar(ctx, withPhone.ID, &avatarKey)
	require.NoError(t, err)
	storedKey, ok := repo.UserAvatarStorageKey(ctx, withPhone.ID)
	require.True(t, ok)
	assert.Equal(t, avatarKey, storedKey)
	_, err = repo.UpdateUserAvatar(ctx, withPhone.ID, nil)
	require.NoError(t, err)
	_, ok = repo.UserAvatarStorageKey(ctx, withPhone.ID)
	assert.False(t, ok)

	_, err = repo.UpdateUserDisplayName(ctx, "missing-user-org-support", "ignored")
	assert.EqualError(t, err, "user not found")
	_, err = repo.UpdateUserAvatar(ctx, "missing-user-org-support", nil)
	assert.EqualError(t, err, "user not found")
	assert.EqualError(t, repo.UpdateUserPasswordHash(ctx, "missing-user-org-support", "ignored"), "user not found")
}

func TestRepositoryCoverageUserOrgSupport_OrganizationOwnershipAndEmptyStates(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	created, err := repo.CreateOrganization(ctx, "org_cov_user_org_support", "Coverage Organization", "coverage-org", "COVORG", "usr_suci")
	require.NoError(t, err)
	assert.Equal(t, "usr_suci", created.CreatedBy)

	byID, err := repo.GetOrganizationByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Name, byID.Name)
	byOwner, err := repo.GetOrganizationByUserID(ctx, "usr_suci")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byOwner.ID)
	byCode, err := repo.GetOrganizationByGroupCode(ctx, "COVORG")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byCode.ID)

	_, err = repo.GetOrganizationByUserID(ctx, "usr_without_organization")
	assert.EqualError(t, err, "tidak ada grup")
	_, err = repo.GetOrganizationByID(ctx, "missing-organization")
	assert.EqualError(t, err, "organisasi tidak ditemukan")
	_, err = repo.GetOrganizationByGroupCode(ctx, "missing-code")
	assert.EqualError(t, err, "kode grup tidak valid")

	joinedAt := time.Date(2026, time.September, 5, 10, 0, 0, 0, time.UTC)
	assert.NoError(t, repo.CreateOrganizationMember(ctx, "member-cov-user-org-support", created.ID, "usr_gading", "viewer", "active", &joinedAt))
	members, err := repo.ListOrganizationMembers(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, members, 4)
	assert.Equal(t, "member", members[0].Role)
	assert.Equal(t, "active", members[0].Status)
	assert.NoError(t, repo.RemoveOrganizationMember(ctx, created.ID, "usr_gading"))
	_, err = repo.GetOrganizationMember(ctx, created.ID, "usr_gading")
	assert.EqualError(t, err, "not found")

	assert.Equal(t, 2, mustCountPendingApprovalsCoverage(t, repo, ctx, created.ID))
	summary, err := repo.GetMemberProgressSummary(ctx, "member-progress-user-org-support")
	require.NoError(t, err)
	assert.Equal(t, 1, summary.ActiveDevices)
	assert.Equal(t, 5+len("member-progress-user-org-support")%20, summary.BlockedAttempts)
	assert.Equal(t, 2+len("member-progress-user-org-support")%3, summary.CompletedMissions)

	emptyRepo := New(nil, store.New())
	emptyMembers, err := emptyRepo.ListOrganizationMembers(ctx, "empty-org")
	require.NoError(t, err)
	assert.Empty(t, emptyMembers)
	_, err = emptyRepo.GetOrganizationByUserID(ctx, "nobody")
	assert.EqualError(t, err, "tidak ada grup")
	_, err = emptyRepo.GetOrganizationByGroupCode(ctx, "none")
	assert.EqualError(t, err, "kode grup tidak valid")
	assert.NoError(t, emptyRepo.CreateOrganizationMember(ctx, "ignored-member", "empty-org", "nobody", "member", "active", nil))
	assert.NoError(t, emptyRepo.RemoveOrganizationMember(ctx, "empty-org", "nobody"))
}

func mustCountPendingApprovalsCoverage(t *testing.T, repo *Repository, ctx context.Context, organizationID string) int {
	t.Helper()
	count, err := repo.CountPendingApprovalsForOrg(ctx, organizationID)
	require.NoError(t, err)
	return count
}

func TestRepositoryCoverageUserOrgSupport_PartnerOwnershipAndPhoneStates(t *testing.T) {
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	st := store.New()
	st.Partners = []model.Partner{
		{ID: "partner-cov-owner", UserID: "usr_owner_cov", PartnerUserID: "usr_partner_cov", Name: "Partner", Contact: "owner@example.test | +628111", PartnerEmail: "partner@example.test", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "partner-cov-invited", UserID: "usr_owner_cov", PartnerUserID: "usr_partner_cov", InviteTokenHash: "token-cov-owner", Name: "Invited", Status: "invited", CreatedAt: now, UpdatedAt: now},
		{ID: "partner-cov-expired", UserID: "usr_owner_cov", InviteTokenHash: "token-cov-expired", Status: "invited", CreatedAt: now.Add(-8 * 24 * time.Hour), UpdatedAt: now},
		{ID: "partner-cov-no-phone", UserID: "usr_owner_cov", PartnerUserID: "usr_partner_cov", Contact: "email-only@example.test", Status: "active", CreatedAt: now, UpdatedAt: now},
	}
	repo := New(nil, st)
	ctx := context.Background()

	active, items, err := repo.GetPartners(ctx, "usr_owner_cov")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, "owner", active.RelationshipRole)
	assert.Len(t, items, 4)

	partnerView, partnerItems, err := repo.GetPartners(ctx, "usr_partner_cov")
	require.NoError(t, err)
	require.NotNil(t, partnerView)
	assert.Equal(t, "partner", partnerView.RelationshipRole)
	assert.Len(t, partnerItems, 3)

	noActive, noItems, err := repo.GetPartners(ctx, "usr_unrelated_cov")
	require.NoError(t, err)
	assert.Nil(t, noActive)
	assert.Empty(t, noItems)

	linkID, err := repo.GetPartnerLinkByToken(ctx, "token-cov-owner")
	require.NoError(t, err)
	assert.Equal(t, "partner-cov-invited", linkID)
	_, err = repo.GetPartnerLinkByToken(ctx, "token-cov-expired")
	assert.EqualError(t, err, "invitation not found")
	_, err = repo.GetPartnerLinkByToken(ctx, "missing-token-cov")
	assert.EqualError(t, err, "invitation not found")

	assert.True(t, repo.IsActivePartnerLinkOwnedBy(ctx, "partner-cov-owner", "usr_owner_cov"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "partner-cov-owner", "usr_partner_cov"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "partner-cov-invited", "usr_owner_cov"))
	assert.Equal(t, "+628111", repo.GetActivePartnerPhone(ctx, "partner-cov-owner", "usr_owner_cov"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "partner-cov-no-phone", "usr_owner_cov"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "partner-cov-owner", "usr_partner_cov"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "missing-partner-cov", "usr_owner_cov"))
}

func TestRepositoryCoverageUserOrgSupport_SupportOwnershipAndTransitions(t *testing.T) {
	repo, st := newRepo(t)
	ctx := context.Background()
	now := time.Date(2026, time.September, 5, 13, 0, 0, 0, time.UTC)

	st.Lock()
	st.SupportCases = append(st.SupportCases,
		model.SupportCase{ID: "case-cov-closed", UserID: "usr_gading", Title: "Closed case", Type: "account", Status: "closed", Priority: "low", Owner: "usr_nasywa", CreatedAt: now, UpdatedAt: now},
		model.SupportCase{ID: "case-cov-resolved", UserID: "usr_dery", Title: "Resolved case", Type: "device", Status: "resolved", Priority: "normal", Owner: "usr_nasywa", CreatedAt: now, UpdatedAt: now},
	)
	st.Unlock()

	all, err := repo.GetSupportCases(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 4)
	owned, err := repo.GetSupportCasesForUser(ctx, "usr_gading")
	require.NoError(t, err)
	for _, item := range owned {
		assert.Equal(t, "usr_gading", item.UserID)
	}
	emptyOwned, err := repo.GetSupportCasesForUser(ctx, "nobody-support-cov")
	require.NoError(t, err)
	assert.Empty(t, emptyOwned)

	_, err = repo.ClaimSupportCase(ctx, "case-cov-closed", "usr_nasywa", "retry", now)
	assert.EqualError(t, err, "support case is already assigned or closed")
	_, err = repo.ClaimSupportCase(ctx, "case-cov-resolved", "usr_nasywa", "retry", now)
	assert.EqualError(t, err, "support case is already assigned or closed")

	message := model.SupportMessage{
		ID: "message-cov-missing-case", SupportCaseID: "missing-case-support-cov", AuthorID: "unknown-author-cov",
		AuthorRole: "requester", Content: "encrypted", CreatedAt: now,
	}
	added, err := repo.AddSupportMessage(ctx, message, "waiting_user")
	require.NoError(t, err)
	assert.Equal(t, message.ID, added.ID)
	caseDetail, err := repo.GetSupportCaseDetail(ctx, "case-cov-closed")
	require.NoError(t, err)
	assert.Empty(t, caseDetail.Messages)
	unknownMessages, err := repo.ListSupportMessages(ctx, "missing-case-support-cov")
	require.NoError(t, err)
	require.Len(t, unknownMessages, 1)
	assert.Empty(t, unknownMessages[0].AuthorName)

	transitionAt := now.Add(time.Minute)
	require.NoError(t, repo.TransitionSupportCase(ctx, "case-cov-closed", "waiting_support", "", transitionAt))
	caseDetail, err = repo.GetSupportCaseDetail(ctx, "case-cov-closed")
	require.NoError(t, err)
	assert.Equal(t, "waiting_support", caseDetail.Status)
	assert.Empty(t, caseDetail.Owner)

	emptyRepo := New(nil, store.New())
	page, err := emptyRepo.GetAdminSupportCasesPaginated(ctx, "admin-cov", model.PaginationQuery{Page: 4, Limit: 2})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Equal(t, 0, page.TotalCount)
	assert.Empty(t, mustGetSupportCasesCoverage(t, emptyRepo, ctx))
	assert.Empty(t, mustGetSupportCasesForUserCoverage(t, emptyRepo, ctx, "nobody"))
	assert.Empty(t, mustGetSupportMessagesCoverage(t, emptyRepo, ctx, "missing"))
	_, err = emptyRepo.GetSupportCaseDetail(ctx, "missing")
	assert.EqualError(t, err, "support case not found")
	assert.EqualError(t, emptyRepo.ReleaseSupportCase(ctx, "missing", "admin-cov", "release", now), "support case is not assigned to operator")
}

func mustGetSupportCasesCoverage(t *testing.T, repo *Repository, ctx context.Context) []model.SupportCase {
	t.Helper()
	items, err := repo.GetSupportCases(ctx)
	require.NoError(t, err)
	return items
}

func mustGetSupportCasesForUserCoverage(t *testing.T, repo *Repository, ctx context.Context, userID string) []model.SupportCase {
	t.Helper()
	items, err := repo.GetSupportCasesForUser(ctx, userID)
	require.NoError(t, err)
	return items
}

func mustGetSupportMessagesCoverage(t *testing.T, repo *Repository, ctx context.Context, caseID string) []model.SupportMessage {
	t.Helper()
	items, err := repo.ListSupportMessages(ctx, caseID)
	require.NoError(t, err)
	return items
}

func TestRepositoryCoverageUserOrgSupport_DataRequestsNotificationsRemindersAndPush(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.September, 5, 14, 0, 0, 0, time.UTC)
	emptyRepo := New(nil, store.New())

	all, err := emptyRepo.GetAllDataRequests(ctx)
	require.NoError(t, err)
	assert.Empty(t, all)
	byUser, err := emptyRepo.GetDataRequests(ctx, "nobody")
	require.NoError(t, err)
	assert.Empty(t, byUser)
	page, err := emptyRepo.GetAllDataRequestsPaginated(ctx, model.PaginationQuery{Bucket: "history", Page: 3, Limit: 2})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	_, err = emptyRepo.DataRequestByID(ctx, "missing-data-request-cov")
	assert.EqualError(t, err, "data request not found")
	_, err = emptyRepo.DataRequestByConfirmationToken(ctx, "missing-confirmation-cov", now)
	assert.EqualError(t, err, "confirmation token is invalid or expired")
	assert.EqualError(t, emptyRepo.UpdateDataRequest(ctx, model.DataRequest{ID: "missing-data-request-cov"}), "data request not found")

	activeUntil := now.Add(time.Hour)
	require.NoError(t, emptyRepo.CreateDataRequestRecord(ctx, model.DataRequest{
		ID: "data-request-cov", UserID: "usr_gading", Type: "export", Status: "pending",
		ConfirmationTokenHash: "confirmation-cov", ConfirmationExpiresAt: &activeUntil, CreatedAt: now, UpdatedAt: now,
	}))
	confirmed, err := emptyRepo.DataRequestByConfirmationToken(ctx, "confirmation-cov", now)
	require.NoError(t, err)
	assert.Equal(t, "data-request-cov", confirmed.ID)
	updated := confirmed
	updated.Status = "completed"
	updated.ConfirmationTokenHash = ""
	updated.ConfirmationExpiresAt = nil
	updated.ResultPath = "exports/data-request-cov.zip"
	updated.ResultExpiresAt = &activeUntil
	updated.FailureCode = ""
	updated.ConfirmedAt = &now
	updated.CompletedAt = &now
	require.NoError(t, emptyRepo.UpdateDataRequest(ctx, updated))
	updatedStored, err := emptyRepo.DataRequestByID(ctx, updated.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updatedStored.Status)
	assert.Equal(t, updated.ResultPath, updatedStored.ResultPath)

	require.NoError(t, emptyRepo.QueueNotification(ctx, "notification-cov", "approval-cov", "support-cov", "whatsapp", "+628111"))

	preference, err := emptyRepo.ReminderPreference(ctx, "reminder-cov")
	require.NoError(t, err)
	assert.False(t, preference.Enabled)
	assert.Equal(t, defaultReminderTime, preference.LocalTime)
	preference, err = emptyRepo.UpsertReminderPreference(ctx, "reminder-cov", true, "07:30", "Asia/Jakarta", "id")
	require.NoError(t, err)
	assert.True(t, preference.Enabled)
	require.NoError(t, emptyRepo.MarkReminderFired(ctx, "reminder-cov", now))
	preference, err = emptyRepo.ReminderPreference(ctx, "reminder-cov")
	require.NoError(t, err)
	require.NotNil(t, preference.LastFiredAt)
	updatedPreference, err := emptyRepo.UpsertReminderPreference(ctx, "reminder-cov", false, "21:00", "UTC", "en")
	require.NoError(t, err)
	assert.False(t, updatedPreference.Enabled)
	assert.Empty(t, mustEnabledRemindersCoverage(t, emptyRepo, ctx))
	require.NoError(t, emptyRepo.MarkReminderFired(ctx, "missing-reminder-cov", now))

	userAgent := "coverage-user-agent"
	subscription, err := emptyRepo.UpsertPushSubscription(ctx, "usr_gading", "https://push.cov/endpoint", "p256dh-cov", "auth-cov", &userAgent)
	require.NoError(t, err)
	assert.Equal(t, "usr_gading", subscription.UserID)
	updatedSubscription, err := emptyRepo.UpsertPushSubscription(ctx, "usr_dery", subscription.Endpoint, "p256dh-updated", "auth-updated", nil)
	require.NoError(t, err)
	assert.Equal(t, subscription.ID, updatedSubscription.ID)
	assert.Nil(t, updatedSubscription.UserAgent)

	owned, err := emptyRepo.PushSubscriptionsForUser(ctx, "usr_dery")
	require.NoError(t, err)
	require.Len(t, owned, 1)
	assert.Empty(t, mustPushSubscriptionsCoverage(t, emptyRepo, ctx, "nobody"))
	require.NoError(t, emptyRepo.DeletePushSubscription(ctx, "usr_gading", subscription.Endpoint))
	owned, err = emptyRepo.PushSubscriptionsForUser(ctx, "usr_dery")
	require.NoError(t, err)
	assert.Len(t, owned, 1)
	require.NoError(t, emptyRepo.RemovePushSubscriptionByID(ctx, subscription.ID))
	assert.Empty(t, mustPushSubscriptionsCoverage(t, emptyRepo, ctx, "usr_dery"))
	require.NoError(t, emptyRepo.DeletePushSubscription(ctx, "nobody", "missing-endpoint-cov"))
	require.NoError(t, emptyRepo.RemovePushSubscriptionByID(ctx, "missing-subscription-cov"))
}

func mustEnabledRemindersCoverage(t *testing.T, repo *Repository, ctx context.Context) []store.ReminderPreference {
	t.Helper()
	items, err := repo.EnabledReminderPreferences(ctx)
	require.NoError(t, err)
	return items
}

func mustPushSubscriptionsCoverage(t *testing.T, repo *Repository, ctx context.Context, userID string) []model.PushSubscription {
	t.Helper()
	items, err := repo.PushSubscriptionsForUser(ctx, userID)
	require.NoError(t, err)
	return items
}
