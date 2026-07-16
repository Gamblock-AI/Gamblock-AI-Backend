package repository

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/modelrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/rulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) GetPortalOverview(ctx context.Context) (model.PortalOverview, error) {
	if r.db == nil {
		snapshot := r.store.Snapshot()
		protected := make(map[string]struct{})
		healthy := 0
		for _, item := range snapshot.Devices {
			if item.ProtectionStatus == "active" {
				healthy++
				protected[item.UserID] = struct{}{}
			}
		}
		pending := 0
		for _, item := range snapshot.Approvals {
			if item.Status == "pending" || item.Status == "Pending partner approval" {
				pending++
			}
		}
		openSupport := 0
		for _, item := range snapshot.SupportCases {
			if item.Status != "resolved" && item.Status != "closed" {
				openSupport++
			}
		}
		overview := model.PortalOverview{
			ProtectedUsers: len(protected), PartnerApprovals: pending,
			HealthyDevicesPercent: percentage(healthy, len(snapshot.Devices)),
			OpenSupport:           openSupport,
		}
		for _, item := range snapshot.ModelReleases {
			if item.Status == "published" {
				overview.ModelRelease = item.Version
				break
			}
		}
		for _, item := range snapshot.RulesetReleases {
			if item.Status == "published" {
				overview.RulesetRelease = item.Version
				break
			}
		}
		return overview, nil
	}

	allDevices, err := r.db.Device.Query().Count(ctx)
	if err != nil {
		return model.PortalOverview{}, err
	}
	healthyDevices, err := r.db.Device.Query().Where(device.ProtectionStatusEQ(device.ProtectionStatusActive)).All(ctx)
	if err != nil {
		return model.PortalOverview{}, err
	}
	protected := make(map[string]struct{}, len(healthyDevices))
	for _, item := range healthyDevices {
		protected[item.UserID] = struct{}{}
	}
	pending, err := r.db.ApprovalRequest.Query().Where(approvalrequest.StatusEQ(approvalrequest.StatusPending)).Count(ctx)
	if err != nil {
		return model.PortalOverview{}, err
	}
	openSupport, err := r.db.SupportCase.Query().Where(
		supportcase.StatusNotIn(supportcase.StatusResolved, supportcase.StatusClosed),
	).Count(ctx)
	if err != nil {
		return model.PortalOverview{}, err
	}

	overview := model.PortalOverview{
		ProtectedUsers: len(protected), PartnerApprovals: pending,
		HealthyDevicesPercent: percentage(len(healthyDevices), allDevices),
		OpenSupport:           openSupport,
	}
	if item, queryErr := r.db.ModelRelease.Query().Where(modelrelease.StatusEQ(modelrelease.StatusPublished)).Order(ent.Desc(modelrelease.FieldPublishedAt)).First(ctx); queryErr == nil {
		overview.ModelRelease = item.Version
	}
	if item, queryErr := r.db.RulesetRelease.Query().Where(rulesetrelease.StatusEQ(rulesetrelease.StatusPublished)).Order(ent.Desc(rulesetrelease.FieldPublishedAt)).First(ctx); queryErr == nil {
		overview.RulesetRelease = item.Version
	}
	return overview, nil
}

func percentage(value, total int) int {
	if total == 0 {
		return 0
	}
	return int(float64(value)/float64(total)*100 + 0.5)
}
