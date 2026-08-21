package content_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codepuke/codepuke/internal/content"
)

func TestKrokiMermaid(t *testing.T) {
	t.Parallel()

	t.Run("renders and scopes the container id", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/svg", r.URL.Path)
			body, _ := io.ReadAll(r.Body)
			require.Equal(t, "graph TD; A-->B;", string(body))
			io.WriteString(w, `<svg id="container"><style>#container{fill:red}</style></svg>`)
		}))
		t.Cleanup(srv.Close)

		k := content.NewKrokiMermaid(srv.URL)
		svg, err := k.RenderSVG(t.Context(), []byte("graph TD; A-->B;"))
		require.NoError(t, err)
		assert.NotContains(t, string(svg), "container")
		assert.Regexp(t, `id="mermaid-[0-9a-f]{12}"`, string(svg))
		assert.Regexp(t, `#mermaid-[0-9a-f]{12}\{fill:red\}`, string(svg))
	})

	t.Run("caches by source hash", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			io.WriteString(w, `<svg id="container"></svg>`)
		}))
		t.Cleanup(srv.Close)

		k := content.NewKrokiMermaid(srv.URL)
		_, err := k.RenderSVG(t.Context(), []byte("graph TD; A-->B;"))
		require.NoError(t, err)
		_, err = k.RenderSVG(t.Context(), []byte("graph TD; A-->B;"))
		require.NoError(t, err)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("non-200 surfaces the sidecar error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":{"message":"No diagram type detected"}}`, http.StatusBadRequest)
		}))
		t.Cleanup(srv.Close)

		k := content.NewKrokiMermaid(srv.URL)
		_, err := k.RenderSVG(t.Context(), []byte("not a diagram"))
		require.ErrorContains(t, err, "status 400")
		require.ErrorContains(t, err, "No diagram type detected")
	})

	t.Run("ready reflects reachability", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		k := content.NewKrokiMermaid(srv.URL)
		require.NoError(t, k.Ready(t.Context()))
		srv.Close()
		require.Error(t, k.Ready(t.Context()))
	})
}
