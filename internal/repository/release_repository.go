package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/modelrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/networkrulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/rulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetModelReleases(ctx context.Context) ([]model.Release, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.Release
		for _, rel := range snapshot.ModelReleases {
			list = append(list, model.Release{
				ID:              rel.ID,
				Version:         rel.Version,
				Platform:        rel.Platform,
				SHA256:          rel.SHA256,
				Status:          rel.Status,
				DownloadURL:     rel.DownloadURL,
				Metrics:         rel.Metrics,
				PublishedAtText: rel.PublishedAtText,
				CreatedAt:       rel.CreatedAt,
				UpdatedAt:       rel.UpdatedAt,
			})
		}
		return list, nil
	}

	rows, err := r.db.ModelRelease.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.Release
	for _, item := range rows {
		list = append(list, model.Release{
			ID:              item.ID,
			Version:         item.Version,
			Platform:        item.Platform.String(),
			SHA256:          item.Sha256,
			Status:          item.Status.String(),
			DownloadURL:     "/v1/releases/model/" + item.Version + "/download",
			Metrics:         item.MetricsJSON,
			PublishedAtText: humanPublished(item.PublishedAt),
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) CreateModelRelease(ctx context.Context, id, version, platformVal, artifactPath, sha256Val, contract string, threshold float64, metrics map[string]any) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.ModelReleases = append(r.store.ModelReleases, store.Release{
			ID: id, Version: version, Platform: platformVal, SHA256: sha256Val,
			Status: "published", DownloadURL: "/v1/releases/model/" + version + "/download",
			Metrics: metrics, PublishedAtText: "Published",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		})
		return nil
	}
	_, err := r.db.ModelRelease.Create().
		SetID(id).
		SetVersion(version).
		SetPlatform(modelrelease.Platform(platformVal)).
		SetArtifactPath(artifactPath).
		SetSha256(sha256Val).
		SetContractVersion(contract).
		SetThreshold(threshold).
		SetMetricsJSON(metrics).
		SetStatus(modelrelease.StatusValidated).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) GetRulesetReleases(ctx context.Context) ([]model.Release, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.Release
		for _, rel := range snapshot.RulesetReleases {
			list = append(list, model.Release{
				ID:              rel.ID,
				Version:         rel.Version,
				Platform:        rel.Platform,
				SHA256:          rel.SHA256,
				Status:          rel.Status,
				DownloadURL:     rel.DownloadURL,
				Metrics:         rel.Metrics,
				PublishedAtText: rel.PublishedAtText,
				CreatedAt:       rel.CreatedAt,
				UpdatedAt:       rel.UpdatedAt,
			})
		}
		return list, nil
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
			Status: "published", DownloadURL: "/v1/releases/ruleset/" + version + "/download",
			Metrics: rules, PublishedAtText: "Published",
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

func (r *Repository) GetNetworkRulesets(ctx context.Context) ([]model.Release, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.Release
		for _, rel := range snapshot.NetworkRulesets {
			list = append(list, model.Release{
				ID:              rel.ID,
				Version:         rel.Version,
				Platform:        rel.Platform,
				SHA256:          rel.SHA256,
				Status:          rel.Status,
				DownloadURL:     rel.DownloadURL,
				Metrics:         rel.Metrics,
				PublishedAtText: rel.PublishedAtText,
				CreatedAt:       rel.CreatedAt,
				UpdatedAt:       rel.UpdatedAt,
			})
		}
		return list, nil
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
			Status: "validated", DownloadURL: "/v1/releases/network-rulesets/" + version + "/download",
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
