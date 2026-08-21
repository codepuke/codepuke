package content

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// blockRenderers is the single registry of custom ::: block extensions.
// Adding an extension means adding an entry here and nothing else.
var blockRenderers = map[string]func(w util.BufWriter, p *Pipeline, d *directiveBlock) error{
	"examples": renderExamplesBlock,
}

// directiveBlock is a single-line ":::name args" block. DefaultLang carries
// the render call's WithExamplesDefault value.
type directiveBlock struct {
	ast.BaseBlock
	Name        string
	Args        string
	DefaultLang string
}

var kindDirectiveBlock = ast.NewNodeKind("DirectiveBlock")

func (*directiveBlock) Kind() ast.NodeKind { return kindDirectiveBlock }

func (n *directiveBlock) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"name": n.Name, "args": n.Args}, nil)
}

var directiveRe = regexp.MustCompile(`^:{3}([a-z][a-z0-9-]*)(?:[ \t]+(.*?))?[ \t]*$`)

type directiveParser struct{}

func (directiveParser) Trigger() []byte { return []byte{':'} }

func (directiveParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	m := directiveRe.FindSubmatch(bytes.TrimRight(line, "\r\n"))
	if m == nil {
		return nil, parser.NoChildren
	}
	reader.Advance(segment.Len() - 1)
	defaultLang, _ := pc.Get(defaultLangKey).(string)
	return &directiveBlock{Name: string(m[1]), Args: string(m[2]), DefaultLang: defaultLang}, parser.NoChildren
}

func (directiveParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return parser.Close
}

func (directiveParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (directiveParser) CanInterruptParagraph() bool { return false }

func (directiveParser) CanAcceptIndentedLine() bool { return false }

func renderDirective(w util.BufWriter, p *Pipeline, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	d := node.(*directiveBlock)
	render, ok := blockRenderers[d.Name]
	if !ok {
		return ast.WalkStop, fmt.Errorf("unknown block directive %q", d.Name)
	}
	if err := render(w, p, d); err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkContinue, nil
}
