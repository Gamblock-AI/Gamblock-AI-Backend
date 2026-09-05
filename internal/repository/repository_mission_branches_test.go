package repository

import (
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissionRepository_InMemoryClaimabilityAndExperience(t *testing.T) {
	now := time.Now().UTC()
	dayStart := now.Add(-time.Hour)
	dayEnd := now.Add(time.Hour)
	completedAt := now
	st := &store.Store{
		Users:             []model.User{{ID: "student", ExperiencePoints: 20}},
		Devices:           []model.Device{{UserID: "student", ProtectionStatus: "active", LastSeenAt: now}},
		CheckIns:          []model.CheckIn{{UserID: "student", CreatedAt: now}},
		EducationProgress: []model.EducationProgress{{UserID: "student", CompletedSectionIDs: []string{"section"}, UpdatedAt: now, CompletedAt: &completedAt}},
		Partners:          []model.Partner{{UserID: "student", Status: "active"}},
	}
	repo := New(nil, st)
	ctx := t.Context()

	for missionNumber := 1; missionNumber <= 5; missionNumber++ {
		claimable, err := repo.IsMissionClaimable(ctx, "student", missionNumber, dayStart, dayEnd)
		require.NoError(t, err)
		assert.True(t, claimable, "mission %d should be claimable", missionNumber)
	}
	claimable, err := repo.IsMissionClaimable(ctx, "student", 6, dayStart, dayEnd)
	require.NoError(t, err)
	assert.False(t, claimable)

	mission, points, err := repo.GetMissionByDate(ctx, "student", "2026-09-05", dayStart, dayEnd)
	require.NoError(t, err)
	assert.Equal(t, "day_2026-09-05", mission.ID)
	assert.Equal(t, 20, points)
	assert.Empty(t, mission.TaskRecords)

	points, err = repo.AddUserExperiencePoints(ctx, "student", -50)
	require.NoError(t, err)
	assert.Zero(t, points)
	assert.EqualError(t, func() error { _, err := repo.AddUserExperiencePoints(ctx, "missing", 5); return err }(), "user missing not found")

	for missionNumber := 1; missionNumber <= 6; missionNumber++ {
		var daily model.DailyMission
		daily, _, err = repo.UpsertMission(ctx, "student", "2026-09-05", dayStart, dayEnd, missionNumber, true, 10)
		require.NoError(t, err)
		assert.True(t, missionCompleted(daily, missionNumber))
	}
	assert.Equal(t, 60, repoUserExperience(t, repo, "student"))
	for missionNumber := 1; missionNumber <= 6; missionNumber++ {
		assert.True(t, missionCompleted(model.DailyMission{Mission1: missionNumber == 1, Mission2: missionNumber == 2, Mission3: missionNumber == 3, Mission4: missionNumber == 4, Mission5: missionNumber == 5, Mission6: missionNumber == 6}, missionNumber))
	}
	assert.False(t, missionCompleted(model.DailyMission{}, 99))
	for missionNumber := 1; missionNumber <= 6; missionNumber++ {
		mission := model.DailyMission{}
		setMissionFlag(&mission, missionNumber, true)
		assert.True(t, missionCompleted(mission, missionNumber))
	}
	assert.Equal(t, 3, missionNumber("mission_3"))
	assert.Zero(t, missionNumber("not-a-number"))
}

func TestMissionRepository_InMemoryCustomMissionLifecycle(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := t.Context()
	date := "2026-09-05"
	now := time.Now().UTC()

	created, points, err := repo.CreateCustomMission(ctx, "usr_gading", date, now.Add(-time.Hour), now.Add(time.Hour), "encrypted title", 7)
	require.NoError(t, err)
	assert.Equal(t, 20, points)
	require.Len(t, created.TaskRecords, 1)
	customID := created.TaskRecords[0].ID
	assert.Equal(t, "custom", created.TaskRecords[0].Source)

	updated, _, err := repo.UpdateCustomMission(ctx, "usr_gading", date, customID, "updated encrypted title")
	require.NoError(t, err)
	assert.Equal(t, "updated encrypted title", updated.TaskRecords[0].TitleEncrypted)
	_, _, err = repo.UpdateCustomMission(ctx, "usr_gading", date, "missing-custom", "missing")
	assert.ErrorIs(t, err, ErrCustomMissionNotFound)

	completed, points, err := repo.CompleteCustomMission(ctx, "usr_gading", date, customID, 7)
	require.NoError(t, err)
	assert.Equal(t, "completed", completed.TaskRecords[0].Status)
	assert.Equal(t, 27, points)
	completedAgain, points, err := repo.CompleteCustomMission(ctx, "usr_gading", date, customID, 99)
	require.NoError(t, err)
	assert.Equal(t, "completed", completedAgain.TaskRecords[0].Status)
	assert.Equal(t, 27, points)
	_, _, err = repo.UpdateCustomMission(ctx, "usr_gading", date, customID, "cannot update")
	assert.ErrorIs(t, err, ErrCustomMissionResolved)
	_, _, err = repo.DeleteCustomMission(ctx, "usr_gading", date, customID)
	assert.ErrorIs(t, err, ErrCustomMissionResolved)

	pending, _, err := repo.CreateCustomMission(ctx, "usr_gading", date, now.Add(-time.Hour), now.Add(time.Hour), "pending", 5)
	require.NoError(t, err)
	pendingID := pending.TaskRecords[len(pending.TaskRecords)-1].ID
	deleted, _, err := repo.DeleteCustomMission(ctx, "usr_gading", date, pendingID)
	require.NoError(t, err)
	for _, record := range deleted.TaskRecords {
		assert.NotEqual(t, pendingID, record.ID)
	}
	_, _, err = repo.DeleteCustomMission(ctx, "usr_gading", date, "missing-custom")
	assert.ErrorIs(t, err, ErrCustomMissionNotFound)
	_, _, err = repo.CompleteCustomMission(ctx, "usr_gading", date, "missing-custom", 5)
	assert.ErrorIs(t, err, ErrCustomMissionNotFound)
}
