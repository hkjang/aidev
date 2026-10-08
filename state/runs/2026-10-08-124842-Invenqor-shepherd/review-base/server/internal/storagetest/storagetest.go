// Package storagetest opens the store that tests run against.
//
// SQLite by default, because it needs nothing installed. Set
// INVENQOR_TEST_POSTGRES_DSN - or run scripts/test-postgres.sh - to run the same
// tests against a real PostgreSQL.
//
// That switch is not a nicety. The engines disagree in ways that let a
// SQLite-only suite report success over a broken deployment, and twice did:
// JSONB has no text operators, so a duplicate-detection query that worked under
// SQLite failed on every ingest with SQLSTATE 42883; and PostgreSQL deduces a
// parameter's type from every place it appears, so an UPDATE that reused one
// parameter for two columns made every relation approval return HTTP 500.
// SQLite also ignores ASCII case in LIKE where PostgreSQL does not.
//
// It lives in its own package rather than in a _test.go file so that every
// package's tests share one implementation instead of eight copies that drift.
package storagetest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/hkjang/invenqor/server/internal/storage"
)

// DSNVariable names the environment variable that switches engines.
const DSNVariable = "INVENQOR_TEST_POSTGRES_DSN"

var schemaCounter atomic.Uint64

// Open returns a migrated store, closed when the test ends.
//
// Against PostgreSQL each call gets its own schema, so tests calling t.Parallel
// do not collide and a failure leaves its rows behind for inspection.
func Open(t *testing.T) *storage.Runtime {
	t.Helper()
	options := storage.Options{SQLitePath: filepath.Join(t.TempDir(), "invenqor.db")}
	if dsn := os.Getenv(DSNVariable); dsn != "" {
		schema := schemaName(t)
		createSchema(t, dsn, schema)
		options = storage.Options{PostgresDSN: dsn, Schema: schema}
	}
	runtime, err := storage.Open(context.Background(), options)
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	return runtime
}

// OpenSQLite forces the SQLite fallback regardless of DSNVariable, for the few
// tests whose subject is the fallback itself or which use SQLite-only SQL.
// Prefer Open: a test pinned here is a test PostgreSQL never checks.
func OpenSQLite(t *testing.T) *storage.Runtime {
	t.Helper()
	runtime, err := storage.Open(context.Background(), storage.Options{
		SQLitePath: filepath.Join(t.TempDir(), "invenqor.db"),
	})
	if err != nil {
		t.Fatalf("storage.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	return runtime
}

// Postgres reports whether the suite is running against PostgreSQL, for tests
// that must issue dialect-specific SQL of their own.
func Postgres() bool { return os.Getenv(DSNVariable) != "" }

var stateDirs sync.Map

// StateDir is where this test's master.key, sealed bootstrap values and initial
// admin token live.
//
// In SQLite mode that is the directory holding the database file. A PostgreSQL
// runtime has no SQLite path, and filepath.Dir("") is ".", so deriving it that
// way puts key material in the package directory and gives every test one shared
// bootstrap store - a secret written by one test then answers another test's
// lookup. Memoised so every caller for a given runtime gets the same directory.
// Production takes this from INVENQOR_STATE_DIR and never derives it.
func StateDir(t *testing.T, runtime *storage.Runtime) string {
	t.Helper()
	if path := runtime.SQLitePath(); path != "" {
		return filepath.Dir(path)
	}
	if existing, ok := stateDirs.Load(runtime); ok {
		return existing.(string)
	}
	actual, _ := stateDirs.LoadOrStore(runtime, t.TempDir())
	return actual.(string)
}

// createSchema makes the schema the runtime migrates into. The server sets
// search_path but does not create the schema, mirroring what an operator does
// once before pointing the server at a non-public schema.
func createSchema(t *testing.T, dsn, schema string) {
	t.Helper()
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer database.Close()
	if _, err := database.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`); err != nil {
		t.Fatalf("drop test schema: %v", err)
	}
	if _, err := database.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
}

// schemaName derives a PostgreSQL identifier from the test name.
func schemaName(t *testing.T) string {
	t.Helper()
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '_'
		}
	}, t.Name())
	if len(safe) > 40 {
		safe = safe[:40]
	}
	return fmt.Sprintf("t_%s_%d", safe, schemaCounter.Add(1))
}
