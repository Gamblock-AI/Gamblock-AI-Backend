package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent/contactverification"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (r *Repository) SaveContactVerification(ctx context.Context, item model.ContactVerification) error {
	if r.db == nil {
		r.store.Lock()
		r.store.ContactVerifications = append(r.store.ContactVerifications, item)
		r.store.Unlock()
		return nil
	}
	_, err := r.db.ContactVerification.Create().SetID(item.ID).SetUserID(item.UserID).
		SetKind(contactverification.Kind(item.Kind)).SetDestination(item.Destination).
		SetTokenHash(item.TokenHash).SetExpiresAt(item.ExpiresAt).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) ConsumeContactVerification(ctx context.Context, tokenHash, kind string, now time.Time) (model.ContactVerification, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.ContactVerifications {
			item := &r.store.ContactVerifications[i]
			if item.TokenHash == tokenHash && item.Kind == kind && item.ConsumedAt == nil && now.Before(item.ExpiresAt) {
				item.ConsumedAt = &now
				return *item, nil
			}
		}
		return model.ContactVerification{}, fmt.Errorf("verification token is invalid or expired")
	}
	row, err := r.db.ContactVerification.Query().Where(
		contactverification.TokenHashEQ(tokenHash),
		contactverification.KindEQ(contactverification.Kind(kind)),
		contactverification.ConsumedAtIsNil(),
		contactverification.ExpiresAtGT(now),
	).Only(ctx)
	if err != nil {
		return model.ContactVerification{}, fmt.Errorf("verification token is invalid or expired")
	}
	row, err = row.Update().SetConsumedAt(now).Save(ctx)
	if err != nil {
		return model.ContactVerification{}, err
	}
	r.RefreshStore(ctx)
	return model.ContactVerification{
		ID: row.ID, UserID: row.UserID, Kind: row.Kind.String(), Destination: row.Destination,
		TokenHash: row.TokenHash, AttemptCount: row.AttemptCount, ExpiresAt: row.ExpiresAt,
		ConsumedAt: row.ConsumedAt, CreatedAt: row.CreatedAt,
	}, nil
}

func (r *Repository) MarkEmailVerified(ctx context.Context, userID string, verifiedAt time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.Users {
			if r.store.Users[i].ID == userID {
				r.store.Users[i].EmailVerifiedAt = &verifiedAt
				r.store.Users[i].UpdatedAt = verifiedAt
				return nil
			}
		}
		return fmt.Errorf("user not found")
	}
	_, err := r.db.User.UpdateOneID(userID).SetEmailVerifiedAt(verifiedAt).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) MarkPhoneVerified(ctx context.Context, userID, phone string, verifiedAt time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.Users {
			if r.store.Users[i].ID == userID {
				r.store.Users[i].PhoneE164 = phone
				r.store.Users[i].PhoneVerifiedAt = &verifiedAt
				r.store.Users[i].UpdatedAt = verifiedAt
				return nil
			}
		}
		return fmt.Errorf("user not found")
	}
	_, err := r.db.User.UpdateOneID(userID).SetPhoneE164(phone).SetPhoneVerifiedAt(verifiedAt).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

func (r *Repository) SetPendingPhone(ctx context.Context, userID, phone string) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for i := range r.store.Users {
			if r.store.Users[i].ID == userID {
				r.store.Users[i].PhoneE164 = phone
				r.store.Users[i].PhoneVerifiedAt = nil
				return nil
			}
		}
		return fmt.Errorf("user not found")
	}
	_, err := r.db.User.UpdateOneID(userID).SetPhoneE164(phone).ClearPhoneVerifiedAt().Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}
