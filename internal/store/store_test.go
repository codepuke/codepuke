package store_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/codepuke/codepuke/internal/store"
)

var (
	databaseURL string
	testStore   *store.Store
	testPool    *pgxpool.Pool // raw SQL access for fixtures and assertions
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx, "postgres:18-alpine",
		tcpostgres.WithDatabase("codepuke"),
		tcpostgres.WithUsername("codepuke"),
		tcpostgres.WithPassword("codepuke"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("start postgres container: %v", err)
		return 1
	}
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}()

	databaseURL, err = ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("connection string: %v", err)
		return 1
	}

	if err := store.RunMigrations(ctx, databaseURL); err != nil {
		log.Printf("migrate: %v", err)
		return 1
	}

	testStore, err = store.New(ctx, databaseURL)
	if err != nil {
		log.Printf("connect store: %v", err)
		return 1
	}
	defer testStore.Close()

	testPool, err = pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Printf("connect fixture pool: %v", err)
		return 1
	}
	defer testPool.Close()

	return m.Run()
}

// TestMain already ran the migrations once; a second Up must be a no-op
// because the server migrates on every boot.
func TestMigrationsIdempotent(t *testing.T) {
	t.Parallel()

	require.NoError(t, store.RunMigrations(t.Context(), databaseURL))
}

func TestSeed(t *testing.T) {
	t.Parallel()

	families, err := testStore.ListFamilies(t.Context())
	require.NoError(t, err)
	require.Len(t, families, 1)

	gob := families[0]
	assert.Equal(t, "gob", gob.Name)
	assert.NotEmpty(t, gob.Descriptor)

	projects, err := testStore.ListProjectsByFamily(t.Context(), gob.ID)
	require.NoError(t, err)

	var slugs []string
	for _, p := range projects {
		slugs = append(slugs, p.Slug)
		assert.NotEmpty(t, p.Name, "project %s has no name", p.Slug)
		assert.NotEmpty(t, p.Description, "project %s has no description", p.Slug)
	}
	assert.Equal(t,
		[]string{"gobdotnet", "gobspect", "gobspect-mcp", "gobts", "gq", "pygob"},
		slugs, "seeded projects, ordered by name")
}

func insertDocument(t *testing.T, kind, slug string, version *string) (int64, error) {
	t.Helper()

	var id int64
	err := testPool.QueryRow(t.Context(), `
		insert into documents (kind, slug, title, author, body_md, body_html, render_version, version)
		values ($1, $2, 'title', 'dan wolf', '# md', '<h1>md</h1>', 1, $3)
		returning id`,
		kind, slug, version).Scan(&id)
	if err == nil {
		t.Cleanup(func() {
			_, _ = testPool.Exec(context.Background(), `delete from documents where id = $1`, id)
		})
	}
	return id, err
}

func TestDocumentConstraints(t *testing.T) {
	t.Parallel()

	t.Run("valid article insert defaults latest true", func(t *testing.T) {
		t.Parallel()

		id, err := insertDocument(t, "article", "doc-valid", nil)
		require.NoError(t, err)

		var latest bool
		var publishedAt *string
		err = testPool.QueryRow(t.Context(),
			`select latest, published_at::text from documents where id = $1`, id).
			Scan(&latest, &publishedAt)
		require.NoError(t, err)
		assert.True(t, latest)
		assert.Nil(t, publishedAt, "unpublished insert stays a draft")
	})

	t.Run("invalid kind rejected", func(t *testing.T) {
		t.Parallel()

		_, err := insertDocument(t, "podcast", "doc-badkind", nil)
		require.ErrorContains(t, err, "documents_kind_check")
	})

	t.Run("duplicate slug with null version rejected", func(t *testing.T) {
		t.Parallel()

		_, err := insertDocument(t, "article", "doc-dup", nil)
		require.NoError(t, err)

		_, err = insertDocument(t, "article", "doc-dup", nil)
		require.ErrorContains(t, err, "duplicate key")
	})

	t.Run("same slug with distinct versions accepted", func(t *testing.T) {
		t.Parallel()

		_, err := insertDocument(t, "doc", "doc-versioned", new("v1.0"))
		require.NoError(t, err)

		_, err = insertDocument(t, "doc", "doc-versioned", new("v1.1"))
		require.NoError(t, err)
	})

	t.Run("same kind and slug across kinds accepted", func(t *testing.T) {
		t.Parallel()

		_, err := insertDocument(t, "article", "doc-crosskind", nil)
		require.NoError(t, err)

		_, err = insertDocument(t, "doc", "doc-crosskind", nil)
		require.NoError(t, err)
	})
}

func TestCategoryCascade(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	docID, err := insertDocument(t, "article", "doc-cascade", nil)
	require.NoError(t, err)

	var catID int64
	require.NoError(t, testPool.QueryRow(ctx, `
		insert into categories (slug, name)
		values ('cascade-cat', 'cascade cat')
		returning id`).Scan(&catID))
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `delete from categories where id = $1`, catID)
	})

	_, err = testPool.Exec(ctx, `
		insert into document_categories (document_id, category_id)
		values ($1, $2)`, docID, catID)
	require.NoError(t, err)

	_, err = testPool.Exec(ctx, `delete from documents where id = $1`, docID)
	require.NoError(t, err)

	var joinRows, catRows int
	require.NoError(t, testPool.QueryRow(ctx,
		`select count(*) from document_categories where document_id = $1`, docID).Scan(&joinRows))
	require.NoError(t, testPool.QueryRow(ctx,
		`select count(*) from categories where id = $1`, catID).Scan(&catRows))

	assert.Zero(t, joinRows, "join rows cascade with the document")
	assert.Equal(t, 1, catRows, "the category itself survives")
}
