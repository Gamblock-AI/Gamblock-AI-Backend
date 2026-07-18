package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationmedia"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationmodule"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationprogress"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

var (
	ErrEducationNotFound = errors.New("education resource not found")
	ErrEducationConflict = errors.New("education draft was updated by another editor")
)

func moduleFromEnt(row *ent.PsychoeducationModule) model.EducationModule {
	var published *model.EducationDocument
	if row.PublishedRevision > 0 {
		document := row.PublishedDocumentJSON
		published = &document
	}
	return model.EducationModule{
		ID: row.ID, Slug: row.Slug, Title: row.Title, Summary: row.Summary,
		BodyMarkdown: row.BodyMarkdown, EstimatedMinutes: row.EstimatedMinutes,
		Status: row.Status.String(), DraftDocument: row.DraftDocumentJSON,
		PublishedDocument: published, DraftRevision: row.DraftRevision,
		PublishedRevision: row.PublishedRevision, PublishedAt: row.PublishedAt,
		ArchivedAt: row.ArchivedAt, CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mediaFromEnt(row *ent.EducationMedia) model.EducationMedia {
	return model.EducationMedia{
		ID: row.ID, Kind: row.Kind.String(), Purpose: row.Purpose.String(),
		MediaType: row.MediaType.String(), MIMEType: row.MimeType,
		StorageKey: row.StorageKey, ExternalURL: row.ExternalURL,
		OriginalName: row.OriginalName, SizeBytes: row.SizeBytes,
		Width: row.Width, Height: row.Height, SHA256: row.Sha256,
		Status: row.Status.String(), CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func progressFromEnt(row *ent.PsychoeducationProgress) model.EducationProgress {
	return model.EducationProgress{
		ID: row.ID, UserID: row.UserID, ModuleID: row.ModuleID, Revision: row.Revision,
		CompletedSectionIDs: nonNilStrings(row.CompletedSectionIds), OpenedMediaIDs: nonNilStrings(row.OpenedMediaIds),
		CorrectCheckIDs: nonNilStrings(row.CorrectCheckIds), ProgressPercent: row.ProgressPercent,
		CompletedAt: row.CompletedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func emptyEducationProgress(userID, moduleID string, revision int) model.EducationProgress {
	return model.EducationProgress{
		UserID: userID, ModuleID: moduleID, Revision: revision,
		CompletedSectionIDs: []string{}, OpenedMediaIDs: []string{}, CorrectCheckIDs: []string{},
	}
}

func (r *Repository) GetEducationModules(ctx context.Context) ([]model.EducationModule, error) {
	if r.db == nil {
		return r.store.Snapshot().Modules, nil
	}
	rows, err := r.db.PsychoeducationModule.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]model.EducationModule, 0, len(rows))
	for _, row := range rows {
		list = append(list, moduleFromEnt(row))
	}
	return list, nil
}

func (r *Repository) GetPublishedEducationModules(ctx context.Context) ([]model.EducationModule, error) {
	modules, err := r.GetEducationModules(ctx)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(modules, func(module model.EducationModule) bool {
		return module.Status != "published" || module.PublishedDocument == nil
	}), nil
}

func (r *Repository) GetEducationModuleByID(ctx context.Context, id string) (model.EducationModule, error) {
	if r.db == nil {
		for _, module := range r.store.Snapshot().Modules {
			if module.ID == id {
				return module, nil
			}
		}
		return model.EducationModule{}, ErrEducationNotFound
	}
	row, err := r.db.PsychoeducationModule.Get(ctx, id)
	if ent.IsNotFound(err) {
		return model.EducationModule{}, ErrEducationNotFound
	}
	if err != nil {
		return model.EducationModule{}, err
	}
	return moduleFromEnt(row), nil
}

func (r *Repository) GetEducationModuleBySlug(ctx context.Context, slug string) (model.EducationModule, error) {
	if r.db == nil {
		for _, module := range r.store.Snapshot().Modules {
			if strings.EqualFold(module.Slug, slug) {
				return module, nil
			}
		}
		return model.EducationModule{}, ErrEducationNotFound
	}
	row, err := r.db.PsychoeducationModule.Query().Where(psychoeducationmodule.SlugEQ(slug)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.EducationModule{}, ErrEducationNotFound
	}
	if err != nil {
		return model.EducationModule{}, err
	}
	return moduleFromEnt(row), nil
}

func (r *Repository) CreateEducationModule(ctx context.Context, module model.EducationModule) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.Modules = append(r.store.Modules, store.EducationModule(module))
		return nil
	}
	_, err := r.db.PsychoeducationModule.Create().
		SetID(module.ID).SetSlug(module.Slug).SetTitle(module.Title).
		SetSummary(module.Summary).SetBodyMarkdown(module.BodyMarkdown).
		SetEstimatedMinutes(module.EstimatedMinutes).
		SetStatus(psychoeducationmodule.Status(module.Status)).
		SetDraftDocumentJSON(module.DraftDocument).SetDraftRevision(module.DraftRevision).
		SetCreatedBy(module.CreatedBy).SetUpdatedBy(module.UpdatedBy).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) UpdateEducationDraft(ctx context.Context, id string, expectedRevision int, slug string, document model.EducationDocument, actor string) (model.EducationModule, error) {
	translation := document.Translations["id"]
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Modules {
			module := &r.store.Modules[index]
			if module.ID != id {
				continue
			}
			if module.DraftRevision != expectedRevision {
				return model.EducationModule{}, ErrEducationConflict
			}
			module.Slug, module.Title, module.Summary = slug, translation.Title, translation.Summary
			module.EstimatedMinutes, module.DraftDocument = document.EstimatedMinutes, document
			module.DraftRevision++
			if module.Status == "published" {
				module.Status = "draft"
			}
			module.UpdatedBy, module.UpdatedAt = actor, time.Now().UTC()
			return *module, nil
		}
		return model.EducationModule{}, ErrEducationNotFound
	}
	count, err := r.db.PsychoeducationModule.Update().
		Where(psychoeducationmodule.IDEQ(id), psychoeducationmodule.DraftRevisionEQ(expectedRevision)).
		SetSlug(slug).SetTitle(translation.Title).SetSummary(translation.Summary).
		SetEstimatedMinutes(document.EstimatedMinutes).SetDraftDocumentJSON(document).
		SetDraftRevision(expectedRevision + 1).SetUpdatedBy(actor).
		SetStatus(psychoeducationmodule.StatusDraft).Save(ctx)
	if err != nil {
		return model.EducationModule{}, err
	}
	if count == 0 {
		if _, getErr := r.GetEducationModuleByID(ctx, id); getErr != nil {
			return model.EducationModule{}, getErr
		}
		return model.EducationModule{}, ErrEducationConflict
	}
	r.RefreshStore(ctx)
	return r.GetEducationModuleByID(ctx, id)
}

func (r *Repository) SetEducationStatus(ctx context.Context, id, status, actor string, publish bool) (model.EducationModule, error) {
	module, err := r.GetEducationModuleByID(ctx, id)
	if err != nil {
		return model.EducationModule{}, err
	}
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Modules {
			current := &r.store.Modules[index]
			if current.ID != id {
				continue
			}
			current.Status, current.UpdatedBy, current.UpdatedAt = status, actor, now
			if publish {
				doc := current.DraftDocument
				current.PublishedDocument = &doc
				current.PublishedRevision = current.DraftRevision
				current.PublishedAt, current.ArchivedAt = &now, nil
			}
			if status == "archived" {
				current.ArchivedAt = &now
			}
			return *current, nil
		}
		return model.EducationModule{}, ErrEducationNotFound
	}
	update := r.db.PsychoeducationModule.UpdateOneID(id).
		SetStatus(psychoeducationmodule.Status(status)).SetUpdatedBy(actor)
	if publish {
		update.SetPublishedDocumentJSON(module.DraftDocument).
			SetPublishedRevision(module.DraftRevision).SetPublishedAt(now).ClearArchivedAt()
	}
	if status == "archived" {
		update.SetArchivedAt(now)
	}
	if _, err = update.Save(ctx); ent.IsNotFound(err) {
		return model.EducationModule{}, ErrEducationNotFound
	}
	if err != nil {
		return model.EducationModule{}, err
	}
	r.RefreshStore(ctx)
	return r.GetEducationModuleByID(ctx, id)
}

