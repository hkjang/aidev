package ingest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/classify"
	"github.com/hkjang/invenqor/server/internal/softwarecatalog"
)

var ErrInProgress = errors.New("event is already being processed")

type Result struct {
	Duplicate     bool   `json:"duplicate"`
	PolicyVersion string `json:"policy_version,omitempty"`
}

type Service struct {
	database   *sql.DB
	now        func() time.Time
	classifier *classify.Store
}

const maximumAgentClockSkew = 10 * time.Minute

func NewService(database *sql.DB) *Service {
	return &Service{
		database:   database,
		now:        func() time.Time { return time.Now().UTC() },
		classifier: classify.NewStore(database),
	}
}

// Classifier exposes the rule store so the administrative API can invalidate the
// cache after an edit and reuse the same engine for a full reclassification.
func (s *Service) Classifier() *classify.Store {
	return s.classifier
}

func (s *Service) Process(
	ctx context.Context,
	agent agents.Agent,
	envelope Envelope,
	raw []byte,
	agentVersion string,
) (Result, error) {
	if agent.AgentID != envelope.AgentID {
		return Result{}, errors.New("authenticated agent does not match event agent_id")
	}
	// Read the rule set before opening the transaction. SQLite serialises
	// writers, so a query on another connection while this event holds the write
	// lock deadlocks; the rules are also immutable for the length of an event.
	rules, err := s.classifier.Rules(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("load classification rules: %w", err)
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, fmt.Errorf("begin event processing: %w", err)
	}
	defer tx.Rollback()

	eventInternalID := uuid.NewString()
	insert, err := tx.ExecContext(
		ctx,
		`INSERT INTO agent_events(
			id, agent_id, event_id, schema_version, kind, snapshot_hash,
			raw_event, processing_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'processing')
		ON CONFLICT(agent_id, event_id) DO NOTHING`,
		eventInternalID,
		agent.ID,
		envelope.EventID,
		envelope.SchemaVersion,
		envelope.Kind,
		envelope.SnapshotHash,
		string(raw),
	)
	if err != nil {
		return Result{}, fmt.Errorf("reserve event: %w", err)
	}
	inserted, _ := insert.RowsAffected()
	if inserted == 0 {
		status := ""
		if err := tx.QueryRowContext(
			ctx,
			`SELECT id, processing_status FROM agent_events
			 WHERE agent_id = $1 AND event_id = $2`,
			agent.ID,
			envelope.EventID,
		).Scan(&eventInternalID, &status); err != nil {
			return Result{}, fmt.Errorf("read existing event: %w", err)
		}
		if status == "processed" {
			if err := tx.Commit(); err != nil {
				return Result{}, err
			}
			return Result{
				Duplicate:     true,
				PolicyVersion: agent.PolicyVersion,
			}, nil
		}
		if status == "processing" {
			return Result{}, ErrInProgress
		}
		claim, err := tx.ExecContext(
			ctx,
			`UPDATE agent_events
			 SET processing_status = 'processing', processing_error = NULL
			 WHERE id = $1 AND processing_status IN ('pending', 'failed')`,
			eventInternalID,
		)
		if err != nil {
			return Result{}, err
		}
		if count, _ := claim.RowsAffected(); count == 0 {
			return Result{}, ErrInProgress
		}
	}

	var receivedAt apitime.Time
	if err := tx.QueryRowContext(
		ctx,
		"SELECT received_at FROM agent_events WHERE id=$1",
		eventInternalID,
	).Scan(&receivedAt); err != nil || !receivedAt.Valid {
		if err == nil {
			err = errors.New("agent event received_at is invalid")
		}
		return Result{}, fmt.Errorf("read event receive time: %w", err)
	}
	futureThreshold := receivedAt.Time.Add(maximumAgentClockSkew)
	createdAt, timestampClamped := effectiveAgentTimestamp(
		envelope.CreatedAt, receivedAt.Time,
	)
	if envelopeHasFutureRecord(envelope, receivedAt.Time) {
		timestampClamped = true
	}
	if timestampClamped {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO agent_event_errors(
			 id,agent_event_id,error_code,message
			 ) VALUES($1,$2,'FUTURE_TIMESTAMP_CLAMPED',$3)`,
			uuid.NewString(),
			eventInternalID,
			"Agent timestamps exceeded the database clock by more than 10 minutes and were clamped to received_at.",
		); err != nil {
			return Result{}, fmt.Errorf("record future timestamp diagnostic: %w", err)
		}
	}
	// v0.2.15 introduced collision-free RPM and Windows package IDs. Move a
	// strictly matching legacy source before applying the delta so a legacy
	// "removed" followed by the canonical "added" cannot retire and recreate
	// the representative asset in between.
	if err := s.migrateLegacyPackageSources(ctx, tx, agent, envelope); err != nil {
		return s.failEvent(
			ctx, tx, eventInternalID, agent, envelope, raw, err,
		)
	}
	systemRecordApplied := false
	if envelope.Snapshot != nil {
		for _, record := range envelope.Snapshot.Records {
			assetID, applied, err := s.upsertRecord(
				ctx, tx, agent, eventInternalID, record, "",
				createdAt, receivedAt.Time, envelope.EventID,
			)
			if err != nil {
				return s.failEvent(
					ctx, tx, eventInternalID, agent, envelope, raw, err,
				)
			}
			if applied {
				if err := s.classifyRecord(
					ctx, tx, agent, rules, assetID, record,
				); err != nil {
					return s.failEvent(
						ctx, tx, eventInternalID, agent, envelope, raw, err,
					)
				}
				systemRecordApplied = systemRecordApplied || record.Category == "system"
			}
		}
	}
	for _, change := range envelope.Changes {
		if change.Kind == "removed" {
			if err := s.removeRecord(
				ctx,
				tx,
				agent,
				eventInternalID,
				change.Category,
				change.AssetID,
				createdAt,
				receivedAt.Time,
				envelope.EventID,
			); err != nil {
				return s.failEvent(
					ctx, tx, eventInternalID, agent, envelope, raw, err,
				)
			}
			continue
		}
		assetID, applied, err := s.upsertRecord(
			ctx, tx, agent, eventInternalID, *change.Record, change.Kind,
			createdAt, receivedAt.Time, envelope.EventID,
		)
		if err != nil {
			return s.failEvent(
				ctx, tx, eventInternalID, agent, envelope, raw, err,
			)
		}
		if applied {
			if err := s.classifyRecord(
				ctx, tx, agent, rules, assetID, *change.Record,
			); err != nil {
				return s.failEvent(
					ctx, tx, eventInternalID, agent, envelope, raw, err,
				)
			}
			systemRecordApplied = systemRecordApplied ||
				change.Record.Category == "system"
		}
	}
	// Raw processes are useful evidence but poor CMDB assets: one database may
	// expose dozens of PIDs and a Windows workstation may expose hundreds. Build
	// one host-scoped, explainable software product from all active process,
	// service, and package evidence. The reconciliation is part of this event's
	// transaction, so another Server pod can never see stale runtime state.
	reconcileSoftware, err := softwarecatalog.ReconciliationRequired(
		ctx, tx, agent.ID,
	)
	if err != nil {
		return s.failEvent(
			ctx, tx, eventInternalID, agent, envelope, raw, err,
		)
	}
	// Inventory changes always rebuild the view. A heartbeat does so only once
	// per catalogue version, which upgrades dormant v0.2.13 Agents without
	// repeatedly scanning unchanged raw evidence.
	reconcileSoftware = envelope.Kind == "inventory" || reconcileSoftware
	if reconcileSoftware {
		if err := softwarecatalog.Reconcile(
			ctx,
			tx,
			agent.ID,
			agent.AgentID,
			eventInternalID,
			s.now(),
		); err != nil {
			return s.failEvent(
				ctx, tx, eventInternalID, agent, envelope, raw, err,
			)
		}
	}
	errorsToStore := append(
		append([]CollectionError{}, envelope.CollectionErrors...),
		snapshotErrors(envelope.Snapshot)...,
	)
	for _, collectionError := range errorsToStore {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO collector_errors(
				id, agent_event_id, collector, message
			) VALUES ($1, $2, $3, $4)`,
			uuid.NewString(),
			eventInternalID,
			collectionError.Collector,
			collectionError.Message,
		); err != nil {
			return s.failEvent(
				ctx, tx, eventInternalID, agent, envelope, raw, err,
			)
		}
	}
	if systemRecordApplied {
		if err := s.proposeDuplicates(ctx, tx, agent, envelope); err != nil {
			return s.failEvent(
				ctx, tx, eventInternalID, agent, envelope, raw, err,
			)
		}
	}
	now := s.now()
	isInventory := envelope.Kind == "inventory"
	hostname, osName, architecture, err := agentMetadataWithStoredFallback(
		ctx, tx, agent, envelope, systemRecordApplied,
	)
	if err != nil {
		return s.failEvent(
			ctx, tx, eventInternalID, agent, envelope, raw, err,
		)
	}
	_, err = tx.ExecContext(
		ctx,
		`UPDATE agents SET
			hostname = CASE
			  WHEN (last_event_created_at IS NULL OR last_event_created_at > $10 OR
			        last_event_received_at > $10 OR last_event_created_at < $7 OR
			        (last_event_created_at = $7 AND
			         (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			          (last_event_received_at = $8 AND last_event_id < $9))))
			       AND $1 <> '' THEN $1 ELSE hostname END,
			os_name = CASE
			  WHEN (last_event_created_at IS NULL OR last_event_created_at > $10 OR
			        last_event_received_at > $10 OR last_event_created_at < $7 OR
			        (last_event_created_at = $7 AND
			         (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			          (last_event_received_at = $8 AND last_event_id < $9))))
			       AND $2 <> '' THEN $2 ELSE os_name END,
			architecture = CASE
			  WHEN (last_event_created_at IS NULL OR last_event_created_at > $10 OR
			        last_event_received_at > $10 OR last_event_created_at < $7 OR
			        (last_event_created_at = $7 AND
			         (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			          (last_event_received_at = $8 AND last_event_id < $9))))
			       AND $3 <> '' THEN $3 ELSE architecture END,
			version = CASE
			  WHEN (last_event_created_at IS NULL OR last_event_created_at > $10 OR
			        last_event_received_at > $10 OR last_event_created_at < $7 OR
			        (last_event_created_at = $7 AND
			         (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			          (last_event_received_at = $8 AND last_event_id < $9))))
			       AND $4 <> '' THEN $4 ELSE version END,
			status = 'active', last_seen_at = $5,
			last_inventory_at = CASE
			  WHEN $6 = FALSE THEN last_inventory_at
			  WHEN last_inventory_at > $10 THEN $7
			  WHEN last_inventory_at IS NULL OR last_inventory_at < $7 THEN $7
			  ELSE last_inventory_at
			END,
			last_event_created_at = CASE
			  WHEN last_event_created_at IS NULL OR last_event_created_at > $10 OR
			       last_event_received_at > $10 OR last_event_created_at < $7 OR
			       (last_event_created_at = $7 AND
			        (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			         (last_event_received_at = $8 AND last_event_id < $9)))
			  THEN $7
			  ELSE last_event_created_at
			END,
			last_event_received_at = CASE
			  WHEN last_event_created_at IS NULL OR last_event_created_at > $10 OR
			       last_event_received_at > $10 OR last_event_created_at < $7 OR
			       (last_event_created_at = $7 AND
			        (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			         (last_event_received_at = $8 AND last_event_id < $9)))
			  THEN $8
			  ELSE last_event_received_at
			END,
			last_event_id = CASE
			  WHEN last_event_created_at IS NULL OR last_event_created_at > $10 OR
			       last_event_received_at > $10 OR last_event_created_at < $7 OR
			       (last_event_created_at = $7 AND
			        (last_event_received_at IS NULL OR last_event_received_at < $8 OR
			         (last_event_received_at = $8 AND last_event_id < $9)))
			  THEN $9
			  ELSE last_event_id
			END,
			updated_at = $5
		 WHERE id = $11`,
		hostname,
		osName,
		architecture,
		agentVersion,
		now,
		isInventory,
		createdAt,
		receivedAt.Time,
		envelope.EventID,
		futureThreshold,
		agent.ID,
	)
	if err != nil {
		return s.failEvent(
			ctx, tx, eventInternalID, agent, envelope, raw, err,
		)
	}
	_, err = tx.ExecContext(
		ctx,
		`UPDATE agent_events
		 SET processing_status = 'processed', processed_at = $1,
		     processing_error = NULL
		 WHERE id = $2`,
		now,
		eventInternalID,
	)
	if err != nil {
		return s.failEvent(
			ctx, tx, eventInternalID, agent, envelope, raw, err,
		)
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("commit event: %w", err)
	}
	return Result{PolicyVersion: agent.PolicyVersion}, nil
}

