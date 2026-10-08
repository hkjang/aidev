package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/storage"
)

func TestLateFailureCannotOverwriteProcessedEvent(t *testing.T) {
	runtime, agent, service := testService(t)
	envelope := heartbeatEnvelope(agent.AgentID)
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DB().Exec(
		`INSERT INTO agent_events(
		 id,agent_id,event_id,schema_version,kind,snapshot_hash,raw_event,
		 processing_status,processed_at
		 ) VALUES($1,$2,$3,$4,$5,$6,$7,'processed',CURRENT_TIMESTAMP)`,
		uuid.NewString(),
		agent.ID,
		envelope.EventID,
		envelope.SchemaVersion,
		envelope.Kind,
		envelope.SnapshotHash,
		string(raw),
	); err != nil {
		t.Fatal(err)
	}
	transaction, err := runtime.DB().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("late failure from another Pod")
	if _, err := service.failEvent(
		context.Background(),
		transaction,
		uuid.NewString(),
		agent,
		envelope,
		raw,
		cause,
	); !errors.Is(err, cause) {
		t.Fatalf("failEvent() error = %v", err)
	}
	var status string
	var processingError any
	if err := runtime.DB().QueryRow(
		`SELECT processing_status,processing_error
		 FROM agent_events WHERE agent_id=$1 AND event_id=$2`,
		agent.ID,
		envelope.EventID,
	).Scan(&status, &processingError); err != nil {
		t.Fatal(err)
	}
	if status != "processed" || processingError != nil {
		t.Fatalf(
			"late failure changed completed event to %q (%v)",
			status,
			processingError,
		)
	}
}

func TestInventoryIsIdempotentAndPreservesRawEvent(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	envelope := inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host-source", "system", `{"hostname":"node-01","architecture":"x86_64"}`),
		record("pkg-source", "software.package", `{"name":"curl","version":"1"}`),
		record("nic-source", "network.interface", `{"interface":"eth0","mac":"aa:bb"}`),
	})
	raw, _ := json.Marshal(envelope)
	result, err := service.Process(
		context.Background(), agent, envelope, raw, "0.1.0",
	)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if result.Duplicate {
		t.Fatal("first event reported as duplicate")
	}
	result, err = service.Process(
		context.Background(), agent, envelope, raw, "0.1.0",
	)
	if err != nil {
		t.Fatalf("duplicate Process() error = %v", err)
	}
	if !result.Duplicate {
		t.Fatal("second event was not reported as duplicate")
	}
	assertCount(t, runtime, "agent_events", 1)
	assertCount(t, runtime, "assets", 3)
	assertCount(t, runtime, "asset_sources", 3)
	assertCount(t, runtime, "asset_changes", 3)
	// Exactly one relationship: the interface is part of its host. A plain OS
	// package earns no edge - a host with thousands of packages would otherwise
	// produce a graph nobody can read.
	assertCount(t, runtime, "asset_relations", 1)
	assertCountWhere(
		t, runtime, "asset_relations",
		"relation_type = 'part_of' AND source = 'inferred' AND status = 'active'",
		1,
	)
	// Classification runs during ingest, with the rule trail recorded.
	var hostType, packageType, classificationSource string
	var confidence float64
	if err := runtime.DB().QueryRow(
		`SELECT a.type, a.classification_source, a.classification_confidence
		   FROM assets a JOIN asset_sources s ON s.asset_id = a.id
		  WHERE s.category = 'system'`,
	).Scan(&hostType, &classificationSource, &confidence); err != nil {
		t.Fatal(err)
	}
	if hostType != "host" || classificationSource != "rule" || confidence <= 0 {
		t.Fatalf(
			"host classification = %q/%q/%v",
			hostType, classificationSource, confidence,
		)
	}
	if err := runtime.DB().QueryRow(
		`SELECT a.type FROM assets a JOIN asset_sources s ON s.asset_id = a.id
		  WHERE s.category = 'software.package'`,
	).Scan(&packageType); err != nil {
		t.Fatal(err)
	}
	if packageType != "software" {
		t.Fatalf("package type = %q, want software", packageType)
	}
	var stored string
	if err := runtime.DB().QueryRow(
		"SELECT raw_event FROM agent_events",
	).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	// Compared as a value, not as bytes. raw_event is JSONB on PostgreSQL, which
	// re-serialises what it stores: key order is normalised and whitespace
	// changes. So the retained event is semantically identical, not byte
	// identical, and anything that ever needs the agent's exact bytes - verifying
	// a signature over the payload, say - would need a TEXT or BYTEA column
	// instead. Nothing reads raw_event back today; it is written and kept.
	var storedEvent, sentEvent any
	if err := json.Unmarshal([]byte(stored), &storedEvent); err != nil {
		t.Fatalf("stored raw event is not JSON: %s", stored)
	}
	if err := json.Unmarshal(raw, &sentEvent); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(storedEvent, sentEvent) {
		t.Fatalf("raw event was changed\n got: %s\nwant: %s", stored, raw)
	}
	var hostname, architecture, version string
	if err := runtime.DB().QueryRow(
		"SELECT hostname, architecture, version FROM agents WHERE id = $1",
		agent.ID,
	).Scan(&hostname, &architecture, &version); err != nil {
		t.Fatal(err)
	}
	if hostname != "node-01" || architecture != "x86_64" || version != "0.1.0" {
		t.Fatalf(
			"agent metadata = %q/%q/%q",
			hostname,
			architecture,
			version,
		)
	}
}

