package content_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codepuke/codepuke/internal/content"
)

type fakeExamples map[string][]content.Example

func (f fakeExamples) Examples(topic string) ([]content.Example, error) {
	ex, ok := f[topic]
	if !ok {
		return nil, fmt.Errorf("unknown topic %q", topic)
	}
	return ex, nil
}

type fakeMermaid struct {
	svg []byte
	err error
}

func (f fakeMermaid) RenderSVG(context.Context, []byte) ([]byte, error) {
	return f.svg, f.err
}

func render(t *testing.T, p *content.Pipeline, md string) string {
	t.Helper()
	out, err := p.Render(t.Context(), []byte(md))
	require.NoError(t, err)
	return string(out)
}

func TestRenderMarkdown(t *testing.T) {
	t.Parallel()
	p := content.New(content.Options{})

	t.Run("gfm table", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "| a | b |\n| --- | --- |\n| 1 | 2 |\n")
		assert.Contains(t, out, "<table>")
		assert.Contains(t, out, "<td>1</td>")
	})

	t.Run("gfm strikethrough", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "~~gone~~")
		assert.Contains(t, out, "<del>gone</del>")
	})

	t.Run("footnote shape matches the 4e contract", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "assigned type ID 65.[^1]\n\n[^1]: Type IDs 0 through 64 are reserved.\n")
		assert.Contains(t, out, `<sup id="fnref:1">`)
		assert.Contains(t, out, `class="footnote-ref"`)
		assert.Contains(t, out, `role="doc-noteref"`)
		assert.Contains(t, out, `role="doc-endnotes"`)
		assert.Contains(t, out, `class="footnote-backref"`)
		assert.Contains(t, out, `role="doc-backlink"`)
	})

	t.Run("code block wrapped in scroll-box with chroma classes", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "```go\nfunc main() {}\n```\n")
		assert.Contains(t, out, "<scroll-box><pre")
		assert.Contains(t, out, `class="chroma"`)
		assert.Contains(t, out, `<span class="`)
		assert.Contains(t, out, "</pre></scroll-box>")
	})

	t.Run("script stripped", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "hello\n\n<script>alert(1)</script>\n")
		assert.NotContains(t, out, "<script")
		assert.NotContains(t, out, "alert(1)")
	})

	t.Run("event handlers stripped", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, `<p onclick="evil()">click</p>`)
		assert.NotContains(t, out, "onclick")
		assert.Contains(t, out, "click")
	})

	t.Run("hexdump figure passes sanitization", func(t *testing.T) {
		t.Parallel()
		md := "<figure class=\"hexdump\">\n<pre class=\"hexdump-16\">00000000  1f ff <span class=\"hl\">81</span>  |...|</pre>\n<figcaption>gob.Encode, all of it.</figcaption>\n</figure>\n"
		out := render(t, p, md)
		assert.Contains(t, out, `<figure class="hexdump">`)
		assert.Contains(t, out, `<pre class="hexdump-16">`)
		assert.Contains(t, out, `<span class="hl">`)
		assert.Contains(t, out, "<figcaption>")
	})
}

func TestHeadingAnchors(t *testing.T) {
	t.Parallel()
	p := content.New(content.Options{})

	t.Run("h2 gets id and 0x chip", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "## The Type Descriptor Comes First\n\ntext\n\n## Second Section\n")
		assert.Contains(t, out,
			`<h2 id="the-type-descriptor-comes-first"><a class="offset-anchor" href="#the-type-descriptor-comes-first">0x01</a> The Type Descriptor Comes First</h2>`)
		assert.Contains(t, out,
			`<a class="offset-anchor" href="#second-section">0x02</a>`)
	})

	t.Run("h3 gets id only", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "### Deep Dive\n")
		assert.Contains(t, out, `<h3 id="deep-dive">Deep Dive</h3>`)
		assert.NotContains(t, out, "offset-anchor")
	})

	t.Run("collisions get -2 suffixes", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "## Dup\n\n## Dup\n\n## Dup\n")
		assert.Contains(t, out, `id="dup"`)
		assert.Contains(t, out, `id="dup-2"`)
		assert.Contains(t, out, `id="dup-3"`)
	})

	t.Run("explicit attribute id wins", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "## Fancy Title {#custom-id}\n")
		assert.Contains(t, out, `id="custom-id"`)
		assert.Contains(t, out, `href="#custom-id"`)
		assert.NotContains(t, out, "fancy-title")
	})

	t.Run("symbols-only heading falls back to section", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "## !!!\n\n## ???\n")
		assert.Contains(t, out, `id="section"`)
		assert.Contains(t, out, `id="section-2"`)
	})

	t.Run("chip label counts only h2", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, "### Three\n\n## One\n\n### Also Three\n\n## Two\n")
		assert.Contains(t, out, `href="#one">0x01</a>`)
		assert.Contains(t, out, `href="#two">0x02</a>`)
	})
}

