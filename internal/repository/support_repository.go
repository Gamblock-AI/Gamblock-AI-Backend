package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/datarequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportmessage"
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
				ID:          c.ID,
				UserID:      c.UserID,
				Title:       c.Title,
				Type:        c.Type,
				Status:      c.Status,
				Priority:    c.Priority,
				Owner:       c.Owner,
				Impact:      c.Impact,
				UnreadCount: c.UnreadCount,
				ResolvedAt:  c.ResolvedAt,
				ClosedAt:    c.ClosedAt,
				CreatedAt:   c.CreatedAt,
				UpdatedAt:   c.UpdatedAt,
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
		list = append(list, model.SupportCase{
			ID:         item.ID,
			UserID:     item.UserID,
			Title:      item.Summary,
			Type:       item.Type.String(),
			Status:     item.Status.String(),
			Priority:   item.Priority.String(),
			Owner:      value(item.AssignedOperatorID),
			Impact:     item.Impact,
			ResolvedAt: item.ResolvedAt,
			ClosedAt:   item.ClosedAt,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
		})
	}
	return list, nil
}

func (r *Repository) GetAdminSupportCasesPaginated(ctx context.Context, adminID string, query model.PaginationQuery) (model.PaginatedList[model.SupportCase], error) {
	page, limit, offset := query.Normalize(10)
	status := strings.TrimSpace(query.Status)
	priority := strings.TrimSpace(query.Priority)
	search := strings.ToLower(strings.TrimSpace(query.Query))

	if r.db == nil {
		snapshot := r.store.Snapshot()
		var filtered []model.SupportCase
		for _, c := range snapshot.SupportCases {
			if adminID != "" && c.UserID == adminID {
				continue
			}
			if status != "" && c.Status != status {
				continue
			}
			if priority != "" && c.Priority != priority {
				continue
			}
			if search != "" {
				titleMatch := strings.Contains(strings.ToLower(c.Title), search)
				idMatch := strings.Contains(strings.ToLower(c.ID), search)
				if !titleMatch && !idMatch {
					continue
				}
			}
			filtered = append(filtered, model.SupportCase{
				ID:          c.ID,
				UserID:      c.UserID,
				Title:       c.Title,
				Type:        c.Type,
				Status:      c.Status,
				Priority:    c.Priority,
				Owner:       c.Owner,
				Impact:      c.Impact,
				UnreadCount: c.UnreadCount,
				ResolvedAt:  c.ResolvedAt,
				ClosedAt:    c.ClosedAt,
				CreatedAt:   c.CreatedAt,
				UpdatedAt:   c.UpdatedAt,
			})
		}
		total := len(filtered)
		start := offset
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		paged := filtered[start:end]
		return model.NewPaginatedList(paged, total, page, limit), nil
	}

	qb := r.db.SupportCase.Query()
	if adminID != "" {
		qb = qb.Where(supportcase.UserIDNEQ(adminID))
	}
	if status != "" {
		qb = qb.Where(supportcase.StatusEQ(supportcase.Status(status)))
	}
	if priority != "" {
		qb = qb.Where(supportcase.PriorityEQ(supportcase.Priority(priority)))
	}
	if search != "" {
		qb = qb.Where(supportcase.Or(
			supportcase.SummaryContainsFold(search),
			supportcase.IDContainsFold(search),
		))
	}

	total, err := qb.Count(ctx)
	if err != nil {
		return model.PaginatedList[model.SupportCase]{}, err
	}

	rows, err := qb.Order(ent.Desc(supportcase.FieldUpdatedAt)).Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return model.PaginatedList[model.SupportCase]{}, err
	}

	var list []model.SupportCase
	for _, item := range rows {
		list = append(list, model.SupportCase{
			ID:         item.ID,
			UserID:     item.UserID,
			Title:      item.Summary,
			Type:       item.Type.String(),
			Status:     item.Status.String(),
			Priority:   item.Priority.String(),
			Owner:      value(item.AssignedOperatorID),
			Impact:     item.Impact,
			ResolvedAt: item.ResolvedAt,
			ClosedAt:   item.ClosedAt,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
		})
	}
	return model.NewPaginatedList(list, total, page, limit), nil
}

