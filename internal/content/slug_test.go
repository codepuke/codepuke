package content

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, in, want string
	}{
		// valid
		{"simple words", "The Type Descriptor Comes First", "the-type-descriptor-comes-first"},
		{"already a slug", "already-a-slug", "already-a-slug"},
		{"digits kept", "Go 1.27 rc2", "go-1-27-rc2"},

		// invalid characters collapse
		{"punctuation runs", "what?? really!!", "what-really"},
		{"leading and trailing junk", "  --hello--  ", "hello"},
		{"symbols only", "!!! ???", ""},
		{"empty", "", ""},

		// edge
		{"unicode letters dropped", "übertype schnell", "bertype-schnell"},
		{"emoji dropped", "ship it 🚀 now", "ship-it-now"},
		{"uppercase ascii", "ENCODE/DECODE", "encode-decode"},
		{"code span text", "the gob.Encoder type", "the-gob-encoder-type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Slugify(tt.in))
		})
	}
}

func TestDedupe(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	assert.Equal(t, "dup", dedupe(seen, "dup"))
	assert.Equal(t, "dup-2", dedupe(seen, "dup"))
	assert.Equal(t, "dup-3", dedupe(seen, "dup"))
	assert.Equal(t, "section", dedupe(seen, ""))
	assert.Equal(t, "section-2", dedupe(seen, ""))
	// an explicit heading that already looks like a suffix must not collide
	assert.Equal(t, "dup-4", dedupe(seen, "dup"), "dup-2 and dup-3 are taken")
}

func FuzzSlugify(f *testing.F) {
	for _, seed := range []string{"", "Hello World", "a--b", "über 🚀", "0x01 style", strings.Repeat("-", 64)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := Slugify(s)
		if got != "" && !slugRe.MatchString(got) {
			t.Fatalf("Slugify(%q) = %q, not slug-shaped", s, got)
		}
		if again := Slugify(got); again != got {
			t.Fatalf("Slugify not idempotent: %q -> %q -> %q", s, got, again)
		}
	})
}

// FuzzHeadingIDs fuzzes the anchor id derivation: any sequence of heading
// texts must produce unique, slug-shaped, non-empty ids.
func FuzzHeadingIDs(f *testing.F) {
	f.Add("Alpha\nAlpha\n!!!\nAlpha-2")
	f.Add("\n\n\n")
	f.Add("same\nsame\nsame\nsame-2\nsame-2")
	f.Fuzz(func(t *testing.T, input string) {
		seen := map[string]bool{}
		issued := map[string]bool{}
		for line := range strings.SplitSeq(input, "\n") {
			id := dedupe(seen, Slugify(line))
			if id == "" {
				t.Fatalf("empty id for heading %q", line)
			}
			if issued[id] {
				t.Fatalf("duplicate id %q for heading %q", id, line)
			}
			issued[id] = true
			if !slugRe.MatchString(id) {
				t.Fatalf("id %q for heading %q is not slug-shaped", id, line)
			}
		}
	})
}
