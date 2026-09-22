package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresWorksetsPrivacyCurrentACLAndContext(t *testing.T) {
	_, owner, _, _, wid := jobTestFixture(t)
	owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "worksets-reader@example.test", "name": "참고 사용자", "role": "viewer", "password": "Worksets-Test-Password-2026!"}, 200)
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "worksets-reader@example.test", "role": "viewer"}, 200)
	reader := newIntegrationTestClient(t, owner.base)
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": "worksets-reader@example.test", "password": "Worksets-Test-Password-2026!"}, 200)
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "권한 있는 참고 자료", "markdown": "첫줄\n\n둘째줄\n", "visibility": "workspace"}, 200))
	source := map[string]any{"kind": "document", "resource_id": doc["id"], "context": map[string]any{"version": 1, "line": 3, "scroll_y": 300, "selection_start": 0, "selection_end": 2}}
	input := map[string]any{"workspace_id": wid, "name": "내 참고 선반", "kind": "reference", "items": []any{source}}
	set := testJSONObject(t, reader.request("POST", "/api/v1/worksets", input, 200))
	id := str(set, "id")
	owner.request("GET", "/api/v1/worksets/"+id, nil, 404)
	if strings.Contains(string(owner.request("GET", "/api/v1/worksets?workspace_id="+wid, nil, 200)), "내 참고 선반") {
		t.Fatal("admin read another user's private workset")
	}
	data := testJSONObject(t, reader.request("GET", "/api/v1/worksets/"+id, nil, 200))
	resolved := data["resolved"].([]any)[0].(map[string]any)
	if !boolean(resolved, "available") || !strings.Contains(str(resolved, "url"), "line=3") || resolved["restore_context"] == nil {
		t.Fatal("document context not restorable", resolved)
	}
	owner.request("PUT", "/api/v1/documents/"+str(doc, "id"), map[string]any{"version": 1, "markdown": "서버의 새로운 원문"}, 200)
	data = testJSONObject(t, reader.request("GET", "/api/v1/worksets/"+id, nil, 200))
	resolved = data["resolved"].([]any)[0].(map[string]any)
	if !boolean(resolved, "context_changed") || strings.Contains(str(resolved, "url"), "line=") || resolved["restore_context"] != nil {
		t.Fatal("stale selection restored", resolved)
	}
	owner.request("PUT", "/api/v1/documents/"+str(doc, "id"), map[string]any{"version": 2, "visibility": "private"}, 200)
	raw := reader.request("GET", "/api/v1/worksets/"+id, nil, 200)
	if strings.Contains(string(raw), "권한 있는 참고 자료") || strings.Contains(string(raw), "서버의 새로운 원문") {
		t.Fatal("revoked source metadata leaked", string(raw))
	}
	input["expected_version"] = 1
	input["name"] = "정렬 뒤에도 숨긴 슬롯 보존"
	reader.request("PUT", "/api/v1/worksets/"+id, input, 200)
	reader.request("PUT", "/api/v1/worksets/"+id, input, 409)
	reader.request("POST", "/api/v1/worksets", map[string]any{"workspace_id": wid, "name": "권한 없는 추가", "kind": "reference", "items": []any{source}}, 403)
	reader.request("DELETE", "/api/v1/worksets/"+id, map[string]any{"expected_version": 1}, 409)
	reader.request("DELETE", "/api/v1/worksets/"+id, map[string]any{"expected_version": 2}, 200)
	owner.request("GET", "/api/v1/documents/"+str(doc, "id"), nil, 200)
	// A safe saved database view restores the actual configuration, not just its label.
	db := testJSONObject(t, owner.request("POST", "/api/v1/databases", map[string]any{"workspace_id": wid, "name": "보기 참고"}, 200))
	view := testJSONObject(t, reader.request("POST", "/api/v1/databases/"+str(db, "id")+"/views", map[string]any{"name": "개인 보드", "visibility": "private", "data": map[string]any{"view": "board", "board_property_id": "status", "filters": []any{}}}, 200))
	set = testJSONObject(t, reader.request("POST", "/api/v1/worksets", map[string]any{"workspace_id": wid, "name": "업무 보기", "kind": "workset", "items": []any{map[string]any{"kind": "database", "resource_id": db["id"], "context": map[string]any{"view_id": view["id"]}}}}, 200))
	data = testJSONObject(t, reader.request("GET", "/api/v1/worksets/"+str(set, "id"), nil, 200))
	resolved = data["resolved"].([]any)[0].(map[string]any)
	if !strings.Contains(str(resolved, "url"), "view=board") {
		t.Fatal("saved view configuration missing", resolved)
	}
	for _, c := range []map[string]any{{"url": "https://outside.example"}, {"line": -1, "version": 1}, {"selection_start": 4, "selection_end": 1, "version": 1}, {"scroll_y": 20}, {"mode": "source"}} {
		reader.request("POST", "/api/v1/worksets", map[string]any{"workspace_id": wid, "name": "안전하지 않은 위치", "kind": "workset", "items": []any{map[string]any{"kind": "document", "resource_id": doc["id"], "context": c}}}, 400)
	}
}

