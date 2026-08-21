package web_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/codepuke/codepuke/internal/web"
)

type fakePinger struct {
	err error
}

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestProbes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		ping       error
		wantStatus int
	}{
		// valid
		{"healthz ok", http.MethodGet, "/healthz", nil, http.StatusOK},
		{"readyz ok", http.MethodGet, "/readyz", nil, http.StatusOK},

		// invalid
		{"readyz db down", http.MethodGet, "/readyz", errors.New("no db"), http.StatusServiceUnavailable},
		{"healthz wrong method", http.MethodPost, "/healthz", nil, http.StatusMethodNotAllowed},

		// edge
		{"healthz ignores db state", http.MethodGet, "/healthz", errors.New("no db"), http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			web.Probes(mux, fakePinger{err: tt.ping})
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