func (s *Service) failEvent(
	ctx context.Context,
	tx *sql.Tx,
	eventID string,
	agent agents.Agent,
	envelope Envelope,
	raw []byte,
	cause error,
) (Result, error) {
	_ = tx.Rollback()
	message := cause.Error()
	if len(message) > 1000 {
		message = message[:1000]
	}
	_, _ = s.database.ExecContext(
		ctx,
		`INSERT INTO agent_events(
			id, agent_id, event_id, schema_version, kind, snapshot_hash,
			raw_event, processing_status, processing_error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'failed', $8)
		ON CONFLICT(agent_id, event_id) DO UPDATE
		SET processing_status = 'failed', processing_error = excluded.processing_error
		WHERE agent_events.processing_status <> 'processed'`,
		eventID,
		agent.ID,
		envelope.EventID,
		envelope.SchemaVersion,
		envelope.Kind,
		envelope.SnapshotHash,
		string(raw),
		message,
	)
	return Result{}, cause
}

func (s *Service) migrateLegacyPackageSources(
	ctx context.Context,
	tx *sql.Tx,
	agent agents.Agent,
	envelope Envelope,
) error {
	records := make([]AssetRecord, 0)
	if envelope.Snapshot != nil {
		records = append(records, envelope.Snapshot.Records...)
	}
	for _, change := range envelope.Changes {
		if change.Record != nil &&
			(change.Kind == "added" || change.Kind == "updated") {
			records = append(records, *change.Record)
		}
	}
	type candidate struct {
		record         AssetRecord
		identityFields []string
	}
	groups := make(map[string][]candidate)
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		legacyID, fields, ok := legacyPackageIdentity(record)
		if !ok {
			continue
		}
		key := record.Category + "\x00" + record.AssetID
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		groups[legacyID] = append(groups[legacyID], candidate{
			record: record, identityFields: fields,
		})
	}
	for legacyID, candidates := range groups {
		recordsForLegacy := make([]AssetRecord, len(candidates))
		fields := candidates[0].identityFields
		for index := range candidates {
			recordsForLegacy[index] = candidates[index].record
		}
		if err := migrateLegacyPackageSource(
			ctx, tx, agent, legacyID, recordsForLegacy, fields,
		); err != nil {
			return err
		}
	}
	return nil
}

