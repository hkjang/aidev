package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMCPSoftwareInventoryReturnsProductStateAndEvidence(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	host := insertSoftwareTestAsset(t, server, "host-mcp", "web-mcp-01", "host", `{}`, 1, now)
	product := insertSoftwareTestAsset(t, server, "product-mcp", "NGINX", "software_product", `{
		"product_key":"nginx","product_name":"NGINX","role":"web_server",
		"vendor":"F5","version":"1.26.2","install_state":"installed",
		"runtime_state":"running","process_names":["nginx"],
		"detection_method":"builtin_catalog","confidence":0.95,
		"evidence":[{"kind":"process","name":"nginx","source_asset_id":"nginx-pid"}]
	}`, 0.95, now)
	insertRunsOn(t, server, product, host)

	request := httptest.NewRequest("POST", "/mcp", nil)
	result, err := server.mcpSoftwareInventory(
		request,
		mcpArgumentsForTest("software_inventory", []byte(`{"q":"web-mcp-01","role":"web_server","vendor":"F5","runtime_state":"running","confidence":"high","limit":10}`)),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var payload softwareInventoryResponse
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 1 || len(payload.Items) != 1 {
		t.Fatalf("software_inventory payload = %s", encoded)
	}
	item := payload.Items[0]
	if item.ProductKey != "nginx" || item.Host.Name != "web-mcp-01" ||
		item.RuntimeState != "running" || len(item.Evidence) != 1 {
		t.Fatalf("software_inventory item = %+v", item)
	}

	found := false
	for _, tool := range mcpTools {
		if tool.Name == "software_inventory" {
			properties, _ := tool.InputSchema["properties"].(map[string]any)
			_, hasVendor := properties["vendor"]
			found = tool.Scope == "assets.read" && hasVendor
		}
	}
	if !found {
		t.Fatal("software_inventory tool is not exposed with assets.read")
	}
}

func TestMCPAssetSearchHidesRawProcessesUnlessRequested(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	insertSoftwareTestAsset(t, server, "host-search", "search-host", "host", `{}`, 1, now)
	insertSoftwareTestAsset(t, server, "process-search", "noisy-process", "process", `{}`, 1, now)
	request := httptest.NewRequest("POST", "/mcp", nil)

	count := func(raw string) int {
		t.Helper()
		result, err := server.mcpAssetSearch(request, mcpArgumentsForTest("asset_search", []byte(raw)))
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
		return len(payload.Items)
	}
	if got := count(`{"limit":10}`); got != 1 {
		t.Fatalf("managed MCP search returned %d items, want 1", got)
	}
	if got := count(`{"limit":10,"include_observations":true}`); got != 2 {
		t.Fatalf("forensic MCP search returned %d items, want 2", got)
	}
}

// mcpArgumentsForTest builds the reader the tool dispatch builds, from the same
// schema the client is given, so a test cannot pass arguments the real path
// would reject.
func mcpArgumentsForTest(tool string, raw []byte) *mcpArguments {
	for index := range mcpTools {
		if mcpTools[index].Name == tool {
			return newMCPArguments(tool, raw, mcpTools[index].InputSchema)
		}
	}
	panic("unknown tool " + tool)
}
