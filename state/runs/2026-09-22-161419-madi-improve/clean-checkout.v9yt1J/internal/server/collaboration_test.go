package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/reearth/ygo/crdt"
)

func TestBrowserCollaboration(t *testing.T) {
	if os.Getenv("MADI_BROWSER_COLLABORATION") != "1" {
		t.Skip("set MADI_BROWSER_COLLABORATION=1 after npm run build to verify native browser collaboration")
	}
	s, _, _, _, _ := collaborationTestSetup(t)
	app, e := New(context.Background(), s.DB, s.EncryptionKey, "test", "admin@example.test", "Integration-Test-Password-2026!", os.DirFS("../../web/dist"))
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(app)
	defer server.Close()
	defer app.CloseCollaboration()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "node", "../../web/src/collaboration/browser-test.mjs")
	command.Env = append(os.Environ(), "MADI_BASE_URL="+server.URL)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("browser collaboration: %v\n%s", e, output)
	}
	t.Log(string(output))
}

func collaborationTestDoc(text string) *crdt.Doc {
	doc := crdt.New()
	root := doc.GetXmlFragment("content")
	doc.Transact(func(tx *crdt.Transaction) {
		paragraph := crdt.NewYXmlElement("paragraph")
		paragraph.SetAttribute(tx, "id", "block-test-1")
		content := crdt.NewYXmlText()
		content.Insert(tx, 0, text, nil)
		paragraph.InsertText(tx, 0, content)
		root.InsertElement(tx, 0, paragraph)
	})
	return doc
}

func collaborationTestClone(t *testing.T, state []byte) *crdt.Doc {
	t.Helper()
	d := crdt.New()
	d.GetXmlFragment("content")
	if e := crdt.ApplyUpdateV1(d, state, nil); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(d.Destroy)
	return d
}

func collaborationTestInsert(d *crdt.Doc, text string) {
	paragraph := d.GetXmlFragment("content").Children()[0].(*crdt.YXmlElement)
	content := paragraph.Children()[0].(*crdt.YXmlText)
	d.Transact(func(tx *crdt.Transaction) { content.Insert(tx, content.Len(), text, nil) })
}

func TestCollaborationRealTiptapFixture(t *testing.T) {
	// Generated with actual @tiptap/y-tiptap 3.0.7, StarterKit 3.31.3 and
	// JS yjs 13.6.32: prosemirrorJSONToYDoc(schema,heading+marked paragraph,'content').
	fixture := "AQuMotLEAQAHAQdjb250ZW50AwdoZWFkaW5nBwCMotLEAQAGBACMotLEAQEN6rO164+ZIOusuOyEnCgAjKLSxAEABWxldmVsAX0Ch4yi0sQBAAMJcGFyYWdyYXBoBwCMotLEAQgGBACMotLEAQkH7JWI64WVIIaMotLEAQwEYm9sZAJ7fYSMotLEAQ0EbWFkaYaMotLEAREEYm9sZARudWxshIyi0sQBEhogW1vsmrTsmIEg6rCA7J2065OcXV0g8J+YgAA="
	state, e := base64.StdEncoding.DecodeString(fixture)
	if e != nil {
		t.Fatal(e)
	}
	doc := collaborationTestClone(t, state)
	markdown, _, e := collaborationMarkdown(doc.GetXmlFragment("content"))
	if e != nil {
		t.Fatal(e)
	}
	if markdown != "## 공동 문서\n\n안녕 **madi** [[운영 가이드]] 😀" {
		t.Fatalf("actual JS/TipTap fixture projection: %q", markdown)
	}
}

