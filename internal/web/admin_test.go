package web_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codepuke/codepuke/internal/auth"
	"github.com/codepuke/codepuke/internal/content"
	"github.com/codepuke/codepuke/internal/web"
)

// adminHarness builds a handler with the admin surface enabled and a codec
// that can forge session cookies for it. OIDC discovery is lazy, so the fake
// issuer is never contacted by /admin page loads.
type adminHarness struct {
	site  http.Handler
	codec *auth.Codec
}

func newAdminHarness(t *testing.T) adminHarness {
	t.Helper()

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i * 7)
	}
	a, err := auth.New(auth.Options{
		Issuer:       "https://idp.invalid/application/o/codepuke/",
		ClientID:     "codepuke",
		ClientSecret: "secret",
		BaseURL:      "https://codepuke.example",
		Group:        "codepuke-authors",
		SessionKey:   key,
	})
	require.NoError(t, err)

	handler, err := web.New(web.Deps{
		Store:    testStore,
		Content:  fstest.MapFS{"manifest.json": {Data: []byte(testManifest)}},
		BaseURL:  "https://codepuke.example",
		Auth:     a,
		Renderer: content.New(content.Options{}),
	})
	require.NoError(t, err)

	codec, err := auth.NewCodec(key)
	require.NoError(t, err)
	return adminHarness{site: handler, codec: codec}
}

