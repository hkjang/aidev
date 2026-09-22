package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func structuredTestProposal(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	for line := range strings.SplitSeq(string(raw), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event) == nil {
			if p, ok := event["proposal"].(map[string]any); ok {
				return p
			}
		}
	}
	t.Fatalf("missing completed structured proposal: %s", raw)
	return nil
}
func TestPostgresStructuredDraftStreamReviewAtomicRowAndACL(t *testing.T) {
	s, owner, other, wid, _ := collaborationTestSetup(t)
	ctx := t.Context()
	requests := make(chan map[string]any, 10)
	output := `{"fields":[{"property_id":"amount","value":2,"quote":"2개"},{"property_id":"state","value":"미등록","quote":"상태는 운영"}]}`
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		if json.NewDecoder(r.Body).Decode(&data) != nil {
			w.WriteHeader(400)
			return
		}
		requests <- data
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": output}}}}))
	}))
	defer provider.Close()
	owner.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "structured-fixture", "ai_max_tokens": 262144}, 200)
	props := []any{map[string]any{"id": "amount", "name": "수량", "type": "number"}, map[string]any{"id": "state", "name": "상태", "type": "select", "options": []string{"운영", "검토"}}, map[string]any{"id": "hidden", "name": "PROPERTY_NOT_TRANSMITTED", "type": "text"}}
	db := testJSONObject(t, owner.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "DB_NAME_NOT_TRANSMITTED", "properties": props}, 200))
	dbID := str(db, "id")
	md := "문서의 앞 PRIVATE_NOT_SELECTED\nGPU 수량은 2개입니다. 상태는 운영입니다.\n끝 SECRET_NOT_SELECTED"
	selected := "GPU 수량은 2개입니다. 상태는 운영입니다."
	start := strings.Index(md, selected)
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "TITLE_NOT_TRANSMITTED", "markdown": md, "visibility": "private"}, 200))
	id := str(doc, "id")
	path := "/api/v1/documents/" + id
	meta := testJSONObject(t, owner.request("GET", path+"/structured-context?database_id="+dbID, nil, 200))
	input := map[string]any{"database_id": dbID, "expected_version": 1, "start_byte": start, "end_byte": start + len(selected), "selected_text": selected, "property_ids": []string{"amount", "state"}, "provider_fingerprint": meta["provider"].(map[string]any)["fingerprint"], "schema_hash": meta["schema_hash"], "destination_hash": meta["destination_hash"], "consent": true}
	input["consent"] = false
	owner.request("POST", path+"/structured-draft", input, 400)
	input["consent"] = true
	proposal := structuredTestProposal(t, owner.request("POST", path+"/structured-draft", input, 200))
	upstream := <-requests
	sent := string(jsonValue(upstream))
	for _, marker := range []string{"PRIVATE_NOT_SELECTED", "SECRET_NOT_SELECTED", "PROPERTY_NOT_TRANSMITTED", "DB_NAME_NOT_TRANSMITTED", "TITLE_NOT_TRANSMITTED"} {
		if strings.Contains(sent, marker) {
			t.Fatal("provider boundary leaked", marker)
		}
	}
	if upstream["stream"] != true || number(upstream, "max_tokens", 0) != 262144 {
		t.Fatal(upstream)
	}
	var n int
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_structured_drafts`).Scan(&n); e != nil || n != 0 {
		t.Fatal("generation persisted before consent", n, e)
	}
	fields := proposal["fields"].([]any)
	if !boolean(fields[0].(map[string]any), "valid") || boolean(fields[1].(map[string]any), "valid") {
		t.Fatal(fields)
	}
	save := map[string]any{"ticket": proposal["draft_ticket"], "consent": true}
	made := testJSONObject(t, owner.request("POST", "/api/v1/knowledge/structured-drafts", save, 201))
	did := str(made, "id")
	draftPath := "/api/v1/knowledge/structured-drafts/" + did
	owner.request("POST", "/api/v1/knowledge/structured-drafts", save, 201)
	var cipher string
	if e := s.DB.QueryRow(ctx, `SELECT ciphertext FROM knowledge_structured_drafts WHERE id=$1`, did).Scan(&cipher); e != nil || strings.Contains(cipher, "상태는 운영") {
		t.Fatal("plaintext draft", e)
	}
	other.request("GET", draftPath, nil, 404)
	review := map[string]any{"revision": 1, "fields": []structuredProviderField{{PropertyID: "amount", Value: 2, Quote: "2개"}, {PropertyID: "state", Value: "미등록", Quote: "상태는 운영"}}}
	owner.request("POST", draftPath+"/preview", review, 400)
	review["fields"].([]structuredProviderField)[1].Value = "운영"
	preview := testJSONObject(t, owner.request("POST", draftPath+"/preview", review, 200))
	if !boolean(preview["fields"].([]any)[1].(map[string]any), "human_edited") {
		t.Fatal("human correction not separated", preview)
	}
	apply := map[string]any{"review_ticket": preview["review_ticket"], "consent": false}
	owner.request("POST", draftPath+"/apply", apply, 400)
	apply["consent"] = true
	// Fail after canonical row creation: both row and receipt must roll back.
	if _, e := s.DB.Exec(ctx, `CREATE FUNCTION madi_structured_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture receipt failure'; END $$; CREATE TRIGGER madi_structured_fail BEFORE UPDATE ON knowledge_structured_drafts FOR EACH ROW WHEN(NEW.state='committed') EXECUTE FUNCTION madi_structured_fail()`); e != nil {
		t.Fatal(e)
	}
	owner.request("POST", draftPath+"/apply", apply, 500)
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM database_rows WHERE database_id=$1`, dbID).Scan(&n); e != nil || n != 0 {
		t.Fatal("partial row persisted", n, e)
	}
	if _, e := s.DB.Exec(ctx, `DROP TRIGGER madi_structured_fail ON knowledge_structured_drafts`); e != nil {
		t.Fatal(e)
	}
	row := testJSONObject(t, owner.request("POST", draftPath+"/apply", apply, 200))
	replay := testJSONObject(t, owner.request("POST", draftPath+"/apply", apply, 200))
	if replay["id"] != row["id"] || !boolean(replay, "replayed") {
		t.Fatal("duplicate row replay", replay, row)
	}
	view := testJSONObject(t, owner.request("GET", draftPath, nil, 200))
	if str(view, "state") != "committed" || number(view, "revision", 0) != 2 || len(view["reviewed_fields"].([]any)) != 2 {
		t.Fatal(view)
	}
	owner.request("PUT", path, map[string]any{"version": 1, "markdown": md + "\n이후 변경"}, 200)
	view = testJSONObject(t, owner.request("GET", draftPath, nil, 200))
	if boolean(view, "fresh") {
		t.Fatal("stale source shown fresh", view)
	}
	owner.request("POST", draftPath+"/apply", apply, 409)
	owner.request("POST", "/api/v1/knowledge/structured-drafts", save, 409)
	if _, e := s.DB.Exec(ctx, `DELETE FROM documents WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	owner.request("GET", draftPath, nil, 404)
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM knowledge_structured_drafts WHERE id=$1`, did).Scan(&n); e != nil || n != 0 {
		t.Fatal("deleted source retained quote", n, e)
	}
	if e := s.DB.QueryRow(ctx, `SELECT count(*) FROM database_rows WHERE database_id=$1`, dbID).Scan(&n); e != nil || n != 1 {
		t.Fatal("source deletion falsely recalled shared copy", n, e)
	}
}
