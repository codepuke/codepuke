// Package web wires HTTP routing, handlers, and middleware. Handlers only
// read stored HTML; nothing here renders markdown.
package web

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"

	"github.com/codepuke/codepuke/internal/content"
	"github.com/codepuke/codepuke/internal/store"
)

// Deps is everything the handlers need.
type Deps struct {
	Store *store.Store
	// Content is the synced content tree; its manifest drives docs
	// navigation.
	Content fs.FS
	// BaseURL is the public origin for absolute links (RSS).
	BaseURL string
}

// Pinger reports whether the backing database is reachable.
type Pinger interface {
	Ping(context.Context) error
}

// Probes registers the probe split from deploy/PLAN.md: healthz has no
// dependencies, readyz touches the database.
func Probes(mux *http.ServeMux, db Pinger) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
		io.WriteString(w, "ok\n")
	})
}

// New builds the site handler.
func New(deps Deps) (http.Handler, error) {
	manifest, err := content.LoadManifest(deps.Content)
	if err != nil {
		return nil, fmt.Errorf("load content manifest: %w", err)
	}

	h := &handlers{
		store:    deps.Store,
		baseURL:  deps.BaseURL,
		manifest: manifest,
		docs:     map[string]content.ManifestProject{},
	}
	for _, src := range manifest.Sources {
		for _, proj := range src.Projects {
			if len(proj.Docs) > 0 {
				h.docs[proj.Slug] = proj
			}
		}
	}

	mux := http.NewServeMux()
	Probes(mux, deps.Store)

	mux.HandleFunc("/", h.notFound) // anything no other pattern claims
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /articles/{slug}", h.article)
	mux.HandleFunc("GET /tags/{slug}", h.tag)
	mux.HandleFunc("GET /archive", h.archive)
	mux.HandleFunc("GET /projects", h.projects)
	mux.HandleFunc("GET /docs", h.docsIndex)
	mux.HandleFunc("GET /docs/{project}", h.docsProject)
	mux.HandleFunc("GET /docs/{project}/{slug}", h.docPage)
	mux.HandleFunc("GET /rss.xml", h.rss)

	static := http.FileServerFS(staticFS())
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheStatic(static)))

	return mux, nil
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		next.ServeHTTP(w, r)
	})
}