func (r *Repository) CreateEducationMedia(ctx context.Context, media model.EducationMedia) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.EducationMedia = append(r.store.EducationMedia, media)
		return nil
	}
	_, err := r.db.EducationMedia.Create().SetID(media.ID).
		SetKind(educationmedia.Kind(media.Kind)).SetPurpose(educationmedia.Purpose(media.Purpose)).
		SetMediaType(educationmedia.MediaType(media.MediaType)).SetMimeType(media.MIMEType).
		SetStorageKey(media.StorageKey).SetExternalURL(media.ExternalURL).
		SetOriginalName(media.OriginalName).SetSizeBytes(media.SizeBytes).
		SetWidth(media.Width).SetHeight(media.Height).SetSha256(media.SHA256).
		SetStatus(educationmedia.Status(media.Status)).SetCreatedBy(media.CreatedBy).Save(ctx)
	return err
}

func (r *Repository) GetEducationMedia(ctx context.Context, id string) (model.EducationMedia, error) {
	if r.db == nil {
		for _, media := range r.store.Snapshot().EducationMedia {
			if media.ID == id {
				return media, nil
			}
		}
		return model.EducationMedia{}, ErrEducationNotFound
	}
	row, err := r.db.EducationMedia.Get(ctx, id)
	if ent.IsNotFound(err) {
		return model.EducationMedia{}, ErrEducationNotFound
	}
	if err != nil {
		return model.EducationMedia{}, err
	}
	return mediaFromEnt(row), nil
}

