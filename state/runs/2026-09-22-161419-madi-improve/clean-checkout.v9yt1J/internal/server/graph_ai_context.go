package server

import (
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

func graphAISelected(raw string) ([]string, error) {
	ids := strings.Split(raw, ",")
	if len(ids) < 1 || len(ids) > 12 {
		return nil, errors.New("분석 문서는 1~12개를 선택하세요")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !validID(id) || seen[id] {
			return nil, errors.New("중복되지 않은 실제 문서를 선택하세요")
		}
		seen[id] = true
	}
	sort.Strings(ids)
	return ids, nil
}
func (s *Server) graphAIContext(r *http.Request, wid string, ids []string) ([]aiSource, []graphAISource, []map[string]any, error) {
	p := current(r)
	if !hasIntegrationScope(p, "document:read") || !hasIntegrationScope(p, "ai:execute") || !s.canWorkspace(r.Context(), p, wid, false) {
		return nil, nil, nil, errGraphAIChanged
	}
	sources, snapshots, metadata := []aiSource{}, []graphAISource{}, []map[string]any{}
	scannedBytes := 0
	for _, id := range ids {
		var title, markdown, visibility, classification string
		var version int
		e := s.DB.QueryRow(r.Context(), `SELECT d.title,d.markdown,d.version,d.visibility,coalesce(m.classification,sp.classification,'internal') FROM documents d LEFT JOIN knowledge_document_meta m ON m.document_id=d.id LEFT JOIN spaces sp ON sp.id=d.space_id WHERE d.id=$1 AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)`, id, wid, p.ID).Scan(&title, &markdown, &version, &visibility, &classification)
		if e != nil || strings.TrimSpace(markdown) == "" || !utf8.ValidString(markdown) {
			return nil, nil, nil, errors.New("모든 선택 문서에 현재 접근 가능하고 분석할 원문이 있어야 합니다")
		}
		scannedBytes += len(markdown)
		if scannedBytes > 16<<20 {
			return nil, nil, nil, errors.New("선택 문서의 전체 원문 합계는 16MiB 이하여야 합니다")
		}
		end := min(len(markdown), (48<<10)/len(ids))
		for end > 0 && !utf8.ValidString(markdown[:end]) {
			end--
		}
		body := strings.Clone(markdown[:end])
		chunk := ragChunk{Start: 0, End: end, Content: body, Hash: digest(body), StartLine: 1, EndLine: 1 + strings.Count(body, "\n")}
		sources = append(sources, sourceFromChunk(id, title, version, chunk))
		snapshots = append(snapshots, graphAISource{DocumentID: id, Version: version, DocumentHash: graphAICanonicalHash(title, markdown), Start: 0, End: end, ContentHash: chunk.Hash})
		metadata = append(metadata, map[string]any{"id": id, "title": title, "version": version, "visibility": visibility, "classification": classification, "can_write": hasIntegrationScope(p, "document:write") && s.canDocument(r.Context(), p, id, true), "truncated": end < len(markdown), "sent_bytes": end, "total_bytes": len(markdown)})
	}
	return sources, snapshots, metadata, nil
}
func (s *Server) getGraphAIContext(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	ids, e := graphAISelected(r.URL.Query().Get("document_ids"))
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	sources, snapshots, metadata, e := s.graphAIContext(r, wid, ids)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	_, endpointError := aiEndpoint(str(cfg, "ai_base_url"))
	configured := boolean(cfg, "ai_enabled") && str(cfg, "ai_model") != "" && endpointError == nil
	jsonResponse(w, 200, map[string]any{"sources": sources, "snapshots": snapshots, "documents": metadata, "provider": map[string]any{"base_url": ragDisplayURL(str(cfg, "ai_base_url")), "model": str(cfg, "ai_model"), "fingerprint": aiHistoryProvider(cfg), "configured": configured}, "enabled": s.canFeature(r.Context(), current(r), wid, "ai-graph"), "kinds": graphAIKinds, "max_documents": 12, "max_reference_bytes": 48 << 10, "automatic_apply": false, "notice": "선택한 현재 문서의 명시된 구간만 전송합니다. 제공 자료에 한정한 관계·유사성·주제·엔터티·지식 공백 후보이며 전체 워크스페이스 분석이 아닙니다. 후보를 한 건씩 확인하기 전에는 원본을 변경하지 않습니다."})
}
func graphAIValidKinds(kinds []string) bool {
	if len(kinds) < 1 || len(kinds) > len(graphAIKinds) {
		return false
	}
	seen := map[string]bool{}
	for _, kind := range kinds {
		if !slices.Contains(graphAIKinds, kind) || seen[kind] {
			return false
		}
		seen[kind] = true
	}
	return true
}
