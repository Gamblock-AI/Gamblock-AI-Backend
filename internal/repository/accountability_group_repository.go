package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitygroup"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitymembership"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/membershipexitrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnercontactrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

var liveMembershipStatuses = map[string]bool{
	"active": true, "leave_pending": true, "support_review": true, "safety_suspended": true,
}

func (r *Repository) CreateAccountabilityGroup(ctx context.Context, group model.AccountabilityGroup) (model.AccountabilityGroup, error) {
	if r.db == nil {
		r.store.Lock()
		r.store.AccountabilityGroups = append(r.store.AccountabilityGroups, group)
		r.store.Unlock()
		return group, nil
	}
	row, err := r.db.AccountabilityGroup.Create().
		SetID(group.ID).
		SetOwnerPartnerID(group.OwnerPartnerID).
		SetName(group.Name).
		SetDescription(group.Description).
		SetJoinCodeHash(group.JoinCodeHash).
		SetJoinCodeHint(group.JoinCodeHint).
		SetCodeRotatedAt(group.CodeRotatedAt).
		Save(ctx)
	if err != nil {
		return model.AccountabilityGroup{}, err
	}
	r.RefreshStore(ctx)
	return groupFromEnt(row), nil
}

func groupFromEnt(row *ent.AccountabilityGroup) model.AccountabilityGroup {
	return model.AccountabilityGroup{
		ID: row.ID, OwnerPartnerID: row.OwnerPartnerID, Name: row.Name,
		Description: row.Description, JoinCodeHash: row.JoinCodeHash,
		JoinCodeHint: row.JoinCodeHint, Status: row.Status.String(),
		CodeRotatedAt: row.CodeRotatedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func membershipFromEnt(row *ent.AccountabilityMembership) model.AccountabilityMembership {
	return model.AccountabilityMembership{
		ID: row.ID, GroupID: row.GroupID, StudentID: row.StudentID,
		Status: row.Status.String(), JoinedAt: row.JoinedAt, EndedAt: row.EndedAt,
		Sharing: model.SharingPreferences{
			ProtectionHealth:   row.ShareProtectionHealth,
			ProtectionActivity: row.ShareProtectionActivity,
			RecoveryEngagement: row.ShareRecoveryEngagement,
			EducationProgress:  row.ShareEducationProgress,
		},
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *Repository) ListAccountabilityGroups(ctx context.Context, partnerID string) ([]model.AccountabilityGroup, error) {
	var groups []model.AccountabilityGroup
	if r.db == nil {
		for _, group := range r.store.Snapshot().AccountabilityGroups {
			if group.OwnerPartnerID == partnerID {
				groups = append(groups, group)
			}
		}
	} else {
		rows, err := r.db.AccountabilityGroup.Query().
			Where(accountabilitygroup.OwnerPartnerIDEQ(partnerID)).
			Order(ent.Desc(accountabilitygroup.FieldUpdatedAt)).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			groups = append(groups, groupFromEnt(row))
		}
	}
	for i := range groups {
		members, err := r.ListMembershipsForGroup(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		for _, membership := range members {
			if liveMembershipStatuses[membership.Status] {
				groups[i].MemberCount++
			}
		}
	}
	return groups, nil
}

func (r *Repository) AccountabilityGroupByID(ctx context.Context, groupID string) (model.AccountabilityGroup, error) {
	if r.db == nil {
		for _, group := range r.store.Snapshot().AccountabilityGroups {
			if group.ID == groupID {
				return group, nil
			}
		}
		return model.AccountabilityGroup{}, fmt.Errorf("accountability group not found")
	}
	row, err := r.db.AccountabilityGroup.Get(ctx, groupID)
	if err != nil {
		return model.AccountabilityGroup{}, fmt.Errorf("accountability group not found")
	}
	return groupFromEnt(row), nil
}

func (r *Repository) AccountabilityGroupByCodeHash(ctx context.Context, codeHash string) (model.AccountabilityGroup, error) {
	if r.db == nil {
		for _, group := range r.store.Snapshot().AccountabilityGroups {
			if group.JoinCodeHash == codeHash && group.Status == "active" {
				return group, nil
			}
		}
		return model.AccountabilityGroup{}, fmt.Errorf("join code is invalid")
	}
	row, err := r.db.AccountabilityGroup.Query().Where(
		accountabilitygroup.JoinCodeHashEQ(codeHash),
		accountabilitygroup.StatusEQ(accountabilitygroup.StatusActive),
	).Only(ctx)
	if err != nil {
		return model.AccountabilityGroup{}, fmt.Errorf("join code is invalid")
	}
	return groupFromEnt(row), nil
}

func (r *Repository) RotateAccountabilityGroupCode(ctx context.Context, groupID, partnerID, codeHash, hint string, rotatedAt time.Time) error {
	group, err := r.AccountabilityGroupByID(ctx, groupID)
	if err != nil || group.OwnerPartnerID != partnerID || group.Status != "active" {
		return fmt.Errorf("partner is not authorized for this group")
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.AccountabilityGroups {
			if r.store.AccountabilityGroups[i].ID == groupID {
				r.store.AccountabilityGroups[i].JoinCodeHash = codeHash
				r.store.AccountabilityGroups[i].JoinCodeHint = hint
				r.store.AccountabilityGroups[i].CodeRotatedAt = rotatedAt
				r.store.AccountabilityGroups[i].UpdatedAt = rotatedAt
				return nil
			}
		}
		return fmt.Errorf("accountability group not found")
	}
	_, err = r.db.AccountabilityGroup.UpdateOneID(groupID).
		SetJoinCodeHash(codeHash).SetJoinCodeHint(hint).SetCodeRotatedAt(rotatedAt).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) ActiveMembershipForStudent(ctx context.Context, studentID string) (*model.AccountabilityMembership, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().AccountabilityMemberships {
			if item.StudentID == studentID && liveMembershipStatuses[item.Status] {
				result := r.hydrateMembership(item)
				return &result, nil
			}
		}
		return nil, nil
	}
	row, err := r.db.AccountabilityMembership.Query().Where(
		accountabilitymembership.StudentIDEQ(studentID),
		accountabilitymembership.StatusIn(
			accountabilitymembership.StatusActive,
			accountabilitymembership.StatusLeavePending,
			accountabilitymembership.StatusSupportReview,
			accountabilitymembership.StatusSafetySuspended,
		),
	).Order(ent.Desc(accountabilitymembership.FieldUpdatedAt)).First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := r.hydrateMembership(membershipFromEnt(row))
	return &result, nil
}

func (r *Repository) MembershipByID(ctx context.Context, membershipID string) (model.AccountabilityMembership, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().AccountabilityMemberships {
			if item.ID == membershipID {
				return r.hydrateMembership(item), nil
			}
		}
		return model.AccountabilityMembership{}, fmt.Errorf("membership not found")
	}
	row, err := r.db.AccountabilityMembership.Get(ctx, membershipID)
	if err != nil {
		return model.AccountabilityMembership{}, fmt.Errorf("membership not found")
	}
	return r.hydrateMembership(membershipFromEnt(row)), nil
}

func (r *Repository) ListMembershipsForGroup(ctx context.Context, groupID string) ([]model.AccountabilityMembership, error) {
	var result []model.AccountabilityMembership
	if r.db == nil {
		for _, item := range r.store.Snapshot().AccountabilityMemberships {
			if item.GroupID == groupID {
				result = append(result, r.hydrateMembership(item))
			}
		}
	} else {
		rows, err := r.db.AccountabilityMembership.Query().Where(
			accountabilitymembership.GroupIDEQ(groupID),
		).Order(ent.Desc(accountabilitymembership.FieldUpdatedAt)).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, r.hydrateMembership(membershipFromEnt(row)))
		}
	}
	return result, nil
}

