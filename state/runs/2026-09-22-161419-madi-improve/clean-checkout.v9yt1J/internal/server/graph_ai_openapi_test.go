package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestGraphAIOpenAPIStreamingAndHumanConfirmation(t *testing.T) {
	s := &Server{Version: "test", apiRoutes: []string{"GET /api/v1/workspaces/{id}/graph-ai/context", "POST /api/v1/workspaces/{id}/graph-ai/analyze", "POST /api/v1/graph-ai/actions/{id}/confirm"}}
	w := httptest.NewRecorder()
	s.openAPI(w, httptest.NewRequest("GET", "/api/v1/openapi.json", nil))
	var spec map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &spec); e != nil {
		t.Fatal(e)
	}
	paths := spec["paths"].(map[string]any)
	analyze := paths["/workspaces/{id}/graph-ai/analyze"].(map[string]any)["post"].(map[string]any)
	response := analyze["responses"].(map[string]any)["200"].(map[string]any)
	if _, ok := response["content"].(map[string]any)["text/event-stream"]; !ok {
		t.Fatal("actual SSE contract missing")
	}
	confirm := paths["/graph-ai/actions/{id}/confirm"].(map[string]any)["post"].(map[string]any)
	security := confirm["security"].([]any)
	if len(security) != 1 {
		t.Fatal("confirmation has nonhuman authentication")
	}
	if _, ok := security[0].(map[string]any)["cookieAuth"]; !ok {
		t.Fatal("confirmation is not cookie-only")
	}
	schema := spec["components"].(map[string]any)["schemas"].(map[string]any)["GraphAIAnalyze"].(map[string]any)
	if schema["properties"].(map[string]any)["consent"].(map[string]any)["const"] != true {
		t.Fatal("explicit consent absent")
	}
}
