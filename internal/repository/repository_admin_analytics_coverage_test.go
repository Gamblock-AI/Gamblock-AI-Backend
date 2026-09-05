package repository

import (
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func adminAnalyticsCoverageStringPtr(value string) *string {
	return &value
}

func adminAnalyticsCoverageTime(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

func TestRepositoryAdminAnalyticsCoverage_AdminAccountsAndPhones(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := adminAnalyticsCoverageTime(2026, time.January, 10)

	st.Lock()
	st.Users = append(st.Users,
		model.User{ID: "usr_disabled_cov", Email: "disabled@example.com", DisplayName: "Disabled Admin", Role: "admin", PhoneE164: "+62000", DisabledAt: adminAnalyticsCoverageTimePtr(now), CreatedAt: now},
		model.User{ID: "usr_empty_phone_cov", Email: "empty@example.com", DisplayName: "Empty Phone", Role: "admin", CreatedAt: now},
		model.User{ID: "usr_unknown_role_cov", Email: "legacy@example.com", DisplayName: "Legacy Role", Role: "operator", PhoneE164: "+62999", CreatedAt: now},
	)
	st.Unlock()

	accounts, err := repo.ListAdminAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 6)
	accountIDs := make([]string, 0, len(accounts))
	for _, account := range accounts {
		accountIDs = append(accountIDs, account.ID)
	}
	assert.ElementsMatch(t, []string{"usr_gading", "usr_dery", "usr_suci", "usr_nasywa", "usr_disabled_cov", "usr_empty_phone_cov"}, accountIDs)

	page, err := repo.ListAdminAccountsPaginated(ctx, model.PaginationQuery{Role: "admin", Status: "disabled"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "usr_disabled_cov", page.Items[0].ID)

	page, err = repo.ListAdminAccountsPaginated(ctx, model.PaginationQuery{Status: "active", Query: "NASYWA"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "usr_nasywa", page.Items[0].ID)

	page, err = repo.ListAdminAccountsPaginated(ctx, model.PaginationQuery{Role: "user", Query: "gading@gmail.com", Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "usr_gading", page.Items[0].ID)

	page, err = repo.ListAdminAccountsPaginated(ctx, model.PaginationQuery{Page: 99, Limit: 2, Query: "does-not-exist"})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	assert.Equal(t, 0, page.TotalCount)

	phones, err := repo.ListActiveAdminPhones(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"+6282328514811"}, phones)

	assert.NoError(t, repo.SetAccountDisabled(ctx, "usr_gading", true, now))
	assert.NotNil(t, st.Snapshot().Users[0].DisabledAt)
	assert.NoError(t, repo.SetAccountDisabled(ctx, "usr_gading", false, now.Add(time.Hour)))
	assert.Nil(t, st.Snapshot().Users[0].DisabledAt)
	assert.EqualError(t, repo.SetAccountDisabled(ctx, "missing-account", true, now), "account not found")
}

func adminAnalyticsCoverageTimePtr(value time.Time) *time.Time {
	return &value
}

func TestRepositoryAdminAnalyticsCoverage_SocialAuditAndEducationQueries(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)
	urlA := "https://example.test/a"
	urlB := "https://example.test/b"

	require.NoError(t, repo.ReplaceSiteSocialLinks(ctx, "admin@example.com", []model.SiteSocialLink{
		{Platform: "zeta", Label: "Zeta", URL: &urlA, Enabled: true, SortOrder: 2},
		{Platform: "alpha", Label: "Alpha", URL: &urlB, Enabled: true, SortOrder: 1},
		{Platform: "hidden", Label: "Hidden", URL: &urlA, Enabled: false, SortOrder: 0},
		{Platform: "empty", Label: "Empty", Enabled: true, SortOrder: 3},
	}))

	publicLinks, err := repo.ListSiteSocialLinks(ctx, true)
	require.NoError(t, err)
	require.Len(t, publicLinks, 2)
	assert.Equal(t, "alpha", publicLinks[0].Platform)
	assert.Equal(t, "zeta", publicLinks[1].Platform)
	allLinks, err := repo.ListSiteSocialLinks(ctx, false)
	require.NoError(t, err)
	assert.Len(t, allLinks, 4)
	assert.Equal(t, "hidden", allLinks[0].Platform)

	oldAudit := model.AuditEvent{ID: "audit_cov_old", Actor: "old@example.com", Action: "old_action", Target: "target-old", Reason: "old reason", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)}
	newAudit := model.AuditEvent{ID: "audit_cov_new", Actor: "Admin@Example.com", Action: "account_disabled", Target: "usr_gading", Reason: "security review", Metadata: map[string]any{"scope": "admin"}, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)}
	require.NoError(t, repo.SaveAuditEvent(ctx, oldAudit))
	require.NoError(t, repo.SaveAuditEvent(ctx, newAudit))

	audits, err := repo.ListAuditEvents(ctx, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(audits), 3)
	assert.Equal(t, "audit_cov_new", audits[0].ID)
	audits, err = repo.ListAuditEvents(ctx, 201)
	require.NoError(t, err)
	assert.NotEmpty(t, audits)

	page, err := repo.ListAuditEventsPaginated(ctx, model.PaginationQuery{Action: "account_disabled", Actor: "ADMIN@EXAMPLE", Query: "usr_gading", Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "audit_cov_new", page.Items[0].ID)
	page, err = repo.ListAuditEventsPaginated(ctx, model.PaginationQuery{Page: 20, Limit: 1, Query: "nothing"})
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	require.NoError(t, repo.PurgeAuditEventsBefore(ctx, now.Add(-24*time.Hour)))
	audits, err = repo.ListAuditEvents(ctx, 100)
	require.NoError(t, err)
	for _, item := range audits {
		assert.False(t, item.CreatedAt.Before(now.Add(-24*time.Hour)))
	}

	first := model.EducationRevision{ID: "rev_cov_1", ModuleID: "module_cov", Revision: 1, Kind: "draft", CreatedBy: "usr_nasywa", CreatedAt: now.Add(-time.Hour)}
	second := model.EducationRevision{ID: "rev_cov_2", ModuleID: "module_cov", Revision: 2, Kind: "published", CreatedBy: "usr_nasywa", CreatedAt: now}
	require.NoError(t, repo.SaveEducationRevision(ctx, first))
	require.NoError(t, repo.SaveEducationRevision(ctx, first))
	require.NoError(t, repo.SaveEducationRevision(ctx, second))
	revisions, err := repo.ListEducationRevisions(ctx, "module_cov")
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	assert.Equal(t, "rev_cov_2", revisions[0].ID)
	found, err := repo.EducationRevisionByID(ctx, "module_cov", "rev_cov_1")
	require.NoError(t, err)
	assert.Equal(t, "draft", found.Kind)
	_, err = repo.EducationRevisionByID(ctx, "other-module", "rev_cov_1")
	assert.ErrorIs(t, err, ErrEducationNotFound)

	snapshot := st.Snapshot()
	assert.NotNil(t, snapshot.SiteSocialLinks)
}

func TestRepositoryAdminAnalyticsCoverage_AnalyticsEmptyAndFiltering(t *testing.T) {
	now := adminAnalyticsCoverageTime(2026, time.March, 10)
	st := &store.Store{
		AccountabilityGroups: []model.AccountabilityGroup{{ID: "grp_cov_analytics", OwnerPartnerID: "partner_cov", Status: "active"}},
		AccountabilityMemberships: []model.AccountabilityMembership{
			{ID: "member_cov_private", GroupID: "grp_cov_analytics", StudentID: "student_private", Status: "active", Sharing: model.SharingPreferences{ProtectionActivity: false}},
			{ID: "member_cov_left", GroupID: "grp_cov_analytics", StudentID: "student_left", Status: "left", Sharing: model.SharingPreferences{ProtectionActivity: true}},
		},
		AggregateEvents: []model.AggregateEvent{
			{UserID: "student_private", EventType: "unknown_event", EventDate: now, Count: 99},
			{UserID: "student_private", EventType: "block_count_sync", EventDate: now.AddDate(0, 0, -20), Count: 50},
		},
	}
	repo := New(nil, st)

	summary, err := repo.PartnerAnalytics(t.Context(), "partner_cov", "", 14, now)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.MemberCount)
	assert.Equal(t, 0, summary.SharedMemberCount)
	assert.Equal(t, "empty", summary.DataState)
	assert.Len(t, summary.Daily, 14)

	summary, err = repo.PartnerAnalytics(t.Context(), "partner_cov", "missing-group", 7, now)
	assert.Error(t, err)
	assert.Empty(t, summary.Daily)

	platform, err := repo.PlatformAnalytics(t.Context(), 7, now)
	require.NoError(t, err)
	assert.Equal(t, "empty", platform.DataState)
	assert.Equal(t, 0, platform.ProtectedUsers)

	events, err := repo.aggregateEventsForUsers(t.Context(), nil, 7, now)
	require.NoError(t, err)
	assert.Empty(t, events)
	events, err = repo.aggregateEventsForUsers(t.Context(), []string{"student_private"}, 14, now)
	require.NoError(t, err)
	assert.Empty(t, events)
	events, err = repo.allAggregateEvents(t.Context(), 14, now)
	require.NoError(t, err)
	assert.Empty(t, events)

	hourly := make([]model.AnalyticsHour, 24)
	for index := range hourly {
		hourly[index].Hour = index
	}
	addHourlyHistogram(hourly, map[string]any{"hourly": []any{1.0}})
	addHourlyHistogram(hourly, map[string]any{"hourly": make([]any, 24)})
	assert.Equal(t, 0, hourlyTotal(hourly))
	addHourlyHistogram(hourly, map[string]any{"hourly": adminAnalyticsCoverageHourlyValues(2, -1, 3)})
	assert.Equal(t, 5, hourlyTotal(hourly))
}