func TestWindowsSystemMetadataUpdatesRegisteredAgent(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	envelope := inventoryEnvelope(agent.AgentID, []AssetRecord{
		record(
			"windows-host",
			"system",
			`{"hostname":"win-ops-01","os_family":"windows","os_name":"Windows 11 Enterprise","os_version":"24H2","os_build":"26100.4652","architecture":"x86_64"}`,
		),
	})
	processEnvelope(t, service, agent, envelope)

	var hostname, osName, architecture string
	if err := runtime.DB().QueryRow(
		"SELECT hostname, os_name, architecture FROM agents WHERE id = $1",
		agent.ID,
	).Scan(&hostname, &osName, &architecture); err != nil {
		t.Fatal(err)
	}
	if hostname != "win-ops-01" || osName != "Windows 11 Enterprise" ||
		architecture != "x86_64" {
		t.Fatalf(
			"Windows metadata = %q/%q/%q",
			hostname, osName, architecture,
		)
	}
}

func TestWindowsFamilyIsSafeMetadataFallback(t *testing.T) {
	t.Parallel()
	agentID := uuid.NewString()
	envelope := inventoryEnvelope(agentID, []AssetRecord{
		record(
			"windows-host",
			"system",
			`{"hostname":"win-core","os_family":"Windows","architecture":"aarch64"}`,
		),
	})
	hostname, osName, architecture := agentMetadata(envelope)
	if hostname != "win-core" || osName != "Windows" || architecture != "aarch64" {
		t.Fatalf("fallback metadata = %q/%q/%q", hostname, osName, architecture)
	}
}

func TestWindowsMetadataRecoversFromDeltaAfterAgentUpgrade(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	system := record(
		"windows-host",
		"system",
		`{"hostname":"win-upgraded","os_family":"windows","os_name":"Windows Server 2022 Datacenter","architecture":"x86_64"}`,
	)
	envelope := inventoryEnvelope(agent.AgentID, nil)
	envelope.Changes = []AssetChange{{
		Kind: "updated", AssetID: system.AssetID,
		Category: system.Category, Record: &system,
	}}
	processEnvelope(t, service, agent, envelope)

	var hostname, osName string
	if err := runtime.DB().QueryRow(
		"SELECT hostname, os_name FROM agents WHERE id = $1", agent.ID,
	).Scan(&hostname, &osName); err != nil {
		t.Fatal(err)
	}
	if hostname != "win-upgraded" || osName != "Windows Server 2022 Datacenter" {
		t.Fatalf("delta metadata = %q/%q", hostname, osName)
	}
}

func TestWindowsMetadataRecoversFromStoredSystemSourceOnHeartbeat(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, []AssetRecord{
		record(
			"windows-host",
			"system",
			`{"hostname":"win-existing","os_family":"windows","os_name":"Windows Server 2019 Datacenter","architecture":"x86_64"}`,
		),
	}))
	// v0.2.13 retained the raw source but failed to project these Windows fields
	// onto the agents row. The next heartbeat must repair that existing state.
	if _, err := runtime.DB().Exec(
		`UPDATE agents SET hostname = '', os_name = '', architecture = ''
		  WHERE id = $1`,
		agent.ID,
	); err != nil {
		t.Fatal(err)
	}
	legacyAgent := agent
	legacyAgent.Hostname = ""
	legacyAgent.OSName = ""
	legacyAgent.Architecture = ""
	processEnvelope(t, service, legacyAgent, heartbeatEnvelope(agent.AgentID))

	var hostname, osName, architecture string
	if err := runtime.DB().QueryRow(
		"SELECT hostname, os_name, architecture FROM agents WHERE id = $1",
		agent.ID,
	).Scan(&hostname, &osName, &architecture); err != nil {
		t.Fatal(err)
	}
	if hostname != "win-existing" || osName != "Windows Server 2019 Datacenter" ||
		architecture != "x86_64" {
		t.Fatalf("heartbeat metadata recovery = %q/%q/%q", hostname, osName, architecture)
	}
}

func TestStoredMetadataFallbackSkipsLookupForCompleteAgent(t *testing.T) {
	t.Parallel()
	hostname, osName, architecture, err := agentMetadataWithStoredFallback(
		context.Background(),
		nil, // A complete agent must return before touching the transaction.
		agents.Agent{
			Hostname: "complete-host", OSName: "Windows 11", Architecture: "x86_64",
		},
		heartbeatEnvelope(uuid.NewString()),
		false,
	)
	if err != nil || hostname != "" || osName != "" || architecture != "" {
		t.Fatalf(
			"complete agent fallback = %q/%q/%q, error = %v",
			hostname, osName, architecture, err,
		)
	}
}

