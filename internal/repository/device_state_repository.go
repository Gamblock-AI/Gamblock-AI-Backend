package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) UpdateDevice(ctx context.Context, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	return r.UpdateOwnedDevice(ctx, "", devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion)
}

func (r *Repository) UpdateOwnedDevice(ctx context.Context, userID, devID, label, appVersion, osVersion, status, modelVersion, rulesetVersion string) (model.Device, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Devices {
			item := &r.store.Devices[index]
			if item.ID != devID || (userID != "" && item.UserID != userID) {
				continue
			}
			applyDeviceUpdate(item, label, appVersion, osVersion, status, modelVersion, rulesetVersion, time.Now().UTC())
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
	return deviceFromEnt(item), nil
}

func (r *Repository) RecordHeartbeat(ctx context.Context, deviceID string) error {
	return r.RecordOwnedHeartbeat(ctx, "", deviceID)
}

func (r *Repository) RecordOwnedHeartbeat(ctx context.Context, userID, deviceID string) error {
	now := time.Now().UTC()
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index, item := range r.store.Devices {
			if item.ID == deviceID && (userID == "" || item.UserID == userID) {
				r.store.Devices[index].LastSeenAt = now
				r.store.Devices[index].UpdatedAt = now
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
		return false
	}
	if r.db == nil {
		for _, item := range r.store.Snapshot().Devices {
			if item.ID == deviceID && item.UserID == userID {
				return true
			}
		}
		return false
	}
	exists, err := r.db.Device.Query().Where(device.IDEQ(deviceID), device.UserID(userID)).Exist(ctx)
	return err == nil && exists
}

func applyDeviceUpdate(item *model.Device, label, appVersion, osVersion, status, modelVersion, rulesetVersion string, now time.Time) {
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
	item.LastSeenAt = now
	item.UpdatedAt = now
}
