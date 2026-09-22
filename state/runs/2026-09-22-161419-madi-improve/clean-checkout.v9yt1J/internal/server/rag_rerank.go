package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
)

// Cohere/Jina-compatible vLLM /rerank contract. Returned document strings are
// ignored: only a complete, finite, unique permutation of the inputs is accepted.
func ragRerank(ctx context.Context, p ragProvider, query string, documents []string) ([]int, error) {
	if len(documents) < 1 || len(documents) > 200 || query == "" || len(query) > 8192 || p.Model == "" {
		return nil, errors.New("재정렬 입력 범위를 확인하세요")
	}
	for _, d := range documents {
		if len(d) > 8192 || d == "" {
			return nil, errors.New("재정렬 출처 범위를 확인하세요")
		}
	}
	endpoint, e := ragEndpoint(p.BaseURL, "rerank")
	if e != nil {
		return nil, e
	}
	client, e := ragHTTP(p)
	if e != nil {
		return nil, e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonValue(map[string]any{"model": p.Model, "query": query, "documents": documents, "top_n": len(documents), "return_documents": false})))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("api-key", p.APIKey)
	}
	res, e := client.Do(req)
	if e != nil {
		return nil, errors.New("재정렬 공급자에 연결할 수 없습니다. 주소·네트워크·인증서를 확인하세요")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("재정렬 공급자가 HTTP %d 오류를 반환했습니다", res.StatusCode)
	}
	if !strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "application/json") {
		return nil, errors.New("재정렬 공급자가 JSON 응답을 반환하지 않았습니다")
	}
	raw, e := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if e != nil || len(raw) > 8<<20 {
		return nil, errors.New("재정렬 응답 크기 또는 읽기 오류입니다")
	}
	var body struct {
		Results []struct {
			Index *int     `json:"index"`
			Score *float64 `json:"relevance_score"`
		} `json:"results"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil || len(body.Results) != len(documents) || (len(body.Error) > 0 && string(body.Error) != "null") {
		return nil, errors.New("재정렬 응답의 출처 개수가 맞지 않습니다")
	}
	seen := map[int]bool{}
	out := []int{}
	last := math.Inf(1)
	for _, item := range body.Results {
		if item.Index == nil || item.Score == nil || *item.Index < 0 || *item.Index >= len(documents) || seen[*item.Index] || math.IsNaN(*item.Score) || math.IsInf(*item.Score, 0) || *item.Score > last {
			return nil, errors.New("재정렬 응답의 순서·중복·점수가 올바르지 않습니다")
		}
		seen[*item.Index] = true
		last = *item.Score
		out = append(out, *item.Index)
	}
	return out, nil
}

func (s *Server) ragRerankConsent(ctx context.Context, sources []aiSource, cfg map[string]any) error {
	seen := map[string]bool{}
	for _, source := range sources {
		if seen[source.ID] {
			continue
		}
		seen[source.ID] = true
		g, e := ragGrantTx(ctx, s.DB, source.ID, false)
		if e != nil || !g.Active || g.Version != source.Version || g.Provider != ragProviderFingerprint(cfg) || g.Rerank == "" || g.Rerank != ragRerankFingerprint(cfg) {
			return errRAGChanged
		}
		if _, e = s.ragCurrentActor(ctx, g); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) ragApplyRerank(r *http.Request, p *Principal, wid, prompt string, sources []aiSource, cfg map[string]any, diagnostic *ragDiagnostics) ([]aiSource, error) {
	if !boolean(cfg, "rag_rerank_enabled") || len(sources) == 0 {
		return sources, nil
	}
	if e := s.ragRerankConsent(r.Context(), sources, cfg); e != nil {
		diagnostic.Warnings = append(diagnostic.Warnings, "일부 출처에 현재 재정렬 공급자 전송 동의가 없어 재정렬을 생략했습니다.")
		return sources, nil
	}
	// Pin consent before the network call, including keyword-only candidates.
	// A revoke/reconsent between two monitor ticks cannot bless an older request.
	sources = append([]aiSource{}, sources...)
	for i := range sources {
		g, e := ragGrantTx(r.Context(), s.DB, sources[i].ID, false)
		if e != nil || !g.Active || g.Version != sources[i].Version || g.Rerank != ragRerankFingerprint(cfg) {
			return nil, errRAGChanged
		}
		if sources[i].RAGGrantID != "" && (sources[i].RAGGrantID != g.ID || sources[i].RAGGrantRevision != g.Revision) {
			return nil, errRAGChanged
		}
		sources[i].RAGGrantID = g.ID
		sources[i].RAGGrantRevision = g.Revision
	}
	check := func(ctx context.Context) error {
		if e := s.ragRetrievalGuard(r.WithContext(ctx), p, wid, sources, cfg); e != nil {
			return e
		}
		return s.ragRerankConsent(ctx, sources, cfg)
	}
	if e := check(r.Context()); e != nil {
		return nil, e
	}
	texts := []string{}
	for _, source := range sources {
		texts = append(texts, source.Markdown)
	}
	child, stop, guardErr := ragWatch(r.Context(), check)
	order, e := ragRerank(child, ragRerankProvider(cfg), prompt, texts)
	stop()
	if *guardErr != nil {
		return nil, *guardErr
	}
	if e != nil {
		return nil, e
	}
	if e = check(r.Context()); e != nil {
		return nil, e
	}
	out := make([]aiSource, 0, len(order))
	for _, i := range order {
		out = append(out, sources[i])
	}
	if e = s.validateRAGSourceGrants(r.Context(), out, cfg); e != nil {
		return nil, e
	}
	diagnostic.Reranked = true
	return out, nil
}
