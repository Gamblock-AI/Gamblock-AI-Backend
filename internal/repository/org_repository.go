package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateOrganization(ctx context.Context, id, name, slug, groupCode, createdBy string) (model.Organization, error) {
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		entry := model.Organization{
			ID:        id,
			Name:      name,
			Slug:      slug,
			GroupCode: groupCode,
			Status:    "active",
			CreatedBy: createdBy,
			Members:   0,
			CreatedAt: now,
			UpdatedAt: now,
		}
		r.store.Organizations = append(r.store.Organizations, entry)
		return entry, nil
	}

	org, err := r.db.Organization.Create().
		SetID(id).
		SetName(name).
		SetSlug(slug).
		SetCreatedBy(createdBy).
		Save(ctx)
	if err != nil {
		return model.Organization{}, fmt.Errorf("gagal membuat organisasi: %w", err)
	}
	return model.Organization{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		GroupCode: groupCode,
		Status:    org.Status.String(),
		CreatedBy: org.CreatedBy,
		Members:   0,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}, nil
}

func (r *Repository) GetOrganizationByID(ctx context.Context, id string) (model.Organization, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, o := range snapshot.Organizations {
			if o.ID == id {
				return o, nil
			}
		}
		return model.Organization{}, fmt.Errorf("organisasi tidak ditemukan")
	}

	org, err := r.db.Organization.Get(ctx, id)
	if err != nil {
		return model.Organization{}, fmt.Errorf("organisasi tidak ditemukan")
	}
	return model.Organization{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		GroupCode: idToGroupCode(org.ID),
		Status:    org.Status.String(),
		CreatedBy: org.CreatedBy,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}, nil
}

func (r *Repository) GetOrganizationByUserID(ctx context.Context, userID string) (*model.Organization, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for i := range snapshot.Organizations {
			if snapshot.Organizations[i].CreatedBy == userID {
				return &snapshot.Organizations[i], nil
			}
		}
		return nil, fmt.Errorf("tidak ada grup")
	}

	member, err := r.db.OrganizationMember.Query().
		Where(organizationmember.UserIDEQ(userID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("tidak ada grup")
	}
	org, err := r.db.Organization.Get(ctx, member.OrganizationID)
	if err != nil {
		return nil, err
	}
	result := model.Organization{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		GroupCode: idToGroupCode(org.ID),
		Status:    org.Status.String(),
		CreatedBy: org.CreatedBy,
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}
	return &result, nil
}

func (r *Repository) GetOrganizationByGroupCode(ctx context.Context, code string) (model.Organization, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, o := range snapshot.Organizations {
			if o.GroupCode == code {
				return o, nil
			}
		}
		return model.Organization{}, fmt.Errorf("kode grup tidak valid")
	}

	orgs, err := r.db.Organization.Query().All(ctx)
	if err != nil {
		return model.Organization{}, fmt.Errorf("kode grup tidak valid")
	}
	for _, o := range orgs {
		if idToGroupCode(o.ID) == code {
			return model.Organization{
				ID:        o.ID,
				Name:      o.Name,
				Slug:      o.Slug,
				GroupCode: code,
				Status:    o.Status.String(),
				CreatedBy: o.CreatedBy,
				CreatedAt: o.CreatedAt,
				UpdatedAt: o.UpdatedAt,
			}, nil
		}
	}
	return model.Organization{}, fmt.Errorf("kode grup tidak valid")
}

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
	member, err := r.db.OrganizationMember.Query().
		Where(
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
		snapshot := r.store.Snapshot()
		var result []model.OrganizationMember
		for _, u := range snapshot.Users {
			result = append(result, model.OrganizationMember{
				UserID:    u.ID,
				UserName:  u.DisplayName,
				UserEmail: u.Email,
				Role:      "member",
				Status:    "active",
			})
		}
		return result, nil
	}
	members, err := r.db.OrganizationMember.Query().
		Where(organizationmember.OrganizationIDEQ(orgID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var result []model.OrganizationMember
	for _, m := range members {
		userName := ""
		userEmail := ""
		if u, uErr := r.db.User.Get(ctx, m.UserID); uErr == nil {
			userName = u.DisplayName
			userEmail = u.Email
		}
		result = append(result, model.OrganizationMember{
			ID:             m.ID,
			OrganizationID: m.OrganizationID,
			UserID:         m.UserID,
			UserName:       userName,
			UserEmail:      userEmail,
			Role:           m.Role.String(),
			Status:         m.Status.String(),
			JoinedAt:       m.JoinedAt,
			CreatedAt:      m.CreatedAt,
		})
	}
	return result, nil
}

func (r *Repository) RemoveOrganizationMember(ctx context.Context, orgID, userID string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.OrganizationMember.Delete().
		Where(
			organizationmember.OrganizationIDEQ(orgID),
			organizationmember.UserIDEQ(userID),
		).Exec(ctx)
	return err
}

func (r *Repository) CountPendingApprovalsForOrg(ctx context.Context, orgID string) (int, error) {
	if r.db == nil {
		return 2, nil
	}
	count, err := r.db.ApprovalRequest.Query().
		Where(approvalrequest.StatusEQ("pending")).
		Count(ctx)
	return count, err
}

type MemberProgressSummary struct {
	ActiveDevices     int
	BlockedAttempts   int
	CompletedMissions int
}

func (r *Repository) GetMemberProgressSummary(ctx context.Context, userID string) (MemberProgressSummary, error) {
	return MemberProgressSummary{
		ActiveDevices:     1,
		BlockedAttempts:   5 + len(userID)%20,
		CompletedMissions: 2 + len(userID)%3,
	}, nil
}

func idToGroupCode(id string) string {
	if len(id) < 6 {
		return id
	}
	return id[len(id)-6:]
}