func TestPostgresWorksetPassportCleanupCASAndFrontMatter(t *testing.T) {
	_, owner, _, _, wid := jobTestFixture(t)
	md := "---\n# 소유자 설명\naliases: [기존별칭]\ntags: [기존태그]\ncustom: keep-me\n---\n\n# GPU 서버 운영\n\nGPU 서버 운영을 위한 점검과 배포 기록입니다. #운영\n\n```text\n이 코드와 마지막 줄은 그대로\n```\n"
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "제목 없는 문서", "markdown": md, "visibility": "private"}, 200))
	id := str(doc, "id")
	passport := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+id+"/passport", nil, 200))
	if passport["markdown"] != nil || str(passport, "owner_name") == "" || str(passport, "visibility") != "private" {
		t.Fatal("passport metadata contract", passport)
	}
	preview := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/cleanup-preview", map[string]any{"expected_version": 1}, 200))
	if str(preview["after"].(map[string]any), "markdown") != md {
		t.Fatal("unchosen preview reformatted original")
	}
	preview = testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/cleanup-preview", map[string]any{"expected_version": 1, "title": "GPU 서버 운영", "tags": []string{"기존태그", "운영"}}, 200))
	after := preview["after"].(map[string]any)
	next := str(after, "markdown")
	if !boolean(preview, "front_matter_reformatted") || !strings.HasSuffix(next, md[strings.Index(md, "\n\n# GPU"):]) || !strings.Contains(next, "소유자 설명") || !strings.Contains(next, "keep-me") || !strings.Contains(next, "기존별칭") {
		t.Fatal("front matter preservation", next)
	}
	before := testJSONObject(t, owner.request("GET", "/api/v1/documents/"+id, nil, 200))
	if str(before, "markdown") != md || number(before, "version", 0) != 1 {
		t.Fatal("preview mutated document")
	}
	body := map[string]any{"version": 1, "title": after["title"], "tags": after["tags"], "markdown": after["markdown"]}
	saved := testJSONObject(t, owner.request("PUT", "/api/v1/documents/"+id, body, 200))
	if str(saved, "visibility") != "private" || str(saved, "title") != "GPU 서버 운영" || !strings.Contains(string(jsonValue(saved["tags"])), "운영") {
		t.Fatal("cleanup canonical save", saved)
	}
	owner.request("PUT", "/api/v1/documents/"+id, body, 409)
	owner.request("POST", "/api/v1/documents/"+id+"/cleanup-preview", map[string]any{"expected_version": 1}, 409)
	// Cleanup candidates contain no Markdown code tokens and remain bounded.
	var tags []string
	if e := json.Unmarshal(jsonValue(preview["tag_candidates"]), &tags); e != nil || len(tags) > 32 {
		t.Fatal("candidate bound", tags, e)
	}
}

