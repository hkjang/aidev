package softwarecatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/classify"
)

type existingProduct struct {
	SourceID      string
	AssetID       string
	ProductKey    string
	Previous      string
	SourceDeleted bool
	AssetStatus   string
	AssetDeleted  bool
}

type productHost struct {
	Environment string
	Name        string
}

// Reconcile rebuilds the host's normalized product view from all currently
// active package, service and process evidence in the same transaction as the
// inventory event. This is important for two reasons: a removed process cannot
// leave a product looking "running", and a second Server pod cannot observe a
// half-written product view.
func Reconcile(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	agentExternalID string,
	eventID string,
	now time.Time,
) error {
	observations, err := loadObservations(ctx, transaction, agentInternalID)
	if err != nil {
		return err
	}
	detections := Detect(observations)
	existing, err := loadExisting(ctx, transaction, agentInternalID)
	if err != nil {
		return err
	}
	host, err := productHostForAgent(ctx, transaction, agentInternalID)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(detections))
	for _, detection := range detections {
		seen[detection.ProductKey] = true
		row, found := existing[detection.ProductKey]
		if err := upsertDetection(
			ctx, transaction, agentInternalID, agentExternalID, eventID,
			now, host, detection, row, found,
		); err != nil {
			return err
		}
	}
	for productKey, row := range existing {
		if seen[productKey] || row.SourceDeleted {
			continue
		}
		if err := retireDetection(
			ctx, transaction, agentInternalID, eventID, now, row,
		); err != nil {
			return err
		}
	}
	// Record the version even when no products were detected. Without this
	// zero-detection hosts would repeat a full evidence scan on every heartbeat.
	return recordReconciliation(ctx, transaction, agentInternalID, now)
}

func loadObservations(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
) ([]Observation, error) {
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT category, source_asset_id, payload_json
		   FROM asset_sources
		  WHERE agent_id = $1
		    AND category IN ('process','service','software.package')
		    AND deleted_at IS NULL
		  ORDER BY category, source_asset_id`,
		agentInternalID,
	)
	if err != nil {
		return nil, fmt.Errorf("load software evidence: %w", err)
	}
	defer rows.Close()
	observations := make([]Observation, 0, 128)
	for rows.Next() {
		var observation Observation
		var payload any
		if err := rows.Scan(
			&observation.Category, &observation.SourceAssetID, &payload,
		); err != nil {
			return nil, fmt.Errorf("scan software evidence: %w", err)
		}
		observation.Payload = json.RawMessage(databaseText(payload))
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate software evidence: %w", err)
	}
	return observations, nil
}

func loadExisting(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
) (map[string]existingProduct, error) {
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT s.id, s.asset_id, s.source_asset_id, s.payload_json,
		        s.deleted_at, a.status, a.deleted_at
		   FROM asset_sources s
		   JOIN assets a ON a.id = s.asset_id
		  WHERE s.agent_id = $1 AND s.category = 'software.product'
		  ORDER BY s.source_asset_id`,
		agentInternalID,
	)
	if err != nil {
		return nil, fmt.Errorf("load normalized software products: %w", err)
	}
	defer rows.Close()
	result := map[string]existingProduct{}
	for rows.Next() {
		var row existingProduct
		var previous any
		var sourceDeleted, assetDeleted any
		if err := rows.Scan(
			&row.SourceID, &row.AssetID, &row.ProductKey, &previous,
			&sourceDeleted, &row.AssetStatus, &assetDeleted,
		); err != nil {
			return nil, fmt.Errorf("scan normalized software product: %w", err)
		}
		row.Previous = databaseText(previous)
		row.SourceDeleted = sourceDeleted != nil
		row.AssetDeleted = assetDeleted != nil
		result[row.ProductKey] = row
	}
	return result, rows.Err()
}

