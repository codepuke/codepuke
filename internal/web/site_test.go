package web_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/codepuke/codepuke/internal/store"
	"github.com/codepuke/codepuke/internal/web"
)

var (
	testStore *store.Store
	testPool  *pgxpool.Pool
	site      http.Handler
)

const testManifest = `{
  "sources": [
    {
      "name": "gobspect", "ref": "main", "commit": "abc", "lang": "go",
      "topics": [],
      "projects": [
        {"slug": "gobspect", "docs": [
          {"slug": "overview", "title": "Overview", "file": "docs/gobspect/overview.md", "source": "docs/00-overview.md"},
          {"slug": "streams", "title": "Reading Streams", "file": "docs/gobspect/streams.md", "source": "docs/01-streams.md"}
        ]}
      ]
    }
  ]
}`

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

	databaseURL, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("connection string: %v", err)
		return 1
	}
	if err := store.RunMigrations(ctx, databaseURL); err != nil {
		log.Printf("migrate: %v", err)
		return 1
	}
	if testStore, err = store.New(ctx, databaseURL); err != nil {
		log.Printf("connect store: %v", err)
		return 1
	}
	defer testStore.Close()
	if testPool, err = pgxpool.New(ctx, databaseURL); err != nil {
		log.Printf("connect fixture pool: %v", err)
		return 1
	}
	defer testPool.Close()

	if err := seedFixtures(ctx); err != nil {
		log.Printf("seed fixtures: %v", err)
		return 1
	}

	site, err = web.New(web.Deps{
		Store:   testStore,
		Content: fstest.MapFS{"manifest.json": {Data: []byte(testManifest)}},
		BaseURL: "https://codepuke.example",
	})
	if err != nil {
		log.Printf("build handler: %v", err)
		return 1
	}

	return m.Run()
}

// seedFixtures inserts two published articles (2025 and 2026, one tagged),
// one draft that must stay invisible, and two docs for gobspect.
func seedFixtures(ctx context.Context) error {
	sql := `
	insert into categories (slug, name) values ('wire-format', 'wire format'), ('unused', 'unused');

	insert into documents (kind, slug, title, author, body_md, body_html, render_version, published_at)
	values
	  ('article', 'first-record', 'The First Record', 'dan wolf', '# md',
	   '<h2 id="alpha"><a class="offset-anchor" href="#alpha">0x01</a> Alpha</h2><p>body one</p>',
	   1, '2025-03-01T12:00:00Z'),
	  ('article', 'second-record', 'The Second Record', 'dan wolf', '# md',
	   '<p>body two</p><scroll-box><pre class="chroma"><span class="kd">func</span></pre></scroll-box>',
	   1, '2026-01-15T12:00:00Z'),
	  ('article', 'a-draft', 'A Draft', 'dan wolf', '# md', '<p>draft</p>', 1, null);

	insert into document_categories (document_id, category_id)
	select d.id, c.id from documents d, categories c
	where d.slug = 'second-record' and c.slug = 'wire-format';

	insert into documents (kind, project_id, slug, title, author, body_md, body_html, render_version, published_at)
	select 'doc', p.id, v.slug, v.title, 'dan wolf', '# md', v.body, 1, now()
	from projects p,
	     (values ('overview', 'Overview', '<p>the overview doc</p>'),
	             ('streams', 'Reading Streams',
	              '<h2 id="frames"><a class="offset-anchor" href="#frames">0x01</a> Reading Frames</h2><p>the streams doc</p>')) as v(slug, title, body)
	where p.slug = 'gobspect';
	`
	_, err := testPool.Exec(ctx, sql)
	return err
}

func get(t *testing.T, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	site.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Body.String()
}

func TestHome(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/")
	require.Equal(t, http.StatusOK, code)

	assert.Contains(t, body, `<h1 class="wordmark">codepuke</h1>`)
	assert.Contains(t, body, `class="record-row"`)
	assert.Contains(t, body, "The Second Record", "newest first")
	assert.Contains(t, body, `<span class="offset">0x0000</span>`, "offsets restart at 0x0000")
	assert.Contains(t, body, `<span class="offset">0x0040</span>`, "position times 0x40")
	assert.NotContains(t, body, "A Draft", "drafts are invisible")
	assert.Contains(t, body, `href="/tags/wire-format"`)
	assert.Contains(t, body, "gob", "family aside renders")
	assert.Contains(t, body, "2 records")
}