func (r *Repository) ClaimSupportCase(ctx context.Context, caseID, operatorID, reason string, now time.Time) (model.SupportCase, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.SupportCases {
			item := &r.store.SupportCases[index]
			if item.ID != caseID {
				continue
			}
			if item.Status == "resolved" || item.Status == "closed" || (item.Owner != "" && item.Owner != operatorID) {
				return model.SupportCase{}, fmt.Errorf("support case is already assigned or closed")
			}
			item.Owner, item.UpdatedAt = operatorID, now
			return *item, nil
		}
		return model.SupportCase{}, fmt.Errorf("support case not found")
	}
	count, err := r.db.SupportCase.Update().Where(
		supportcase.IDEQ(caseID), supportcase.AssignedOperatorIDIsNil(),
		supportcase.StatusNotIn(supportcase.StatusResolved, supportcase.StatusClosed),
	).SetAssignedOperatorID(operatorID).Save(ctx)
	if err != nil {
		return model.SupportCase{}, err
	}
	if count == 0 {
		current, getErr := r.db.SupportCase.Get(ctx, caseID)
		if getErr != nil || current.AssignedOperatorID == nil || *current.AssignedOperatorID != operatorID {
			return model.SupportCase{}, fmt.Errorf("support case is already assigned or closed")
		}
	}
	_, err = r.db.SupportActionAudit.Create().SetID("sca_" + operatorID + "_" + caseID + "_claim_" + fmt.Sprint(now.UnixNano())).
		SetSupportCaseID(caseID).SetOperatorID(operatorID).SetAction("claim").SetReason(reason).
		SetBeforeJSON(map[string]any{"assigned": false}).SetAfterJSON(map[string]any{"assigned": true}).Save(ctx)
	if err != nil && !ent.IsConstraintError(err) {
		return model.SupportCase{}, err
	}
	r.RefreshStore(ctx)
	return r.GetSupportCaseDetail(ctx, caseID)
}

