package db

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func loadContentStore(ctx context.Context, client *ent.Client, out *store.Store) error {
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

	return nil
}