func adminAnalyticsCoverageHourlyValues(values ...float64) []any {
	hourly := make([]any, 24)
	for index := range values {
		hourly[index] = values[index]
	}
	return hourly
}

func TestRepositoryAdminAnalyticsCoverage_OrganizationsAndMembers(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := adminAnalyticsCoverageTime(2026, time.April, 10)

	organization, err := repo.GetOrganizationByID(ctx, "org_uty")
	require.NoError(t, err)
	assert.Equal(t, "Universitas Teknologi Yogyakarta", organization.Name)
	_, err = repo.GetOrganizationByID(ctx, "missing-org")
	assert.EqualError(t, err, "organisasi tidak ditemukan")

	st.Lock()
	st.Organizations = append(st.Organizations, model.Organization{ID: "org_cov_owner", Name: "Owned", CreatedBy: "usr_gading", GroupCode: "COV123", CreatedAt: now, UpdatedAt: now})
	st.Unlock()
	owned, err := repo.GetOrganizationByUserID(ctx, "usr_gading")
	require.NoError(t, err)
	assert.Equal(t, "org_cov_owner", owned.ID)
	_, err = repo.GetOrganizationByUserID(ctx, "usr_dery")
	assert.EqualError(t, err, "tidak ada grup")

	byCode, err := repo.GetOrganizationByGroupCode(ctx, "COV123")
	require.NoError(t, err)
	assert.Equal(t, "org_cov_owner", byCode.ID)

	members, err := repo.ListOrganizationMembers(ctx, "org_cov_owner")
	require.NoError(t, err)
	assert.Len(t, members, len(st.Snapshot().Users))
	assert.Equal(t, "member", members[0].Role)
	require.NoError(t, repo.RemoveOrganizationMember(ctx, "org_cov_owner", "usr_gading"))
	_, err = repo.GetOrganizationMember(ctx, "org_cov_owner", "usr_gading")
	assert.EqualError(t, err, "not found")

	assert.Equal(t, "abc", idToGroupCode("abc"))
	assert.Equal(t, "345678", idToGroupCode("12345678"))
	pending, err := repo.CountPendingApprovalsForOrg(ctx, "org_cov_owner")
	require.NoError(t, err)
	assert.Equal(t, 2, pending)
}

