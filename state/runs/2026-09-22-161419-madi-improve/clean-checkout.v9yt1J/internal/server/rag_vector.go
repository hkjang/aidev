package server

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type ragDiagnostics struct {
	Mode             string   `json:"mode"`
	Backend          string   `json:"backend"`
	Scanned          int      `json:"scanned"`
	Truncated        bool     `json:"truncated"`
	Reranked         bool     `json:"reranked"`
	Warnings         []string `json:"warnings"`
	VectorMode       string   `json:"vector_mode,omitempty"`
	GenerationID     string   `json:"generation_id,omitempty"`
	IndexName        string   `json:"index_name,omitempty"`
	PlannedIndexUsed bool     `json:"planned_index_used,omitempty"`
	RecallAtK        *float64 `json:"recall_at_k,omitempty"`
}
type ragVectorCandidate struct {
	DocumentID string
	Version    int
	Chunk      ragChunk
	Score      float64
}
type ragVectorHeap []ragVectorCandidate

func (h ragVectorHeap) Len() int { return len(h) }
func (h ragVectorHeap) Less(i, j int) bool {
	if h[i].Score == h[j].Score {
		if h[i].DocumentID == h[j].DocumentID {
			return h[i].Chunk.Index > h[j].Chunk.Index
		}
		return h[i].DocumentID > h[j].DocumentID
	}
	return h[i].Score < h[j].Score
}
func (h ragVectorHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *ragVectorHeap) Push(v any)   { *h = append(*h, v.(ragVectorCandidate)) }
func (h *ragVectorHeap) Pop() any     { old := *h; v := old[len(old)-1]; *h = old[:len(old)-1]; return v }

// Both backends intentionally use the same bounded, current-ACL candidate set.
// The array path streams one vector at a time; it never loads 50k*8192 vectors.
const ragVectorEligible = ` FROM rag_vector_chunks c JOIN rag_vector_indexes i ON i.id=c.index_id JOIN rag_index_grants g ON g.id=i.grant_id JOIN documents d ON d.id=i.document_id JOIN users owner ON owner.id=g.actor_id
 WHERE g.active AND NOT owner.disabled AND madi_document_allowed(owner.id,d.id,true)
 AND g.generation_id IS NOT DISTINCT FROM NULLIF($7,'')::uuid
 AND (coalesce(cardinality($8::uuid[]),0)=0 OR d.id=ANY($8::uuid[]))
 AND i.status='ready' AND i.grant_revision=g.revision AND i.document_version=d.version AND i.provider_fingerprint=g.provider_fingerprint AND g.provider_fingerprint=$1
 AND d.deleted_at IS NULL AND d.workspace_id=$2 AND ($3='' OR d.id::text=$3) AND madi_document_allowed($4,d.id,false)
 AND i.dimensions=$5 AND cardinality(c.embedding)=$5
 AND (g.token_id IS NULL OR EXISTS(SELECT 1 FROM api_keys k WHERE k.id=g.token_id AND k.user_id=g.actor_id AND k.workspace_id=g.workspace_id AND k.revoked_at IS NULL AND k.expires_at>now() AND k.scopes @> ARRAY['document:write','ai:execute']::text[]))
 ORDER BY i.updated_at DESC,d.id,c.ordinal LIMIT $6`

