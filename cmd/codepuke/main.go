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
	default:
		return fmt.Errorf("unknown command %q (want: serve)", cmd)
	}
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

	handler, err := web.New(web.Deps{Store: st, Content: contentfs.FS, BaseURL: cfg.BaseURL})
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
