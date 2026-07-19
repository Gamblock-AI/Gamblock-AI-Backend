package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	n, err := rand.Read(b)
	require.NoError(t, err)
	require.Equal(t, 32, n)
	return hex.EncodeToString(b)
}

// When an encryption key is configured, stored text must be encrypted (≠ plaintext)
// and retrieval must return the original plaintext (PRD §4 / §7.1).
func TestReflectionService_EncryptionRoundTrip(t *testing.T) {
	cfg := config.Config{JournalEncryptionKey: newKey(t)}
	st := store.New()
	repo := repository.New(nil, st)
	svc := NewReflectionService(repo, cfg, zap.NewNop())

	plain := "saya hampir tergoda hari ini"
	created, err := svc.CreateReflection(context.Background(), "usr_gading", plain, "cemas")
	require.NoError(t, err)
	stored := st.Snapshot().JournalEntries
	require.Len(t, stored, 1)
	assert.NotEqual(t, plain, stored[0].Text, "stored text must be encrypted")
	assert.NotEmpty(t, stored[0].Text)

	got, err := svc.GetReflections(context.Background(), "usr_gading")
	require.NoError(t, err)
	var found bool
	for _, e := range got {
		if e.ID == created.ID {
			assert.Equal(t, plain, e.Text, "retrieved text must be decrypted plaintext")
			found = true
		}
	}
	assert.True(t, found, "created entry must be retrievable")
}

// Without a key, the service fails closed rather than storing plaintext.
func TestReflectionService_NoKeyFailsClosed(t *testing.T) {
	cfg := config.Config{}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	svc := NewReflectionService(repo, cfg, zap.NewNop())

	plain := "refleksi tanpa enkripsi"
	_, err := svc.CreateReflection(context.Background(), "usr_gading", plain, "baik")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encryption is required")
}
