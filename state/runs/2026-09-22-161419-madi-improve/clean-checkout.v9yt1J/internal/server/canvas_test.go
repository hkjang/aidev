package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func canvasTestNode(kind string) canvasNode {
	return canvasNode{ID: newID(), Kind: kind, X: 100, Y: 120, Width: 300, Height: 200, Color: "mint", Text: "아이디어"}
}
func TestCanvasDataBoundsAndURLs(t *testing.T) {
	valid := canvasData{Nodes: []canvasNode{canvasTestNode("note")}}
	if e := validateCanvasData(&valid); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*canvasData){func(d *canvasData) { d.Nodes[0].X = math.Inf(1) }, func(d *canvasData) { d.Nodes[0].Width = -5 }, func(d *canvasData) { d.Nodes[0].Kind = "script" }, func(d *canvasData) { d.Nodes[0].Color = "url(http://bad)" }, func(d *canvasData) { d.Nodes = append(d.Nodes, d.Nodes[0]) }, func(d *canvasData) {
		d.Edges = []canvasEdge{{ID: newID(), Source: d.Nodes[0].ID, Target: newID(), Color: "mint"}}
	}, func(d *canvasData) { d.Nodes[0].Kind = "drawing"; d.Nodes[0].Points = make([][2]float64, 2001) }} {
		data := canvasData{Nodes: []canvasNode{canvasTestNode("note")}}
		mutate(&data)
		if validateCanvasData(&data) == nil {
			t.Fatal("invalid canvas accepted")
		}
	}
	for _, address := range []string{"javascript:alert(1)", "data:text/html,hi", "file:///etc/passwd", "https://user:password@example.com", "https:///missing"} {
		data := canvasData{Nodes: []canvasNode{canvasTestNode("url")}}
		data.Nodes[0].URL = address
		if validateCanvasData(&data) == nil {
			t.Fatalf("unsafe URL accepted: %s", address)
		}
	}
	data := canvasData{Nodes: []canvasNode{canvasTestNode("url")}}
	data.Nodes[0].URL = "https://intranet.example/path"
	if e := validateCanvasData(&data); e != nil {
		t.Fatal(e)
	}
}

