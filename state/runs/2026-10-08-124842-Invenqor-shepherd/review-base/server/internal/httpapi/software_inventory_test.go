package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type softwareInventoryResponse struct {
	Summary softwareInventorySummary `json:"summary"`
	Items   []softwareProductItem    `json:"items"`
	Total   int                      `json:"total"`
	Limit   int                      `json:"limit"`
	Offset  int                      `json:"offset"`
	HasMore bool                     `json:"has_more"`
	Filters struct {
		Roles   []string `json:"roles"`
		Vendors []string `json:"vendors"`
	} `json:"filters"`
}

func TestSoftwareProductsAreSummarizedWithHostAndEvidence(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	hostOne := insertSoftwareTestAsset(t, server, "host-1", "web-prd-01", "host", `{}`, 1, now)
	hostTwo := insertSoftwareTestAsset(t, server, "host-2", "db-prd-01", "host", `{}`, 1, now)
	nginx := insertSoftwareTestAsset(t, server, "sw-nginx", "NGINX", "software_product", `{
		"product_key":"nginx","product_name":"NGINX","role":"web_proxy",
		"vendor":"F5","version":"1.26","install_state":"installed",
		"runtime_state":"running","service_names":["nginx.service"],
		"process_names":["nginx"],"detection_method":"builtin_catalog",
		"confidence":0.97,
		"evidence":[{"kind":"service","name":"nginx.service","source_asset_id":"svc-1"}]
	}`, 0.5, now)
	postgres := insertSoftwareTestAsset(t, server, "sw-postgres", "PostgreSQL", "software_product", `{
		"product_key":"postgresql","product_name":"PostgreSQL","role":"database",
		"vendor":"PostgreSQL Global Development Group","version":"16",
		"install_state":"observed","runtime_state":"running",
		"process_names":["postgres"],"process_count":7,"evidence_count":7,
		"catalog_version":"2026.08.1","confidence":0.91
	}`, 0.5, now)
	unknown := insertSoftwareTestAsset(t, server, "sw-custom", "Custom Runtime", "software_product", `{
		"product_key":"custom-runtime","role":"application_runtime",
		"runtime_state":"stopped","confidence":0.62
	}`, 0.5, now)
	insertRunsOn(t, server, nginx, hostOne)
	insertRunsOn(t, server, postgres, hostTwo)
	insertRunsOn(t, server, unknown, hostOne)

	response := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets/software-products",
		nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload softwareInventoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Summary.Products != 3 || payload.Summary.Instances != 3 ||
		payload.Summary.Hosts != 2 || payload.Summary.Running != 2 ||
		payload.Summary.Stopped != 1 || payload.Summary.HighConfidence != 2 ||
		payload.Summary.NeedsReview != 1 || payload.Summary.WithProcessEvidence != 2 {
		t.Fatalf("unexpected summary: %+v", payload.Summary)
	}
	if len(payload.Summary.TopProducts) != 3 || payload.Total != 3 {
		t.Fatalf("top/total = %d/%d", len(payload.Summary.TopProducts), payload.Total)
	}
	var found softwareProductItem
	for _, item := range payload.Items {
		if item.ProductKey == "postgresql" {
			found = item
		}
	}
	if found.Host.Name != "db-prd-01" || found.Version != "16" {
		t.Fatalf("postgres product = %+v", found)
	}
	if found.ProcessCount != 7 || found.EvidenceCount != 7 ||
		found.CatalogVersion != "2026.08.1" || payload.Summary.MappedProcesses != 8 {
		t.Fatalf("catalog counters = %+v summary = %+v", found, payload.Summary)
	}
	if len(found.Evidence) != 1 || found.Evidence[0].Kind != "process" ||
		found.Evidence[0].Name != "postgres" {
		t.Fatalf("fallback evidence = %+v", found.Evidence)
	}
	if len(payload.Filters.Roles) != 3 || len(payload.Filters.Vendors) != 3 {
		t.Fatalf("filters = %+v", payload.Filters)
	}

	filtered := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/assets/software-products?q=db-prd-01&role=database&vendor=PostgreSQL+Global+Development+Group&runtime_state=running&confidence=high",
		nil, cookie, csrf,
	)
	if filtered.Code != http.StatusOK {
		t.Fatalf("filtered status = %d body = %s", filtered.Code, filtered.Body.String())
	}
	if err := json.Unmarshal(filtered.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 1 || len(payload.Items) != 1 || payload.Items[0].ProductKey != "postgresql" {
		t.Fatalf("filtered payload = %+v", payload)
	}

	paged := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/assets/software-products?limit=1&offset=1",
		nil, cookie, csrf,
	)
	if paged.Code != http.StatusOK {
		t.Fatalf("paged status = %d body = %s", paged.Code, paged.Body.String())
	}
	if err := json.Unmarshal(paged.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 3 || payload.Limit != 1 || payload.Offset != 1 ||
		!payload.HasMore || len(payload.Items) != 1 ||
		payload.Items[0].ProductKey != "nginx" {
		t.Fatalf("paged payload = %+v", payload)
	}
	if payload.Summary.Instances != 3 || len(payload.Filters.Roles) != 3 {
		t.Fatalf("paging changed global summary/facets = %+v", payload)
	}
}

