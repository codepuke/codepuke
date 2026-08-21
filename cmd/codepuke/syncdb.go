package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	contentfs "github.com/codepuke/codepuke/content"
	"github.com/codepuke/codepuke/internal/config"
	"github.com/codepuke/codepuke/internal/content"
	"github.com/codepuke/codepuke/internal/store"
)

// buildPipeline wires the render pipeline the server uses: embedded example
// snippets, and the kroki-mermaid sidecar when MERMAID_URL is set.
func buildPipeline(ctx context.Context, cfg config.Config) (*content.Pipeline, error) {
	opts := content.Options{Examples: content.NewFSSource(contentfs.FS)}
	if cfg.MermaidURL != "" {
		kroki := content.NewKrokiMermaid(cfg.MermaidURL)
		if err := waitForMermaid(ctx, kroki); err != nil {
			return nil, err
		}
		opts.Mermaid = kroki
	}
	return content.New(opts), nil
}

// waitForMermaid polls the sidecar until it answers; pod containers start in
// no particular order.
func waitForMermaid(ctx context.Context, kroki *content.KrokiMermaid) error {
	deadline := time.Now().Add(90 * time.Second)
	for {
		err := kroki.Ready(ctx)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("mermaid sidecar never became ready: %w", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// syncDB renders every docs page in the embedded content tree through the
// pipeline and upserts it, then prunes rows the manifest no longer lists.
// Render happens here, at deploy time, never in request handlers.
func syncDB(ctx context.Context, st *store.Store, pipeline *content.Pipeline) error {
	manifest, err := content.LoadManifest(contentfs.FS)
	if err != nil {
		return err
	}

	var keep []string
	for _, src := range manifest.Sources {
		for _, proj := range src.Projects {
			for _, doc := range proj.Docs {
				md, err := contentfs.FS.ReadFile(doc.File)
				if err != nil {
					return fmt.Errorf("read %s: %w", doc.File, err)
				}
				html, err := pipeline.Render(ctx, md, content.WithExamplesDefault(src.Lang))
				if err != nil {
					return fmt.Errorf("render %s: %w", doc.File, err)
				}
				if err := st.UpsertDoc(ctx, proj.Slug, doc.Slug, doc.Title, string(md), string(html), content.RenderVersion); err != nil {
					return err
				}
				keep = append(keep, proj.Slug+"/"+doc.Slug)
			}
		}
	}

	pruned, err := st.PruneDocs(ctx, keep)
	if err != nil {
		return err
	}
	slog.Info("docs synced", "upserted", len(keep), "pruned", pruned)
	return nil
}
