package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type ragGenerationUnavailable string

func (e ragGenerationUnavailable) Error() string { return string(e) }

func (s *Server) ragVectorCandidates(ctx context.Context, p *Principal, wid, docID string, cfg map[string]any, query []float32) ([]ragVectorCandidate, ragDiagnostics, error) {
	generation, e := ragSelectedGeneration(ctx, s.DB, wid, "")
	if e != nil {
		return nil, ragDiagnostics{}, e
	}
	mode := str(cfg, "rag_vector_mode")
	if mode == "" {
		mode = "exact"
	}
	if generation != "" {
		var servingMode string
		if e = s.DB.QueryRow(ctx, `SELECT mode FROM rag_generation_state WHERE workspace_id=$1 AND active_id=$2`, wid, generation).Scan(&servingMode); e != nil || servingMode != mode {
			return nil, ragDiagnostics{}, ragGenerationUnavailable("벡터 모드와 검증된 활성 세대 상태가 다릅니다. 세대 검증·전환에서 모드를 적용하세요")
		}
	} else if mode != "exact" {
		return nil, ragDiagnostics{}, ragGenerationUnavailable("ANN·검증 모드는 색인 세대를 준비하고 검증한 뒤 활성화하세요")
	}
	return s.ragGenerationCandidates(ctx, p, wid, docID, cfg, query, generation, mode, nil)
}
func (s *Server) ragGenerationCandidates(ctx context.Context, p *Principal, wid, docID string, cfg map[string]any, query []float32, generation, mode string, cohort []string) ([]ragVectorCandidate, ragDiagnostics, error) {
	if mode == "exact" {
		return s.ragExactVectorCandidates(ctx, p, wid, docID, cfg, query, generation, cohort)
	}
	if !oneOf(mode, "ann", "verify") {
		return nil, ragDiagnostics{}, ragGenerationUnavailable("벡터 실행 모드를 확인하세요")
	}
	result, diagnostic, e := s.ragANNVectorCandidates(ctx, p, wid, docID, cfg, query, generation, cohort)
	if e != nil {
		return nil, diagnostic, e
	}
	if mode == "verify" {
		reference := map[string]any{}
		for key, value := range cfg {
			reference[key] = value
		}
		reference["rag_backend"] = "array"
		exact, check, e := s.ragExactVectorCandidates(ctx, p, wid, docID, reference, query, generation, cohort)
		if e != nil {
			return nil, diagnostic, e
		}
		score := ragRecall(exact, result)
		diagnostic.RecallAtK = &score
		diagnostic.VectorMode = "verify"
		diagnostic.Truncated = check.Truncated
		diagnostic.Warnings = append(diagnostic.Warnings, "검증 모드는 같은 현재 접근 범위에서 배열 정확 검색을 추가 실행해 후보 Recall@k를 비교하며 ANN 결과를 반환합니다.")
		if check.Truncated {
			diagnostic.Warnings = append(diagnostic.Warnings, "정확 기준도 scan_limit 범위로 제한되어 전체 자료의 전역 recall 값이 아닙니다.")
		}
	}
	return result, diagnostic, nil
}
func ragRecall(exact, approx []ragVectorCandidate) float64 {
	if len(exact) == 0 {
		return 1
	}
	seen := map[string]bool{}
	for _, v := range approx {
		seen[v.DocumentID+":"+strconv.Itoa(v.Chunk.Index)] = true
	}
	matched := 0
	for _, v := range exact {
		if seen[v.DocumentID+":"+strconv.Itoa(v.Chunk.Index)] {
			matched++
		}
	}
	return float64(matched) / float64(len(exact))
}
func ragHNSWIndexName(generation string) string {
	return "madi_rag_hnsw_" + strings.ReplaceAll(generation, "-", "")
}
func ragPlanUsesIndex(value any, name string) bool {
	switch v := value.(type) {
	case map[string]any:
		if str(v, "Index Name") == name {
			return true
		}
		for _, child := range v {
			if ragPlanUsesIndex(child, name) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if ragPlanUsesIndex(child, name) {
				return true
			}
		}
	}
	return false
}
func (s *Server) ragVectorExtension(ctx context.Context) (string, string, error) {
	var namespace, version string
	e := s.DB.QueryRow(ctx, `SELECT n.nspname,e.extversion FROM pg_extension e JOIN pg_namespace n ON n.oid=e.extnamespace WHERE e.extname='vector'`).Scan(&namespace, &version)
	if e != nil {
		return "", "", ragGenerationUnavailable("pgvector 확장은 운영자가 먼저 설치해야 합니다. 서비스는 확장을 자동 설치하지 않습니다")
	}
	return namespace, version, nil
}

