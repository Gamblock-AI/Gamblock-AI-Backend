package db

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func loadOperationsStore(ctx context.Context, client *ent.Client, out *store.Store) error {
	groups, err := client.AccountabilityGroup.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range groups {
		out.AccountabilityGroups = append(out.AccountabilityGroups, store.AccountabilityGroup{
			ID: item.ID, OwnerPartnerID: item.OwnerPartnerID, Name: item.Name,
			Description: item.Description, JoinCodeHash: item.JoinCodeHash, JoinCodeHint: item.JoinCodeHint,
			Status: item.Status.String(), CodeRotatedAt: item.CodeRotatedAt,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	memberships, err := client.AccountabilityMembership.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range memberships {
		out.AccountabilityMemberships = append(out.AccountabilityMemberships, store.AccountabilityMembership{
			ID: item.ID, GroupID: item.GroupID, StudentID: item.StudentID,
			Status: item.Status.String(), JoinedAt: item.JoinedAt, EndedAt: item.EndedAt,
			Sharing: store.SharingPreferences{
				ProtectionHealth:   item.ShareProtectionHealth,
				ProtectionActivity: item.ShareProtectionActivity,
				RecoveryEngagement: item.ShareRecoveryEngagement,
				EducationProgress:  item.ShareEducationProgress,
			},
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	exitRequests, err := client.MembershipExitRequest.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range exitRequests {
		out.MembershipExitRequests = append(out.MembershipExitRequests, store.MembershipExitRequest{
			ID: item.ID, MembershipID: item.MembershipID, RequestedBy: item.RequestedBy,
			Kind: item.Kind.String(), Status: item.Status.String(), Reason: item.Reason,
			ReviewDueAt: item.ReviewDueAt, ResolvedBy: value(item.ResolvedBy),
			ResolvedAt: item.ResolvedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	contactRequests, err := client.PartnerContactRequest.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range contactRequests {
		out.PartnerContactRequests = append(out.PartnerContactRequests, store.PartnerContactRequest{
			ID: item.ID, MembershipID: item.MembershipID, StudentID: item.StudentID,
			PartnerID: item.PartnerID, Category: item.Category.String(), Status: item.Status.String(),
			AcknowledgedAt: item.AcknowledgedAt, ClosedAt: item.ClosedAt,
			EscalatedAt: item.EscalatedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	orgs, err := client.Organization.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range orgs {
		members, err := client.OrganizationMember.Query().
			Where(organizationmember.OrganizationIDEQ(item.ID)).
			Count(ctx)
		if err != nil {
			return err
		}
		out.Organizations = append(out.Organizations, store.Organization{
			ID:        item.ID,
			Name:      item.Name,
			Slug:      item.Slug,
			Status:    item.Status.String(),
			Members:   members,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	supportCases, err := client.SupportCase.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range supportCases {
		out.SupportCases = append(out.SupportCases, store.SupportCase{
			ID:         item.ID,
			UserID:     item.UserID,
			Title:      item.Summary,
			Type:       item.Type.String(),
			Status:     item.Status.String(),
			Priority:   item.Priority.String(),
			Impact:     item.Impact,
			ResolvedAt: item.ResolvedAt,
			ClosedAt:   item.ClosedAt,
			Owner:      value(item.AssignedOperatorID),
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
		})
	}

	supportMessages, err := client.SupportMessage.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range supportMessages {
		out.SupportMessages = append(out.SupportMessages, store.SupportMessage{
			ID: item.ID, SupportCaseID: item.SupportCaseID, AuthorID: item.AuthorID,
			AuthorRole: item.AuthorRole.String(), Content: item.ContentEncrypted,
			ReadAt: item.ReadAt, CreatedAt: item.CreatedAt,
		})
	}

	dataRequests, err := client.DataRequest.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range dataRequests {
		out.DataRequests = append(out.DataRequests, store.DataRequest{
			ID: item.ID, UserID: item.UserID, Title: humanDataRequestTitle(item.Type.String()),
			Type: item.Type.String(), Status: item.Status.String(),
			ConfirmationTokenHash: value(item.ConfirmationTokenHash), ConfirmationExpiresAt: item.ConfirmationExpiresAt,
			ConfirmedAt: item.ConfirmedAt, ResultPath: value(item.ResultPath), ResultExpiresAt: item.ResultExpiresAt,
			FailureCode: value(item.FailureCode), RetryCount: item.RetryCount, CompletedAt: item.CompletedAt,
			CreatedAt: item.RequestedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	audits, err := client.AuditLog.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range audits {
		out.AuditEvents = append(out.AuditEvents, store.AuditEvent{
			ID: item.ID, ActorID: item.ActorID, Actor: item.ActorEmail, Action: item.Action,
			TargetType: item.TargetType, Target: item.TargetID, Reason: item.Reason,
			Metadata: item.MetadataJSON, CreatedAt: item.CreatedAt, UpdatedAt: time.Time{},
		})
	}

	revisions, err := client.EducationRevision.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range revisions {
		out.EducationRevisions = append(out.EducationRevisions, store.EducationRevision{
			ID: item.ID, ModuleID: item.ModuleID, Revision: item.Revision, Document: item.DocumentJSON,
			Slug: item.Slug, Kind: item.Kind.String(), CreatedBy: item.CreatedBy, CreatedAt: item.CreatedAt,
		})
	}

	socialLinks, err := client.SiteSocialLink.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range socialLinks {
		out.SiteSocialLinks = append(out.SiteSocialLinks, store.SiteSocialLink{
			ID: item.ID, Platform: item.Platform.String(), Label: item.Label, URL: item.URL,
			Enabled: item.Enabled, SortOrder: item.SortOrder, UpdatedBy: item.UpdatedBy,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	invitations, err := client.OperatorInvitation.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range invitations {
		out.OperatorInvitations = append(out.OperatorInvitations, store.OperatorInvitation{
			ID: item.ID, Email: item.Email, Role: item.Role.String(), TokenHash: item.TokenHash,
			Status: item.Status.String(), InvitedBy: item.InvitedBy, ExpiresAt: item.ExpiresAt,
			AcceptedAt: item.AcceptedAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}

	cohorts, err := client.ReleaseCohort.Query().All(ctx)
	if err != nil {
		return err
	}
	cohortByRollout := make(map[string]*ent.ReleaseCohort, len(cohorts))
	for _, cohort := range cohorts {
		cohortByRollout[cohort.RolloutID] = cohort
	}
	rollouts, err := client.ModelRollout.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range rollouts {
		kind, releaseID := releaseKindAndID(item.ModelReleaseID, item.RulesetReleaseID, item.NetworkRulesetReleaseID)
		rollout := store.ReleaseRollout{ID: item.ID, Kind: kind, ReleaseID: releaseID, Status: item.Status.String(), CreatedBy: item.CreatedBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
		if cohort := cohortByRollout[item.ID]; cohort != nil {
			rollout.Platform = cohort.Platform.String()
			rollout.Percentage = cohort.Percentage
			rollout.AppVersionConstraint = value(cohort.AppVersionConstraint)
		}
		out.ReleaseRollouts = append(out.ReleaseRollouts, rollout)
	}

	notifications, err := client.NotificationDelivery.Query().All(ctx)
	if err != nil {
		return err
	}
	for _, item := range notifications {
		out.NotificationEvents = append(out.NotificationEvents, store.NotificationItem{
			ID:        item.ID,
			Channel:   item.Channel.String(),
			Recipient: item.Recipient,
			Status:    item.Status.String(),
			Reason:    value(item.ApprovalRequestID),
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return nil
}

func releaseKindAndID(modelID, rulesetID, networkID *string) (string, string) {
	if modelID != nil {
		return "model", *modelID
	}
	if rulesetID != nil {
		return "ruleset", *rulesetID
	}
	if networkID != nil {
		return "network", *networkID
	}
	return "", ""
}
