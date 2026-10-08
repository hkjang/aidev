package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/config"
	"github.com/hkjang/umm/internal/cryptoutil"
	"github.com/hkjang/umm/internal/dream"
	"github.com/hkjang/umm/internal/httpapi"
	"github.com/hkjang/umm/internal/observability"
	"github.com/hkjang/umm/internal/realtime"
	"github.com/hkjang/umm/internal/store"
	"github.com/hkjang/umm/internal/webhook"
)

var version = "dev"

const httpWriteTimeoutBuffer = time.Minute

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: time.Duration(dream.MaxGatewayTimeoutSeconds)*time.Second + httpWriteTimeoutBuffer,
		IdleTimeout:  90 * time.Second, MaxHeaderBytes: 1 << 20,
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid startup configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	shutdownTelemetry, err := observability.Init(ctx, version)
	if err != nil {
		slog.Error("OpenTelemetry initialization failed", "error", err)
		os.Exit(1)
	}
	db, err := store.Open(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer db.Pool.Close()
	if err = db.Migrate(ctx); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	if err = db.BootstrapAdmin(ctx, cfg.BootstrapAdmin, cfg.BootstrapAdminPassword); err != nil {
		slog.Error("bootstrap admin failed", "error", err)
		os.Exit(1)
	}
	cipher, err := cryptoutil.NewWithPrevious(cfg.EncryptionKey, cfg.EncryptionPreviousKeys...)
	if err != nil {
		slog.Error("encryption initialization failed", "error", err)
		os.Exit(1)
	}
	db.Cipher = cipher
	authService := &auth.Service{Store: db}
	oidcService := &auth.OIDCService{Store: db, Cipher: cipher, Sessions: authService}
	dreamService := &dream.Service{Store: db, Cipher: cipher, Version: version}
	dreamService.Start(ctx)
	defer dreamService.Stop()
	webhookService := webhook.New(db, cipher)
	webhookService.Start(ctx)
	events := realtime.New(db.Pool)
	go events.Run(ctx)
	db.StartJanitor(ctx)
	// Indexing runs here rather than on whoever opens a space next. Started
	// even on an installation nobody has opened yet, so an import is searchable
	// without somebody having to go and look at it first.
	db.StartEmbeddingSweep(ctx)
	webDir := "web/dist"
	if _, err := os.Stat("/app/web/index.html"); err == nil {
		webDir = "/app/web"
	}
	metrics := observability.NewRegistry()
	api := &httpapi.Server{Store: db, Auth: authService, OIDC: oidcService, Cipher: cipher, Dreams: dreamService, Webhooks: webhookService, Metrics: metrics, Events: events, Version: version, WebDir: webDir, TrustedProxies: cfg.TrustedProxyCIDRs}
	httpServer := newHTTPServer(cfg.HTTPAddr, api.Handler())
	go func() {
		slog.Info("umm started", "version", version, "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			cancel()
		}
	}()
	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	telemetryCtx, telemetryCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer telemetryCancel()
	if err := shutdownTelemetry(telemetryCtx); err != nil {
		slog.Warn("OpenTelemetry shutdown failed", "error", err)
	}
}
