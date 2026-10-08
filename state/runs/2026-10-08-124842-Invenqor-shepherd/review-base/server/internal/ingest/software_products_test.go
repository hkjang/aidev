package ingest

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/softwarecatalog"
)

func TestFirstHeartbeatBackfillsExistingRawSoftwareEvidenceOnce(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host", "system", `{"hostname":"legacy-proxy-01"}`),
		record("nginx-process", "process", `{"pid":55,"name":"nginx","executable":"/usr/sbin/nginx"}`),
		record("nginx-package", "software.package", `{"name":"nginx","version":"1.26.1"}`),
	}))

	// Recreate the database state left by v0.2.13: raw observations exist, but
	// normalized catalogue products and the v0.2.14 marker do not.
	for _, statement := range []string{
		`DELETE FROM asset_changes WHERE reason = 'automatic_software_catalog'`,
		`DELETE FROM asset_relations WHERE source_asset_id IN (
			SELECT asset_id FROM asset_sources WHERE category = 'software.product'
		)`,
		`DELETE FROM software_product_inventory WHERE agent_id = '` + agent.ID + `'`,
		`DELETE FROM asset_sources WHERE agent_id = '` + agent.ID + `' AND category = 'software.product'`,
		`DELETE FROM assets WHERE type = 'software_product'`,
		`DELETE FROM software_catalog_reconciliations WHERE agent_id = '` + agent.ID + `'`,
	} {
		if _, err := runtime.DB().Exec(statement); err != nil {
			t.Fatalf("prepare legacy inventory with %q: %v", statement, err)
		}
	}

	firstReconciledAt := time.Date(2026, 8, 21, 11, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return firstReconciledAt }
	processEnvelope(t, service, agent, heartbeatEnvelope(agent.AgentID))
	product := readProductDetection(t, runtime.DB(), agent.ID, "nginx")
	if product.Version != "1.26.1" || product.RuntimeState != "running" {
		t.Fatalf("heartbeat backfill product = %+v", product)
	}
	assertCountWhere(t, runtime, "software_product_inventory", "agent_id = '"+agent.ID+"'", 1)
	assertCountWhere(
		t, runtime, "asset_changes",
		"reason = 'automatic_software_catalog'", 1,
	)
	var version, firstMarker string
	if err := runtime.DB().QueryRow(
		`SELECT catalog_version, CAST(reconciled_at AS TEXT)
		   FROM software_catalog_reconciliations WHERE agent_id = $1`,
		agent.ID,
	).Scan(&version, &firstMarker); err != nil {
		t.Fatal(err)
	}
	if version != softwarecatalog.CatalogVersion || firstMarker == "" {
		t.Fatalf("backfill marker = %q/%q", version, firstMarker)
	}

	service.now = func() time.Time { return firstReconciledAt.Add(time.Hour) }
	processEnvelope(t, service, agent, heartbeatEnvelope(agent.AgentID))
	var secondMarker string
	if err := runtime.DB().QueryRow(
		`SELECT CAST(reconciled_at AS TEXT)
		   FROM software_catalog_reconciliations WHERE agent_id = $1`,
		agent.ID,
	).Scan(&secondMarker); err != nil {
		t.Fatal(err)
	}
	if secondMarker != firstMarker {
		t.Fatalf("second heartbeat repeated reconciliation: %q -> %q", firstMarker, secondMarker)
	}
	assertCountWhere(
		t, runtime, "asset_changes",
		"reason = 'automatic_software_catalog'", 1,
	)
}

