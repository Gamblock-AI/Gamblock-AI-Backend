package repository

import (
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportRepository_InMemoryFiltersClaimMessagesAndTransitions(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	require.NoError(t, repo.CreateSupportCase(ctx, "case_cov_unassigned", "usr_gading", "Unassigned coverage case", "device_recovery", "high"))
	_, createErr := repo.CreateSupportCaseWithMessage(ctx, model.SupportCase{
		ID: "case_cov_message", UserID: "usr_dery", Title: "Message coverage case", Type: "account", Status: "waiting_support", Priority: "normal", Impact: "blocked", CreatedAt: now, UpdatedAt: now,
	}, "encrypted initial detail")
	require.NoError(t, createErr)

	all, err := repo.GetSupportCases(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), 4)
	userCases, err := repo.GetSupportCasesForUser(ctx, "usr_dery")
	require.NoError(t, err)
	assert.Len(t, userCases, 2)

	page, err := repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Bucket: "active", Priority: "high"})
	require.NoError(t, err)
	assert.Equal(t, 1, page.TotalCount)
	assert.Equal(t, "case_cov_unassigned", page.Items[0].ID)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Query: "message coverage", Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, page.TotalCount)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Assignee: "unassigned", Limit: 100})
	require.NoError(t, err)
	assert.NotEmpty(t, page.Items)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Assignee: "me"})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Assignee: "others"})
	require.NoError(t, err)
	assert.NotNil(t, page.Items)
	page, err = repo.GetAdminSupportCasesPaginated(ctx, "usr_nasywa", model.PaginationQuery{Bucket: "history"})
	require.NoError(t, err)
	assert.NotNil(t, page.Items)

	claimed, err := repo.ClaimSupportCase(ctx, "case_cov_unassigned", "usr_nasywa", "take ownership", now)
	require.NoError(t, err)
	assert.Equal(t, "usr_nasywa", claimed.Owner)
	assert.EqualError(t, func() error {
		_, err := repo.ClaimSupportCase(ctx, "case_cov_unassigned", "usr_suci", "other", now)
		return err
	}(), "support case is already assigned or closed")
	require.NoError(t, repo.ReleaseSupportCase(ctx, "case_cov_unassigned", "usr_nasywa", "handoff", now.Add(time.Minute)))
	assert.EqualError(t, repo.ReleaseSupportCase(ctx, "case_cov_unassigned", "usr_suci", "wrong", now), "support case is not assigned to operator")
	assert.EqualError(t, func() error {
		_, err := repo.ClaimSupportCase(ctx, "missing-case", "usr_nasywa", "missing", now)
		return err
	}(), "support case not found")

	detail, err := repo.GetSupportCaseDetail(ctx, "case_cov_message")
	require.NoError(t, err)
	assert.Equal(t, "Dery", detail.UserName)
	require.Len(t, detail.Messages, 1)
	assert.Equal(t, "encrypted initial detail", detail.Messages[0].Content)
	_, err = repo.GetSupportCaseDetail(ctx, "missing-case")
	assert.EqualError(t, err, "support case not found")

	adminMessage, err := repo.AddSupportMessage(ctx, model.SupportMessage{ID: "msg_cov_admin", SupportCaseID: "case_cov_message", AuthorID: "usr_nasywa", AuthorRole: "admin", Content: "encrypted response", CreatedAt: now.Add(time.Minute)}, "waiting_user")
	require.NoError(t, err)
	assert.Equal(t, "msg_cov_admin", adminMessage.ID)
	detail, err = repo.GetSupportCaseDetail(ctx, "case_cov_message")
	require.NoError(t, err)
	assert.Equal(t, 1, detail.UnreadCount)
	assert.Equal(t, "Nasywa", detail.Messages[1].AuthorName)
	assert.Equal(t, "waiting_user", detail.Status)

	require.NoError(t, repo.TransitionSupportCase(ctx, "case_cov_message", "resolved", "usr_nasywa", now.Add(2*time.Minute)))
	detail, err = repo.GetSupportCaseDetail(ctx, "case_cov_message")
	require.NoError(t, err)
	assert.Equal(t, "resolved", detail.Status)
	assert.NotNil(t, detail.ResolvedAt)
	require.NoError(t, repo.TransitionSupportCase(ctx, "case_cov_message", "closed", "usr_nasywa", now.Add(3*time.Minute)))
	detail, err = repo.GetSupportCaseDetail(ctx, "case_cov_message")
	require.NoError(t, err)
	assert.NotNil(t, detail.ClosedAt)
	assert.EqualError(t, repo.TransitionSupportCase(ctx, "missing-case", "closed", "usr_nasywa", now), "support case not found")

	messages, err := repo.ListSupportMessages(ctx, "case_cov_message")
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestDataRequestRepository_InMemoryFiltersUpdatesAndConfirmation(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()

	activeExpiry := now.Add(time.Hour)
	historyCompleted := now.Add(-time.Hour)
	require.NoError(t, repo.CreateDataRequestRecord(ctx, model.DataRequest{ID: "dr_cov_active", UserID: "usr_gading", Type: "delete", Status: "processing", ConfirmationTokenHash: "token-cov", ConfirmationExpiresAt: &activeExpiry, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, repo.CreateDataRequestRecord(ctx, model.DataRequest{ID: "dr_cov_history", UserID: "usr_gading", Type: "export", Status: "completed", CompletedAt: &historyCompleted, CreatedAt: historyCompleted, UpdatedAt: historyCompleted}))

	owned, err := repo.GetDataRequests(ctx, "usr_gading")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(owned), 3)
	all, err := repo.GetAllDataRequests(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all), len(owned))
	page, err := repo.GetAllDataRequestsPaginated(ctx, model.PaginationQuery{Bucket: "active", Type: "delete", Limit: 100})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, page.TotalCount, 1)
	page, err = repo.GetAllDataRequestsPaginated(ctx, model.PaginationQuery{Bucket: "history", Status: "completed", Limit: 100})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, page.TotalCount, 2)

	request, err := repo.DataRequestByID(ctx, "dr_cov_active")
	require.NoError(t, err)
	assert.Equal(t, "delete", request.Type)
	_, err = repo.DataRequestByID(ctx, "missing-request")
	assert.EqualError(t, err, "data request not found")
	confirmed, err := repo.DataRequestByConfirmationToken(ctx, "token-cov", now)
	require.NoError(t, err)
	assert.Equal(t, "dr_cov_active", confirmed.ID)
	_, err = repo.DataRequestByConfirmationToken(ctx, "token-cov", now.Add(2*time.Hour))
	assert.EqualError(t, err, "confirmation token is invalid or expired")
	_, err = repo.DataRequestByConfirmationToken(ctx, "missing-token", now)
	assert.EqualError(t, err, "confirmation token is invalid or expired")

	request.Status = "completed"
	request.ResultPath = "exports/result.zip"
	request.ConfirmationTokenHash = ""
	request.ConfirmationExpiresAt = nil
	request.CompletedAt = &now
	require.NoError(t, repo.UpdateDataRequest(ctx, request))
	updated, err := repo.DataRequestByID(ctx, request.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, "exports/result.zip", updated.ResultPath)
	assert.EqualError(t, repo.UpdateDataRequest(ctx, model.DataRequest{ID: "missing-request"}), "data request not found")

	require.NoError(t, repo.CreateDataRequest(ctx, "dr_cov_legacy", "usr_dery", "delete"))
	legacy, err := repo.DataRequestByID(ctx, "dr_cov_legacy")
	require.NoError(t, err)
	assert.Equal(t, "Delete archived support notes", legacy.Title)
}
