package storage

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Two instances starting at once against one empty database is what a rolling
// update does, and what a Deployment scaled past a single replica does at boot.
//
// It used to lose all but one of them. The advisory lock guarding the
// migrations was taken after the tracking table was created, and CREATE TABLE
// IF NOT EXISTS is not atomic against a concurrent creation in PostgreSQL: each
// instance finds no table, each creates it, and every loser fails with a
// duplicate key on pg_type_typname_nsp_index. Five of six instances exited at
// boot. SQLite serialises writers, so it never showed this.
func TestInstancesStartingTogetherAllMigrateSuccessfully(t *testing.T) {
	if !postgresConfigured() {
		t.Skip("set INVENQOR_TEST_POSTGRES_DSN to exercise concurrent migration")
	}
	const instances = 6
	schema := freshSchema(t)
	failures := make([]error, instances)
	start := make(chan struct{})
	var running sync.WaitGroup
	for index := range instances {
		running.Add(1)
		go func() {
			defer running.Done()
			// Released together so the migrations genuinely overlap; started
			// one at a time, the first would finish before the next looked.
			<-start
			runtime, err := Open(context.Background(), Options{
				PostgresDSN: postgresDSN(),
				Schema:      schema,
			})
			if err != nil {
				failures[index] = err
				return
			}
			_ = runtime.Close()
		}()
	}
	close(start)
	running.Wait()

	for index, err := range failures {
		if err != nil {
			t.Errorf("instance %d did not start: %v", index, err)
		}
	}
}

func postgresDSN() string { return os.Getenv("INVENQOR_TEST_POSTGRES_DSN") }

func postgresConfigured() bool { return postgresDSN() != "" }

// freshSchema gives this test its own schema, created empty so the migrations
// genuinely run rather than finding everything already applied.
func freshSchema(t *testing.T) string {
	t.Helper()
	name := "concurrent_migration_test"
	database, err := sql.Open("pgx", postgresDSN())
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer database.Close()
	if _, err := database.Exec(`DROP SCHEMA IF EXISTS "` + name + `" CASCADE`); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := database.Exec(`CREATE SCHEMA "` + name + `"`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return name
}
