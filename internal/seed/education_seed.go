package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationmodule"
)

func SeedEducationModules(ctx context.Context, client *ent.Client) error {
	modules := []struct {
		id, slug, title, summary, body string
		minutes                        int
	}{
		{"mod_pause", "pause-before-impulse", "Pause before impulse", "A short exercise to identify triggers and choose one safer action.", "## Pause\n\nName the impulse, breathe for ten seconds, and choose one safe next action.", 4},
		{"mod_finance", "financial-reality-check", "Financial reality check", "A simple reflection on losses, debt risk, and recovery support.", "## Reality check\n\nWrite down the amount at risk and contact your accountability partner.", 6},
		{"mod_support", "ask-for-support", "Ask for support", "How to talk with your partner without shame or blame.", "## Ask\n\nUse a short, concrete message and state the help you need now.", 5},
	}
	for _, item := range modules {
		if _, err := client.PsychoeducationModule.Create().SetID(item.id).SetSlug(item.slug).SetTitle(item.title).SetSummary(item.summary).SetBodyMarkdown(item.body).SetEstimatedMinutes(item.minutes).SetStatus(psychoeducationmodule.StatusPublished).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
