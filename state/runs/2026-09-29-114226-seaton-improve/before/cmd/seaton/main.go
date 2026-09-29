package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hkjang/seaton/internal/app"
	"github.com/hkjang/seaton/internal/database"
	"github.com/hkjang/seaton/internal/platform"
)

var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

//go:embed webdist
var embeddedWeb embed.FS

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get("http://127.0.0.1:8080/healthz")
		if err != nil || response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		_ = response.Body.Close()
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := platform.LoadConfig()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	db, err := database.OpenWithRetry(ctx, cfg.PostgresDSN, 2*time.Minute)
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	keyring, err := platform.NewKeyring("/var/lib/seaton/master.key")
	if err != nil {
		logger.Error("keyring startup failed", "error", err)
		os.Exit(1)
	}
	webFS, err := fs.Sub(embeddedWeb, "webdist")
	if err != nil {
		logger.Error("embedded ui failed", "error", err)
		os.Exit(1)
	}
	service := app.NewServer(db, keyring, logger, webFS, version, commit, builtAt)
	if err := service.EnsureBootstrapAdmin(ctx, cfg.BootstrapAdmin, cfg.BootstrapAdminPassword); err != nil {
		logger.Error("bootstrap administrator failed", "error", err)
		os.Exit(1)
	}
	go service.RunBackground(ctx)
	httpServer := &http.Server{Addr: ":8080", Handler: service.Routes(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		logger.Info("SeatOn started", "address", httpServer.Addr, "version", version, "commit", commit)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped", "error", err)
			cancel()
		}
	}()
	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
