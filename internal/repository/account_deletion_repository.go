package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitygroup"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitymembership"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/auditlog"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/checkin"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/contactverification"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/contentprogress"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/dailymission"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/datarequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/intention"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/membershipexitrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/notificationdelivery"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnercontactrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationprogress"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoverypracticesession"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryrecord"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryspace"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reflection"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/refreshtoken"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reportrollup"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportmessage"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) DeleteUserAccountData(ctx context.Context, userID string, now time.Time) error {
	user, ok := r.UserByID(ctx, userID)
	if !ok {
		return fmt.Errorf("user not found")
	}
	pseudoID := fmt.Sprintf("deleted:%x", sha256.Sum256([]byte(userID)))[:24]
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		groupIDs := make([]string, 0)
		for _, item := range r.store.AccountabilityGroups {
			if item.OwnerPartnerID == userID {
				groupIDs = append(groupIDs, item.ID)
			}
		}
		membershipIDs := make([]string, 0)
		for _, item := range r.store.AccountabilityMemberships {
			if item.StudentID == userID || slices.Contains(groupIDs, item.GroupID) {
				membershipIDs = append(membershipIDs, item.ID)
			}
		}
		caseIDs := make([]string, 0)
		for _, item := range r.store.SupportCases {
			if item.UserID == userID {
				caseIDs = append(caseIDs, item.ID)
			}
		}
		r.store.Users = filterSlice(r.store.Users, func(item model.User) bool { return item.ID != userID })
		r.store.ContactVerifications = filterSlice(r.store.ContactVerifications, func(item model.ContactVerification) bool { return item.UserID != userID })
		r.store.Devices = filterSlice(r.store.Devices, func(item model.Device) bool { return item.UserID != userID })
		r.store.Partners = filterSlice(r.store.Partners, func(item model.Partner) bool { return item.UserID != userID && item.PartnerUserID != userID })
		r.store.AccountabilityGroups = filterSlice(r.store.AccountabilityGroups, func(item model.AccountabilityGroup) bool { return item.OwnerPartnerID != userID })
		r.store.AccountabilityMemberships = filterSlice(r.store.AccountabilityMemberships, func(item model.AccountabilityMembership) bool { return !slices.Contains(membershipIDs, item.ID) })
		r.store.MembershipExitRequests = filterSlice(r.store.MembershipExitRequests, func(item model.MembershipExitRequest) bool {
			return !slices.Contains(membershipIDs, item.MembershipID) && item.RequestedBy != userID && item.ResolvedBy != userID
		})
		r.store.PartnerContactRequests = filterSlice(r.store.PartnerContactRequests, func(item model.PartnerContactRequest) bool {
			return !slices.Contains(membershipIDs, item.MembershipID) && item.StudentID != userID && item.PartnerID != userID
		})
		r.store.Approvals = filterSlice(r.store.Approvals, func(item model.ApprovalRequest) bool {
			return item.UserID != userID && item.ResolvedBy != userID && !slices.Contains(membershipIDs, item.MembershipID)
		})
		r.store.SupportCases = filterSlice(r.store.SupportCases, func(item model.SupportCase) bool { return item.UserID != userID })
		r.store.SupportMessages = filterSlice(r.store.SupportMessages, func(item model.SupportMessage) bool { return !slices.Contains(caseIDs, item.SupportCaseID) })
		r.store.JournalEntries = filterSlice(r.store.JournalEntries, func(item model.JournalEntry) bool { return item.UserID != userID })
		r.store.Missions = filterSlice(r.store.Missions, func(item model.DailyMission) bool { return item.UserID != userID })
		r.store.Intentions = filterSlice(r.store.Intentions, func(item model.Intention) bool { return item.UserID != userID })
		r.store.CheckIns = filterSlice(r.store.CheckIns, func(item model.CheckIn) bool { return item.UserID != userID })
		r.store.RecoveryRecords = filterSlice(r.store.RecoveryRecords, func(item model.RecoveryRecord) bool { return item.UserID != userID })
		r.store.RecoveryPracticeSessions = filterSlice(r.store.RecoveryPracticeSessions, func(item model.RecoveryPracticeSession) bool { return item.UserID != userID })
		r.store.RecoverySpaces = filterSlice(r.store.RecoverySpaces, func(item model.RecoverySpace) bool { return item.UserID != userID })
		r.store.AggregateEvents = filterSlice(r.store.AggregateEvents, func(item model.AggregateEvent) bool { return item.UserID != userID })
		for index := range r.store.DataRequests {
			if r.store.DataRequests[index].UserID == userID {
				if r.store.DataRequests[index].Type == "delete" && r.store.DataRequests[index].Status == "processing" {
					r.store.DataRequests[index].Status, r.store.DataRequests[index].CompletedAt = "completed", &now
				}
				r.store.DataRequests[index].UserID = pseudoID
				r.store.DataRequests[index].ConfirmationTokenHash = ""
				r.store.DataRequests[index].ResultPath = ""
			}
		}
		for index := range r.store.AuditEvents {
			if r.store.AuditEvents[index].ActorID == userID {
				r.store.AuditEvents[index].ActorID, r.store.AuditEvents[index].Actor = pseudoID, "deleted-account"
			}
		}
		return nil
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return err
	}
	rollback := func(cause error) error { _ = tx.Rollback(); return cause }
	groups, err := tx.AccountabilityGroup.Query().Where(accountabilitygroup.OwnerPartnerIDEQ(userID)).All(ctx)
	if err != nil {
		return rollback(err)
	}
	groupIDs := make([]string, 0, len(groups))
	for _, item := range groups {
		groupIDs = append(groupIDs, item.ID)
	}
	membershipQuery := tx.AccountabilityMembership.Query().Where(accountabilitymembership.StudentIDEQ(userID))
	if len(groupIDs) > 0 {
		membershipQuery = tx.AccountabilityMembership.Query().Where(accountabilitymembership.Or(accountabilitymembership.StudentIDEQ(userID), accountabilitymembership.GroupIDIn(groupIDs...)))
	}
	memberships, err := membershipQuery.All(ctx)
	if err != nil {
		return rollback(err)
	}
	membershipIDs := make([]string, 0, len(memberships))
	for _, item := range memberships {
		membershipIDs = append(membershipIDs, item.ID)
	}
	cases, err := tx.SupportCase.Query().Where(supportcase.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return rollback(err)
	}
	caseIDs := make([]string, 0, len(cases))
	for _, item := range cases {
		caseIDs = append(caseIDs, item.ID)
	}
	approvals, err := tx.ApprovalRequest.Query().Where(approvalrequest.UserIDEQ(userID)).All(ctx)
	if err != nil {
		return rollback(err)
	}
	approvalIDs := make([]string, 0, len(approvals))
	for _, item := range approvals {
		approvalIDs = append(approvalIDs, item.ID)
	}
	if len(caseIDs) > 0 {
		if _, err = tx.SupportMessage.Delete().Where(supportmessage.SupportCaseIDIn(caseIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
	}
	if len(approvalIDs) > 0 {
		if _, err = tx.NotificationDelivery.Delete().Where(notificationdelivery.ApprovalRequestIDIn(approvalIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
	}
	if len(caseIDs) > 0 {
		if _, err = tx.NotificationDelivery.Delete().Where(notificationdelivery.SupportCaseIDIn(caseIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
	}
	if _, err = tx.NotificationDelivery.Delete().Where(notificationdelivery.RecipientIn(user.Email, user.PhoneE164)).Exec(ctx); err != nil {
		return rollback(err)
	}
	if len(membershipIDs) > 0 {
		if _, err = tx.MembershipExitRequest.Delete().Where(membershipexitrequest.MembershipIDIn(membershipIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
		if _, err = tx.PartnerContactRequest.Delete().Where(partnercontactrequest.MembershipIDIn(membershipIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
		if _, err = tx.ApprovalRequest.Delete().Where(approvalrequest.MembershipIDIn(membershipIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
		if _, err = tx.AccountabilityMembership.Delete().Where(accountabilitymembership.IDIn(membershipIDs...)).Exec(ctx); err != nil {
			return rollback(err)
		}
	}
	deleteOps := []func() error{
		func() error { _, e := tx.SupportCase.Delete().Where(supportcase.UserIDEQ(userID)).Exec(ctx); return e },
		func() error {
			_, e := tx.ApprovalRequest.Delete().Where(approvalrequest.Or(approvalrequest.UserIDEQ(userID), approvalrequest.ResolvedByEQ(userID))).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.PartnerContactRequest.Delete().Where(partnercontactrequest.Or(partnercontactrequest.StudentIDEQ(userID), partnercontactrequest.PartnerIDEQ(userID))).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.MembershipExitRequest.Delete().Where(membershipexitrequest.Or(membershipexitrequest.RequestedByEQ(userID), membershipexitrequest.ResolvedByEQ(userID))).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.AccountabilityMembership.Delete().Where(accountabilitymembership.StudentIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.AccountabilityGroup.Delete().Where(accountabilitygroup.OwnerPartnerIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.PartnerLink.Delete().Where(partnerlink.Or(partnerlink.UserIDEQ(userID), partnerlink.PartnerUserIDEQ(userID))).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.OrganizationMember.Delete().Where(organizationmember.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.PsychoeducationProgress.Delete().Where(psychoeducationprogress.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.ContentProgress.Delete().Where(contentprogress.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.RecoveryRecord.Delete().Where(recoveryrecord.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.RecoveryPracticeSession.Delete().Where(recoverypracticesession.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.RecoverySpace.Delete().Where(recoveryspace.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error { _, e := tx.Reflection.Delete().Where(reflection.UserIDEQ(userID)).Exec(ctx); return e },
		func() error {
			_, e := tx.DailyMission.Delete().Where(dailymission.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error { _, e := tx.Intention.Delete().Where(intention.UserIDEQ(userID)).Exec(ctx); return e },
		func() error { _, e := tx.CheckIn.Delete().Where(checkin.UserIDEQ(userID)).Exec(ctx); return e },
		func() error {
			_, e := tx.AggregateEvent.Delete().Where(aggregateevent.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error { _, e := tx.Device.Delete().Where(device.UserIDEQ(userID)).Exec(ctx); return e },
		func() error {
			_, e := tx.RefreshToken.Delete().Where(refreshtoken.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.ContactVerification.Delete().Where(contactverification.UserIDEQ(userID)).Exec(ctx)
			return e
		},
		func() error {
			_, e := tx.ReportRollup.Delete().Where(reportrollup.ScopeIDEQ(userID)).Exec(ctx)
			return e
		},
	}
	for _, operation := range deleteOps {
		if err = operation(); err != nil {
			return rollback(err)
		}
	}
	if _, err = tx.DataRequest.Update().Where(datarequest.UserIDEQ(userID), datarequest.TypeEQ(datarequest.TypeDelete), datarequest.StatusEQ(datarequest.StatusProcessing)).SetStatus(datarequest.StatusCompleted).SetCompletedAt(now).Save(ctx); err != nil {
		return rollback(err)
	}
	if _, err = tx.DataRequest.Update().Where(datarequest.UserIDEQ(userID)).SetUserID(pseudoID).ClearConfirmationTokenHash().ClearConfirmationExpiresAt().ClearResultPath().ClearResultExpiresAt().Save(ctx); err != nil {
		return rollback(err)
	}
	if _, err = tx.AuditLog.Update().Where(auditlog.ActorIDEQ(userID)).SetActorID(pseudoID).SetActorEmail("deleted-account").Save(ctx); err != nil {
		return rollback(err)
	}
	if _, err = tx.User.Delete().Where(entuser.IDEQ(userID)).Exec(ctx); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	r.RefreshStore(ctx)
	return nil
}
