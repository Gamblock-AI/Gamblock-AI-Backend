package db

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func loadReleaseStore(ctx context.Context, client *ent.Client, out *store.Store) error {
	modules, err := client.PsychoeducationModule.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range modules {
		var publishedDocument = item.PublishedDocumentJSON
		var publishedDocumentPointer *model.EducationDocument
		if item.PublishedRevision > 0 {
			publishedDocumentPointer = &publishedDocument
		}
		out.Modules = append(out.Modules, store.EducationModule{
			ID:                item.ID,
			Slug:              item.Slug,
			Title:             item.Title,
			Summary:           item.Summary,
			BodyMarkdown:      item.BodyMarkdown,
			EstimatedMinutes:  item.EstimatedMinutes,
			Progress:          0,
			Status:            item.Status.String(),
			DraftDocument:     item.DraftDocumentJSON,
			PublishedDocument: publishedDocumentPointer,
			DraftRevision:     item.DraftRevision,
			PublishedRevision: item.PublishedRevision,
			PublishedAt:       item.PublishedAt,
			ArchivedAt:        item.ArchivedAt,
			CreatedBy:         item.CreatedBy,
			UpdatedBy:         item.UpdatedBy,
			CreatedAt:         item.CreatedAt,
			UpdatedAt:         item.UpdatedAt,
		})
	}

	mediaRows, err := client.EducationMedia.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range mediaRows {
		out.EducationMedia = append(out.EducationMedia, store.EducationMedia{
			ID: item.ID, Kind: item.Kind.String(), Purpose: item.Purpose.String(),
			MediaType: item.MediaType.String(), MIMEType: item.MimeType,
			StorageKey: item.StorageKey, ExternalURL: item.ExternalURL,
			OriginalName: item.OriginalName, SizeBytes: item.SizeBytes,
			Width: item.Width, Height: item.Height, SHA256: item.Sha256,
			Status: item.Status.String(), CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	progressRows, err := client.PsychoeducationProgress.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range progressRows {
		out.EducationProgress = append(out.EducationProgress, store.EducationProgress{
			ID: item.ID, UserID: item.UserID, ModuleID: item.ModuleID, Revision: item.Revision,
			CompletedSectionIDs: item.CompletedSectionIds, OpenedMediaIDs: item.OpenedMediaIds,
			CorrectCheckIDs: item.CorrectCheckIds, ProgressPercent: item.ProgressPercent,
			CompletedAt: item.CompletedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	modelReleases, err := client.ModelRelease.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range modelReleases {
		out.ModelReleases = append(out.ModelReleases, store.Release{
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

	rulesets, err := client.RulesetRelease.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range rulesets {
		out.RulesetReleases = append(out.RulesetReleases, store.Release{
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

	networkRules, err := client.NetworkRulesetRelease.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range networkRules {
		out.NetworkRulesets = append(out.NetworkRulesets, store.Release{
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
	return nil
}
