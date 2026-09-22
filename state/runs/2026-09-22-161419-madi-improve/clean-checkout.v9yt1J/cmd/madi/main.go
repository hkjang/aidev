package main

import (
	"context"
	"encoding/base64"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hkjang/madi/internal/server"
	"github.com/hkjang/madi/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

var version = "0.3.0"

func main() {
	if handled, code := server.RunAttachmentWorker(os.Args[1:]); handled {
		os.Exit(code)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for _, name := range []string{"POSTGRES_DSN", "BOOTSTRAP_ADMIN", "BOOTSTRAP_ADMIN_PASSWORD", "ENCRYPTION_KEY"} {
		if os.Getenv(name) == "" {
			slog.Error("필수 환경변수가 없습니다", "name", name)
			os.Exit(1)
		}
	}
	key, e := base64.StdEncoding.DecodeString(os.Getenv("ENCRYPTION_KEY"))
	if e != nil || len(key) != 32 {
		slog.Error("ENCRYPTION_KEY는 base64로 인코딩한 32바이트 키가 필요합니다")
		os.Exit(1)
	}
	cfg, e := pgxpool.ParseConfig(os.Getenv("POSTGRES_DSN"))
	if e != nil {
		slog.Error("POSTGRES_DSN 형식이 올바르지 않습니다")
		os.Exit(1)
	}
	cfg.MaxConns = 20
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		slog.Error("PostgreSQL 연결을 준비하지 못했습니다")
		os.Exit(1)
	}
	defer db.Close()
	assets, _ := fs.Sub(web.Assets, "dist")
	app, e := server.New(ctx, db, key, version, os.Getenv("BOOTSTRAP_ADMIN"), os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"), assets)
	if e != nil {
		slog.Error("서비스 초기화 실패", "error", e)
		os.Exit(1)
	}
	app.StartMaintenance(ctx)
	app.StartJobs(ctx)
	app.StartBackupSchedule(ctx)
	app.StartKnowledgeLifecycle(ctx)
	app.StartImports(ctx)
	app.StartConnectors(ctx)
	app.StartNotificationDelivery(ctx)
	app.StartSearchIndex(ctx)
	app.StartRAGIndex(ctx)
	app.StartInboundCaptures(ctx)
	app.StartRunbookMaintenance(ctx)
	app.StartGitSync(ctx)
	app.StartOperations(ctx)
	app.StartExports(ctx)
	app.StartAttachmentExtraction(ctx)
	srv := &http.Server{Addr: ":8080", Handler: app, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		slog.Info("madi 시작", "version", version, "address", srv.Addr)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			slog.Error("서버 실행 실패", "error", e)
			stop()
		}
	}()
	<-ctx.Done()
	app.CloseCollaboration()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if e = srv.Shutdown(shutdown); e != nil {
		slog.Error("서버 종료 제한 시간 초과", "error", e)
	}
}
