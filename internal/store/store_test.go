package store

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSeeded_Populated(t *testing.T) {
	s := NewSeeded()
	require.NotNil(t, s)
	assert.NotEmpty(t, s.Users)
	assert.NotEmpty(t, s.Devices)
	assert.NotEmpty(t, s.Partners)
	assert.NotEmpty(t, s.Approvals)
	assert.NotEmpty(t, s.Modules)
	assert.NotEmpty(t, s.SupportCases)
	assert.NotEmpty(t, s.Organizations)
	assert.NotEmpty(t, s.Missions)
}

func TestNew_IsEmpty(t *testing.T) {
	s := New()
	require.NotNil(t, s)
	assert.Empty(t, s.Users)
	assert.Empty(t, s.AggregateEvents)
}

func TestUserByEmail(t *testing.T) {
	s := NewSeeded()
	u, ok := s.UserByEmail("gading@gmail.com")
	require.True(t, ok)
	assert.Equal(t, "usr_gading", u.ID)

	// case-insensitive
	_, ok = s.UserByEmail("GADING@gmail.com")
	assert.True(t, ok)

	_, ok = s.UserByEmail("nobody@example.com")
	assert.False(t, ok)
}

func TestDefaultUser(t *testing.T) {
	s := NewSeeded()
	u := s.DefaultUser()
	assert.NotEmpty(t, u.ID)
	assert.NotEmpty(t, u.Email)
}

func TestSnapshot_IsIndependentCopy(t *testing.T) {
	s := NewSeeded()
	snap := s.Snapshot()
	// Mutate the snapshot; the original must be unaffected.
	snap.Users[0].Email = "mutated@example.com"
	orig, _ := s.UserByEmail(s.Users[0].Email)
	assert.NotEqual(t, "mutated@example.com", orig.Email, "snapshot must be a copy")
}

func TestLockUnlock_NoDeadlock(t *testing.T) {
	s := NewSeeded()
	s.Lock()
	s.Unlock()
	s.RLock()
	s.RUnlock()
}

func TestTokenMapping_SetGet(t *testing.T) {
	s := NewSeeded()
	s.SetTokenMapping("hash1", ApprovalRequest{ID: "APR-X", Status: "pending"})
	got, ok := s.GetTokenMapping("hash1")
	require.True(t, ok)
	assert.Equal(t, "APR-X", got.ID)

	_, ok = s.GetTokenMapping("unknown")
	assert.False(t, ok)
}

