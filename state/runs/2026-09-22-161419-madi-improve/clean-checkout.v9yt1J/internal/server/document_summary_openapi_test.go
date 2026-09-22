package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocumentSummaryOpenAPI(t *testing.T) {
	s := &Server{Version: "test", apiRoutes: []string{"GET /api/v1/documents", "GET /api/v1/spaces/{id}/documents"}}
	w := httptest.NewRecorder()
	s.openAPI(w, httptest.NewRequest("GET", "/api/v1/openapi.json", nil))
	var spec map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &spec); e != nil {
		t.Fatal(e)
	}
	paths := spec["paths"].(map[string]any)
	for _, path := range []string{"/documents", "/spaces/{id}/documents"} {
		op := paths[path].(map[string]any)["get"].(map[string]any)
		response := op["responses"].(map[string]any)["200"]
		if !strings.Contains(string(jsonValue(response)), "#/components/schemas/DocumentSummary") || !strings.Contains(str(op, "description"), "document:read") {
			t.Fatalf("summary route schema missing %s", path)
		}
	}
	schema := spec["components"].(map[string]any)["schemas"].(map[string]any)["DocumentSummary"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	if properties["markdown"] != nil || properties["block_metadata"] != nil || properties["excerpt"] == nil {
		t.Fatal("summary schema exposes unbounded original")
	}
}