// legacyPackageIdentity accepts only IDs that can be reproduced exactly from
// the new record payload. This prevents a merely similar package name from
// aliasing an unrelated representative asset and its manual metadata.
func legacyPackageIdentity(
	record AssetRecord,
) (string, []string, bool) {
	if record.Category != "software.package" {
		return "", nil, false
	}
	var payload map[string]any
	if json.Unmarshal(record.Payload, &payload) != nil {
		return "", nil, false
	}
	text := func(key string) string {
		value, _ := payload[key].(string)
		return value
	}
	manager := text("manager")
	name := text("name")
	architecture := text("architecture")
	if name == "" || architecture == "" {
		return "", nil, false
	}
	switch manager {
	case "rpm":
		version := text("version")
		legacyID := fmt.Sprintf("package:rpm:%s:%s", name, architecture)
		canonicalID := legacyID + ":" + version
		if version == "" || record.AssetID != canonicalID {
			return "", nil, false
		}
		return legacyID,
			[]string{"manager", "name", "architecture", "version"}, true
	case "windows":
		scope := text("scope")
		ownerSID := text("owner_sid")
		registryKey := text("registry_key")
		canonicalID := fmt.Sprintf(
			"package:windows:%s:%s:%s:%s",
			strings.ToLower(architecture),
			strings.ToLower(scope),
			strings.ToLower(ownerSID),
			strings.ToLower(registryKey),
		)
		if scope == "" || registryKey == "" || record.AssetID != canonicalID {
			return "", nil, false
		}
		return fmt.Sprintf("package:windows:%s:%s", name, architecture),
			[]string{
				"manager", "name", "architecture", "version", "scope",
				"registry_key",
			}, true
	default:
		return "", nil, false
	}
}