func TestCollectorErrorNeverInfersRemovalAndExplicitRemovalIsLogical(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	first := inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host-source", "system", `{"hostname":"node-02"}`),
		record("pkg-source", "software.package", `{"name":"curl","version":"1"}`),
	})
	processEnvelope(t, service, agent, first)

	withError := inventoryEnvelope(agent.AgentID, nil)
	withError.CollectionErrors = []CollectionError{{
		Collector: "packages",
		Message:   "permission denied",
	}}
	processEnvelope(t, service, agent, withError)
	var active int
	if err := runtime.DB().QueryRow(
		`SELECT COUNT(*) FROM asset_sources
		 WHERE category = 'software.package' AND deleted_at IS NULL`,
	).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatal("collector error incorrectly removed an asset source")
	}
	assertCount(t, runtime, "collector_errors", 1)

	removed := inventoryEnvelope(agent.AgentID, nil)
	removed.Changes = []AssetChange{{
		Kind: "removed", AssetID: "pkg-source", Category: "software.package",
	}}
	processEnvelope(t, service, agent, removed)
	var status string
	var deleted any
	if err := runtime.DB().QueryRow(
		`SELECT a.status, a.deleted_at FROM assets a
		 JOIN asset_sources s ON s.asset_id = a.id
		 WHERE s.category = 'software.package'`,
	).Scan(&status, &deleted); err != nil {
		t.Fatal(err)
	}
	if status != "removed" || deleted == nil {
		t.Fatalf("removed asset status/deleted_at = %q/%v", status, deleted)
	}
	assertCountWhere(
		t,
		runtime,
		"asset_changes",
		"change_type = 'removed'",
		1,
	)
}

func TestUpdatedChangeKeepsAssetHistory(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	first := inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("pkg-source", "software.package", `{"name":"curl","version":"1"}`),
	})
	processEnvelope(t, service, agent, first)
	updatedRecord := record(
		"pkg-source",
		"software.package",
		`{"name":"curl","version":"2"}`,
	)
	updated := inventoryEnvelope(agent.AgentID, nil)
	updated.Changes = []AssetChange{{
		Kind: "updated", AssetID: updatedRecord.AssetID,
		Category: updatedRecord.Category, Record: &updatedRecord,
	}}
	processEnvelope(t, service, agent, updated)
	assertCount(t, runtime, "assets", 1)
	assertCount(t, runtime, "asset_snapshots", 2)
	assertCountWhere(
		t,
		runtime,
		"asset_changes",
		"change_type = 'updated'",
		1,
	)
}

func TestOlderInventoryCannotRegressSourceClassificationOrAgentMetadata(
	t *testing.T,
) {
	runtime, agent, service := testService(t)
	base := uint64(time.Now().Add(-time.Hour).Unix())
	newSystem := record(
		"system-source",
		"system",
		`{"hostname":"new-host","os_family":"windows","os_name":"Windows 11 Enterprise","architecture":"x86_64"}`,
	)
	newSystem.CollectedAt = base + 200
	newPackage := record(
		"package-source",
		"software.package",
		`{"manager":"test","name":"new-package","version":"2","architecture":"x86_64"}`,
	)
	newPackage.CollectedAt = base + 200
	newer := inventoryEnvelope(agent.AgentID, []AssetRecord{newSystem, newPackage})
	newer.CreatedAt = base + 200
	newer.Snapshot.CollectedAt = base + 200
	processEnvelopeVersion(t, service, agent, newer, "2.0.0")

	if _, err := runtime.DB().Exec(
		`UPDATE assets SET environment='production', criticality='critical',
		 custom_fields_json='{"owner":"operations"}', classification_source='manual'
		 WHERE id=(SELECT asset_id FROM asset_sources
		           WHERE source_asset_id='package-source')`,
	); err != nil {
		t.Fatal(err)
	}

	oldSystem := record(
		"system-source",
		"system",
		`{"hostname":"old-host","os_family":"windows","os_name":"Windows 7","architecture":"x86"}`,
	)
	oldSystem.CollectedAt = base + 100
	oldPackage := record(
		"package-source",
		"software.package",
		`{"manager":"test","name":"old-package","version":"1","architecture":"x86"}`,
	)
	oldPackage.CollectedAt = base + 100
	older := inventoryEnvelope(agent.AgentID, []AssetRecord{oldSystem, oldPackage})
	older.CreatedAt = base + 100
	older.Snapshot.CollectedAt = base + 100
	processEnvelopeVersion(t, service, agent, older, "1.0.0")

	var sourcePayload, assetName, environment, criticality, customFields string
	if err := runtime.DB().QueryRow(
		`SELECT s.payload_json,a.name,a.environment,a.criticality,a.custom_fields_json
		 FROM asset_sources s JOIN assets a ON a.id=s.asset_id
		 WHERE s.source_asset_id='package-source'`,
	).Scan(
		&sourcePayload, &assetName, &environment, &criticality, &customFields,
	); err != nil {
		t.Fatal(err)
	}
	if !sameJSON(sourcePayload, string(newPackage.Payload)) ||
		assetName != "new-package" || environment != "production" ||
		criticality != "critical" ||
		!sameJSON(customFields, `{"owner":"operations"}`) {
		t.Fatalf(
			"stale inventory regressed package = %s/%q/%q/%q/%s",
			sourcePayload, assetName, environment, criticality, customFields,
		)
	}
	var hostname, osName, architecture, agentVersion string
	if err := runtime.DB().QueryRow(
		`SELECT hostname,os_name,architecture,version FROM agents WHERE id=$1`,
		agent.ID,
	).Scan(&hostname, &osName, &architecture, &agentVersion); err != nil {
		t.Fatal(err)
	}
	if hostname != "new-host" || osName != "Windows 11 Enterprise" ||
		architecture != "x86_64" || agentVersion != "2.0.0" {
		t.Fatalf(
			"stale metadata = %q/%q/%q/%q",
			hostname, osName, architecture, agentVersion,
		)
	}
	assertCount(t, runtime, "asset_snapshots", 2)

	staleRemoval := inventoryEnvelope(agent.AgentID, nil)
	staleRemoval.CreatedAt = base + 150
	staleRemoval.Changes = []AssetChange{{
		Kind: "removed", AssetID: "package-source", Category: "software.package",
	}}
	processEnvelope(t, service, agent, staleRemoval)
	assertCountWhere(
		t, runtime, "asset_sources",
		"source_asset_id='package-source' AND deleted_at IS NULL", 1,
	)
	assertCountWhere(t, runtime, "asset_changes", "change_type='removed'", 0)
}

