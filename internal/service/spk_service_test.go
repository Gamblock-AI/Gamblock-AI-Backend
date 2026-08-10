package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newSpkTestService(st *store.Store) *SpkService {
	return NewSpkService(repository.New(nil, st), config.Config{}, zap.NewNop(), nil)
}

// A seeded, active user produces a rule-based recommendation with a feature
// mapping and a persisted daily record (idempotent within the day).
func TestSpkService_Recommend_SeededUser(t *testing.T) {
	svc := newSpkTestService(store.NewSeeded())

	first, err := svc.Recommend(context.Background(), "usr_gading")
	require.NoError(t, err)

	assert.Equal(t, "MEDIUM", first.SupportLevel)
	assert.Equal(t, "MEDIUM", first.EngagementLevel)
	assert.True(t, first.InterventionNeeded)
	assert.NotEmpty(t, first.RecommendationID)
	assert.Equal(t, model.SpkDataSufficient, first.DataState)
	assert.Equal(t, "recovery_practice", first.Feature.FeatureID)
	assert.Equal(t, "/recovery", first.Feature.Route)
	assert.False(t, first.LLMUsed)

	reasonKeys := make([]string, 0, len(first.Reason.Factors))
	for _, factor := range first.Reason.Factors {
		reasonKeys = append(reasonKeys, factor.Key)
	}
	assert.Contains(t, reasonKeys, "blocked_active_days_7d")
	assert.Contains(t, reasonKeys, "recovery_streak_days")
	assert.Contains(t, reasonKeys, "learning_activities_7d", "seeded learning must be available")
	assert.Contains(t, reasonKeys, "change_readiness", "seeded intention quiz must be available")
	assert.Empty(t, first.DataGaps, "seeded user has enough data for the recommendation")

	second, err := svc.Recommend(context.Background(), "usr_gading")
	require.NoError(t, err)
	assert.Equal(t, first.RecommendationID, second.RecommendationID, "daily record must be idempotent")
}

// A user with only block aggregates still gets a valid recommendation.
func TestSpkService_Recommend_PartialAggregateOnly(t *testing.T) {
	svc := newSpkTestService(store.NewSeeded())

	recommendation, err := svc.Recommend(context.Background(), "usr_dery")
	require.NoError(t, err)

	assert.Equal(t, "MEDIUM", recommendation.SupportLevel)
	assert.True(t, recommendation.InterventionNeeded)
	assert.NotEmpty(t, recommendation.Feature.FeatureID)
	assert.NotEmpty(t, recommendation.UnavailableFields)
}

// A brand-new user with no data must not break: the engine re-normalizes,
// reports insufficient data, and falls back to a light recommendation.
func TestSpkService_Recommend_NewUserInsufficientData(t *testing.T) {
	st := store.NewSeeded()
	st.Users = append(st.Users, store.User{ID: "usr_new", Email: "new@example.com", DisplayName: "New", Role: "user"})
	svc := newSpkTestService(st)

	recommendation, err := svc.Recommend(context.Background(), "usr_new")
	require.NoError(t, err)

	assert.Equal(t, model.SpkDataInsufficient, recommendation.DataState)
	assert.Equal(t, "LOW", recommendation.SupportLevel)
	assert.Equal(t, "MEDIUM", recommendation.EngagementLevel)
	assert.True(t, recommendation.InterventionNeeded)
	assert.Equal(t, "education", recommendation.Feature.FeatureID)
	assert.NotEmpty(t, recommendation.UnavailableFields)

	gapActions := make([]string, 0, len(recommendation.DataGaps))
	for _, gap := range recommendation.DataGaps {
		gapActions = append(gapActions, gap.Action)
	}
	assert.Contains(t, gapActions, "learn")
	assert.Contains(t, gapActions, "check_in")
	assert.Contains(t, gapActions, "set_intention")
}

// Marking a recommendation completed flips its persisted status.
func TestSpkService_MarkCompleted(t *testing.T) {
	svc := newSpkTestService(store.NewSeeded())

	recommendation, err := svc.Recommend(context.Background(), "usr_gading")
	require.NoError(t, err)

	record, err := svc.MarkCompleted(context.Background(), "usr_gading", recommendation.RecommendationID)
	require.NoError(t, err)
	assert.Equal(t, "completed", record.Status)
	require.NotNil(t, record.CompletedAt)

	_, err = svc.MarkCompleted(context.Background(), "usr_gading", "spk_missing")
	require.Error(t, err)
}

// The SPK privacy set defaults to all-on (master, data categories, and LLM)
// and persists the full set.
func TestSpkService_Preference(t *testing.T) {
	svc := newSpkTestService(store.NewSeeded())

	pref, err := svc.GetPreference(context.Background(), "usr_gading")
	require.NoError(t, err)
	assert.True(t, pref.SpkRecommendationEnabled)
	assert.True(t, pref.SpkUseProtection)
	assert.True(t, pref.SpkUseRecovery)
	assert.True(t, pref.SpkUsePersonal)
	assert.True(t, pref.LLMPersonalizationEnabled)

	updated, err := svc.UpdatePreference(context.Background(), "usr_gading", model.SpkPreference{
		SpkRecommendationEnabled:  true,
		SpkUseProtection:          false,
		SpkUseRecovery:            true,
		SpkUsePersonal:            false,
		LLMPersonalizationEnabled: false,
	})
	require.NoError(t, err)
	assert.False(t, updated.SpkUseProtection)
	assert.False(t, updated.SpkUsePersonal)
	assert.False(t, updated.LLMPersonalizationEnabled)

	reloaded, err := svc.GetPreference(context.Background(), "usr_gading")
	require.NoError(t, err)
	assert.False(t, reloaded.SpkUseProtection)
	assert.False(t, reloaded.LLMPersonalizationEnabled)
}

