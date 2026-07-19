package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoverypracticesession"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryspace"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) ListRecoveryPracticeSessions(ctx context.Context, userID string, since time.Time) ([]model.RecoveryPracticeSession, error) {
	if r.db == nil {
		result := make([]model.RecoveryPracticeSession, 0)
		for _, item := range r.store.Snapshot().RecoveryPracticeSessions {
			if item.UserID == userID && !item.CompletedAt.Before(since) {
				result = append(result, item)
			}
		}
		return result, nil
	}
	rows, err := r.db.RecoveryPracticeSession.Query().Where(
		recoverypracticesession.UserIDEQ(userID),
		recoverypracticesession.CompletedAtGTE(since),
	).Order(ent.Desc(recoverypracticesession.FieldCompletedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]model.RecoveryPracticeSession, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.RecoveryPracticeSession{
			ID: row.ID, UserID: row.UserID, PracticeKind: row.PracticeKind.String(),
			DurationSeconds: row.DurationSeconds, Feedback: recoveryFeedbackValue(row.Feedback),
			CompletedAt: row.CompletedAt, CreatedAt: row.CreatedAt,
		})
	}
	return result, nil
}

func (r *Repository) SaveRecoveryPracticeSession(ctx context.Context, item model.RecoveryPracticeSession) (model.RecoveryPracticeSession, error) {
	if r.db == nil {
		r.store.Lock()
		r.store.RecoveryPracticeSessions = append(r.store.RecoveryPracticeSessions, item)
		r.store.Unlock()
		return item, nil
	}
	var feedback *recoverypracticesession.Feedback
	if item.Feedback != "" {
		value := recoverypracticesession.Feedback(item.Feedback)
		feedback = &value
	}
	row, err := r.db.RecoveryPracticeSession.Create().
		SetID(item.ID).
		SetUserID(item.UserID).
		SetPracticeKind(recoverypracticesession.PracticeKind(item.PracticeKind)).
		SetDurationSeconds(item.DurationSeconds).
		SetNillableFeedback(feedback).
		SetCompletedAt(item.CompletedAt).
		Save(ctx)
	if err != nil {
		return model.RecoveryPracticeSession{}, err
	}
	r.RefreshStore(ctx)
	return model.RecoveryPracticeSession{
		ID: row.ID, UserID: row.UserID, PracticeKind: row.PracticeKind.String(),
		DurationSeconds: row.DurationSeconds, Feedback: recoveryFeedbackValue(row.Feedback),
		CompletedAt: row.CompletedAt, CreatedAt: row.CreatedAt,
	}, nil
}

func (r *Repository) DeleteExpiredRecoveryPracticeSessions(ctx context.Context, userID string, before time.Time) error {
	if r.db == nil {
		r.store.Lock()
		r.store.RecoveryPracticeSessions = filterSlice(r.store.RecoveryPracticeSessions, func(item model.RecoveryPracticeSession) bool {
			return item.UserID != userID || !item.CompletedAt.Before(before)
		})
		r.store.Unlock()
		return nil
	}
	_, err := r.db.RecoveryPracticeSession.Delete().Where(
		recoverypracticesession.UserIDEQ(userID),
		recoverypracticesession.CompletedAtLT(before),
	).Exec(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) GetRecoverySpace(ctx context.Context, userID string) (model.RecoverySpace, bool, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().RecoverySpaces {
			if item.UserID == userID {
				return item, true, nil
			}
		}
		return model.RecoverySpace{}, false, nil
	}
	row, err := r.db.RecoverySpace.Query().Where(recoveryspace.UserIDEQ(userID)).Only(ctx)
	if ent.IsNotFound(err) {
		return model.RecoverySpace{}, false, nil
	}
	if err != nil {
		return model.RecoverySpace{}, false, err
	}
	return recoverySpaceModel(row), true, nil
}

func (r *Repository) SaveRecoverySpace(ctx context.Context, item model.RecoverySpace) (model.RecoverySpace, error) {
	if r.db == nil {
		r.store.Lock()
		for index := range r.store.RecoverySpaces {
			if r.store.RecoverySpaces[index].UserID == item.UserID {
				item.CreatedAt = r.store.RecoverySpaces[index].CreatedAt
				r.store.RecoverySpaces[index] = item
				r.store.Unlock()
				return item, nil
			}
		}
		r.store.RecoverySpaces = append(r.store.RecoverySpaces, item)
		r.store.Unlock()
		return item, nil
	}
	row, err := r.db.RecoverySpace.Query().Where(recoveryspace.UserIDEQ(item.UserID)).Only(ctx)
	if ent.IsNotFound(err) {
		row, err = r.db.RecoverySpace.Create().
			SetID(item.ID).
			SetUserID(item.UserID).
			SetTheme(recoveryspace.Theme(item.Theme)).
			SetUnlockedItemsJSON(item.UnlockedItems).
			SetPlacedItemsJSON(item.PlacedItems).
			SetUnlockRuleVersion(item.UnlockRuleVersion).
			Save(ctx)
	} else if err == nil {
		row, err = row.Update().
			SetTheme(recoveryspace.Theme(item.Theme)).
			SetUnlockedItemsJSON(item.UnlockedItems).
			SetPlacedItemsJSON(item.PlacedItems).
			SetUnlockRuleVersion(item.UnlockRuleVersion).
			Save(ctx)
	}
	if err != nil {
		return model.RecoverySpace{}, err
	}
	r.RefreshStore(ctx)
	return recoverySpaceModel(row), nil
}

func recoveryFeedbackValue(value *recoverypracticesession.Feedback) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func recoverySpaceModel(row *ent.RecoverySpace) model.RecoverySpace {
	return model.RecoverySpace{
		ID: row.ID, UserID: row.UserID, Theme: row.Theme.String(),
		UnlockedItems: row.UnlockedItemsJSON, PlacedItems: row.PlacedItemsJSON,
		UnlockRuleVersion: row.UnlockRuleVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *Repository) RecoveryUnlockEvidence(ctx context.Context, userID string) model.RecoveryUnlockEvidence {
	if r.db != nil {
		r.RefreshStore(ctx)
	}
	snapshot := r.store.Snapshot()
	evidence := model.RecoveryUnlockEvidence{PracticeKinds: map[string]bool{}}
	days := map[string]bool{}
	for _, item := range snapshot.RecoveryPracticeSessions {
		if item.UserID == userID {
			evidence.PracticeKinds[item.PracticeKind] = true
			days[item.CompletedAt.UTC().Format("2006-01-02")] = true
		}
	}
	for _, item := range snapshot.JournalEntries {
		if item.UserID == userID {
			evidence.HasFocusJournal = evidence.HasFocusJournal || item.IsFocus
			days[item.CreatedAt.UTC().Format("2006-01-02")] = true
		}
	}
	for _, item := range snapshot.RecoveryRecords {
		if item.UserID == userID && item.Kind == "weekly_review" {
			evidence.HasWeeklyReview = true
			days[item.RecordDate] = true
		}
	}
	for _, item := range snapshot.CheckIns {
		if item.UserID == userID {
			days[item.CreatedAt.UTC().Format("2006-01-02")] = true
		}
	}
	for _, item := range snapshot.Missions {
		if item.UserID == userID && (item.Mission1 || item.Mission2 || item.Mission3 || item.Mission4 || item.Mission5) {
			days[item.Date] = true
		}
	}
	evidence.ActiveDays = len(days)
	return evidence
}
