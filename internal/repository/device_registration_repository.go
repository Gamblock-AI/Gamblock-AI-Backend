package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) CreateDevice(ctx context.Context, devID, userID, clientInstanceID, platformVal, label, appVersion, osVersion string, modelVersion, rulesetVersion *string) (model.Device, error) {
	if r.db == nil {
		now := time.Now().UTC()
		item := model.Device{
			ID: devID, UserID: userID, ClientInstanceID: clientInstanceID,
			Platform: platformVal, Label: label, AppVersion: appVersion, OSVersion: osVersion,
			ModelVersion: value(modelVersion), RulesetVersion: value(rulesetVersion),
			ProtectionStatus: "inactive", LastSeenAt: now, CreatedAt: now, UpdatedAt: now,
		}
		r.store.Lock()
		r.store.Devices = append(r.store.Devices, item)
		r.store.Unlock()
		return item, nil
	}
	item, err := r.db.Device.Create().
		SetID(devID).
		SetUserID(userID).
		SetClientInstanceID(clientInstanceID).
		SetPlatform(device.Platform(platformVal)).
		SetLabel(label).
		SetAppVersion(appVersion).
		SetOsVersion(osVersion).
		SetNillableModelVersion(modelVersion).
		SetNillableRulesetVersion(rulesetVersion).
		SetProtectionStatus(device.ProtectionStatusInactive).
		SetLastSeenAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return model.Device{}, err
	}
	r.RefreshStore(ctx)
	return deviceFromEnt(item), nil
}

func (r *Repository) UpsertDevice(ctx context.Context, devID, userID, clientInstanceID, platformVal, label, appVersion, osVersion string, modelVersion, rulesetVersion *string) (model.Device, error) {
	if r.db == nil {
		r.store.Lock()
		for index := range r.store.Devices {
			item := &r.store.Devices[index]
			if item.UserID != userID || item.ClientInstanceID != clientInstanceID {
				continue
			}
			item.Platform = platformVal
			item.Label = label
			item.AppVersion = appVersion
			item.OSVersion = osVersion
			item.ModelVersion = value(modelVersion)
			item.RulesetVersion = value(rulesetVersion)
			item.UpdatedAt = time.Now().UTC()
			item.LastSeenAt = item.UpdatedAt
			result := *item
			r.store.Unlock()
			return result, nil
		}
		r.store.Unlock()
		return r.CreateDevice(ctx, devID, userID, clientInstanceID, platformVal, label, appVersion, osVersion, modelVersion, rulesetVersion)
	}

	existing, err := r.db.Device.Query().Where(
		device.UserID(userID),
		device.ClientInstanceIDEQ(clientInstanceID),
	).Only(ctx)
	if err == nil {
		update := existing.Update().
			SetPlatform(device.Platform(platformVal)).
			SetLabel(label).
			SetAppVersion(appVersion).
			SetOsVersion(osVersion).
			SetLastSeenAt(time.Now().UTC())
		if modelVersion != nil {
			update.SetModelVersion(*modelVersion)
		}
		if rulesetVersion != nil {
			update.SetRulesetVersion(*rulesetVersion)
		}
		item, updateErr := update.Save(ctx)
		if updateErr != nil {
			return model.Device{}, updateErr
		}
		r.RefreshStore(ctx)
		return deviceFromEnt(item), nil
	}
	return r.CreateDevice(ctx, devID, userID, clientInstanceID, platformVal, label, appVersion, osVersion, modelVersion, rulesetVersion)
}