func TestCollaborationProjectionSafety(t *testing.T) {
	doc := collaborationTestDoc("한국어 😀 [[연결 문서]]")
	defer doc.Destroy()
	markdown, metadata, e := collaborationMarkdown(doc.GetXmlFragment("content"))
	if e != nil {
		t.Fatal(e)
	}
	if markdown != "한국어 😀 [[연결 문서]]" || !strings.Contains(string(jsonValue(metadata)), "block-test-1") {
		t.Fatalf("projection %q %v", markdown, metadata)
	}
	root := doc.GetXmlFragment("content")
	doc.Transact(func(tx *crdt.Transaction) { root.InsertElement(tx, 1, crdt.NewYXmlElement("unknownSensitiveBlock")) })
	if _, _, e = collaborationMarkdown(root); e == nil {
		t.Fatal("unknown block was silently discarded")
	}
	if _, e = collaborationURL("javascript:alert(1)"); e == nil {
		t.Fatal("unsafe mark accepted")
	}
	front := "---\r\ntags: [운영]\r\naliases:\r\n  - 예전\r\n---\r\n"
	if got, body := collaborationFrontMatter(front + "body"); got != front || body != "body" {
		t.Fatalf("front matter changed %q %q", got, body)
	}
	for _, value := range []string{"literal `code` *star*", "[[a]] and [[b|label]]", "# not heading"} {
		if _, e = collaborationRender(&collaborationNode{Type: "paragraph", Children: []*collaborationNode{{Type: "text", Text: value}}}); e != nil {
			t.Fatal(e)
		}
	}
}

func collaborationTestSetup(t *testing.T) (*Server, *integrationTestClient, *integrationTestClient, string, string) {
	t.Helper()
	s, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var workspaces []map[string]any
	if e := json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &workspaces); e != nil {
		t.Fatal(e)
	}
	wid := str(workspaces[0], "id")
	user := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "collaborator@example.test", "name": "공동 편집자", "password": "Collaboration-Password-2026!", "role": "editor"}, 0))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "collaborator@example.test", "role": "editor"}, 0)
	editor := newIntegrationTestClient(t, server.URL)
	editor.request("POST", "/api/v1/auth/login", map[string]any{"email": "collaborator@example.test", "password": "Collaboration-Password-2026!"}, 200)
	return s, admin, editor, wid, str(user, "id")
}

func collaborationTestSession(t *testing.T, client *integrationTestClient) string {
	t.Helper()
	u, _ := url.Parse(client.base)
	for _, cookie := range client.client.Jar.Cookies(u) {
		if cookie.Name == "madi_session" {
			return digest(cookie.Value)
		}
	}
	t.Fatal("missing login cookie")
	return ""
}

func TestPostgresCollaborationConcurrentDurabilityAndRESTEpoch(t *testing.T) {
	s, admin, editor, wid, _ := collaborationTestSetup(t)
	front := "---\r\ntags: [운영]\r\n---\r\n"
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "동시 편집", "markdown": front + "기본"}, 0))
	id := str(created, "id")
	ctx := context.Background()
	adminSession := collaborationTestSession(t, admin)
	editorSession := collaborationTestSession(t, editor)
	initial, e := s.collaborationState(ctx, id, adminSession, nil)
	if e != nil {
		t.Fatal(e)
	}
	seed := collaborationTestDoc("기본")
	defer seed.Destroy()
	seeded, e := s.collaborationState(ctx, id, adminSession, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: initial.Version, State: seed.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	alice, bob := collaborationTestClone(t, seeded.State), collaborationTestClone(t, seeded.State)
	collaborationTestInsert(alice, " Alice 😀")
	collaborationTestInsert(bob, " Bob 한글")
	var wg sync.WaitGroup
	errorsOut := make(chan error, 2)
	for _, item := range []struct {
		d       *crdt.Doc
		session string
	}{{alice, adminSession}, {bob, editorSession}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.collaborationState(ctx, id, item.session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: item.d.EncodeStateAsUpdate()})
			errorsOut <- err
		}()
	}
	wg.Wait()
	close(errorsOut)
	for e := range errorsOut {
		if e != nil {
			t.Fatal(e)
		}
	}
	actual := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+id, nil, 200))
	markdown := str(actual, "markdown")
	if !strings.HasPrefix(markdown, front) || !strings.Contains(markdown, "Alice 😀") || !strings.Contains(markdown, "Bob 한글") {
		t.Fatalf("REST did not expose all committed CRDT edits: %q", markdown)
	}
	rejoined, e := s.collaborationState(ctx, id, editorSession, nil)
	if e != nil {
		t.Fatal(e)
	}
	rehydrated := collaborationTestClone(t, rejoined.State)
	body, _, e := collaborationMarkdown(rehydrated.GetXmlFragment("content"))
	if e != nil || front+body != markdown {
		t.Fatalf("durable rejoin mismatch %q %v", body, e)
	}
	var count int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM document_versions WHERE document_id=$1", id).Scan(&count); e != nil || count < 3 {
		t.Fatalf("missing version snapshots: %d %v", count, e)
	}
	admin.request("PUT", "/api/v1/documents/"+id, map[string]any{"version": actual["version"], "markdown": "REST replacement"}, 200)
	stale, e := s.collaborationState(ctx, id, editorSession, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: bob.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	if stale.Type != "reset" || stale.Epoch == initial.Epoch || stale.Markdown != "REST replacement" {
		t.Fatalf("REST epoch barrier failed: %+v", stale)
	}
	if value := testJSONObject(t, admin.request("GET", "/api/v1/documents/"+id, nil, 200)); str(value, "markdown") != "REST replacement" {
		t.Fatal("stale editor overwrote REST")
	}
}

