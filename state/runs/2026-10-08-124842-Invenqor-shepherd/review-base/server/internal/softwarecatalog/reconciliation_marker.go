package softwarecatalog

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ReconciliationRequired reserves the per-agent marker row and reports
// whether the active built-in catalogue has not yet been applied. Reserving a
// row also serializes first-heartbeat backfills from multiple Server pods: a
// concurrent transaction waits for the winner, then observes its version.
func ReconciliationRequired(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
) (bool, error) {
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO software_catalog_reconciliations(
			agent_id, catalog_version, reconciled_at
		 ) VALUES ($1, '', NULL)
		 ON CONFLICT(agent_id) DO UPDATE SET agent_id = excluded.agent_id`,
		agentInternalID,
	); err != nil {
		return false, fmt.Errorf("reserve software catalogue reconciliation: %w", err)
	}
	var version string
	var reconciledAt any
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT catalog_version, reconciled_at
		   FROM software_catalog_reconciliations
		  WHERE agent_id = $1`,
		agentInternalID,
	).Scan(&version, &reconciledAt); err != nil {
		return false, fmt.Errorf("read software catalogue reconciliation: %w", err)
	}
	return version != CatalogVersion || reconciledAt == nil, nil
}

func recordReconciliation(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	now time.Time,
) error {
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO software_catalog_reconciliations(
			agent_id, catalog_version, reconciled_at
		 ) VALUES ($1, $2, $3)
		 ON CONFLICT(agent_id) DO UPDATE SET
			catalog_version = excluded.catalog_version,
			reconciled_at = excluded.reconciled_at`,
		agentInternalID,
		CatalogVersion,
		now,
	); err != nil {
		return fmt.Errorf("record software catalogue reconciliation: %w", err)
	}
	return nil
}