func TestWorksetContextStrictBounds(t *testing.T) {
	for _, value := range []struct {
		raw   string
		valid bool
	}{
		{`{"version":2147483647,"range":{"start":[0,1],"end":[0,2],"startOffset":0,"endOffset":10}}`, true},
		{`{"version":2147483648}`, false},
		{`{"version":1,"range":{"start":["script"],"end":[],"startOffset":0,"endOffset":0}}`, false},
		{`{"range":{"start":[],"end":[],"startOffset":0,"endOffset":0}}`, false},
		{`{"version":1,"range":{"start":[],"end":[],"startOffset":0,"endOffset":0,"html":"unsafe"}}`, false},
		{`{"version":1,"scroll_y":10000001}`, false},
		{`{"version":1,"scroll_y":1.5}`, false},
	} {
		var c map[string]any
		if e := json.Unmarshal([]byte(value.raw), &c); e != nil {
			t.Fatal(e)
		}
		e := validateWorksetContext(worksetItem{Kind: "document", Context: c})
		if (e == nil) != value.valid {
			t.Errorf("context %s: %v", value.raw, e)
		}
	}
}

func TestPostgresWorksetsIsolationProtectionAndReadonly(t *testing.T) {
	s, owner, _, _, wid := jobTestFixture(t)
	owner.request("POST", "/api/v1/admin/users", map[string]any{"email": "workset-privacy@example.test", "name": "열람자", "role": "viewer", "password": "Workset-Privacy-Password-2026!"}, 200)
	owner.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "workset-privacy@example.test", "role": "viewer"}, 200)
	reader := newIntegrationTestClient(t, owner.base)
	reader.request("POST", "/api/v1/auth/login", map[string]any{"email": "workset-privacy@example.test", "password": "Workset-Privacy-Password-2026!"}, 200)
	doc := testJSONObject(t, owner.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "정리 대상", "markdown": "# PRIVATE-TOPIC\n\n참고 자료\n", "visibility": "workspace"}, 200))
	id := str(doc, "id")
	reader.request("GET", "/api/v1/documents/"+id+"/passport", nil, 200)
	reader.request("POST", "/api/v1/documents/"+id+"/cleanup-preview", map[string]any{"expected_version": 1}, 404)
	otherWS := testJSONObject(t, owner.request("POST", "/api/v1/workspaces", map[string]any{"name": "다른 영역"}, 200))
	owner.request("POST", "/api/v1/worksets", map[string]any{"workspace_id": otherWS["id"], "name": "영역 혼동", "kind": "workset", "items": []any{map[string]any{"kind": "document", "resource_id": id, "context": map[string]any{}}}}, 403)
	if _, e := s.DB.Exec(t.Context(), `UPDATE protection_settings SET data=data||'{"enabled":true,"mode":"mask","detectors":[],"custom_terms":["PRIVATE-TOPIC"]}'::jsonb,revision=revision+1 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	preview := testJSONObject(t, owner.request("POST", "/api/v1/documents/"+id+"/cleanup-preview", map[string]any{"expected_version": 1}, 200))
	if strings.Contains(string(jsonValue(preview["title_candidates"])), "PRIVATE-TOPIC") || strings.Contains(str(preview["after"].(map[string]any), "markdown"), "PRIVATE-TOPIC") {
		t.Fatal("cleanup candidate/canonical output bypassed live mask policy")
	}
	workset := testJSONObject(t, owner.request("POST", "/api/v1/worksets", map[string]any{"workspace_id": wid, "name": "PRIVATE-TOPIC 선반", "kind": "reference", "items": []any{}}, 200))
	fetched := owner.request("GET", "/api/v1/worksets/"+str(workset, "id"), nil, 200)
	if strings.Contains(string(fetched), "PRIVATE-TOPIC") {
		t.Fatal("workset name bypassed protection")
	}
	owner.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": 1, "visibility": "private"}, 200)
	reader.request("GET", "/api/v1/documents/"+id+"/passport", nil, 404)
	reader.request("POST", "/api/v1/documents/"+id+"/cleanup-preview", map[string]any{"expected_version": 2}, 404)
}
