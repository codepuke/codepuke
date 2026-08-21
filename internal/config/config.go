// Package config parses server configuration from the environment once at
// startup. Nothing else in the codebase reads environment variables.
package config

import (
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Config is the complete runtime configuration of the server.
type Config struct {
	// Addr is the listen address, from ADDR.
	Addr string
	// DatabaseURL is the pgx connection string, from DATABASE_URL. Required.
	DatabaseURL string
	// LogLevel is the minimum slog level, from LOG_LEVEL.
	LogLevel slog.Level
	// LogFormat is "text" or "json", from LOG_FORMAT.
	LogFormat string
	// BaseURL is the site's public origin, from BASE_URL; absolute links
	// (the RSS feed) derive from it.
	BaseURL string
}

// Parse builds a Config by reading through lookup, typically os.LookupEnv.
// Invalid values are startup errors, never silent fallbacks.
func Parse(lookup func(string) (string, bool)) (Config, error) {
	env := func(key string) string {
		v, _ := lookup(key)
		return v
	}

	cfg := Config{
		Addr:        cmp.Or(env("ADDR"), ":8080"),
		DatabaseURL: env("DATABASE_URL"),
		LogFormat:   cmp.Or(env("LOG_FORMAT"), "text"),
		BaseURL:     strings.TrimRight(cmp.Or(env("BASE_URL"), "https://codepuke.com"), "/"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(cmp.Or(env("LOG_LEVEL"), "info"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
	}

	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		return Config{}, fmt.Errorf("LOG_FORMAT must be text or json, got %q", cfg.LogFormat)
	}

	return cfg, nil
}

// Logger builds the slog logger the config describes, writing to w.
func (c Config) Logger(w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: c.LogLevel}
	if c.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
