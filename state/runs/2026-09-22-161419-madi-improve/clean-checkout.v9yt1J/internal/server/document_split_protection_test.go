package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPostgresDocumentSplitProtectionAndExpiredTicket(t *testing.T) {
	s, owner, ctx, _, wid := jobTestFixture(t)
	selected := "CHILD_PROTECTION_SECRET\n"
	md := "# 원본\n\n" + selected + "\n유지할 문단\n"
	source := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "원본", "markdown": md, "visibility": "private"}, 200))
	id := str(source, "id")
	preview := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/split-preview", map[string]any{"expected_version": 1, "start_byte": strings.Index(md, selected), "end_byte": strings.Index(md, selected) + len(selected), "selected_text": selected, "title": "새 초안"}, 200))
	policy := defaultProtectionSettings()
	policy["enabled"], policy["mode"], policy["custom_terms"] = true, "block", []string{"CHILD_PROTECTION_SECRET"}
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	body := map[string]any{"ticket": preview["ticket"], "client_request_id": newID(), "consent": true}
	response := owner.request("POST", "/api/v1/documents/"+id+"/split", body, 422)
	if strings.Contains(string(response), selected) {
		t.Fatal("raw protected source in error")
	}
	got := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(got, "markdown") != md || number(got, "version", 0) != 1 {
		t.Fatal("child protection block changed source")
	}
	var children, receipts int
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM documents WHERE parent_id=$1", id).Scan(&children); e != nil || children != 0 {
		t.Fatal("blocked child persisted", children, e)
	}
	if e := s.DB.QueryRow(ctx, "SELECT count(*) FROM document_split_receipts WHERE source_id=$1", id).Scan(&receipts); e != nil || receipts != 0 {
		t.Fatal("blocked receipt persisted", receipts, e)
	}
	policy["mode"] = "mask"
	if _, e := s.DB.Exec(ctx, "UPDATE protection_settings SET data=$1", jsonValue(policy)); e != nil {
		t.Fatal(e)
	}
	var ticket documentSplitTicket
	raw, e := s.decrypt(str(preview, "ticket"))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal([]byte(raw), &ticket); e != nil {
		t.Fatal(e)
	}
	ticket.Expires = time.Now().Add(-time.Second).Unix()
	expired, e := s.encrypt(string(jsonValue(ticket)))
	if e != nil {
		t.Fatal(e)
	}
	expiredBody := map[string]any{"ticket": expired, "client_request_id": newID(), "consent": true}
	owner.request("POST", "/api/v1/documents/"+id+"/split", expiredBody, 409)
	result := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/split", body, 200))
	child := result["child"].(map[string]any)
	if strings.Contains(str(child, "markdown"), "CHILD_PROTECTION_SECRET") || str(child, "markdown") == "" {
		t.Fatal("child canonical masking missing", child)
	}
	if number(result["source"].(map[string]any), "version", 0) != 2 {
		t.Fatal("masked atomic source CAS failed")
	}
}
