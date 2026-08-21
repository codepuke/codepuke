package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// KrokiMermaid renders diagrams via the kroki-mermaid sidecar's HTTP API
// (POST <base>/svg with the diagram source as the body). Results are cached
// by source hash for the lifetime of the client, which spans one sync run.
// Sidecar output is trusted and inlined without sanitization, per the
// pipeline's contract.
type KrokiMermaid struct {
	base   string
	client *http.Client

	mu    sync.Mutex
	cache map[[32]byte][]byte
}

// NewKrokiMermaid points at the sidecar, e.g. http://localhost:8002.
func NewKrokiMermaid(baseURL string) *KrokiMermaid {
	return &KrokiMermaid{
		base:   strings.TrimRight(baseURL, "/"),
		client: &http.Client{Timeout: 30 * time.Second},
		cache:  map[[32]byte][]byte{},
	}
}

// Ready reports whether the sidecar answers HTTP yet; boot polls it before
// rendering so a pod's containers can start in any order.
func (k *KrokiMermaid) Ready(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.base+"/", nil)
	if err != nil {
		return err
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// RenderSVG implements MermaidRenderer.
func (k *KrokiMermaid) RenderSVG(ctx context.Context, source []byte) ([]byte, error) {
	sum := sha256.Sum256(source)
	k.mu.Lock()
	cached, ok := k.cache[sum]
	k.mu.Unlock()
	if ok {
		return cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.base+"/svg", bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mermaid sidecar: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("mermaid sidecar: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mermaid sidecar: status %d: %s", resp.StatusCode, truncate(body, 300))
	}

	// Every kroki-mermaid SVG arrives as id="container" with #container-scoped
	// styles; two diagrams on one page would collide, so scope the id to the
	// source hash.
	id := "mermaid-" + hex.EncodeToString(sum[:6])
	svg := bytes.ReplaceAll(body, []byte(`id="container"`), []byte(`id="`+id+`"`))
	svg = bytes.ReplaceAll(svg, []byte("#container"), []byte("#"+id))

	k.mu.Lock()
	k.cache[sum] = svg
	k.mu.Unlock()
	return svg, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
