package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/spk"
	"github.com/google/uuid"
)

const (
	spkRecordPrefix = "spk_"
	spkLLMTimeout   = 8 * time.Second
)

// SpkService wires the rule-based SPK engine to real user data and optionally
// enriches the deterministic result with a consented DeepSeek LLM message. The
// engine stays the decision authority; the LLM only writes copy.
type SpkService struct {
	repo     *repository.Repository
	engine   *spk.Engine
	deepseek *DeepSeekService
	cfg      config.Config
	logger   *zap.Logger
}

func NewSpkService(repo *repository.Repository, cfg config.Config, logger *zap.Logger, deepseek *DeepSeekService) *SpkService {
	return &SpkService{
		repo:     repo,
		engine:   spk.NewEngine(spk.DefaultConfig()),
		deepseek: deepseek,
		cfg:      cfg,
		logger:   logger,
	}
}

// Recommend runs the full SPK pipeline and returns the dashboard-facing
// recommendation. It is safe for data-poor users: unavailable factors are left
// nil and the engine re-normalizes and falls back gracefully.
func (s *SpkService) Recommend(ctx context.Context, userID string) (model.SpkRecommendation, error) {
	now := time.Now().UTC()

	pref, err := s.repo.SpkPreference(ctx, userID)
	if err != nil {
		return model.SpkRecommendation{}, err
	}
	if !pref.SpkRecommendationEnabled {
		return model.SpkRecommendation{
			RecommendedAt:         now,
			RecommendationEnabled: false,
			Feature: model.SpkFeature{
				InterventionKey: "NO_INTERVENTION", ResponseType: "APPRECIATION",
				FeatureID: "none", Category: "maintain", Action: "continue",
			},
			DataState: model.SpkDataInsufficient,
		}, nil
	}

	data, err := s.repo.SpkInputData(ctx, userID, now)
	if err != nil {
		return model.SpkRecommendation{}, err
	}

	records, err := s.repo.InterventionRecords(ctx, userID)
	if err != nil {
		return model.SpkRecommendation{}, err
	}
	records = s.evaluatePendingEffectiveness(ctx, userID, records, now)

	condition := s.buildCondition(data, pref, records)
	readinessValue := spk.ChangeReadiness("")
	if condition.ChangeReadiness != nil {
		readinessValue = *condition.ChangeReadiness
	}

	result := s.engine.EvaluateAt(condition, now)

	available := result.ScoreBreakdown.AvailableWeight
	total := result.ScoreBreakdown.TotalWeight
	dataState := classifySpkDataState(available, total)
	feature := mapSpkFeature(result, s.engine.Config())

	llmUsed := false
	message := ""
	explanation := ""
	if s.cfg.SPKLLMEnrichment && pref.LLMPersonalizationEnabled && pref.SpkUsePersonal && data.HasIntention {
		llmUsed, message, explanation = s.personalize(ctx, result, dataState, data)
	}

	_, dayStart, dayEnd := jakartaDay(now)
	rec := model.InterventionRecord{
		UserID:                  userID,
		InterventionKey:         string(result.InterventionKey),
		ResponseType:            string(result.ResponseType),
		SupportLevel:            string(result.SupportLevel),
		EngagementLevel:         string(result.EngagementLevel),
		ReadinessLevel:          string(readinessValue),
		Status:                  "recommended",
		RecommendedAt:           now,
		EffectivenessStatus:     string(spk.EffectivenessNotEvaluated),
		PersonalizedMessage:     message,
		PersonalizedExplanation: explanation,
		LLMUsed:                 llmUsed,
	}

	recommendationID := ""
	todayRecord, err := s.repo.TodayInterventionRecord(ctx, userID, dayStart, dayEnd)
	if err != nil {
		return model.SpkRecommendation{}, err
	}
	if todayRecord != nil {
		rec.ID = todayRecord.ID
		if todayRecord.Status == "completed" {
			rec.Status = "completed"
			rec.CompletedAt = todayRecord.CompletedAt
		}
		saved, updateErr := s.repo.UpdateInterventionRecord(ctx, userID, rec)
		if updateErr != nil {
			return model.SpkRecommendation{}, updateErr
		}
		recommendationID = saved.ID
	} else {
		rec.ID = spkRecordPrefix + uuid.NewString()
		saved, saveErr := s.repo.UpsertInterventionRecord(ctx, rec)
		if saveErr != nil {
			return model.SpkRecommendation{}, saveErr
		}
		recommendationID = saved.ID
	}

	return model.SpkRecommendation{
		RecommendationID:        recommendationID,
		RecommendedAt:           now,
		RecommendationEnabled:   true,
		Feature:                 feature,
		SupportLevel:            string(result.SupportLevel),
		SupportScore:            result.SupportScore,
		EngagementLevel:         string(result.EngagementLevel),
		InterventionNeeded:      result.InterventionNeeded,
		ReasonCode:              string(result.ReasonCode),
		Reason:                  buildSpkReason(result),
		TimeTrigger:             toSpkTimeTrigger(result.TimeTrigger),
		EffectivenessUsed:       result.EffectivenessHistoryUsed,
		TriggeredRules:          result.TriggeredRules,
		DataState:               dataState,
		DataGaps:                buildSpkDataGaps(data, pref),
		AvailableWeightPercent:  available,
		UnavailableFields:       result.UnavailableFields,
		PersonalizedMessage:     message,
		PersonalizedExplanation: explanation,
		LLMUsed:                 llmUsed,
	}, nil
}

