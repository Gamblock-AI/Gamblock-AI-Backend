package repository

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/refreshtoken"
)

func (r *Repository) CreateRefreshToken(ctx context.Context, rtID, userID, tokenHash string, deviceID *string, expiresAt time.Time) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.RefreshToken.Create().
		SetID(rtID).
		SetUserID(userID).
		SetTokenHash(tokenHash).
		SetNillableDeviceID(deviceID).
		SetExpiresAt(expiresAt).
		Save(ctx)
	return err
}

func (r *Repository) GetActiveRefreshToken(ctx context.Context, tokenHash string) (rtID, userID string, deviceID *string, err error) {
	if r.db == nil {
		return "", "", nil, nil
	}
	now := time.Now().UTC()
	existing, err := r.db.RefreshToken.Query().
		Where(
			refreshtoken.TokenHashEQ(tokenHash),
			refreshtoken.RevokedAtIsNil(),
			refreshtoken.ExpiresAtGT(now),
		).
		Only(ctx)
	if err != nil {
		return "", "", nil, err
	}
	return existing.ID, existing.UserID, existing.DeviceID, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.RefreshToken.Update().
		Where(refreshtoken.TokenHashEQ(tokenHash), refreshtoken.RevokedAtIsNil()).
		SetRevokedAt(time.Now().UTC()).
		Save(ctx)
	return err
}

func (r *Repository) RevokeRefreshTokenByID(ctx context.Context, id string) error {
	if r.db == nil {
		return nil
	}
	_, err := r.db.RefreshToken.UpdateOneID(id).SetRevokedAt(time.Now().UTC()).Save(ctx)
	return err
}
