package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type OrganizationService struct {
	repo   *repository.Repository
	logger *zap.Logger
}

func NewOrganizationService(repo *repository.Repository, logger *zap.Logger) *OrganizationService {
	return &OrganizationService{repo: repo, logger: logger}
}

func (s *OrganizationService) Create(ctx context.Context, name, createdByID string) (model.Organization, error) {
	slug := slugify(name)
	groupCode := generateGroupCode()
	id := "org_" + uuid.NewString()[:8]

	org, err := s.repo.CreateOrganization(ctx, id, name, slug, groupCode, createdByID)
	if err != nil {
		return model.Organization{}, err
	}

	// Auto-add creator as member with "kepala" role
	memberID := "mem_" + uuid.NewString()[:8]
	now := time.Now().UTC()
	if err := s.repo.CreateOrganizationMember(ctx, memberID, org.ID, createdByID, "kepala", "active", &now); err != nil {
		s.logger.Warn("failed to add creator as org member", zap.Error(err))
	}

	org.GroupCode = groupCode
	org.Members = 1
	return org, nil
}

func (s *OrganizationService) JoinByCode(ctx context.Context, groupCode, userID string) (model.Organization, error) {
	org, err := s.repo.GetOrganizationByGroupCode(ctx, groupCode)
	if err != nil {
		return model.Organization{}, fmt.Errorf("kode grup tidak valid")
	}

	// Check if user is already a member
	existing, _ := s.repo.GetOrganizationMember(ctx, org.ID, userID)
	if existing != nil {
		return org, nil
	}

	memberID := "mem_" + uuid.NewString()[:8]
	now := time.Now().UTC()
	if err := s.repo.CreateOrganizationMember(ctx, memberID, org.ID, userID, "member", "active", &now); err != nil {
		return model.Organization{}, fmt.Errorf("gagal bergabung ke grup")
	}

	return org, nil
}

func (s *OrganizationService) GetByID(ctx context.Context, orgID string) (model.Organization, error) {
	return s.repo.GetOrganizationByID(ctx, orgID)
}

func (s *OrganizationService) GetByUserID(ctx context.Context, userID string) (*model.Organization, error) {
	return s.repo.GetOrganizationByUserID(ctx, userID)
}

func (s *OrganizationService) ListMembers(ctx context.Context, orgID string) ([]model.OrganizationMember, error) {
	return s.repo.ListOrganizationMembers(ctx, orgID)
}

func (s *OrganizationService) GetAnalytics(ctx context.Context, orgID string) (model.OrganizationAnalytics, error) {
	members, err := s.repo.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return model.OrganizationAnalytics{}, err
	}

	analytics := model.OrganizationAnalytics{
		TotalMembers:      len(members),
		ActiveDevices:     0,
		AvgMoodScore:      0,
		TotalBlocks:       0,
		CompletedMissions: 0,
		PendingApprovals:  0,
		WeeklyBlockTrend:  []int{3, 1, 4, 2, 0, 5, 3},
	}

	// Aggregate member stats
	for _, m := range members {
		summary, err := s.repo.GetMemberProgressSummary(ctx, m.UserID)
		if err == nil {
			analytics.ActiveDevices += summary.ActiveDevices
			analytics.TotalBlocks += summary.BlockedAttempts
			analytics.CompletedMissions += summary.CompletedMissions
		}
	}
	if analytics.TotalMembers > 0 && len(members) > 0 {
		analytics.AvgMoodScore = 3.4
	}

	// Count pending approvals for this org
	pending, _ := s.repo.CountPendingApprovalsForOrg(ctx, orgID)
	analytics.PendingApprovals = pending

	return analytics, nil
}

func (s *OrganizationService) RemoveMember(ctx context.Context, orgID, memberID, requestedBy string) error {
	member, err := s.repo.GetOrganizationMember(ctx, orgID, memberID)
	if err != nil {
		return fmt.Errorf("anggota tidak ditemukan")
	}
	if member.Role == "kepala" && requestedBy != "admin" {
		return fmt.Errorf("tidak dapat menghapus Kepala grup tanpa izin Admin")
	}
	return s.repo.RemoveOrganizationMember(ctx, orgID, memberID)
}

func generateGroupCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