func TestPostgresCollaborationSeedRaceRollbackAndACL(t *testing.T) {
	s, admin, editor, wid, userID := collaborationTestSetup(t)
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "초기화 경합", "markdown": "original"}, 0))
	id := str(created, "id")
	ctx := context.Background()
	session := collaborationTestSession(t, admin)
	initial, e := s.collaborationState(ctx, id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	doc := collaborationTestDoc("original")
	defer doc.Destroy()
	message := &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: initial.Version, State: doc.EncodeStateAsUpdate()}
	first, e := s.collaborationState(ctx, id, session, message)
	if e != nil {
		t.Fatal(e)
	}
	second, e := s.collaborationState(ctx, id, session, message)
	if e != nil || second.Type != "reset" {
		t.Fatalf("seed race loser must reset: %+v %v", second, e)
	}
	if _, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: first.Epoch, State: []byte{255, 255, 255}}); e == nil {
		t.Fatal("malformed binary accepted")
	}
	after, e := s.collaborationState(ctx, id, session, nil)
	if e != nil || after.Sequence != first.Sequence {
		t.Fatalf("rejected binary mutated state: %+v %v", after, e)
	}
	// Force failure after document UPDATE: the whole CRDT/Markdown/version commit
	// must roll back, and the server must never ACK it as saved.
	_, e = s.DB.Exec(ctx, `CREATE FUNCTION reject_collaboration_version() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'rollback test'; END $$; CREATE TRIGGER reject_collaboration_version BEFORE INSERT ON document_versions FOR EACH ROW EXECUTE FUNCTION reject_collaboration_version()`)
	if e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(doc, " must roll back")
	if _, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: first.Epoch, State: doc.EncodeStateAsUpdate()}); e == nil {
		t.Fatal("forced version failure did not fail update")
	}
	after, e = s.collaborationState(ctx, id, session, nil)
	if e != nil || after.Sequence != first.Sequence || after.Markdown != "original" {
		t.Fatalf("rollback changed state: %+v %v", after, e)
	}
	_, _ = s.DB.Exec(ctx, "DROP TRIGGER reject_collaboration_version ON document_versions")
	_, e = s.DB.Exec(ctx, "UPDATE documents SET visibility='private' WHERE id=$1", id)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.collaborationState(ctx, id, collaborationTestSession(t, editor), nil); e == nil {
		t.Fatal("private doc leaked to workspace peer")
	}
	_, _ = s.DB.Exec(ctx, "UPDATE documents SET visibility='selected' WHERE id=$1", id)
	_, _ = s.DB.Exec(ctx, "INSERT INTO document_shares(document_id,user_id,permission) VALUES($1,$2,'read')", id, userID)
	readOnly, e := s.collaborationState(ctx, id, collaborationTestSession(t, editor), nil)
	if e != nil || readOnly.CanWrite {
		t.Fatalf("read-only shared doc: %+v %v", readOnly, e)
	}
	if _, e = s.collaborationState(ctx, id, collaborationTestSession(t, editor), &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: first.Epoch, State: doc.EncodeStateAsUpdate()}); e == nil {
		t.Fatal("read-only peer wrote")
	}
}

