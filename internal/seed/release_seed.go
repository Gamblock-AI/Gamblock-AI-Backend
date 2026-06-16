package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/modelrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/networkrulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/rulesetrelease"
)

func SeedReleases(ctx context.Context, client *ent.Client, now time.Time) error {
	if _, err := client.ModelRelease.Create().SetID("rel_model_031").SetVersion("artifact-v0.3.1").SetPlatform(modelrelease.PlatformAll).SetArtifactPath("./var/artifacts/artifact-v0.3.1.onnx").SetSha256("c3a12b939f2923c21d3e729a514610a3989cab321c895c9d2f63ac8eb8a0199c").SetThreshold(0.72).SetContractVersion("v1").SetStatus(modelrelease.StatusPublished).SetMetricsJSON(map[string]any{"false_positive_reviewed": true, "latency_ms_p95": 42}).SetPublishedAt(now.Add(-48 * time.Hour)).Save(ctx); err != nil {
		return err
	}
	if _, err := client.RulesetRelease.Create().SetID("rel_rules_202605").SetVersion("ruleset-2026.05.1").SetArtifactPath("./var/artifacts/ruleset-2026.05.1.json").SetSha256("c9a31a473ca232c060d49d431e6a1029670df9c4888f32dbc4d06554a41bf586").SetStatus(rulesetrelease.StatusPublished).SetRulesJSON(map[string]any{"rules": 42}).SetPublishedAt(now.Add(-2 * time.Hour)).Save(ctx); err != nil {
		return err
	}
	if _, err := client.NetworkRulesetRelease.Create().SetID("rel_net_12").SetVersion("global-risk-category-v12").SetArtifactPath("./var/artifacts/global-risk-category-v12.json").SetSha256("e6091a4405a8db35789ea9197f2b44d46d16d425ee8b805c35c8f4b7f5d76127").SetStatus(networkrulesetrelease.StatusValidated).SetRulesJSON(map[string]any{"domains": 0, "privacy": "metadata_only"}).SetCreatedBy("usr_nasywa").Save(ctx); err != nil {
		return err
	}
	return nil
}