func migrateLegacyPackageSource(
	ctx context.Context,
	tx *sql.Tx,
	agent agents.Agent,
	legacyID string,
	candidates []AssetRecord,
	identityFields []string,
) error {
	if len(candidates) == 0 {
		return nil
	}
	category := candidates[0].Category
	for _, candidate := range candidates {
		var canonicalCount int
		if err := tx.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM asset_sources
			 WHERE agent_id=$1 AND category=$2 AND source_asset_id=$3`,
			agent.ID,
			category,
			candidate.AssetID,
		).Scan(&canonicalCount); err != nil {
			return fmt.Errorf("check canonical package source: %w", err)
		}
		if canonicalCount != 0 {
			return nil
		}
	}

	var sourceID, assetID, assetKey string
	var rawLegacyPayload any
	err := tx.QueryRowContext(
		ctx,
		`SELECT s.id, s.asset_id, s.payload_json, a.asset_key
		 FROM asset_sources s JOIN assets a ON a.id=s.asset_id
		 WHERE s.agent_id=$1 AND s.category=$2 AND s.source_asset_id=$3`,
		agent.ID,
		category,
		legacyID,
	).Scan(&sourceID, &assetID, &rawLegacyPayload, &assetKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read legacy package source: %w", err)
	}
	expectedLegacyAssetKey := agent.AgentID + ":" + category + ":" + legacyID
	if assetKey != expectedLegacyAssetKey {
		return nil
	}
	legacyPayload, err := jsonObject(rawLegacyPayload)
	if err != nil {
		return fmt.Errorf("decode legacy package source: %w", err)
	}
	matches := make([]AssetRecord, 0, len(candidates))
	for _, candidate := range candidates {
		var currentPayload map[string]any
		if err := json.Unmarshal(candidate.Payload, &currentPayload); err != nil {
			return fmt.Errorf("decode canonical package source: %w", err)
		}
		matched := true
		for _, field := range identityFields {
			legacyValue := stringField(legacyPayload, field)
			currentValue := stringField(currentPayload, field)
			if stringField(currentPayload, "manager") == "windows" &&
				(field == "manager" || field == "architecture" ||
					field == "scope" || field == "registry_key") {
				matched = strings.EqualFold(legacyValue, currentValue)
			} else {
				matched = legacyValue == currentValue
			}
			if !matched {
				break
			}
		}
		// v0.2.14 did not collect owner_sid. Adoption is safe only when the
		// remaining immutable registry identity selects exactly one canonical
		// candidate from this first upgraded event. If the legacy payload does
		// contain a SID, require it as well.
		if matched {
			if legacyOwner, present := legacyPayload["owner_sid"]; present {
				legacySID, _ := legacyOwner.(string)
				matched = strings.EqualFold(
					legacySID, stringField(currentPayload, "owner_sid"),
				)
			}
		}
		if matched {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return nil
	}
	record := matches[0]
	canonicalAssetKey := agent.AgentID + ":" + category + ":" + record.AssetID
	result, err := tx.ExecContext(
		ctx,
		`UPDATE assets SET asset_key=$1
		 WHERE id=$2 AND asset_key=$3`,
		canonicalAssetKey,
		assetID,
		expectedLegacyAssetKey,
	)
	if err != nil {
		return fmt.Errorf("migrate legacy package asset key: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return errors.New("legacy package asset key changed concurrently")
	}
	result, err = tx.ExecContext(
		ctx,
		`UPDATE asset_sources SET source_asset_id=$1
		 WHERE id=$2 AND source_asset_id=$3`,
		record.AssetID,
		sourceID,
		legacyID,
	)
	if err != nil {
		return fmt.Errorf("migrate legacy package source ID: %w", err)
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return errors.New("legacy package source changed concurrently")
	}
	return nil
}

func jsonObject(raw any) (map[string]any, error) {
	var encoded []byte
	switch value := raw.(type) {
	case string:
		encoded = []byte(value)
	case []byte:
		encoded = value
	default:
		encoded = []byte(fmt.Sprint(value))
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil || object == nil {
		return nil, errors.New("JSON value is not an object")
	}
	return object, nil
}

func stringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}

func envelopeHasFutureRecord(envelope Envelope, receivedAt time.Time) bool {
	isFuture := func(record AssetRecord) bool {
		_, clamped := effectiveAgentTimestamp(record.CollectedAt, receivedAt)
		return clamped
	}
	if envelope.Snapshot != nil {
		for _, record := range envelope.Snapshot.Records {
			if isFuture(record) {
				return true
			}
		}
	}
	for _, change := range envelope.Changes {
		if change.Record != nil && isFuture(*change.Record) {
			return true
		}
	}
	return false
}

func effectiveAgentTimestamp(raw uint64, receivedAt time.Time) (time.Time, bool) {
	// Converting an untrusted uint64 directly to int64 can wrap values above
	// MaxInt64 into pre-epoch dates. Compare the raw range first.
	if raw > uint64(1<<63-1) {
		return receivedAt, true
	}
	candidate := time.Unix(int64(raw), 0).UTC()
	if candidate.After(receivedAt.Add(maximumAgentClockSkew)) {
		return receivedAt, true
	}
	return candidate, false
}

func (s *Service) upsertRecord(
	ctx context.Context,
	tx *sql.Tx,
	agent agents.Agent,
	eventID string,
	record AssetRecord,
	explicitKind string,
	sourceEventCreatedAt time.Time,
	sourceEventReceivedAt time.Time,
	sourceEventID string,
) (string, bool, error) {
	payload := compactJSON(record.Payload)
	var sourceID, assetID, previous string
	var deleted any
	var sourceCollectedAt apitime.Time
	var lastEventCreatedAt apitime.Time
	var lastEventReceivedAt apitime.Time
	var lastEventID string
	err := tx.QueryRowContext(
		ctx,
		`SELECT s.id, s.asset_id, s.payload_json, s.deleted_at, s.collected_at,
		        s.last_event_created_at, s.last_event_received_at, s.last_event_id
		 FROM asset_sources s
		 WHERE s.agent_id = $1 AND s.category = $2 AND s.source_asset_id = $3`,
		agent.ID,
		record.Category,
		record.AssetID,
	).Scan(
		&sourceID,
		&assetID,
		&previous,
		&deleted,
		&sourceCollectedAt,
		&lastEventCreatedAt,
		&lastEventReceivedAt,
		&lastEventID,
	)
	newRecord := errors.Is(err, sql.ErrNoRows)
	if err != nil && !newRecord {
		return "", false, fmt.Errorf("read asset source: %w", err)
	}
	now := s.now()
	futureThreshold := sourceEventReceivedAt.Add(maximumAgentClockSkew)
	collectedAt, _ := effectiveAgentTimestamp(
		record.CollectedAt, sourceEventReceivedAt,
	)
	if !newRecord {
		if !sourceCollectedAt.Valid {
			return "", false, errors.New("asset source collected_at is invalid")
		}
		poisoned := sourceCollectedAt.Time.After(futureThreshold) ||
			lastEventCreatedAt.Valid && lastEventCreatedAt.Time.After(futureThreshold) ||
			lastEventReceivedAt.Valid && lastEventReceivedAt.Time.After(futureThreshold)
		if !poisoned && (collectedAt.Before(sourceCollectedAt.Time) ||
			lastEventCreatedAt.Valid &&
				(sourceEventCreatedAt.Before(lastEventCreatedAt.Time) ||
					sourceEventCreatedAt.Equal(lastEventCreatedAt.Time) &&
						lastEventReceivedAt.Valid &&
						(sourceEventReceivedAt.Before(lastEventReceivedAt.Time) ||
							sourceEventReceivedAt.Equal(lastEventReceivedAt.Time) &&
								lastEventID >= sourceEventID))) {
			return "", false, nil
		}
	}
	assetType := assetType(record.Category)
	name := assetName(record)
	representativeApplied := true
	if newRecord {
		sourceID = uuid.NewString()
		reusedEnrollmentAsset := false
		if record.Category == "system" {
			err = tx.QueryRowContext(
				ctx,
				`SELECT asset_id FROM asset_sources
				 WHERE agent_id = $1 AND category = 'enrollment'
				   AND deleted_at IS NULL
				 ORDER BY first_seen_at LIMIT 1`,
				agent.ID,
			).Scan(&assetID)
			switch {
			case err == nil:
				reusedEnrollmentAsset = true
			case errors.Is(err, sql.ErrNoRows):
				err = nil
			default:
				return "", false, fmt.Errorf(
					"find enrollment asset for inventory: %w",
					err,
				)
			}
		}
		if reusedEnrollmentAsset {
			_, err = tx.ExecContext(
				ctx,
				`UPDATE assets SET name=$1, type=$2, status='active',
				 confidence=1.0, attributes_json=$3, source='agent',
				 last_seen_at=$4, updated_at=$5, deleted_at=NULL
				 WHERE id=$6`,
				name,
				assetType,
				payload,
				collectedAt,
				now,
				assetID,
			)
			if err != nil {
				return "", false, fmt.Errorf("promote enrollment asset: %w", err)
			}
		} else {
			assetID = uuid.NewString()
			assetKey := agent.AgentID + ":" + record.Category + ":" + record.AssetID
			_, err = tx.ExecContext(
				ctx,
				`INSERT INTO assets(
					id, asset_key, name, type, status, attributes_json, source,
					first_seen_at, last_seen_at, created_at, updated_at
				) VALUES (
					$1, $2, $3, $4, 'active', $5, 'agent',
					$6, $6, $6, $6
				)`,
				assetID,
				assetKey,
				name,
				assetType,
				payload,
				collectedAt,
			)
			if err != nil {
				return "", false, fmt.Errorf("insert representative asset: %w", err)
			}
		}
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO asset_sources(
				id, asset_id, agent_id, category, source_asset_id,
				source_name, payload_json, collected_at, first_seen_at,
				last_seen_at, last_event_created_at, last_event_received_at,
				last_event_id
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $8, $8, $9, $10, $11
			)`,
			sourceID,
			assetID,
			agent.ID,
			record.Category,
			record.AssetID,
			record.Source,
			payload,
			collectedAt,
			sourceEventCreatedAt,
			sourceEventReceivedAt,
			sourceEventID,
		)
		if err != nil {
			return "", false, fmt.Errorf("insert asset source: %w", err)
		}
	} else {
		result, updateErr := tx.ExecContext(
			ctx,
			`UPDATE asset_sources SET source_name = $1, payload_json = $2,
			 collected_at = $3, last_seen_at = $3, deleted_at = NULL,
			 last_event_created_at = $4, last_event_received_at = $5,
			 last_event_id = $6
			 WHERE id = $7 AND (collected_at <= $3 OR collected_at > $8)
			   AND (last_event_created_at IS NULL OR last_event_created_at > $8 OR
			        last_event_received_at > $8 OR last_event_created_at < $4 OR
			        (last_event_created_at = $4 AND
			         (last_event_received_at IS NULL OR last_event_received_at < $5 OR
			          (last_event_received_at = $5 AND last_event_id < $6))))`,
			record.Source,
			payload,
			collectedAt,
			sourceEventCreatedAt,
			sourceEventReceivedAt,
			sourceEventID,
			sourceID,
			futureThreshold,
		)
		if updateErr != nil {
			return "", false, fmt.Errorf("update asset source: %w", updateErr)
		}
		if changed, _ := result.RowsAffected(); changed == 0 {
			return "", false, nil
		}
		result, err = tx.ExecContext(
			ctx,
			`UPDATE assets SET name = $1, attributes_json = $2,
			 status = 'active', last_seen_at = $3, updated_at = $4,
			 deleted_at = NULL WHERE id = $5
			 AND (last_seen_at <= $3 OR last_seen_at > $6)`,
			name,
			payload,
			collectedAt,
			now,
			assetID,
			futureThreshold,
		)
		if err != nil {
			return "", false, fmt.Errorf("update representative asset: %w", err)
		}
		if changed, _ := result.RowsAffected(); changed == 0 {
			representativeApplied = false
		}
	}
	hash := sha256.Sum256([]byte(payload))
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO asset_snapshots(
			id, asset_id, snapshot_json, snapshot_hash, captured_at
		) VALUES ($1, $2, $3, $4, $5)`,
		uuid.NewString(),
		assetID,
		payload,
		hex.EncodeToString(hash[:]),
		collectedAt,
	)
	if err != nil {
		return "", false, fmt.Errorf("store asset snapshot: %w", err)
	}
	changeType := explicitKind
	if changeType == "" {
		if newRecord || deleted != nil {
			changeType = "added"
		} else if !sameJSON(previous, payload) {
			changeType = "updated"
		}
	}
	if changeType != "" {
		var before any
		if !newRecord {
			before = previous
		}
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO asset_changes(
				id, asset_id, source_event_id, change_type, before_json,
				after_json, actor_type, actor_id
			) VALUES ($1, $2, $3, $4, $5, $6, 'agent', $7)`,
			uuid.NewString(),
			assetID,
			eventID,
			changeType,
			before,
			payload,
			agent.ID,
		)
		if err != nil {
			return "", false, fmt.Errorf("store asset change: %w", err)
		}
	}
	return assetID, representativeApplied, nil
}