func ragANNQuery(namespace, generation string, dimensions int) string {
	qualified := pgx.Identifier{namespace}.Sanitize()
	vectorType := qualified + ".vector(" + strconv.Itoa(dimensions) + ")"
	distance := `(c.embedding::` + vectorType + ` OPERATOR(` + qualified + `.<=>) $9::real[]::` + vectorType + `)`
	eligible := strings.Split(ragVectorEligible, " ORDER BY i.updated_at")[0]
	eligible = strings.TrimPrefix(eligible, ` FROM rag_vector_chunks c JOIN rag_vector_indexes i ON i.id=c.index_id`)
	eligible = strings.Replace(eligible, " WHERE g.active", " WHERE i.id=c.index_id AND $6::int>0 AND g.active", 1)
	return `SELECT permitted.document_id,permitted.version,c.ordinal,c.content_hash,c.start_byte,c.end_byte,c.start_line,c.end_line,c.heading,1-` + distance + ` AS score FROM rag_vector_chunks c CROSS JOIN LATERAL (SELECT d.id::text AS document_id,d.version FROM rag_vector_indexes i` + eligible + ` OFFSET 0) permitted WHERE c.generation_id='` + generation + `'::uuid AND cardinality(c.embedding)=` + strconv.Itoa(dimensions) + ` ORDER BY ` + distance + ` LIMIT $10`
}