func TestRepositoryAdminAnalyticsCoverage_PartnerOwnershipAndInvitations(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	phone := "+628111111111"

	active, items, err := repo.GetPartners(ctx, "usr_gading")
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, "owner", active.RelationshipRole)
	assert.Len(t, items, 1)
	partnerView, items, err := repo.GetPartners(ctx, "usr_suci")
	require.NoError(t, err)
	require.NotNil(t, partnerView)
	assert.Equal(t, "partner", partnerView.RelationshipRole)
	assert.Len(t, items, 1)
	emptyActive, emptyItems, err := repo.GetPartners(ctx, "no-partner")
	require.NoError(t, err)
	assert.Nil(t, emptyActive)
	assert.Empty(t, emptyItems)

	created, err := repo.CreatePartnerInvitation(ctx, "pl_cov_invited", "usr_gading", "suci@gmail.com", &phone, "token-cov-invited")
	require.NoError(t, err)
	assert.Equal(t, "invited", created.Status)
	require.NoError(t, repo.AcceptPartnerInvitation(ctx, "pl_cov_invited", "usr_suci"))
	assert.Equal(t, "active", findPartnerForCoverage(repo, "pl_cov_invited").Status)
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "pl_cov_invited", "usr_suci"), "invitation not found")
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "missing-link", "usr_suci"), "invitation not found")

	_, err = repo.CreatePartnerInvitation(ctx, "pl_cov_bad_email", "usr_gading", "other@example.com", nil, "token-cov-bad")
	require.NoError(t, err)
	assert.EqualError(t, repo.AcceptPartnerInvitation(ctx, "pl_cov_bad_email", "usr_suci"), "invitation not found")
	assert.EqualError(t, repo.RevokePartner(ctx, "missing-link", "usr_gading"), "partner link not found")
	require.NoError(t, repo.RevokePartner(ctx, "pl_cov_invited", "usr_suci"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "pl_cov_invited", "usr_gading"))

	valid, err := repo.CreatePartnerInvitation(ctx, "pl_cov_token", "usr_gading", "token@example.com", nil, "token-cov-valid")
	require.NoError(t, err)
	assert.Equal(t, "pl_cov_token", valid.ID)
	linkID, err := repo.GetPartnerLinkByToken(ctx, "token-cov-valid")
	require.NoError(t, err)
	assert.Equal(t, "pl_cov_token", linkID)
	_, err = repo.GetPartnerLinkByToken(ctx, "missing-token")
	assert.EqualError(t, err, "invitation not found")

	st := repo.store
	st.Lock()
	st.Partners = append(st.Partners,
		model.Partner{ID: "pl_cov_stale", UserID: "usr_gading", InviteTokenHash: "token-cov-stale", Status: "invited", CreatedAt: time.Now().UTC().Add(-8 * 24 * time.Hour)},
		model.Partner{ID: "pl_cov_phone", UserID: "usr_gading", PartnerUserID: "usr_suci", Contact: "Suci", Status: "active"},
	)
	st.Unlock()
	_, err = repo.GetPartnerLinkByToken(ctx, "token-cov-stale")
	assert.EqualError(t, err, "invitation not found")
	assert.True(t, repo.IsActivePartnerLinkOwnedBy(ctx, "pl_active", "usr_gading"))
	assert.False(t, repo.IsActivePartnerLinkOwnedBy(ctx, "pl_active", "usr_suci"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "pl_cov_phone", "usr_gading"))
	assert.Empty(t, repo.GetActivePartnerPhone(ctx, "missing-link", "usr_gading"))

	assert.Equal(t, "owner", partnerForUser(model.Partner{UserID: "usr_gading", Name: "Owner"}, "usr_gading").RelationshipRole)
	assert.Equal(t, "partner", partnerForUser(model.Partner{UserID: "usr_suci", Name: "Partner"}, "usr_gading").RelationshipRole)
}