func (r *Repository) SaveAccountabilityMembership(ctx context.Context, item model.AccountabilityMembership) (model.AccountabilityMembership, error) {
	if r.db == nil {
		r.store.Lock()
		for i := range r.store.AccountabilityMemberships {
			if r.store.AccountabilityMemberships[i].GroupID == item.GroupID && r.store.AccountabilityMemberships[i].StudentID == item.StudentID {
				item.ID = r.store.AccountabilityMemberships[i].ID
				item.CreatedAt = r.store.AccountabilityMemberships[i].CreatedAt
				r.store.AccountabilityMemberships[i] = item
				r.store.Unlock()
				return r.hydrateMembership(item), nil
			}
		}
		r.store.AccountabilityMemberships = append(r.store.AccountabilityMemberships, item)
		r.store.Unlock()
		return r.hydrateMembership(item), nil
	}
	row, err := r.db.AccountabilityMembership.Query().Where(
		accountabilitymembership.GroupIDEQ(item.GroupID),
		accountabilitymembership.StudentIDEQ(item.StudentID),
	).Only(ctx)
	if err == nil {
		row, err = row.Update().SetStatus(accountabilitymembership.Status(item.Status)).
			SetShareProtectionHealth(item.Sharing.ProtectionHealth).
			SetShareProtectionActivity(item.Sharing.ProtectionActivity).
			SetShareRecoveryEngagement(item.Sharing.RecoveryEngagement).
			SetShareEducationProgress(item.Sharing.EducationProgress).
			SetJoinedAt(item.JoinedAt).ClearEndedAt().Save(ctx)
	} else if ent.IsNotFound(err) {
		row, err = r.db.AccountabilityMembership.Create().SetID(item.ID).
			SetGroupID(item.GroupID).SetStudentID(item.StudentID).
			SetStatus(accountabilitymembership.Status(item.Status)).
			SetShareProtectionHealth(item.Sharing.ProtectionHealth).
			SetShareProtectionActivity(item.Sharing.ProtectionActivity).
			SetShareRecoveryEngagement(item.Sharing.RecoveryEngagement).
			SetShareEducationProgress(item.Sharing.EducationProgress).
			SetJoinedAt(item.JoinedAt).Save(ctx)
	}
	if err != nil {
		return model.AccountabilityMembership{}, err
	}
	r.RefreshStore(ctx)
	return r.hydrateMembership(membershipFromEnt(row)), nil
}