func TestHeartbeatMarksZeroDetectionHostWithoutRepeatingScan(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	firstReconciledAt := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return firstReconciledAt }
	processEnvelope(t, service, agent, heartbeatEnvelope(agent.AgentID))
	assertCountWhere(t, runtime, "software_product_inventory", "agent_id = '"+agent.ID+"'", 0)

	var version, firstMarker string
	if err := runtime.DB().QueryRow(
		`SELECT catalog_version, CAST(reconciled_at AS TEXT)
		   FROM software_catalog_reconciliations WHERE agent_id = $1`,
		agent.ID,
	).Scan(&version, &firstMarker); err != nil {
		t.Fatal(err)
	}
	if version != softwarecatalog.CatalogVersion || firstMarker == "" {
		t.Fatalf("zero-detection marker = %q/%q", version, firstMarker)
	}
	service.now = func() time.Time { return firstReconciledAt.Add(time.Hour) }
	processEnvelope(t, service, agent, heartbeatEnvelope(agent.AgentID))
	var secondMarker string
	if err := runtime.DB().QueryRow(
		`SELECT CAST(reconciled_at AS TEXT)
		   FROM software_catalog_reconciliations WHERE agent_id = $1`,
		agent.ID,
	).Scan(&secondMarker); err != nil {
		t.Fatal(err)
	}
	if secondMarker != firstMarker {
		t.Fatalf("zero-detection heartbeat repeated scan: %q -> %q", firstMarker, secondMarker)
	}
}

func TestIngestCreatesOneHostScopedProductFromMultipleEvidenceKinds(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	records := []AssetRecord{
		record("host", "system", `{"hostname":"db-prd-01"}`),
		record("postgres-101", "process", `{"pid":101,"name":"postgres","executable":"/usr/lib/postgresql/16/bin/postgres"}`),
		record("postgres-102", "process", `{"pid":102,"name":"postgres","executable":"/usr/lib/postgresql/16/bin/postgres"}`),
		record("postgres-service", "service", `{"name":"postgresql@16-main.service","active_state":"active","sub_state":"running"}`),
		record("postgres-package", "software.package", `{"name":"postgresql-16","version":"16.4"}`),
		record("iis-service", "service", `{"name":"W3SVC","display_name":"World Wide Web Publishing Service","state":"stopped","active":false}`),
		record("ordinary-process", "process", `{"pid":700,"name":"explorer.exe","executable":"C:\\Windows\\explorer.exe"}`),
	}
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, records))

	type productRow struct {
		name, assetType, environment, source, attributes string
		confidence                                       float64
	}
	rows, err := runtime.DB().Query(
		`SELECT a.name, a.type, a.environment, a.source, a.confidence,
		        a.attributes_json
		   FROM assets a
		   JOIN asset_sources s ON s.asset_id = a.id
		  WHERE s.agent_id = $1 AND s.category = 'software.product'
		    AND s.deleted_at IS NULL
		  ORDER BY a.name`,
		agent.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	products := map[string]productRow{}
	for rows.Next() {
		var row productRow
		if err := rows.Scan(
			&row.name, &row.assetType, &row.environment, &row.source,
			&row.confidence, &row.attributes,
		); err != nil {
			t.Fatal(err)
		}
		products[row.name] = row
	}
	if len(products) != 2 {
		t.Fatalf("normalized products = %+v, want PostgreSQL and IIS only", products)
	}
	postgres := products["PostgreSQL"]
	if postgres.assetType != "software_product" || postgres.source != "inferred" ||
		postgres.environment != "production" || postgres.confidence != 0.99 {
		t.Fatalf("PostgreSQL row = %+v", postgres)
	}
	var attributes softwarecatalog.Detection
	if err := json.Unmarshal([]byte(postgres.attributes), &attributes); err != nil {
		t.Fatal(err)
	}
	if attributes.ProductKey != "postgresql" || attributes.Role != "database" ||
		attributes.Version != "16.4" || attributes.InstallState != "installed" ||
		attributes.RuntimeState != "running" || attributes.ProcessCount != 2 ||
		attributes.EvidenceCount != 4 || attributes.Detection != "builtin_catalog" {
		t.Fatalf("PostgreSQL attributes = %+v", attributes)
	}
	if len(attributes.ProcessNames) != 1 || attributes.ProcessNames[0] != "postgres" ||
		len(attributes.ServiceNames) != 1 || len(attributes.PackageNames) != 1 {
		t.Fatalf("normalized evidence names = %+v", attributes)
	}
	var projected struct {
		Name, Role, Vendor, Version, Install, Runtime, Catalog, Search string
		Confidence                                                     float64
		Processes, Evidence                                            int
	}
	if err := runtime.DB().QueryRow(
		`SELECT product_name, role, vendor, version, install_state,
		        runtime_state, confidence, process_count, evidence_count,
		        catalog_version, search_text
		   FROM software_product_inventory
		  WHERE agent_id = $1 AND product_key = 'postgresql'`,
		agent.ID,
	).Scan(
		&projected.Name, &projected.Role, &projected.Vendor, &projected.Version,
		&projected.Install, &projected.Runtime, &projected.Confidence,
		&projected.Processes, &projected.Evidence, &projected.Catalog,
		&projected.Search,
	); err != nil {
		t.Fatal(err)
	}
	if projected.Name != "PostgreSQL" || projected.Role != "database" ||
		projected.Version != "16.4" || projected.Install != "installed" ||
		projected.Runtime != "running" || projected.Confidence != 0.99 ||
		projected.Processes != 2 || projected.Evidence != 4 ||
		projected.Catalog != softwarecatalog.CatalogVersion ||
		!strings.Contains(projected.Search, "db-prd-01") ||
		!strings.Contains(projected.Search, "postgresql@16-main.service") {
		t.Fatalf("software projection = %+v", projected)
	}

	var productRelations int
	if err := runtime.DB().QueryRow(
		`SELECT COUNT(*) FROM asset_relations r
		   JOIN assets a ON a.id = r.source_asset_id
		  WHERE a.type = 'software_product' AND r.relation_type = 'runs_on'
		    AND r.status = 'active' AND r.valid_to IS NULL`,
	).Scan(&productRelations); err != nil {
		t.Fatal(err)
	}
	if productRelations != 2 {
		t.Fatalf("active product relations = %d, want 2", productRelations)
	}

	// A new event with the same evidence refreshes timestamps but creates no
	// duplicate product, source, relation, or misleading change entry.
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, records))
	assertCountWhere(t, runtime, "assets", "type = 'software_product'", 2)
	assertCountWhere(t, runtime, "asset_sources", "category = 'software.product'", 2)
	assertCountWhere(t, runtime, "software_product_inventory", "agent_id = '"+agent.ID+"'", 2)
	assertCountWhere(
		t, runtime, "asset_changes",
		"reason = 'automatic_software_catalog'", 2,
	)
}