func findPartnerForCoverage(repo *Repository, id string) model.Partner {
	for _, item := range repo.store.Snapshot().Partners {
		if item.ID == id {
			return item
		}
	}
	return model.Partner{}
}

func TestRepositoryAdminAnalyticsCoverage_SupportAdminFiltersAndEmptyStates(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := adminAnalyticsCoverageTime(2026, time.May, 10)

	require.NoError(t, repo.CreateSupportCase(ctx, "case_cov_admin", "usr_nasywa", "Admin's own case", "account", "low"))
	st.Lock()
	st.SupportCases = append(st.SupportCases,
		model.SupportCase{ID: "case_cov_active", UserID: "usr_gading", Title: "Active high case", Type: "device", Status: "waiting_support", Priority: "high", CreatedAt: now, UpdatedAt: now},
		model.SupportCase{ID: "case_cov_resolved", UserID: "usr_dery", Title: "Resolved history", Type: "account", Status: "resolved", Priority: "normal", Owner: "usr_nasywa", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
		model.SupportCase{ID: "case_cov_closed", UserID: "usr_suci", Title: "Closed other", Type: "account", Status: "closed", Priority: "low", Owner: "usr_suci", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
	)
	st.Unlock()

	page, err := repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Status: "waiting_support", Priority: "high"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "case_cov_active", page.Items[0].ID)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Bucket: "active", Assignee: "unassigned"})
	require.NoError(t, err)
	assert.NotEmpty(t, page.Items)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Bucket: "history", Assignee: "me"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "case_cov_resolved", page.Items[0].ID)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Bucket: "history", Assignee: "others"})
	require.NoError(t, err)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, "case_cov_closed", page.Items[0].ID)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Query: "case_cov_closed"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "case_cov_closed", page.Items[0].ID)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Page: 99, Limit: 1})
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	cases, err := repo.GetSupportCasesForUser(ctx, "does-not-exist")
	require.NoError(t, err)
	assert.Empty(t, cases)
	messages, err := repo.ListSupportMessages(ctx, "case-without-messages")
	require.NoError(t, err)
	assert.Empty(t, messages)
}