func (s *Service) removeRecord(
	ctx context.Context,
	tx *sql.Tx,
	agent agents.Agent,
	eventID string,
	category string,
	sourceAssetID string,
	eventCreatedAt time.Time,
	eventReceivedAt time.Time,
	sourceEventID string,
) error {
	var sourceID, assetID, previous string
	var lastEventCreatedAt apitime.Time
	var lastEventReceivedAt apitime.Time
	var lastEventID string
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, asset_id, payload_json, last_event_created_at,
		        last_event_received_at, last_event_id
		 FROM asset_sources
		 WHERE agent_id = $1 AND category = $2 AND source_asset_id = $3
		   AND deleted_at IS NULL`,
		agent.ID,
		category,
		sourceAssetID,
	).Scan(
		&sourceID, &assetID, &previous, &lastEventCreatedAt,
		&lastEventReceivedAt, &lastEventID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	futureThreshold := eventReceivedAt.Add(maximumAgentClockSkew)
	poisoned := lastEventCreatedAt.Valid &&
		lastEventCreatedAt.Time.After(futureThreshold) ||
		lastEventReceivedAt.Valid && lastEventReceivedAt.Time.After(futureThreshold)
	if !poisoned && lastEventCreatedAt.Valid &&
		(eventCreatedAt.Before(lastEventCreatedAt.Time) ||
			eventCreatedAt.Equal(lastEventCreatedAt.Time) &&
				lastEventReceivedAt.Valid &&
				(eventReceivedAt.Before(lastEventReceivedAt.Time) ||
					eventReceivedAt.Equal(lastEventReceivedAt.Time) &&
						lastEventID >= sourceEventID)) {
		return nil
	}
	now := s.now()
	result, err := tx.ExecContext(
		ctx,
		`UPDATE asset_sources SET deleted_at = $1,
		 last_event_created_at = $2, last_event_received_at = $3,
		 last_event_id = $4
		 WHERE id = $5 AND deleted_at IS NULL
		   AND (last_event_created_at IS NULL OR last_event_created_at > $6 OR
		        last_event_received_at > $6 OR last_event_created_at < $2 OR
		        (last_event_created_at = $2 AND
		         (last_event_received_at IS NULL OR last_event_received_at < $3 OR
		          (last_event_received_at = $3 AND last_event_id < $4))))`,
		now,
		eventCreatedAt,
		eventReceivedAt,
		sourceEventID,
		sourceID,
		futureThreshold,
	)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return nil
	}
	var active int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM asset_sources
		 WHERE asset_id = $1 AND deleted_at IS NULL`,
		assetID,
	).Scan(&active); err != nil {
		return err
	}
	if active > 0 {
		return nil
	}
	result, err = tx.ExecContext(
		ctx,
		`UPDATE assets SET status = 'removed', deleted_at = $1,
			updated_at = $1 WHERE id = $2`,
		now,
		assetID,
	)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE asset_relations SET valid_to = $1
		 WHERE (source_asset_id = $2 OR target_asset_id = $2)
		   AND valid_to IS NULL`,
		now,
		assetID,
	); err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO asset_changes(
			id, asset_id, source_event_id, change_type, before_json,
			actor_type, actor_id, reason
		) VALUES ($1, $2, $3, 'removed', $4, 'agent', $5, 'explicit_agent_change')`,
		uuid.NewString(),
		assetID,
		eventID,
		previous,
		agent.ID,
	)
	return err
}