func TestProductReconciliationDropsStaleRuntimeEvidenceAndReactivates(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	first := inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host", "system", `{"hostname":"proxy-01"}`),
		record("nginx-process", "process", `{"pid":55,"name":"nginx","executable":"/usr/sbin/nginx"}`),
		record("nginx-package", "software.package", `{"name":"nginx","version":"1.26.1"}`),
	})
	processEnvelope(t, service, agent, first)

	removedProcess := inventoryEnvelope(agent.AgentID, nil)
	removedProcess.Changes = []AssetChange{{
		Kind: "removed", AssetID: "nginx-process", Category: "process",
	}}
	processEnvelope(t, service, agent, removedProcess)
	product := readProductDetection(t, runtime.DB(), agent.ID, "nginx")
	if product.RuntimeState != "unknown" || product.InstallState != "installed" ||
		product.ProcessCount != 0 || len(product.ProcessNames) != 0 {
		t.Fatalf("after process removal = %+v", product)
	}
	var runtimeState string
	var processCount int
	if err := runtime.DB().QueryRow(
		`SELECT runtime_state, process_count
		   FROM software_product_inventory
		  WHERE agent_id = $1 AND product_key = 'nginx'`,
		agent.ID,
	).Scan(&runtimeState, &processCount); err != nil {
		t.Fatal(err)
	}
	if runtimeState != "unknown" || processCount != 0 {
		t.Fatalf("projection after process removal = %q/%d", runtimeState, processCount)
	}

	removedPackage := inventoryEnvelope(agent.AgentID, nil)
	removedPackage.Changes = []AssetChange{{
		Kind: "removed", AssetID: "nginx-package", Category: "software.package",
	}}
	processEnvelope(t, service, agent, removedPackage)
	var status string
	var deleted any
	if err := runtime.DB().QueryRow(
		`SELECT a.status, a.deleted_at
		   FROM assets a JOIN asset_sources s ON s.asset_id = a.id
		  WHERE s.agent_id = $1 AND s.category = 'software.product'
		    AND s.source_asset_id = 'nginx'`,
		agent.ID,
	).Scan(&status, &deleted); err != nil {
		t.Fatal(err)
	}
	if status != "removed" || deleted == nil {
		t.Fatalf("retired product = status %q deleted %#v", status, deleted)
	}
	assertCountWhere(t, runtime, "software_product_inventory", "agent_id = '"+agent.ID+"'", 0)

	stoppedService := record(
		"nginx-service", "service",
		`{"name":"nginx.service","active_state":"inactive","sub_state":"dead"}`,
	)
	reactivated := inventoryEnvelope(agent.AgentID, nil)
	reactivated.Changes = []AssetChange{{
		Kind: "added", AssetID: stoppedService.AssetID,
		Category: stoppedService.Category, Record: &stoppedService,
	}}
	processEnvelope(t, service, agent, reactivated)
	product = readProductDetection(t, runtime.DB(), agent.ID, "nginx")
	if product.RuntimeState != "stopped" || product.InstallState != "installed" ||
		len(product.ServiceNames) != 1 {
		t.Fatalf("reactivated product = %+v", product)
	}
	if err := runtime.DB().QueryRow(
		`SELECT runtime_state, process_count
		   FROM software_product_inventory
		  WHERE agent_id = $1 AND product_key = 'nginx'`,
		agent.ID,
	).Scan(&runtimeState, &processCount); err != nil {
		t.Fatal(err)
	}
	if runtimeState != "stopped" || processCount != 0 {
		t.Fatalf("reactivated projection = %q/%d", runtimeState, processCount)
	}
	assertCountWhere(t, runtime, "assets", "type = 'software_product'", 1)
	var activeRelation int
	if err := runtime.DB().QueryRow(
		`SELECT COUNT(*) FROM asset_relations r
		   JOIN assets a ON a.id = r.source_asset_id
		  WHERE a.type = 'software_product' AND r.status = 'active'
		    AND r.valid_to IS NULL`,
	).Scan(&activeRelation); err != nil {
		t.Fatal(err)
	}
	if activeRelation != 1 {
		t.Fatalf("reactivated relation count = %d", activeRelation)
	}
}

func readProductDetection(
	t *testing.T,
	database *sql.DB,
	agentID string,
	productKey string,
) softwarecatalog.Detection {
	t.Helper()
	var raw string
	if err := database.QueryRow(
		`SELECT a.attributes_json
		   FROM assets a JOIN asset_sources s ON s.asset_id = a.id
		  WHERE s.agent_id = $1 AND s.category = 'software.product'
		    AND s.source_asset_id = $2 AND s.deleted_at IS NULL`,
		agentID,
		productKey,
	).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var product softwarecatalog.Detection
	if err := json.Unmarshal([]byte(raw), &product); err != nil {
		t.Fatal(err)
	}
	return product
}

func heartbeatEnvelope(agentID string) Envelope {
	return Envelope{
		SchemaVersion: 1,
		EventID:       uuid.NewString(),
		AgentID:       agentID,
		CreatedAt:     nextTestEventTimestamp(),
		Kind:          "heartbeat",
	}
}
