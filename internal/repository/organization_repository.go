package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateOrganization(ctx context.Context, id, name, slug, groupCode, createdBy string) (model.Organization, error) {
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		entry := model.Organization{
			ID: id, Name: name, Slug: slug, GroupCode: groupCode, Status: "active",
			CreatedBy: createdBy, Members: 0, CreatedAt: now, UpdatedAt: now,
		}
		r.store.Organizations = append(r.store.Organizations, entry)
		return entry, nil
	}
	organization, err := r.db.Organization.Create().
		SetID(id).
		SetName(name).
		SetSlug(slug).
		SetCreatedBy(createdBy).
		Save(ctx)
	if err != nil {
		return model.Organization{}, fmt.Errorf("gagal membuat organisasi: %w", err)
	}
	return organizationFromEnt(organization.ID, organization.Name, organization.Slug, groupCode, organization.Status.String(), organization.CreatedBy, organization.CreatedAt, organization.UpdatedAt), nil
}

func (r *Repository) GetOrganizationByID(ctx context.Context, id string) (model.Organization, error) {
	if r.db == nil {
		for _, organization := range r.store.Snapshot().Organizations {
			if organization.ID == id {
				return organization, nil
			}
		}
		return model.Organization{}, fmt.Errorf("organisasi tidak ditemukan")
	}
	organization, err := r.db.Organization.Get(ctx, id)
	if err != nil {
		return model.Organization{}, fmt.Errorf("organisasi tidak ditemukan")
	}
	return organizationFromEnt(organization.ID, organization.Name, organization.Slug, idToGroupCode(organization.ID), organization.Status.String(), organization.CreatedBy, organization.CreatedAt, organization.UpdatedAt), nil
}

func (r *Repository) GetOrganizationByUserID(ctx context.Context, userID string) (*model.Organization, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for index := range snapshot.Organizations {
			if snapshot.Organizations[index].CreatedBy == userID {
				return &snapshot.Organizations[index], nil
			}
		}
		return nil, fmt.Errorf("tidak ada grup")
	}
	member, err := r.db.OrganizationMember.Query().Where(organizationmember.UserIDEQ(userID)).First(ctx)
	if err != nil {
		return nil, fmt.Errorf("tidak ada grup")
	}
	organization, err := r.db.Organization.Get(ctx, member.OrganizationID)
	if err != nil {
		return nil, err
	}
	result := organizationFromEnt(organization.ID, organization.Name, organization.Slug, idToGroupCode(organization.ID), organization.Status.String(), organization.CreatedBy, organization.CreatedAt, organization.UpdatedAt)
	return &result, nil
}

func (r *Repository) GetOrganizationByGroupCode(ctx context.Context, code string) (model.Organization, error) {
	if r.db == nil {
		for _, organization := range r.store.Snapshot().Organizations {
			if organization.GroupCode == code {
				return organization, nil
			}
		}
		return model.Organization{}, fmt.Errorf("kode grup tidak valid")
	}
	organizations, err := r.db.Organization.Query().All(ctx)
	if err != nil {
		return model.Organization{}, fmt.Errorf("kode grup tidak valid")
	}
	for _, organization := range organizations {
		if idToGroupCode(organization.ID) == code {
			return organizationFromEnt(organization.ID, organization.Name, organization.Slug, code, organization.Status.String(), organization.CreatedBy, organization.CreatedAt, organization.UpdatedAt), nil
		}
	}
	return model.Organization{}, fmt.Errorf("kode grup tidak valid")
}

func organizationFromEnt(id, name, slug, groupCode, status, createdBy string, createdAt, updatedAt time.Time) model.Organization {
	return model.Organization{
		ID: id, Name: name, Slug: slug, GroupCode: groupCode, Status: status,
		CreatedBy: createdBy, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func idToGroupCode(id string) string {
	if len(id) < 6 {
		return id
	}
	return id[len(id)-6:]
}
