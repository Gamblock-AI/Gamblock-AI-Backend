package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationmodule"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetEducationModules(ctx context.Context) ([]model.EducationModule, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.EducationModule
		for _, m := range snapshot.Modules {
			list = append(list, model.EducationModule{
				ID:               m.ID,
				Slug:             m.Slug,
				Title:            m.Title,
				Summary:          m.Summary,
				BodyMarkdown:     m.BodyMarkdown,
				EstimatedMinutes: m.EstimatedMinutes,
				Progress:         m.Progress,
				Status:           m.Status,
				CreatedAt:        m.CreatedAt,
				UpdatedAt:        m.UpdatedAt,
			})
		}
		return list, nil
	}

	rows, err := r.db.PsychoeducationModule.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.EducationModule
	for _, item := range rows {
		list = append(list, model.EducationModule{
			ID:               item.ID,
			Slug:             item.Slug,
			Title:            item.Title,
			Summary:          item.Summary,
			BodyMarkdown:     item.BodyMarkdown,
			EstimatedMinutes: item.EstimatedMinutes,
			Progress:         moduleProgress(item.Slug),
			Status:           item.Status.String(),
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) GetEducationModuleBySlug(ctx context.Context, slug string) (model.EducationModule, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, m := range snapshot.Modules {
			if strings.EqualFold(m.Slug, slug) {
				return model.EducationModule{
					ID:               m.ID,
					Slug:             m.Slug,
					Title:            m.Title,
					Summary:          m.Summary,
					BodyMarkdown:     m.BodyMarkdown,
					EstimatedMinutes: m.EstimatedMinutes,
					Progress:         m.Progress,
					Status:           m.Status,
					CreatedAt:        m.CreatedAt,
					UpdatedAt:        m.UpdatedAt,
				}, nil
			}
		}
		return model.EducationModule{}, fmt.Errorf("module not found")
	}

	row, err := r.db.PsychoeducationModule.Query().Where(psychoeducationmodule.SlugEQ(slug)).Only(ctx)
	if err != nil {
		return model.EducationModule{}, err
	}
	return model.EducationModule{
		ID:               row.ID,
		Slug:             row.Slug,
		Title:            row.Title,
		Summary:          row.Summary,
		BodyMarkdown:     row.BodyMarkdown,
		EstimatedMinutes: row.EstimatedMinutes,
		Progress:         moduleProgress(row.Slug),
		Status:           row.Status.String(),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

func (r *Repository) CreateEducationModule(ctx context.Context, m model.EducationModule) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.PsychoeducationModule.Create().
		SetID(m.ID).
		SetSlug(m.Slug).
		SetTitle(m.Title).
		SetSummary(m.Summary).
		SetBodyMarkdown(m.BodyMarkdown).
		SetEstimatedMinutes(m.EstimatedMinutes).
		SetStatus(psychoeducationmodule.Status(m.Status)).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