func (r *Repository) PublishEducationMedia(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.EducationMedia {
			if slices.Contains(ids, r.store.EducationMedia[index].ID) {
				r.store.EducationMedia[index].Status = "published"
			}
		}
		return nil
	}
	_, err := r.db.EducationMedia.Update().Where(educationmedia.IDIn(ids...)).SetStatus(educationmedia.StatusPublished).Save(ctx)
	return err
}

func (r *Repository) GetEducationProgress(ctx context.Context, userID, moduleID string, revision int) (model.EducationProgress, error) {
	if r.db == nil {
		for _, progress := range r.store.Snapshot().EducationProgress {
			if progress.UserID == userID && progress.ModuleID == moduleID && progress.Revision == revision {
				progress.CompletedSectionIDs = nonNilStrings(progress.CompletedSectionIDs)
				progress.OpenedMediaIDs = nonNilStrings(progress.OpenedMediaIDs)
				progress.CorrectCheckIDs = nonNilStrings(progress.CorrectCheckIDs)
				return progress, nil
			}
		}
		return emptyEducationProgress(userID, moduleID, revision), nil
	}
	row, err := r.db.PsychoeducationProgress.Query().Where(
		psychoeducationprogress.UserIDEQ(userID), psychoeducationprogress.ModuleIDEQ(moduleID),
		psychoeducationprogress.RevisionEQ(revision)).Only(ctx)
	if ent.IsNotFound(err) {
		return emptyEducationProgress(userID, moduleID, revision), nil
	}
	if err != nil {
		return model.EducationProgress{}, err
	}
	return progressFromEnt(row), nil
}

func (r *Repository) SaveEducationProgress(ctx context.Context, progress model.EducationProgress) (model.EducationProgress, error) {
	now := time.Now().UTC()
	progress.CompletedSectionIDs = nonNilStrings(progress.CompletedSectionIDs)
	progress.OpenedMediaIDs = nonNilStrings(progress.OpenedMediaIDs)
	progress.CorrectCheckIDs = nonNilStrings(progress.CorrectCheckIDs)
	if progress.ID == "" {
		progress.ID = fmt.Sprintf("edp_%s_%d", progress.ModuleID, progress.Revision) + "_" + progress.UserID
	}
	if progress.CreatedAt.IsZero() {
		progress.CreatedAt = now
	}
	progress.UpdatedAt = now
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.EducationProgress {
			current := &r.store.EducationProgress[index]
			if current.UserID == progress.UserID && current.ModuleID == progress.ModuleID && current.Revision == progress.Revision {
				progress.ID, progress.CreatedAt = current.ID, current.CreatedAt
				*current = progress
				return progress, nil
			}
		}
		r.store.EducationProgress = append(r.store.EducationProgress, progress)
		return progress, nil
	}
	row, err := r.db.PsychoeducationProgress.Query().Where(
		psychoeducationprogress.UserIDEQ(progress.UserID), psychoeducationprogress.ModuleIDEQ(progress.ModuleID),
		psychoeducationprogress.RevisionEQ(progress.Revision)).Only(ctx)
	if ent.IsNotFound(err) {
		created, createErr := r.db.PsychoeducationProgress.Create().SetID(progress.ID).
			SetUserID(progress.UserID).SetModuleID(progress.ModuleID).SetRevision(progress.Revision).
			SetCompletedSectionIds(progress.CompletedSectionIDs).SetOpenedMediaIds(progress.OpenedMediaIDs).
			SetCorrectCheckIds(progress.CorrectCheckIDs).SetProgressPercent(progress.ProgressPercent).
			SetNillableCompletedAt(progress.CompletedAt).Save(ctx)
		if createErr != nil {
			return model.EducationProgress{}, createErr
		}
		return progressFromEnt(created), nil
	}
	if err != nil {
		return model.EducationProgress{}, err
	}
	updated, err := row.Update().SetCompletedSectionIds(progress.CompletedSectionIDs).
		SetOpenedMediaIds(progress.OpenedMediaIDs).SetCorrectCheckIds(progress.CorrectCheckIDs).
		SetProgressPercent(progress.ProgressPercent).SetNillableCompletedAt(progress.CompletedAt).Save(ctx)
	if err != nil {
		return model.EducationProgress{}, err
	}
	return progressFromEnt(updated), nil
}
