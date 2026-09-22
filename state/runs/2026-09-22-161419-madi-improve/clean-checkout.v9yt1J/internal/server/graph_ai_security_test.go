package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestPostgresGraphAIInFlightRevocation(t *testing.T) {
	for _, mode := range []string{"provider", "feature", "source_version", "logout"} {
		t.Run(mode, func(t *testing.T) {
			s, c, wid, _ := graphAITestSetup(t)
			doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 기준", "markdown": "PostgreSQL 운영 기준"}, 200))
			did := str(doc, "id")
			release := make(chan struct{})
			var once sync.Once
			defer once.Do(func() { close(release) })
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprintf(w, "data: %s\n\n", jsonValue(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": `{"candidates":[`}}}}))
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				candidate := graphAITestCandidates(did, newID())[2]
				agentTestSSE(w, string(jsonValue(candidate))+`]}`, "", "")
			}))
			defer provider.Close()
			c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "graph-guard", "ai_max_tokens": 4096}, 200)
			preview := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+did, nil, 200))
			id := newID()
			req, e := http.NewRequest("POST", c.base+"/api/v1/workspaces/"+wid+"/graph-ai/analyze", bytes.NewReader(jsonValue(map[string]any{"request_id": id, "snapshots": preview["snapshots"], "provider_fingerprint": preview["provider"].(map[string]any)["fingerprint"], "kinds": []string{"topic"}, "consent": true})))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("X-Madi-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			response, e := c.client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				raw, _ := io.ReadAll(response.Body)
				t.Fatalf("stream start %d %s", response.StatusCode, raw)
			}
			reader := bufio.NewReader(response.Body)
			for {
				line, e := reader.ReadString('\n')
				if e != nil {
					t.Fatal(e)
				}
				if strings.Contains(line, `"text"`) {
					break
				}
			}
			switch mode {
			case "provider":
				c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "changed-provider-model"}, 200)
			case "feature":
				c.request("PUT", "/api/v1/admin/settings", map[string]any{"feature_flags": map[string]any{"ai-graph": false}}, 200)
			case "source_version":
				c.request("PUT", "/api/v1/documents/"+did, map[string]any{"version": 1, "markdown": "새 원문"}, 200)
			case "logout":
				c.request("POST", "/api/v1/auth/logout", nil, 200)
			}
			once.Do(func() { close(release) })
			tail, e := io.ReadAll(reader)
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(string(tail), `"retract":true`) || strings.Contains(string(tail), `"proposal":`) {
				t.Fatalf("revoked stream accepted %s", tail)
			}
			var count int
			s.DB.QueryRow(t.Context(), `SELECT count(*) FROM graph_ai_actions WHERE run_id=$1`, id).Scan(&count)
			if count != 0 {
				t.Fatal("revoked proposal persisted")
			}
		})
	}
}

