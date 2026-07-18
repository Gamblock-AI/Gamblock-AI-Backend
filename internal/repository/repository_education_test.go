package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func TestEducation_CreateAppendsToStore(t *testing.T) {
	repo, st := newRepo(t)
	before := len(st.Modules)
	err := repo.CreateEducationModule(context.Background(), model.EducationModule{
		ID: "mod_x", Slug: "x", Title: "X", Status: "published",
	})
	require.NoError(t, err)
	assert.Equal(t, before+1, len(st.Modules))
}

func TestGetEducationModuleBySlug(t *testing.T) {
	repo, _ := newRepo(t)
	module, err := repo.GetEducationModuleBySlug(context.Background(), "memahami-siklus-dorongan")
	require.NoError(t, err)
	assert.Equal(t, "memahami-siklus-dorongan", module.Slug)
}

func TestGetEducationModuleBySlug_NotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.GetEducationModuleBySlug(context.Background(), "no-such-slug")
	require.Error(t, err)
}
