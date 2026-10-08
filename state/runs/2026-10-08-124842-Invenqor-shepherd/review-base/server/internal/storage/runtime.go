package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hkjang/invenqor/server/internal/durablefs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Mode string

const (
	ModeSQLiteFallback   Mode = "SQLITE_FALLBACK"
	ModePostgresActive   Mode = "POSTGRES_ACTIVE"
	ModePostgresDegraded Mode = "POSTGRES_DEGRADED"
)

type FailureCode string

const (
	FailureInvalidDSN      FailureCode = "INVALID_DSN"
	FailureDNS             FailureCode = "DNS_FAILURE"
	FailureConnection      FailureCode = "CONNECTION_FAILURE"
	FailureAuthentication  FailureCode = "AUTHENTICATION_FAILURE"
	FailurePermission      FailureCode = "PERMISSION_FAILURE"
	FailureSchemaMigration FailureCode = "SCHEMA_MIGRATION_FAILURE"
	FailureUnknown         FailureCode = "UNKNOWN_FAILURE"
)

type ConnectionFailure struct {
	Code      FailureCode `json:"code"`
	Summary   string      `json:"summary"`
	Host      string      `json:"host,omitempty"`
	CheckedAt time.Time   `json:"checked_at"`
	// Detail is the underlying cause, for the operator's log only. It is never
	// serialised, because this type is returned by the health endpoints and the
	// classified fields above are deliberately credential-free. Set it only
	// where the cause cannot carry a connection string - a failing migration
	// statement can, a connection error cannot.
	Detail string `json:"-"`
}

// migrationTimeoutFactor is how much longer a migration may take than a
// connection attempt. Both are still bounded, so a migration that is genuinely
// stuck still fails rather than hanging the start-up forever.
const migrationTimeoutFactor = 12

type Options struct {
	PostgresDSN string
	SQLitePath  string
	Schema      string
	Timeout     time.Duration
}

type Runtime struct {
	db              *sql.DB
	modeMu          sync.RWMutex
	mode            Mode
	postgresPrimary bool
	sqlitePath      string
	postgresFailure *ConnectionFailure
	openedAt        time.Time
}

func Open(ctx context.Context, options Options) (*Runtime, error) {
	if options.Timeout <= 0 {
		options.Timeout = 5 * time.Second
	}
	if options.Schema == "" {
		options.Schema = "public"
	}
	if strings.TrimSpace(options.PostgresDSN) != "" {
		database, postgresFailure := openPostgres(ctx, options)
		if postgresFailure == nil {
			return &Runtime{
				db:              database,
				mode:            ModePostgresActive,
				postgresPrimary: true,
				openedAt:        time.Now().UTC(),
			}, nil
		}
		// An explicitly configured PostgreSQL database is the operator's source
		// of truth. Falling through to a fresh Pod-local SQLite database makes a
		// broken production Pod look ready while serving an unrelated data set.
		// Return only the classified, credential-free failure metadata.
		if postgresFailure.Detail != "" {
			return nil, fmt.Errorf(
				"start PostgreSQL: %s (%s): %s",
				postgresFailure.Summary,
				postgresFailure.Code,
				postgresFailure.Detail,
			)
		}
		return nil, fmt.Errorf(
			"start PostgreSQL: %s (%s)",
			postgresFailure.Summary,
			postgresFailure.Code,
		)
	}
	database, err := openSQLite(ctx, options.SQLitePath)
	if err != nil {
		return nil, fmt.Errorf("start SQLite fallback database: %w", err)
	}
	return &Runtime{
		db:         database,
		mode:       ModeSQLiteFallback,
		sqlitePath: options.SQLitePath,
		openedAt:   time.Now().UTC(),
	}, nil
}

func (r *Runtime) DB() *sql.DB {
	return r.db
}

func (r *Runtime) Mode() Mode {
	r.modeMu.RLock()
	defer r.modeMu.RUnlock()
	return r.mode
}

func (r *Runtime) MarkPostgresDegraded() {
	if !r.postgresPrimary {
		return
	}
	r.modeMu.Lock()
	r.mode = ModePostgresDegraded
	r.modeMu.Unlock()
}

func (r *Runtime) MarkPostgresActive() {
	if !r.postgresPrimary {
		return
	}
	r.modeMu.Lock()
	r.mode = ModePostgresActive
	r.modeMu.Unlock()
}

func (r *Runtime) SQLitePath() string {
	return r.sqlitePath
}

func (r *Runtime) PostgresFailure() *ConnectionFailure {
	if r.postgresFailure == nil {
		return nil
	}
	copy := *r.postgresFailure
	return &copy
}

func (r *Runtime) OpenedAt() time.Time {
	return r.openedAt
}

