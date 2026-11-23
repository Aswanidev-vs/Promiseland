// Command server starts the PromisedLand API.
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

	"github.com/Aswanidev-vs/promiseland/backend/internal/api"
	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const shutdownTimeout = 15 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(1)
	}

	logger := newLogger(cfg).With("app", "promisedland", "env", cfg.AppEnv)
	// Middleware without access to Deps (for example Recoverer) logs through
	// slog.Default(), so point it at the configured handler.
	slog.SetDefault(logger)
	if cfg.IsProduction() && cfg.UsesDevSessionSecret() {
		logger.Warn("SESSION_SECRET is still at the development default; set a strong secret")
	}

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("failed to open database", "error", err, "path", cfg.DatabasePath)
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()

	if err := db.Migrate(database, logger); err != nil {
		logger.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	deps := &shared.Deps{DB: database, Log: logger, Config: cfg}
	server := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           api.New(deps),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serveErrors := make(chan error, 1)
	go func() {
		logger.Info("promisedland api listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrors <- err
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case sig := <-signals:
		logger.Info("shutdown requested", "signal", sig.String())
	case err := <-serveErrors:
		logger.Error("http server stopped unexpectedly", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed, forcing close", "error", err)
		_ = server.Close()
	}
	logger.Info("shutdown complete")
}

func newLogger(cfg config.Config) *slog.Logger {
	if cfg.IsProduction() {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
