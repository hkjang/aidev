package server

import (
	"context"
	"strings"
	"testing"
)

func TestPostgresEditorBlockLiveACLAndEpoch(t *testing.T) {
	s, admin, reader, wid, _ := collaborationTestSetup(t)
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "동기화 원본", "markdown": "원본 블록"}, 0))
	id := str(created, "id")
	ctx := context.Background()
	session := collaborationTestSession(t, admin)
	initial, e := s.collaborationState(ctx, id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	doc := collaborationTestDoc("원본 블록")
	defer doc.Destroy()
	_, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: initial.Version, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	path := "/api/v1/documents/" + id + "/blocks/block-test-1"
	body := testJSONObject(t, reader.request("GET", path, nil, 200))
	if str(body, "markdown") != "원본 블록" {
		t.Fatal(body)
	}
	collaborationTestInsert(doc, " 실시간 갱신")
	_, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	body = testJSONObject(t, reader.request("GET", path, nil, 200))
	if !strings.Contains(str(body, "markdown"), "실시간 갱신") {
		t.Fatal(body)
	}
	currentDoc := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+id, nil, 200))
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": currentDoc["version"], "visibility": "private"}, 200)
	reader.request("GET", path, nil, 404)
	admin.request("GET", path, nil, 200)
	currentDoc = testJSONObject(t, admin.request("GET", "/api/v1/documents/"+id, nil, 200))
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": currentDoc["version"], "markdown": "원문 대체"}, 200)
	admin.request("GET", path, nil, 409)
}

func TestPostgresCollaborationOutboxAtomicActor(t *testing.T) {
	s, admin, _, wid, _ := collaborationTestSetup(t)
	ctx := context.Background()
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "편집 이벤트", "markdown": "본문"}, 0))
	id := str(created, "id")
	session := collaborationTestSession(t, admin)
	initial, e := s.collaborationState(ctx, id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	doc := collaborationTestDoc("본문")
	defer doc.Destroy()
	_, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: initial.Version, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(doc, " 수정")
	if _, e = s.DB.Exec(ctx, `UPDATE settings SET data=jsonb_set(data,'{approval_enabled}','true')`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `UPDATE documents SET status='published' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	// No request principal in ctx: authenticated session must supply original actor.
	_, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	var count int
	if e = s.DB.QueryRow(ctx, `SELECT count(*) FROM automation_events e JOIN sessions se ON se.user_id=e.actor_id WHERE se.token_hash=$1 AND resource_id=$2 AND type IN ('document.updated','document.status_changed')`, session, id).Scan(&count); e != nil || count != 2 {
		t.Fatalf("outbox actor/count: %d %v", count, e)
	}
	before, e := s.collaborationState(ctx, id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `CREATE FUNCTION reject_editor_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced outbox rollback'; END $$; CREATE TRIGGER reject_editor_outbox BEFORE INSERT ON automation_events FOR EACH ROW EXECUTE FUNCTION reject_editor_outbox()`); e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(doc, " 실패한 변경")
	if _, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: doc.EncodeStateAsUpdate()}); e == nil {
		t.Fatal("outbox failure accepted")
	}
	after, e := s.collaborationState(ctx, id, session, nil)
	if e != nil || before.Version != after.Version || before.Sequence != after.Sequence || before.Markdown != after.Markdown {
		t.Fatalf("outbox rollback not atomic: before=%+v after=%+v err=%v", before, after, e)
	}
}

func TestAdvancedCollaborationProjection(t *testing.T) {
	p := func(text string) *collaborationNode {
		return &collaborationNode{Type: "paragraph", Children: []*collaborationNode{{Type: "text", Text: text}}}
	}
	cases := []struct {
		node     *collaborationNode
		contains string
	}{
		{&collaborationNode{Type: "callout", Attrs: map[string]any{"type": "warning", "title": "주의"}, Children: []*collaborationNode{p("내용")}}, "> [!WARNING] 주의\n> 내용"},
		{&collaborationNode{Type: "inlineMath", Attrs: map[string]any{"latex": "x^2"}}, "$x^2$"},
		{&collaborationNode{Type: "footnoteDefinition", Attrs: map[string]any{"label": "1"}, Children: []*collaborationNode{p("각주")}}, "[^1]: 각주"},
		{&collaborationNode{Type: "paragraph", Attrs: map[string]any{"textAlign": "center"}, Children: []*collaborationNode{{Type: "text", Text: "정렬 <script>"}}}, `<p style="text-align:center">정렬 &lt;script&gt;</p>`},
		{&collaborationNode{Type: "table", Children: []*collaborationNode{{Type: "tableRow", Children: []*collaborationNode{{Type: "tableCell", Attrs: map[string]any{"colspan": int64(2), "colwidth": []any{int64(120), int64(180)}}, Children: []*collaborationNode{p("병합")}}}}}}, `<td colspan="2" colwidth="120,180"><p>병합</p></td>`},
	}
	for _, item := range cases {
		value, e := collaborationRender(item.node)
		if e != nil || !strings.Contains(value, item.contains) {
			t.Errorf("%s got %q err %v want %q", item.node.Type, value, e, item.contains)
		}
	}
	// A malformed hidden node in HTML tables must not silently vanish.
	_, e := collaborationHTML(&collaborationNode{Type: "futureUnknown", Children: []*collaborationNode{p("keep")}})
	if e == nil {
		t.Fatal("unknown HTML node discarded")
	}
}
