// Package config parses server configuration from the environment once at
// startup. Nothing else in the codebase reads environment variables.
package config

import (
	"cmp"
	"encoding/hex"
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
	// MermaidURL is the kroki-mermaid sidecar base URL, from MERMAID_URL.
	// Empty means no renderer: diagrams stay highlighted code blocks.
	MermaidURL string
	// SessionKey seals the admin session cookie, hex-decoded from
	// SESSION_KEY. Empty (with the OIDC vars also unset) disables /admin.
	SessionKey []byte
	// OIDCIssuer, OIDCClientID, and OIDCClientSecret configure the
	// Authentik OIDC client, from OIDC_ISSUER, OIDC_CLIENT_ID, and
	// OIDC_CLIENT_SECRET.
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	// OIDCGroup is the Authentik group whose members may author, from
	// OIDC_GROUP.
	OIDCGroup string
}

// AuthEnabled reports whether the admin surface is configured. The four auth
// variables are all-or-none; Parse rejects a partial set.
func (c Config) AuthEnabled() bool { return len(c.SessionKey) > 0 }

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
		MermaidURL:  env("MERMAID_URL"),
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

	cfg.OIDCIssuer = env("OIDC_ISSUER")
	cfg.OIDCClientID = env("OIDC_CLIENT_ID")
	cfg.OIDCClientSecret = env("OIDC_CLIENT_SECRET")

	sessionKey := env("SESSION_KEY")
	authVars := []string{sessionKey, cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret}
	set := 0
	for _, v := range authVars {
		if v != "" {
			set++
		}
	}
	switch set {
	case 0: // auth disabled, /admin is a 404
	case len(authVars):
		key, err := hex.DecodeString(sessionKey)
		if err != nil {
			return Config{}, fmt.Errorf("SESSION_KEY: %w", err)
		}
		if len(key) != 32 {
			return Config{}, fmt.Errorf("SESSION_KEY must be 32 hex-encoded bytes, got %d", len(key))
		}
		cfg.SessionKey = key
		cfg.OIDCGroup = cmp.Or(env("OIDC_GROUP"), "codepuke-authors")
	default:
		return Config{}, fmt.Errorf("SESSION_KEY, OIDC_ISSUER, OIDC_CLIENT_ID, and OIDC_CLIENT_SECRET must be set together")
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