func (h adminHarness) cookie(t *testing.T, groups ...string) *http.Cookie {
	t.Helper()
	token, err := h.codec.Seal(auth.Session{
		Subject: "sub-1",
		Name:    "dan.wolf",
		Groups:  groups,
		Expires: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	return &http.Cookie{Name: "codepuke_session", Value: token}
}

func (h adminHarness) do(t *testing.T, method, path string, form url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.site.ServeHTTP(rec, req)
	return rec
}

func TestAdminDisabled(t *testing.T) {
	t.Parallel()

	// The package-level handler has no Auth, so the admin surface simply
	// does not exist.
	code, _ := get(t, "/admin")
	assert.Equal(t, http.StatusNotFound, code)
	code, _ = get(t, "/auth/login")
	assert.Equal(t, http.StatusNotFound, code)
}

func TestAdminGate(t *testing.T) {
	t.Parallel()
	h := newAdminHarness(t)

	t.Run("no session gets the sign-in page", func(t *testing.T) {
		t.Parallel()
		rec := h.do(t, http.MethodGet, "/admin", nil)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "AUTHORS ONLY")
		assert.Contains(t, rec.Body.String(), "continue to sign in")
		assert.Contains(t, rec.Body.String(), `/auth/login?return=%2Fadmin`)
	})

	t.Run("session without the group is refused with the group note", func(t *testing.T) {
		t.Parallel()
		rec := h.do(t, http.MethodGet, "/admin", nil, h.cookie(t, "some-other-group"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "missing the authors group")
	})

	t.Run("expired session is no session", func(t *testing.T) {
		t.Parallel()
		token, err := h.codec.Seal(auth.Session{
			Subject: "sub-1", Name: "dan.wolf",
			Groups:  []string{"codepuke-authors"},
			Expires: time.Now().Add(-time.Minute),
		})
		require.NoError(t, err)
		rec := h.do(t, http.MethodGet, "/admin", nil, &http.Cookie{Name: "codepuke_session", Value: token})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("author session passes", func(t *testing.T) {
		t.Parallel()
		rec := h.do(t, http.MethodGet, "/admin", nil, h.cookie(t, "codepuke-authors"))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "create draft")
		assert.Contains(t, rec.Body.String(), "A Draft", "drafts are listed")
		assert.Contains(t, rec.Body.String(), "dan.wolf // authors")
	})

	t.Run("logout clears the cookie and returns home", func(t *testing.T) {
		t.Parallel()
		rec := h.do(t, http.MethodGet, "/auth/logout", nil, h.cookie(t, "codepuke-authors"))
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/", rec.Header().Get("Location"))
		found := false
		for _, c := range rec.Result().Cookies() {
			if c.Name == "codepuke_session" {
				found = true
				assert.Less(t, c.MaxAge, 0, "cookie is dropped")
			}
		}
		assert.True(t, found)
	})
}

func TestAdminEditorFlow(t *testing.T) {
	t.Parallel()
	h := newAdminHarness(t)
	author := h.cookie(t, "codepuke-authors")

	// Create a draft from a title; the slug derives once, at creation.
	rec := h.do(t, http.MethodPost, "/admin/articles", url.Values{"title": {"Editor Flow Test!"}}, author)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	editorPath := rec.Header().Get("Location")
	require.Regexp(t, regexp.MustCompile(`^/admin/articles/\d+$`), editorPath)

	rec = h.do(t, http.MethodGet, editorPath, nil, author)
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "editor-flow-test", "slug derived from the title")
	assert.Contains(t, body, `value="Editor Flow Test!"`)
	assert.Contains(t, body, "saved ", "fresh page is in the saved state")
	assert.Contains(t, body, "publish", "drafts offer publish")

	md := "## The First Section\n\nHello **world**."

	t.Run("preview round trip renders without persisting", func(t *testing.T) {
		rec := h.do(t, http.MethodPost, editorPath, url.Values{
			"action": {"preview"}, "title": {"Editor Flow Test!"},
			"author": {"dan wolf"}, "tags": {""}, "body_md": {md},
		}, author)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "unsaved changes")
		assert.Contains(t, rec.Body.String(), "The First Section")
		assert.Contains(t, rec.Body.String(), `class="offset-anchor"`, "full pipeline renders the preview")

		reloaded := h.do(t, http.MethodGet, editorPath, nil, author)
		assert.NotContains(t, reloaded.Body.String(), "The First Section", "preview did not save")
	})

	t.Run("fetch preview endpoint returns the fragment", func(t *testing.T) {
		rec := h.do(t, http.MethodPost, "/admin/preview", url.Values{"body_md": {md}}, author)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "The First Section")
		assert.NotContains(t, rec.Body.String(), "<html", "fragment, not a page")
	})

	t.Run("save persists body and tags", func(t *testing.T) {
		rec := h.do(t, http.MethodPost, editorPath, url.Values{
			"action": {"save"}, "title": {"Editor Flow Test!"},
			"author": {"dan wolf"}, "tags": {"flow, Flow, testing tags"}, "body_md": {md},
		}, author)
		require.Equal(t, http.StatusSeeOther, rec.Code)

		reloaded := h.do(t, http.MethodGet, editorPath, nil, author)
		body := reloaded.Body.String()
		assert.Contains(t, body, "The First Section", "stored render feeds the preview")
		assert.Contains(t, body, "flow, testing-tags", "tags slugified and deduplicated")

		code, _ := get(t, "/articles/editor-flow-test")
		assert.Equal(t, http.StatusNotFound, code, "saved draft stays unpublished")
	})

	t.Run("publish makes it public, unpublish takes it back", func(t *testing.T) {
		rec := h.do(t, http.MethodPost, editorPath, url.Values{
			"action": {"publish"}, "title": {"Editor Flow Test!"},
			"author": {"dan wolf"}, "date": {"2026-08-20"}, "tags": {"flow"}, "body_md": {md},
		}, author)
		require.Equal(t, http.StatusSeeOther, rec.Code)

		code, public := get(t, "/articles/editor-flow-test")
		require.Equal(t, http.StatusOK, code)
		assert.Contains(t, public, "article // 2026-08-20 // dan wolf")
		assert.Contains(t, public, "Hello <strong>world</strong>")

		reloaded := h.do(t, http.MethodGet, editorPath, nil, author)
		assert.Contains(t, reloaded.Body.String(), "unpublish", "published articles offer unpublish")

		rec = h.do(t, http.MethodPost, editorPath, url.Values{"action": {"unpublish"},
			"title": {"Editor Flow Test!"}, "author": {"dan wolf"}, "tags": {""}, "body_md": {md}}, author)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		code, _ = get(t, "/articles/editor-flow-test")
		assert.Equal(t, http.StatusNotFound, code)
	})

	t.Run("mutations demand the author group", func(t *testing.T) {
		rec := h.do(t, http.MethodPost, editorPath, url.Values{"action": {"save"}, "body_md": {"x"}},
			h.cookie(t, "other"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
		rec = h.do(t, http.MethodPost, "/admin/preview", url.Values{"body_md": {"x"}})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("duplicate title gets a suffixed slug", func(t *testing.T) {
		rec := h.do(t, http.MethodPost, "/admin/articles", url.Values{"title": {"Editor Flow Test!"}}, author)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		dup := h.do(t, http.MethodGet, rec.Header().Get("Location"), nil, author)
		assert.Contains(t, dup.Body.String(), "editor-flow-test-2")
	})
}

func TestAdminStatic(t *testing.T) {
	t.Parallel()

	code, body := get(t, "/static/admin.js")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `customElements.define("admin-editor"`)
}
