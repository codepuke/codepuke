package web

import (
	"cmp"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/a-h/templ"

	"github.com/codepuke/codepuke/internal/content"
	"github.com/codepuke/codepuke/internal/store"
	"github.com/codepuke/codepuke/ui"
)

func staticFS() fs.FS { return ui.StaticFS }

type handlers struct {
	store    *store.Store
	baseURL  string
	manifest *content.Manifest
	docs     map[string]content.ManifestProject // project slug -> docs
}

func (h *handlers) render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		slog.Error("render", "path", r.URL.Path, "err", err)
	}
}

func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		h.render(w, r, http.StatusNotFound, ui.NotFoundPage())
		return
	}
	slog.Error("handler", "path", r.URL.Path, "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func (h *handlers) notFound(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusNotFound, ui.NotFoundPage())
}

// familyGroups assembles the data-driven project chrome: families and their
// projects, both query-ordered, with each card's open link resolved to the
// project's first docs page or its repository.
func (h *handlers) familyGroups(r *http.Request) ([]ui.FamilyGroup, error) {
	families, err := h.store.ListFamilies(r.Context())
	if err != nil {
		return nil, err
	}
	var groups []ui.FamilyGroup
	for _, f := range families {
		projects, err := h.store.ListProjectsByFamily(r.Context(), f.ID)
		if err != nil {
			return nil, err
		}
		group := ui.FamilyGroup{Family: f}
		for _, p := range projects {
			open := "/projects"
			if docs, ok := h.docs[p.Slug]; ok {
				open = "/docs/" + p.Slug + "/" + docs.Docs[0].Slug
			} else if p.RepoURL != nil {
				open = *p.RepoURL
			}
			group.Cards = append(group.Cards, ui.ProjectCard{Project: p, OpenURL: open})
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (h *handlers) home(w http.ResponseWriter, r *http.Request) {
	articles, err := h.store.ListPublishedArticles(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	groups, err := h.familyGroups(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, ui.Home(articles, groups))
}

func (h *handlers) article(w http.ResponseWriter, r *http.Request) {
	article, err := h.store.GetArticle(r.Context(), r.PathValue("slug"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, ui.ArticlePage(article))
}

func (h *handlers) tag(w http.ResponseWriter, r *http.Request) {
	category, err := h.store.GetCategory(r.Context(), r.PathValue("slug"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	articles, err := h.store.ListArticlesByTag(r.Context(), category.Slug)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, ui.TagPage(category, articles))
}

func (h *handlers) archive(w http.ResponseWriter, r *http.Request) {
	articles, err := h.store.ListPublishedArticles(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var years []ui.ArchiveYear
	for _, a := range articles {
		year := a.PublishedAt.Year()
		if len(years) == 0 || years[len(years)-1].Year != year {
			years = append(years, ui.ArchiveYear{Year: year})
		}
		last := &years[len(years)-1]
		last.Articles = append(last.Articles, a)
	}
	h.render(w, r, http.StatusOK, ui.ArchivePage(years))
}

func (h *handlers) projects(w http.ResponseWriter, r *http.Request) {
	groups, err := h.familyGroups(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, http.StatusOK, ui.ProjectsPage(groups))
}

// docsIndex sends /docs to the first documented project; there is no
// designed docs index surface.
func (h *handlers) docsIndex(w http.ResponseWriter, r *http.Request) {
	slugs := slices.Sorted(maps.Keys(h.docs))
	if len(slugs) == 0 {
		h.notFound(w, r)
		return
	}
	h.docsProjectRedirect(w, r, slugs[0])
}

func (h *handlers) docsProject(w http.ResponseWriter, r *http.Request) {
	h.docsProjectRedirect(w, r, r.PathValue("project"))
}

func (h *handlers) docsProjectRedirect(w http.ResponseWriter, r *http.Request, project string) {
	docs, ok := h.docs[project]
	if !ok {
		h.notFound(w, r)
		return
	}
	http.Redirect(w, r, "/docs/"+project+"/"+docs.Docs[0].Slug, http.StatusFound)
}

func (h *handlers) docPage(w http.ResponseWriter, r *http.Request) {
	projectSlug := r.PathValue("project")
	docSlug := r.PathValue("slug")

	manifestDocs, ok := h.docs[projectSlug]
	if !ok {
		h.notFound(w, r)
		return
	}
	doc, err := h.store.GetDoc(r.Context(), projectSlug, docSlug)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	nav := ui.DocsNavModel{
		ProjectName: doc.ProjectName,
		Version:     cmp.Or(deref(doc.Version), ""),
	}
	for _, d := range manifestDocs.Docs {
		nav.Items = append(nav.Items, ui.DocsNavItem{
			Title:  d.Title,
			URL:    "/docs/" + projectSlug + "/" + d.Slug,
			Active: d.Slug == docSlug,
		})
	}
	h.render(w, r, http.StatusOK, ui.DocPage(doc, nav))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
