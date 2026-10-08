package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// Software search stores search_text lowercased, so the pattern has to be
// lowercased too. SQLite's LIKE ignores ASCII case and hid this: searching for a
// product by the capitalisation the console actually displays - "NGINX", not
// "nginx" - matched here and returned nothing at all on PostgreSQL, where LIKE is
// case-sensitive.
func TestSoftwareSearchIgnoresCase(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	host := insertSoftwareTestAsset(t, server, "host-case", "web-case-01", "host", `{}`, 1, now)
	product := insertSoftwareTestAsset(t, server, "product-case", "NGINX", "software_product", `{
		"product_key":"nginx","product_name":"NGINX","role":"web_server",
		"vendor":"F5","version":"1.26.2","install_state":"installed",
		"runtime_state":"running","detection_method":"builtin_catalog","confidence":0.95
	}`, 0.95, now)
	insertRunsOn(t, server, product, host)

	request := httptest.NewRequest("POST", "/mcp", nil)
	search := func(query string) int {
		t.Helper()
		result, err := server.mcpSoftwareInventory(
			request,
			mcpArgumentsForTest("software_inventory", []byte(`{"q":"`+query+`","limit":10}`)),
		)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(result)
		var payload softwareInventoryResponse
		if err := json.Unmarshal(encoded, &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Total
	}

	// The name as the console displays it, the name as it is stored, and a mixed
	// form must all find the same product.
	for _, query := range []string{"NGINX", "nginx", "NgInX", "WEB-CASE-01"} {
		if got := search(query); got != 1 {
			t.Fatalf("searching %q returned %d products, want 1", query, got)
		}
	}
}
