package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestClientServiceAggregateValidationAndPrivacyBoundaries(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewClientService(repo, testCfg())

	for _, tc := range []struct {
		name    string
		date    string
		event   string
		key     string
		count   int
		wantErr string
	}{
		{"bad date", "not-a-date", "block_count_sync", "idempotent", 1, "invalid aggregate event"},
		{"bad event", "2026-09-05", "raw_url", "idempotent", 1, "invalid aggregate event"},
		{"negative count", "2026-09-05", "block_count_sync", "idempotent", -1, "invalid aggregate event"},
		{"short idempotency key", "2026-09-05", "block_count_sync", "short", 1, "invalid aggregate event"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RecordAggregate(ctx, "usr_gading", "dev_android", tc.event, tc.date, tc.key, tc.count, false, nil)
			require.EqualError(t, err, tc.wantErr)
		})
	}

	date := time.Now().UTC().Format("2006-01-02")
	hourly := make([]any, 24)
	for i := range hourly {
		hourly[i] = float64(0)
	}
	hourly[2] = float64(2)
	event, err := svc.RecordAggregate(ctx, "usr_gading", "dev_android", "block_count_sync", date, "valid-key", 3, false, map[string]any{"hourly": hourly})
	require.NoError(t, err)
	assert.Equal(t, 3, event.Count)

	_, err = svc.RecordAggregate(ctx, "usr_gading", "dev_android", "block_count_sync", date, "another-key", 1, false, map[string]any{"unexpected": true})
	require.EqualError(t, err, "metadata contains no hourly histogram")
	_, err = svc.RecordAggregate(ctx, "usr_gading", "dev_android", "block_count_sync", date, "another-key", 1, false, map[string]any{"hourly": []any{1}})
	require.EqualError(t, err, "hourly histogram must contain 24 values")
	hourly[3] = 0.5
	_, err = svc.RecordAggregate(ctx, "usr_gading", "dev_android", "block_count_sync", date, "another-key", 3, false, map[string]any{"hourly": hourly})
	require.EqualError(t, err, "hourly histogram values must be non-negative integers")
	hourly[3] = float64(2)
	_, err = svc.RecordAggregate(ctx, "usr_gading", "dev_android", "block_count_sync", date, "another-key", 3, false, map[string]any{"hourly": hourly})
	require.EqualError(t, err, "hourly histogram total exceeds the event count")

	_, err = svc.RecordAggregate(ctx, "usr_gading", "dev_android", "block_count_sync", date, "future-key", 1, true, nil)
	if time.Now().UTC().Format("2006-01-02") == date {
		require.NoError(t, err)
	}
	_, err = svc.RecordAggregate(ctx, "usr_gading", "dev_windows", "block_count_sync", date, "owned-by-other", 1, false, nil)
	require.NoError(t, err)
	_, err = svc.RecordAggregate(ctx, "usr_dery", "dev_android", "block_count_sync", date, "wrong-owner", 1, false, nil)
	require.EqualError(t, err, "device does not belong to user")
}

func TestClientServiceBlockedEventsProfileAndAnalytics(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewClientService(repo, testCfg())

	require.NoError(t, svc.SaveBlockedEvents(ctx, "usr_gading", "dev_android", nil))
	require.EqualError(t, svc.SaveBlockedEvents(ctx, "usr_dery", "dev_android", []time.Time{time.Now()}), "device does not belong to user")
	require.EqualError(t, svc.SaveBlockedEvents(ctx, "usr_gading", "dev_android", []time.Time{time.Now().UTC().Add(48 * time.Hour)}), "blocked event timestamp outside accepted window")
	many := make([]time.Time, 501)
	require.EqualError(t, svc.SaveBlockedEvents(ctx, "usr_gading", "dev_android", many), "blocked event batch exceeds 500 timestamps")
	require.NoError(t, svc.SaveBlockedEvents(ctx, "usr_gading", "dev_android", []time.Time{time.Now().UTC(), time.Now().UTC().Add(-time.Hour)}))

	_, err := svc.GetProfile(ctx, "missing")
	require.EqualError(t, err, "user not found")
	updated, err := svc.UpdateProfile(ctx, "usr_gading", "  Nama Baru  ")
	require.NoError(t, err)
	assert.Equal(t, "Nama Baru", updated.DisplayName)
	_, err = svc.UpdateProfile(ctx, "usr_gading", "")
	require.EqualError(t, err, "display name must contain 1-80 characters")
	_, err = svc.UpdateProfile(ctx, "usr_gading", strings.Repeat("x", 81))
	require.EqualError(t, err, "display name must contain 1-80 characters")

	_, err = svc.ProtectionAnalytics(ctx, "usr_gading", "dev_android", 14)
	require.EqualError(t, err, "period must be 7 or 30 days")
	_, err = svc.ProtectionAnalytics(ctx, "usr_gading", "", 7)
	require.EqualError(t, err, "device id is required")
	analytics, err := svc.ProtectionAnalytics(ctx, "usr_gading", "dev_android", 7)
	require.NoError(t, err)
	assert.Equal(t, "dev_android", analytics.DeviceID)
	_, _, _, err = svc.Dashboard(ctx, "usr_gading")
	require.NoError(t, err)
	_, err = svc.Progress(ctx, "usr_gading", 30)
	require.NoError(t, err)
}

