package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Both clients use the production router and credentials issued by the real
// authentication endpoints. Raw bodies also let us exercise malformed JSON.
func relationValidationClient(t *testing.T, external bool) (*Server, func(string, string, string) *httptest.ResponseRecorder) {
	t.Helper()
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	prefix := "/api/v1/assets/"
	var secret string
	if external {
		prefix = "/api/v1/external/assets/"
		secret = createAPIKeySecret(t, server, cookie, csrf, "relation-validation", "relations.write")
	}
	return server, func(method, path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, prefix+path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		if external {
			request.Header.Set("Authorization", "Bearer "+secret)
		} else {
			request.AddCookie(cookie)
			request.Header.Set("X-CSRF-Token", csrf)
		}
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		return response
	}
}

func relationValidationAssets(t *testing.T, server *Server) (string, string) {
	t.Helper()
	insert := func() string {
		key := uuid.NewString()
		return insertSoftwareTestAsset(t, server, key, key, "host", `{}`, 1, time.Now().UTC())
	}
	return insert(), insert()
}

func relationValidationBody(targetID, relationType string) string {
	body, _ := json.Marshal(map[string]string{"target_asset_id": targetID, "relation_type": relationType})
	return string(body)
}

// relationValidationBody cannot carry a numeric confidence, so range cases
// build their own body and omit the field entirely when confidence is nil.
func relationConfidenceBody(targetID, relationType string, confidence *float64) string {
	payload := map[string]any{"target_asset_id": targetID, "relation_type": relationType}
	if confidence != nil {
		payload["confidence"] = *confidence
	}
	body, _ := json.Marshal(payload)
	return string(body)
}

func relationConfidence(t *testing.T, server *Server, id string) float64 {
	t.Helper()
	var confidence float64
	if err := server.database.DB().QueryRow(
		`SELECT confidence FROM asset_relations WHERE id=$1`, id,
	).Scan(&confidence); err != nil {
		t.Fatalf("read relation confidence: %v", err)
	}
	return confidence
}

func assertRelationResponse(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if got := errorCode(t, response); response.Code != status || got != code {
		t.Errorf("relation response = %d %q, want %d %s: %s", response.Code, got, status, code, response.Body.String())
	}
}

func assertRelationState(t *testing.T, server *Server, id, sourceID, targetID string, ended bool) {
	t.Helper()
	var source, target string
	var validTo any
	if err := server.database.DB().QueryRow(
		`SELECT source_asset_id,target_asset_id,valid_to FROM asset_relations WHERE id=$1`, id,
	).Scan(&source, &target, &validTo); err != nil {
		t.Fatalf("read relation: %v", err)
	}
	if source != sourceID || target != targetID || (validTo != nil) != ended {
		t.Errorf("relation state = (%s, %s, ended=%v), want (%s, %s, ended=%v)",
			source, target, validTo != nil, sourceID, targetID, ended)
	}
}

func createdRelationID(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	if response.Code != http.StatusCreated {
		t.Fatalf("create relation = %d, want 201: %s", response.Code, response.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("decode created relation: %s (%v)", response.Body.String(), err)
	}
	return created.ID
}

func TestRelationCreateRejectsInvalidInputWithoutWrites(t *testing.T) {
	for _, external := range []bool{false, true} {
		name := "console"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			server, request := relationValidationClient(t, external)
			for _, position := range []string{"source", "target"} {
				for label := range assetIDSpellings(uuid.NewString()) {
					t.Run(position+"/"+label, func(t *testing.T) {
						source, target := relationValidationAssets(t, server)
						if position == "source" {
							source = assetIDSpellings(source)[label]
						} else {
							target = assetIDSpellings(target)[label]
						}
						before := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`)
						audits := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`)
						response := request(http.MethodPost, source+"/relations", relationValidationBody(target, "depends_on"))
						assertRelationResponse(t, response, http.StatusBadRequest, "INVALID_RELATION")
						if got := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`); got != before {
							t.Errorf("relations after rejected create = %d, want unchanged %d", got, before)
						}
						if got := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`); got != audits {
							t.Errorf("create audits after rejected create = %d, want unchanged %d", got, audits)
						}
					})
				}
			}
			for _, label := range []string{"empty target", "empty relation type", "malformed JSON"} {
				t.Run(label, func(t *testing.T) {
					source, target := relationValidationAssets(t, server)
					body := "{"
					switch label {
					case "empty target":
						body = relationValidationBody("", "depends_on")
					case "empty relation type":
						body = relationValidationBody(target, "")
					}
					before := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`)
					audits := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`)
					assertRelationResponse(t, request(http.MethodPost, source+"/relations", body), http.StatusBadRequest, "INVALID_RELATION")
					if got := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`); got != before {
						t.Errorf("relations = %d, want unchanged %d", got, before)
					}
					if got := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`); got != audits {
						t.Errorf("create audits = %d, want unchanged %d", got, audits)
					}
				})
			}
		})
	}
}