func (r *Runtime) Ping(ctx context.Context) error {
	err := r.db.PingContext(ctx)
	if err != nil {
		r.MarkPostgresDegraded()
	} else {
		r.MarkPostgresActive()
	}
	return err
}

func (r *Runtime) Close() error {
	return r.db.Close()
}

func openPostgres(ctx context.Context, options Options) (*sql.DB, *ConnectionFailure) {
	database, connectionFailure := connectPostgres(ctx, options)
	if connectionFailure != nil {
		return nil, connectionFailure
	}
	// Migrating is not the same job as connecting, and it does not deserve the
	// same deadline. Connecting either answers quickly or is unreachable; the
	// first migration creates every table and index this product has, against a
	// database that may be serving something else, and taking longer than five
	// seconds there is ordinary rather than a fault. Sharing one timeout meant a
	// slow first migration was reported as a failure and the Server refused to
	// start - a crash loop over a database that was only busy.
	migrationTimeout := options.Timeout * migrationTimeoutFactor
	if migrationTimeout < time.Minute {
		migrationTimeout = time.Minute
	}
	timeoutContext, cancel := context.WithTimeout(ctx, migrationTimeout)
	defer cancel()
	if err := applyMigrations(timeoutContext, database, "postgres"); err != nil {
		database.Close()
		classified := failure(
			FailureSchemaMigration,
			"PostgreSQL schema migration failed",
			postgresHost(options.PostgresDSN),
			time.Now(),
		)
		// Without this the operator gets a code and nothing else, and the only
		// way to learn which statement failed is to reproduce it.
		classified.Detail = err.Error()
		return nil, classified
	}
	return database, nil
}

// CheckPostgres validates and connects to PostgreSQL without changing its
// schema. It returns only classified, credential-free failure metadata.
func CheckPostgres(ctx context.Context, options Options) *ConnectionFailure {
	if options.Timeout <= 0 {
		options.Timeout = 5 * time.Second
	}
	database, connectionFailure := connectPostgres(ctx, options)
	if database != nil {
		_ = database.Close()
	}
	return connectionFailure
}

func connectPostgres(
	ctx context.Context,
	options Options,
) (*sql.DB, *ConnectionFailure) {
	config, err := pgx.ParseConfig(options.PostgresDSN)
	if err != nil {
		return nil, failure(FailureInvalidDSN, "PostgreSQL DSN validation failed", "", time.Now())
	}
	host := config.Host
	config.RuntimeParams["search_path"] = options.Schema
	database := stdlib.OpenDB(*config)
	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)
	timeoutContext, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	if err := database.PingContext(timeoutContext); err != nil {
		database.Close()
		return nil, classifyPostgresFailure(err, host)
	}
	return database, nil
}

func postgresHost(dsn string) string {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return ""
	}
	return config.Host
}

func openSQLite(ctx context.Context, path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("SQLite path is required")
	}
	parent := filepath.Dir(path)
	if err := durablefs.EnsurePrivateDirectory(parent); err != nil {
		return nil, fmt.Errorf("secure SQLite directory: %w", err)
	}
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = FULL",
	} {
		if _, err := database.ExecContext(ctx, pragma); err != nil {
			database.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	if err := applyMigrations(ctx, database, "sqlite"); err != nil {
		database.Close()
		return nil, fmt.Errorf("migrate SQLite: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		database.Close()
		return nil, fmt.Errorf("secure SQLite database: %w", err)
	}
	return database, nil
}

func classifyPostgresFailure(err error, host string) *ConnectionFailure {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		switch pgError.Code {
		case "28P01", "28000":
			return failure(
				FailureAuthentication,
				"PostgreSQL authentication failed",
				host,
				time.Now(),
			)
		case "42501":
			return failure(
				FailurePermission,
				"PostgreSQL permission check failed",
				host,
				time.Now(),
			)
		}
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return failure(FailureDNS, "PostgreSQL host lookup failed", host, time.Now())
	}
	var netError net.Error
	if errors.As(err, &netError) {
		return failure(
			FailureConnection,
			"PostgreSQL connection failed",
			host,
			time.Now(),
		)
	}
	return failure(FailureUnknown, "PostgreSQL startup check failed", host, time.Now())
}

func failure(code FailureCode, summary, host string, checkedAt time.Time) *ConnectionFailure {
	return &ConnectionFailure{
		Code:      code,
		Summary:   summary,
		Host:      safeHost(host),
		CheckedAt: checkedAt.UTC(),
	}
}

func safeHost(host string) string {
	if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
		host = parsed.Hostname()
	}
	host = strings.TrimSpace(host)
	if len(host) > 255 {
		return host[:255]
	}
	return host
}
