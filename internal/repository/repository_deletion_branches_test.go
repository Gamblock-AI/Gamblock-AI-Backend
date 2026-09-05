package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteUserAccountData_InMemoryRemovesAndAnonymizesOwnedRecords(t *testing.T) {
	repo, st := newRepo(t)
	ctx := t.Context()
	now := time.Now().UTC()
	st.DataRequests = append(st.DataRequests, model.DataRequest{ID: "dr_delete_cov", UserID: "usr_gading", Type: "delete", Status: "processing", ResultPath: "exports/private.zip"})
	st.AuditEvents = append(st.AuditEvents, model.AuditEvent{ID: "audit_delete_cov", ActorID: "usr_gading", Actor: "Gading", CreatedAt: now, UpdatedAt: now})

	assert.EqualError(t, repo.DeleteUserAccountData(ctx, "missing-user", now), "user not found")
	require.NoError(t, repo.DeleteUserAccountData(ctx, "usr_gading", now))

	_, ok := repo.UserByID(ctx, "usr_gading")
	assert.False(t, ok)
	snapshot := st.Snapshot()
	for _, item := range snapshot.ContactVerifications {
		assert.NotEqual(t, "usr_gading", item.UserID)
	}
	for _, item := range snapshot.Devices {
		assert.NotEqual(t, "usr_gading", item.UserID)
	}
	for _, item := range snapshot.Partners {
		assert.NotEqual(t, "usr_gading", item.UserID)
		assert.NotEqual(t, "usr_gading", item.PartnerUserID)
	}
	for _, item := range snapshot.AccountabilityMemberships {
		assert.NotEqual(t, "usr_gading", item.StudentID)
	}
	for _, item := range snapshot.JournalEntries {
		assert.NotEqual(t, "usr_gading", item.UserID)
	}
	for _, item := range snapshot.AggregateEvents {
		assert.NotEqual(t, "usr_gading", item.UserID)
	}

	var anonymized model.DataRequest
	for _, item := range snapshot.DataRequests {
		if item.ID == "dr_delete_cov" {
			anonymized = item
		}
	}
	require.NotEmpty(t, anonymized.ID)
	assert.True(t, strings.HasPrefix(anonymized.UserID, "deleted:"))
	assert.Equal(t, "completed", anonymized.Status)
	assert.Empty(t, anonymized.ResultPath)

	var audit model.AuditEvent
	for _, item := range snapshot.AuditEvents {
		if item.ID == "audit_delete_cov" {
			audit = item
		}
	}
	assert.Equal(t, "deleted-account", audit.Actor)
	assert.True(t, strings.HasPrefix(audit.ActorID, "deleted:"))
}