// classifyRecord gives one collected record its business context and, when the
// rule set says the component belongs to its host, the typed relationship that
// makes the dependency graph real.
//
// The previous behaviour created one generic 'belongs_to_host' edge for every
// record, which on a single Linux host means thousands of edges for installed
// packages alone - a graph nobody can read. Which components earn an edge is now
// a curation decision in the rule set.
func (s *Service) classifyRecord(
	ctx context.Context,
	tx *sql.Tx,
	agent agents.Agent,
	rules []classify.Rule,
	assetID string,
	record AssetRecord,
) error {
	if assetID == "" {
		return nil
	}
	var currentType, environment, criticality string
	if err := tx.QueryRowContext(
		ctx,
		"SELECT type, environment, criticality FROM assets WHERE id = $1",
		assetID,
	).Scan(&currentType, &environment, &criticality); err != nil {
		return fmt.Errorf("read asset for classification: %w", err)
	}
	// A component's environment is the environment of the machine it was read
	// from. Inferring it from the component's own name would be guessing twice,
	// and it is what let a unit called postgresql read as staging.
	if record.Category != "system" && environment == defaultEnvironment {
		inherited, err := s.hostEnvironment(ctx, tx, agent.ID)
		if err != nil {
			return err
		}
		if inherited != "" {
			environment = inherited
		}
	}
	result, err := s.classifier.ClassifyAndStore(
		ctx,
		tx,
		rules,
		classify.AssetContext{
			AssetID:     assetID,
			Category:    record.Category,
			Name:        assetName(record),
			Type:        currentType,
			Environment: environment,
			Criticality: criticality,
			Payload:     compactJSON(record.Payload),
		},
	)
	if err != nil {
		return err
	}
	if !result.RelateToHost {
		return nil
	}
	return classify.LinkToHost(
		ctx, tx, agent.ID, assetID, result.Relation, result.Confidence,
	)
}

