package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed testdata/search-korean-evaluation.json
var searchKoreanEvaluationJSON []byte

func (s *Server) observedSearch(r *http.Request, wid string, filters map[string]string) (map[string]any, *searchObservation, int) {
	params := url.Values{}
	for _, key := range searchFilterKeys {
		if value := filters[key]; value != "" {
			params.Set(key, value)
		}
	}
	params.Set("workspace_id", wid)
	observation := &searchObservation{}
	request := r.Clone(context.WithValue(r.Context(), searchObservationKey, observation))
	request.Method = "GET"
	request.URL = &url.URL{Path: "/api/v1/search", RawQuery: params.Encode()}
	response := httptest.NewRecorder()
	s.universalSearch(response, request)
	var data map[string]any
	if e := json.Unmarshal(response.Body.Bytes(), &data); e != nil {
		return map[string]any{"error": "검색 결과를 해석할 수 없습니다"}, observation, 500
	}
	return data, observation, response.Code
}

func (s *Server) currentSearchOperator(r *http.Request, wid string) bool {
	p := current(r)
	cookie, e := r.Cookie("madi_session")
	if p == nil || e != nil || p.TokenID != "" || p.ScopeRestricted {
		return false
	}
	var ok bool
	e = s.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id JOIN sessions ss ON ss.user_id=u.id AND ss.token_hash=$3 AND ss.expires_at>now() WHERE u.id=$1 AND NOT u.disabled AND u.kind='user' AND u.role<>'viewer' AND m.workspace_id=$2 AND m.role IN ('owner','admin'))`, p.ID, wid, digest(cookie.Value)).Scan(&ok)
	return e == nil && ok
}

func (s *Server) searchDiagnostics(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	p := current(r)
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "검색 진단 권한이 없습니다")
		return
	}
	var in struct {
		Filters map[string]string `json:"filters"`
		RunPlan bool              `json:"run_plan"`
		Confirm bool              `json:"confirm"`
	}
	if decode(r, &in) != nil || !in.Confirm {
		apiError(w, 400, "현재 권한으로 읽기 검색 진단을 실행할지 확인하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	if in.Filters == nil {
		in.Filters = map[string]string{}
	}
	in.Filters["limit"] = "20"
	data, observation, status := s.observedSearch(r, wid, in.Filters)
	if status != 200 {
		jsonResponse(w, status, data)
		return
	}
	result := map[string]any{"interpretation": data["interpretation"], "elapsed_ms": float64(observation.Elapsed.Microseconds()) / 1000, "matched_in_page": len(data["results"].([]any)), "has_more": data["has_more"], "outcome": data["outcome"], "plan": []any{}, "plan_executed": false, "scope": "현재 열람 권한의 검색. 미열람 모집단·제거 행 수·SQL 상수는 표시하지 않습니다.", "history_recorded": false}
	if in.RunPlan {
		planCtx, done := context.WithTimeout(ctx, 5*time.Second)
		tx, e := s.DB.BeginTx(planCtx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if e != nil {
			done()
			respond(w, nil, e)
			return
		}
		if _, e = tx.Exec(planCtx, `SET LOCAL statement_timeout='3s';SET LOCAL lock_timeout='1s'`); e != nil {
			_ = tx.Rollback(planCtx)
			done()
			respond(w, nil, e)
			return
		}
		var raw []byte
		e = tx.QueryRow(planCtx, "EXPLAIN (ANALYZE TRUE,BUFFERS TRUE,TIMING FALSE,SUMMARY TRUE,FORMAT JSON) "+observation.SQL, observation.Arguments...).Scan(&raw)
		_ = tx.Rollback(planCtx)
		done()
		if e != nil {
			result["plan_warning"] = "실행 계획의 읽기 제한시간을 초과했거나 현재 DB 상태에서 진단할 수 없습니다. 운영 로그의 요청 식별자로 확인하세요."
		} else {
			var plans []map[string]any
			if json.Unmarshal(raw, &plans) == nil && len(plans) == 1 {
				result["plan"] = safeSearchPlan(plans[0]["Plan"])
				result["planning_ms"] = plans[0]["Planning Time"]
				result["execution_ms"] = plans[0]["Execution Time"]
				result["plan_executed"] = true
			}
		}
	}
	if !s.currentSearchOperator(r, wid) {
		apiError(w, 403, "현재 검색 진단 권한이 없습니다")
		return
	}
	for _, row := range data["results"].([]any) {
		value, _ := row.(map[string]any)
		id := str(value, "document_id")
		if id != "" && !s.canDocument(ctx, p, id, false) {
			apiError(w, 409, "검색 자료의 접근 범위가 변경되었습니다. 다시 진단하세요")
			return
		}
	}
	// Compute current-ACL counts after the potentially slow plan, never reuse
	// pre-plan population counts after a permission change.
	projection, e := s.one(ctx, `SELECT jsonb_build_object('readable_documents',count(*),'normalized_current',count(*) FILTER(WHERE z.document_version=d.version),'normalized_pending',count(*) FILTER(WHERE z.document_version IS DISTINCT FROM d.version),'incomplete_gram_prefilter',count(*) FILTER(WHERE z.document_version=d.version AND NOT z.grams_complete)) FROM documents d LEFT JOIN search_folded_documents z ON z.document_id=d.id WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)`, wid, p.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	result["projection"] = projection
	s.audit(r, "SEARCH_DIAGNOSTIC", wid, map[string]any{"plan_requested": in.RunPlan, "plan_executed": result["plan_executed"]})
	respond(w, result, nil)
}

func safeSearchPlan(raw any) []map[string]any {
	out := []map[string]any{}
	var visit func(any, int)
	visit = func(value any, depth int) {
		node, ok := value.(map[string]any)
		if !ok || len(out) >= 100 || depth > 30 {
			return
		}
		entry := map[string]any{"depth": depth, "node": str(node, "Node Type")}
		for _, key := range []string{"Index Name", "Join Type", "Strategy", "Scan Direction"} {
			if value := str(node, key); value != "" {
				entry[key] = value
			}
		}
		out = append(out, entry)
		children, _ := node["Plans"].([]any)
		for _, child := range children {
			visit(child, depth+1)
		}
	}
	visit(raw, 0)
	return out
}

func (s *Server) searchEvaluationFixture(w http.ResponseWriter, r *http.Request) {
	if !s.workspaceAdmin(r, r.PathValue("id")) {
		apiError(w, 403, "검색 평가 권한이 없습니다")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(searchKoreanEvaluationJSON)
}

func (s *Server) evaluateSearch(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "검색 평가 권한이 없습니다")
		return
	}
	var in struct {
		Confirm bool `json:"confirm"`
		TopK    int  `json:"top_k"`
		Cases   []struct {
			Query    string   `json:"query"`
			Expected []string `json:"expected_document_ids"`
		} `json:"cases"`
	}
	if decode(r, &in) != nil || !in.Confirm || in.TopK < 1 || in.TopK > 20 || len(in.Cases) < 1 || len(in.Cases) > 30 {
		apiError(w, 400, "확인한 평가 1~30개와 top_k 1~20을 입력하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	validateExpected := func(ids []string) error {
		if len(ids) < 1 || len(ids) > 20 {
			return errors.New("정답 문서는 평가별 1~20개입니다")
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if !validID(id) || seen[id] || !s.canDocument(ctx, current(r), id, false) {
				return errors.New("정답 문서에 현재 접근할 수 없습니다")
			}
			var allowed bool
			if e := s.DB.QueryRow(ctx, `SELECT workspace_id=$2 FROM documents WHERE id=$1`, id, wid).Scan(&allowed); e != nil || !allowed {
				return errors.New("정답 문서에 현재 접근할 수 없습니다")
			}
			seen[id] = true
		}
		return nil
	}
	for _, item := range in.Cases {
		if e := validateExpected(item.Expected); e != nil {
			apiError(w, 400, e.Error())
			return
		}
	}
	results := []map[string]any{}
	recall, mrr := 0.0, 0.0
	for _, item := range in.Cases {
		if !s.currentSearchOperator(r, wid) {
			apiError(w, 403, "현재 검색 평가 권한이 없습니다")
			return
		}
		if ctx.Err() != nil {
			apiError(w, 408, "평가 실행 제한시간을 초과했습니다. 평가 개수를 줄여주세요")
			return
		}
		data, observed, status := s.observedSearch(r, wid, map[string]string{"q": item.Query, "type": "document", "limit": "20"})
		if status != 200 {
			jsonResponse(w, status, data)
			return
		}
		if e := validateExpected(item.Expected); e != nil {
			apiError(w, 403, "평가 중 문서 접근 범위가 변경되었습니다")
			return
		}
		ranks := map[string]int{}
		rows := data["results"].([]any)
		for i, row := range rows {
			if i >= in.TopK {
				break
			}
			if value, ok := row.(map[string]any); ok {
				ranks[str(value, "document_id")] = i + 1
			}
		}
		matched, first := 0, 0
		expected := []map[string]any{}
		for _, id := range item.Expected {
			rank := ranks[id]
			expected = append(expected, map[string]any{"document_id": id, "rank": rank})
			if rank > 0 {
				matched++
				if first == 0 || rank < first {
					first = rank
				}
			}
		}
		r := float64(matched) / float64(len(item.Expected))
		rr := 0.0
		if first > 0 {
			rr = 1 / float64(first)
		}
		recall += r
		mrr += rr
		results = append(results, map[string]any{"query": item.Query, "expected": expected, "recall_at_k": r, "reciprocal_rank": rr, "elapsed_ms": float64(observed.Elapsed.Microseconds()) / 1000, "interpretation": data["interpretation"]})
	}
	if !s.currentSearchOperator(r, wid) {
		apiError(w, 403, "현재 검색 평가 권한이 없습니다")
		return
	}
	s.audit(r, "SEARCH_EVALUATION", wid, map[string]any{"cases": len(results), "top_k": in.TopK})
	respond(w, map[string]any{"cases": results, "top_k": in.TopK, "recall_at_k": recall / float64(len(results)), "mrr_at_k": mrr / float64(len(results)), "history_recorded": false, "scope": "제공한 정답 문서와 현재 열람 범위에 한정한 평가입니다. 전체 자료의 품질을 보증하지 않습니다."}, nil)
}
