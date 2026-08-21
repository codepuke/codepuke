package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codepuke/codepuke/internal/content"
)

// initRepo builds a throwaway git repository with the given files committed.
func initRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	git("init", "-q")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "test")
	writeRepoFiles(t, dir, files)
	git("add", "-A")
	git("commit", "-q", "-m", "init")
	return dir
}

func writeRepoFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, data := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(data), 0o644))
	}
}

func writeSources(t *testing.T, dir string, sources sourcesFile) string {
	t.Helper()
	data, err := json.Marshal(sources)
	require.NoError(t, err)
	path := filepath.Join(dir, "sources.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

const goFixture = `package demo

func Demo() {
	// snippet:start encode-struct
	enc := gob.NewEncoder(&buf)
	enc.Encode(Point{X: 3, Y: 4})
	// snippet:end
}
`

const pyFixture = `# module
# snippet:start encode-struct
enc = Encoder(buf)
enc.encode(Point(x=3, y=4))
# snippet:end
`

func TestSync(t *testing.T) {
	t.Parallel()

	goRepo := initRepo(t, map[string]string{
		"demo.go":             goFixture,
		"docs/00-overview.md": "---\ntitle: Overview\n---\nThe overview body.\n",
		"docs/01-usage.md":    "# Using The Library\n\nUsage body.\n",
		"README.md":           "# not docs\n",
	})
	pyRepo := initRepo(t, map[string]string{
		"demo.py": pyFixture,
	})

	base := t.TempDir()
	sourcesPath := writeSources(t, base, sourcesFile{Sources: []sourceSpec{
		{Name: "gorepo", Path: goRepo, Ref: "HEAD", Lang: "go",
			Projects: []projectSpec{{Slug: "gorepo", Docs: "docs"}}},
		{Name: "pyrepo", Path: pyRepo, Ref: "HEAD", Lang: "python"},
	}})

	out := filepath.Join(base, "content")
	require.NoError(t, run(sourcesPath, out))

	t.Run("snippets extracted and dedented", func(t *testing.T) {
		goCode, err := os.ReadFile(filepath.Join(out, "examples", "encode-struct", "go.txt"))
		require.NoError(t, err)
		assert.Equal(t, "enc := gob.NewEncoder(&buf)\nenc.Encode(Point{X: 3, Y: 4})\n", string(goCode))

		pyCode, err := os.ReadFile(filepath.Join(out, "examples", "encode-struct", "python.py"))
		require.NoError(t, err)
		assert.Equal(t, "enc = Encoder(buf)\nenc.encode(Point(x=3, y=4))\n", string(pyCode))
	})

	t.Run("docs collected with front matter stripped", func(t *testing.T) {
		overview, err := os.ReadFile(filepath.Join(out, "docs", "gorepo", "overview.md"))
		require.NoError(t, err)
		assert.Equal(t, "The overview body.\n", string(overview))

		_, err = os.Stat(filepath.Join(out, "docs", "gorepo", "usage.md"))
		require.NoError(t, err)
	})

	t.Run("manifest records order, titles, and commits", func(t *testing.T) {
		m, err := content.LoadManifest(os.DirFS(out))
		require.NoError(t, err)
		require.Len(t, m.Sources, 2)

		gorepo := m.Sources[0]
		assert.Equal(t, "gorepo", gorepo.Name)
		assert.Equal(t, "HEAD", gorepo.Ref)
		assert.Len(t, gorepo.Commit, 40)
		assert.Equal(t, []string{"encode-struct"}, gorepo.Topics)
		require.Len(t, gorepo.Projects, 1)
		docs := gorepo.Projects[0].Docs
		require.Len(t, docs, 2)
		assert.Equal(t, "overview", docs[0].Slug)
		assert.Equal(t, "Overview", docs[0].Title, "front matter title wins")
		assert.Equal(t, "usage", docs[1].Slug)
		assert.Equal(t, "Using The Library", docs[1].Title, "first heading fallback")
		assert.Equal(t, "docs/00-overview.md", docs[0].Source)
	})

	t.Run("output feeds the render pipeline", func(t *testing.T) {
		p := content.New(content.Options{Examples: content.NewFSSource(os.DirFS(out))})
		html, err := p.Render(t.Context(), []byte(":::examples encode-struct\n"))
		require.NoError(t, err)
		assert.Contains(t, string(html), `<code-tabs data-topic="encode-struct">`)
		assert.Contains(t, string(html), `data-lang="go"`)
		assert.Contains(t, string(html), `data-lang="python"`)
		assert.NotContains(t, string(html), `data-lang="typescript"`)
	})

	t.Run("rerun is deterministic", func(t *testing.T) {
		out2 := filepath.Join(base, "content2")
		require.NoError(t, run(sourcesPath, out2))
		m1, err := os.ReadFile(filepath.Join(out, "manifest.json"))
		require.NoError(t, err)
		m2, err := os.ReadFile(filepath.Join(out2, "manifest.json"))
		require.NoError(t, err)
		assert.Equal(t, string(m1), string(m2))
	})
}

func TestSyncRefPinning(t *testing.T) {
	t.Parallel()

	repo := initRepo(t, map[string]string{"demo.go": goFixture})
	git := func(args ...string) {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	git("-c", "tag.gpgSign=false", "tag", "-m", "v1", "v1")

	writeRepoFiles(t, repo, map[string]string{"demo.go": "package demo\n\n// snippet:start other\n// changed := true\n// snippet:end\n"})
	git("add", "-A")
	git("commit", "-q", "-m", "change")

	base := t.TempDir()
	sourcesPath := writeSources(t, base, sourcesFile{Sources: []sourceSpec{
		{Name: "repo", Path: repo, Ref: "v1", Lang: "go"},
	}})
	out := filepath.Join(base, "content")
	require.NoError(t, run(sourcesPath, out))

	_, err := os.Stat(filepath.Join(out, "examples", "encode-struct", "go.txt"))
	assert.NoError(t, err, "content at the pinned tag is synced")
	_, err = os.Stat(filepath.Join(out, "examples", "other"))
	assert.True(t, os.IsNotExist(err), "content newer than the pinned tag is not")
}

func TestSyncErrors(t *testing.T) {
	t.Parallel()

	t.Run("duplicate topic and lang across sources", func(t *testing.T) {
		t.Parallel()
		a := initRepo(t, map[string]string{"a.go": goFixture})
		b := initRepo(t, map[string]string{"b.go": goFixture})
		base := t.TempDir()
		sourcesPath := writeSources(t, base, sourcesFile{Sources: []sourceSpec{
			{Name: "a", Path: a, Ref: "HEAD", Lang: "go"},
			{Name: "b", Path: b, Ref: "HEAD", Lang: "go"},
		}})
		err := run(sourcesPath, filepath.Join(base, "content"))
		require.ErrorContains(t, err, `topic "encode-struct" already provided in a`)
	})

	t.Run("unknown lang", func(t *testing.T) {
		t.Parallel()
		repo := initRepo(t, map[string]string{"a.rb": "puts 1\n"})
		base := t.TempDir()
		sourcesPath := writeSources(t, base, sourcesFile{Sources: []sourceSpec{
			{Name: "a", Path: repo, Ref: "HEAD", Lang: "ruby"},
		}})
		err := run(sourcesPath, filepath.Join(base, "content"))
		require.ErrorContains(t, err, `unknown lang "ruby"`)
	})

	t.Run("bad ref", func(t *testing.T) {
		t.Parallel()
		repo := initRepo(t, map[string]string{"a.go": "package a\n"})
		base := t.TempDir()
		sourcesPath := writeSources(t, base, sourcesFile{Sources: []sourceSpec{
			{Name: "a", Path: repo, Ref: "does-not-exist", Lang: "go"},
		}})
		err := run(sourcesPath, filepath.Join(base, "content"))
		require.ErrorContains(t, err, "resolve")
	})

	t.Run("marker error names the file", func(t *testing.T) {
		t.Parallel()
		repo := initRepo(t, map[string]string{"bad.go": "// snippet:start t\nnever closed\n"})
		base := t.TempDir()
		sourcesPath := writeSources(t, base, sourcesFile{Sources: []sourceSpec{
			{Name: "a", Path: repo, Ref: "HEAD", Lang: "go"},
		}})
		err := run(sourcesPath, filepath.Join(base, "content"))
		require.ErrorContains(t, err, "bad.go")
		require.ErrorContains(t, err, "never closed")
	})
}
