package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateDevice(ctx context.Context, devID, userID, platformVal, label, appVersion, osVersion string, modelVersion, rulesetVersion *string) (model.Device, error) {
	if r.db == nil {
		return model.Device{ID: devID, UserID: userID, Platform: platformVal, Label: label, AppVersion: appVersion, OSVersion: osVersion, ProtectionStatus: "active", LastSeenAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
	}
	item, err := r.db.Device.Create().
		SetID(devID).
		SetUserID(userID).
		SetPlatform(device.Platform(platformVal)).
		SetLabel(label).
		SetAppVersion(appVersion).
		SetOsVersion(osVersion).
		SetNillableModelVersion(modelVersion).
		SetNillableRulesetVersion(rulesetVersion).
		SetProtectionStatus(device.ProtectionStatusActive).
		SetLastSeenAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return model.Device{}, err
	}
	r.RefreshStore(ctx)
	return model.Device{
		ID:               item.ID,
		UserID:           item.UserID,
		Platform:         item.Platform.String(),
		Label:            item.Label,
		AppVersion:       item.AppVersion,
		OSVersion:        item.OsVersion,
		ModelVersion:     value(item.ModelVersion),
		RulesetVersion:   value(item.RulesetVersion),
		ProtectionStatus: item.ProtectionStatus.String(),
		LastSeenAt:       *item.LastSeenAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}, nil
}

func (r *Repository) UpdateDevice(ctx context.Context, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	if r.db == nil {
		return model.Device{ID: devID, Label: label, AppVersion: appVersion, OSVersion: osVersion, ProtectionStatus: status, LastSeenAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}, nil
	}
	update := r.db.Device.UpdateOneID(devID).SetLastSeenAt(time.Now().UTC())
	if label != "" {
		update.SetLabel(label)
	}
	if appVersion != "" {
		update.SetAppVersion(appVersion)
	}
	if osVersion != "" {
		update.SetOsVersion(osVersion)
	}
	if status != "" {
		update.SetProtectionStatus(device.ProtectionStatus(status))
	}
	if modelVersion != "" {
		update.SetModelVersion(modelVersion)
	}
	if rulesetVersion != "" {
		update.SetRulesetVersion(rulesetVersion)
	}
	item, err := update.Save(ctx)
	if err != nil {
		return model.Device{}, err
	}
	r.RefreshStore(ctx)
	return model.Device{
		ID:               item.ID,
		UserID:           item.UserID,
		Platform:         item.Platform.String(),
		Label:            item.Label,
		AppVersion:       item.AppVersion,
		OSVersion:        item.OsVersion,
		ModelVersion:     value(item.ModelVersion),
		RulesetVersion:   value(item.RulesetVersion),
		ProtectionStatus: item.ProtectionStatus.String(),
		LastSeenAt:       *item.LastSeenAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}, nil
}

func (r *Repository) RecordHeartbeat(ctx context.Context, deviceID string) error {
	now := time.Now().UTC()
	r.store.Lock()
	defer r.store.Unlock()
	for i, d := range r.store.Devices {
		if d.ID == deviceID {
			r.store.Devices[i].LastSeenAt = now
			return nil
		}
	}
	if r.db == nil {
		return nil
	}
	_, err := r.db.Device.UpdateOneID(deviceID).SetLastSeenAt(now).Save(ctx)
	return err
}
