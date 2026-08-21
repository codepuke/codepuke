package config_test

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codepuke/codepuke/internal/config"
)

func lookupFrom(m map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr string
	}{
		// valid
		{
			name: "minimal env gets defaults",
			env:  map[string]string{"DATABASE_URL": "postgres://u:p@localhost/db"},
			want: config.Config{
				Addr:        ":8080",
				DatabaseURL: "postgres://u:p@localhost/db",
				LogLevel:    slog.LevelInfo,
				LogFormat:   "text",
				BaseURL:     "https://codepuke.com",
			},
		},
		{
			name: "every value set",
			env: map[string]string{
				"ADDR":         ":9999",
				"DATABASE_URL": "postgres://u:p@localhost/db",
				"LOG_LEVEL":    "debug",
				"LOG_FORMAT":   "json",
				"BASE_URL":     "https://dev.codepuke.com/",
			},
			want: config.Config{
				Addr:        ":9999",
				DatabaseURL: "postgres://u:p@localhost/db",
				LogLevel:    slog.LevelDebug,
				LogFormat:   "json",
				BaseURL:     "https://dev.codepuke.com",
			},
		},

		// invalid
		{
			name:    "missing DATABASE_URL",
			env:     map[string]string{},
			wantErr: "DATABASE_URL is required",
		},
		{
			name: "bad LOG_LEVEL",
			env: map[string]string{
				"DATABASE_URL": "postgres://u:p@localhost/db",
				"LOG_LEVEL":    "shouty",
			},
			wantErr: "LOG_LEVEL",
		},
		{
			name: "bad LOG_FORMAT",
			env: map[string]string{
				"DATABASE_URL": "postgres://u:p@localhost/db",
				"LOG_FORMAT":   "xml",
			},
			wantErr: "LOG_FORMAT",
		},

		// edge
		{
			name: "empty values fall back to defaults",
			env: map[string]string{
				"ADDR":         "",
				"DATABASE_URL": "postgres://u:p@localhost/db",
				"LOG_LEVEL":    "",
				"LOG_FORMAT":   "",
			},
			want: config.Config{
				Addr:        ":8080",
				DatabaseURL: "postgres://u:p@localhost/db",
				LogLevel:    slog.LevelInfo,
				LogFormat:   "text",
				BaseURL:     "https://codepuke.com",
			},
		},
		{
			name: "log level is case-insensitive",
			env: map[string]string{
				"DATABASE_URL": "postgres://u:p@localhost/db",
				"LOG_LEVEL":    "WARN",
			},
			want: config.Config{
				Addr:        ":8080",
				DatabaseURL: "postgres://u:p@localhost/db",
				LogLevel:    slog.LevelWarn,
				LogFormat:   "text",
				BaseURL:     "https://codepuke.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.Parse(lookupFrom(tt.env))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogger(t *testing.T) {
	t.Parallel()

	t.Run("json format emits json", func(t *testing.T) {
		t.Parallel()

		var buf strings.Builder
		cfg := config.Config{LogLevel: slog.LevelInfo, LogFormat: "json"}
		cfg.Logger(&buf).Info("hello")
		assert.True(t, strings.HasPrefix(buf.String(), "{"), "got %q", buf.String())
	})

	t.Run("text format emits logfmt", func(t *testing.T) {
		t.Parallel()

		var buf strings.Builder
		cfg := config.Config{LogLevel: slog.LevelInfo, LogFormat: "text"}
		cfg.Logger(&buf).Info("hello")
		assert.Contains(t, buf.String(), "msg=hello")
	})

	t.Run("level filters records", func(t *testing.T) {
		t.Parallel()

		var buf strings.Builder
		cfg := config.Config{LogLevel: slog.LevelWarn, LogFormat: "text"}
		cfg.Logger(&buf).Info("hidden")
		assert.Empty(t, buf.String())
	})
}
