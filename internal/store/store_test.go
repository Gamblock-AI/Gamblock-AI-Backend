package store

import (
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
	assert.NotEmpty(t, s.ModelReleases)
	assert.NotEmpty(t, s.Missions)
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
