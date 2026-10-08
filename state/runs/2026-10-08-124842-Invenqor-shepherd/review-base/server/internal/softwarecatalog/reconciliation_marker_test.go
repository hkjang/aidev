package softwarecatalog

import (
	"context"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
)

func TestReconciliationMarkerTracksMissingCurrentAndOutdatedCatalog(t *testing.T) {
	t.Parallel()
	runtime := storagetest.Open(t)
	t.Cleanup(func() { _ = runtime.Close() })
	agentID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO agents(id, agent_id, status) VALUES($1, $2, 'active')`,
		agentID,
		uuid.NewString(),
	); err != nil {
		t.Fatal(err)
	}

	tx, err := runtime.DB().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	required, err := ReconciliationRequired(context.Background(), tx, agentID)
	if err != nil || !required {
		t.Fatalf("missing marker required = %v, error = %v", required, err)
	}
	when := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	if err := recordReconciliation(context.Background(), tx, agentID, when); err != nil {
		t.Fatal(err)
	}
	required, err = ReconciliationRequired(context.Background(), tx, agentID)
	if err != nil || required {
		t.Fatalf("current marker required = %v, error = %v", required, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if _, err := runtime.DB().Exec(
		`UPDATE software_catalog_reconciliations
		    SET catalog_version = $1, reconciled_at = NULL WHERE agent_id = $2`,
		CatalogVersion,
		agentID,
	); err != nil {
		t.Fatal(err)
	}
	tx, err = runtime.DB().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	required, err = ReconciliationRequired(context.Background(), tx, agentID)
	if err != nil || !required {
		t.Fatalf("unfinished marker required = %v, error = %v", required, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DB().Exec(
		`UPDATE software_catalog_reconciliations
		    SET catalog_version = 'outdated' WHERE agent_id = $1`,
		agentID,
	); err != nil {
		t.Fatal(err)
	}
	tx, err = runtime.DB().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	required, err = ReconciliationRequired(context.Background(), tx, agentID)
	if err != nil || !required {
		t.Fatalf("outdated marker required = %v, error = %v", required, err)
	}
}
