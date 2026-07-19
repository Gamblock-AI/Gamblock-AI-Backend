package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/modelrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/modelrollout"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/networkrulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/releasecohort"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/rulesetrelease"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func (r *Repository) ListReleaseRollouts(ctx context.Context) ([]model.ReleaseRollout, error) {
	if r.db == nil {
		items := append([]model.ReleaseRollout(nil), r.store.Snapshot().ReleaseRollouts...)
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
		return items, nil
	}
	rows, err := r.db.ModelRollout.Query().Order(ent.Desc(modelrollout.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]model.ReleaseRollout, 0, len(rows))
	for _, row := range rows {
		cohort, cohortErr := r.db.ReleaseCohort.Query().Where(releasecohort.RolloutIDEQ(row.ID)).Only(ctx)
		if cohortErr != nil {
			return nil, cohortErr
		}
		kind, releaseID := rolloutReleaseKindAndID(row.ModelReleaseID, row.RulesetReleaseID, row.NetworkRulesetReleaseID)
		version, versionErr := r.releaseVersion(ctx, kind, releaseID)
		if versionErr != nil {
			return nil, versionErr
		}
		items = append(items, model.ReleaseRollout{ID: row.ID, Kind: kind, ReleaseID: releaseID,
			ReleaseVersion: version, Status: row.Status.String(), Platform: cohort.Platform.String(),
			Percentage: cohort.Percentage, AppVersionConstraint: value(cohort.AppVersionConstraint),
			CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return items, nil
}

func rolloutReleaseKindAndID(modelReleaseID, rulesetReleaseID, networkReleaseID *string) (string, string) {
	if modelReleaseID != nil {
		return "model", *modelReleaseID
	}
	if rulesetReleaseID != nil {
		return "ruleset", *rulesetReleaseID
	}
	if networkReleaseID != nil {
		return "network", *networkReleaseID
	}
	return "", ""
}

func (r *Repository) CreateReleaseRollout(ctx context.Context, item model.ReleaseRollout) (model.ReleaseRollout, error) {
	now := time.Now().UTC()
	item.Status, item.CreatedAt, item.UpdatedAt = "staged", now, now
	version, err := r.releaseVersion(ctx, item.Kind, item.ReleaseID)
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	item.ReleaseVersion = version
	if r.db == nil {
		if err := r.setMemoryReleaseStatus(item.Kind, item.ReleaseID, "staged", now); err != nil {
			return model.ReleaseRollout{}, err
		}
		r.store.Lock()
		r.store.ReleaseRollouts = append(r.store.ReleaseRollouts, store.ReleaseRollout(item))
		r.store.Unlock()
		return item, nil
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	creator := tx.ModelRollout.Create().SetID(item.ID).SetStatus(modelrollout.StatusStaged).
		SetCreatedBy(item.CreatedBy).SetCohortJSON(map[string]any{"platform": item.Platform, "percentage": item.Percentage, "app_version_constraint": item.AppVersionConstraint})
	switch item.Kind {
	case "model":
		creator.SetModelReleaseID(item.ReleaseID)
	case "ruleset":
		creator.SetRulesetReleaseID(item.ReleaseID)
	case "network":
		creator.SetNetworkRulesetReleaseID(item.ReleaseID)
	default:
		_ = tx.Rollback()
		return model.ReleaseRollout{}, fmt.Errorf("invalid release kind")
	}
	if _, err = creator.Save(ctx); err != nil {
		_ = tx.Rollback()
		return model.ReleaseRollout{}, err
	}
	cohortCreator := tx.ReleaseCohort.Create().SetID("cohort_" + item.ID).SetRolloutID(item.ID).
		SetPlatform(releasecohort.Platform(item.Platform)).SetPercentage(item.Percentage)
	if item.AppVersionConstraint != "" {
		cohortCreator.SetAppVersionConstraint(item.AppVersionConstraint)
	}
	if _, err = cohortCreator.Save(ctx); err != nil {
		_ = tx.Rollback()
		return model.ReleaseRollout{}, err
	}
	if err = setTxReleaseStatus(ctx, tx, item.Kind, item.ReleaseID, "staged", now); err != nil {
		_ = tx.Rollback()
		return model.ReleaseRollout{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.ReleaseRollout{}, err
	}
	r.RefreshStore(ctx)
	return item, nil
}

func (r *Repository) TransitionReleaseRollout(ctx context.Context, rolloutID, status string, now time.Time) (model.ReleaseRollout, error) {
	items, err := r.ListReleaseRollouts(ctx)
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	var current model.ReleaseRollout
	for _, item := range items {
		if item.ID == rolloutID {
			current = item
			break
		}
	}
	if current.ID == "" {
		return model.ReleaseRollout{}, fmt.Errorf("rollout not found")
	}
	releaseStatus := map[string]string{"active": "published", "paused": "paused", "completed": "published", "rolled_back": "rolled_back"}[status]
	if releaseStatus == "" {
		return model.ReleaseRollout{}, fmt.Errorf("invalid rollout transition")
	}
	if r.db == nil {
		r.store.Lock()
		for index := range r.store.ReleaseRollouts {
			if r.store.ReleaseRollouts[index].ID == rolloutID {
				r.store.ReleaseRollouts[index].Status = status
				r.store.ReleaseRollouts[index].UpdatedAt = now
				current = r.store.ReleaseRollouts[index]
			}
		}
		r.store.Unlock()
		if err := r.setMemoryReleaseStatus(current.Kind, current.ReleaseID, releaseStatus, now); err != nil {
			return model.ReleaseRollout{}, err
		}
		return current, nil
	}
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return model.ReleaseRollout{}, err
	}
	if _, err = tx.ModelRollout.UpdateOneID(rolloutID).SetStatus(modelrollout.Status(status)).Save(ctx); err != nil {
		_ = tx.Rollback()
		return model.ReleaseRollout{}, err
	}
	if err = setTxReleaseStatus(ctx, tx, current.Kind, current.ReleaseID, releaseStatus, now); err != nil {
		_ = tx.Rollback()
		return model.ReleaseRollout{}, err
	}
	if err = tx.Commit(); err != nil {
		return model.ReleaseRollout{}, err
	}
	r.RefreshStore(ctx)
	current.Status, current.UpdatedAt = status, now
	return current, nil
}

func (r *Repository) releaseVersion(ctx context.Context, kind, releaseID string) (string, error) {
	if r.db == nil {
		groups := map[string][]model.Release{"model": r.store.Snapshot().ModelReleases, "ruleset": r.store.Snapshot().RulesetReleases, "network": r.store.Snapshot().NetworkRulesets}
		for _, item := range groups[kind] {
			if item.ID == releaseID {
				return item.Version, nil
			}
		}
		return "", fmt.Errorf("release not found")
	}
	switch kind {
	case "model":
		row, err := r.db.ModelRelease.Query().Where(modelrelease.IDEQ(releaseID)).Only(ctx)
		if err != nil {
			return "", fmt.Errorf("release not found")
		}
		return row.Version, nil
	case "ruleset":
		row, err := r.db.RulesetRelease.Query().Where(rulesetrelease.IDEQ(releaseID)).Only(ctx)
		if err != nil {
			return "", fmt.Errorf("release not found")
		}
		return row.Version, nil
	case "network":
		row, err := r.db.NetworkRulesetRelease.Query().Where(networkrulesetrelease.IDEQ(releaseID)).Only(ctx)
		if err != nil {
			return "", fmt.Errorf("release not found")
		}
		return row.Version, nil
	default:
		return "", fmt.Errorf("invalid release kind")
	}
}

func (r *Repository) setMemoryReleaseStatus(kind, releaseID, status string, now time.Time) error {
	r.store.Lock()
	defer r.store.Unlock()
	var releases *[]store.Release
	switch kind {
	case "model":
		releases = &r.store.ModelReleases
	case "ruleset":
		releases = &r.store.RulesetReleases
	case "network":
		releases = &r.store.NetworkRulesets
	default:
		return fmt.Errorf("invalid release kind")
	}
	for index := range *releases {
		if (*releases)[index].ID == releaseID {
			(*releases)[index].Status, (*releases)[index].UpdatedAt = status, now
			if status == "published" {
				(*releases)[index].PublishedAtText = "Published"
			}
			return nil
		}
	}
	return fmt.Errorf("release not found")
}

func setTxReleaseStatus(ctx context.Context, tx *ent.Tx, kind, releaseID, status string, now time.Time) error {
	switch kind {
	case "model":
		update := tx.ModelRelease.UpdateOneID(releaseID).SetStatus(modelrelease.Status(status))
		if status == "published" {
			update.SetPublishedAt(now)
		}
		_, err := update.Save(ctx)
		return err
	case "ruleset":
		update := tx.RulesetRelease.UpdateOneID(releaseID).SetStatus(rulesetrelease.Status(status))
		if status == "published" {
			update.SetPublishedAt(now)
		}
		_, err := update.Save(ctx)
		return err
	case "network":
		update := tx.NetworkRulesetRelease.UpdateOneID(releaseID).SetStatus(networkrulesetrelease.Status(status))
		if status == "published" {
			update.SetPublishedAt(now)
		}
		_, err := update.Save(ctx)
		return err
	default:
		return fmt.Errorf("invalid release kind")
	}
}
