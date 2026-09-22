package server

import "testing"

func TestWorksetsOpenAPI(t *testing.T) {
	paths, schemas := map[string]any{}, map[string]any{}
	for path, methods := range map[string][]string{"/worksets": {"get", "post"}, "/worksets/{id}": {"get", "put", "delete"}, "/documents/{id}/passport": {"get"}, "/documents/{id}/cleanup-preview": {"post"}} {
		ops := map[string]any{}
		for _, method := range methods {
			ops[method] = map[string]any{}
		}
		paths[path] = ops
	}
	worksetsOpenAPI(paths, schemas)
	if len(schemas) != 4 {
		t.Fatal("schema count", len(schemas))
	}
	for _, entry := range []struct{ path, method string }{{"/worksets", "get"}, {"/worksets/{id}", "delete"}, {"/documents/{id}/cleanup-preview", "post"}} {
		op := paths[entry.path].(map[string]any)[entry.method].(map[string]any)
		if op["security"].([]any)[0].(map[string]any)["cookieAuth"] == nil {
			t.Fatal("private browser gate missing")
		}
		if entry.method != "get" && op["requestBody"].(map[string]any)["required"] != true {
			t.Fatal("required CAS body missing")
		}
	}
}
