package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/modelrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetModelReleases(ctx context.Context) ([]model.Release, error) {
	if r.db == nil {
		return copyReleases(r.store.Snapshot().ModelReleases), nil
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
			ArtifactPath:    item.ArtifactPath,
			ContractVersion: item.ContractVersion,
			Threshold:       item.Threshold,
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
			ArtifactPath: artifactPath, ContractVersion: contract, Threshold: threshold,
			Status: "validated", DownloadURL: "/v1/releases/model/" + version + "/download",
			Metrics: metrics, PublishedAtText: "Not published",
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
