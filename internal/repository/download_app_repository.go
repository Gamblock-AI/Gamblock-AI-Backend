package repository

import (
	"context"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/downloadapp"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) ListDownloadApps(ctx context.Context, publicOnly bool) ([]model.DownloadApp, error) {
	if r.db == nil {
		items := append([]model.DownloadApp(nil), r.store.Snapshot().DownloadApps...)
		return filterAndSortDownloadApps(items, publicOnly), nil
	}
	query := r.db.DownloadApp.Query()
	if publicOnly {
		query = query.Where(downloadapp.PublishedEQ(true))
	}
	rows, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.DownloadApp, 0, len(rows))
	for _, row := range rows {
		items = append(items, downloadAppFromEnt(row))
	}
	return filterAndSortDownloadApps(items, publicOnly), nil
}

func (r *Repository) SaveDownloadApp(ctx context.Context, actor string, item model.DownloadApp) (model.DownloadApp, error) {
	now := time.Now().UTC()
	if r.db == nil {
		item.ID = "download_" + item.Platform
		item.UpdatedBy = actor
		item.UpdatedAt = now
		r.store.Lock()
		for index, existing := range r.store.DownloadApps {
			if existing.Platform == item.Platform {
				item.CreatedAt = existing.CreatedAt
				r.store.DownloadApps[index] = item
				r.store.Unlock()
				return item, nil
			}
		}
		item.CreatedAt = now
		r.store.DownloadApps = append(r.store.DownloadApps, item)
		r.store.Unlock()
		return item, nil
	}

	row, err := r.db.DownloadApp.Query().Where(downloadapp.PlatformEQ(downloadapp.Platform(item.Platform))).Only(ctx)
	if ent.IsNotFound(err) {
		created, createErr := r.db.DownloadApp.Create().SetID("download_" + item.Platform).
			SetPlatform(downloadapp.Platform(item.Platform)).SetEyebrowJSON(item.Eyebrow).
			SetTitleJSON(item.Title).SetDescriptionJSON(item.Description).
			SetRequirementsJSON(item.Requirements).SetArchitectureJSON(item.Architecture).
			SetFeaturesJSON(item.Features).SetVersion(item.Version).SetAssetsJSON(item.Assets).
			SetPublished(item.Published).SetUpdatedBy(actor).Save(ctx)
		if createErr != nil {
			return model.DownloadApp{}, createErr
		}
		r.RefreshStore(ctx)
		return downloadAppFromEnt(created), nil
	}
	if err != nil {
		return model.DownloadApp{}, err
	}
	updated, err := r.db.DownloadApp.UpdateOneID(row.ID).SetEyebrowJSON(item.Eyebrow).
		SetTitleJSON(item.Title).SetDescriptionJSON(item.Description).
		SetRequirementsJSON(item.Requirements).SetArchitectureJSON(item.Architecture).
		SetFeaturesJSON(item.Features).SetVersion(item.Version).SetAssetsJSON(item.Assets).
		SetPublished(item.Published).SetUpdatedBy(actor).Save(ctx)
	if err != nil {
		return model.DownloadApp{}, err
	}
	r.RefreshStore(ctx)
	return downloadAppFromEnt(updated), nil
}

func filterAndSortDownloadApps(items []model.DownloadApp, publicOnly bool) []model.DownloadApp {
	filtered := make([]model.DownloadApp, 0, len(items))
	for _, item := range items {
		if publicOnly && !item.Published {
			continue
		}
		filtered = append(filtered, item)
	}
	order := map[string]int{"android": 0, "windows": 1, "browser_extension": 2}
	sort.Slice(filtered, func(i, j int) bool { return order[filtered[i].Platform] < order[filtered[j].Platform] })
	return filtered
}

func downloadAppFromEnt(row *ent.DownloadApp) model.DownloadApp {
	return model.DownloadApp{
		ID: row.ID, Platform: row.Platform.String(), Eyebrow: row.EyebrowJSON,
		Title: row.TitleJSON, Description: row.DescriptionJSON,
		Requirements: row.RequirementsJSON, Architecture: row.ArchitectureJSON,
		Features: row.FeaturesJSON, Version: row.Version, Assets: row.AssetsJSON,
		Published: row.Published, UpdatedBy: row.UpdatedBy,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}
