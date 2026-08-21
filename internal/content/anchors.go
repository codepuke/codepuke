package content

import (
	"fmt"
	stdhtml "html"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// headingAnchors implements the 0x-style anchor contract from
// design-system.md section 4f: every heading gets a text-derived id
// (deduplicated with -2 suffixes), and h2 additionally gets a leading
// offset-anchor chip labeled with its 1-based ordinal as 0x%02x. Ids are
// text-derived so deep links survive reordering; labels are positional and
// renumber on the next render.
type headingAnchors struct{}

func (headingAnchors) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	seen := map[string]bool{}
	ordinal := 0

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		id := explicitID(h)
		if id != "" {
			seen[id] = true
		} else {
			id = dedupe(seen, Slugify(textOf(h, source)))
			h.SetAttributeString("id", []byte(id))
		}

		if h.Level == 2 {
			ordinal++
			chip := &anchorChip{id: id, label: fmt.Sprintf("0x%02x", ordinal)}
			if first := h.FirstChild(); first != nil {
				h.InsertBefore(h, first, chip)
			} else {
				h.AppendChild(h, chip)
			}
		}
		return ast.WalkSkipChildren, nil
	})
}

// explicitID returns an id the author set via the attributes extension
// (## Heading {#custom}), or "".
func explicitID(h *ast.Heading) string {
	v, ok := h.AttributeString("id")
	if !ok {
		return ""
	}
	switch id := v.(type) {
	case []byte:
		return string(id)
	case string:
		return id
	}
	return ""
}

// textOf concatenates the plain text content of a node's subtree.
func textOf(n ast.Node, source []byte) string {
	var sb strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			sb.Write(t.Segment.Value(source))
		case *ast.String:
			sb.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return sb.String()
}

// anchorChip is the visible h2 anchor: <a class="offset-anchor">0x01</a>.
type anchorChip struct {
	ast.BaseInline
	id    string
	label string
}

var kindAnchorChip = ast.NewNodeKind("AnchorChip")

func (*anchorChip) Kind() ast.NodeKind { return kindAnchorChip }

func (n *anchorChip) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"id": n.id, "label": n.label}, nil)
}

func renderAnchorChip(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	c := node.(*anchorChip)
	fmt.Fprintf(w, `<a class="offset-anchor" href="#%s">%s</a> `, stdhtml.EscapeString(c.id), c.label)
	return ast.WalkContinue, nil
}
