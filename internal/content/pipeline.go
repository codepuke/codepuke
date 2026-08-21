// Package content is the markdown render pipeline: goldmark with GFM,
// footnotes, and attributes; 0x-style heading anchors; chroma with classed
// output; the ::: block extension registry; mermaid rendering; and
// bluemonday sanitization. Rendering happens at write time; request handlers
// never call this package.
package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// RenderVersion is stamped on every stored render. Bump it whenever the
// pipeline's output changes shape, then run codepuke reflow.
const RenderVersion = 1

// Options configures a Pipeline.
type Options struct {
	// Mermaid renders diagram source to SVG; nil means NoopMermaid, which
	// leaves diagrams as highlighted code blocks.
	Mermaid MermaidRenderer
	// Examples resolves :::examples topics; nil makes any :::examples block
	// a render error.
	Examples ExampleSource
}

// Pipeline converts markdown to sanitized HTML. It is safe for concurrent
// use.
type Pipeline struct {
	md       goldmark.Markdown
	policy   *bluemonday.Policy
	mermaid  MermaidRenderer
	examples ExampleSource
}

// New builds a Pipeline.
func New(opts Options) *Pipeline {
	p := &Pipeline{
		policy:   buildPolicy(),
		mermaid:  opts.Mermaid,
		examples: opts.Examples,
	}
	if p.mermaid == nil {
		p.mermaid = NoopMermaid{}
	}

	p.md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
				highlighting.WithGuessLanguage(true),
				highlighting.WithWrapperRenderer(scrollBoxWrapper),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAttribute(),
			parser.WithASTTransformers(util.Prioritized(headingAnchors{}, 100)),
			parser.WithBlockParsers(util.Prioritized(directiveParser{}, 500)),
		),
		goldmark.WithRendererOptions(
			// Raw author HTML (hexdump figures) passes goldmark untouched;
			// bluemonday is the gate.
			html.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(&nodeRenderer{p: p}, 100)),
		),
	)
	return p
}

// scrollBoxWrapper wraps every highlighted code block in the scroll-box
// element per design-system.md 4c; the chroma <pre> lands inside it.
func scrollBoxWrapper(w util.BufWriter, c highlighting.CodeBlockContext, entering bool) {
	if entering {
		_, _ = w.WriteString("<scroll-box>")
	} else {
		_, _ = w.WriteString("</scroll-box>")
	}
}

// nodeRenderer renders the nodes this package invents.
type nodeRenderer struct {
	p *Pipeline
}

func (r *nodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindAnchorChip, renderAnchorChip)
	reg.Register(kindDirectiveBlock, func(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
		return renderDirective(w, r.p, node, entering)
	})
}

// Render converts markdown to sanitized HTML. Mermaid diagrams are rendered
// to SVG first and spliced in after sanitization; if the renderer is
// unavailable they stay highlighted code blocks.
func (p *Pipeline) Render(ctx context.Context, source []byte) ([]byte, error) {
	doc := p.md.Parser().Parse(text.NewReader(source))

	slots, err := p.renderMermaid(ctx, doc, source)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := p.md.Renderer().Render(&buf, source, doc); err != nil {
		return nil, err
	}

	out := p.policy.SanitizeBytes(buf.Bytes())

	for _, s := range slots {
		figure := append([]byte(`<scroll-box><figure class="fig">`), s.svg...)
		figure = append(figure, []byte(`</figure></scroll-box>`)...)
		wrapped := []byte("<p>" + s.token + "</p>")
		if bytes.Contains(out, wrapped) {
			out = bytes.Replace(out, wrapped, figure, 1)
		} else {
			out = bytes.Replace(out, []byte(s.token), figure, 1)
		}
	}
	return out, nil
}

type mermaidSlot struct {
	token string
	svg   []byte
}

// renderMermaid renders every ```mermaid fence and replaces its node with a
// placeholder token that survives sanitization; Render splices the SVG back
// in afterwards so bluemonday never needs an SVG allowlist.
func (p *Pipeline) renderMermaid(ctx context.Context, doc ast.Node, source []byte) ([]mermaidSlot, error) {
	var fences []*ast.FencedCodeBlock
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if fcb, ok := n.(*ast.FencedCodeBlock); entering && ok && string(fcb.Language(source)) == "mermaid" {
			fences = append(fences, fcb)
		}
		return ast.WalkContinue, nil
	})

	var slots []mermaidSlot
	for _, fcb := range fences {
		diagram := fencedContent(fcb, source)
		svg, err := p.mermaid.RenderSVG(ctx, diagram)
		if errors.Is(err, ErrMermaidUnavailable) {
			continue // stays a highlighted code block
		}
		if err != nil {
			return nil, fmt.Errorf("render mermaid: %w", err)
		}

		sum := sha256.Sum256(diagram)
		token := fmt.Sprintf("codepuke-mermaid-%d-%s", len(slots), hex.EncodeToString(sum[:8]))
		para := ast.NewParagraph()
		para.AppendChild(para, ast.NewString([]byte(token)))
		parent := fcb.Parent()
		parent.ReplaceChild(parent, fcb, para)
		slots = append(slots, mermaidSlot{token: token, svg: svg})
	}
	return slots, nil
}

func fencedContent(n *ast.FencedCodeBlock, source []byte) []byte {
	var buf bytes.Buffer
	lines := n.Lines()
	for i := range lines.Len() {
		seg := lines.At(i)
		buf.Write(seg.Value(source))
	}
	return buf.Bytes()
}
