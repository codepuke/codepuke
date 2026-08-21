package content

import (
	"io"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// chromaFormatter emits classed output only; the classes are styled once by
// the stylesheet in design-system.md section 2, so the style passed to Format
// never reaches the HTML.
var chromaFormatter = chromahtml.New(chromahtml.WithClasses(true))

// highlightTo writes code as a classed <pre class="chroma"> block. Unknown
// languages fall back to the plaintext lexer, which still produces the chroma
// wrapper so the design's base rule applies.
func highlightTo(w io.Writer, lang, code string) error {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, code)
	if err != nil {
		return err
	}
	return chromaFormatter.Format(w, styles.Fallback, it)
}
