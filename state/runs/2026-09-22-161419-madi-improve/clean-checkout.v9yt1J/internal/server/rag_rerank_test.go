package server

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRAGRerankPermutationAndFiniteScores(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"valid", `{"results":[{"index":1,"relevance_score":0.9},{"index":0,"relevance_score":0.2}]}`, true},
		{"duplicate", `{"results":[{"index":0,"relevance_score":0.9},{"index":0,"relevance_score":0.2}]}`, false},
		{"missing", `{"results":[{"index":0,"relevance_score":0.9}]}`, false},
		{"missing-score", `{"results":[{"index":0},{"index":1,"relevance_score":0.2}]}`, false},
		{"missing-index", `{"results":[{"relevance_score":0.9},{"index":1,"relevance_score":0.2}]}`, false},
		{"range", `{"results":[{"index":2,"relevance_score":0.9},{"index":0,"relevance_score":0.2}]}`, false},
		{"nonfinite", `{"results":[{"index":1,"relevance_score":1e1000},{"index":0,"relevance_score":0.2}]}`, false},
		{"unordered", `{"results":[{"index":1,"relevance_score":0.1},{"index":0,"relevance_score":0.9}]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.body))
			}))
			defer provider.Close()
			order, e := ragRerank(context.Background(), ragProvider{BaseURL: provider.URL + "/v1", Model: "local", AllowHTTP: true}, "question", []string{"a", "b"})
			if (e == nil) != tc.valid {
				t.Fatal(order, e)
			}
			if tc.valid && (order[0] != 1 || order[1] != 0) {
				t.Fatal(order)
			}
		})
	}
}

func TestRAGCosineBoundedFiniteAndFusion(t *testing.T) {
	if value, ok := ragCosine([]float32{1, 2}, []float32{1, 2}); !ok || math.Abs(value-1) > 1e-7 {
		t.Fatal(value, ok)
	}
	for _, vector := range [][]float32{{0, 0}, {float32(math.NaN()), 1}, {float32(math.Inf(1)), 1}, {1}} {
		if _, ok := ragCosine([]float32{1, 2}, vector); ok {
			t.Fatal("invalid cosine accepted", vector)
		}
	}
	a, b, c := aiSource{CitationID: "a"}, aiSource{CitationID: "b"}, aiSource{CitationID: "c"}
	result := ragFuse([]aiSource{a, b}, []aiSource{b, c}, 3)
	if len(result) != 3 || result[0].CitationID != "b" {
		t.Fatal(result)
	}
	if strings.Contains(ragDisplayURL("https://provider.test/v1?api-key=private#secret"), "private") {
		t.Fatal("display URL secret")
	}
}

func TestPostgresRAGVectorBoundedScanAndOptionalPGVector(t *testing.T) {
	s, c, wid, uid := ragTestSetup(t)
	provider, _, _ := ragTestProvider(t, nil)
	ragTestConfigure(t, c, provider.URL)
	id := str(testJSONObject(t, c.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "벡터 한도", "markdown": "# GPU 검증"}, 200)), "id")
	consent := ragTestConsent(t, c, id, 1, false, false)
	jobID := str(consent, "job_id")
	ragTestRun(t, s, jobID)
	_, e := s.DB.Exec(t.Context(), `INSERT INTO rag_vector_chunks(index_id,ordinal,content_hash,start_byte,end_byte,start_line,end_line,heading,embedding) SELECT index_id,n,content_hash,start_byte,end_byte,start_line,end_line,heading,ARRAY[1::real,n::real/1000,0::real] FROM rag_vector_chunks CROSS JOIN generate_series(1,120) n WHERE index_id=$1 AND ordinal=0`, jobID)
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.workerPrincipal(t.Context(), uid, "", wid)
	if e != nil {
		t.Fatal(e)
	}
	cfg, e := s.effectiveSettings(t.Context(), wid)
	if e != nil {
		t.Fatal(e)
	}
	cfg["rag_scan_limit"] = 100
	cfg["rag_candidates"] = 20
	native, diag, e := s.ragVectorCandidates(t.Context(), p, wid, id, cfg, []float32{1, 0, 0})
	if e != nil || len(native) != 20 || diag.Scanned != 100 || !diag.Truncated || len(diag.Warnings) != 1 {
		t.Fatal(e, diag, len(native))
	}
	cfg["rag_backend"] = "pgvector"
	var installed bool
	s.DB.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector')`).Scan(&installed)
	pg, pgdiag, e := s.ragVectorCandidates(t.Context(), p, wid, id, cfg, []float32{1, 0, 0})
	if !installed {
		if e == nil || !strings.Contains(e.Error(), "설치") {
			t.Fatal("missing extension silently fell back", e, pgdiag)
		}
		t.Log("pgvector extension absent: explicit error verified; exact native array path verified")
		return
	}
	if e != nil || len(pg) != len(native) || pgdiag.Scanned != diag.Scanned || pgdiag.Truncated != diag.Truncated {
		t.Fatal(e, pgdiag, len(pg))
	}
	for i := range pg {
		if pg[i].Chunk.Index != native[i].Chunk.Index || math.Abs(pg[i].Score-native[i].Score) > 1e-5 {
			t.Fatal("pgvector/native disagreement", pg[i], native[i])
		}
	}
}
