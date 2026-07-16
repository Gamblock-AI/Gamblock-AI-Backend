package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/rulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetRulesetReleases(ctx context.Context) ([]model.Release, error) {
	if r.db == nil {
		return copyReleases(r.store.Snapshot().RulesetReleases), nil
	}
	rows, err := r.db.RulesetRelease.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	var list []model.Release
	for _, item := range rows {
		list = append(list, model.Release{
			ID:              item.ID,
			Version:         item.Version,
			Platform:        "all",
			SHA256:          item.Sha256,
			ArtifactPath:    item.ArtifactPath,
			Status:          item.Status.String(),
			DownloadURL:     "/v1/releases/ruleset/" + item.Version + "/download",
			Metrics:         item.RulesJSON,
			PublishedAtText: humanPublished(item.PublishedAt),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) CreateRulesetRelease(ctx context.Context, id, version, artifactPath, sha256Val string, rules map[string]any) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.RulesetReleases = append(r.store.RulesetReleases, store.Release{
			ID: id, Version: version, Platform: "all", SHA256: sha256Val,
			ArtifactPath: artifactPath,
			Status:       "validated", DownloadURL: "/v1/releases/ruleset/" + version + "/download",
			Metrics: rules, PublishedAtText: "Not published",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		})
		return nil
	}
	_, err := r.db.RulesetRelease.Create().
		SetID(id).
		SetVersion(version).
		SetArtifactPath(artifactPath).
		SetSha256(sha256Val).
		SetRulesJSON(rules).
		SetStatus(rulesetrelease.StatusValidated).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
