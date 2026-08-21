package web

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/codepuke/codepuke/ui"
)

func TestExtractSections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		want []ui.DocSection
	}{
		// valid
		{
			name: "two sections",
			html: `<h2 id="inspector"><a class="offset-anchor" href="#inspector">0x01</a> Inspector and Sessions</h2><p>x</p>` +
				`<h2 id="queries"><a class="offset-anchor" href="#queries">0x02</a> Queries</h2>`,
			want: []ui.DocSection{
				{ID: "inspector", Label: "0x01", Title: "Inspector and Sessions"},
				{ID: "queries", Label: "0x02", Title: "Queries"},
			},
		},
		{
			name: "inline markup stripped from titles",
			html: `<h2 id="the-gob-encoder"><a class="offset-anchor" href="#the-gob-encoder">0x01</a> The <code>gob.Encoder</code> type</h2>`,
			want: []ui.DocSection{{ID: "the-gob-encoder", Label: "0x01", Title: "The gob.Encoder type"}},
		},
		{
			name: "entities unescaped",
			html: `<h2 id="a-b"><a class="offset-anchor" href="#a-b">0x01</a> A &amp; B</h2>`,
			want: []ui.DocSection{{ID: "a-b", Label: "0x01", Title: "A & B"}},
		},

		// invalid / edge
		{name: "no headings", html: `<p>plain</p>`, want: nil},
		{
			name: "h2 without anchor chip ignored",
			html: `<h2 id="raw">Raw</h2>`,
			want: nil,
		},
		{
			name: "h3 ignored",
			html: `<h3 id="deep">Deep</h3>`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, extractSections(tt.html))
		})
	}
}
