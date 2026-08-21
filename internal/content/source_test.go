package content_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codepuke/codepuke/internal/content"
)

func TestFSSource(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"examples/encode-struct/python.py": {Data: []byte("py code\n")},
		"examples/encode-struct/go.go":     {Data: []byte("go code\n")},
		"examples/encode-struct/.gitkeep":  {Data: nil},
		"examples/encode-struct/ruby.rb":   {Data: []byte("ignored\n")},
		"examples/empty-topic/.gitkeep":    {Data: nil},
		"docs/gobspect/overview.md":        {Data: []byte("# hi\n")},
	}
	src := content.NewFSSource(fsys)

	t.Run("variants in site-wide order", func(t *testing.T) {
		t.Parallel()
		got, err := src.Examples("encode-struct")
		require.NoError(t, err)
		require.Len(t, got, 2, "dotfiles and unknown langs are ignored")
		assert.Equal(t, "go", got[0].Lang)
		assert.Equal(t, "go code\n", got[0].Code)
		assert.Equal(t, "python", got[1].Lang)
	})

	t.Run("unknown topic errors", func(t *testing.T) {
		t.Parallel()
		_, err := src.Examples("missing")
		require.ErrorContains(t, err, `unknown topic "missing"`)
	})

	t.Run("topic without variants errors", func(t *testing.T) {
		t.Parallel()
		_, err := src.Examples("empty-topic")
		require.ErrorContains(t, err, "no language variants")
	})

	t.Run("invalid topic errors without fs access", func(t *testing.T) {
		t.Parallel()
		_, err := src.Examples("../escape")
		require.ErrorContains(t, err, "invalid topic")
	})
}
