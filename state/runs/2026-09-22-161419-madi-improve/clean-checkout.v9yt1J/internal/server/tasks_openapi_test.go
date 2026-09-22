package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTasksOpenAPIScopeAndBoundedCount(t *testing.T) {
	s := &Server{Version: "test", apiRoutes: []string{"GET /api/v1/tasks/board", "GET /api/v1/tasks/calendar"}}
	w := httptest.NewRecorder()
	s.openAPI(w, httptest.NewRequest("GET", "/api/v1/openapi.json", nil))
	var spec map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &spec); e != nil {
		t.Fatal(e)
	}
	paths := spec["paths"].(map[string]any)
	calendar := paths["/tasks/calendar"].(map[string]any)["get"].(map[string]any)
	found := false
	for _, raw := range calendar["parameters"].([]any) {
		param := raw.(map[string]any)
		if param["name"] == "document_id" {
			found = param["in"] == "query" && strings.Contains(str(param, "description"), "현재 권한") && param["schema"].(map[string]any)["format"] == "uuid"
		}
	}
	if !found {
		t.Fatal("calendar document scope is not documented")
	}
	board := paths["/tasks/board"].(map[string]any)["get"].(map[string]any)
	response := board["responses"].(map[string]any)["200"].(map[string]any)
	schema := response["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	properties := schema["properties"].(map[string]any)
	for _, key := range []string{"total_documents_exact", "total_documents_is_lower_bound", "truncated"} {
		if properties[key].(map[string]any)["type"] != "boolean" {
			t.Fatal("bounded count signal missing", key)
		}
	}
	if properties["total_documents"].(map[string]any)["maximum"] != float64(2001) {
		t.Fatal("bounded count incorrectly describes an exact full-workspace count")
	}
}
