package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontMatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		in       string
		wantMeta map[string]string
		wantBody string
		wantErr  string
	}{
		// valid
		{
			name:     "title and body",
			in:       "---\ntitle: Reading Streams\n---\n# Heading\n\nbody\n",
			wantMeta: map[string]string{"title": "Reading Streams"},
			wantBody: "# Heading\n\nbody\n",
		},
		{
			name:     "no front matter passes through",
			in:       "# Heading\n\nbody\n",
			wantMeta: map[string]string{},
			wantBody: "# Heading\n\nbody\n",
		},
		{
			name:     "several keys and blank lines",
			in:       "---\ntitle: A\n\nweight: 3\n---\nbody\n",
			wantMeta: map[string]string{"title": "A", "weight": "3"},
			wantBody: "body\n",
		},
		{
			name:     "crlf",
			in:       "---\r\ntitle: A\r\n---\r\nbody\r\n",
			wantMeta: map[string]string{"title": "A"},
			wantBody: "body\r\n",
		},

		// invalid
		{
			name:    "unterminated block",
			in:      "---\ntitle: A\n",
			wantErr: "never closed",
		},
		{
			name:    "non key-value line",
			in:      "---\njust some text\n---\nbody\n",
			wantErr: `not "key: value"`,
		},

		// edge
		{
			name:     "empty block",
			in:       "---\n---\nbody\n",
			wantMeta: map[string]string{},
			wantBody: "body\n",
		},
		{
			name:     "closing delimiter at end of file",
			in:       "---\ntitle: A\n---",
			wantMeta: map[string]string{"title": "A"},
			wantBody: "",
		},
		{
			name:     "dashes mid-document are body",
			in:       "body\n---\nmore\n",
			wantMeta: map[string]string{},
			wantBody: "body\n---\nmore\n",
		},
		{
			name:     "empty value",
			in:       "---\ntitle:\n---\nbody\n",
			wantMeta: map[string]string{"title": ""},
			wantBody: "body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			meta, body, err := parseFrontMatter([]byte(tt.in))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMeta, meta)
			assert.Equal(t, tt.wantBody, string(body))
		})
	}
}

func FuzzParseFrontMatter(f *testing.F) {
	f.Add("---\ntitle: A\n---\nbody\n")
	f.Add("---\n---\n")
	f.Add("no front matter\n")
	f.Add("---\nunclosed\n")
	f.Add("---\r\nk: v\r\n---\r\n")

	f.Fuzz(func(t *testing.T, in string) {
		data := []byte(in)
		meta, body, err := parseFrontMatter(data)
		if err != nil {
			return
		}
		if body == nil {
			t.Fatal("nil body without error")
		}
		if !bytes.HasSuffix(data, body) {
			t.Fatalf("body %q is not a suffix of input %q", body, in)
		}
		for k := range meta {
			if !frontMatterLineRe.MatchString(k + ":") {
				t.Fatalf("invalid key %q", k)
			}
		}
	})
}
