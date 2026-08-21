// Command codepuke runs the codepuke.com server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	contentfs "github.com/codepuke/codepuke/content"
	"github.com/codepuke/codepuke/internal/auth"
	"github.com/codepuke/codepuke/internal/config"
	"github.com/codepuke/codepuke/internal/store"
	"github.com/codepuke/codepuke/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "serve":
		return serve()
	case "sync-db":
		return syncDBCmd()
	default:
		return fmt.Errorf("unknown command %q (want: serve, sync-db)", cmd)
	}
}

// syncDBCmd runs one standalone docs sync against the configured database.
func syncDBCmd() error {
	cfg, err := config.Parse(os.LookupEnv)
	if err != nil {
		return err
	}
	slog.SetDefault(cfg.Logger(os.Stdout))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := store.RunMigrations(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	pipeline, err := buildPipeline(ctx, cfg)
	if err != nil {
		return err
	}
	return syncDB(ctx, st, pipeline)
}

func serve() error {
	cfg, err := config.Parse(os.LookupEnv)
	if err != nil {
		return err
	}
	slog.SetDefault(cfg.Logger(os.Stdout))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := store.RunMigrations(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	slog.Info("migrations up to date")

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	pipeline, err := buildPipeline(ctx, cfg)
	if err != nil {
		return err
	}
	if err := syncDB(ctx, st, pipeline); err != nil {
		return err
	}

	deps := web.Deps{Store: st, Content: contentfs.FS, BaseURL: cfg.BaseURL}
	if cfg.AuthEnabled() {
		deps.Auth, err = auth.New(auth.Options{
			Issuer:       cfg.OIDCIssuer,
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			BaseURL:      cfg.BaseURL,
			Group:        cfg.OIDCGroup,
			SessionKey:   cfg.SessionKey,
		})
		if err != nil {
			return err
		}
		deps.Renderer = pipeline
		slog.Info("admin enabled", "issuer", cfg.OIDCIssuer, "group", cfg.OIDCGroup)
	} else {
		slog.Info("admin disabled: auth is not configured")
	}
	handler, err := web.New(deps)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
