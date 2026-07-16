package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateOrganizationMember(ctx context.Context, id, orgID, userID, role, status string, joinedAt *time.Time) error {
	if r.db == nil {
		return nil
	}
	create := r.db.OrganizationMember.Create().
		SetID(id).
		SetOrganizationID(orgID).
		SetUserID(userID).
		SetRole(organizationmember.Role(role)).
		SetStatus(organizationmember.Status(status))
	if joinedAt != nil {
		create.SetJoinedAt(*joinedAt)
	}
	return create.Exec(ctx)
}

func (r *Repository) GetOrganizationMember(ctx context.Context, orgID, userID string) (*model.OrganizationMember, error) {
	if r.db == nil {
		return nil, fmt.Errorf("not found")
	}
	member, err := r.db.OrganizationMember.Query().Where(
		organizationmember.OrganizationIDEQ(orgID),
		organizationmember.UserIDEQ(userID),
	).First(ctx)
	if err != nil {
		return nil, err
	}
	return &model.OrganizationMember{
		ID:             member.ID,
		OrganizationID: member.OrganizationID,
		UserID:         member.UserID,
		Role:           member.Role.String(),
		Status:         member.Status.String(),
		JoinedAt:       member.JoinedAt,
		CreatedAt:      member.CreatedAt,
	}, nil
}

func (r *Repository) ListOrganizationMembers(ctx context.Context, orgID string) ([]model.OrganizationMember, error) {
	if r.db == nil {
		var result []model.OrganizationMember
		for _, user := range r.store.Snapshot().Users {
			result = append(result, model.OrganizationMember{
				UserID: user.ID, UserName: user.DisplayName, UserEmail: user.Email,
				Role: "member", Status: "active",
			})
		}
		return result, nil
	}
	members, err := r.db.OrganizationMember.Query().Where(organizationmember.OrganizationIDEQ(orgID)).All(ctx)
	if err != nil {
		return nil, err
	}
	var result []model.OrganizationMember
	for _, member := range members {
		userName := ""
		userEmail := ""
		if user, userErr := r.db.User.Get(ctx, member.UserID); userErr == nil {
			userName = user.DisplayName
			userEmail = user.Email
		}
		result = append(result, model.OrganizationMember{
			ID:             member.ID,
			OrganizationID: member.OrganizationID,
			UserID:         member.UserID,
			UserName:       userName,
			UserEmail:      userEmail,
			Role:           member.Role.String(),
			Status:         member.Status.String(),
			JoinedAt:       member.JoinedAt,
			CreatedAt:      member.CreatedAt,
		})
	}
	return result, nil
}

func (r *Repository) RemoveOrganizationMember(ctx context.Context, orgID, userID string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.OrganizationMember.Delete().Where(
		organizationmember.OrganizationIDEQ(orgID),
		organizationmember.UserIDEQ(userID),
	).Exec(ctx)
	return err
}