func TestPostgresCanvasACLVersionReferencesAndTrash(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	editorUser := testJSONObject(t, admin.request("POST", "/api/v1/admin/users", map[string]any{"email": "canvas-editor@example.test", "name": "캔버스 편집자", "role": "editor", "password": "Canvas-editor-password-2026!"}, 200))
	admin.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "canvas-editor@example.test", "role": "editor"}, 200)
	editor := newIntegrationTestClient(t, server.URL)
	editor.request("POST", "/api/v1/auth/login", map[string]any{"email": "canvas-editor@example.test", "password": "Canvas-editor-password-2026!"}, 200)
	privateDoc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "CANVAS_PRIVATE_TITLE", "markdown": "CANVAS_PRIVATE_SENTINEL_945", "visibility": "private"}, 200))
	reference := canvasTestNode("document")
	reference.Text = ""
	reference.RefID = str(privateDoc, "id")
	note := canvasTestNode("note")
	data := canvasData{Nodes: []canvasNode{note, reference}, Edges: []canvasEdge{{ID: newID(), Source: note.ID, Target: reference.ID, Color: "mint", Label: "관련 문서"}}}
	c := testJSONObject(t, admin.request("POST", "/api/v1/canvases", map[string]any{"workspace_id": wid, "title": "개인 설계", "data": data}, 200))
	id := str(c, "id")
	editor.request("GET", "/api/v1/canvases/"+id, nil, 404)
	c = testJSONObject(t, admin.request("PUT", "/api/v1/canvases/"+id, map[string]any{"title": "선택 공유", "visibility": "selected", "data": data, "version": c["version"]}, 200))
	admin.request("PUT", "/api/v1/canvases/"+id+"/shares", map[string]any{"email": "canvas-editor@example.test", "permission": "read"}, 200)
	visible := testJSONObject(t, editor.request("GET", "/api/v1/canvases/"+id, nil, 200))
	serialized := string(jsonValue(visible))
	if boolean(visible, "can_write") || strings.Contains(serialized, "CANVAS_PRIVATE_SENTINEL") || strings.Contains(serialized, "CANVAS_PRIVATE_TITLE") {
		t.Fatalf("shared canvas leaked private source or granted write: %s", serialized)
	}
	editor.request("PUT", "/api/v1/canvases/"+id, map[string]any{"title": "변경", "data": data, "version": c["version"]}, 403)
	admin.request("PUT", "/api/v1/canvases/"+id+"/shares", map[string]any{"email": "canvas-editor@example.test", "permission": "write"}, 200)
	editor.request("PUT", "/api/v1/canvases/"+id, map[string]any{"title": "공개 전환 공격", "visibility": "workspace", "data": data, "version": c["version"]}, 403)
	newReference := reference
	newReference.ID = newID()
	invalid := data
	invalid.Nodes = append(append([]canvasNode{}, data.Nodes...), newReference)
	editor.request("POST", "/api/v1/canvases/"+id+"/validate", invalid, 400)
	changed := testJSONObject(t, editor.request("PUT", "/api/v1/canvases/"+id, map[string]any{"title": "협업한 설계", "data": data, "version": c["version"]}, 200))
	editor.request("PUT", "/api/v1/canvases/"+id, map[string]any{"title": "오래된 덮어쓰기", "data": data, "version": c["version"]}, 409)
	if changed["version"] == c["version"] {
		t.Fatal("version did not increment")
	}
	personal := testJSONObject(t, editor.request("POST", "/api/v1/canvases", map[string]any{"workspace_id": wid, "title": "편집자 개인 캔버스", "data": canvasData{}}, 200))
	admin.request("GET", "/api/v1/canvases/"+str(personal, "id"), nil, 404)
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"user_id": editorUser["id"], "workspace_id": wid, "name": "캔버스 조회 키", "scopes": []string{"document:read"}}, 201))
	key := newIntegrationTestClient(t, server.URL)
	key.token = str(issued, "token")
	keyView := testJSONObject(t, key.request("GET", "/api/v1/canvases/"+id, nil, 200))
	if boolean(keyView, "can_write") || boolean(keyView, "can_manage") {
		t.Fatal("read-only key has writable UI capabilities")
	}
	key.request("PUT", "/api/v1/canvases/"+id, map[string]any{}, 403)
	editor.request("DELETE", "/api/v1/canvases/"+id, nil, 200)
	trashed := testJSONObject(t, editor.request("GET", "/api/v1/canvases/"+id, nil, 200))
	if boolean(trashed, "can_write") || trashed["deleted_at"] == nil {
		t.Fatal("trash state missing")
	}
	editor.request("POST", "/api/v1/canvases/"+id+"/restore", map[string]any{}, 200)
	admin.request("PUT", "/api/v1/canvases/"+id+"/shares", map[string]any{"email": "canvas-editor@example.test", "permission": "remove"}, 200)
	editor.request("GET", "/api/v1/canvases/"+id, nil, 404)
}

func TestPostgresCanvasAIStreamingAndScope(t *testing.T) {
	_, server := integrationTestServer(t)
	admin := newIntegrationTestClient(t, server.URL)
	admin.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	var ws []map[string]any
	json.Unmarshal(admin.request("GET", "/api/v1/workspaces", nil, 200), &ws)
	wid := str(ws[0], "id")
	node := canvasTestNode("ai")
	node.Text = "개념 연결을 제안해주세요"
	canvas := testJSONObject(t, admin.request("POST", "/api/v1/canvases", map[string]any{"workspace_id": wid, "title": "AI 캔버스", "data": canvasData{Nodes: []canvasNode{node}}}, 200))
	id := str(canvas, "id")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["stream"] != true {
			t.Error("non-streamed AI request")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"아이디어를 문서에 연결하세요.\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer provider.Close()
	admin.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL + "/v1", "ai_model": "test"}, 200)
	body := string(admin.request("POST", "/api/v1/canvases/"+id+"/ai/"+node.ID, map[string]any{}, 200))
	if !strings.Contains(body, "아이디어를 문서에") || !strings.Contains(body, "[DONE]") {
		t.Fatal(body)
	}
	issued := testJSONObject(t, admin.request("POST", "/api/v1/keys", map[string]any{"workspace_id": wid, "name": "캔버스 편집 전용", "scopes": []string{"document:read", "document:write"}}, 201))
	key := newIntegrationTestClient(t, server.URL)
	key.token = str(issued, "token")
	key.request("POST", "/api/v1/canvases/"+id+"/ai/"+node.ID, map[string]any{}, 403)
}