func TestArticlePage(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/articles/second-record")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `class="kicker article-kicker"`)
	assert.Contains(t, body, "article // 2026-01-15 // dan wolf")
	assert.Contains(t, body, "The Second Record")
	assert.Contains(t, body, `<scroll-box><pre class="chroma">`, "stored HTML passes through unescaped")
	assert.Contains(t, body, `<span class="kd">func</span>`)

	code, _ = get(t, "/articles/nope")
	assert.Equal(t, http.StatusNotFound, code)

	code, _ = get(t, "/articles/a-draft")
	assert.Equal(t, http.StatusNotFound, code, "drafts are not addressable")
}

func TestTagPage(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/tags/wire-format")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "wire format")
	assert.Contains(t, body, "The Second Record")
	assert.NotContains(t, body, "The First Record", "untagged article filtered out")

	code, _ = get(t, "/tags/nope")
	assert.Equal(t, http.StatusNotFound, code)
}

func TestArchivePage(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/archive")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `id="y2026"`)
	assert.Contains(t, body, `id="y2025"`)
	assert.Contains(t, body, `href="#y2026"`, "seek row targets year heads")
	assert.Contains(t, body, `class="archive-row"`)
	assert.Contains(t, body, "1 records")
}

func TestProjectsPage(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/projects")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `class="family-head"`)
	assert.Contains(t, body, "6 projects")
	assert.Contains(t, body, `class="card"`)
	assert.Contains(t, body, "gobdotnet")
	assert.Contains(t, body, `href="/docs/gobspect/overview"`, "documented project opens to docs")
	assert.Contains(t, body, `href="https://github.com/codepuke/gobts"`, "undocumented project opens to repo")
}

func TestDocsPages(t *testing.T) {
	t.Parallel()

	t.Run("doc page renders nav twice", func(t *testing.T) {
		t.Parallel()
		code, body := get(t, "/docs/gobspect/streams")
		require.Equal(t, http.StatusOK, code)
		assert.Contains(t, body, "the streams doc")
		assert.Equal(t, 2, countOccurrences(body, `class="docs-nav"`), "sidebar copy plus contents-bar copy")
		assert.Contains(t, body, `<details class="docs-toc">`)
		assert.Contains(t, body, `aria-current="page"`)
		assert.Contains(t, body, `<span>00</span>Overview`)
		assert.Contains(t, body, `<span>01</span>Reading Streams`)
	})

	t.Run("active item lists its h2 sections", func(t *testing.T) {
		t.Parallel()
		_, body := get(t, "/docs/gobspect/streams")
		assert.Equal(t, 2, countOccurrences(body, `class="nav-sub"`), "both nav copies carry the outline")
		assert.Contains(t, body, `data-title="Reading Frames"`)
		assert.Contains(t, body, `href="#frames"`)
		assert.Contains(t, body, `<span>0x01</span>`)

		_, overview := get(t, "/docs/gobspect/overview")
		assert.NotContains(t, overview, "nav-sub", "pages without h2s get no outline")
	})

	t.Run("project root redirects to first doc", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		site.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs/gobspect", nil))
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/docs/gobspect/overview", rec.Header().Get("Location"))
	})

	t.Run("docs index redirects to first documented project", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		site.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/docs", nil))
		assert.Equal(t, http.StatusFound, rec.Code)
	})

	t.Run("unknown project and doc 404", func(t *testing.T) {
		t.Parallel()
		code, _ := get(t, "/docs/nope/overview")
		assert.Equal(t, http.StatusNotFound, code)
		code, _ = get(t, "/docs/gobspect/nope")
		assert.Equal(t, http.StatusNotFound, code)
	})
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/definitely/not/a/page")
	assert.Equal(t, http.StatusNotFound, code)
	assert.Contains(t, body, "NOT FOUND")
	assert.Contains(t, body, "0x194")
}

func TestStatic(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/static/site.css")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, "--color-accent")

	code, body = get(t, "/static/site.js")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `customElements.define("code-tabs"`)
	assert.Contains(t, body, `customElements.define("scroll-box"`)
}

func TestRSS(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/rss.xml")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `<rss version="2.0">`)
	assert.Contains(t, body, "https://codepuke.example/articles/second-record")
	assert.NotContains(t, body, "a-draft")

	pub, err := time.Parse(time.RFC1123Z, "Thu, 15 Jan 2026 12:00:00 +0000")
	require.NoError(t, err)
	assert.Contains(t, body, pub.Format(time.RFC1123Z))
}

func countOccurrences(s, sub string) int {
	return strings.Count(s, sub)
}
