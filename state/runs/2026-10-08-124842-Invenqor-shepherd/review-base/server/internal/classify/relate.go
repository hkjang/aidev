package classify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Relationship inference, and its deliberate limits.
//
// Only relationships the collected data actually supports are derived. The one
// unambiguous fact in an agent inventory is co-location: every record in an
// event was read from the same machine, so a service, a container runtime, an
// interface or a volume in that event belongs to that machine. Those edges are
// created with high confidence and marked active.
//
// What is NOT inferred, on purpose:
//
//   - Service-to-service dependencies. The agent reports listening sockets but
//     never a peer, so any "A depends on B" would be invented. That needs a
//     collector that reports established connections; until then a guess in a
//     CMDB is worse than a gap, because change management would act on it.
//   - One edge per installed package. A host with 2,000 packages would bury the
//     graph. Which software earns an edge is a curation decision expressed in
//     the rule set (relate_to_host), not a hard-coded list.
//
// Duplicate detection is a proposal, never an automatic merge: two agents
// reporting the same machine identifier usually means a cloned image, and
// merging the wrong pair of hosts is expensive to undo.

const (
	// DerivationSameAgent marks an edge that follows from co-location in one
	// inventory event.
	DerivationSameAgent = "same_agent_inventory"
	// DerivationMachineIdentity marks a duplicate proposal from a shared
	// hardware or machine identifier.
	DerivationMachineIdentity = "machine_identity"
)

// LinkToHost records that a collected component belongs to the host the agent
// runs on. Manual edges are authoritative: if a person already stated this
// relationship, the derived pass leaves their row untouched.
func LinkToHost(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	componentAssetID string,
	relation string,
	confidence float64,
) error {
	if relation == "" {
		relation = "runs_on"
	}
	hostAssetID, err := HostAssetForAgent(ctx, transaction, agentInternalID)
	if err != nil {
		return err
	}
	if hostAssetID == "" || hostAssetID == componentAssetID {
		// The host record itself, or an agent whose system inventory has not
		// arrived yet; the next event links it.
		return nil
	}
	var existing string
	var existingSource string
	err = transaction.QueryRowContext(
		ctx,
		`SELECT id, source FROM asset_relations
		  WHERE source_asset_id = $1 AND relation_type = $2 AND target_asset_id = $3`,
		componentAssetID,
		relation,
		hostAssetID,
	).Scan(&existing, &existingSource)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO asset_relations(
				id, source_asset_id, relation_type, target_asset_id,
				source, confidence, derivation, status
			) VALUES ($1, $2, $3, $4, 'inferred', $5, $6, 'active')`,
			uuid.NewString(),
			componentAssetID,
			relation,
			hostAssetID,
			confidence,
			DerivationSameAgent,
		); err != nil {
			return fmt.Errorf("record inferred relationship: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("look up existing relationship: %w", err)
	}
	if existingSource == "manual" {
		return nil
	}
	// Refresh the derived row so a relationship that reappears leaves the
	// retired state behind.
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE asset_relations
		    SET confidence = $1, derivation = $2, status = 'active', valid_to = NULL
		  WHERE id = $3`,
		confidence,
		DerivationSameAgent,
		existing,
	); err != nil {
		return fmt.Errorf("refresh inferred relationship: %w", err)
	}
	return nil
}

// HostAssetForAgent resolves the host asset an agent represents. The system
// inventory record is the authority; the enrollment placeholder covers the
// window before the first inventory arrives.
func HostAssetForAgent(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
) (string, error) {
	var assetID string
	err := transaction.QueryRowContext(
		ctx,
		`SELECT asset_id FROM asset_sources
		  WHERE agent_id = $1 AND category IN ('system','enrollment')
		    AND deleted_at IS NULL
		  ORDER BY CASE WHEN category = 'system' THEN 0 ELSE 1 END,
		           first_seen_at
		  LIMIT 1`,
		agentInternalID,
	).Scan(&assetID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve host asset for agent: %w", err)
	}
	return assetID, nil
}

// ProposeDuplicate records that two host assets look like the same machine. It
// is stored as a proposal so a person decides whether to merge.
func ProposeDuplicate(
	ctx context.Context,
	transaction *sql.Tx,
	assetID string,
	otherAssetID string,
	confidence float64,
) error {
	if assetID == "" || otherAssetID == "" || assetID == otherAssetID {
		return nil
	}
	// One direction only, ordered, so the same pair cannot be proposed twice.
	source, target := assetID, otherAssetID
	if source > target {
		source, target = target, source
	}
	var existing string
	err := transaction.QueryRowContext(
		ctx,
		`SELECT id FROM asset_relations
		  WHERE source_asset_id = $1 AND relation_type = 'duplicate_of'
		    AND target_asset_id = $2`,
		source,
		target,
	).Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("look up duplicate proposal: %w", err)
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO asset_relations(
			id, source_asset_id, relation_type, target_asset_id,
			source, confidence, derivation, status
		) VALUES ($1, $2, 'duplicate_of', $3, 'inferred', $4, $5, 'proposed')`,
		uuid.NewString(),
		source,
		target,
		confidence,
		DerivationMachineIdentity,
	); err != nil {
		return fmt.Errorf("record duplicate proposal: %w", err)
	}
	return nil
}

// ProposeDuplicatesByMachineIdentity looks for another agent reporting the same
// machine identifier and proposes a merge. Cloned images are the usual cause and
// they quietly double an inventory, so this is worth surfacing.
func ProposeDuplicatesByMachineIdentity(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	assetID string,
	machineIdentifier string,
) error {
	if machineIdentifier == "" || assetID == "" {
		return nil
	}
	// payload_json is JSONB on PostgreSQL and TEXT on the SQLite fallback, and
	// PostgreSQL has no LIKE for jsonb - the query failed with SQLSTATE 42883
	// ("operator does not exist: jsonb ~~ unknown") on every ingest, which the
	// SQLite-backed tests could never have caught. Casting gives one statement
	// that both engines accept.
	//
	// The LIKE is only a prefilter. A machine identifier is a UUID that can
	// legitimately appear elsewhere in a system payload - a disk serial, a
	// tenant id - and treating any occurrence as a match proposes a merge
	// between two unrelated hosts. The payload comes back so the identity
	// fields can be compared exactly.
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT s.asset_id, CAST(s.payload_json AS TEXT) FROM asset_sources s
		  WHERE s.category = 'system' AND s.agent_id <> $1
		    AND s.deleted_at IS NULL
		    AND CAST(s.payload_json AS TEXT) LIKE $2`,
		agentInternalID,
		"%"+machineIdentifier+"%",
	)
	if err != nil {
		return fmt.Errorf("search matching machine identifiers: %w", err)
	}
	defer rows.Close()
	others := make([]string, 0, 2)
	for rows.Next() {
		var other, payload string
		if err := rows.Scan(&other, &payload); err != nil {
			return err
		}
		if !identifiesMachine(payload, machineIdentifier) {
			continue
		}
		others = append(others, other)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, other := range others {
		if err := ProposeDuplicate(ctx, transaction, assetID, other, 0.6); err != nil {
			return err
		}
	}
	return nil
}

// identifiesMachine reports whether the payload names this identifier in one of
// the fields that actually identify a machine, rather than merely containing the
// text somewhere.
func identifiesMachine(payload string, identifier string) bool {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		// An unreadable payload is not evidence of anything.
		return false
	}
	for _, field := range []string{"machine_id", "firmware_uuid", "dmi_product_uuid"} {
		if value, ok := decoded[field].(string); ok &&
			strings.EqualFold(strings.TrimSpace(value), identifier) {
			return true
		}
	}
	return false
}
