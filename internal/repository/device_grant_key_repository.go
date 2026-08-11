package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
)

var ErrDeviceGrantKeyConflict = errors.New("device grant key is already bound")

func (r *Repository) BindOwnedDeviceGrantKey(ctx context.Context, userID, deviceID, publicJWK, thumbprint string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.Devices {
			item := &r.store.Devices[index]
			if item.ID != deviceID || item.UserID != userID {
				continue
			}
			if item.GrantKeyThumbprint != "" {
				if item.GrantKeyThumbprint == thumbprint && item.GrantPublicJWK == publicJWK {
					return nil
				}
				return ErrDeviceGrantKeyConflict
			}
			item.GrantPublicJWK = publicJWK
			item.GrantKeyThumbprint = thumbprint
			return nil
		}
		return fmt.Errorf("device not found")
	}

	item, err := r.db.Device.Query().Where(
		device.IDEQ(deviceID),
		device.UserID(userID),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("device not found")
	}
	if item.GrantKeyThumbprint != nil {
		if *item.GrantKeyThumbprint == thumbprint && item.GrantPublicJwk != nil && *item.GrantPublicJwk == publicJWK {
			return nil
		}
		return ErrDeviceGrantKeyConflict
	}
	changed, err := r.db.Device.Update().Where(
		device.IDEQ(deviceID),
		device.UserID(userID),
		device.GrantKeyThumbprintIsNil(),
	).SetGrantPublicJwk(publicJWK).SetGrantKeyThumbprint(thumbprint).Save(ctx)
	if err != nil {
		return err
	}
	if changed != 1 {
		reloaded, reloadErr := r.db.Device.Query().Where(
			device.IDEQ(deviceID),
			device.UserID(userID),
		).Only(ctx)
		if reloadErr == nil && reloaded.GrantKeyThumbprint != nil && reloaded.GrantPublicJwk != nil &&
			*reloaded.GrantKeyThumbprint == thumbprint && *reloaded.GrantPublicJwk == publicJWK {
			return nil
		}
		return ErrDeviceGrantKeyConflict
	}
	r.RefreshStore(ctx)
	return nil
}

func (r *Repository) OwnedDeviceGrantKeyThumbprint(ctx context.Context, userID, deviceID string) (string, error) {
	if r.db == nil {
		for _, item := range r.store.Snapshot().Devices {
			if item.ID == deviceID && item.UserID == userID {
				if item.GrantKeyThumbprint == "" {
					return "", fmt.Errorf("device grant key is not enrolled")
				}
				return item.GrantKeyThumbprint, nil
			}
		}
		return "", fmt.Errorf("device not found")
	}
	item, err := r.db.Device.Query().Where(
		device.IDEQ(deviceID),
		device.UserID(userID),
	).Only(ctx)
	if err != nil {
		return "", fmt.Errorf("device not found")
	}
	if item.GrantKeyThumbprint == nil || *item.GrantKeyThumbprint == "" {
		return "", fmt.Errorf("device grant key is not enrolled")
	}
	return *item.GrantKeyThumbprint, nil
}
