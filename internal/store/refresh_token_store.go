package store

import (
	"sync"
	"time"
)

// RefreshTokenRecord is the in-memory backing for refresh tokens when the DB
// is unavailable. It mirrors the ent fields needed by the auth flow.
type RefreshTokenRecord struct {
	ID        string
	UserID    string
	TokenHash string
	DeviceID  *string
	AuthTime  time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

var (
	refreshTokenMu  sync.RWMutex
	refreshTokenMap = make(map[string]RefreshTokenRecord)
)

func (s *Store) SaveRefreshToken(rec RefreshTokenRecord) {
	refreshTokenMu.Lock()
	defer refreshTokenMu.Unlock()
	refreshTokenMap[rec.TokenHash] = rec
}

func (s *Store) GetRefreshToken(tokenHash string) (RefreshTokenRecord, bool) {
	refreshTokenMu.RLock()
	defer refreshTokenMu.RUnlock()
	rec, ok := refreshTokenMap[tokenHash]
	return rec, ok
}

func (s *Store) RevokeRefreshToken(tokenHash string) bool {
	refreshTokenMu.Lock()
	defer refreshTokenMu.Unlock()
	rec, ok := refreshTokenMap[tokenHash]
	if !ok {
		return false
	}
	now := time.Now().UTC()
	rec.RevokedAt = &now
	refreshTokenMap[tokenHash] = rec
	return true
}

func (s *Store) RevokeRefreshTokenByID(id string) bool {
	refreshTokenMu.Lock()
	defer refreshTokenMu.Unlock()
	for hash, rec := range refreshTokenMap {
		if rec.ID != id {
			continue
		}
		now := time.Now().UTC()
		rec.RevokedAt = &now
		refreshTokenMap[hash] = rec
		return true
	}
	return false
}

func (s *Store) RevokeRefreshTokensForUser(userID string) int {
	refreshTokenMu.Lock()
	defer refreshTokenMu.Unlock()
	now := time.Now().UTC()
	count := 0
	for hash, rec := range refreshTokenMap {
		if rec.UserID == userID && rec.RevokedAt == nil {
			rec.RevokedAt = &now
			refreshTokenMap[hash] = rec
			count++
		}
	}
	return count
}