// Disabling a data category removes those factors from the engine input, which
// changes the available weight without breaking the recommendation.
func TestSpkService_Recommend_PrivacyGatesProtection(t *testing.T) {
	svc := newSpkTestService(store.NewSeeded())
	ctx := context.Background()

	_, err := svc.UpdatePreference(ctx, "usr_gading", model.SpkPreference{
		SpkRecommendationEnabled: true,
		SpkUseProtection:         false,
		SpkUseRecovery:           true,
		SpkUsePersonal:           true,
		LLMPersonalizationEnabled: false,
	})
	require.NoError(t, err)

	recommendation, err := svc.Recommend(ctx, "usr_gading")
	require.NoError(t, err)
	assert.True(t, recommendation.RecommendationEnabled)
	assert.InDelta(t, float64(60), recommendation.AvailableWeightPercent, 0.01, "protection weight must be excluded from 100")
	assert.Contains(t, recommendation.UnavailableFields, "blocked_attempts_today")
}

// Turning off the master switch returns a disabled recommendation that never
// uses the student's data and creates no daily record.
func TestSpkService_Recommend_Disabled(t *testing.T) {
	svc := newSpkTestService(store.NewSeeded())
	ctx := context.Background()

	_, err := svc.UpdatePreference(ctx, "usr_gading", model.SpkPreference{
		SpkRecommendationEnabled:  false,
		SpkUseProtection:          true,
		SpkUseRecovery:            true,
		SpkUsePersonal:            true,
		LLMPersonalizationEnabled: true,
	})
	require.NoError(t, err)

	recommendation, err := svc.Recommend(ctx, "usr_gading")
	require.NoError(t, err)
	assert.False(t, recommendation.RecommendationEnabled)
	assert.Equal(t, "none", recommendation.Feature.FeatureID)
	assert.False(t, recommendation.InterventionNeeded)
	assert.Empty(t, recommendation.RecommendationID)
}

// Completed recommendations past the observation window are lazily evaluated
// against aggregate block counts and feed the engine as history.
func TestSpkService_EffectivenessFeedback(t *testing.T) {
	st := store.NewSeeded()
	now := time.Now().UTC()
	recommendedAt := now.AddDate(0, 0, -6)
	completedAt := recommendedAt.Add(2 * time.Hour)
	aggregateDate := func(offset time.Time) time.Time {
		y, m, d := offset.UTC().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
	// Heavy blocks before, light after -> should evaluate EFFECTIVE.
	st.AggregateEvents = append(st.AggregateEvents,
		store.AggregateEvent{ID: "agg_b1", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "b1", EventType: "block_count_sync", EventDate: aggregateDate(recommendedAt.AddDate(0, 0, -2)), Count: 5, CreatedAt: now},
		store.AggregateEvent{ID: "agg_b2", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "b2", EventType: "block_count_sync", EventDate: aggregateDate(recommendedAt.AddDate(0, 0, -1)), Count: 5, CreatedAt: now},
		store.AggregateEvent{ID: "agg_a1", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "a1", EventType: "block_count_sync", EventDate: aggregateDate(recommendedAt.AddDate(0, 0, 1)), Count: 1, CreatedAt: now},
		store.AggregateEvent{ID: "agg_a2", UserID: "usr_gading", DeviceID: "dev_android", IdempotencyKey: "a2", EventType: "block_count_sync", EventDate: aggregateDate(recommendedAt.AddDate(0, 0, 2)), Count: 1, CreatedAt: now},
	)
	st.InterventionRecords = append(st.InterventionRecords, store.InterventionRecord{
		ID: "spk_past", UserID: "usr_gading",
		InterventionKey: "SHORT_RECOVERY_PRACTICE", ResponseType: "RECOVERY_PRACTICE",
		SupportLevel: "HIGH", EngagementLevel: "LOW", ReadinessLevel: "",
		Status: "completed", RecommendedAt: recommendedAt, CompletedAt: &completedAt,
		EffectivenessStatus: "NOT_EVALUATED",
	})

	svc := newSpkTestService(st)
	_, err := svc.Recommend(context.Background(), "usr_gading")
	require.NoError(t, err)

	records := st.Snapshot().InterventionRecords
	var updated *model.InterventionRecord
	for i := range records {
		if records[i].ID == "spk_past" {
			updated = &records[i]
			break
		}
	}
	require.NotNil(t, updated, "past record must still exist")
	assert.Equal(t, "EFFECTIVE", updated.EffectivenessStatus)
}