// buildSpkDataGaps lists actionable missing data the student can add to
// sharpen the recommendation. Gaps are only produced for categories the student
// allows (privacy toggles on) but that genuinely lack data; data the student
// turned off is intentionally not surfaced as a gap.
func buildSpkDataGaps(data repository.SpkInputData, pref model.SpkPreference) []model.SpkDataGap {
	var gaps []model.SpkDataGap
	if pref.SpkUseRecovery {
		if !data.HasAnyLearning {
			gaps = append(gaps, model.SpkDataGap{Key: "learning_activities_7d", Action: "learn", Route: "/skills"})
		}
		if !data.HasActivity || !data.HasAnyMission {
			gaps = append(gaps, model.SpkDataGap{Key: "recovery_activity", Action: "check_in", Route: "/recovery"})
		}
	}
	if pref.SpkUsePersonal && !data.HasIntention {
		gaps = append(gaps, model.SpkDataGap{Key: "change_readiness", Action: "set_intention"})
	}
	return gaps
}

// MarkCompleted records that the student finished the daily recommendation.
func (s *SpkService) MarkCompleted(ctx context.Context, userID, recordID string) (model.InterventionRecord, error) {
	record, err := s.repo.CompleteIntervention(ctx, userID, recordID)
	if err != nil {
		return model.InterventionRecord{}, err
	}
	return *record, nil
}

// GetPreference returns the stored SPK opt-in preference (default-off).
func (s *SpkService) GetPreference(ctx context.Context, userID string) (model.SpkPreference, error) {
	return s.repo.SpkPreference(ctx, userID)
}

// UpdatePreference stores the full SPK privacy set.
func (s *SpkService) UpdatePreference(ctx context.Context, userID string, pref model.SpkPreference) (model.SpkPreference, error) {
	return s.repo.UpsertSpkPreference(ctx, userID, pref)
}

// buildCondition converts privacy-safe repository data into the engine input,
// honoring the student's privacy toggles. Disabled categories are left nil so
// the engine never uses them (and never mistakes "unknown" for "bad").
// Cold-start factors (no activity/mission/learning history ever) are also left
// nil regardless of the toggles.
func (s *SpkService) buildCondition(data repository.SpkInputData, pref model.SpkPreference, records []model.InterventionRecord) spk.Condition {
	cond := spk.Condition{}

	if pref.SpkUseProtection {
		blockedToday := data.BlockedAttemptsToday
		blockedDays := data.BlockedActiveDays7d
		cond.BlockedAttemptsToday = &blockedToday
		cond.BlockedActiveDays7d = &blockedDays
		if len(data.BlockedEventTimes) > 0 {
			cond.BlockedEventTimes = data.BlockedEventTimes
		}
	}

	if pref.SpkUseRecovery {
		if data.HasActivity {
			streak := data.RecoveryStreakDays
			cond.RecoveryStreakDays = &streak
		}
		if data.HasAnyMission {
			completed := data.DailyMissionsCompleted
			cond.DailyMissionsCompleted = &completed
			if data.DailyMissionsTotal > 0 {
				total := data.DailyMissionsTotal
				cond.DailyMissionsTotal = &total
			}
		}
		if data.HasAnyLearning {
			learning := data.LearningActivities7d
			cond.LearningActivities7d = &learning
		}
	}

	if pref.SpkUsePersonal {
		if readiness, ok := mapReadiness(data.QuitMotivation); ok && data.HasIntention {
			cond.ChangeReadiness = &readiness
		}
		cond.EducationImpact = optionalTextPtr(data.SchoolImpact)
		cond.FinancialImpact = optionalTextPtr(data.MoneySpent)
		cond.GamblingScreenTime = optionalTextPtr(data.ScreenTime)
		cond.PreviousQuitAttempts = mapQuitAttempts(data.QuitAttempts)
		cond.PersonalIntention = optionalTextPtr(data.PersonalIntention)
	}

	accountability := data.AccountabilityEnabled
	cond.AccountabilityEnabled = &accountability
	cond.PreviousInterventions = recordsToInterventions(records)
	return cond
}