func (r *Repository) UpdateMembershipSharing(ctx context.Context, membershipID, studentID string, sharing model.SharingPreferences) (model.AccountabilityMembership, error) {
	item, err := r.MembershipByID(ctx, membershipID)
	if err != nil || item.StudentID != studentID || !liveMembershipStatuses[item.Status] {
		return model.AccountabilityMembership{}, fmt.Errorf("student is not authorized for this membership")
	}
	if r.db == nil {
		r.store.Lock()
		for i := range r.store.AccountabilityMemberships {
			if r.store.AccountabilityMemberships[i].ID == membershipID {
				r.store.AccountabilityMemberships[i].Sharing = sharing
				r.store.AccountabilityMemberships[i].UpdatedAt = time.Now().UTC()
				item = r.store.AccountabilityMemberships[i]
				break
			}
		}
		r.store.Unlock()
		return r.hydrateMembership(item), nil
	}
	row, err := r.db.AccountabilityMembership.UpdateOneID(membershipID).
		SetShareProtectionHealth(sharing.ProtectionHealth).
		SetShareProtectionActivity(sharing.ProtectionActivity).
		SetShareRecoveryEngagement(sharing.RecoveryEngagement).
		SetShareEducationProgress(sharing.EducationProgress).Save(ctx)
	if err != nil {
		return model.AccountabilityMembership{}, err
	}
	r.RefreshStore(ctx)
	return r.hydrateMembership(membershipFromEnt(row)), nil
}