func TestExactAgentAndServerTimestampTieUsesEventIDDeterministically(
	t *testing.T,
) {
	runtime, agent, service := testService(t)
	// SQLite CURRENT_TIMESTAMP is second-granular, but a slow CI runner could
	// still cross a boundary. Pin the DB receive clock so this is an exact tuple
	// tie and verifies the final deterministic event_id comparison itself.
	//
	// Written for both engines rather than skipped on one: the tie-break this
	// checks is ordering logic that has to hold wherever the server runs, and a
	// test pinned to SQLite is a test PostgreSQL never checks.
	pinReceivedAt := []string{
		`CREATE TRIGGER pin_agent_event_received_at
		 AFTER INSERT ON agent_events
		 BEGIN
		   UPDATE agent_events SET received_at='2030-01-01 00:00:00'
		   WHERE id=NEW.id;
		 END`,
	}
	if storagetest.Postgres() {
		pinReceivedAt = []string{
			`CREATE FUNCTION pin_agent_event_received_at() RETURNS TRIGGER AS $$
			 BEGIN
			   NEW.received_at := TIMESTAMPTZ '2030-01-01 00:00:00+00';
			   RETURN NEW;
			 END;
			 $$ LANGUAGE plpgsql`,
			`CREATE TRIGGER pin_agent_event_received_at
			 BEFORE INSERT ON agent_events
			 FOR EACH ROW EXECUTE FUNCTION pin_agent_event_received_at()`,
		}
	}
	for _, statement := range pinReceivedAt {
		if _, err := runtime.DB().Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	timestamp := uint64(time.Now().Add(-time.Minute).Unix())
	highID := "ffffffff-ffff-4fff-bfff-ffffffffffff"
	lowID := "00000000-0000-4000-8000-000000000001"
	firstRecord := record(
		"system-source",
		"system",
		`{"hostname":"tie-old","os_family":"windows","os_name":"Windows 7","architecture":"x86"}`,
	)
	firstRecord.CollectedAt = timestamp
	first := inventoryEnvelope(agent.AgentID, []AssetRecord{firstRecord})
	first.EventID = lowID
	first.CreatedAt = timestamp
	first.Snapshot.CollectedAt = timestamp
	processEnvelopeVersion(t, service, agent, first, "1.0.0")

	winnerRecord := record(
		"system-source",
		"system",
		`{"hostname":"tie-new","os_family":"windows","os_name":"Windows 11","architecture":"x86_64"}`,
	)
	winnerRecord.CollectedAt = timestamp
	winner := inventoryEnvelope(agent.AgentID, []AssetRecord{winnerRecord})
	winner.EventID = highID
	winner.CreatedAt = timestamp
	winner.Snapshot.CollectedAt = timestamp
	processEnvelopeVersion(t, service, agent, winner, "9.0.0")

	var payload, hostname, version, lastEventID string
	if err := runtime.DB().QueryRow(
		`SELECT s.payload_json,a.hostname,a.version,a.last_event_id
		 FROM asset_sources s JOIN agents a ON a.id=s.agent_id
		 WHERE s.source_asset_id='system-source'`,
	).Scan(&payload, &hostname, &version, &lastEventID); err != nil {
		t.Fatal(err)
	}
	if !sameJSON(payload, string(winnerRecord.Payload)) || hostname != "tie-new" ||
		version != "9.0.0" || lastEventID != highID {
		t.Fatalf(
			"event ID did not deterministically resolve exact tuple tie = %s/%q/%q/%q",
			payload,
			hostname,
			version,
			lastEventID,
		)
	}

	staleRemoval := inventoryEnvelope(agent.AgentID, nil)
	staleRemoval.EventID = lowID[:len(lowID)-1] + "2"
	staleRemoval.CreatedAt = timestamp
	staleRemoval.Changes = []AssetChange{{
		Kind: "removed", AssetID: "system-source", Category: "system",
	}}
	processEnvelope(t, service, agent, staleRemoval)
	assertCountWhere(
		t,
		runtime,
		"asset_sources",
		"source_asset_id='system-source' AND deleted_at IS NULL",
		1,
	)
}

func TestFutureAgentClockIsClampedDiagnosedAndPoisonedWatermarkHeals(
	t *testing.T,
) {
	runtime, agent, service := testService(t)
	future := ^uint64(0)
	futureRecord := record(
		"system-source",
		"system",
		`{"hostname":"future-host","os_name":"Future OS","architecture":"future"}`,
	)
	futureRecord.CollectedAt = future
	futureEnvelope := inventoryEnvelope(agent.AgentID, []AssetRecord{futureRecord})
	futureEnvelope.CreatedAt = future
	futureEnvelope.Snapshot.CollectedAt = future
	processEnvelopeVersion(t, service, agent, futureEnvelope, "future-version")
	assertCountWhere(
		t,
		runtime,
		"agent_event_errors",
		"error_code='FUTURE_TIMESTAMP_CLAMPED'",
		1,
	)

	poison := time.Now().Add(365 * 24 * time.Hour).UTC()
	if _, err := runtime.DB().Exec(
		`UPDATE asset_sources SET collected_at=$1,last_seen_at=$1,
		 last_event_created_at=$1,last_event_received_at=$1
		 WHERE source_asset_id='system-source'`,
		poison,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DB().Exec(
		`UPDATE assets SET last_seen_at=$1 WHERE asset_key LIKE '%:system:system-source'`,
		poison,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.DB().Exec(
		`UPDATE agents SET hostname='poison-host',version='poison-version',
		 last_inventory_at=$1,last_event_created_at=$1,last_event_received_at=$1
		 WHERE id=$2`,
		poison,
		agent.ID,
	); err != nil {
		t.Fatal(err)
	}

	normalRecord := record(
		"system-source",
		"system",
		`{"hostname":"healed-host","os_name":"Windows 11","architecture":"x86_64"}`,
	)
	normal := inventoryEnvelope(agent.AgentID, []AssetRecord{normalRecord})
	processEnvelopeVersion(t, service, agent, normal, "healed-version")

	var payload, hostname, version string
	var collectedAt, sourceEventAt, agentEventAt apitime.Time
	if err := runtime.DB().QueryRow(
		`SELECT s.payload_json,s.collected_at,s.last_event_created_at,
		        a.hostname,a.version,a.last_event_created_at
		 FROM asset_sources s JOIN agents a ON a.id=s.agent_id
		 WHERE s.source_asset_id='system-source'`,
	).Scan(
		&payload,
		&collectedAt,
		&sourceEventAt,
		&hostname,
		&version,
		&agentEventAt,
	); err != nil {
		t.Fatal(err)
	}
	threshold := time.Now().Add(maximumAgentClockSkew + time.Minute)
	if !sameJSON(payload, string(normalRecord.Payload)) || hostname != "healed-host" ||
		version != "healed-version" || !collectedAt.Valid ||
		collectedAt.Time.After(threshold) || !sourceEventAt.Valid ||
		sourceEventAt.Time.After(threshold) || !agentEventAt.Valid ||
		agentEventAt.Time.After(threshold) {
		t.Fatalf(
			"poisoned clock did not heal = %s/%q/%q/%v/%v/%v",
			payload,
			hostname,
			version,
			collectedAt,
			sourceEventAt,
			agentEventAt,
		)
	}
}

func TestCanonicalPackageIDsPreserveLegacyAssetIdentityAndManualMetadata(
	t *testing.T,
) {
	testCases := []struct {
		name             string
		legacyID         string
		canonicalID      string
		legacyPayload    string
		canonicalPayload string
	}{
		{
			name:             "RPM version instance",
			legacyID:         "package:rpm:kernel:x86_64",
			canonicalID:      "package:rpm:kernel:x86_64:0:5.14.0-503.el9_5",
			legacyPayload:    `{"manager":"rpm","name":"kernel","version":"0:5.14.0-503.el9_5","architecture":"x86_64"}`,
			canonicalPayload: `{"manager":"rpm","name":"kernel","version":"0:5.14.0-503.el9_5","architecture":"x86_64"}`,
		},
		{
			name:             "Windows registry instance without legacy owner SID",
			legacyID:         "package:windows:Contoso Agent:x64",
			canonicalID:      "package:windows:x64:user:s-1-5-21-1000:{contoso-agent}",
			legacyPayload:    `{"manager":"windows","name":"Contoso Agent","version":"4.2","architecture":"x64","scope":"user","registry_key":"{CONTOSO-AGENT}"}`,
			canonicalPayload: `{"manager":"windows","name":"Contoso Agent","version":"4.2","architecture":"x64","scope":"user","owner_sid":"S-1-5-21-1000","registry_key":"{CONTOSO-AGENT}"}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			runtime, agent, service := testService(t)
			base := uint64(time.Now().Add(-time.Hour).Unix())
			legacy := record(
				testCase.legacyID, "software.package", testCase.legacyPayload,
			)
			legacy.CollectedAt = base + 100
			first := inventoryEnvelope(agent.AgentID, []AssetRecord{legacy})
			first.CreatedAt = base + 100
			first.Snapshot.CollectedAt = base + 100
			processEnvelope(t, service, agent, first)

			var sourceID, assetID string
			if err := runtime.DB().QueryRow(
				`SELECT id,asset_id FROM asset_sources WHERE source_asset_id=$1`,
				testCase.legacyID,
			).Scan(&sourceID, &assetID); err != nil {
				t.Fatal(err)
			}
			if _, err := runtime.DB().Exec(
				`UPDATE assets SET environment='production',criticality='critical',
				 custom_fields_json='{"owner":"asset-team"}',
				 classification_source='manual' WHERE id=$1`,
				assetID,
			); err != nil {
				t.Fatal(err)
			}

			canonical := record(
				testCase.canonicalID, "software.package", testCase.canonicalPayload,
			)
			canonical.CollectedAt = base + 200
			upgrade := inventoryEnvelope(agent.AgentID, nil)
			upgrade.CreatedAt = base + 200
			upgrade.Changes = []AssetChange{
				{Kind: "removed", AssetID: testCase.legacyID, Category: "software.package"},
				{
					Kind: "added", AssetID: canonical.AssetID,
					Category: canonical.Category, Record: &canonical,
				},
			}
			processEnvelope(t, service, agent, upgrade)

			assertCount(t, runtime, "assets", 1)
			assertCount(t, runtime, "asset_sources", 1)
			var gotSourceID, gotAssetID, sourceAssetID, assetKey string
			var environment, criticality, customFields string
			if err := runtime.DB().QueryRow(
				`SELECT s.id,s.asset_id,s.source_asset_id,a.asset_key,
				        a.environment,a.criticality,a.custom_fields_json
				 FROM asset_sources s JOIN assets a ON a.id=s.asset_id`,
			).Scan(
				&gotSourceID, &gotAssetID, &sourceAssetID, &assetKey,
				&environment, &criticality, &customFields,
			); err != nil {
				t.Fatal(err)
			}
			wantAssetKey := agent.AgentID + ":software.package:" + testCase.canonicalID
			if gotSourceID != sourceID || gotAssetID != assetID ||
				sourceAssetID != testCase.canonicalID || assetKey != wantAssetKey ||
				environment != "production" || criticality != "critical" ||
				!sameJSON(customFields, `{"owner":"asset-team"}`) {
				t.Fatalf(
					"legacy migration = %q/%q/%q/%q/%q/%q/%s",
					gotSourceID, gotAssetID, sourceAssetID, assetKey,
					environment, criticality, customFields,
				)
			}
			assertCountWhere(
				t, runtime, "asset_changes",
				"asset_id='"+assetID+"' AND change_type='removed'", 0,
			)
		})
	}
}

func TestLegacyPackageAliasRequiresMatchingStoredIdentity(t *testing.T) {
	runtime, agent, service := testService(t)
	legacy := record(
		"package:windows:Contoso Agent:x64",
		"software.package",
		`{"manager":"windows","name":"Contoso Agent","version":"4.2","architecture":"x64","scope":"machine","registry_key":"{OLD-KEY}"}`,
	)
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, []AssetRecord{legacy}))
	canonical := record(
		"package:windows:x64:machine::{new-key}",
		"software.package",
		`{"manager":"windows","name":"Contoso Agent","version":"4.2","architecture":"x64","scope":"machine","owner_sid":"","registry_key":"{NEW-KEY}"}`,
	)
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, []AssetRecord{canonical}))
	assertCount(t, runtime, "assets", 2)
	assertCount(t, runtime, "asset_sources", 2)
}

func TestLegacyWindowsPackageWithoutOwnerSIDRejectsAmbiguousAdoption(
	t *testing.T,
) {
	runtime, agent, service := testService(t)
	base := uint64(time.Now().Add(-time.Hour).Unix())
	legacy := record(
		"package:windows:Contoso User Tool:x64",
		"software.package",
		`{"manager":"windows","name":"Contoso User Tool","version":"4.2","architecture":"x64","scope":"user","registry_key":"{SHARED-KEY}"}`,
	)
	legacy.CollectedAt = base + 100
	first := inventoryEnvelope(agent.AgentID, []AssetRecord{legacy})
	first.CreatedAt = base + 100
	first.Snapshot.CollectedAt = base + 100
	processEnvelope(t, service, agent, first)

	canonical := make([]AssetRecord, 0, 2)
	for _, sid := range []string{"s-1-5-21-1000", "s-1-5-21-2000"} {
		record := record(
			"package:windows:x64:user:"+sid+":{shared-key}",
			"software.package",
			fmt.Sprintf(
				`{"manager":"windows","name":"Contoso User Tool","version":"4.2","architecture":"x64","scope":"user","owner_sid":%q,"registry_key":"{SHARED-KEY}"}`,
				strings.ToUpper(sid),
			),
		)
		record.CollectedAt = base + 200
		canonical = append(canonical, record)
	}
	upgrade := inventoryEnvelope(agent.AgentID, nil)
	upgrade.CreatedAt = base + 200
	upgrade.Changes = []AssetChange{{
		Kind: "removed", AssetID: legacy.AssetID, Category: legacy.Category,
	}}
	for index := range canonical {
		upgrade.Changes = append(upgrade.Changes, AssetChange{
			Kind: "added", AssetID: canonical[index].AssetID,
			Category: canonical[index].Category, Record: &canonical[index],
		})
	}
	processEnvelope(t, service, agent, upgrade)

	assertCount(t, runtime, "assets", 3)
	assertCount(t, runtime, "asset_sources", 3)
	assertCountWhere(
		t,
		runtime,
		"asset_sources",
		"source_asset_id='package:windows:Contoso User Tool:x64' AND deleted_at IS NOT NULL",
		1,
	)
}

func TestLegacyWindowsNativeAndX86InstancesPreserveSeparateIdentityAndHistory(
	t *testing.T,
) {
	runtime, agent, service := testService(t)
	base := uint64(time.Now().Add(-time.Hour).Unix())
	type packageCase struct {
		architecture string
		scope        string
		owner        string
	}
	packages := []packageCase{
		{architecture: "x64", scope: "machine", owner: "native-team"},
		{architecture: "x86", scope: "machine-x86", owner: "x86-team"},
	}
	legacyRecords := make([]AssetRecord, 0, len(packages))
	for _, item := range packages {
		legacy := record(
			"package:windows:Contoso Agent:"+item.architecture,
			"software.package",
			fmt.Sprintf(
				`{"manager":"windows","name":"Contoso Agent","version":"7.0","architecture":%q,"scope":%q,"registry_key":"{SHARED-KEY}"}`,
				item.architecture,
				item.scope,
			),
		)
		legacy.CollectedAt = base + 100
		legacyRecords = append(legacyRecords, legacy)
	}
	first := inventoryEnvelope(agent.AgentID, legacyRecords)
	first.CreatedAt = base + 100
	first.Snapshot.CollectedAt = base + 100
	processEnvelope(t, service, agent, first)

	type preserved struct {
		sourceID string
		assetID  string
		history  int
	}
	preservedByArchitecture := make(map[string]preserved, len(packages))
	for _, item := range packages {
		legacyID := "package:windows:Contoso Agent:" + item.architecture
		var saved preserved
		if err := runtime.DB().QueryRow(
			`SELECT s.id,s.asset_id,
			        (SELECT COUNT(*) FROM asset_changes c WHERE c.asset_id=s.asset_id)
			 FROM asset_sources s WHERE s.source_asset_id=$1`,
			legacyID,
		).Scan(&saved.sourceID, &saved.assetID, &saved.history); err != nil {
			t.Fatal(err)
		}
		if saved.history == 0 {
			t.Fatalf("legacy %s asset has no history to preserve", item.architecture)
		}
		if _, err := runtime.DB().Exec(
			`UPDATE assets SET environment='production',criticality='critical',
			 custom_fields_json=$1,classification_source='manual' WHERE id=$2`,
			fmt.Sprintf(`{"owner":%q}`, item.owner),
			saved.assetID,
		); err != nil {
			t.Fatal(err)
		}
		preservedByArchitecture[item.architecture] = saved
	}

	upgrade := inventoryEnvelope(agent.AgentID, nil)
	upgrade.CreatedAt = base + 200
	for _, item := range packages {
		legacyID := "package:windows:Contoso Agent:" + item.architecture
		canonicalID := fmt.Sprintf(
			"package:windows:%s:%s::{shared-key}",
			item.architecture,
			item.scope,
		)
		canonical := record(
			canonicalID,
			"software.package",
			fmt.Sprintf(
				`{"manager":"windows","name":"Contoso Agent","version":"7.0","architecture":%q,"scope":%q,"owner_sid":"","registry_key":"{SHARED-KEY}"}`,
				item.architecture,
				item.scope,
			),
		)
		canonical.CollectedAt = base + 200
		upgrade.Changes = append(upgrade.Changes,
			AssetChange{Kind: "removed", AssetID: legacyID, Category: "software.package"},
			AssetChange{
				Kind: "added", AssetID: canonicalID,
				Category: "software.package", Record: &canonical,
			},
		)
	}
	processEnvelope(t, service, agent, upgrade)

	assertCount(t, runtime, "assets", 2)
	assertCount(t, runtime, "asset_sources", 2)
	for _, item := range packages {
		canonicalID := fmt.Sprintf(
			"package:windows:%s:%s::{shared-key}",
			item.architecture,
			item.scope,
		)
		want := preservedByArchitecture[item.architecture]
		var sourceID, assetID, environment, criticality, customFields string
		var history, removedHistory int
		if err := runtime.DB().QueryRow(
			`SELECT s.id,s.asset_id,a.environment,a.criticality,a.custom_fields_json,
			        (SELECT COUNT(*) FROM asset_changes c WHERE c.asset_id=a.id),
			        (SELECT COUNT(*) FROM asset_changes c
			          WHERE c.asset_id=a.id AND c.change_type='removed')
			 FROM asset_sources s JOIN assets a ON a.id=s.asset_id
			 WHERE s.source_asset_id=$1`,
			canonicalID,
		).Scan(
			&sourceID,
			&assetID,
			&environment,
			&criticality,
			&customFields,
			&history,
			&removedHistory,
		); err != nil {
			t.Fatal(err)
		}
		if sourceID != want.sourceID || assetID != want.assetID ||
			environment != "production" || criticality != "critical" ||
			!sameJSON(customFields, fmt.Sprintf(`{"owner":%q}`, item.owner)) ||
			history < want.history || removedHistory != 0 {
			t.Fatalf(
				"%s migration lost identity/metadata/history: %q/%q/%q/%q/%s/%d/%d",
				item.architecture,
				sourceID,
				assetID,
				environment,
				criticality,
				customFields,
				history,
				removedHistory,
			)
		}
	}
}

func TestFirstInventoryPromotesEnrollmentHostWithoutDuplicateAsset(
	t *testing.T,
) {
	t.Parallel()
	runtime := storagetest.Open(t)
	defer runtime.Close()
	agentService := agents.NewService(runtime.DB())
	externalID := uuid.NewString()
	provisioned, err := agentService.AutoEnroll(
		context.Background(),
		externalID,
		"enrollment-name",
		"ivq_ec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"10.50.60.70",
	)
	if err != nil {
		t.Fatal(err)
	}
	assertCount(t, runtime, "assets", 1)
	envelope := inventoryEnvelope(externalID, []AssetRecord{
		record(
			"host-source",
			"system",
			`{"hostname":"inventory-name","architecture":"x86_64"}`,
		),
		record(
			"pkg-source",
			"software.package",
			`{"name":"curl","version":"8"}`,
		),
	})
	processEnvelope(
		t,
		NewService(runtime.DB()),
		provisioned.Agent,
		envelope,
	)
	assertCount(t, runtime, "assets", 2)
	assertCount(t, runtime, "asset_sources", 3)
	var name, status, source string
	var sourceCount int
	if err := runtime.DB().QueryRow(
		`SELECT a.name,a.status,a.source,COUNT(s.id)
		 FROM assets a
		 JOIN asset_sources s ON s.asset_id=a.id
		 WHERE a.asset_key=$1
		 GROUP BY a.name,a.status,a.source`,
		"agent:"+externalID,
	).Scan(&name, &status, &source, &sourceCount); err != nil {
		t.Fatal(err)
	}
	if name != "inventory-name" || status != "active" ||
		source != "agent" || sourceCount != 2 {
		t.Fatalf(
			"promoted asset = %q/%q/%q with %d sources",
			name,
			status,
			source,
			sourceCount,
		)
	}
}

func TestEnvelopeValidationRejectsIdentityAndHeartbeatPayload(t *testing.T) {
	t.Parallel()
	valid := inventoryEnvelope(uuid.NewString(), nil)
	valid.EventID = "not-a-uuid"
	if err := valid.Validate(); err == nil {
		t.Fatal("invalid event UUID was accepted")
	}
	valid = inventoryEnvelope(uuid.NewString(), nil)
	valid.Kind = "heartbeat"
	valid.Changes = []AssetChange{{
		Kind: "removed", AssetID: "x", Category: "process",
	}}
	if err := valid.Validate(); err == nil {
		t.Fatal("heartbeat with changes was accepted")
	}
}

func testService(t *testing.T) (*storage.Runtime, agents.Agent, *Service) {
	t.Helper()
	runtime := storagetest.Open(t)
	t.Cleanup(func() { runtime.Close() })
	agentService := agents.NewService(runtime.DB())
	provisioned, err := agentService.ProvisionBearer(
		context.Background(), uuid.NewString(), "test-host", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	return runtime, provisioned.Agent, NewService(runtime.DB())
}

func inventoryEnvelope(agentID string, records []AssetRecord) Envelope {
	now := nextTestEventTimestamp()
	var snapshot *Snapshot
	if records != nil {
		snapshot = &Snapshot{
			SchemaVersion: 1,
			AgentID:       agentID,
			CollectedAt:   now,
			Records:       records,
		}
	}
	return Envelope{
		SchemaVersion: 1,
		EventID:       uuid.NewString(),
		AgentID:       agentID,
		CreatedAt:     now,
		Kind:          "inventory",
		SnapshotHash:  uuid.NewString(),
		Snapshot:      snapshot,
	}
}

var testEventClock struct {
	sync.Mutex
	last uint64
}

func nextTestEventTimestamp() uint64 {
	testEventClock.Lock()
	defer testEventClock.Unlock()
	now := uint64(time.Now().Unix())
	if now <= testEventClock.last {
		now = testEventClock.last + 1
	}
	testEventClock.last = now
	return now
}

func record(id string, category string, payload string) AssetRecord {
	return AssetRecord{
		AssetID: id, Category: category, Source: "test",
		CollectedAt: uint64(time.Now().Unix()), Payload: json.RawMessage(payload),
	}
}

func processEnvelope(
	t *testing.T,
	service *Service,
	agent agents.Agent,
	envelope Envelope,
) {
	processEnvelopeVersion(t, service, agent, envelope, "test")
}

func processEnvelopeVersion(
	t *testing.T,
	service *Service,
	agent agents.Agent,
	envelope Envelope,
	agentVersion string,
) {
	t.Helper()
	if err := envelope.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(envelope)
	if _, err := service.Process(
		context.Background(), agent, envelope, raw, agentVersion,
	); err != nil {
		t.Fatalf("Process() error = %v", err)
	}
}

func assertCount(
	t *testing.T,
	runtime *storage.Runtime,
	table string,
	want int,
) {
	t.Helper()
	var got int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM " + table,
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}

func assertCountWhere(
	t *testing.T,
	runtime *storage.Runtime,
	table string,
	condition string,
	want int,
) {
	t.Helper()
	var got int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM " + table + " WHERE " + condition,
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