func TestDeviceServiceOwnershipDefaultsAndHeartbeats(t *testing.T) {
	ctx := context.Background()
	repo, _ := newRepo(t)
	svc := NewDeviceService(repo, testCfg(), zap.NewNop())

	_, err := svc.CreateDevice(ctx, "usr_gading", "", "", "", "", "", nil, nil)
	require.EqualError(t, err, "client instance id is required")
	created, err := svc.CreateDevice(ctx, "usr_gading", "client-new", "", "", "", "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "android", created.Platform)
	assert.Equal(t, "Protected device", created.Label)
	assert.Equal(t, "1.0.0", created.AppVersion)
	assert.Equal(t, "Unknown OS", created.OSVersion)

	updated, err := svc.UpdateOwnedDevice(ctx, "usr_gading", created.ID, "Laptop", "2.0", "Linux", "active", "m", "r")
	require.NoError(t, err)
	assert.Equal(t, "Laptop", updated.Label)
	_, err = svc.UpdateOwnedDevice(ctx, "usr_dery", created.ID, "bad", "", "", "", "", "")
	require.Error(t, err)
	require.NoError(t, svc.RecordHeartbeat(ctx, created.ID))
	require.NoError(t, svc.RecordOwnedHeartbeat(ctx, "usr_gading", created.ID))
	require.Error(t, svc.RecordOwnedHeartbeat(ctx, "usr_dery", created.ID))

	_, err = svc.IssueGrantKeyChallenge(ctx, "usr_dery", created.ID)
	require.EqualError(t, err, "device does not belong to user")
	challenge, err := svc.IssueGrantKeyChallenge(ctx, "usr_gading", created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, challenge.ChallengeToken)
	_, err = svc.BindGrantKey(ctx, "usr_gading", created.ID, "bad", nil, "bad")
	require.Error(t, err)
}

func TestParseDeviceGrantPublicJWKValidation(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	encode := func(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }
	valid := func() json.RawMessage {
		x, y := key.PublicKey.X.Bytes(), key.PublicKey.Y.Bytes()
		xPadded, yPadded := make([]byte, 32), make([]byte, 32)
		copy(xPadded[32-len(x):], x)
		copy(yPadded[32-len(y):], y)
		payload, _ := json.Marshal(map[string]string{"kty": "EC", "crv": "P-256", "x": encode(xPadded), "y": encode(yPadded)})
		return payload
	}

	_, _, thumbprint, err := parseDeviceGrantPublicJWK(valid())
	require.NoError(t, err)
	assert.NotEmpty(t, thumbprint)
	for _, raw := range []json.RawMessage{nil, []byte(`{"kty":"RSA"}`), []byte(`{"kty":"EC","crv":"P-256","x":"bad","y":"bad"}`), append(valid(), []byte(` {}`)...)} {
		_, _, _, err := parseDeviceGrantPublicJWK(raw)
		require.Error(t, err)
	}
}

func TestRecoveryServiceValidationPersistenceAndWeeklyReview(t *testing.T) {
	ctx := context.Background()
	plainRepo := repository.New(nil, store.New())
	svc := NewRecoveryService(plainRepo)
	_, err := svc.GetActiveIntention(ctx, "usr_missing")
	require.NoError(t, err)
	_, err = svc.SaveIntention(ctx, "usr_1", "  ", "", model.Intention{})
	require.EqualError(t, err, "intention is required")
	_, err = svc.SaveIntention(ctx, "usr_1", "goal", "invalid", model.Intention{})
	require.EqualError(t, err, "invalid intention status")
	intention, err := svc.SaveIntention(ctx, "usr_1", "  goal  ", "", model.Intention{QuitMotivation: "ready"})
	require.NoError(t, err)
	assert.Equal(t, "goal", intention.Text)
	_, err = svc.CreateCheckIn(ctx, "usr_1", 0, 1, "")
	require.Error(t, err)
	checkin, err := svc.CreateCheckIn(ctx, "usr_1", 4, 0, "context")
	require.NoError(t, err)
	assert.Equal(t, 4, checkin.Mood)

	full := NewRecoveryServiceWithConfig(plainRepo, testCfg())
	_, err = full.SaveRecoveryRecord(ctx, "usr_1", "", "bad", "2026-01-01", "x", "", nil)
	require.Error(t, err)
	_, err = full.SaveRecoveryRecord(ctx, "usr_1", "", "roadmap", "bad", "x", "", nil)
	require.Error(t, err)
	_, err = full.SaveRecoveryRecord(ctx, "usr_1", "", "roadmap", "2026-01-01", strings.Repeat("x", 8001), "", nil)
	require.Error(t, err)
	record, err := full.SaveRecoveryRecord(ctx, "usr_1", "rec_1", "roadmap", "2026-01-01", "encrypted content", "", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "encrypted content", record.Content)
	items, err := full.GetRecoveryRecords(ctx, "usr_1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "encrypted content", items[0].Content)

	legacy := model.WeeklyReview{WhatHelped: []string{"walk"}, WhatWasHard: []string{"urge"}, Adjustment: "continue", NextMission: "mission_1", RecommendedSkill: "grounding"}
	result, err := full.SaveCurrentWeeklyReviewWithResult(ctx, "usr_1", legacy)
	require.NoError(t, err)
	assert.True(t, result.EXPGranted)
	loaded, err := full.GetCurrentWeeklyReview(ctx, "usr_1")
	require.NoError(t, err)
	assert.Equal(t, "mission_1", loaded.NextMission)

	normalized := model.WeeklyReview{IntentionID: "int_1", Outcome: "helped", HelpfulAction: "trusted_person", Adjustment: "continue", NextMissionNumber: 2, SelectedSkillID: "skill_1"}
	result, err = full.SaveCurrentWeeklyReviewWithResult(ctx, "usr_1", normalized)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Review.ID)
	_, err = full.SaveCurrentWeeklyReview(ctx, "usr_1", model.WeeklyReview{Outcome: "invalid", HelpfulAction: "bad", Adjustment: "bad", NextMissionNumber: 9, SelectedSkillID: "x"})
	require.Error(t, err)
}

func TestRecoveryServiceSpaceAndNilPersistence(t *testing.T) {
	ctx := context.Background()
	nilSvc := NewRecoveryService(repository.New(nil, store.New()))
	practices, err := nilSvc.GetRecoveryPractices(ctx, "usr_1")
	require.NoError(t, err)
	assert.Empty(t, practices)
	records, err := nilSvc.GetRecoveryRecords(ctx, "usr_1")
	require.NoError(t, err)
	assert.Empty(t, records)

	repo, st := newRepo(t)
	full := NewRecoveryServiceWithConfig(repo, testCfg())
	space, err := full.GetRecoverySpace(ctx, "usr_gading")
	require.NoError(t, err)
	assert.Equal(t, "dorm_room", space.Theme)
	assert.Equal(t, recoveryUnlockRuleVersion, space.UnlockRuleVersion)
	st.Lock()
	st.RecoverySpaces[0].UnlockedItems = append(st.RecoverySpaces[0].UnlockedItems, "manual_item")
	st.Unlock()
	space, err = full.GetRecoverySpace(ctx, "usr_gading")
	require.NoError(t, err)
	assert.Contains(t, space.UnlockedItems, "manual_item")
}

func TestReflectionServiceDailyJournalAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := repository.New(nil, store.New())
	svc := NewReflectionService(repo, testCfg(), zap.NewNop())
	doc := map[string]any{"type": "doc", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Today"}}}}}
	for _, invalid := range []map[string]any{{}, {"type": "doc"}, {"type": "doc", "content": []any{map[string]any{"type": "heading", "attrs": map[string]any{"level": float64(1)}}}}} {
		_, err := svc.UpsertDailyJournal(ctx, "usr_1", invalid)
		require.Error(t, err)
	}
	entry, err := svc.UpsertDailyJournal(ctx, "usr_1", doc)
	require.NoError(t, err)
	assert.Equal(t, 3, entry.PayloadVersion)
	loaded, err := svc.GetDailyJournal(ctx, "usr_1")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "Today", loaded.Document["content"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["text"])
	doc["content"] = []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Updated"}}}}
	updated, err := svc.UpsertDailyJournal(ctx, "usr_1", doc)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, updated.ID)
	journals, err := svc.GetDailyJournals(ctx, "usr_1")
	require.NoError(t, err)
	assert.Len(t, journals, 1)

	mood := 4
	reflection, err := svc.CreateReflectionEntry(ctx, "usr_1", "hello", &mood, "next", true)
	require.NoError(t, err)
	assert.Equal(t, 2, reflection.PayloadVersion)
	badMood := 6
	_, err = svc.UpdateReflection(ctx, "usr_1", reflection.ID, model.ReflectionUpdate{MoodScore: &badMood})
	require.EqualError(t, err, "reflection mood is invalid")
	status := "archived"
	next := "updated next"
	changed, err := svc.UpdateReflection(ctx, "usr_1", reflection.ID, model.ReflectionUpdate{Status: &status, NextStep: &next})
	require.NoError(t, err)
	assert.Equal(t, "archived", changed.Status)
	assert.Equal(t, "updated next", changed.NextStep)
	_, err = svc.UpdateReflection(ctx, "usr_1", "missing", model.ReflectionUpdate{})
	require.EqualError(t, err, "reflection not found")
}