func (s *Server) ragExactVectorCandidates(ctx context.Context, p *Principal, wid, docID string, cfg map[string]any, query []float32, generation string, cohort []string) ([]ragVectorCandidate, ragDiagnostics, error) {
	limit := number(cfg, "rag_scan_limit", 5000)
	top := number(cfg, "rag_candidates", 60)
	diagnostic := ragDiagnostics{Mode: str(cfg, "rag_search_mode"), Backend: str(cfg, "rag_backend"), VectorMode: "exact", GenerationID: generation, Warnings: []string{}}
	if limit < 100 || limit > 50000 || top < 1 || top > 200 {
		return nil, diagnostic, errors.New("검색 후보·조회 한도 설정을 확인하세요")
	}
	cols := `d.id::text,d.version,c.ordinal,c.content_hash,c.start_byte,c.end_byte,c.start_line,c.end_line,c.heading,c.embedding`
	args := []any{ragProviderFingerprint(cfg), wid, docID, p.ID, len(query), limit + 1, generation, cohort}
	backend := str(cfg, "rag_backend")
	sql := "SELECT " + cols + ragVectorEligible
	if backend == "pgvector" {
		var namespace string
		if e := s.DB.QueryRow(ctx, `SELECT n.nspname FROM pg_extension e JOIN pg_namespace n ON n.oid=e.extnamespace WHERE e.extname='vector'`).Scan(&namespace); e != nil {
			return nil, diagnostic, errors.New("pgvector 확장이 설치되어 있지 않습니다. 운영자가 확장을 설치하거나 배열 검색을 선택하세요")
		}
		qualified := pgx.Identifier{namespace}.Sanitize()
		sql = `WITH eligible AS MATERIALIZED (SELECT d.id::text AS document_id,d.version,c.ordinal,c.content_hash,start_byte,end_byte,start_line,end_line,c.heading,c.embedding,row_number() OVER(ORDER BY i.updated_at DESC,d.id,c.ordinal) AS scan_ordinal` + ragVectorEligible + `) SELECT document_id,version,ordinal,content_hash,start_byte,end_byte,start_line,end_line,heading,1-(embedding::` + qualified + `.vector OPERATOR(` + qualified + `.<=>) $9::real[]::` + qualified + `.vector) AS score,(SELECT count(*) FROM eligible) FROM eligible WHERE scan_ordinal<$6 ORDER BY score DESC,document_id,ordinal LIMIT $10`
		args = append(args, query, top)
	} else if backend != "array" {
		return nil, diagnostic, errors.New("벡터 검색 방식을 확인하세요")
	}
	rows, e := s.DB.Query(ctx, sql, args...)
	if e != nil {
		return nil, diagnostic, e
	}
	defer rows.Close()
	ranked := &ragVectorHeap{}
	heap.Init(ranked)
	for rows.Next() {
		var item ragVectorCandidate
		var vec []float32
		var count int
		dest := []any{&item.DocumentID, &item.Version, &item.Chunk.Index, &item.Chunk.Hash, &item.Chunk.Start, &item.Chunk.End, &item.Chunk.StartLine, &item.Chunk.EndLine, &item.Chunk.Heading}
		if backend == "pgvector" {
			dest = append(dest, &item.Score, &count)
		} else {
			dest = append(dest, &vec)
		}
		if e = rows.Scan(dest...); e != nil {
			return nil, diagnostic, e
		}
		if backend == "pgvector" {
			diagnostic.Scanned = min(count, limit)
			diagnostic.Truncated = count > limit
			if math.IsNaN(item.Score) || math.IsInf(item.Score, 0) {
				return nil, diagnostic, errors.New("pgvector가 유효하지 않은 유사도를 반환했습니다")
			}
		} else {
			diagnostic.Scanned++
			if diagnostic.Scanned > limit {
				diagnostic.Scanned = limit
				diagnostic.Truncated = true
				break
			}
			var valid bool
			item.Score, valid = ragCosine(query, vec)
			if !valid {
				continue
			}
		}
		heap.Push(ranked, item)
		if ranked.Len() > top {
			heap.Pop(ranked)
		}
	}
	if e = rows.Err(); e != nil {
		return nil, diagnostic, e
	}
	out := []ragVectorCandidate(*ranked)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			if out[i].DocumentID == out[j].DocumentID {
				return out[i].Chunk.Index < out[j].Chunk.Index
			}
			return out[i].DocumentID < out[j].DocumentID
		}
		return out[i].Score > out[j].Score
	})
	if diagnostic.Truncated {
		diagnostic.Warnings = append(diagnostic.Warnings, fmt.Sprintf("설정된 %d개 조각 범위에서 정확 코사인 검색했습니다. 전체 지식 저장소의 전역 최근접 결과는 아닙니다.", limit))
	}
	return out, diagnostic, nil
}

func (s *Server) ragCandidateSources(ctx context.Context, p *Principal, wid string, candidates []ragVectorCandidate, cfg map[string]any) ([]aiSource, error) {
	out := []aiSource{}
	checked := map[string]ragIndexGrant{}
	for _, item := range candidates {
		if _, ok := checked[item.DocumentID]; !ok {
			g, e := ragGrantTx(ctx, s.DB, item.DocumentID, false)
			if e != nil || !g.Active || g.Provider != ragProviderFingerprint(cfg) {
				continue
			}
			if _, e = s.ragCurrentActor(ctx, g); e != nil {
				continue
			}
			checked[item.DocumentID] = g
		}
		c := item.Chunk
		if c.Start < 0 || c.End <= c.Start || c.End-c.Start > 8192 {
			continue
		}
		var title, body string
		e := s.DB.QueryRow(ctx, `SELECT title,convert_from(substring(convert_to(markdown,'UTF8') FROM $4+1 FOR $5-$4),'UTF8') FROM documents WHERE id=$1 AND workspace_id=$2 AND version=$3 AND deleted_at IS NULL AND madi_document_allowed($6,id,false) AND octet_length(markdown)>=$5`, item.DocumentID, wid, item.Version, c.Start, c.End, p.ID).Scan(&title, &body)
		if errors.Is(e, pgx.ErrNoRows) {
			continue
		}
		if e != nil {
			return nil, e
		}
		if !utf8.ValidString(body) || digest(body) != c.Hash {
			continue
		}
		c.Content = body
		source := sourceFromChunk(item.DocumentID, title, item.Version, c)
		source.RAGGrantID = checked[item.DocumentID].ID
		source.RAGGrantRevision = checked[item.DocumentID].Revision
		out = append(out, source)
	}
	return out, nil
}

func (s *Server) ragRetrievalGuard(r *http.Request, p *Principal, wid string, sources []aiSource, cfg map[string]any) error {
	if e := s.validateAIStream(r, p, wid, sources, cfg); e != nil {
		return errRAGChanged
	}
	if !boolean(cfg, "rag_enabled") {
		return errRAGChanged
	}
	return nil
}

