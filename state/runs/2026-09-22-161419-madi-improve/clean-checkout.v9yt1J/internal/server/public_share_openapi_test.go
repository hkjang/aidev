package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestPublicShareOpenAPIAnonymousHeaderContract(t *testing.T) {
	s := &Server{Version: "test", apiRoutes: []string{"POST /api/v1/documents/{id}/public-shares", "PUT /api/v1/admin/information-protection"}}
	w := httptest.NewRecorder()
	s.openAPI(w, httptest.NewRequest("GET", "/api/v1/openapi.json", nil))
	var spec map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &spec); e != nil {
		t.Fatal(e)
	}
	paths := spec["paths"].(map[string]any)
	for _, suffix := range []string{"", "/unlock", "/attachments/{file}"} {
		method := "get"
		if suffix == "/unlock" {
			method = "post"
		}
		op := paths["/public-shares/{share}"+suffix].(map[string]any)[method].(map[string]any)
		if len(op["security"].([]any)) != 0 {
			t.Fatal("public route inherits user credentials")
		}
		found := false
		for _, raw := range op["parameters"].([]any) {
			param := raw.(map[string]any)
			if param["name"] == "X-Madi-Share-Token" && param["in"] == "header" && param["required"] == true {
				found = true
			}
		}
		if !found {
			t.Fatal("share token header absent", suffix)
		}
	}
	op := paths["/documents/{id}/public-shares"].(map[string]any)["post"].(map[string]any)
	if _, ok := op["security"].([]any)[0].(map[string]any)["cookieAuth"]; !ok {
		t.Fatal("owner share management must be cookie-only")
	}
	if op["requestBody"] == nil {
		t.Fatal("explicit share confirmation body absent")
	}
}