// evaluatePendingEffectiveness lazily classifies completed recommendations
// whose observation window has elapsed, using aggregate block counts before and
// after the recommendation. Updated records feed the engine as history on the
// next evaluation.
func (s *SpkService) evaluatePendingEffectiveness(ctx context.Context, userID string, records []model.InterventionRecord, now time.Time) []model.InterventionRecord {
	cfg := s.engine.Config()
	window := time.Duration(cfg.EffectivenessWindowDays) * 24 * time.Hour
	for i := range records {
		rec := &records[i]
		if rec.Status != "completed" || rec.CompletedAt == nil ||
			rec.EffectivenessStatus != string(spk.EffectivenessNotEvaluated) {
			continue
		}
		if now.Before(rec.CompletedAt.Add(window)) {
			continue
		}
		anchor := startOfUTCDay(rec.RecommendedAt)
		counts, err := s.repo.BlockCountsByDate(ctx, userID, anchor.Add(-window), anchor.Add(2*window))
		if err != nil {
			continue
		}
		status := cfg.EvaluateEffectiveness(spk.EffectivenessInput{
			InterventionKey: spk.InterventionKey(rec.InterventionKey),
			Completed:       true,
			BeforeBlocked:   dailySeries(counts, anchor.Add(-window), anchor),
			AfterBlocked:    dailySeries(counts, anchor, anchor.Add(window)),
		})
		if string(status) != rec.EffectivenessStatus {
			rec.EffectivenessStatus = string(status)
			_ = s.repo.UpdateInterventionEffectiveness(ctx, userID, rec.ID, string(status))
		}
	}
	return records
}

// personalize sends only the SPK decision plus the user's self-reported context
// to DeepSeek and returns the generated message/explanation. Any failure falls
// back to a rule-based recommendation with no copy.
func (s *SpkService) personalize(ctx context.Context, result spk.Result, dataState model.SpkDataState, data repository.SpkInputData) (bool, string, string) {
	if s.deepseek == nil {
		return false, "", ""
	}
	systemPrompt := "You are a supportive recovery coach for an Indonesian university student using a gambling self-control app. " +
		"The rule-based decision engine has already chosen the recommended activity category. " +
		"Do NOT change or invent the category, do not mention gambling websites, URLs, or browsing data, and never diagnose or make medical claims. " +
		"Keep the tone warm, concise, and encouraging. " +
		"Respond ONLY with JSON: {\"message\":\"short headline, max 15 words, Bahasa Indonesia\",\"explanation\":\"why this fits today, max 2 sentences, Bahasa Indonesia\"}."
	userMessage := fmt.Sprintf(
		"Recommendation (decided by the engine): intervention=%s, response_type=%s, support_level=%s, engagement_level=%s, needed=%t, data_state=%s.\n"+
			"Self-reported user context: intention=%q, education_impact=%q, financial_impact=%q, screen_time=%q, quit_attempts=%q",
		result.InterventionKey, result.ResponseType, result.SupportLevel, result.EngagementLevel,
		result.InterventionNeeded, dataState,
		data.PersonalIntention, data.SchoolImpact, data.MoneySpent, data.ScreenTime, data.QuitAttempts,
	)

	ctx, cancel := context.WithTimeout(ctx, spkLLMTimeout)
	defer cancel()

	var out struct {
		Message     string `json:"message"`
		Explanation string `json:"explanation"`
	}
	if err := s.deepseek.PersonalizeJSON(ctx, systemPrompt, userMessage, &out); err != nil {
		s.logger.Warn("spk llm personalization failed", zap.Error(err))
		return false, "", ""
	}
	message := strings.TrimSpace(out.Message)
	explanation := strings.TrimSpace(out.Explanation)
	if message == "" {
		return false, "", ""
	}
	return true, message, explanation
}

func classifySpkDataState(available, total float64) model.SpkDataState {
	if total <= 0 {
		return model.SpkDataInsufficient
	}
	ratio := available / total
	switch {
	case ratio >= 0.9:
		return model.SpkDataSufficient
	case ratio >= 0.5:
		return model.SpkDataPartial
	default:
		return model.SpkDataInsufficient
	}
}