func TestPostgresGraphAIConcurrentConfirmationAndPrivacy(t *testing.T) {
	s, c, wid, uid := graphAITestSetup(t)
	doc := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "운영 기준", "markdown": "PostgreSQL 운영 기준", "visibility": "private"}, 200))
	did := str(doc, "id")
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO knowledge_document_meta(document_id,classification) VALUES($1,'confidential')`, did); e != nil {
		t.Fatal(e)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		candidate := graphAICandidate{Kind: "gap", Title: "운영 보완 초안", Description: "확인할 내용과 검증 순서를 작성합니다.", Reason: "제공 구간의 보완 질문입니다.", Evidence: []graphAIEvidence{{DocumentID: did, Quote: "PostgreSQL 운영 기준"}}}
		agentTestSSE(w, string(jsonValue(map[string]any{"candidates": []graphAICandidate{candidate}})), "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "graph-confirm", "ai_max_tokens": 4096}, 200)
	id := graphAITestGenerate(t, c, wid, []string{did}, []string{"gap"})
	view := testJSONObject(t, c.request("GET", "/api/v1/graph-ai/runs/"+id, nil, 200))
	action := view["actions"].([]any)[0].(map[string]any)
	issued := testJSONObject(t, c.request("POST", "/api/v1/keys", map[string]any{"name": "그래프 검증 키", "workspace_id": wid, "scopes": []string{"ai:execute", "document:read", "document:write"}, "expires_in_days": 1, "rate_limit": 300}, 201))
	keyClient := newIntegrationTestClient(t, c.base)
	keyClient.token = str(issued, "token")
	path := "/api/v1/graph-ai/actions/" + str(action, "id") + "/confirm"
	keyClient.request("POST", path, map[string]any{"action_hash": action["action_hash"], "confirm": true}, 403)
	oldPool := s.DB
	limitedConfig := oldPool.Config()
	limitedConfig.MaxConns = 4
	limitedPool, e := pgxpool.NewWithConfig(t.Context(), limitedConfig)
	if e != nil {
		t.Fatal(e)
	}
	s.DB = limitedPool
	t.Cleanup(func() { s.DB = oldPool; limitedPool.Close() })
	const writers = 6
	results := make(chan string, writers)
	failures := make(chan error, writers)
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, e := http.NewRequest("POST", c.base+path, bytes.NewReader(jsonValue(map[string]any{"action_hash": action["action_hash"], "confirm": true})))
			if e != nil {
				failures <- e
				return
			}
			req.Header.Set("X-Madi-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			response, e := c.client.Do(req)
			if e != nil {
				failures <- e
				return
			}
			defer response.Body.Close()
			raw, _ := io.ReadAll(response.Body)
			if response.StatusCode != 200 {
				failures <- fmt.Errorf("confirmation %d %s", response.StatusCode, raw)
				return
			}
			var result map[string]any
			if json.Unmarshal(raw, &result) != nil {
				failures <- fmt.Errorf("invalid result %s", raw)
				return
			}
			results <- str(result["result"].(map[string]any), "document_id")
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for e := range failures {
		t.Error(e)
	}
	unique := map[string]bool{}
	for id := range results {
		unique[id] = true
	}
	if len(unique) != 1 {
		t.Fatalf("same action created %d result IDs", len(unique))
	}
	var count int
	s.DB.QueryRow(t.Context(), `SELECT count(*) FROM documents WHERE workspace_id=$1`, wid).Scan(&count)
	if count != 2 {
		t.Fatalf("duplicate writes %d", count)
	}
	for id := range unique {
		var visibility, classification, owner string
		e := s.DB.QueryRow(t.Context(), `SELECT d.visibility,m.classification,d.owner_id::text FROM documents d JOIN knowledge_document_meta m ON m.document_id=d.id WHERE d.id=$1`, id).Scan(&visibility, &classification, &owner)
		if e != nil || visibility != "private" || classification != "confidential" || owner != uid {
			t.Fatalf("result privacy %s %s %s %v", visibility, classification, owner, e)
		}
	}
}

func TestPostgresGraphAITopicSingleSourceAndPrivateACL(t *testing.T) {
	s, c, wid, uid := graphAITestSetup(t)
	user := testJSONObject(t, c.request("POST", "/api/v1/admin/users", map[string]any{"email": "graph-viewer@example.test", "name": "조회자", "password": "Graph-Viewer-Password!", "role": "viewer"}, 200))
	c.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "graph-viewer@example.test", "role": "viewer"}, 200)
	viewer := newIntegrationTestClient(t, c.base)
	viewer.request("POST", "/api/v1/auth/login", map[string]any{"email": "graph-viewer@example.test", "password": "Graph-Viewer-Password!"}, 200)
	shared := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "공유 기준", "markdown": "PostgreSQL 운영 기준"}, 200))
	private := testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "개인 운영 기밀", "markdown": "PRIVATE_GRAPH_SENTINEL PostgreSQL 장애 복구", "visibility": "private"}, 200))
	aid, bid := str(shared, "id"), str(private, "id")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		candidate := graphAITestCandidates(aid, bid)[2]
		agentTestSSE(w, string(jsonValue(map[string]any{"candidates": []graphAICandidate{candidate}})), "", "")
	}))
	defer provider.Close()
	c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_enabled": true, "ai_base_url": provider.URL, "ai_model": "graph-private"}, 200)
	viewer.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+bid, nil, 403)
	preview := testJSONObject(t, c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+aid+","+bid, nil, 200))
	c.request("POST", "/api/v1/workspaces/"+wid+"/graph-ai/analyze", map[string]any{"request_id": newID(), "snapshots": preview["snapshots"], "provider_fingerprint": preview["provider"].(map[string]any)["fingerprint"], "kinds": []string{"topic"}, "consent": true}, 400)
	id := graphAITestGenerate(t, c, wid, []string{aid}, []string{"topic"})
	view := testJSONObject(t, c.request("GET", "/api/v1/graph-ai/runs/"+id, nil, 200))
	action := view["actions"].([]any)[0].(map[string]any)
	viewer.request("GET", "/api/v1/graph-ai/runs/"+id, nil, 404)
	// Simulate a pre-hardening or persisted multi-source candidate: confirmation
	// must independently reject it, rather than relying only on the analyzer.
	if _, e := s.DB.Exec(t.Context(), `INSERT INTO graph_ai_sources(run_id,document_id,document_version,document_hash,start_byte,end_byte,content_hash) VALUES($1,$2,1,$3,0,$4,$5)`, id, bid, graphAICanonicalHash(str(private, "title"), str(private, "markdown")), len(str(private, "markdown")), digest(str(private, "markdown"))); e != nil {
		t.Fatal(e)
	}
	c.request("POST", "/api/v1/graph-ai/actions/"+str(action, "id")+"/confirm", map[string]any{"action_hash": action["action_hash"], "confirm": true}, 409)
	metadata := viewer.request("GET", "/api/v1/documents/"+aid+"/knowledge", nil, 200)
	if strings.Contains(string(metadata), "ai_topics") || strings.Contains(string(metadata), "PRIVATE_GRAPH_SENTINEL") {
		t.Fatal("multi-source inferred metadata leaked to viewer")
	}
	// Service admin status never bypasses another person's private source.
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET owner_id=$2 WHERE id=$1`, bid, str(user, "id")); e != nil {
		t.Fatal(e)
	}
	c.request("GET", "/api/v1/workspaces/"+wid+"/graph-ai/context?document_ids="+bid, nil, 403)
	if _, e := s.DB.Exec(t.Context(), `UPDATE documents SET owner_id=$2 WHERE id=$1`, bid, uid); e != nil {
		t.Fatal(e)
	}
}
