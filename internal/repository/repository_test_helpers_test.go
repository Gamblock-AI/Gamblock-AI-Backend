package repository

import (
	"testing"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newRepo(t *testing.T) (*Repository, *store.Store) {
	t.Helper()
	st := store.NewSeeded()
	return New(nil, st), st
}
