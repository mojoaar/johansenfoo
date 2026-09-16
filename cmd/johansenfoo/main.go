package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/web"
)

var version = "0.1.0"

func main() {
	dataDir := flag.String("data", "./data", "data directory")
	flag.Parse()

	if err := run(*dataDir); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func buildHandler(dataDir string) (http.Handler, *config.Config, func(), error) {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Migrate(d); err != nil {
		_ = d.Close()
		return nil, nil, nil, fmt.Errorf("migrate: %w", err)
	}

	store, err := web.NewContentStore(d)
	if err != nil {
		_ = d.Close()
		return nil, nil, nil, fmt.Errorf("load content: %w", err)
	}

	handler := web.New(web.Deps{
		DB:      d,
		Cfg:     cfg,
		Content: store,
		Version: version,
		Started: time.Now(),
	})
	return handler, cfg, func() { _ = d.Close() }, nil
}

func run(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	handler, cfg, cleanup, err := buildHandler(dataDir)
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