func collaborationTestDial(t *testing.T, client *integrationTestClient, id, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	u, _ := url.Parse(client.base)
	headers := http.Header{}
	headers.Set("Origin", origin)
	for _, cookie := range client.client.Jar.Cookies(u) {
		headers.Add("Cookie", cookie.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, response, e := websocket.Dial(ctx, "ws"+strings.TrimPrefix(client.base, "http")+"/api/v1/documents/"+id+"/collaboration", &websocket.DialOptions{HTTPHeader: headers})
	if e == nil {
		t.Cleanup(func() { c.CloseNow() })
		c.SetReadLimit(24 << 20)
	}
	return c, response, e
}

func collaborationTestRead(t *testing.T, c *websocket.Conn, kind string) collaborationMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		var m collaborationMessage
		if e := wsjson.Read(ctx, c, &m); e != nil {
			t.Fatal(e)
		}
		if m.Type == kind {
			return m
		}
	}
}

func TestPostgresCollaborationWebSocketOriginPresenceRevocation(t *testing.T) {
	s, admin, editor, wid, userID := collaborationTestSetup(t)
	created := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "웹소켓", "markdown": "초안"}, 0))
	id := str(created, "id")
	if c, response, e := collaborationTestDial(t, editor, id, "https://evil.example"); e == nil {
		c.CloseNow()
		t.Fatal("cross-origin socket accepted")
	} else if response == nil || response.StatusCode != 403 {
		t.Fatalf("origin status: %v %v", response, e)
	}
	a, _, e := collaborationTestDial(t, admin, id, admin.base)
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := collaborationTestDial(t, editor, id, editor.base)
	if e != nil {
		t.Fatal(e)
	}
	aHello, bHello := collaborationTestRead(t, a, "hello"), collaborationTestRead(t, b, "hello")
	if aHello.Epoch != bHello.Epoch {
		t.Fatal("same document split into rooms")
	}
	seed := collaborationTestDoc("초안")
	defer seed.Destroy()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e = wsjson.Write(ctx, a, collaborationMessage{Type: "seed", ID: 1, Schema: collaborationSchemaID, Epoch: aHello.Epoch, Version: aHello.Version, State: seed.EncodeStateAsUpdate()}); e != nil {
		t.Fatal(e)
	}
	ack := collaborationTestRead(t, a, "ack")
	sync := collaborationTestRead(t, b, "sync")
	if ack.Sequence != sync.Sequence || sync.State == nil {
		t.Fatal("committed update not broadcast")
	}
	if e = wsjson.Write(ctx, b, collaborationMessage{Type: "awareness", ClientID: 123, Clock: 1, Awareness: json.RawMessage(`{"user":{"name":"spoofed admin"},"cursor":null,"html":"unsafe"}`)}); e != nil {
		t.Fatal(e)
	}
	for {
		presence := collaborationTestRead(t, a, "presence")
		if len(presence.Presence) == 0 {
			continue
		}
		raw := string(jsonValue(presence.Presence))
		if strings.Contains(raw, "spoofed") || strings.Contains(raw, "unsafe") || !strings.Contains(raw, "공동 편집자") {
			t.Fatalf("unverified awareness %s", raw)
		}
		break
	}
	// A quiet, revoked subscriber must close without seeing the next document body.
	if _, e = s.DB.Exec(ctx, "UPDATE users SET disabled=true WHERE id=$1", userID); e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(seed, " private after revocation")
	if e = wsjson.Write(ctx, a, collaborationMessage{Type: "update", ID: 2, Schema: collaborationSchemaID, Epoch: ack.Epoch, State: seed.EncodeStateAsUpdate()}); e != nil {
		t.Fatal(e)
	}
	_ = collaborationTestRead(t, a, "ack")
	for {
		var m collaborationMessage
		e = wsjson.Read(ctx, b, &m)
		if e != nil {
			if websocket.CloseStatus(e) != websocket.StatusPolicyViolation && !errors.Is(e, context.Canceled) {
				t.Fatalf("unexpected revocation close: %v", e)
			}
			break
		}
		if strings.Contains(m.Markdown, "private after revocation") {
			t.Fatal("revoked idle subscriber received new document body")
		}
	}
}