func TestRefreshToken_SaveGetRevoke(t *testing.T) {
	s := NewSeeded()
	rec := RefreshTokenRecord{
		ID: "rt_1", UserID: "usr_gading", TokenHash: "th1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.SaveRefreshToken(rec)
	got, ok := s.GetRefreshToken("th1")
	require.True(t, ok)
	assert.Equal(t, "rt_1", got.ID)

	// Revoke by ID.
	ok = s.RevokeRefreshTokenByID("rt_1")
	assert.True(t, ok)
	// Re-fetch: revoked now set.
	got, _ = s.GetRefreshToken("th1")
	assert.NotNil(t, got.RevokedAt)

	// Revoke by hash (idempotent-ish).
	ok = s.RevokeRefreshToken("th1")
	assert.True(t, ok)
}

func TestRevokeRefreshTokenByID_Unknown(t *testing.T) {
	s := NewSeeded()
	ok := s.RevokeRefreshTokenByID("does-not-exist")
	assert.False(t, ok)
}

func TestRevokeRefreshToken_Unknown(t *testing.T) {
	s := NewSeeded()
	ok := s.RevokeRefreshToken("does-not-exist")
	assert.False(t, ok)
}

func TestGetRefreshToken_Unknown(t *testing.T) {
	s := NewSeeded()
	_, ok := s.GetRefreshToken("nope")
	assert.False(t, ok)
}

func TestConsumeRefreshTokenByID_OnlyActiveUnexpiredToken(t *testing.T) {
	s := New()
	now := time.Now().UTC()
	activeHash := "consume-active-" + t.Name()
	expiredHash := "consume-expired-" + t.Name()
	revokedHash := "consume-revoked-" + t.Name()
	defer deleteRefreshTokenRecords(activeHash, expiredHash, revokedHash)

	s.SaveRefreshToken(RefreshTokenRecord{ID: "consume-active", UserID: "user-1", TokenHash: activeHash, ExpiresAt: now.Add(time.Hour)})
	s.SaveRefreshToken(RefreshTokenRecord{ID: "consume-expired", UserID: "user-1", TokenHash: expiredHash, ExpiresAt: now.Add(-time.Second)})
	revokedAt := now.Add(-time.Minute)
	s.SaveRefreshToken(RefreshTokenRecord{ID: "consume-revoked", UserID: "user-1", TokenHash: revokedHash, ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt})

	assert.True(t, s.ConsumeRefreshTokenByID("consume-active"))
	assert.False(t, s.ConsumeRefreshTokenByID("consume-active"), "a consumed token cannot be consumed twice")
	assert.False(t, s.ConsumeRefreshTokenByID("consume-expired"))
	assert.False(t, s.ConsumeRefreshTokenByID("consume-revoked"))
	assert.False(t, s.ConsumeRefreshTokenByID("unknown"))

	got, ok := s.GetRefreshToken(activeHash)
	require.True(t, ok)
	assert.NotNil(t, got.RevokedAt)
}

func TestConsumeRefreshTokenByID_IsAtomic(t *testing.T) {
	s := New()
	hash := "consume-race-" + t.Name()
	defer deleteRefreshTokenRecords(hash)
	s.SaveRefreshToken(RefreshTokenRecord{
		ID: "consume-race", UserID: "user-1", TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})

	const attempts = 32
	results := make(chan bool, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- s.ConsumeRefreshTokenByID("consume-race")
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for consumed := range results {
		if consumed {
			successes++
		}
	}
	assert.Equal(t, 1, successes, "refresh-token rotation must allow exactly one consumer")
}

func TestRevokeRefreshTokensForUser_OnlyActiveMatchingTokens(t *testing.T) {
	s := New()
	now := time.Now().UTC()
	hashes := []string{
		"revoke-user-active-1-" + t.Name(),
		"revoke-user-active-2-" + t.Name(),
		"revoke-user-revoked-" + t.Name(),
		"revoke-other-user-" + t.Name(),
	}
	defer deleteRefreshTokenRecords(hashes...)

	revokedAt := now.Add(-time.Minute)
	s.SaveRefreshToken(RefreshTokenRecord{ID: "rt-1", UserID: "user-1", TokenHash: hashes[0], ExpiresAt: now.Add(time.Hour)})
	s.SaveRefreshToken(RefreshTokenRecord{ID: "rt-2", UserID: "user-1", TokenHash: hashes[1], ExpiresAt: now.Add(time.Hour)})
	s.SaveRefreshToken(RefreshTokenRecord{ID: "rt-3", UserID: "user-1", TokenHash: hashes[2], ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt})
	s.SaveRefreshToken(RefreshTokenRecord{ID: "rt-4", UserID: "user-2", TokenHash: hashes[3], ExpiresAt: now.Add(time.Hour)})

	assert.Equal(t, 2, s.RevokeRefreshTokensForUser("user-1"))
	assert.Equal(t, 0, s.RevokeRefreshTokensForUser("user-1"), "already revoked tokens are not counted again")

	for _, hash := range hashes[:3] {
		got, ok := s.GetRefreshToken(hash)
		require.True(t, ok)
		assert.NotNil(t, got.RevokedAt)
	}
	other, ok := s.GetRefreshToken(hashes[3])
	require.True(t, ok)
	assert.Nil(t, other.RevokedAt)
}

func deleteRefreshTokenRecords(hashes ...string) {
	refreshTokenMu.Lock()
	defer refreshTokenMu.Unlock()
	for _, hash := range hashes {
		delete(refreshTokenMap, hash)
	}
}