func TestExamplesBlock(t *testing.T) {
	t.Parallel()

	source := fakeExamples{
		"encode-struct": {
			{Lang: "go", Code: "package main"},
			{Lang: "typescript", Code: "const x = 1"},
			{Lang: "python", Code: "x = 1"},
			{Lang: "csharp", Code: "var x = 1;"},
		},
		"partial": {
			{Lang: "python", Code: "x = 1"},
			{Lang: "go", Code: "package main"},
		},
	}
	p := content.New(content.Options{Examples: source})

	t.Run("full expansion in site-wide language order", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, ":::examples encode-struct\n")
		assert.Contains(t, out, `<code-tabs data-topic="encode-struct">`)
		for _, want := range []string{
			`<section data-lang="go"><h4 class="ct-label">Go</h4><scroll-box>`,
			`<section data-lang="typescript"><h4 class="ct-label">TypeScript</h4>`,
			`<section data-lang="python"><h4 class="ct-label">Python</h4>`,
			`<section data-lang="csharp"><h4 class="ct-label">C#</h4>`,
		} {
			assert.Contains(t, out, want)
		}
		order := []int{
			strings.Index(out, `data-lang="go"`),
			strings.Index(out, `data-lang="typescript"`),
			strings.Index(out, `data-lang="python"`),
			strings.Index(out, `data-lang="csharp"`),
		}
		assert.IsIncreasing(t, order, "sections must follow the site-wide order")
		assert.Contains(t, out, `class="chroma"`, "variants ship pre-highlighted")
	})

	t.Run("partial topic renders only its variants, ordered", func(t *testing.T) {
		t.Parallel()
		out := render(t, p, ":::examples partial\n")
		assert.Contains(t, out, `data-lang="go"`)
		assert.Contains(t, out, `data-lang="python"`)
		assert.NotContains(t, out, `data-lang="typescript"`)
		assert.Less(t, strings.Index(out, `data-lang="go"`), strings.Index(out, `data-lang="python"`))
	})

	t.Run("unknown topic errors", func(t *testing.T) {
		t.Parallel()
		_, err := p.Render(t.Context(), []byte(":::examples nope\n"))
		require.ErrorContains(t, err, `unknown topic "nope"`)
	})

	t.Run("missing topic id errors", func(t *testing.T) {
		t.Parallel()
		_, err := p.Render(t.Context(), []byte(":::examples\n"))
		require.ErrorContains(t, err, "requires a topic id")
	})

	t.Run("unknown directive errors", func(t *testing.T) {
		t.Parallel()
		_, err := p.Render(t.Context(), []byte(":::mystery box\n"))
		require.ErrorContains(t, err, `unknown block directive "mystery"`)
	})

	t.Run("no source configured errors", func(t *testing.T) {
		t.Parallel()
		bare := content.New(content.Options{})
		_, err := bare.Render(t.Context(), []byte(":::examples encode-struct\n"))
		require.ErrorContains(t, err, "no example source configured")
	})
}

func TestMermaid(t *testing.T) {
	t.Parallel()

	const md = "before\n\n```mermaid\ngraph TD; A-->B;\n```\n\nafter\n"

	t.Run("rendered svg is inlined in a figure", func(t *testing.T) {
		t.Parallel()
		svg := []byte(`<svg viewBox="0 0 10 10"><path d="M0 0L10 10"/></svg>`)
		p := content.New(content.Options{Mermaid: fakeMermaid{svg: svg}})
		out := render(t, p, md)
		assert.Contains(t, out, `<scroll-box><figure class="fig"><svg viewBox="0 0 10 10">`)
		assert.NotContains(t, out, "codepuke-mermaid")
		assert.NotContains(t, out, "graph TD")
	})

	t.Run("noop renderer leaves a highlighted code block", func(t *testing.T) {
		t.Parallel()
		p := content.New(content.Options{})
		out := render(t, p, md)
		assert.Contains(t, out, "graph TD")
		assert.Contains(t, out, "<scroll-box><pre")
		assert.NotContains(t, out, "<svg")
	})

	t.Run("renderer failure is a render error", func(t *testing.T) {
		t.Parallel()
		p := content.New(content.Options{Mermaid: fakeMermaid{err: fmt.Errorf("sidecar exploded")}})
		_, err := p.Render(t.Context(), []byte(md))
		require.ErrorContains(t, err, "sidecar exploded")
	})
}

// FuzzRender feeds arbitrary markdown through the full pipeline: it must
// never panic, and any error must come from the directive path.
func FuzzRender(f *testing.F) {
	f.Add("# Title\n\nsome *text*\n")
	f.Add("## A\n## A\n### A\n")
	f.Add("```go\nfunc main() {}\n```\n")
	f.Add(":::examples encode-struct\n")
	f.Add("| a |\n| - |\n| b |\n")
	f.Add("<figure class=\"hexdump\"><pre>00</pre></figure>\n")
	f.Add("text[^1]\n\n[^1]: note\n")

	p := content.New(content.Options{Examples: fakeExamples{}})
	f.Fuzz(func(t *testing.T, md string) {
		_, _ = p.Render(t.Context(), []byte(md))
	})
}
