package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func openTestSQLite(t *testing.T, dbName string) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)", dbName)
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	drv := entsql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestGetLearningHubTaxonomy_AutoEnsuresUTYInstitution(t *testing.T) {
	client := openTestSQLite(t, "taxonomy_auto_ensure_test")
	defer client.Close()
	ctx := context.Background()

	// Initially, database has 0 institutions, clusters, and programs.
	repo := New(client, store.New())

	taxonomy, err := repo.GetLearningHubTaxonomy(ctx)
	require.NoError(t, err)
	assert.Equal(t, "uty", taxonomy.Institution.Slug)
	assert.Equal(t, "Universitas Teknologi Yogyakarta", taxonomy.Institution.Name)
	assert.Equal(t, "inst_uty", taxonomy.Institution.ID)
	assert.Empty(t, taxonomy.Clusters)
	assert.Empty(t, taxonomy.Programs)
}

func TestCreateLearningProgram_AutoEnsuresUTYInstitution(t *testing.T) {
	client := openTestSQLite(t, "program_auto_ensure_test")
	defer client.Close()
	ctx := context.Background()

	repo := New(client, store.New())

	// Create a cluster first
	active := true
	cluster, err := repo.CreateLearningCluster(ctx, model.LearningClusterInput{
		Slug:          "it-software",
		TitleID:       "Teknologi Informasi & Perangkat Lunak",
		TitleEN:       "Information Technology & Software",
		DescriptionID: "Pengembangan perangkat lunak.",
		DescriptionEN: "Software development.",
		SortOrder:     1,
		Active:        &active,
	})
	require.NoError(t, err)
	assert.Equal(t, "it-software", cluster.Slug)

	// Create program without pre-seeded institution
	program, err := repo.CreateLearningProgram(ctx, model.AcademicProgramInput{
		Slug:               "s1-informatika",
		Name:               "Informatika",
		Degree:             "S1",
		PrimaryClusterSlug: "it-software",
		SortOrder:          1,
		Active:             &active,
	})
	require.NoError(t, err)
	assert.Equal(t, "s1-informatika", program.Slug)
	assert.Equal(t, "inst_uty", program.InstitutionID)

	// Fetch taxonomy and verify
	taxonomy, err := repo.GetLearningHubTaxonomy(ctx)
	require.NoError(t, err)
	assert.Len(t, taxonomy.Clusters, 1)
	assert.Len(t, taxonomy.Programs, 1)
	assert.Equal(t, "s1-informatika", taxonomy.Programs[0].Slug)
}