func (r *Repository) ReleaseSupportCase(ctx context.Context, caseID, operatorID, reason string, now time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.SupportCases {
			item := &r.store.SupportCases[index]
			if item.ID == caseID && item.Owner == operatorID {
				item.Owner, item.UpdatedAt = "", now
				return nil
			}
		}
		return fmt.Errorf("support case is not assigned to operator")
	}
	count, err := r.db.SupportCase.Update().Where(supportcase.IDEQ(caseID), supportcase.AssignedOperatorIDEQ(operatorID)).ClearAssignedOperatorID().Save(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("support case is not assigned to operator")
	}
	_, err = r.db.SupportActionAudit.Create().SetID("sca_" + operatorID + "_" + caseID + "_release_" + fmt.Sprint(now.UnixNano())).
		SetSupportCaseID(caseID).SetOperatorID(operatorID).SetAction("release").SetReason(reason).
		SetBeforeJSON(map[string]any{"assigned": true}).SetAfterJSON(map[string]any{"assigned": false}).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) CreateSupportCase(ctx context.Context, id, userID, title, cType, priorityVal string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.SupportCases = append(r.store.SupportCases, store.SupportCase{
			ID: id, UserID: userID, Title: title, Type: cType,
			Priority: priorityVal, Status: "waiting_support", Impact: "blocked",
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
		SetStatus(supportcase.StatusWaitingSupport).
		Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) CreateSupportCaseWithMessage(ctx context.Context, supportCase model.SupportCase, encryptedDetail string) (model.SupportCase, error) {
	if r.db == nil {
		r.store.Lock()
		r.store.SupportCases = append(r.store.SupportCases, supportCase)
		message := store.SupportMessage{
			ID: "msg_" + supportCase.ID, SupportCaseID: supportCase.ID,
			AuthorID: supportCase.UserID, AuthorRole: "requester", Content: encryptedDetail,
			CreatedAt: supportCase.CreatedAt,
		}
		r.store.SupportMessages = append(r.store.SupportMessages, message)
		r.store.Unlock()
		return supportCase, nil
	}
	_, err := r.db.SupportCase.Create().SetID(supportCase.ID).SetUserID(supportCase.UserID).
		SetType(supportcase.Type(supportCase.Type)).SetPriority(supportcase.Priority(supportCase.Priority)).
		SetSummary(supportCase.Title).SetImpact(supportCase.Impact).SetStatus(supportcase.StatusWaitingSupport).Save(ctx)
	if err != nil {
		return model.SupportCase{}, err
	}
	_, err = r.db.SupportMessage.Create().SetID("msg_" + supportCase.ID).
		SetSupportCaseID(supportCase.ID).SetAuthorID(supportCase.UserID).
		SetAuthorRole(supportmessage.AuthorRoleRequester).SetContentEncrypted(encryptedDetail).Save(ctx)
	if err != nil {
		return model.SupportCase{}, err
	}
	r.RefreshStore(ctx)
	return supportCase, nil
}

func (r *Repository) GetSupportCaseDetail(ctx context.Context, caseID string) (model.SupportCase, error) {
	var result model.SupportCase
	if r.db == nil {
		for _, item := range r.store.Snapshot().SupportCases {
			if item.ID == caseID {
				result = item
				break
			}
		}
		if result.ID == "" {
			return model.SupportCase{}, fmt.Errorf("support case not found")
		}
	} else {
		item, err := r.db.SupportCase.Get(ctx, caseID)
		if err != nil {
			return model.SupportCase{}, fmt.Errorf("support case not found")
		}
		result = model.SupportCase{
			ID: item.ID, UserID: item.UserID, Title: item.Summary, Type: item.Type.String(),
			Status: item.Status.String(), Priority: item.Priority.String(), Impact: item.Impact,
			Owner: value(item.AssignedOperatorID), ResolvedAt: item.ResolvedAt, ClosedAt: item.ClosedAt,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		}
	}
	if result.UserID != "" {
		if user, ok := r.UserByID(ctx, result.UserID); ok {
			result.UserName = user.DisplayName
			result.UserAvatarURL = user.AvatarURL
		}
	}
	messages, err := r.ListSupportMessages(ctx, caseID)
	if err != nil {
		return model.SupportCase{}, err
	}
	result.Messages = messages
	for _, message := range messages {
		if message.AuthorRole == "admin" && message.ReadAt == nil {
			result.UnreadCount++
		}
	}
	return result, nil
}

func (r *Repository) ListSupportMessages(ctx context.Context, caseID string) ([]model.SupportMessage, error) {
	var result []model.SupportMessage
	if r.db == nil {
		for _, item := range r.store.Snapshot().SupportMessages {
			if item.SupportCaseID == caseID {
				msg := item
				if msg.AuthorID != "" {
					if u, ok := r.UserByID(ctx, msg.AuthorID); ok {
						msg.AuthorName = u.DisplayName
						msg.AuthorAvatarURL = u.AvatarURL
					}
				}
				result = append(result, msg)
			}
		}
		return result, nil
	}
	rows, err := r.db.SupportMessage.Query().Where(supportmessage.SupportCaseIDEQ(caseID)).
		Order(ent.Asc(supportmessage.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range rows {
		msg := model.SupportMessage{
			ID: item.ID, SupportCaseID: item.SupportCaseID, AuthorID: item.AuthorID,
			AuthorRole: item.AuthorRole.String(), Content: item.ContentEncrypted,
			ReadAt: item.ReadAt, CreatedAt: item.CreatedAt,
		}
		if item.AuthorID != "" {
			if u, ok := r.UserByID(ctx, item.AuthorID); ok {
				msg.AuthorName = u.DisplayName
				msg.AuthorAvatarURL = u.AvatarURL
			}
		}
		result = append(result, msg)
	}
	return result, nil
}

func (r *Repository) AddSupportMessage(ctx context.Context, item model.SupportMessage, nextStatus string) (model.SupportMessage, error) {
	if r.db == nil {
		r.store.Lock()
		r.store.SupportMessages = append(r.store.SupportMessages, item)
		for i := range r.store.SupportCases {
			if r.store.SupportCases[i].ID == item.SupportCaseID {
				r.store.SupportCases[i].Status = nextStatus
				r.store.SupportCases[i].UpdatedAt = item.CreatedAt
			}
		}
		r.store.Unlock()
		return item, nil
	}
	row, err := r.db.SupportMessage.Create().SetID(item.ID).SetSupportCaseID(item.SupportCaseID).
		SetAuthorID(item.AuthorID).SetAuthorRole(supportmessage.AuthorRole(item.AuthorRole)).
		SetContentEncrypted(item.Content).Save(ctx)
	if err != nil {
		return model.SupportMessage{}, err
	}
	_, err = r.db.SupportCase.UpdateOneID(item.SupportCaseID).SetStatus(supportcase.Status(nextStatus)).Save(ctx)
	if err != nil {
		return model.SupportMessage{}, err
	}
	r.RefreshStore(ctx)
	respMsg := model.SupportMessage{
		ID: row.ID, SupportCaseID: row.SupportCaseID, AuthorID: row.AuthorID,
		AuthorRole: row.AuthorRole.String(), Content: row.ContentEncrypted,
		CreatedAt: row.CreatedAt,
	}
	if row.AuthorID != "" {
		if u, ok := r.UserByID(ctx, row.AuthorID); ok {
			respMsg.AuthorName = u.DisplayName
			respMsg.AuthorAvatarURL = u.AvatarURL
		}
	}
	return respMsg, nil
}

func (r *Repository) TransitionSupportCase(ctx context.Context, caseID, status, operatorID string, now time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.SupportCases {
			item := &r.store.SupportCases[i]
			if item.ID != caseID {
				continue
			}
			item.Status = status
			item.Owner = operatorID
			item.UpdatedAt = now
			if status == "resolved" {
				item.ResolvedAt = &now
			}
			if status == "closed" {
				item.ClosedAt = &now
			}
			return nil
		}
		return fmt.Errorf("support case not found")
	}
	update := r.db.SupportCase.UpdateOneID(caseID).SetStatus(supportcase.Status(status))
	if operatorID != "" {
		update.SetAssignedOperatorID(operatorID)
	}
	if status == "resolved" {
		update.SetResolvedAt(now)
	}
	if status == "closed" {
		update.SetClosedAt(now)
	}
	if status == "waiting_support" {
		update.ClearResolvedAt().ClearClosedAt()
	}
	_, err := update.Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) GetDataRequests(ctx context.Context, userID string) ([]model.DataRequest, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		var list []model.DataRequest
		for _, dr := range snapshot.DataRequests {
			if dr.UserID != userID {
				continue
			}
			list = append(list, dr)
		}
		return list, nil
	}

	rows, err := r.db.DataRequest.Query().Where(datarequest.UserID(userID)).All(ctx)
	if err != nil {
		return nil, err
	}

	var list []model.DataRequest
	for _, item := range rows {
		list = append(list, dataRequestFromEnt(item))
	}
	return list, nil
}

func (r *Repository) GetAllDataRequests(ctx context.Context) ([]model.DataRequest, error) {
	if r.db == nil {
		return append([]model.DataRequest(nil), r.store.Snapshot().DataRequests...), nil
	}
	rows, err := r.db.DataRequest.Query().Order(ent.Desc(datarequest.FieldRequestedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.DataRequest, 0, len(rows))
	for _, row := range rows {
		items = append(items, dataRequestFromEnt(row))
	}
	return items, nil
}

func (r *Repository) GetAllDataRequestsPaginated(ctx context.Context, query model.PaginationQuery) (model.PaginatedList[model.DataRequest], error) {
	page, limit, offset := query.Normalize(10)
	status := strings.TrimSpace(query.Status)
	reqType := strings.TrimSpace(query.Type)

	if r.db == nil {
		all := r.store.Snapshot().DataRequests
		var filtered []model.DataRequest
		for _, req := range all {
			if status != "" && req.Status != status {
				continue
			}
			if reqType != "" && req.Type != reqType {
				continue
			}
			filtered = append(filtered, req)
		}
		total := len(filtered)
		start := offset
		if start > total {
			start = total
		}
		end := start + limit
		if end > total {
			end = total
		}
		paged := filtered[start:end]
		return model.NewPaginatedList(paged, total, page, limit), nil
	}

	qb := r.db.DataRequest.Query()
	if status != "" {
		qb = qb.Where(datarequest.StatusEQ(datarequest.Status(status)))
	}
	if reqType != "" {
		qb = qb.Where(datarequest.TypeEQ(datarequest.Type(reqType)))
	}

	total, err := qb.Count(ctx)
	if err != nil {
		return model.PaginatedList[model.DataRequest]{}, err
	}

	rows, err := qb.Order(ent.Desc(datarequest.FieldRequestedAt)).Limit(limit).Offset(offset).All(ctx)
	if err != nil {
		return model.PaginatedList[model.DataRequest]{}, err
	}

	items := make([]model.DataRequest, 0, len(rows))
	for _, row := range rows {
		items = append(items, dataRequestFromEnt(row))
	}
	return model.NewPaginatedList(items, total, page, limit), nil
}

func (r *Repository) DataRequestByID(ctx context.Context, id string) (model.DataRequest, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().DataRequests {
			if item.ID == id {
				return item, nil
			}
		}
		return model.DataRequest{}, fmt.Errorf("data request not found")
	}
	row, err := r.db.DataRequest.Get(ctx, id)
	if err != nil {
		return model.DataRequest{}, fmt.Errorf("data request not found")
	}
	return dataRequestFromEnt(row), nil
}

// CreateDataRequest preserves the original repository contract used by
// standalone component consumers. New workflows should call
// CreateDataRequestRecord so confirmation and result metadata are retained.
func (r *Repository) CreateDataRequest(ctx context.Context, id, userID, requestType string) error {
	now := time.Now().UTC()
	return r.CreateDataRequestRecord(ctx, model.DataRequest{
		ID: id, UserID: userID, Type: requestType, Status: "queued",
		Title: humanDataRequestTitle(requestType), CreatedAt: now, UpdatedAt: now,
	})
}

func (r *Repository) CreateDataRequestRecord(ctx context.Context, item model.DataRequest) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		r.store.DataRequests = append(r.store.DataRequests, store.DataRequest(item))
		return nil
	}
	creator := r.db.DataRequest.Create().SetID(item.ID).SetUserID(item.UserID).
		SetType(datarequest.Type(item.Type)).SetStatus(datarequest.Status(item.Status)).SetRequestedAt(item.CreatedAt)
	if item.ConfirmationTokenHash != "" {
		creator.SetConfirmationTokenHash(item.ConfirmationTokenHash)
	}
	if item.ConfirmationExpiresAt != nil {
		creator.SetConfirmationExpiresAt(*item.ConfirmationExpiresAt)
	}
	_, err := creator.Save(ctx)
	if err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) UpdateDataRequest(ctx context.Context, item model.DataRequest) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.DataRequests {
			if r.store.DataRequests[index].ID == item.ID {
				r.store.DataRequests[index] = item
				return nil
			}
		}
		return fmt.Errorf("data request not found")
	}
	update := r.db.DataRequest.UpdateOneID(item.ID).SetStatus(datarequest.Status(item.Status)).SetRetryCount(item.RetryCount)
	if item.ConfirmationTokenHash != "" {
		update.SetConfirmationTokenHash(item.ConfirmationTokenHash)
	} else {
		update.ClearConfirmationTokenHash()
	}
	if item.ConfirmationExpiresAt != nil {
		update.SetConfirmationExpiresAt(*item.ConfirmationExpiresAt)
	} else {
		update.ClearConfirmationExpiresAt()
	}
	if item.ConfirmedAt != nil {
		update.SetConfirmedAt(*item.ConfirmedAt)
	}
	if item.ResultPath != "" {
		update.SetResultPath(item.ResultPath)
	} else {
		update.ClearResultPath()
	}
	if item.ResultExpiresAt != nil {
		update.SetResultExpiresAt(*item.ResultExpiresAt)
	} else {
		update.ClearResultExpiresAt()
	}
	if item.FailureCode != "" {
		update.SetFailureCode(item.FailureCode)
	} else {
		update.ClearFailureCode()
	}
	if item.CompletedAt != nil {
		update.SetCompletedAt(*item.CompletedAt)
	}
	if _, err := update.Save(ctx); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) DataRequestByConfirmationToken(ctx context.Context, tokenHash string, now time.Time) (model.DataRequest, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().DataRequests {
			if item.ConfirmationTokenHash == tokenHash && item.ConfirmationExpiresAt != nil && now.Before(*item.ConfirmationExpiresAt) {
				return item, nil
			}
		}
		return model.DataRequest{}, fmt.Errorf("confirmation token is invalid or expired")
	}
	row, err := r.db.DataRequest.Query().Where(datarequest.ConfirmationTokenHashEQ(tokenHash), datarequest.ConfirmationExpiresAtGT(now)).Only(ctx)
	if err != nil {
		return model.DataRequest{}, fmt.Errorf("confirmation token is invalid or expired")
	}
	return dataRequestFromEnt(row), nil
}

func dataRequestFromEnt(item *ent.DataRequest) model.DataRequest {
	return model.DataRequest{ID: item.ID, UserID: item.UserID, Title: humanDataRequestTitle(item.Type.String()),
		Type: item.Type.String(), Status: item.Status.String(), ConfirmationTokenHash: value(item.ConfirmationTokenHash),
		ConfirmationExpiresAt: item.ConfirmationExpiresAt, ConfirmedAt: item.ConfirmedAt,
		ResultPath: value(item.ResultPath), ResultExpiresAt: item.ResultExpiresAt, FailureCode: value(item.FailureCode),
		RetryCount: item.RetryCount, CompletedAt: item.CompletedAt, CreatedAt: item.RequestedAt, UpdatedAt: item.UpdatedAt}
}
