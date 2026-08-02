package db

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

// loadLearningHubStore hydrates the in-memory fallback/cache for the Learning
// Hub. It is best-effort so older databases without the supporting tables can
// still boot and rely on direct ent queries after migration.
func loadLearningHubStore(ctx context.Context, client *ent.Client, out *store.Store) error {
	institutions, err := client.Institution.Query().All(ctx)
	if err != nil {
		return err
	}
	clusters, err := client.LearningCluster.Query().All(ctx)
	if err != nil {
		return err
	}
	programs, err := client.AcademicProgram.Query().All(ctx)
	if err != nil {
		return err
	}
	items, err := client.LearningItem.Query().All(ctx)
	if err != nil {
		return err
	}
	revisions, err := client.LearningRevision.Query().All(ctx)
	if err != nil {
		return err
	}
	progress, err := client.LearningProgress.Query().All(ctx)
	if err != nil {
		return err
	}
	grants, err := client.ExperienceGrant.Query().All(ctx)
	if err != nil {
		return err
	}

	for _, row := range institutions {
		out.Institutions = append(out.Institutions, model.Institution{ID: row.ID, Slug: row.Slug, Name: row.Name, Status: row.Status.String(), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	for _, row := range clusters {
		cluster := model.AdminLearningCluster{LearningCluster: model.LearningCluster{ID: row.ID, Slug: row.Slug, Title: row.TitleID, Description: row.DescriptionID, SortOrder: row.SortOrder}, TitleID: row.TitleID, TitleEN: row.TitleEn, DescriptionID: row.DescriptionID, DescriptionEN: row.DescriptionEn, Active: row.Active}
		out.AdminLearningClusters = append(out.AdminLearningClusters, cluster)
		if row.Active {
			out.LearningClusters = append(out.LearningClusters, cluster.LearningCluster)
		}
	}
	for _, row := range programs {
		program := model.AdminAcademicProgram{AcademicProgram: model.AcademicProgram{ID: row.ID, InstitutionID: row.InstitutionID, Slug: row.Slug, Name: row.Name, Degree: row.Degree, PrimaryClusterSlug: row.PrimaryClusterSlug, SortOrder: row.SortOrder}, Active: row.Active}
		out.AdminAcademicPrograms = append(out.AdminAcademicPrograms, program)
		if row.Active {
			out.AcademicPrograms = append(out.AcademicPrograms, program.AcademicProgram)
		}
	}
	for _, row := range items {
		admin := model.AdminLearningItem{LearningItem: model.LearningItem{ID: row.ID, Slug: row.Slug, Kind: row.Kind.String(), Title: row.TitleID, Summary: row.SummaryID}, TitleID: row.TitleID, TitleEN: row.TitleEn, SummaryID: row.SummaryID, SummaryEN: row.SummaryEn, Status: row.Status.String(), DraftRevision: row.DraftRevision, PublishedRevision: row.PublishedRevision, DraftDocument: row.DocumentJSON, PublishedAt: row.PublishedAt, ArchivedAt: row.ArchivedAt, CreatedBy: row.CreatedBy, UpdatedBy: row.UpdatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
		out.AdminLearningItems = append(out.AdminLearningItems, admin)
		if row.Status.String() == "published" {
			out.LearningItems = append(out.LearningItems, admin.LearningItem)
		}
	}
	for _, row := range revisions {
		revision := model.LearningRevision{ID: row.ID, ItemID: row.ItemID, Revision: row.Revision, Document: row.DocumentJSON, Kind: row.Kind.String(), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}
		out.LearningRevisions = append(out.LearningRevisions, revision)
		if revision.Kind == "published" {
			for index := range out.AdminLearningItems {
				if out.AdminLearningItems[index].ID == revision.ItemID && revision.Revision >= out.AdminLearningItems[index].PublishedRevision {
					out.AdminLearningItems[index].PublishedDocument = revision.Document
				}
			}
		}
	}
	for _, row := range progress {
		reflection, outcome := "", ""
		if row.ReflectionEncrypted != nil {
			reflection = *row.ReflectionEncrypted
		}
		if row.OutcomeEncrypted != nil {
			outcome = *row.OutcomeEncrypted
		}
		out.LearningProgress = append(out.LearningProgress, model.LearningProgress{UserID: row.UserID, ItemID: row.ItemID, State: row.State.String(), ReflectionEncrypted: reflection, OutcomeEncrypted: outcome, CompletedAt: row.CompletedAt})
	}
	for _, row := range grants {
		out.ExperienceGrants = append(out.ExperienceGrants, model.ExperienceGrant{ID: row.ID, UserID: row.UserID, SourceKind: row.SourceKind, SourceID: row.SourceID, GrantDate: row.GrantDate, Amount: row.Amount, IdempotencyKey: row.IdempotencyKey, CreatedAt: row.CreatedAt})
	}
	return nil
}
