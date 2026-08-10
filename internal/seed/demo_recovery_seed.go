package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/dailymission"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/intention"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningprogress"
)

// SeedDemoRecoveryData installs recommendation-relevant demo records (Niat
// Perubahan quiz, daily check-in, completed missions, Learning Hub progress,
// and psychoeducation progress) for the seeded demo students so the SPK
// recommendation has enough data. It is idempotent (fixed IDs, skip if present)
// and only ever runs in the demo seed path; the production seeder never creates
// demo accounts or activity.
func SeedDemoRecoveryData(ctx context.Context, client *ent.Client, now time.Time) error {
	jakarta := time.FixedZone("Asia/Jakarta", 7*60*60)
	today := now.In(jakarta).Format("2006-01-02")

	if err := seedDemoIntention(ctx, client); err != nil {
		return err
	}
	if err := seedDemoCheckIn(ctx, client); err != nil {
		return err
	}
	if err := seedDemoMissions(ctx, client, today); err != nil {
		return err
	}
	if err := seedDemoLearningProgress(ctx, client, now); err != nil {
		return err
	}
	if err := seedDemoEducationProgress(ctx, client, now); err != nil {
		return err
	}
	return nil
}

func seedDemoIntention(ctx context.Context, client *ent.Client) error {
	const id = "int_seed_gading"
	if _, err := client.Intention.Get(ctx, id); err == nil {
		return nil
	} else if !ent.IsNotFound(err) {
		return err
	}
	_, err := client.Intention.Create().
		SetID(id).
		SetUserID("usr_gading").
		SetIntentionText("Saya ingin menyelesaikan kuliah dengan pikiran yang lebih tenang.").
		SetStatus(intention.StatusActive).
		SetSchoolImpact(intention.SchoolImpactHappened).
		SetMoneySpent(intention.MoneySpent("500k_5m")).
		SetScreenTime(intention.ScreenTime("1h_3h")).
		SetQuitAttempts(intention.QuitAttemptsMultiple).
		SetQuitMotivation(intention.QuitMotivationDetermined).
		Save(ctx)
	return err
}

func seedDemoCheckIn(ctx context.Context, client *ent.Client) error {
	const id = "chk_seed_gading"
	if _, err := client.CheckIn.Get(ctx, id); err == nil {
		return nil
	} else if !ent.IsNotFound(err) {
		return err
	}
	_, err := client.CheckIn.Create().
		SetID(id).
		SetUserID("usr_gading").
		SetMoodScore(4).
		SetUrgeScore(2).
		SetNillableContextText(&[]string{"Merasa cukup tenang pagi ini."}[0]).
		Save(ctx)
	return err
}

func seedDemoMissions(ctx context.Context, client *ent.Client, today string) error {
	for _, key := range []string{"mission_1", "mission_2"} {
		id := "dms_seed_gading_" + key
		if _, err := client.DailyMission.Get(ctx, id); err == nil {
			continue
		} else if !ent.IsNotFound(err) {
			return err
		}
		completedAt := time.Now().UTC()
		if _, err := client.DailyMission.Create().
			SetID(id).
			SetUserID("usr_gading").
			SetMissionDate(today).
			SetMissionKey(key).
			SetSource(dailymission.SourceSystem).
			SetStatus(dailymission.StatusCompleted).
			SetExpReward(10).
			SetCompletedAt(completedAt).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func seedDemoLearningProgress(ctx context.Context, client *ent.Client, now time.Time) error {
	items := []struct {
		id    string
		state string
		age   int
	}{
		{id: "learn_know_your_urges", state: "completed", age: 1},
		{id: "learn_pause_technique", state: "started", age: 2},
		{id: "learn_financial_guardrails", state: "started", age: 3},
	}
	for _, item := range items {
		id := "lprog_seed_gading_" + item.id
		if _, err := client.LearningProgress.Get(ctx, id); err == nil {
			continue
		} else if !ent.IsNotFound(err) {
			return err
		}
		updatedAt := now.Add(-time.Duration(item.age) * 24 * time.Hour)
		_, err := client.LearningProgress.Create().
			SetID(id).
			SetUserID("usr_gading").
			SetItemID(item.id).
			SetState(learningprogress.State(item.state)).
			SetUpdatedAt(updatedAt).
			Save(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func seedDemoEducationProgress(ctx context.Context, client *ent.Client, now time.Time) error {
	const id = "eprog_seed_gading_impulse"
	if _, err := client.PsychoeducationProgress.Get(ctx, id); err == nil {
		return nil
	} else if !ent.IsNotFound(err) {
		return err
	}
	_, err := client.PsychoeducationProgress.Create().
		SetID(id).
		SetUserID("usr_gading").
		SetModuleID("mod_impulse_cycle").
		SetRevision(1).
		SetCompletedSectionIds([]string{"cycle-map", "choice-point"}).
		SetProgressPercent(66).
		SetUpdatedAt(now.Add(-2 * 24 * time.Hour)).
		Save(ctx)
	return err
}
