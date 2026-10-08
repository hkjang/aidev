package storage

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/migrations"
)

func TestOpenWithoutPostgresUsesSQLiteFallback(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "invenqor.db")
	runtime, err := Open(context.Background(), Options{
		SQLitePath: path,
		Timeout:    100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer runtime.Close()
	if runtime.Mode() != ModeSQLiteFallback {
		t.Fatalf("Mode() = %s, want %s", runtime.Mode(), ModeSQLiteFallback)
	}
	if runtime.PostgresFailure() != nil {
		t.Fatalf("PostgresFailure() = %#v, want nil", runtime.PostgresFailure())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("SQLite database was not created: %v", err)
	}
	assertTableExists(t, runtime, "assets")
	assertTableExists(t, runtime, "audit_logs")
	assertTableExists(t, runtime, "db_migration_jobs")
	assertTableExists(t, runtime, "software_product_inventory")
	assertTableExists(t, runtime, "software_catalog_reconciliations")
}

func TestMalformedPostgresDSNFailsClosedWithoutLeakingSecret(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "invenqor.db")
	secret := "never-log-this-password"
	_, err := Open(context.Background(), Options{
		PostgresDSN: "postgres://user:" + secret + "@%zz/invenqor",
		SQLitePath:  path,
		Timeout:     100 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), string(FailureInvalidDSN)) {
		t.Fatalf("Open() error = %v, want INVALID_DSN", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("PostgreSQL startup error leaked the password")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("explicit PostgreSQL failure created SQLite database: %v", statErr)
	}
}

func TestUnavailablePostgresFailsClosedWithoutCreatingSQLite(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "invenqor.db")
	_, err := Open(context.Background(), Options{
		PostgresDSN: "postgres://user:secret@127.0.0.1:1/invenqor?sslmode=disable",
		SQLitePath:  path,
		Timeout:     100 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), string(FailureConnection)) {
		t.Fatalf("Open() error = %v, want CONNECTION_FAILURE", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("explicit PostgreSQL failure created SQLite database: %v", statErr)
	}
}

func TestCheckPostgresRejectsMalformedDSNWithoutLeakingPassword(t *testing.T) {
	secret := "check-do-not-expose"
	failure := CheckPostgres(context.Background(), Options{
		PostgresDSN: "postgres://user:" + secret + "@%zz/invenqor",
		Timeout:     time.Second,
	})
	if failure == nil || failure.Code != FailureInvalidDSN {
		t.Fatalf("CheckPostgres() failure = %#v, want INVALID_DSN", failure)
	}
	if strings.Contains(failure.Summary, secret) ||
		strings.Contains(failure.Host, secret) {
		t.Fatal("CheckPostgres() failure leaked the PostgreSQL password")
	}
}

func TestSQLiteCreationFailureFailsStartup(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	parentFile := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := Open(context.Background(), Options{
		SQLitePath: filepath.Join(parentFile, "invenqor.db"),
		Timeout:    100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Open() succeeded with an invalid SQLite path")
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "invenqor.db")
	first, err := Open(context.Background(), Options{SQLitePath: path})
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	second, err := Open(context.Background(), Options{SQLitePath: path})
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	defer second.Close()
	var versions int
	if err := second.DB().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&versions); err != nil {
		t.Fatalf("query schema_migrations error = %v", err)
	}
	// Counting a literal here means the number has to be updated with each
	// migration, which is the point: an accidentally re-runnable migration would
	// otherwise inflate this silently.
	expected := migrationFileCount(t, "sqlite")
	if versions != expected {
		t.Fatalf("migration versions = %d, want %d", versions, expected)
	}
	assertTableExists(t, second, "diagnostic_logs")
	assertTableExists(t, second, "asset_classification_rules")
	assertTableExists(t, second, "software_product_inventory")
	assertTableExists(t, second, "software_catalog_reconciliations")
}

func TestSoftwareProductProjectionMigrationBackfillsExistingCatalogAssets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invenqor.db")
	runtime, err := Open(context.Background(), Options{SQLitePath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	database := runtime.DB()
	if _, err := database.Exec(`DROP TABLE software_product_inventory`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`DROP TABLE software_catalog_reconciliations`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`DELETE FROM schema_migrations WHERE version = 6`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO agents(id,agent_id,hostname,status)
		 VALUES('agent-existing','external-existing','db-existing','active')`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO assets(
			id,asset_key,name,type,status,confidence,attributes_json,
			classification_source
		 ) VALUES(
			'host-existing','host-existing','db-existing','host','active',1,'{}','agent'
		 ),(
			'product-existing','product-existing','PostgreSQL','software_product',
			'active',0.94,
			'{"product_key":"postgresql","product_name":"PostgreSQL","role":"database","vendor":"PostgreSQL Global Development Group","version":"16.4","install_state":"installed","runtime_state":"running","process_names":["postgres"],"process_count":3,"evidence_count":3,"catalog_version":"test-catalog"}',
			'software_catalog'
		 )`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO asset_sources(
			id,asset_id,agent_id,category,source_asset_id,source_name,
			payload_json,collected_at
		 ) VALUES(
			'source-existing','product-existing','agent-existing',
			'software.product','postgresql','builtin_catalog','{}',CURRENT_TIMESTAMP
		 )`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO asset_relations(
			id,source_asset_id,relation_type,target_asset_id,source,
			confidence,status
		 ) VALUES(
			'relation-existing','product-existing','runs_on','host-existing',
			'automatic',0.94,'active'
		 )`,
	); err != nil {
		t.Fatal(err)
	}
	if err := applyMigrations(context.Background(), database, "sqlite"); err != nil {
		t.Fatal(err)
	}
	var role, runtimeState, searchText string
	var processCount int
	if err := database.QueryRow(
		`SELECT role,runtime_state,process_count,search_text
		   FROM software_product_inventory
		  WHERE asset_id='product-existing' AND agent_id='agent-existing'`,
	).Scan(&role, &runtimeState, &processCount, &searchText); err != nil {
		t.Fatal(err)
	}
	if role != "database" || runtimeState != "running" || processCount != 3 ||
		!strings.Contains(searchText, "db-existing") ||
		!strings.Contains(searchText, "postgres") {
		t.Fatalf("backfilled projection = %q/%q/%d/%q", role, runtimeState, processCount, searchText)
	}
}

// migrationFileCount counts the shipped migrations for a dialect so the
// idempotency check stays correct as the schema grows.
func migrationFileCount(t *testing.T, dialect string) int {
	t.Helper()
	entries, err := fs.ReadDir(migrations.Files, dialect)
	if err != nil {
		t.Fatalf("list %s migrations: %v", dialect, err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			count++
		}
	}
	return count
}

func assertTableExists(t *testing.T, runtime *Runtime, name string) {
	t.Helper()
	var count int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		name,
	).Scan(&count); err != nil {
		t.Fatalf("query table %s error = %v", name, err)
	}
	if count != 1 {
		t.Fatalf("table %s does not exist", name)
	}
}