func TestRepositoryAdminAnalyticsCoverage_SupportMessagesAndDataRequests(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := adminAnalyticsCoverageTime(2026, time.June, 10)

	unknownCase := model.SupportCase{ID: "case_cov_unknown_user", UserID: "unknown-user", Title: "Unknown user", Type: "account", Status: "waiting_support", Priority: "normal", CreatedAt: now, UpdatedAt: now}
	_, err := repo.CreateSupportCaseWithMessage(ctx, unknownCase, "encrypted-request")
	require.NoError(t, err)
	_, err = repo.AddSupportMessage(ctx, model.SupportMessage{ID: "msg_cov_unknown", SupportCaseID: unknownCase.ID, AuthorID: "unknown-author", AuthorRole: "requester", Content: "encrypted-follow-up", CreatedAt: now.Add(time.Minute)}, "waiting_user")
	require.NoError(t, err)
	detail, err := repo.GetSupportCaseDetail(ctx, unknownCase.ID)
	require.NoError(t, err)
	assert.Empty(t, detail.UserName)
	require.Len(t, detail.Messages, 2)
	assert.Empty(t, detail.Messages[1].AuthorName)
	assert.Equal(t, "waiting_user", detail.Status)

	blank, err := repo.AddSupportMessage(ctx, model.SupportMessage{ID: "msg_cov_blank", SupportCaseID: unknownCase.ID, Content: "encrypted-blank", CreatedAt: now.Add(2 * time.Minute)}, "waiting_support")
	require.NoError(t, err)
	assert.Equal(t, "msg_cov_blank", blank.ID)

	activeExpiry := now.Add(time.Hour)
	completed := now.Add(-time.Hour)
	require.NoError(t, repo.CreateDataRequestRecord(ctx, model.DataRequest{ID: "dr_cov_full", UserID: "usr_gading", Type: "export", Status: "processing", ConfirmationTokenHash: "hash-full", ConfirmationExpiresAt: &activeExpiry, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, repo.CreateDataRequestRecord(ctx, model.DataRequest{ID: "dr_cov_empty", UserID: "usr_dery", Type: "delete", Status: "completed", CompletedAt: &completed, CreatedAt: completed, UpdatedAt: completed}))

	all, err := repo.GetAllDataRequests(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 4)
	page, err := repo.GetAllDataRequestsPaginated(ctx, model.PaginationQuery{Bucket: "active", Status: "not-a-status", Type: "delete"})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	page, err = repo.GetAllDataRequestsPaginated(ctx, model.PaginationQuery{Page: 50, Limit: 1})
	require.NoError(t, err)
	assert.Empty(t, page.Items)

	request, err := repo.DataRequestByID(ctx, "dr_cov_full")
	require.NoError(t, err)
	confirmed := now.Add(2 * time.Minute)
	resultExpiry := now.Add(2 * time.Hour)
	request.Status = "failed"
	request.ConfirmationTokenHash = ""
	request.ConfirmationExpiresAt = nil
	request.ConfirmedAt = &confirmed
	request.ResultPath = "exports/full.zip"
	request.ResultExpiresAt = &resultExpiry
	request.FailureCode = "temporary_failure"
	request.RetryCount = 2
	request.CompletedAt = &confirmed
	require.NoError(t, repo.UpdateDataRequest(ctx, request))
	updated, err := repo.DataRequestByID(ctx, "dr_cov_full")
	require.NoError(t, err)
	assert.Equal(t, "failed", updated.Status)
	assert.Equal(t, "temporary_failure", updated.FailureCode)
	assert.Equal(t, 2, updated.RetryCount)
	assert.Equal(t, "exports/full.zip", updated.ResultPath)
	assert.EqualError(t, repo.UpdateDataRequest(ctx, model.DataRequest{ID: "missing-data-request"}), "data request not found")
	_, err = repo.DataRequestByConfirmationToken(ctx, "hash-full", now)
	assert.EqualError(t, err, "confirmation token is invalid or expired")

	st.Lock()
	st.DataRequests = nil
	st.Unlock()
	empty, err := repo.GetDataRequests(ctx, "usr_gading")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestRepositoryAdminAnalyticsCoverage_DashboardAndPresentationQueries(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)

	_, _, _, err := repo.GetDashboardData(ctx, "missing-user", now)
	require.NoError(t, err)
	summary, protection, progress, err := repo.GetDashboardData(ctx, "usr_gading", now)
	require.NoError(t, err)
	assert.Equal(t, "Gading", summary.UserName)
	assert.Equal(t, "active", protection.Mode)
	assert.Greater(t, summary.BlockedAttempts, 0)
	assert.Len(t, progress.WeeklyBlocks, 7)

	st.Lock()
	st.CheckIns = append(st.CheckIns,
		model.CheckIn{ID: "check_cov_1", UserID: "usr_gading", Mood: 3, Urge: 2, CreatedAt: now.Add(-24 * time.Hour)},
		model.CheckIn{ID: "check_cov_2", UserID: "usr_gading", Mood: 4, Urge: 1, CreatedAt: now},
		model.CheckIn{ID: "check_cov_old", UserID: "usr_gading", Mood: 1, Urge: 1, CreatedAt: now.Add(-20 * 24 * time.Hour)},
	)
	st.RecoveryRecords = append(st.RecoveryRecords, model.RecoveryRecord{ID: "review_cov", UserID: "usr_gading", Kind: "weekly_review", RecordDate: now.Format("2006-01-02"), CreatedAt: now, UpdatedAt: now})
	st.Unlock()

	progress, err = repo.GetProgressData(ctx, "usr_gading", 30, now)
	require.NoError(t, err)
	assert.Equal(t, 30, progress.RangeDays)
	assert.Len(t, progress.WeeklyBlocks, 7)
	assert.GreaterOrEqual(t, progress.CheckInCount, 2)
	assert.True(t, progress.TrendAvailable)
	assert.NotEmpty(t, progress.ActivityDays)
	_, err = repo.GetProgressData(ctx, "usr_gading", 8, now)
	assert.EqualError(t, err, "progress range must be 7, 30, or 90 days")

	links := []model.Partner{
		{ID: "pl_cov_active", PartnerUserID: "usr_suci", Status: "active"},
		{ID: "pl_cov_revoked", PartnerUserID: "usr_suci", Status: "revoked"},
		{ID: "pl_cov_other", PartnerUserID: "other", Status: "active"},
	}
	activeLinks := activePartnerLinkIDs(links, "usr_suci")
	assert.Equal(t, map[string]struct{}{"pl_cov_active": {}}, activeLinks)
	assert.Equal(t, "Pause protection for 15 minutes", approvalActionLabel("pause_protection", 15))
	assert.Equal(t, "Allow protected app removal", approvalActionLabel("uninstall_detected", 0))
	assert.Equal(t, "custom_action", approvalActionLabel("custom_action", 0))
	assert.Equal(t, "Pending partner approval", approvalStatusLabel("pending"))
	assert.Equal(t, "custom_status", approvalStatusLabel("custom_status"))

	assert.NotNil(t, adminAnalyticsCoverageStringPtr("coverage"))
}