// defaultEnvironment is the schema default, which means "nobody has decided yet".
const defaultEnvironment = "other"

// hostEnvironment reads the environment already classified onto the agent's host
// asset. Records arrive with the system inventory first, so the host is normally
// classified by the time its components are; when it is not, the next event
// completes the inheritance.
func (s *Service) hostEnvironment(
	ctx context.Context,
	tx *sql.Tx,
	agentInternalID string,
) (string, error) {
	hostAssetID, err := classify.HostAssetForAgent(ctx, tx, agentInternalID)
	if err != nil || hostAssetID == "" {
		return "", err
	}
	var environment string
	if err := tx.QueryRowContext(
		ctx,
		"SELECT environment FROM assets WHERE id = $1",
		hostAssetID,
	).Scan(&environment); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("read host environment: %w", err)
	}
	if environment == defaultEnvironment {
		return "", nil
	}
	return environment, nil
}

// proposeDuplicates surfaces two agents reporting the same machine. Cloned
// images silently double an inventory, and the merge decision belongs to a
// person, so this is recorded as a proposal.
func (s *Service) proposeDuplicates(
	ctx context.Context,
	tx *sql.Tx,
	agent agents.Agent,
	envelope Envelope,
) error {
	if envelope.Snapshot == nil {
		return nil
	}
	identifier := ""
	for _, record := range envelope.Snapshot.Records {
		if record.Category != "system" {
			continue
		}
		var payload map[string]any
		if json.Unmarshal(record.Payload, &payload) != nil {
			continue
		}
		for _, key := range []string{"machine_id", "dmi_product_uuid"} {
			if value, ok := payload[key].(string); ok && len(value) >= 16 {
				identifier = value
				break
			}
		}
		break
	}
	if identifier == "" {
		return nil
	}
	hostAssetID, err := classify.HostAssetForAgent(ctx, tx, agent.ID)
	if err != nil || hostAssetID == "" {
		return err
	}
	return classify.ProposeDuplicatesByMachineIdentity(
		ctx, tx, agent.ID, hostAssetID, identifier,
	)
}

