package server

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
)

func defaultRAGSettings() map[string]any {
	return map[string]any{
		"rag_enabled": false, "rag_embedding_base_url": "", "rag_embedding_model": "", "rag_embedding_api_key": "", "rag_embedding_dimensions": 0, "rag_allow_http": false, "rag_ca_pem": "",
		"rag_backend": "array", "rag_top_k": 8, "rag_candidates": 60, "rag_scan_limit": 5000, "rag_search_mode": "hybrid",
		"rag_vector_mode": "exact", "rag_ann_ef_search": 100,
		"rag_rerank_enabled": false, "rag_rerank_base_url": "", "rag_rerank_model": "", "rag_rerank_api_key": "",
	}
}
func ragSettingKeys() []string {
	out := []string{}
	for key := range defaultRAGSettings() {
		out = append(out, key)
	}
	return out
}
func isWorkspaceSecret(key string) bool {
	return oneOf(key, "ai_api_key", "rag_embedding_api_key", "rag_rerank_api_key")
}
func validateRAGSettings(cfg map[string]any) error {
	for _, key := range []string{"rag_enabled", "rag_allow_http", "rag_rerank_enabled"} {
		if _, ok := cfg[key].(bool); !ok {
			return fmt.Errorf("%s 값은 true/false여야 합니다", key)
		}
	}
	for key, bounds := range map[string][2]int{"rag_embedding_dimensions": {0, 8192}, "rag_top_k": {1, 20}, "rag_candidates": {10, 200}, "rag_scan_limit": {100, 50000}, "rag_ann_ef_search": {40, 1000}} {
		n := number(cfg, key, -1)
		if n < bounds[0] || n > bounds[1] {
			return fmt.Errorf("%s 범위는 %d~%d입니다", key, bounds[0], bounds[1])
		}
		switch v := cfg[key].(type) {
		case int:
		case float64:
			if math.Trunc(v) != v {
				return fmt.Errorf("%s 값은 정수여야 합니다", key)
			}
		default:
			return fmt.Errorf("%s 값은 숫자여야 합니다", key)
		}
	}
	if !oneOf(str(cfg, "rag_backend"), "array", "pgvector") || !oneOf(str(cfg, "rag_search_mode"), "keyword", "semantic", "hybrid") {
		return errors.New("검색 방식 또는 벡터 저장 방식을 확인하세요")
	}
	if !oneOf(str(cfg, "rag_vector_mode"), "exact", "ann", "verify") {
		return errors.New("벡터 실행 모드는 exact/ann/verify 중 하나입니다")
	}
	if number(cfg, "rag_candidates", 60) < number(cfg, "rag_top_k", 8) {
		return errors.New("검색 후보 수는 최종 출처 수 이상이어야 합니다")
	}
	for _, prefix := range []string{"rag_embedding", "rag_rerank"} {
		base, model := str(cfg, prefix+"_base_url"), str(cfg, prefix+"_model")
		if len(model) > 256 || len(base) > 4096 {
			return errors.New("검색 AI 모델·주소 길이를 확인하세요")
		}
		if base != "" {
			if _, e := ragEndpoint(base, "embeddings"); e != nil {
				return e
			}
		}
		enabled := boolean(cfg, "rag_enabled")
		if prefix == "rag_rerank" {
			enabled = enabled && boolean(cfg, "rag_rerank_enabled")
		}
		if enabled && (base == "" || strings.TrimSpace(model) == "") {
			return errors.New("검색 AI를 활성화하려면 API 주소와 모델이 필요합니다")
		}
		if enabled && strings.HasPrefix(base, "http:") && !boolean(cfg, "rag_allow_http") {
			return errors.New("내부 HTTP 검색 AI 연결은 명시적인 허용이 필요합니다")
		}
	}
	if len(str(cfg, "rag_ca_pem")) > 65536 {
		return errors.New("검색 AI CA 인증서는 64KiB 이하여야 합니다")
	}
	if _, e := ragHTTP(ragProvider{BaseURL: "https://localhost", CA: str(cfg, "rag_ca_pem")}); e != nil {
		return e
	}
	return nil
}
func ragEmbeddingProvider(cfg map[string]any) ragProvider {
	return ragProvider{BaseURL: str(cfg, "rag_embedding_base_url"), Model: str(cfg, "rag_embedding_model"), APIKey: str(cfg, "rag_embedding_api_key"), CA: str(cfg, "rag_ca_pem"), AllowHTTP: boolean(cfg, "rag_allow_http"), Dimensions: number(cfg, "rag_embedding_dimensions", 0)}
}
func ragProviderFingerprint(cfg map[string]any) string {
	// A key/CA/endpoint change also invalidates the old index grant. A new provider
	// cannot silently reuse vectors or resend private sources approved for another.
	return digest(string(jsonValue(ragEmbeddingProvider(cfg))))
}
func (s *Server) registerRAGSettings() {
	s.admin("POST /api/v1/admin/search-ai/test", s.testRAGProvider)
	s.handle("GET /api/v1/workspaces/{id}/search-ai", s.getWorkspaceRAGSettings)
	s.handle("POST /api/v1/workspaces/{id}/search-ai/test", s.testRAGProvider)
}
func (s *Server) getWorkspaceRAGSettings(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 검색 AI 설정 권한이 없습니다")
		return
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	data := map[string]any{}
	safe := redactSettings(cfg)
	for _, key := range ragSettingKeys() {
		data[key] = safe[key]
		if isWorkspaceSecret(key) {
			data[key+"_configured"] = safe[key+"_configured"]
		}
	}
	var version int
	var overrides []string
	e = s.DB.QueryRow(r.Context(), `SELECT coalesce(x.version,0),ARRAY(SELECT jsonb_object_keys(coalesce(x.data,'{}'::jsonb))) FROM workspaces w LEFT JOIN workspace_settings x ON x.workspace_id=w.id WHERE w.id=$1`, wid).Scan(&version, &overrides)
	// Only RAG override names are returned; unrelated workspace settings are not
	// included in this small settings page or accidentally sent back on save.
	selected := []string{}
	for _, key := range overrides {
		if strings.HasPrefix(key, "rag_") {
			selected = append(selected, key)
		}
	}
	respond(w, map[string]any{"data": data, "version": version, "overrides": selected}, e)
}
func (s *Server) testRAGProvider(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if wid != "" && !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "워크스페이스 AI 설정 권한이 없습니다")
		return
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if e = validateRAGSettings(cfg); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	// This diagnostic sends fixed non-private text, never a random workspace page.
	vectors, e := ragEmbeddings(r.Context(), ragEmbeddingProvider(cfg), []string{"madi 연결 진단: 지식 검색"})
	if e != nil {
		apiError(w, 502, e.Error())
		return
	}
	var vectorInstalled bool
	if e = s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='vector')").Scan(&vectorInstalled); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RAG_CONNECTION_TEST", wid, map[string]any{"dimensions": len(vectors[0])})
	respond(w, map[string]any{"ok": true, "dimensions": len(vectors[0]), "pgvector_installed": vectorInstalled, "content_sent": "고정된 연결 진단 문구만 전송"}, nil)
}