func upsertDetection(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	agentExternalID string,
	eventID string,
	now time.Time,
	host productHost,
	detection Detection,
	existing existingProduct,
	found bool,
) error {
	payloadBytes, err := json.Marshal(detection)
	if err != nil {
		return fmt.Errorf("encode normalized software product: %w", err)
	}
	payload := string(payloadBytes)
	tags, _ := json.Marshal([]string{detection.Role, "managed-software"})
	rules, _ := json.Marshal([]string{
		"builtin_software_catalog:" + CatalogVersion + ":" + detection.ProductKey,
	})
	assetID := existing.AssetID
	changeType := ""
	before := any(nil)
	if !found {
		assetID = uuid.NewString()
		environment := host.Environment
		if environment == "" {
			environment = "other"
		}
		assetKey := agentExternalID + ":software.product:" + detection.ProductKey
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO assets(
				id, asset_key, name, type, status, environment, confidence,
				attributes_json, source, first_seen_at, last_seen_at,
				created_at, updated_at, classification_source,
				classification_confidence, classification_rules_json,
				classified_at, tags_json
			) VALUES (
				$1, $2, $3, 'software_product', 'active', $4, $5,
				$6, 'inferred', $7, $7, $7, $7, 'software_catalog',
				$5, $8, $7, $9
			)`,
			assetID,
			assetKey,
			detection.ProductName,
			environment,
			detection.Confidence,
			payload,
			now,
			string(rules),
			string(tags),
		); err != nil {
			return fmt.Errorf("insert normalized software product %s: %w", detection.ProductKey, err)
		}
		existing.SourceID = uuid.NewString()
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO asset_sources(
				id, asset_id, agent_id, category, source_asset_id,
				source_name, payload_json, collected_at, first_seen_at,
				last_seen_at
			) VALUES ($1, $2, $3, 'software.product', $4,
			          'builtin_catalog', $5, $6, $6, $6)`,
			existing.SourceID,
			assetID,
			agentInternalID,
			detection.ProductKey,
			payload,
			now,
		); err != nil {
			return fmt.Errorf("insert normalized product source %s: %w", detection.ProductKey, err)
		}
		changeType = "added"
	} else {
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE asset_sources
			    SET source_name = 'builtin_catalog', payload_json = $1,
			        collected_at = $2, last_seen_at = $2, deleted_at = NULL
			  WHERE id = $3`,
			payload,
			now,
			existing.SourceID,
		); err != nil {
			return fmt.Errorf("refresh normalized product source %s: %w", detection.ProductKey, err)
		}
		environment := host.Environment
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE assets SET
			    name = $1, type = 'software_product', status = 'active',
			    environment = CASE WHEN $2 = '' THEN environment ELSE $2 END,
			    confidence = $3, attributes_json = $4, source = 'inferred',
			    last_seen_at = $5, updated_at = $5, deleted_at = NULL,
			    classification_source = 'software_catalog',
			    classification_confidence = $3,
			    classification_rules_json = $6, classified_at = $5,
			    tags_json = $7
			  WHERE id = $8`,
			detection.ProductName,
			environment,
			detection.Confidence,
			payload,
			now,
			string(rules),
			string(tags),
			assetID,
		); err != nil {
			return fmt.Errorf("refresh normalized software product %s: %w", detection.ProductKey, err)
		}
		if existing.SourceDeleted || existing.AssetDeleted || existing.AssetStatus != "active" {
			changeType = "added"
			before = existing.Previous
		} else if !sameJSONObject(existing.Previous, payload) {
			changeType = "updated"
			before = existing.Previous
		}
	}
	if changeType != "" {
		if err := recordChange(
			ctx, transaction, assetID, eventID, agentInternalID,
			changeType, before, payload,
		); err != nil {
			return err
		}
	}
	if err := classify.LinkToHost(
		ctx, transaction, agentInternalID, assetID, "runs_on", detection.Confidence,
	); err != nil {
		return fmt.Errorf("link normalized product %s to host: %w", detection.ProductKey, err)
	}
	if err := upsertInventoryProjection(
		ctx, transaction, agentInternalID, assetID, host.Name, detection, now,
	); err != nil {
		return err
	}
	return nil
}

