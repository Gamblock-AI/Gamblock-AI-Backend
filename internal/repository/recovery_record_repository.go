package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryrecord"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) ListRecoveryRecords(ctx context.Context, userID string, since time.Time) ([]model.RecoveryRecord, error) {
	var result []model.RecoveryRecord
	if r.db == nil {
		for _, item := range r.store.Snapshot().RecoveryRecords {
			if item.UserID == userID && !item.CreatedAt.Before(since) {
				result = append(result, item)
			}
		}
		return result, nil
	}
	rows, err := r.db.RecoveryRecord.Query().Where(
		recoveryrecord.UserIDEQ(userID),
		recoveryrecord.CreatedAtGTE(since),
	).Order(ent.Desc(recoveryrecord.FieldRecordDate), ent.Desc(recoveryrecord.FieldUpdatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		result = append(result, model.RecoveryRecord{
			ID: item.ID, UserID: item.UserID, Kind: item.Kind.String(), RecordDate: item.RecordDate,
			Metadata: item.MetadataJSON, Content: value(item.ContentEncrypted), Status: item.Status.String(),
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return result, nil
}

func (r *Repository) SaveRecoveryRecord(ctx context.Context, item model.RecoveryRecord) (model.RecoveryRecord, error) {
	if r.db == nil {
		r.store.Lock()
		for i := range r.store.RecoveryRecords {
			if r.store.RecoveryRecords[i].ID == item.ID && r.store.RecoveryRecords[i].UserID == item.UserID {
				item.CreatedAt = r.store.RecoveryRecords[i].CreatedAt
				r.store.RecoveryRecords[i] = item
				r.store.Unlock()
				return item, nil
			}
		}
		r.store.RecoveryRecords = append(r.store.RecoveryRecords, item)
		r.store.Unlock()
		return item, nil
	}
	row, err := r.db.RecoveryRecord.Query().Where(
		recoveryrecord.IDEQ(item.ID), recoveryrecord.UserIDEQ(item.UserID),
	).Only(ctx)
	if err == nil {
		row, err = row.Update().SetKind(recoveryrecord.Kind(item.Kind)).SetRecordDate(item.RecordDate).
			SetMetadataJSON(item.Metadata).SetNillableContentEncrypted(optional(item.Content)).
			SetStatus(recoveryrecord.Status(item.Status)).Save(ctx)
	} else if ent.IsNotFound(err) {
		row, err = r.db.RecoveryRecord.Create().SetID(item.ID).SetUserID(item.UserID).
			SetKind(recoveryrecord.Kind(item.Kind)).SetRecordDate(item.RecordDate).
			SetMetadataJSON(item.Metadata).SetNillableContentEncrypted(optional(item.Content)).
			SetStatus(recoveryrecord.Status(item.Status)).Save(ctx)
	}
	if err != nil {
		return model.RecoveryRecord{}, err
	}
	r.RefreshStore(ctx)
	return model.RecoveryRecord{
		ID: row.ID, UserID: row.UserID, Kind: row.Kind.String(), RecordDate: row.RecordDate,
		Metadata: row.MetadataJSON, Content: value(row.ContentEncrypted), Status: row.Status.String(),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *Repository) DeleteExpiredRecoveryRecords(ctx context.Context, userID string, before time.Time) error {
	if r.db == nil {
		r.store.Lock()
		filtered := r.store.RecoveryRecords[:0]
		for _, item := range r.store.RecoveryRecords {
			if item.UserID == userID && item.CreatedAt.Before(before) {
				continue
			}
			filtered = append(filtered, item)
		}
		r.store.RecoveryRecords = filtered
		r.store.Unlock()
		return nil
	}
	_, err := r.db.RecoveryRecord.Delete().Where(
		recoveryrecord.UserIDEQ(userID), recoveryrecord.CreatedAtLT(before),
	).Exec(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}