func insertSoftwareTestAsset(
	t *testing.T,
	server *Server,
	key, name, assetType, attributes string,
	confidence float64,
	now time.Time,
) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := server.database.DB().Exec(
		`INSERT INTO assets(
			id,asset_key,name,type,status,confidence,attributes_json,
			custom_fields_json,source,first_seen_at,last_seen_at,created_at,updated_at,
			classification_source
		 ) VALUES($1,$2,$3,$4,'active',$5,$6,'{}','agent',$7,$7,$7,$7,
		          CASE WHEN $4='software_product' THEN 'software_catalog' ELSE '' END)`,
		id, key, name, assetType, confidence, attributes, now,
	); err != nil {
		t.Fatalf("insert %s: %v", name, err)
	}
	if assetType == "host" {
		if _, err := server.database.DB().Exec(
			`INSERT INTO agents(id,agent_id,hostname,status,last_seen_at)
			 VALUES($1,$2,$3,'active',$4)`,
			id, uuid.NewString(), name, now,
		); err != nil {
			t.Fatalf("insert test agent for %s: %v", name, err)
		}
	}
	return id
}

func insertRunsOn(t *testing.T, server *Server, productID, hostID string) {
	t.Helper()
	if _, err := server.database.DB().Exec(
		`INSERT INTO asset_relations(
			id,source_asset_id,relation_type,target_asset_id,source,confidence
		 ) VALUES($1,$2,'runs_on',$3,'automatic',0.95)`,
		uuid.NewString(), productID, hostID,
	); err != nil {
		t.Fatalf("insert runs_on: %v", err)
	}
	var name, attributes, classificationSource, hostName string
	var assetConfidence float64
	if err := server.database.DB().QueryRow(
		`SELECT name, attributes_json, confidence, classification_source
		   FROM assets WHERE id = $1`,
		productID,
	).Scan(&name, &attributes, &assetConfidence, &classificationSource); err != nil {
		t.Fatalf("read projected product: %v", err)
	}
	if classificationSource != "software_catalog" {
		return
	}
	if err := server.database.DB().QueryRow(
		"SELECT name FROM assets WHERE id = $1",
		hostID,
	).Scan(&hostName); err != nil {
		t.Fatalf("read projected product host: %v", err)
	}
	var product softwareProductAttributes
	if err := json.Unmarshal([]byte(attributes), &product); err != nil {
		t.Fatalf("decode projected product: %v", err)
	}
	productKey := firstNonEmpty(product.ProductKey, strings.ToLower(name))
	productName := firstNonEmpty(product.ProductName, name, productKey)
	role := firstNonEmpty(product.Role, "other")
	vendor := firstNonEmpty(product.Vendor, "unknown")
	installState := product.InstallState
	if installState != "installed" && installState != "observed" {
		installState = "unknown"
	}
	runtimeState := product.RuntimeState
	if runtimeState != "running" && runtimeState != "stopped" && runtimeState != "unknown" {
		runtimeState = "unknown"
	}
	confidence := product.Confidence
	if confidence <= 0 || confidence > 1 {
		confidence = assetConfidence
	}
	processCount := product.ProcessCount
	if processCount < len(product.ProcessNames) {
		processCount = len(product.ProcessNames)
	}
	evidenceCount := product.EvidenceCount
	if evidenceCount < len(product.Evidence) {
		evidenceCount = len(product.Evidence)
	}
	searchValues := []string{
		productKey, productName, role, vendor, product.Version, hostName,
	}
	searchValues = append(searchValues, product.ServiceNames...)
	searchValues = append(searchValues, product.ProcessNames...)
	searchValues = append(searchValues, product.PackageNames...)
	if _, err := server.database.DB().Exec(
		`INSERT INTO software_product_inventory(
			asset_id,agent_id,product_key,product_name,role,vendor,version,
			install_state,runtime_state,confidence,process_count,evidence_count,
			catalog_version,search_text,updated_at
		 ) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,CURRENT_TIMESTAMP)
		 ON CONFLICT(asset_id) DO UPDATE SET
			agent_id=excluded.agent_id, product_key=excluded.product_key,
			product_name=excluded.product_name, role=excluded.role,
			vendor=excluded.vendor, version=excluded.version,
			install_state=excluded.install_state, runtime_state=excluded.runtime_state,
			confidence=excluded.confidence, process_count=excluded.process_count,
			evidence_count=excluded.evidence_count,
			catalog_version=excluded.catalog_version,
			search_text=excluded.search_text, updated_at=excluded.updated_at`,
		productID, hostID, productKey, productName, role, vendor, product.Version,
		installState, runtimeState, confidence, processCount, evidenceCount,
		product.CatalogVersion, strings.ToLower(strings.Join(searchValues, " ")),
	); err != nil {
		t.Fatalf("insert software inventory projection: %v", err)
	}
}

