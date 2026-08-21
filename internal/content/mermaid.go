package content

import (
	"context"
	"errors"
)

// ErrMermaidUnavailable signals that no real renderer is wired in; the
// pipeline then leaves the mermaid source as a highlighted code block, which
// is the readable degradation.
var ErrMermaidUnavailable = errors.New("mermaid renderer unavailable")

// MermaidRenderer turns mermaid diagram source into SVG. The real
// implementation (stage 5) is an HTTP client for the kroki-mermaid sidecar,
// called only at publish and sync time; its output is trusted and inlined
// without sanitization.
type MermaidRenderer interface {
	RenderSVG(ctx context.Context, source []byte) ([]byte, error)
}

// NoopMermaid is the stand-in until the sidecar exists.
type NoopMermaid struct{}

func (NoopMermaid) RenderSVG(context.Context, []byte) ([]byte, error) {
	return nil, ErrMermaidUnavailable
}