func TestRelationDeleteValidatesIDBeforeWriting(t *testing.T) {
	for _, external := range []bool{false, true} {
		name := "console"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			server, request := relationValidationClient(t, external)
			labels := []string{"not a uuid at all", "urn prefixed", "brace wrapped", "unhyphenated", "missing canonical", "upper case"}
			for _, label := range labels {
				t.Run(label, func(t *testing.T) {
					// A fresh active relation prevents a prior request from masking a write.
					source, target := relationValidationAssets(t, server)
					id := createdRelationID(t, request(http.MethodPost, source+"/relations", relationValidationBody(target, "depends_on")))
					spelling := assetIDSpellings(id)[label]
					if label == "missing canonical" {
						spelling = uuid.NewString()
					}
					audits := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.delete'`)
					status, code, ended := http.StatusNotFound, "RELATION_NOT_FOUND", false
					if label == "upper case" {
						spelling = strings.ToUpper(id)
						status, code, ended = http.StatusOK, "", true
						audits++
					}
					assertRelationResponse(t, request(http.MethodDelete, source+"/relations/"+spelling, ""), status, code)
					assertRelationState(t, server, id, source, target, ended)
					if got := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.delete'`); got != audits {
						t.Errorf("delete audits = %d, want unchanged %d", got, audits)
					}
				})
			}
		})
	}
}

func TestRelationUpperCaseIDsAreCanonicalInRowsAndAudit(t *testing.T) {
	for _, external := range []bool{false, true} {
		name := "console"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			server, request := relationValidationClient(t, external)
			source, target := relationValidationAssets(t, server)
			path := strings.ToUpper(source) + "/relations"
			id := createdRelationID(t, request(http.MethodPost, path, relationValidationBody(strings.ToUpper(target), "depends_on")))
			assertRelationState(t, server, id, source, target, false)
			var afterJSON string
			if err := server.database.DB().QueryRow(`SELECT after_json FROM audit_logs WHERE action='relation.create' AND resource_id=$1`, id).Scan(&afterJSON); err != nil {
				t.Fatalf("read create audit: %v", err)
			}
			var after struct {
				TargetID string `json:"target_asset_id"`
			}
			if err := json.Unmarshal([]byte(afterJSON), &after); err != nil {
				t.Fatalf("decode create audit: %v", err)
			}
			if after.TargetID != target {
				t.Errorf("audit target = %q, want %q", after.TargetID, target)
			}
			deletePath := path + "/" + strings.ToUpper(id)
			assertRelationResponse(t, request(http.MethodDelete, deletePath, ""), http.StatusOK, "")
			assertRelationState(t, server, id, source, target, true)
			var validToBefore, validToAfter any
			if err := server.database.DB().QueryRow(`SELECT valid_to FROM asset_relations WHERE id=$1`, id).Scan(&validToBefore); err != nil {
				t.Fatal(err)
			}
			assertRelationResponse(t, request(http.MethodDelete, deletePath, ""), http.StatusNotFound, "RELATION_NOT_FOUND")
			if err := server.database.DB().QueryRow(`SELECT valid_to FROM asset_relations WHERE id=$1`, id).Scan(&validToAfter); err != nil {
				t.Fatal(err)
			}
			if validToBefore != validToAfter {
				t.Errorf("repeated delete changed valid_to: %v -> %v", validToBefore, validToAfter)
			}
			if got := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.delete'`); got != 1 {
				t.Errorf("delete audits = %d, want 1", got)
			}
			var resourceID string
			if err := server.database.DB().QueryRow(`SELECT resource_id FROM audit_logs WHERE action='relation.delete'`).Scan(&resourceID); err != nil {
				t.Fatalf("read delete audit: %v", err)
			}
			if resourceID != id {
				t.Errorf("delete audit resource_id = %q, want %q", resourceID, id)
			}
		})
	}
}

