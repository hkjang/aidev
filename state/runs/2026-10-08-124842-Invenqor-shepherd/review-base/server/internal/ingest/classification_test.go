package ingest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/storage"
)

// registerAgent provisions a second agent so cloned-machine detection has two
// independent reporters.
func registerAgent(t *testing.T, runtime *storage.Runtime) agents.Agent {
	t.Helper()
	provisioned, err := agents.NewService(runtime.DB()).ProvisionBearer(
		context.Background(), uuid.NewString(), "clone-host", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	return provisioned.Agent
}

// A realistic inventory has to come out the other side with business context and
// a readable dependency graph, not schema defaults and thousands of edges.
func TestIngestClassifiesInventoryAndBuildsAReadableGraph(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)

	records := []AssetRecord{
		record("host", "system", `{"hostname":"app-prd-01","architecture":"x86_64","machine_id":"0123456789abcdef0123"}`),
		record("nic-eth0", "network.interface", `{"interface":"eth0"}`),
		record("nic-lo", "network.interface", `{"interface":"lo"}`),
		record("mount-root", "hardware.filesystem", `{"mount_point":"/","device":"/dev/sda1"}`),
		record("svc-nginx", "service", `{"manager":"systemd","name":"nginx.service"}`),
		record("svc-postgres", "service", `{"manager":"systemd","name":"postgresql@14-main.service"}`),
		record("svc-cron", "service", `{"manager":"systemd","name":"cron.service"}`),
		record("svc-ssh", "service", `{"manager":"systemd","name":"ssh.service"}`),
		record("pkg-curl", "software.package", `{"name":"curl","version":"8.5"}`),
		record("pkg-fonts", "software.package", `{"name":"fonts-noto","version":"1"}`),
		record("acct-root", "account.user", `{"username":"root","uid":0}`),
	}
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, records))

	type row struct {
		name        string
		assetType   string
		environment string
		criticality string
		tags        string
	}
	assets := map[string]row{}
	rows, err := runtime.DB().Query(
		`SELECT name, type, environment, criticality, tags_json FROM assets`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item row
		if err := rows.Scan(
			&item.name, &item.assetType, &item.environment,
			&item.criticality, &item.tags,
		); err != nil {
			t.Fatal(err)
		}
		assets[item.name] = item
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	// The host name carries the only environment signal a collector can see, and
	// criticality then follows from environment plus role.
	host := assets["app-prd-01"]
	if host.assetType != "host" || host.environment != "production" ||
		host.criticality != "high" {
		t.Fatalf("host = %+v", host)
	}
	// A known database engine is promoted past 'service' and inherits the
	// strictest criticality of the production data tier.
	database := assets["postgresql@14-main.service"]
	if database.assetType != "database" || database.criticality != "critical" {
		t.Fatalf("database = %+v", database)
	}
	if database.tags != `["data-tier"]` {
		t.Fatalf("database tags = %s", database.tags)
	}
	web := assets["nginx.service"]
	if web.assetType != "service" || web.tags != `["web-tier"]` {
		t.Fatalf("web tier = %+v", web)
	}
	// An unremarkable unit stays a plain service with no role tag.
	if plain := assets["cron.service"]; plain.assetType != "service" ||
		plain.tags != "[]" {
		t.Fatalf("plain service = %+v", plain)
	}
	if pkg := assets["curl"]; pkg.assetType != "software" {
		t.Fatalf("package = %+v", pkg)
	}
	if account := assets["root"]; account.assetType != "account" {
		t.Fatalf("account = %+v", account)
	}

	// The graph keeps the component relationships and now adds one stable,
	// normalized product per recognizable service family. Raw packages,
	// processes, accounts and ordinary units still stay out; nginx/postgresql
	// service observations remain as evidence while the product nodes give the
	// CMDB a lifecycle identity that does not change with a PID or unit name.
	type edge struct {
		source     string
		relation   string
		target     string
		derivation string
		status     string
	}
	edgeRows, err := runtime.DB().Query(
		`SELECT s.name, r.relation_type, t.name, r.derivation, r.status
		   FROM asset_relations r
		   JOIN assets s ON s.id = r.source_asset_id
		   JOIN assets t ON t.id = r.target_asset_id
		  ORDER BY r.relation_type, s.name`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer edgeRows.Close()
	edges := []edge{}
	for edgeRows.Next() {
		var item edge
		if err := edgeRows.Scan(
			&item.source, &item.relation, &item.target,
			&item.derivation, &item.status,
		); err != nil {
			t.Fatal(err)
		}
		edges = append(edges, item)
	}
	if err := edgeRows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(edges) != 8 {
		t.Fatalf("edges = %+v, want the curated components and three products", edges)
	}
	expected := map[string]string{
		"eth0": "part_of",
		"lo":   "part_of",
		// A filesystem record is named after its device by the collector naming
		// order, so the volume edge carries that name.
		"/dev/sda1":                  "attached_to",
		"NGINX":                      "runs_on",
		"OpenSSH Server":             "runs_on",
		"PostgreSQL":                 "runs_on",
		"nginx.service":              "runs_on",
		"postgresql@14-main.service": "runs_on",
	}
	for _, item := range edges {
		if expected[item.source] != item.relation {
			t.Fatalf("unexpected edge %+v", item)
		}
		if item.target != "app-prd-01" {
			t.Fatalf("edge %+v does not point at the host", item)
		}
		if item.derivation != "same_agent_inventory" || item.status != "active" {
			t.Fatalf("edge %+v lost its provenance", item)
		}
	}

	// Re-processing must be idempotent: no duplicate edges, no drift.
	second := inventoryEnvelope(agent.AgentID, records)
	processEnvelope(t, service, agent, second)
	assertCount(t, runtime, "asset_relations", 8)
}

// A field an operator has taken over must survive every later automatic pass,
// or the console becomes a fight between the person and the rule set.
func TestManualClassificationSurvivesReingest(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	records := []AssetRecord{
		record("host", "system", `{"hostname":"db-prd-09"}`),
	}
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, records))

	var assetID string
	if err := runtime.DB().QueryRow(
		"SELECT id FROM assets WHERE name = 'db-prd-09'",
	).Scan(&assetID); err != nil {
		t.Fatal(err)
	}
	// The operator disagrees with the inferred environment and criticality.
	if _, err := runtime.DB().Exec(
		`UPDATE assets SET environment = 'dr', criticality = 'medium',
		   manual_fields_json = '["environment","criticality"]',
		   classification_source = 'manual'
		 WHERE id = $1`,
		assetID,
	); err != nil {
		t.Fatal(err)
	}

	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, records))

	var environment, criticality, assetType string
	if err := runtime.DB().QueryRow(
		"SELECT environment, criticality, type FROM assets WHERE id = $1",
		assetID,
	).Scan(&environment, &criticality, &assetType); err != nil {
		t.Fatal(err)
	}
	if environment != "dr" || criticality != "medium" {
		t.Fatalf(
			"the rule set overwrote operator values: environment=%q criticality=%q",
			environment, criticality,
		)
	}
	// Fields the operator did not claim still get classified.
	if assetType != "host" {
		t.Fatalf("type = %q, want host", assetType)
	}
}