func agentMetadata(envelope Envelope) (string, string, string) {
	if envelope.Snapshot != nil {
		for _, record := range envelope.Snapshot.Records {
			if hostname, osName, architecture, found := systemMetadata(record); found {
				return hostname, osName, architecture
			}
		}
	}
	// An upgraded Agent normally sends the newly expanded Windows system payload
	// as a delta, not another full snapshot. Reading only Snapshot left Agents
	// registered before v0.2.14 stuck with an empty os_name even though the new
	// payload had arrived successfully.
	for _, change := range envelope.Changes {
		if change.Record == nil || (change.Kind != "added" && change.Kind != "updated") {
			continue
		}
		if hostname, osName, architecture, found := systemMetadata(*change.Record); found {
			return hostname, osName, architecture
		}
	}
	return "", "", ""
}

// agentMetadataWithStoredFallback repairs agents created by older Servers.
// Their Windows system source already contains useful top-level metadata, but
// the denormalized agents row may still be blank. Heartbeats carry no records,
// so use the latest active system source only when the event has no metadata
// and the authenticated row is incomplete. Fully populated agents stay on the
// query-free hot path.
func agentMetadataWithStoredFallback(
	ctx context.Context,
	transaction *sql.Tx,
	agent agents.Agent,
	envelope Envelope,
	useEnvelopeMetadata bool,
) (string, string, string, error) {
	hostname, osName, architecture := "", "", ""
	if useEnvelopeMetadata {
		hostname, osName, architecture = agentMetadata(envelope)
	}
	if hostname != "" || osName != "" || architecture != "" {
		return hostname, osName, architecture, nil
	}
	if strings.TrimSpace(agent.Hostname) != "" &&
		strings.TrimSpace(agent.OSName) != "" &&
		strings.TrimSpace(agent.Architecture) != "" {
		return "", "", "", nil
	}
	var raw any
	err := transaction.QueryRowContext(
		ctx,
		`SELECT payload_json FROM asset_sources
		  WHERE agent_id = $1 AND category = 'system' AND deleted_at IS NULL
		  ORDER BY last_seen_at DESC, id LIMIT 1`,
		agent.ID,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", fmt.Errorf("read stored agent system metadata: %w", err)
	}
	var payload json.RawMessage
	switch value := raw.(type) {
	case string:
		payload = json.RawMessage(value)
	case []byte:
		payload = json.RawMessage(value)
	default:
		payload = json.RawMessage(fmt.Sprint(value))
	}
	fallbackHostname, fallbackOSName, fallbackArchitecture, found := systemMetadata(
		AssetRecord{Category: "system", Payload: payload},
	)
	if !found {
		return "", "", "", nil
	}
	if strings.TrimSpace(agent.Hostname) == "" {
		hostname = fallbackHostname
	}
	if strings.TrimSpace(agent.OSName) == "" {
		osName = fallbackOSName
	}
	if strings.TrimSpace(agent.Architecture) == "" {
		architecture = fallbackArchitecture
	}
	return hostname, osName, architecture, nil
}

func systemMetadata(record AssetRecord) (string, string, string, bool) {
	if record.Category != "system" {
		return "", "", "", false
	}
	var payload map[string]any
	if json.Unmarshal(record.Payload, &payload) != nil {
		return "", "", "", false
	}
	hostname, _ := payload["hostname"].(string)
	architecture, _ := payload["architecture"].(string)
	// Windows sends a canonical top-level os_name (for example
	// "Windows 11 Enterprise"), while Linux sends os_release. The old
	// Linux-only lookup silently discarded a perfectly valid Windows system
	// record, leaving an active and fully inventoried Agent displayed as
	// "operating system pending" forever.
	osName, _ := payload["os_name"].(string)
	osName = strings.TrimSpace(osName)
	if release, ok := payload["os_release"].(map[string]any); ok {
		if osName == "" {
			osName, _ = release["pretty_name"].(string)
		}
		if strings.TrimSpace(osName) == "" {
			osName, _ = release["name"].(string)
		}
	}
	if strings.TrimSpace(osName) == "" {
		// A partially readable Windows registry should still identify the
		// platform honestly instead of looking like inventory never arrived.
		if family, _ := payload["os_family"].(string); strings.EqualFold(
			strings.TrimSpace(family), "windows",
		) {
			osName = "Windows"
		}
	}
	return strings.TrimSpace(hostname), strings.TrimSpace(osName),
		strings.TrimSpace(architecture), true
}

func assetType(category string) string {
	switch category {
	case "system":
		return "host"
	case "software.package":
		return "software_package"
	case "process":
		return "process"
	case "service":
		return "service"
	case "account":
		return "account"
	case "container":
		return "container"
	case "network.interface":
		return "network_interface"
	default:
		return strings.ReplaceAll(category, ".", "_")
	}
}

func assetName(record AssetRecord) string {
	var payload map[string]any
	_ = json.Unmarshal(record.Payload, &payload)
	for _, key := range []string{
		"hostname", "name", "service", "username", "device", "mount_point",
		"interface", "id",
	} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return record.AssetID
}

func compactJSON(raw json.RawMessage) string {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return string(raw)
	}
	return buffer.String()
}

func sameJSON(left string, right string) bool {
	var leftValue, rightValue any
	if json.Unmarshal([]byte(left), &leftValue) != nil ||
		json.Unmarshal([]byte(right), &rightValue) != nil {
		return left == right
	}
	return reflect.DeepEqual(leftValue, rightValue)
}