// openapi declares `confidence: {minimum: 0, maximum: 1}` on both create paths
// and on the GET /relations response, but neither dialect has a CHECK on
// asset_relations.confidence, so an out-of-range write would make the read path
// answer with a value its own schema forbids.
func TestRelationCreateRejectsConfidenceOutsideDeclaredRange(t *testing.T) {
	for _, external := range []bool{false, true} {
		name := "console"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			server, request := relationValidationClient(t, external)
			for _, confidence := range []float64{-0.5, 1.5, -1, 42} {
				t.Run(strconv.FormatFloat(confidence, 'f', -1, 64), func(t *testing.T) {
					source, target := relationValidationAssets(t, server)
					before := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`)
					audits := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`)
					body := relationConfidenceBody(target, "depends_on", &confidence)
					assertRelationResponse(t, request(http.MethodPost, source+"/relations", body), http.StatusBadRequest, "INVALID_RELATION")
					if got := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`); got != before {
						t.Errorf("relations after rejected create = %d, want unchanged %d", got, before)
					}
					if got := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`); got != audits {
						t.Errorf("create audits after rejected create = %d, want unchanged %d", got, audits)
					}
				})
			}
		})
	}
}

// The boundaries and an omitted field stay accepted. An explicit 0 is still
// promoted to 1 — openapi's `default: 1` only covers a missing value, so that
// promotion is wrong, but telling "absent" from 0 needs a *float64 input and is
// a separate change; this pins today's behaviour so it cannot drift silently.
func TestRelationCreateStoresConfidenceWithinDeclaredRange(t *testing.T) {
	zero, one, fractional := 0.0, 1.0, 0.8
	for _, external := range []bool{false, true} {
		name := "console"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			server, request := relationValidationClient(t, external)
			for _, testCase := range []struct {
				name       string
				confidence *float64
				want       float64
			}{
				{"omitted defaults to one", nil, 1},
				{"explicit zero is promoted to one", &zero, 1},
				{"upper boundary", &one, 1},
				{"fractional", &fractional, 0.8},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					source, target := relationValidationAssets(t, server)
					body := relationConfidenceBody(target, "depends_on", testCase.confidence)
					id := createdRelationID(t, request(http.MethodPost, source+"/relations", body))
					assertRelationState(t, server, id, source, target, false)
					if got := relationConfidence(t, server, id); got != testCase.want {
						t.Errorf("stored confidence = %v, want %v", got, testCase.want)
					}
				})
			}
		})
	}
}

func TestRelationCreatePreservesConflicts(t *testing.T) {
	for _, external := range []bool{false, true} {
		name := "console"
		if external {
			name = "external"
		}
		t.Run(name, func(t *testing.T) {
			server, request := relationValidationClient(t, external)
			source, target := relationValidationAssets(t, server)
			id := createdRelationID(t, request(http.MethodPost, source+"/relations", relationValidationBody(target, "depends_on")))
			for _, tc := range []struct{ name, source, target string }{
				{"missing source", uuid.NewString(), target},
				{"missing target", source, uuid.NewString()},
				{"duplicate", source, target},
			} {
				t.Run(tc.name, func(t *testing.T) {
					assertRelationResponse(t, request(http.MethodPost, tc.source+"/relations", relationValidationBody(tc.target, "depends_on")), http.StatusConflict, "RELATION_CONFLICT")
					assertRelationState(t, server, id, source, target, false)
					if got := countRows(t, server, `SELECT COUNT(*) FROM asset_relations`); got != 1 {
						t.Errorf("relations = %d, want 1", got)
					}
					if got := countRows(t, server, `SELECT COUNT(*) FROM audit_logs WHERE action='relation.create'`); got != 1 {
						t.Errorf("create audits = %d, want 1", got)
					}
				})
			}
		})
	}
}