func (s *Server) ragANNVectorCandidates(ctx context.Context, p *Principal, wid, docID string, cfg map[string]any, query []float32, generation string, cohort []string) ([]ragVectorCandidate, ragDiagnostics, error) {
	diagnostic := ragDiagnostics{Mode: str(cfg, "rag_search_mode"), Backend: "pgvector", VectorMode: "ann", GenerationID: generation, Warnings: []string{}}
	if !validID(generation) || len(query) < 1 || len(query) > 2000 {
		return nil, diagnostic, ragGenerationUnavailable("ANN은 명시적인 색인 세대와 1~2000차원 벡터가 필요합니다")
	}
	g, e := s.ragGenerationTx(ctx, s.DB, generation, false)
	if e != nil || g.WorkspaceID != wid || g.Status == "disabled" || g.Dimensions != len(query) || g.Provider != ragProviderFingerprint(cfg) {
		return nil, diagnostic, errRAGChanged
	}
	namespace, version, e := s.ragVectorExtension(ctx)
	if e != nil {
		return nil, diagnostic, e
	}
	if !ragVectorSupportsIterative(version) {
		return nil, diagnostic, ragGenerationUnavailable("ANN 권한 필터 반복 검색에는 pgvector 0.8 이상이 필요합니다")
	}
	name := ragHNSWIndexName(generation)
	diagnostic.IndexName = name
	var indexReady bool
	e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_am am ON am.oid=c.relam WHERE c.relname=$1 AND c.relnamespace=(SELECT relnamespace FROM pg_class WHERE oid='rag_vector_chunks'::regclass) AND i.indisvalid AND i.indisready AND am.amname='hnsw')`, name).Scan(&indexReady)
	if e != nil || !indexReady {
		return nil, diagnostic, ragGenerationUnavailable("현재 세대의 유효한 HNSW 인덱스를 먼저 준비하세요. 정확 검색으로 자동 대체하지 않습니다")
	}
	limit := number(cfg, "rag_scan_limit", 5000)
	top := number(cfg, "rag_candidates", 60)
	ef := number(cfg, "rag_ann_ef_search", 100)
	if limit < 100 || limit > 50000 || top < 1 || top > 200 || ef < 40 || ef > 1000 {
		return nil, diagnostic, ragGenerationUnavailable("ANN 탐색 예산을 확인하세요")
	}
	// A LATERAL current-ACL lookup (OFFSET 0 prevents join flattening) preserves
	// the vector table as the ordered driver. Filtering remains before LIMIT;
	// iteration can skip unauthorized neighbors without a pre-ACL top-k cutoff.
	// Fixed UUID/dimension expressions match the generation's partial index.
	sql := ragANNQuery(namespace, generation, len(query))
	args := []any{ragProviderFingerprint(cfg), wid, docID, p.ID, len(query), limit + 1, generation, cohort, query, top}
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if e != nil {
		return nil, diagnostic, e
	}
	defer tx.Rollback(ctx)
	// All choices are request-local to an explicitly selected ANN mode. No
	// persistent PostgreSQL tuning or global extension installation occurs.
	if _, e = tx.Exec(ctx, `SELECT set_config('hnsw.ef_search',$1,true),set_config('hnsw.iterative_scan','strict_order',true),set_config('hnsw.max_scan_tuples',$2,true),set_config('enable_seqscan','off',true),set_config('enable_sort','off',true)`, strconv.Itoa(ef), strconv.Itoa(limit)); e != nil {
		return nil, diagnostic, e
	}
	var raw []byte
	execArgs := append([]any{pgx.QueryExecModeExec}, args...)
	if e = tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+sql, execArgs...).Scan(&raw); e != nil {
		return nil, diagnostic, e
	}
	var plan any
	if json.Unmarshal(raw, &plan) != nil || !ragPlanUsesIndex(plan, name) {
		return nil, diagnostic, ragGenerationUnavailable("요청한 HNSW 실행 계획을 사용하지 못했습니다. 검색 진단에서 인덱스 상태를 확인하세요")
	}
	diagnostic.PlannedIndexUsed = true
	rows, e := tx.Query(ctx, sql, execArgs...)
	if e != nil {
		return nil, diagnostic, e
	}
	out := []ragVectorCandidate{}
	for rows.Next() {
		var item ragVectorCandidate
		if e = rows.Scan(&item.DocumentID, &item.Version, &item.Chunk.Index, &item.Chunk.Hash, &item.Chunk.Start, &item.Chunk.End, &item.Chunk.StartLine, &item.Chunk.EndLine, &item.Chunk.Heading, &item.Score); e != nil {
			rows.Close()
			return nil, diagnostic, e
		}
		if math.IsNaN(item.Score) || math.IsInf(item.Score, 0) {
			rows.Close()
			return nil, diagnostic, ragGenerationUnavailable("ANN 유사도가 유효하지 않습니다")
		}
		out = append(out, item)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, diagnostic, e
	}
	diagnostic.Scanned = len(out)
	diagnostic.Warnings = append(diagnostic.Warnings, fmt.Sprintf("HNSW 근사 후보 %d개 반환. scanned는 반환된 현재 권한 후보 수이며 인덱스 내부 방문 수가 아닙니다. 탐색 예산/권한 필터로 결과가 부족할 수 있습니다.", len(out)))
	return out, diagnostic, nil
}
func ragVectorSupportsIterative(version string) bool {
	var major, minor int
	if _, e := fmt.Sscanf(version, "%d.%d", &major, &minor); e != nil {
		return false
	}
	return major > 0 || minor >= 8
}
