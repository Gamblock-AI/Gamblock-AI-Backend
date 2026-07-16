package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateDevice(ctx context.Context, devID, userID, platformVal, label, appVersion, osVersion string, modelVersion, rulesetVersion *string) (model.Device, error) {
	if r.db == nil {
		now := time.Now().UTC()
		item := model.Device{ID: devID, UserID: userID, Platform: platformVal, Label: label, AppVersion: appVersion, OSVersion: osVersion, ModelVersion: value(modelVersion), RulesetVersion: value(rulesetVersion), ProtectionStatus: "active", LastSeenAt: now, CreatedAt: now, UpdatedAt: now}
		r.store.Lock()
		r.store.Devices = append(r.store.Devices, item)
		r.store.Unlock()
		return item, nil
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
	return r.UpdateOwnedDevice(ctx, "", devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion)
}

func (r *Repository) UpdateOwnedDevice(ctx context.Context, userID, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.Devices {
			item := &r.store.Devices[i]
			if item.ID != devID || (userID != "" && item.UserID != userID) {
				continue
			}
			if label != "" {
				item.Label = label
			}
			if appVersion != "" {
				item.AppVersion = appVersion
			}
			if osVersion != "" {
				item.OSVersion = osVersion
			}
			if status != "" {
				item.ProtectionStatus = status
			}
			if modelVersion != "" {
				item.ModelVersion = modelVersion
			}
			if rulesetVersion != "" {
				item.RulesetVersion = rulesetVersion
			}
			item.LastSeenAt = time.Now().UTC()
			item.UpdatedAt = item.LastSeenAt
			return *item, nil
		}
		return model.Device{}, fmt.Errorf("device not found")
	}
	if userID != "" {
		owned, err := r.db.Device.Query().Where(device.IDEQ(devID), device.UserID(userID)).Exist(ctx)
		if err != nil || !owned {
			return model.Device{}, fmt.Errorf("device not found")
		}
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
	return r.RecordOwnedHeartbeat(ctx, "", deviceID)
}

func (r *Repository) RecordOwnedHeartbeat(ctx context.Context, userID, deviceID string) error {
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i, d := range r.store.Devices {
			if d.ID == deviceID && (userID == "" || d.UserID == userID) {
				r.store.Devices[i].LastSeenAt = now
				r.store.Devices[i].UpdatedAt = now
				return nil
			}
		}
		return fmt.Errorf("device not found")
	}
	if userID != "" {
		owned, err := r.db.Device.Query().Where(device.IDEQ(deviceID), device.UserID(userID)).Exist(ctx)
		if err != nil || !owned {
			return fmt.Errorf("device not found")
		}
	}
	_, err := r.db.Device.UpdateOneID(deviceID).SetLastSeenAt(now).Save(ctx)
	return err
}

func (r *Repository) IsDeviceOwnedBy(ctx context.Context, deviceID, userID string) bool {
	if deviceID == "" {
		return true
	}
	if r.db == nil {
		snapshot := r.store.Snapshot()
		for _, item := range snapshot.Devices {
			if item.ID == deviceID && item.UserID == userID {
				return true
			}
		}
		return false
	}
	exists, err := r.db.Device.Query().Where(device.IDEQ(deviceID), device.UserID(userID)).Exist(ctx)
	return err == nil && exists
}
