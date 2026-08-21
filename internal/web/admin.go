package web

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/codepuke/codepuke/internal/auth"
	"github.com/codepuke/codepuke/internal/content"
	"github.com/codepuke/codepuke/internal/store"
	"github.com/codepuke/codepuke/ui"
)

// requireAuthor wraps every admin handler: no session gets the sign-in page,
// a session without the author group gets the same page with the group note.
// The signed-in author's display name is passed to the handler.
func (h *handlers) requireAuthor(next func(http.ResponseWriter, *http.Request, auth.Session)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginURL := "/auth/login?return=" + url.QueryEscape(r.URL.RequestURI())
		session, ok := h.auth.ReadSession(r)
		if !ok {
			h.render(w, r, http.StatusUnauthorized, ui.AdminLoginPage(loginURL, false))
			return
		}
		if !session.InGroup(h.auth.Group()) {
			h.render(w, r, http.StatusForbidden, ui.AdminLoginPage(loginURL, true))
			return
		}
		next(w, r, session)
	})
}

func (h *handlers) authLogin(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Begin(w, r, r.URL.Query().Get("return")); err != nil {
		slog.Error("auth login", "err", err)
		http.Error(w, "identity provider unavailable", http.StatusBadGateway)
	}
}

func (h *handlers) authCallback(w http.ResponseWriter, r *http.Request) {
	returnTo, err := h.auth.Complete(w, r)
	if err != nil {
		slog.Error("auth callback", "err", err)
		http.Error(w, "sign-in failed; start again at /admin", http.StatusForbidden)
		return
	}
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (h *handlers) authLogout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *handlers) adminIndex(w http.ResponseWriter, r *http.Request, session auth.Session) {
	rows, err := h.store.ListArticlesAdmin(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, ui.AdminIndexPage(rows, session.Name))
}

// adminCreate inserts a draft named by the form title. The slug is derived
// once, here; a taken slug gets a numbered suffix rather than an error.
func (h *handlers) adminCreate(w http.ResponseWriter, r *http.Request, session auth.Session) {
	title := strings.TrimSpace(r.FormValue("title"))
	slug := content.Slugify(title)
	if slug == "" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	for attempt := range 5 {
		candidate := slug
		if attempt > 0 {
			candidate = fmt.Sprintf("%s-%d", slug, attempt+1)
		}
		id, err := h.store.CreateArticle(r.Context(), candidate, title, session.Name)
		if err == nil {
			http.Redirect(w, r, fmt.Sprintf("/admin/articles/%d", id), http.StatusSeeOther)
			return
		}
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); !ok || pgErr.Code != "23505" {
			h.fail(w, r, err)
			return
		}
	}
	h.fail(w, r, fmt.Errorf("create article: slug %q taken", slug))
}

func (h *handlers) adminEditor(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.notFound(w, r)
		return
	}
	article, err := h.store.GetArticleByID(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, ui.AdminEditorPage(editorModel(article, session.Name)))
}

// adminSave is the editor form's one endpoint; the pressed button selects
// the action. Preview is the no-JS round trip: render, do not persist, send
// the editor back with the fresh preview. Save and publish persist and
// redirect (PRG) so a reload never re-submits.
func (h *handlers) adminSave(w http.ResponseWriter, r *http.Request, session auth.Session) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.notFound(w, r)
		return
	}
	article, err := h.store.GetArticleByID(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	author := strings.TrimSpace(r.FormValue("author"))
	bodyMD := r.FormValue("body_md")
	tags := parseTags(r.FormValue("tags"))

	switch r.FormValue("action") {
	case "preview":
		html, err := h.renderer.Render(r.Context(), []byte(bodyMD))
		if err != nil {
			h.fail(w, r, err)
			return
		}
		m := editorModel(article, session.Name)
		m.Title = title
		m.Author = author
		m.Date = r.FormValue("date")
		m.Tags = r.FormValue("tags")
		m.BodyMD = bodyMD
		m.PreviewHTML = string(html)
		m.Unsaved = true
		h.render(w, r, http.StatusOK, ui.AdminEditorPage(m))
		return

	case "save", "publish":
		html, err := h.renderer.Render(r.Context(), []byte(bodyMD))
		if err != nil {
			h.fail(w, r, err)
			return
		}
		if err := h.store.UpdateArticle(r.Context(), id, title, author, bodyMD, string(html), content.RenderVersion, tags); err != nil {
			h.fail(w, r, err)
			return
		}
		if r.FormValue("action") == "publish" {
			at := publishTime(r.FormValue("date"))
			if err := h.store.SetPublished(r.Context(), id, &at); err != nil {
				h.fail(w, r, err)
				return
			}
		}

	case "unpublish":
		if err := h.store.SetPublished(r.Context(), id, nil); err != nil {
			h.fail(w, r, err)
			return
		}

	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/articles/%d", id), http.StatusSeeOther)
}

// adminPreview backs the admin-editor element's fetch round trip: markdown
// in, rendered article HTML out. Nothing is persisted.
func (h *handlers) adminPreview(w http.ResponseWriter, r *http.Request, _ auth.Session) {
	html, err := h.renderer.Render(r.Context(), []byte(r.FormValue("body_md")))
	if err != nil {
		slog.Error("admin preview", "err", err)
		http.Error(w, "render failed", http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(html)
}

func editorModel(a store.AdminArticle, user string) ui.AdminEditor {
	m := ui.AdminEditor{
		ID:          a.ID,
		Slug:        a.Slug,
		Title:       a.Title,
		Author:      a.Author,
		Tags:        strings.Join(a.TagSlugs, ", "),
		BodyMD:      a.BodyMD,
		PreviewHTML: a.BodyHTML,
		SavedAt:     a.UpdatedAt.UTC().Format("15:04:05"),
		User:        user,
	}
	if a.PublishedAt != nil {
		m.Published = true
		m.Date = a.PublishedAt.UTC().Format("2006-01-02")
	}
	return m
}

// parseTags turns the comma-separated tags field into deduplicated slugs.
func parseTags(s string) []string {
	var tags []string
	seen := map[string]bool{}
	for raw := range strings.SplitSeq(s, ",") {
		slug := content.Slugify(raw)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		tags = append(tags, slug)
	}
	return tags
}

// publishTime resolves the editor's publish-date field: empty or malformed
// means now, a date means noon UTC that day so list ordering is stable.
func publishTime(date string) time.Time {
	if d, err := time.Parse("2006-01-02", date); err == nil {
		return d.Add(12 * time.Hour)
	}
	return time.Now().UTC()
}
