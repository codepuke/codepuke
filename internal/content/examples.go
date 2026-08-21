package content

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/util"
)

// languages is the site-wide language order and labeling for example
// variants (design-system.md 4a): Lang is the chroma lexer name and the
// data-lang value, Label is the visible ct-label text.
var languages = []struct{ Lang, Label string }{
	{"go", "Go"},
	{"typescript", "TypeScript"},
	{"python", "Python"},
	{"csharp", "C#"},
}

// Example is one language variant of a topic.
type Example struct {
	Lang string // chroma lexer name, also the data-lang value
	Code string
}

// ExampleSource resolves a topic id to its language variants. Stage 3 wires
// the embedded content/examples tree; tests use fakes.
type ExampleSource interface {
	Examples(topic string) ([]Example, error)
}

var topicRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// renderExamplesBlock expands ":::examples <topic>" into the <code-tabs>
// contract of design-system.md 4a: every variant ships pre-highlighted, in
// the site-wide language order.
func renderExamplesBlock(w util.BufWriter, p *Pipeline, args string) error {
	topic := strings.TrimSpace(args)
	if topic == "" {
		return errors.New(":::examples requires a topic id")
	}
	if !topicRe.MatchString(topic) {
		return fmt.Errorf(":::examples topic %q is not a valid topic id", topic)
	}
	if p.examples == nil {
		return fmt.Errorf(":::examples %s: no example source configured", topic)
	}

	examples, err := p.examples.Examples(topic)
	if err != nil {
		return fmt.Errorf(":::examples %s: %w", topic, err)
	}
	if len(examples) == 0 {
		return fmt.Errorf(":::examples %s: no language variants", topic)
	}

	byLang := make(map[string]Example, len(examples))
	for _, ex := range examples {
		byLang[ex.Lang] = ex
	}
	for _, ex := range examples {
		if !knownLang(ex.Lang) {
			return fmt.Errorf(":::examples %s: unknown language %q", topic, ex.Lang)
		}
	}

	fmt.Fprintf(w, `<code-tabs data-topic="%s">`, topic)
	for _, l := range languages {
		ex, ok := byLang[l.Lang]
		if !ok {
			continue
		}
		fmt.Fprintf(w, `<section data-lang="%s"><h4 class="ct-label">%s</h4><scroll-box>`, l.Lang, l.Label)
		if err := highlightTo(w, ex.Lang, ex.Code); err != nil {
			return fmt.Errorf(":::examples %s: highlight %s: %w", topic, ex.Lang, err)
		}
		if _, err := w.WriteString(`</scroll-box></section>`); err != nil {
			return err
		}
	}
	_, err = w.WriteString(`</code-tabs>`)
	return err
}

func knownLang(lang string) bool {
	for _, l := range languages {
		if l.Lang == lang {
			return true
		}
	}
	return false
}
