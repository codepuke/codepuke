package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractSnippets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  string
		prefix  string
		want    []snippet
		wantErr string
	}{
		// valid
		{
			name:   "basic region",
			prefix: "//",
			source: "package main\n// snippet:start hello\nfunc Hello() {}\n// snippet:end\n",
			want:   []snippet{{Topic: "hello", Code: "func Hello() {}\n"}},
		},
		{
			name:   "indented region dedents",
			prefix: "//",
			source: "func main() {\n\t// snippet:start body\n\tx := 1\n\t\ty := 2\n\t// snippet:end\n}\n",
			want:   []snippet{{Topic: "body", Code: "x := 1\n\ty := 2\n"}},
		},
		{
			name:   "python comments",
			prefix: "#",
			source: "# snippet:start enc\nencode(x)\n# snippet:end\n",
			want:   []snippet{{Topic: "enc", Code: "encode(x)\n"}},
		},
		{
			name:   "surrounding blank lines trimmed",
			prefix: "//",
			source: "// snippet:start t\n\n\ncode\n\n// snippet:end\n",
			want:   []snippet{{Topic: "t", Code: "code\n"}},
		},
		{
			name:   "interior blank line kept",
			prefix: "//",
			source: "// snippet:start t\na\n\nb\n// snippet:end\n",
			want:   []snippet{{Topic: "t", Code: "a\n\nb\n"}},
		},
		{
			name:   "nested regions exclude all marker lines",
			prefix: "//",
			source: "// snippet:start outer\nbefore\n// snippet:start inner\nmiddle\n// snippet:end inner\nafter\n// snippet:end outer\n",
			want: []snippet{
				{Topic: "inner", Code: "middle\n"},
				{Topic: "outer", Code: "before\nmiddle\nafter\n"},
			},
		},
		{
			name:   "bare end closes most recent",
			prefix: "//",
			source: "// snippet:start a\n// snippet:start b\nx\n// snippet:end\ny\n// snippet:end\n",
			want: []snippet{
				{Topic: "b", Code: "x\n"},
				{Topic: "a", Code: "x\ny\n"},
			},
		},
		{
			name:   "crlf input",
			prefix: "//",
			source: "// snippet:start t\r\ncode\r\n// snippet:end\r\n",
			want:   []snippet{{Topic: "t", Code: "code\n"}},
		},
		{
			name:   "no markers",
			prefix: "//",
			source: "package main\nfunc main() {}\n",
			want:   nil,
		},
		{
			name:   "marker without space after comment token",
			prefix: "//",
			source: "//snippet:start t\ncode\n//snippet:end\n",
			want:   []snippet{{Topic: "t", Code: "code\n"}},
		},

		// invalid
		{
			name:    "unclosed region",
			prefix:  "//",
			source:  "// snippet:start t\ncode\n",
			wantErr: `snippet "t" is never closed`,
		},
		{
			name:    "stray end",
			prefix:  "//",
			source:  "code\n// snippet:end\n",
			wantErr: "snippet:end without a matching snippet:start",
		},
		{
			name:    "end names an unopened topic",
			prefix:  "//",
			source:  "// snippet:start a\n// snippet:end b\n",
			wantErr: "snippet:end without a matching snippet:start",
		},
		{
			name:    "duplicate topic",
			prefix:  "//",
			source:  "// snippet:start t\n// snippet:end\n// snippet:start t\n// snippet:end\n",
			wantErr: `duplicate snippet topic "t"`,
		},
		{
			name:    "missing topic",
			prefix:  "//",
			source:  "// snippet:start\ncode\n// snippet:end\n",
			wantErr: "snippet:start requires a topic",
		},
		{
			name:    "invalid topic charset",
			prefix:  "//",
			source:  "// snippet:start Bad_Topic\n// snippet:end\n",
			wantErr: `invalid snippet topic "Bad_Topic"`,
		},
		{
			name:    "marker typo is an error",
			prefix:  "//",
			source:  "// snippet:begin t\n",
			wantErr: `unknown snippet marker "snippet:begin"`,
		},

		// edge
		{
			name:   "empty region",
			prefix: "//",
			source: "// snippet:start t\n// snippet:end\n",
			want:   []snippet{{Topic: "t", Code: ""}},
		},
		{
			name:   "marker text in a string is content",
			prefix: "//",
			source: "// snippet:start t\ns := \"// snippet:end\"\n// snippet:end\n",
			// the string literal line is indented by nothing and contains the
			// marker only after other characters, so it is content
			want: []snippet{{Topic: "t", Code: "s := \"// snippet:end\"\n"}},
		},
		{
			name:   "hash prefix ignores go comments",
			prefix: "#",
			source: "// snippet:start t\ncode\n",
			want:   nil,
		},
		{
			name:   "mixed indent keeps common prefix only",
			prefix: "//",
			source: "// snippet:start t\n  two\n    four\n  two again\n// snippet:end\n",
			want:   []snippet{{Topic: "t", Code: "two\n  four\ntwo again\n"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractSnippets([]byte(tt.source), tt.prefix)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func FuzzExtractSnippets(f *testing.F) {
	f.Add("// snippet:start a\ncode\n// snippet:end\n", "//")
	f.Add("# snippet:start b\nx\n# snippet:end b\n", "#")
	f.Add("// snippet:start outer\n// snippet:start inner\n// snippet:end\n// snippet:end\n", "//")
	f.Add("plain code\n", "//")
	f.Add("// snippet:start\n", "//")

	f.Fuzz(func(t *testing.T, source, prefix string) {
		if prefix != "//" && prefix != "#" {
			t.Skip()
		}
		snippets, err := extractSnippets([]byte(source), prefix)
		if err != nil {
			return
		}
		seen := map[string]bool{}
		for _, sn := range snippets {
			if !snippetTopicRe.MatchString(sn.Topic) {
				t.Fatalf("invalid topic %q extracted", sn.Topic)
			}
			if seen[sn.Topic] {
				t.Fatalf("duplicate topic %q extracted", sn.Topic)
			}
			seen[sn.Topic] = true
			for line := range strings.SplitSeq(sn.Code, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, prefix) &&
					strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), "snippet:") {
					t.Fatalf("marker line leaked into snippet %q: %q", sn.Topic, line)
				}
			}
			if sn.Code != "" && !strings.HasSuffix(sn.Code, "\n") {
				t.Fatalf("snippet %q missing trailing newline", sn.Topic)
			}
		}
	})
}
