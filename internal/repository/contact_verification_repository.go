package repository

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
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

// InvalidateContactVerifications makes all earlier active codes for the same
// purpose unusable before a replacement is created.
func (r *Repository) InvalidateContactVerifications(ctx context.Context, kind, destination string, now time.Time) error {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		for index := range r.store.ContactVerifications {
			item := &r.store.ContactVerifications[index]
			if item.Kind == kind && item.Destination == destination && item.ConsumedAt == nil {
				item.ConsumedAt = &now
			}
		}
		return nil
	}
	_, err := r.db.ContactVerification.Update().Where(
		contactverification.KindEQ(contactverification.Kind(kind)),
		contactverification.DestinationEQ(destination),
		contactverification.ConsumedAtIsNil(),
	).SetConsumedAt(now).Save(ctx)
	if err == nil {
		r.RefreshStore(ctx)
	}
	return err
}

// VerifyLatestContactCode checks only the newest active code and consumes it
// after a match. Invalid attempts are counted and capped without storing or
// returning the plaintext code.
func (r *Repository) VerifyLatestContactCode(ctx context.Context, kind, destination, tokenHash string, now time.Time, maxAttempts int) (model.ContactVerification, error) {
	if r.db == nil {
		r.store.Lock()
		defer r.store.Unlock()
		latest := -1
		for index := range r.store.ContactVerifications {
			item := r.store.ContactVerifications[index]
			if item.Kind == kind && item.Destination == destination && item.ConsumedAt == nil && now.Before(item.ExpiresAt) && (latest < 0 || item.CreatedAt.After(r.store.ContactVerifications[latest].CreatedAt)) {
				latest = index
			}
		}
		if latest < 0 {
			return model.ContactVerification{}, fmt.Errorf("verification code is invalid or expired")
		}
		item := &r.store.ContactVerifications[latest]
		if item.AttemptCount >= maxAttempts {
			return model.ContactVerification{}, fmt.Errorf("verification attempt limit reached")
		}
		if item.TokenHash != tokenHash {
			item.AttemptCount++
			return model.ContactVerification{}, fmt.Errorf("verification code is invalid or expired")
		}
		item.ConsumedAt = &now
		return *item, nil
	}

	row, err := r.db.ContactVerification.Query().Where(
		contactverification.KindEQ(contactverification.Kind(kind)),
		contactverification.DestinationEQ(destination),
		contactverification.ConsumedAtIsNil(),
		contactverification.ExpiresAtGT(now),
	).Order(contactverification.ByCreatedAt(sql.OrderDesc())).First(ctx)
	if err != nil || row.AttemptCount >= maxAttempts {
		return model.ContactVerification{}, fmt.Errorf("verification code is invalid or expired")
	}
	if row.TokenHash != tokenHash {
		_, _ = row.Update().AddAttemptCount(1).Save(ctx)
		return model.ContactVerification{}, fmt.Errorf("verification code is invalid or expired")
	}
	updated, err := r.db.ContactVerification.Update().Where(
		contactverification.IDEQ(row.ID),
		contactverification.ConsumedAtIsNil(),
	).SetConsumedAt(now).Save(ctx)
	if err != nil {
		return model.ContactVerification{}, err
	}
	if updated != 1 {
		return model.ContactVerification{}, fmt.Errorf("verification code is invalid or expired")
	}
	r.RefreshStore(ctx)
	return model.ContactVerification{
		ID: row.ID, UserID: row.UserID, Kind: row.Kind.String(), Destination: row.Destination,
		TokenHash: row.TokenHash, AttemptCount: row.AttemptCount, ExpiresAt: row.ExpiresAt,
		ConsumedAt: &now, CreatedAt: row.CreatedAt,
	}, nil
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