func ragFuse(keyword, semantic []aiSource, top int) []aiSource {
	candidates := map[string]rankedAISource{}
	for _, list := range [][]aiSource{keyword, semantic} {
		for i, source := range list {
			v := candidates[source.CitationID]
			v.Source = source
			v.Score += 1 / float64(60+i+1)
			candidates[source.CitationID] = v
		}
	}
	ranked := []rankedAISource{}
	for _, v := range candidates {
		ranked = append(ranked, v)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Source.CitationID < ranked[j].Source.CitationID
		}
		return ranked[i].Score > ranked[j].Score
	})
	out := []aiSource{}
	for _, v := range ranked {
		if len(out) >= top {
			break
		}
		out = append(out, v.Source)
	}
	return out
}

// Root's AI route consumes these exact citations and exposes diagnostics before
// streaming. Missing/unconsented indexes are reported, not fabricated answers.
func (s *Server) retrieveRAGAISources(r *http.Request, p *Principal, documentID, workspaceID, prompt string) ([]aiSource, ragDiagnostics, error) {
	cfg, e := s.effectiveSettings(r.Context(), workspaceID)
	diagnostic := ragDiagnostics{Mode: "keyword", Backend: "none", Warnings: []string{}}
	if e != nil {
		return nil, diagnostic, e
	}
	mode := str(cfg, "rag_search_mode")
	if !boolean(cfg, "rag_enabled") || mode == "keyword" {
		out, e := s.keywordAISources(r, p, documentID, workspaceID, prompt)
		return out, diagnostic, e
	}
	if workspaceID == "" || !hasIntegrationScope(p, "ai:execute") || !hasIntegrationScope(p, "document:read") || !s.canWorkspace(r.Context(), p, workspaceID, false) {
		return nil, diagnostic, errRAGChanged
	}
	if documentID != "" && !s.canDocument(r.Context(), p, documentID, false) {
		return nil, diagnostic, errRAGChanged
	}
	diagnostic.Mode = mode
	diagnostic.Backend = str(cfg, "rag_backend")
	keywords := []aiSource{}
	if mode == "hybrid" {
		keywords, e = s.keywordAISources(r, p, documentID, workspaceID, prompt)
		if e != nil {
			return nil, diagnostic, e
		}
	}
	probe, e := s.ragQuerySource(r.Context(), p, workspaceID, documentID, cfg)
	if e != nil {
		return nil, diagnostic, e
	}
	if probe == nil {
		diagnostic.Warnings = append(diagnostic.Warnings, "현재 공급자·문서 버전에 동의된 벡터 색인이 없습니다. 문서에서 색인을 요청하세요.")
		return keywords, diagnostic, nil
	}
	querySources := append(append([]aiSource{}, keywords...), *probe)
	if e = s.ragRetrievalGuard(r, p, workspaceID, querySources, cfg); e != nil {
		return nil, diagnostic, e
	}
	if strings.TrimSpace(prompt) == "" || len(prompt) > 8192 {
		return nil, diagnostic, errors.New("의미 검색 질문은 비어 있지 않은 8192바이트 이하 텍스트여야 합니다")
	}
	child, stop, guardErr := ragWatch(r.Context(), func(ctx context.Context) error {
		return s.ragRetrievalGuard(r.WithContext(ctx), p, workspaceID, querySources, cfg)
	})
	vectors, e := ragEmbeddings(child, ragEmbeddingProvider(cfg), []string{prompt})
	stop()
	if *guardErr != nil {
		return nil, diagnostic, *guardErr
	}
	if e != nil {
		return nil, diagnostic, e
	}
	if e = s.ragRetrievalGuard(r, p, workspaceID, querySources, cfg); e != nil {
		return nil, diagnostic, e
	}
	candidates, scan, e := s.ragVectorCandidates(r.Context(), p, workspaceID, documentID, cfg, vectors[0])
	if e != nil {
		return nil, scan, e
	}
	diagnostic = scan
	semantic, e := s.ragCandidateSources(r.Context(), p, workspaceID, candidates, cfg)
	if e != nil {
		return nil, diagnostic, e
	}
	if len(semantic) == 0 {
		diagnostic.Warnings = append(diagnostic.Warnings, "현재 권한·공급자·차원에 일치하는 벡터 검색 결과가 없습니다.")
	}
	sources := semantic
	if mode == "hybrid" {
		sources = ragFuse(keywords, semantic, number(cfg, "rag_candidates", 60))
	}
	if e = s.ragRetrievalGuard(r, p, workspaceID, sources, cfg); e != nil {
		return nil, diagnostic, e
	}
	sources, e = s.ragApplyRerank(r, p, workspaceID, prompt, sources, cfg, &diagnostic)
	if e != nil {
		return nil, diagnostic, e
	}
	top := number(cfg, "rag_top_k", 8)
	if len(sources) > top {
		sources = sources[:top]
	}
	return sources, diagnostic, nil
}