func mapSpkFeature(result spk.Result, cfg spk.Config) model.SpkFeature {
	feature := model.SpkFeature{
		InterventionKey: string(result.InterventionKey),
		ResponseType:    string(result.ResponseType),
	}
	if meta, ok := cfg.KnowledgeBase[result.InterventionKey]; ok {
		feature.Load = meta.Load
	}
	switch result.InterventionKey {
	case spk.InterventionNoIntervention:
		feature.FeatureID, feature.Category, feature.Action = "none", "maintain", "continue"
	case spk.InterventionLightEducation:
		feature.FeatureID, feature.Category, feature.Route, feature.Action = "education", "education", "/education", "learn"
	case spk.InterventionShortRecoveryPractice:
		feature.FeatureID, feature.Category, feature.Route, feature.Action = "recovery_practice", "recovery", "/recovery", "practice"
	case spk.InterventionShortGrounding:
		feature.FeatureID, feature.Category, feature.Route, feature.Action = "grounding", "recovery", "/recovery", "grounding"
	case spk.InterventionShortReflection:
		feature.FeatureID, feature.Category, feature.Route, feature.Action = "reflection", "journal", "/journal", "reflect"
	case spk.InterventionAlternativeActivity:
		feature.FeatureID, feature.Category, feature.Route, feature.Action = "alternative_activity", "skills", "/skills", "explore"
	case spk.InterventionAccountabilityOption:
		feature.FeatureID, feature.Category, feature.Route, feature.Action = "accountability", "accountability", "/accountability", "connect"
	default:
		feature.FeatureID, feature.Category, feature.Action = "none", "maintain", "continue"
	}
	return feature
}

func toSpkTimeTrigger(trigger *spk.TimeTrigger) *model.SpkTimeTrigger {
	if trigger == nil {
		return nil
	}
	return &model.SpkTimeTrigger{
		HasTimePattern: trigger.HasTimePattern,
		PatternStart:   trigger.PatternStart,
		PatternEnd:     trigger.PatternEnd,
		TriggerStart:   trigger.TriggerStart,
		TriggerEnd:     trigger.TriggerEnd,
	}
}

// buildSpkReason turns the engine's auditable result into a structured reason:
// the dominant rule plus the available score factors, ordered by weight so
// clients can render the most influential drivers first.
func buildSpkReason(result spk.Result) model.SpkReason {
	reason := model.SpkReason{
		Code:            string(result.ReasonCode),
		SupportLevel:    string(result.SupportLevel),
		EngagementLevel: string(result.EngagementLevel),
		SupportScore:    result.SupportScore,
	}
	for _, component := range result.ScoreBreakdown.Components {
		if !component.Available {
			continue
		}
		reason.Factors = append(reason.Factors, model.SpkReasonFactor{
			Key:    component.Key,
			Score:  component.Score,
			Weight: component.WeightPercent,
		})
	}
	sort.SliceStable(reason.Factors, func(i, j int) bool {
		if reason.Factors[i].Weight != reason.Factors[j].Weight {
			return reason.Factors[i].Weight > reason.Factors[j].Weight
		}
		return reason.Factors[i].Key < reason.Factors[j].Key
	})
	return reason
}

func mapReadiness(quitMotivation string) (spk.ChangeReadiness, bool) {
	switch quitMotivation {
	case "uncertain":
		return spk.ReadinessReadyLow, true
	case "somewhat":
		return spk.ReadinessReadyMedium, true
	case "very":
		return spk.ReadinessReadyHigh, true
	case "determined":
		return spk.ReadinessReadyFirm, true
	default:
		return "", false
	}
}

func mapQuitAttempts(value string) *int {
	switch value {
	case "never":
		return intPtr(0)
	case "once":
		return intPtr(1)
	case "multiple":
		return intPtr(3)
	default:
		return nil
	}
}

func recordsToInterventions(records []model.InterventionRecord) []spk.InterventionRecord {
	out := make([]spk.InterventionRecord, 0, len(records))
	for _, record := range records {
		out = append(out, spk.InterventionRecord{
			InterventionKey:       spk.InterventionKey(record.InterventionKey),
			Timestamp:             record.RecommendedAt,
			Completed:             record.Status == "completed",
			SupportLevelAtTime:    spk.SupportLevel(record.SupportLevel),
			EngagementLevelAtTime: spk.EngagementLevel(record.EngagementLevel),
			ReadinessLevelAtTime:  spk.ChangeReadiness(record.ReadinessLevel),
			EffectivenessStatus:   spk.EffectivenessStatus(record.EffectivenessStatus),
		})
	}
	return out
}

func optionalTextPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func intPtr(value int) *int { return &value }

func startOfUTCDay(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func dailySeries(counts map[string]int, start, end time.Time) []int {
	var out []int
	for day := startOfUTCDay(start); day.Before(end); day = day.AddDate(0, 0, 1) {
		out = append(out, counts[day.Format("2006-01-02")])
	}
	return out
}
