package content

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// Manifest is content/manifest.json, written by cmd/sync and read by the
// server. It records what was synced from where, and it is the source of
// truth for docs navigation order (design-system.md 4d).
type Manifest struct {
	Sources []ManifestSource `json:"sources"`
}

// ManifestSource is one synced repository at one resolved commit.
type ManifestSource struct {
	Name     string            `json:"name"`
	Ref      string            `json:"ref"`
	Commit   string            `json:"commit"`
	Lang     string            `json:"lang"`
	Topics   []string          `json:"topics"`
	Projects []ManifestProject `json:"projects"`
}

// ManifestProject is one project's ordered docs list.
type ManifestProject struct {
	Slug string        `json:"slug"`
	Docs []ManifestDoc `json:"docs"`
}

// ManifestDoc is one synced docs page. File is relative to the content root;
// Source is the path inside the origin repository.
type ManifestDoc struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	File   string `json:"file"`
	Source string `json:"source"`
}

// LoadManifest reads manifest.json from the root of a content tree.
func LoadManifest(fsys fs.FS) (*Manifest, error) {
	data, err := fs.ReadFile(fsys, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}
