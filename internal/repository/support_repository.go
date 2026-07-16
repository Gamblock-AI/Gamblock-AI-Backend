package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/datarequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) GetSupportCases(ctx context.Context) ([]model.SupportCase, error) {
	return r.getSupportCases(ctx, "")
}

func (r *Repository) GetSupportCasesForUser(ctx context.Context, userID string) ([]model.SupportCase, error) {
	return r.getSupportCases(ctx, userID)
}

func (r *Repository) getSupportCases(ctx context.Context, userID string) ([]model.SupportCase, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.SupportCase
		for _, c := range snapshot.SupportCases {
			if userID != "" && c.UserID != userID {
				continue
			}
			list = append(list, model.SupportCase{
				ID:        c.ID,
				UserID:    c.UserID,
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

	query := r.db.SupportCase.Query()
	if userID != "" {
		query.Where(supportcase.UserID(userID))
	}
	rows, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.SupportCase
	for _, item := range rows {
		owner := ""
		if userID == "" {
			owner = item.UserID
		}
		list = append(list, model.SupportCase{
			ID:        item.ID,
			UserID:    item.UserID,
			Title:     item.Summary,
			Type:      item.Type.String(),
			Status:    item.Status.String(),
			Priority:  item.Priority.String(),
			Owner:     owner,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) CreateSupportCase(ctx context.Context, id, userID, title, cType, priorityVal string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.SupportCases = append(r.store.SupportCases, store.SupportCase{
			ID: id, UserID: userID, Title: title, Type: cType,
			Priority: priorityVal, Status: "open",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		})
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
			if dr.UserID != userID {
				continue
			}
			list = append(list, model.DataRequest{
				ID:        dr.ID,
				UserID:    dr.UserID,
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
			UserID:    item.UserID,
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
		r.store.Lock()
		defer r.store.Unlock()
		r.store.DataRequests = append(r.store.DataRequests, store.DataRequest{
			ID: id, UserID: userID, Title: humanDataRequestTitle(reqType), Type: reqType, Status: "pending",
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		})
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