func (r *Repository) SetMembershipStatus(ctx context.Context, membershipID, status string, endedAt *time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.AccountabilityMemberships {
			if r.store.AccountabilityMemberships[i].ID == membershipID {
				r.store.AccountabilityMemberships[i].Status = status
				r.store.AccountabilityMemberships[i].EndedAt = endedAt
				r.store.AccountabilityMemberships[i].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("membership not found")
	}
	update := r.db.AccountabilityMembership.UpdateOneID(membershipID).
		SetStatus(accountabilitymembership.Status(status))
	if endedAt != nil {
		update.SetEndedAt(*endedAt)
	} else {
		update.ClearEndedAt()
	}
	_, err := update.Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) CreateMembershipExitRequest(ctx context.Context, item model.MembershipExitRequest) (model.MembershipExitRequest, error) {
	if r.db == nil {
		r.store.Lock()
		r.store.MembershipExitRequests = append(r.store.MembershipExitRequests, item)
		r.store.Unlock()
		return item, nil
	}
	create := r.db.MembershipExitRequest.Create().SetID(item.ID).
		SetMembershipID(item.MembershipID).SetRequestedBy(item.RequestedBy).
		SetKind(membershipexitrequest.Kind(item.Kind)).SetStatus(membershipexitrequest.Status(item.Status)).
		SetReason(item.Reason)
	if item.ReviewDueAt != nil {
		create.SetReviewDueAt(*item.ReviewDueAt)
	}
	row, err := create.Save(ctx)
	if err != nil {
		return model.MembershipExitRequest{}, err
	}
	r.RefreshStore(ctx)
	return exitRequestFromEnt(row), nil
}

func exitRequestFromEnt(row *ent.MembershipExitRequest) model.MembershipExitRequest {
	return model.MembershipExitRequest{
		ID: row.ID, MembershipID: row.MembershipID, RequestedBy: row.RequestedBy,
		Kind: row.Kind.String(), Status: row.Status.String(), Reason: row.Reason,
		ReviewDueAt: row.ReviewDueAt, ResolvedBy: value(row.ResolvedBy),
		ResolvedAt: row.ResolvedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *Repository) ListExitRequests(ctx context.Context, membershipIDs []string) ([]model.MembershipExitRequest, error) {
	if len(membershipIDs) == 0 {
		return []model.MembershipExitRequest{}, nil
	}
	allowed := make(map[string]bool, len(membershipIDs))
	for _, id := range membershipIDs {
		allowed[id] = true
	}
	var result []model.MembershipExitRequest
	if r.db == nil {
		for _, item := range r.store.Snapshot().MembershipExitRequests {
			if allowed[item.MembershipID] {
				result = append(result, item)
			}
		}
	} else {
		rows, err := r.db.MembershipExitRequest.Query().Where(
			membershipexitrequest.MembershipIDIn(membershipIDs...),
		).Order(ent.Desc(membershipexitrequest.FieldCreatedAt)).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, exitRequestFromEnt(row))
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (r *Repository) ResolveMembershipExitRequest(ctx context.Context, requestID, partnerID, decision string) error {
	var request model.MembershipExitRequest
	if r.db == nil {
		for _, item := range r.store.Snapshot().MembershipExitRequests {
			if item.ID == requestID {
				request = item
				break
			}
		}
	} else {
		row, err := r.db.MembershipExitRequest.Get(ctx, requestID)
		if err != nil {
			return fmt.Errorf("exit request not found")
		}
		request = exitRequestFromEnt(row)
	}
	membership, err := r.MembershipByID(ctx, request.MembershipID)
	if err != nil {
		return err
	}
	group, err := r.AccountabilityGroupByID(ctx, membership.GroupID)
	if err != nil || group.OwnerPartnerID != partnerID || request.Status != "pending" {
		return fmt.Errorf("partner is not authorized for this request")
	}
	if decision != "approved" && decision != "denied" {
		return fmt.Errorf("invalid exit decision")
	}
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		for i := range r.store.MembershipExitRequests {
			if r.store.MembershipExitRequests[i].ID == requestID {
				r.store.MembershipExitRequests[i].Status = decision
				r.store.MembershipExitRequests[i].ResolvedBy = partnerID
				r.store.MembershipExitRequests[i].ResolvedAt = &now
				r.store.MembershipExitRequests[i].UpdatedAt = now
			}
		}
		r.store.Unlock()
	} else {
		_, err = r.db.MembershipExitRequest.UpdateOneID(requestID).
			SetStatus(membershipexitrequest.Status(decision)).SetResolvedBy(partnerID).SetResolvedAt(now).Save(ctx)
		if err != nil {
			return err
		}
	}
	if decision == "approved" {
		err = r.SetMembershipStatus(ctx, membership.ID, "left", &now)
		if err == nil {
			err = r.CancelPendingApprovalsForMembership(ctx, membership.ID, partnerID)
		}
	} else {
		err = r.SetMembershipStatus(ctx, membership.ID, "active", nil)
	}
	if err == nil && r.db != nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) EscalateOverdueExitRequests(ctx context.Context, now time.Time) error {
	if r.db == nil {
		var memberships []string
		r.store.Lock()
		for i := range r.store.MembershipExitRequests {
			item := &r.store.MembershipExitRequests[i]
			if item.Status == "pending" && item.Kind == "normal" && item.ReviewDueAt != nil && !now.Before(*item.ReviewDueAt) {
				item.Status = "auto_reviewed"
				item.ResolvedAt = &now
				item.UpdatedAt = now
				memberships = append(memberships, item.MembershipID)
			}
		}
		r.store.Unlock()
		for _, membershipID := range memberships {
			if err := r.SetMembershipStatus(ctx, membershipID, "support_review", nil); err != nil {
				return err
			}
		}
		return nil
	}
	rows, err := r.db.MembershipExitRequest.Query().Where(
		membershipexitrequest.StatusEQ(membershipexitrequest.StatusPending),
		membershipexitrequest.KindEQ(membershipexitrequest.KindNormal),
		membershipexitrequest.ReviewDueAtLTE(now),
	).All(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := row.Update().SetStatus(membershipexitrequest.StatusAutoReviewed).SetResolvedAt(now).Save(ctx); err != nil {
			return err
		}
		if err := r.SetMembershipStatus(ctx, row.MembershipID, "support_review", nil); err != nil {
			return err
		}
	}
	if len(rows) > 0 {
		r.RefreshStore(ctx)
	}
	return nil
}

func (r *Repository) CreatePartnerContactRequest(ctx context.Context, item model.PartnerContactRequest) (model.PartnerContactRequest, error) {
	if r.db == nil {
		r.store.Lock()
		r.store.PartnerContactRequests = append(r.store.PartnerContactRequests, item)
		r.store.Unlock()
		return item, nil
	}
	row, err := r.db.PartnerContactRequest.Create().SetID(item.ID).
		SetMembershipID(item.MembershipID).SetStudentID(item.StudentID).SetPartnerID(item.PartnerID).
		SetCategory(partnercontactrequest.Category(item.Category)).SetNillableMessageEncrypted(optional(item.Message)).
		SetStatus(partnercontactrequest.StatusPending).Save(ctx)
	if err != nil {
		return model.PartnerContactRequest{}, err
	}
	r.RefreshStore(ctx)
	return contactRequestFromEnt(row), nil
}

func contactRequestFromEnt(row *ent.PartnerContactRequest) model.PartnerContactRequest {
	return model.PartnerContactRequest{
		ID: row.ID, MembershipID: row.MembershipID, StudentID: row.StudentID,
		PartnerID: row.PartnerID, Category: row.Category.String(), Message: value(row.MessageEncrypted),
		Status: row.Status.String(), AcknowledgedAt: row.AcknowledgedAt, ClosedAt: row.ClosedAt,
		EscalatedAt: row.EscalatedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *Repository) ListPartnerContactRequests(ctx context.Context, userID, role string) ([]model.PartnerContactRequest, error) {
	var result []model.PartnerContactRequest
	if r.db == nil {
		for _, item := range r.store.Snapshot().PartnerContactRequests {
			if (role == "partner" && item.PartnerID == userID) || (role == "user" && item.StudentID == userID) {
				result = append(result, item)
			}
		}
	} else {
		query := r.db.PartnerContactRequest.Query()
		if role == "partner" {
			query.Where(partnercontactrequest.PartnerIDEQ(userID))
		} else {
			query.Where(partnercontactrequest.StudentIDEQ(userID))
		}
		rows, err := query.Order(ent.Desc(partnercontactrequest.FieldCreatedAt)).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			result = append(result, contactRequestFromEnt(row))
		}
	}
	for i := range result {
		if user, ok := r.UserByID(ctx, result[i].StudentID); ok {
			result[i].StudentName = user.DisplayName
		}
	}
	return result, nil
}

func (r *Repository) TransitionPartnerContactRequest(ctx context.Context, requestID, actorID, status string) error {
	if status != "acknowledged" && status != "closed" && status != "cancelled" && status != "escalated" {
		return fmt.Errorf("invalid contact request transition")
	}
	var item model.PartnerContactRequest
	if r.db == nil {
		for _, candidate := range r.store.Snapshot().PartnerContactRequests {
			if candidate.ID == requestID {
				item = candidate
			}
		}
	} else {
		row, err := r.db.PartnerContactRequest.Get(ctx, requestID)
		if err != nil {
			return fmt.Errorf("contact request not found")
		}
		item = contactRequestFromEnt(row)
	}
	if actorID != item.StudentID && actorID != item.PartnerID {
		return fmt.Errorf("actor is not authorized for contact request")
	}
	if status == "acknowledged" && actorID != item.PartnerID {
		return fmt.Errorf("only the partner can acknowledge")
	}
	if status == "escalated" && (actorID != item.StudentID || time.Since(item.CreatedAt) < 24*time.Hour) {
		return fmt.Errorf("escalation is available to the student after 24 hours")
	}
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.PartnerContactRequests {
			candidate := &r.store.PartnerContactRequests[i]
			if candidate.ID != requestID {
				continue
			}
			switch status {
			case "acknowledged":
				candidate.Status, candidate.AcknowledgedAt = status, &now
			case "closed", "cancelled":
				candidate.Status, candidate.ClosedAt = status, &now
			case "escalated":
				candidate.EscalatedAt = &now
			}
			candidate.UpdatedAt = now
			return nil
		}
		return fmt.Errorf("contact request not found")
	}
	update := r.db.PartnerContactRequest.UpdateOneID(requestID)
	switch status {
	case "acknowledged":
		update.SetStatus(partnercontactrequest.StatusAcknowledged).SetAcknowledgedAt(now)
	case "closed":
		update.SetStatus(partnercontactrequest.StatusClosed).SetClosedAt(now)
	case "cancelled":
		update.SetStatus(partnercontactrequest.StatusCancelled).SetClosedAt(now)
	case "escalated":
		update.SetEscalatedAt(now)
	}
	_, err := update.Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) ArchiveAccountabilityGroup(ctx context.Context, groupID, partnerID string) error {
	group, err := r.AccountabilityGroupByID(ctx, groupID)
	if err != nil || group.OwnerPartnerID != partnerID {
		return fmt.Errorf("partner is not authorized for this group")
	}
	members, err := r.ListMembershipsForGroup(ctx, groupID)
	if err != nil {
		return err
	}
	for _, member := range members {
		if liveMembershipStatuses[member.Status] {
			return fmt.Errorf("group still has active members")
		}
	}
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.AccountabilityGroups {
			if r.store.AccountabilityGroups[i].ID == groupID {
				r.store.AccountabilityGroups[i].Status = "archived"
				r.store.AccountabilityGroups[i].UpdatedAt = time.Now().UTC()
				return nil
			}
		}
		return fmt.Errorf("group not found")
	}
	_, err = r.db.AccountabilityGroup.UpdateOneID(groupID).SetStatus(accountabilitygroup.StatusArchived).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) hydrateMembership(item model.AccountabilityMembership) model.AccountabilityMembership {
	snapshot := r.store.Snapshot()
	for _, user := range snapshot.Users {
		if user.ID == item.StudentID {
			item.StudentName = user.DisplayName
			item.StudentMail = user.Email
			break
		}
	}
	item.Aggregate = aggregateForMembership(&snapshot, item)
	return item
}

