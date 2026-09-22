package server

import "testing"

func TestUXReviewOpenAPI(t *testing.T) {
	paths, schemas := map[string]any{}, map[string]any{}
	for path, methods := range map[string][]string{"/documents/{id}/split-preview": {"post"}, "/documents/{id}/split": {"post"}, "/documents/{id}/ai-selection": {"get", "post"}, "/ai/selection-drafts": {"post"}, "/documents/{id}/access-requests": {"post"}, "/access-requests": {"get"}, "/access-requests/{id}": {"put"}, "/documents/{id}": {"delete"}, "/documents/{id}/restore": {"post"}, "/documents/{id}/access-preview": {"post"}} {
		ops := map[string]any{}
		for _, method := range methods {
			ops[method] = map[string]any{"responses": map[string]any{"200": map[string]any{}}, "security": []any{map[string]any{"bearerAuth": []string{}}}}
		}
		paths[path] = ops
	}
	for path, methods := range map[string][]string{"/databases/{id}/editing": {"get"}, "/databases/{id}/edit-cells": {"post"}, "/databases/{id}/views": {"get", "post"}, "/databases/{id}/views/{viewID}": {"put", "delete"}, "/databases/{id}/view-preference": {"put"}, "/databases/{id}/rows/{rowId}": {"put", "delete"}} {
		ops := map[string]any{}
		for _, method := range methods {
			ops[method] = map[string]any{"responses": map[string]any{"200": map[string]any{}}}
		}
		paths[path] = ops
	}
	uxReviewOpenAPI(paths, schemas)
	if len(paths) != 16 || len(schemas) != 16 {
		t.Fatal("unexpected path/schema catalogue", len(paths), len(schemas))
	}
	get := func(path, method string) map[string]any { return paths[path].(map[string]any)[method].(map[string]any) }
	sse := get("/documents/{id}/ai-selection", "post")["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)
	if sse["text/event-stream"] == nil {
		t.Fatal("SSE contract missing")
	}
	ack := get("/documents/{id}/access-requests", "post")["responses"].(map[string]any)
	if ack["202"] == nil || ack["200"] != nil {
		t.Fatal("indistinguishable acceptance status missing")
	}
	for _, path := range []string{"/documents/{id}", "/documents/{id}/restore"} {
		method := "delete"
		if path != "/documents/{id}" {
			method = "post"
		}
		if get(path, method)["requestBody"].(map[string]any)["required"] != false {
			t.Fatal("legacy optional CAS compatibility missing")
		}
	}
	if get("/access-requests/{id}", "put")["security"].([]any)[0].(map[string]any)["cookieAuth"] == nil {
		t.Fatal("personal-only endpoint documented as token API")
	}
	if get("/databases/{id}/rows/{rowId}", "delete")["requestBody"].(map[string]any)["required"] != false || get("/databases/{id}/views/{viewID}", "delete")["requestBody"].(map[string]any)["required"] != true {
		t.Fatal("optional legacy row deletion and mandatory saved-view CAS must remain distinct")
	}
	if get("/databases/{id}/edit-cells", "post")["security"].([]any)[0].(map[string]any)["cookieAuth"] == nil {
		t.Fatal("atomic browser editing documented without current session requirement")
	}
}
