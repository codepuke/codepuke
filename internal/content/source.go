package content

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// FSSource resolves :::examples topics from a synced content tree laid out
// as examples/<topic>/<lang>.<ext>, typically the embedded content FS or a
// content directory on disk.
type FSSource struct {
	fsys fs.FS
}

// NewFSSource wraps a content tree root.
func NewFSSource(fsys fs.FS) FSSource {
	return FSSource{fsys: fsys}
}

// Examples returns the topic's language variants in the site-wide language
// order. The variant language is the filename stem (go.go, typescript.ts).
func (s FSSource) Examples(topic string) ([]Example, error) {
	if !topicRe.MatchString(topic) {
		return nil, fmt.Errorf("invalid topic %q", topic)
	}
	entries, err := fs.ReadDir(s.fsys, path.Join("examples", topic))
	if err != nil {
		return nil, fmt.Errorf("unknown topic %q", topic)
	}

	byLang := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		lang := strings.TrimSuffix(name, path.Ext(name))
		if !knownLang(lang) {
			continue
		}
		data, err := fs.ReadFile(s.fsys, path.Join("examples", topic, name))
		if err != nil {
			return nil, err
		}
		byLang[lang] = string(data)
	}

	var out []Example
	for _, l := range languages {
		if code, ok := byLang[l.Lang]; ok {
			out = append(out, Example{Lang: l.Lang, Code: code})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("topic %q has no language variants", topic)
	}
	return out, nil
}
