package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/datarequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetSupportCases(ctx context.Context) ([]model.SupportCase, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.SupportCase
		for _, c := range snapshot.SupportCases {
			list = append(list, model.SupportCase{
				ID:        c.ID,
				Title:     c.Title,
				Type:      c.Type,
				Status:    c.Status,
				Priority:  c.Priority,
				Owner:     c.Owner,
				CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt,
			})
		}
		return list, nil
	}

	rows, err := r.db.SupportCase.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.SupportCase
	for _, item := range rows {
		list = append(list, model.SupportCase{
			ID:        item.ID,
			Title:     item.Summary,
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			Priority:  item.Priority.String(),
			Owner:     "Alfian",
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) CreateSupportCase(ctx context.Context, id, userID, title, cType, priorityVal string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.SupportCase.Create().
		SetID(id).
		SetUserID(userID).
		SetType(supportcase.Type(cType)).
		SetPriority(supportcase.Priority(priorityVal)).
		SetSummary(title).
		SetStatus(supportcase.StatusOpen).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) GetDataRequests(ctx context.Context, userID string) ([]model.DataRequest, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.DataRequest
		for _, dr := range snapshot.DataRequests {
			list = append(list, model.DataRequest{
				ID:        dr.ID,
				Title:     dr.Title,
				Type:      dr.Type,
				Status:    dr.Status,
				CreatedAt: dr.CreatedAt,
				UpdatedAt: dr.UpdatedAt,
			})
		}
		return list, nil
	}

	rows, err := r.db.DataRequest.Query().Where(datarequest.UserID(userID)).All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.DataRequest
	for _, item := range rows {
		list = append(list, model.DataRequest{
			ID:        item.ID,
			Title:     humanDataRequestTitle(item.Type.String()),
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			CreatedAt: item.RequestedAt,
			UpdatedAt: time.Time{},
		})
	}
	return list, nil
}

func (r *Repository) CreateDataRequest(ctx context.Context, id, userID, reqType string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.DataRequest.Create().
		SetID(id).
		SetUserID(userID).
		SetType(datarequest.Type(reqType)).
		SetStatus(datarequest.StatusPending).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