func TestManagedAssetScopeKeepsProcessEvidenceOutOfTheDefaultConsole(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	now := time.Now().UTC()
	insertSoftwareTestAsset(t, server, "host", "managed-host", "host", `{}`, 1, now)
	insertSoftwareTestAsset(t, server, "proc", "raw-process", "process", `{}`, 1, now)

	managed := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets?scope=managed",
		nil, cookie, csrf,
	)
	if managed.Code != http.StatusOK {
		t.Fatalf("managed status = %d body = %s", managed.Code, managed.Body.String())
	}
	var page struct {
		Items []assetView `json:"items"`
		Total int         `json:"total"`
	}
	if err := json.Unmarshal(managed.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Name != "managed-host" {
		t.Fatalf("managed page = %+v", page)
	}
	all := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets",
		nil, cookie, csrf,
	)
	if err := json.Unmarshal(all.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 {
		t.Fatalf("all page total = %d, want 2", page.Total)
	}

	statistics := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/dashboard/statistics?scope=managed",
		nil, cookie, csrf,
	)
	var stats struct {
		Assets struct {
			Total int `json:"total"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(statistics.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.Assets.Total != 1 {
		t.Fatalf("managed dashboard total = %d", stats.Assets.Total)
	}
}

func TestEmptySoftwareInventoryReturnsArraysInsteadOfNull(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	manualID := insertSoftwareTestAsset(
		t, server, "manual-product", "Operator Managed Product",
		"software_product", `{}`, 1, time.Now().UTC(),
	)
	if _, err := server.database.DB().Exec(
		"UPDATE assets SET classification_source='manual' WHERE id=$1", manualID,
	); err != nil {
		t.Fatal(err)
	}
	response := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets/software-products",
		nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload softwareInventoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Items == nil || payload.Summary.TopProducts == nil ||
		payload.Filters.Roles == nil || payload.Filters.Vendors == nil {
		t.Fatalf("empty inventory contains null arrays: %s", response.Body.String())
	}
	if payload.Total != 0 || len(payload.Items) != 0 ||
		len(payload.Summary.TopProducts) != 0 {
		t.Fatalf("manual products leaked into automatic inventory = %+v", payload)
	}
}