func TestValidateJournalDocumentImageRules(t *testing.T) {
	validImage := map[string]any{"type": "doc", "content": []any{map[string]any{"type": "image", "attrs": map[string]any{"src": "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("img"))}}}}
	require.NoError(t, validateJournalDocument(validImage))
	_, err := json.Marshal(validImage)
	require.NoError(t, err)
	invalidImage := map[string]any{"type": "doc", "content": []any{map[string]any{"type": "image", "attrs": map[string]any{"src": "https://example.com/x.png"}}}}
	require.EqualError(t, validateJournalDocument(invalidImage), "journal image is invalid")
	tooMany := map[string]any{"type": "doc", "content": []any{}}
	for i := 0; i < 6; i++ {
		tooMany["content"] = append(tooMany["content"].([]any), validImage["content"].([]any)[0])
	}
	require.EqualError(t, validateJournalDocument(tooMany), "too many journal images")
}

func TestReflectionServiceFailsClosedWithoutKey(t *testing.T) {
	repo := repository.New(nil, store.New())
	svc := NewReflectionService(repo, testCfg(), zap.NewNop())
	svc.encryptMode = false
	_, err := svc.CreateReflection(context.Background(), "usr_1", "plaintext", "")
	require.EqualError(t, err, "encryption is required but not configured")
	_, err = svc.UpsertDailyJournal(context.Background(), "usr_1", map[string]any{"type": "doc"})
	require.EqualError(t, err, "encryption is required but not configured")
}

func TestClientServiceSnapshotRejectsNonCurrentDate(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewClientService(repo, testCfg())
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	_, err := svc.RecordAggregate(context.Background(), "usr_gading", "dev_android", "block_count_sync", yesterday, "snapshot-key", 1, true, nil)
	require.EqualError(t, err, "aggregate snapshot must be for the current UTC date")
}

func TestClientServiceUsesRawHourlyTypesOnly(t *testing.T) {
	values := make([]any, 24)
	values[0] = 1
	require.EqualError(t, validateHourlyMetadata(map[string]any{"hourly": values}, 1), "hourly histogram values must be non-negative integers")
	values[0] = float64(-1)
	require.EqualError(t, validateHourlyMetadata(map[string]any{"hourly": values}, 1), "hourly histogram values must be non-negative integers")
}
