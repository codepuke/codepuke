package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/codepuke/codepuke/internal/content"
)

// sourcesFile is the sync manifest the operator edits: which repos to pull
// from, at which refs. Paths are relative to the sources file's directory.
type sourcesFile struct {
	Sources []sourceSpec `json:"sources"`
}

type sourceSpec struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Ref      string        `json:"ref"`
	Lang     string        `json:"lang"`
	Projects []projectSpec `json:"projects"`
}

type projectSpec struct {
	Slug string `json:"slug"`
	// Docs is the repo-relative docs directory; empty or missing means the
	// project ships no docs (yet).
	Docs string `json:"docs"`
}

// langInfo maps a source language to its line-comment token and the file
// extension both scanned for and written.
var langInfo = map[string]struct {
	CommentPrefix string
	Ext           string
}{
	"go":         {"//", ".go"},
	"typescript": {"//", ".ts"},
	"python":     {"#", ".py"},
	"csharp":     {"//", ".cs"},
}

var docPrefixRe = regexp.MustCompile(`^\d+[-_]`)

func run(sourcesPath, outDir string) error {
	raw, err := os.ReadFile(sourcesPath)
	if err != nil {
		return err
	}
	var sources sourcesFile
	if err := json.Unmarshal(raw, &sources); err != nil {
		return fmt.Errorf("parse %s: %w", sourcesPath, err)
	}
	if len(sources.Sources) == 0 {
		return fmt.Errorf("%s lists no sources", sourcesPath)
	}
	baseDir := filepath.Dir(sourcesPath)

	var (
		manifest     content.Manifest
		snippetFiles = map[string]string{} // content-relative path -> code
		docFiles     = map[string]string{} // content-relative path -> body
		seenVariant  = map[string]string{} // topic/lang -> source name
		seenProject  = map[string]string{} // project slug -> source name
	)

	for _, src := range sources.Sources {
		info, ok := langInfo[src.Lang]
		if !ok {
			return fmt.Errorf("source %s: unknown lang %q", src.Name, src.Lang)
		}
		repoPath := src.Path
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(baseDir, repoPath)
		}
		commit, err := resolveCommit(repoPath, src.Ref)
		if err != nil {
			return err
		}

		ms := content.ManifestSource{
			Name:   src.Name,
			Ref:    src.Ref,
			Commit: commit,
			Lang:   src.Lang,
			Topics: []string{},
		}

		type docPage struct {
			project string
			source  string
			slug    string
			title   string
			body    string
		}
		var pages []docPage

		err = forEachFile(repoPath, commit, func(name string, data []byte) error {
			if strings.HasSuffix(name, info.Ext) {
				snippets, err := extractSnippets(data, info.CommentPrefix)
				if err != nil {
					return fmt.Errorf("%s: %s: %w", src.Name, name, err)
				}
				for _, sn := range snippets {
					key := sn.Topic + "/" + src.Lang
					if from, dup := seenVariant[key]; dup {
						return fmt.Errorf("%s: %s: topic %q already provided in %s for %s", src.Name, name, sn.Topic, from, src.Lang)
					}
					seenVariant[key] = src.Name
					ms.Topics = append(ms.Topics, sn.Topic)
					snippetFiles[path.Join("examples", sn.Topic, src.Lang+info.Ext)] = sn.Code
				}
			}

			for _, proj := range src.Projects {
				if proj.Docs == "" || !strings.HasPrefix(name, proj.Docs+"/") || !strings.HasSuffix(name, ".md") {
					continue
				}
				meta, body, err := parseFrontMatter(data)
				if err != nil {
					return fmt.Errorf("%s: %s: %w", src.Name, name, err)
				}
				slug := content.Slugify(docPrefixRe.ReplaceAllString(strings.TrimSuffix(path.Base(name), ".md"), ""))
				if slug == "" {
					return fmt.Errorf("%s: %s: filename yields an empty slug", src.Name, name)
				}
				title := meta["title"]
				if title == "" {
					title = firstHeading(body)
				}
				if title == "" {
					title = slug
				}
				pages = append(pages, docPage{
					project: proj.Slug,
					source:  name,
					slug:    slug,
					title:   title,
					body:    string(body),
				})
			}
			return nil
		})
		if err != nil {
			return err
		}

		slices.Sort(ms.Topics)
		ms.Topics = slices.Compact(ms.Topics)

		// Nav order is the docs-directory filename order; number filenames
		// (00-overview.md) in the source repo to control it.
		slices.SortFunc(pages, func(a, b docPage) int {
			if c := strings.Compare(a.project, b.project); c != 0 {
				return c
			}
			return strings.Compare(a.source, b.source)
		})

		for _, proj := range src.Projects {
			if from, dup := seenProject[proj.Slug]; dup {
				return fmt.Errorf("project %q declared by both %s and %s", proj.Slug, from, src.Name)
			}
			seenProject[proj.Slug] = src.Name

			mp := content.ManifestProject{Slug: proj.Slug, Docs: []content.ManifestDoc{}}
			seenSlug := map[string]bool{}
			for _, page := range pages {
				if page.project != proj.Slug {
					continue
				}
				if seenSlug[page.slug] {
					return fmt.Errorf("%s: project %s has two docs with slug %q", src.Name, proj.Slug, page.slug)
				}
				seenSlug[page.slug] = true
				file := path.Join("docs", proj.Slug, page.slug+".md")
				docFiles[file] = page.body
				mp.Docs = append(mp.Docs, content.ManifestDoc{
					Slug:   page.slug,
					Title:  page.title,
					File:   file,
					Source: page.source,
				})
			}
			ms.Projects = append(ms.Projects, mp)
		}

		manifest.Sources = append(manifest.Sources, ms)
		fmt.Printf("%s @ %s: %d topics, %d docs\n", src.Name, commit[:12], len(ms.Topics), len(pages))
	}

	return writeOutput(outDir, &manifest, snippetFiles, docFiles)
}

// firstHeading returns the text of the first "# " line, the title fallback
// for docs without front matter.
func firstHeading(body []byte) string {
	for line := range strings.SplitSeq(string(body), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "# "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// writeOutput fully rebuilds the generated portions of the content tree:
// examples/, docs/, and manifest.json. Everything else in outDir is left
// alone. The .gitkeep files keep the go:embed patterns valid when a
// directory would otherwise be empty.
func writeOutput(outDir string, manifest *content.Manifest, snippets, docs map[string]string) error {
	for _, dir := range []string{"examples", "docs"} {
		full := filepath.Join(outDir, dir)
		if err := os.RemoveAll(full); err != nil {
			return err
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(full, ".gitkeep"), nil, 0o644); err != nil {
			return err
		}
	}

	for _, files := range []map[string]string{snippets, docs} {
		for file, data := range files {
			full := filepath.Join(outDir, filepath.FromSlash(file))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
				return err
			}
		}
	}

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "manifest.json"), append(out, '\n'), 0o644)
}
