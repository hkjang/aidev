package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/storagetest"
)

func insertSearchableAsset(
	t *testing.T, server *Server, key, name, owner string,
) {
	t.Helper()
	if _, err := server.database.DB().Exec(
		`INSERT INTO assets(
			id, asset_key, name, type, status, criticality, environment,
			owner_department, confidence, attributes_json, custom_fields_json,
			source, first_seen_at, last_seen_at, created_at, updated_at
		 ) VALUES($1,$2,$3,'host','active','normal','other',$4,
		          1.0,'{}','{}','manual',
		          CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,
		          CURRENT_TIMESTAMP)`,
		uuid.NewString(), key, name, owner,
	); err != nil {
		t.Fatal(err)
	}
}

func assetSearchNames(
	t *testing.T,
	server *Server,
	cookie *http.Cookie,
	csrf string,
	path string,
) []string {
	t.Helper()
	response := performAuthenticatedJSON(
		t, server, http.MethodGet, path, nil, cookie, csrf,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d body = %s",
			path, response.Code, response.Body.String())
	}
	var payload struct {
		Items []assetView `json:"items"`
		Total int64       `json:"total"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if int64(len(payload.Items)) != payload.Total {
		t.Fatalf("GET %s returned %d items but total %d",
			path, len(payload.Items), payload.Total)
	}
	names := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		names = append(names, item.Name)
	}
	return names
}

// An operator types a host name, not a pattern. The search put that text
// straight into a LIKE, where "_" - ordinary in the names this product stores -
// matched any character, so searching for db_prod also returned db-prod.
func TestAssetSearchTreatsAnUnderscoreAsItself(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertSearchableAsset(t, server, "key-underscore", "db_prod", "platform")
	insertSearchableAsset(t, server, "key-hyphen", "db-prod", "platform")

	found := assetSearchNames(t, server, cookie, csrf, "/api/v1/assets?q=db_prod")
	if len(found) != 1 || found[0] != "db_prod" {
		t.Fatalf("q=db_prod returned %v", found)
	}
	// The asset key is searched by the same clause.
	if byKey := assetSearchNames(
		t, server, cookie, csrf, "/api/v1/assets?q=key-underscore",
	); len(byKey) != 1 || byKey[0] != "db_prod" {
		t.Fatalf("q=key-underscore returned %v", byKey)
	}
}

// A percent sign matched every row, so a search that found nothing looked
// exactly like a search that found the whole inventory.
func TestAssetSearchTreatsAPercentSignAsItself(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertSearchableAsset(t, server, "key-cache", "cache 100% hit", "platform")
	insertSearchableAsset(t, server, "key-plain", "web-01", "platform")

	found := assetSearchNames(
		t, server, cookie, csrf, "/api/v1/assets?q=100%25",
	)
	if len(found) != 1 || found[0] != "cache 100% hit" {
		t.Fatalf("q=100%% returned %v", found)
	}
	if none := assetSearchNames(
		t, server, cookie, csrf, "/api/v1/assets?q=%25%25%25",
	); len(none) != 0 {
		t.Fatalf("a search for three percent signs returned %v", none)
	}
}

// PostgreSQL takes a backslash as the default LIKE escape and the SQLite
// fallback has no default escape character at all, so the same search for a
// Windows path answered differently depending on which mode the deployment ran.
func TestAssetSearchReadsABackslashTheSameWayInBothModes(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertSearchableAsset(
		t, server, "key-windows", `C:\Program Files`, "platform",
	)

	found := assetSearchNames(
		t, server, cookie, csrf, "/api/v1/assets?q=C%3A%5CProgram",
	)
	if len(found) != 1 || found[0] != `C:\Program Files` {
		t.Fatalf(`q=C:\Program returned %v`, found)
	}
}

// The owner filter is the same kind of typed text as the free-text search.
func TestAssetOwnerFilterTreatsAnUnderscoreAsItself(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertSearchableAsset(t, server, "key-owner-a", "owned-a", "it_ops")
	insertSearchableAsset(t, server, "key-owner-b", "owned-b", "it-ops")

	found := assetSearchNames(
		t, server, cookie, csrf, "/api/v1/assets?owner_department=it_ops",
	)
	if len(found) != 1 || found[0] != "owned-a" {
		t.Fatalf("owner_department=it_ops returned %v", found)
	}
}

// The MCP asset search runs its own statement, so it needs the same rule: an
// assistant that asks for db_prod must not be told db-prod is the same host.
func TestMCPAssetSearchTreatsAnUnderscoreAsItself(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	insertSoftwareTestAsset(t, server, "mcp_us", "db_prod", "host", `{}`, 1, now)
	insertSoftwareTestAsset(t, server, "mcp-hy", "db-prod", "host", `{}`, 1, now)

	result, err := server.mcpAssetSearch(
		httptest.NewRequest("POST", "/mcp", nil),
		mcpArgumentsForTest("asset_search", []byte(`{"q":"db_prod","limit":10}`)),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var payload struct {
		Items []assetView `json:"items"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 || payload.Items[0].Name != "db_prod" {
		t.Fatalf("MCP asset_search for db_prod returned %+v", payload.Items)
	}
}

// The audit log is an accountability tool, and the actions it records are full
// of underscores: a search for api_key.create must not also answer with the
// rows an unrelated action wrote.
func TestAuditSearchTreatsAnUnderscoreAsItself(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	now := time.Now().UTC()
	insertAuditEvent(t, runtime, "api_key.create", "alice", "success", now)
	insertAuditEvent(t, runtime, "apiXkey.create", "bob", "success", now)

	found := decodeAudit(
		t, server, cookie, csrf, "/api/v1/admin/audit?q=api_key.create",
	)
	if found.Total != 1 || len(found.Items) != 1 ||
		found.Items[0].Action != "api_key.create" {
		t.Fatalf("q=api_key.create found total=%d items=%+v",
			found.Total, found.Items)
	}
	// The actor filter is a separate clause with the same defect.
	actor := decodeAudit(
		t, server, cookie, csrf, "/api/v1/admin/audit?q=%25",
	)
	if actor.Total != 0 {
		t.Fatalf("a search for a percent sign matched %d entries", actor.Total)
	}
}
