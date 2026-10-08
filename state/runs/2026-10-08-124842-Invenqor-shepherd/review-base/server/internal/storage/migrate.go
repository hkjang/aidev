package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/hkjang/invenqor/server/migrations"
)

func applyMigrations(ctx context.Context, db *sql.DB, dialect string) error {
	if dialect != "sqlite" && dialect != "postgres" {
		return fmt.Errorf("unsupported migration dialect %q", dialect)
	}
	// The advisory lock is taken before anything touches the schema, including
	// the tracking table. CREATE TABLE IF NOT EXISTS is not atomic against a
	// concurrent creation in PostgreSQL: both instances find no table, both
	// create it, and the loser fails with a duplicate key on
	// pg_type_typname_nsp_index. Two Pods starting together on an empty database
	// - a rolling update, or a Deployment scaled past one replica - is exactly
	// that case, and all but one of them exited at boot.
	var target migrationTarget = db
	if dialect == "postgres" {
		connection, err := db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("reserve migration connection: %w", err)
		}
		defer connection.Close()
		const lockID int64 = 528_022_026_072_900_001
		if _, err := connection.ExecContext(
			ctx, "SELECT pg_advisory_lock($1)", lockID,
		); err != nil {
			return fmt.Errorf("acquire PostgreSQL migration lock: %w", err)
		}
		defer connection.ExecContext(
			context.Background(), "SELECT pg_advisory_unlock($1)", lockID,
		)
		target = connection
	}
	if _, err := target.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("create migration tracking table: %w", err)
	}
	entries, err := fs.ReadDir(migrations.Files, dialect)
	if err != nil {
		return fmt.Errorf("list %s migrations: %w", dialect, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versionText, _, found := strings.Cut(entry.Name(), "_")
		if !found {
			return fmt.Errorf("migration %q has no numeric prefix", entry.Name())
		}
		version, err := strconv.ParseInt(versionText, 10, 64)
		if err != nil {
			return fmt.Errorf("migration %q has invalid version: %w", entry.Name(), err)
		}
		applied, err := migrationApplied(ctx, target, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := fs.ReadFile(migrations.Files, dialect+"/"+entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		if err := runMigration(ctx, target, version, string(body)); err != nil {
			return fmt.Errorf("apply migration %q: %w", entry.Name(), err)
		}
	}
	return nil
}

type migrationTarget interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func migrationApplied(ctx context.Context, db migrationTarget, version int64) (bool, error) {
	var count int
	if err := db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = $1",
		version,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("query migration version %d: %w", version, err)
	}
	return count > 0, nil
}

func runMigration(ctx context.Context, db migrationTarget, version int64, body string) error {
	transaction, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer transaction.Rollback()
	for _, statement := range splitStatements(body) {
		if _, err := transaction.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := transaction.ExecContext(
		ctx,
		"INSERT INTO schema_migrations(version) VALUES ($1)",
		version,
	); err != nil {
		return fmt.Errorf("record migration version: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}

func splitStatements(body string) []string {
	var statements []string
	var builder strings.Builder
	var quote rune
	var previous rune
	for _, char := range body {
		switch {
		case quote != 0:
			builder.WriteRune(char)
			if char == quote && previous != '\\' {
				quote = 0
			}
		case char == '\'' || char == '"':
			quote = char
			builder.WriteRune(char)
		case char == ';':
			if statement := strings.TrimSpace(builder.String()); statement != "" {
				statements = append(statements, statement)
			}
			builder.Reset()
		default:
			builder.WriteRune(char)
		}
		previous = char
	}
	if statement := strings.TrimSpace(builder.String()); statement != "" {
		statements = append(statements, statement)
	}
	return statements
}