// Two agents reporting the same machine identifier is a cloned image, which
// silently doubles an inventory. It must be surfaced, and only as a proposal.
func TestClonedMachineIsProposedAsADuplicate(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	clone := registerAgent(t, runtime)

	identity := `{"hostname":"tpl-prd-01","machine_id":"ffeeddccbbaa99887766"}`
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host", "system", identity),
	}))
	processEnvelope(t, service, clone, inventoryEnvelope(clone.AgentID, []AssetRecord{
		record("host", "system", identity),
	}))

	var relation, status, derivation string
	var confidence float64
	if err := runtime.DB().QueryRow(
		`SELECT relation_type, status, derivation, confidence
		   FROM asset_relations WHERE relation_type = 'duplicate_of'`,
	).Scan(&relation, &status, &derivation, &confidence); err != nil {
		t.Fatalf("no duplicate proposal was recorded: %v", err)
	}
	if status != "proposed" {
		t.Fatalf("status = %q, want proposed - merging automatically is not safe", status)
	}
	if derivation != "machine_identity" || confidence <= 0 || confidence >= 1 {
		t.Fatalf("proposal = %q %v", derivation, confidence)
	}
	// Repeated events must not pile up proposals for the same pair.
	processEnvelope(t, service, clone, inventoryEnvelope(clone.AgentID, []AssetRecord{
		record("host", "system", identity),
	}))
	assertCountWhere(
		t, runtime, "asset_relations", "relation_type = 'duplicate_of'", 1,
	)
}

func TestClassificationRulesAreCachedButInvalidatable(t *testing.T) {
	t.Parallel()
	runtime, agent, service := testService(t)
	// Disable the environment rule and invalidate, then confirm the change lands.
	if _, err := runtime.DB().Exec(
		`UPDATE asset_classification_rules SET enabled = FALSE
		  WHERE id = '20000000-0000-0000-0000-000000000020'`,
	); err != nil {
		t.Fatal(err)
	}
	service.Classifier().Invalidate()
	processEnvelope(t, service, agent, inventoryEnvelope(agent.AgentID, []AssetRecord{
		record("host", "system", `{"hostname":"api-prd-02"}`),
	}))
	var environment string
	if err := runtime.DB().QueryRow(
		"SELECT environment FROM assets WHERE name = 'api-prd-02'",
	).Scan(&environment); err != nil {
		t.Fatal(err)
	}
	if environment == "production" {
		t.Fatal("a disabled rule still classified the asset")
	}
	var encoded []byte
	if err := runtime.DB().QueryRow(
		"SELECT classification_rules_json FROM assets WHERE name = 'api-prd-02'",
	).Scan(&encoded); err != nil {
		t.Fatal(err)
	}
	applied := []string{}
	if err := json.Unmarshal(encoded, &applied); err != nil {
		t.Fatal(err)
	}
	for _, rule := range applied {
		if rule == "20000000-0000-0000-0000-000000000020" {
			t.Fatal("the disabled rule appears in the provenance trail")
		}
	}
}