func upsertInventoryProjection(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	assetID string,
	hostName string,
	detection Detection,
	now time.Time,
) error {
	searchValues := []string{
		detection.ProductKey,
		detection.ProductName,
		detection.Role,
		detection.Vendor,
		detection.Version,
		hostName,
	}
	searchValues = append(searchValues, detection.ServiceNames...)
	searchValues = append(searchValues, detection.ProcessNames...)
	searchValues = append(searchValues, detection.PackageNames...)
	searchText := strings.ToLower(strings.Join(searchValues, " "))
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO software_product_inventory(
			asset_id, agent_id, product_key, product_name, role, vendor,
			version, install_state, runtime_state, confidence, process_count,
			evidence_count, catalog_version, search_text, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		          $12, $13, $14, $15)
		ON CONFLICT(agent_id, product_key) DO UPDATE SET
			asset_id = excluded.asset_id,
			product_name = excluded.product_name,
			role = excluded.role,
			vendor = excluded.vendor,
			version = excluded.version,
			install_state = excluded.install_state,
			runtime_state = excluded.runtime_state,
			confidence = excluded.confidence,
			process_count = excluded.process_count,
			evidence_count = excluded.evidence_count,
			catalog_version = excluded.catalog_version,
			search_text = excluded.search_text,
			updated_at = excluded.updated_at`,
		assetID,
		agentInternalID,
		detection.ProductKey,
		detection.ProductName,
		detection.Role,
		detection.Vendor,
		detection.Version,
		detection.InstallState,
		detection.RuntimeState,
		detection.Confidence,
		detection.ProcessCount,
		detection.EvidenceCount,
		detection.CatalogVersion,
		searchText,
		now,
	); err != nil {
		return fmt.Errorf("project normalized software product %s: %w", detection.ProductKey, err)
	}
	return nil
}

func retireDetection(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
	eventID string,
	now time.Time,
	row existingProduct,
) error {
	if _, err := transaction.ExecContext(
		ctx,
		"UPDATE asset_sources SET deleted_at = $1 WHERE id = $2",
		now,
		row.SourceID,
	); err != nil {
		return fmt.Errorf("retire normalized product source %s: %w", row.ProductKey, err)
	}
	var activeSources int
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM asset_sources
		  WHERE asset_id = $1 AND deleted_at IS NULL`,
		row.AssetID,
	).Scan(&activeSources); err != nil {
		return fmt.Errorf("count normalized product sources: %w", err)
	}
	if activeSources == 0 {
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE assets SET status = 'removed', deleted_at = $1,
			        updated_at = $1 WHERE id = $2`,
			now,
			row.AssetID,
		); err != nil {
			return fmt.Errorf("retire normalized software product %s: %w", row.ProductKey, err)
		}
		if _, err := transaction.ExecContext(
			ctx,
			`UPDATE asset_relations SET valid_to = $1, status = 'inactive'
			  WHERE source_asset_id = $2 AND relation_type = 'runs_on'
			    AND source <> 'manual' AND valid_to IS NULL`,
			now,
			row.AssetID,
		); err != nil {
			return fmt.Errorf("retire normalized product relation %s: %w", row.ProductKey, err)
		}
	}
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM software_product_inventory WHERE asset_id = $1",
		row.AssetID,
	); err != nil {
		return fmt.Errorf("remove software product projection %s: %w", row.ProductKey, err)
	}
	return recordChange(
		ctx, transaction, row.AssetID, eventID, agentInternalID,
		"removed", row.Previous, "",
	)
}

func recordChange(
	ctx context.Context,
	transaction *sql.Tx,
	assetID string,
	eventID string,
	agentInternalID string,
	changeType string,
	before any,
	after string,
) error {
	var afterValue any
	if strings.TrimSpace(after) != "" {
		afterValue = after
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO asset_changes(
			id, asset_id, source_event_id, change_type, before_json,
			after_json, actor_type, actor_id, reason
		) VALUES ($1, $2, $3, $4, $5, $6, 'agent', $7,
		          'automatic_software_catalog')`,
		uuid.NewString(),
		assetID,
		eventID,
		changeType,
		before,
		afterValue,
		agentInternalID,
	); err != nil {
		return fmt.Errorf("record normalized product change: %w", err)
	}
	return nil
}

func productHostForAgent(
	ctx context.Context,
	transaction *sql.Tx,
	agentInternalID string,
) (productHost, error) {
	hostID, err := classify.HostAssetForAgent(ctx, transaction, agentInternalID)
	if err != nil || hostID == "" {
		return productHost{}, err
	}
	var host productHost
	if err := transaction.QueryRowContext(
		ctx,
		"SELECT environment, name FROM assets WHERE id = $1",
		hostID,
	).Scan(&host.Environment, &host.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return productHost{}, nil
		}
		return productHost{}, fmt.Errorf("read product host context: %w", err)
	}
	return host, nil
}

func sameJSONObject(left, right string) bool {
	var leftValue, rightValue any
	if json.Unmarshal([]byte(left), &leftValue) != nil ||
		json.Unmarshal([]byte(right), &rightValue) != nil {
		return left == right
	}
	leftJSON, _ := json.Marshal(leftValue)
	rightJSON, _ := json.Marshal(rightValue)
	return string(leftJSON) == string(rightJSON)
}

func databaseText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}
