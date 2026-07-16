package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/networkrulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetNetworkRulesets(ctx context.Context) ([]model.Release, error) {
	if r.db == nil {
		return copyReleases(r.store.Snapshot().NetworkRulesets), nil
	}
	rows, err := r.db.NetworkRulesetRelease.Query().All(ctx)
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
			DownloadURL:     "/v1/releases/network-rulesets/" + item.Version + "/download",
			Metrics:         item.RulesJSON,
			PublishedAtText: humanPublished(item.PublishedAt),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       time.Time{},
		})
	}
	return list, nil
}

func (r *Repository) CreateNetworkRulesetRelease(ctx context.Context, id, version, artifactPath, sha256Val string, rules map[string]any) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.NetworkRulesets = append(r.store.NetworkRulesets, store.Release{
			ID: id, Version: version, Platform: "all", SHA256: sha256Val,
			ArtifactPath: artifactPath,
			Status:       "validated", DownloadURL: "/v1/releases/network-rulesets/" + version + "/download",
			Metrics: rules, PublishedAtText: "Validated",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		})
		return nil
	}
	_, err := r.db.NetworkRulesetRelease.Create().
		SetID(id).
		SetVersion(version).
		SetArtifactPath(artifactPath).
		SetSha256(sha256Val).
		SetRulesJSON(rules).
		SetStatus(networkrulesetrelease.StatusValidated).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