func aggregateForMembership(snapshot *store.Store, item model.AccountabilityMembership) model.MemberAggregateSummary {
	result := model.MemberAggregateSummary{}
	now := time.Now().UTC()
	if item.Sharing.ProtectionHealth {
		latest := time.Time{}
		status := "unknown"
		for _, device := range snapshot.Devices {
			if device.UserID != item.StudentID {
				continue
			}
			result.ActiveDeviceCount++
			if status == "unknown" {
				status = "ready"
			}
			if device.ProtectionStatus != "active" {
				status = "attention"
			}
			if device.LastSeenAt.After(latest) {
				latest = device.LastSeenAt
			}
		}
		result.ProtectionStatus = status
		result.LastHeartbeatBucket = heartbeatBucket(now, latest)
	}
	if item.Sharing.ProtectionActivity {
		for _, event := range snapshot.AggregateEvents {
			if event.UserID == item.StudentID && event.EventType == "block_count_sync" && event.EventDate.After(now.Add(-7*24*time.Hour)) {
				result.WeeklyBlockCount += event.Count
			}
		}
	}
	if item.Sharing.RecoveryEngagement {
		days := map[string]bool{}
		for _, checkIn := range snapshot.CheckIns {
			if checkIn.UserID == item.StudentID && checkIn.CreatedAt.After(now.Add(-7*24*time.Hour)) {
				days[checkIn.CreatedAt.In(time.FixedZone("Asia/Jakarta", 7*60*60)).Format("2006-01-02")] = true
			}
		}
		result.CheckInDays = len(days)
		for _, mission := range snapshot.Missions {
			if mission.UserID == item.StudentID {
				for _, done := range []bool{mission.Mission1, mission.Mission2, mission.Mission3, mission.Mission4, mission.Mission5} {
					if done {
						result.MissionCompleted++
					}
				}
			}
		}
	}
	if item.Sharing.EducationProgress {
		count, total := 0, 0
		for _, progress := range snapshot.EducationProgress {
			if progress.UserID == item.StudentID {
				count++
				total += progress.ProgressPercent
			}
		}
		percent := 0
		if count > 0 {
			percent = total / count
		}
		switch {
		case percent == 0:
			result.EducationProgressBand = "not_started"
		case percent < 40:
			result.EducationProgressBand = "starting"
		case percent < 80:
			result.EducationProgressBand = "in_progress"
		default:
			result.EducationProgressBand = "near_complete"
		}
	}
	return result
}

func heartbeatBucket(now, last time.Time) string {
	if last.IsZero() {
		return "never"
	}
	days := int(now.Sub(last).Hours() / 24)
	switch {
	case days < 1:
		return "today"
	case days <= 3:
		return "1-3d"
	case days <= 7:
		return "4-7d"
	default:
		return "older"
	}
}
