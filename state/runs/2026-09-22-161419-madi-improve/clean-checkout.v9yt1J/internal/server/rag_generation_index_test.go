package server

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"os"
	"path/filepath"
	"testing"
)

func TestPostgresRAGGenerationRealHNSW(t *testing.T) {
	s, c, wid, uid := ragTestSetup(t)
	var installed bool
	if e := s.DB.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector')`).Scan(&installed); e != nil {
		t.Fatal(e)
	}
	provider, _, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	generation := str(testJSONObject(t, c.request("POST", "/api/v1/workspaces/"+wid+"/rag-generations", map[string]any{"name": "실제 HNSW 검증", "dimensions": 3, "config": map[string]any{"rag_embedding_model": "ann-generation"}, "confirm": true}, 201)), "id")
	if !installed {
		c.request("POST", "/api/v1/rag-generations/"+generation+"/index", map[string]any{"revision": 1, "confirm": true}, 409)
		t.Log("extension absent: explicit failure, no automatic installation")
		return
	}
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "ANN GPU", "markdown": "GPU 검증", "visibility": "private"}, 200)), "id")
	path := "/api/v1/documents/" + id + "/rag-index?generation_id=" + generation
	state := testJSONObject(t, c.request("GET", path, nil, 200))
	request := testJSONObject(t, c.request("POST", path, map[string]any{"expected_version": 1, "provider_fingerprint": state["provider"].(map[string]any)["fingerprint"], "consent": true}, 202))
	ragTestRun(t, s, str(request, "job_id"))
	index := testJSONObject(t, c.request("POST", "/api/v1/rag-generations/"+generation+"/index", map[string]any{"revision": 1, "confirm": true}, 202))
	ragTestRun(t, s, str(index, "job_id"))
	if status := ragTestStatus(t, s, str(index, "job_id")); status != "succeeded" {
		var message string
		s.DB.QueryRow(t.Context(), `SELECT last_error FROM automation_jobs WHERE id=$1`, str(index, "job_id")).Scan(&message)
		t.Fatal(status, message)
	}
	cfg, e := s.effectiveSettings(t.Context(), wid)
	if e != nil {
		t.Fatal(e)
	}
	cfg, e = s.ragGenerationConfig(t.Context(), s.DB, cfg, wid, generation)
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.workerPrincipal(t.Context(), uid, "", wid)
	if e != nil {
		t.Fatal(e)
	}
	candidates, diag, e := s.ragGenerationCandidates(t.Context(), p, wid, id, cfg, []float32{1, 0, 0}, generation, "verify", []string{id})
	if e != nil || len(candidates) != 1 || !diag.PlannedIndexUsed || diag.RecallAtK == nil || *diag.RecallAtK != 1 {
		t.Fatal(e, len(candidates), diag)
	}
	t.Logf("real pgvector HNSW plan=%s recall=%g candidates=%d", diag.IndexName, *diag.RecallAtK, len(candidates))
	// A generation index is not an ACL capability.
	viewer := newID()
	if _, e = s.DB.Exec(t.Context(), `INSERT INTO users(id,email,name,password_hash,role) VALUES($1,'ann-viewer@example.test','ANN 조회자','not-a-password','viewer')`, viewer); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(t.Context(), `INSERT INTO workspace_members(workspace_id,user_id,role) VALUES($1,$2,'viewer')`, wid, viewer); e != nil {
		t.Fatal(e)
	}
	denied, _, e := s.ragGenerationCandidates(t.Context(), &Principal{ID: viewer, Role: "viewer", Kind: "user"}, wid, id, cfg, []float32{1, 0, 0}, generation, "ann", []string{id})
	if e != nil || len(denied) != 0 {
		t.Fatal("ANN private bypass", e, len(denied))
	}
	// Deterministic synthetic vectors separate the ANN/filter algorithm check
	// from embedding-provider quality. One quarter of the 128 extra documents
	// are private, including near neighbors that must be skipped by iteration.
	tx, e := s.DB.Begin(t.Context())
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(t.Context())
	for n := 0; n < 128; n++ {
		docID, jobID, grantID := newID(), newID(), newID()
		markdown := fmt.Sprintf("GPU synthetic vector %d", n)
		visibility := "workspace"
		if n%4 == 0 {
			visibility = "private"
		}
		_, e = tx.Exec(t.Context(), `WITH d AS (INSERT INTO documents(id,workspace_id,title,markdown,owner_id,visibility) VALUES($1,$2,$3,$4,$5,$6) RETURNING id),j AS (INSERT INTO automation_jobs(id,kind,workspace_id,owner_id,actor_id,status) VALUES($7,'rag.index',$2,$5,$5,'succeeded') RETURNING id),g AS (INSERT INTO rag_index_grants(id,document_id,workspace_id,actor_id,provider_fingerprint,expected_version,last_job_id,generation_id) SELECT $8,d.id,$2,$5,$9,1,j.id,$10 FROM d,j RETURNING id,document_id,last_job_id),i AS (INSERT INTO rag_vector_indexes(id,grant_id,document_id,document_version,grant_revision,provider_fingerprint,status,total_chunks,indexed_chunks,dimensions) SELECT last_job_id,id,document_id,1,1,$9,'ready',1,1,3 FROM g RETURNING id) INSERT INTO rag_vector_chunks(index_id,ordinal,content_hash,start_byte,end_byte,start_line,end_line,heading,embedding,generation_id) SELECT id,0,$11,0,$12,1,1,'',$13,$10 FROM i`, docID, wid, fmt.Sprintf("합성 ANN %03d", n), markdown, uid, visibility, jobID, grantID, ragProviderFingerprint(cfg), generation, digest(markdown), len(markdown), []float32{1, float32(n+1) / 100, 0})
		if e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(t.Context()); e != nil {
		t.Fatal(e)
	}
	cfg["rag_candidates"] = 20
	reports := []map[string]any{}
	for _, actor := range []*Principal{p, {ID: viewer, Role: "viewer", Kind: "user"}} {
		result, check, e := s.ragGenerationCandidates(t.Context(), actor, wid, "", cfg, []float32{1, 0, 0}, generation, "verify", nil)
		if e != nil || len(result) != 20 || check.RecallAtK == nil || *check.RecallAtK != 1 || check.Truncated {
			t.Fatal("filtered ANN recall", e, len(result), check)
		}
		for _, item := range result {
			if !s.canDocument(t.Context(), actor, item.DocumentID, false) {
				t.Fatal("ANN unauthorized neighbor", item.DocumentID)
			}
		}
		t.Logf("129-doc synthetic ACL cohort actor=%s Recall@20=%g plannedHNSW=%v", actor.Role, *check.RecallAtK, check.PlannedIndexUsed)
		namespace, extension, e := s.ragVectorExtension(t.Context())
		if e != nil {
			t.Fatal(e)
		}
		read, e := s.DB.BeginTx(t.Context(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = read.Exec(t.Context(), `SELECT set_config('hnsw.ef_search','100',true),set_config('hnsw.iterative_scan','strict_order',true),set_config('hnsw.max_scan_tuples','5000',true),set_config('enable_seqscan','off',true),set_config('enable_sort','off',true)`); e != nil {
			read.Rollback(t.Context())
			t.Fatal(e)
		}
		var plan json.RawMessage
		e = read.QueryRow(t.Context(), "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+ragANNQuery(namespace, generation, 3), pgx.QueryExecModeExec, ragProviderFingerprint(cfg), wid, "", actor.ID, 3, 5001, generation, []string(nil), []float32{1, 0, 0}, 20).Scan(&plan)
		read.Rollback(t.Context())
		if e != nil {
			t.Fatal(e)
		}
		var actualPlan any
		if e = json.Unmarshal(plan, &actualPlan); e != nil || !ragPlanUsesIndex(actualPlan, ragHNSWIndexName(generation)) {
			t.Fatal("executed plan not HNSW", e)
		}
		reports = append(reports, map[string]any{"actor": actor.Role, "recall_at_20": *check.RecallAtK, "pgvector": extension, "executed_plan": plan})
	}
	var postgres string
	if e = s.DB.QueryRow(t.Context(), `SHOW server_version`).Scan(&postgres); e != nil {
		t.Fatal(e)
	}
	output := filepath.Join("..", "..", "test-results", "rag-generations")
	if e = os.MkdirAll(output, 0755); e != nil {
		t.Fatal(e)
	}
	raw, e := json.MarshalIndent(map[string]any{"postgres": postgres, "documents": 129, "dimensions": 3, "fixture": "1 real-provider private document plus 128 deterministic synthetic vectors, every fourth extra document private; not production embedding quality or latency benchmark", "reports": reports}, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(output, "ann-acl-129.json"), raw, 0644); e != nil {
		t.Fatal(e)
	}
}
