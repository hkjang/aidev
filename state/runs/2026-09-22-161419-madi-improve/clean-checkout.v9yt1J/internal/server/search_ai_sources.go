package server

import (
	"errors"
	"net/http"
)

type selectedAIDocument struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// Explicit search-result selections do not expand to workspace/semantic search.
// Each selected current document contributes its best bounded raw fragment.
func (s *Server) selectedAISources(r *http.Request, p *Principal, wid, prompt string, selected []selectedAIDocument) ([]aiSource, ragDiagnostics, error) {
	diagnostic := ragDiagnostics{Mode: "selected", Backend: "none", Warnings: []string{"사용자가 선택한 각 문서의 질문 관련 원문 조각(문서당 최대 약 6KiB)만 참조합니다. 전체 문서나 전체 검색 결과를 요약한 것으로 보지 마세요."}}
	if len(selected) < 1 || len(selected) > 10 || !validID(wid) || !hasIntegrationScope(p, "document:read") || !s.canWorkspace(r.Context(), p, wid, false) {
		return nil, diagnostic, errors.New("현재 워크스페이스의 문서를 1~10개 선택하세요")
	}
	seen := map[string]bool{}
	out := []aiSource{}
	for _, item := range selected {
		if !validID(item.ID) || item.Version < 1 || seen[item.ID] {
			return nil, diagnostic, errors.New("선택 문서의 식별자·버전·중복을 확인하세요")
		}
		seen[item.ID] = true
		var ok bool
		if e := s.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM documents d WHERE d.id=$1 AND d.workspace_id=$2 AND d.version=$3 AND d.deleted_at IS NULL AND madi_document_allowed($4,d.id,false))`, item.ID, wid, item.Version, p.ID).Scan(&ok); e != nil || !ok {
			return nil, diagnostic, errors.New("선택한 문서의 권한 또는 버전이 변경되었습니다. 검색 결과를 새로고침하세요")
		}
		sources, e := s.keywordAISources(r, p, item.ID, wid, prompt)
		if e != nil {
			return nil, diagnostic, e
		}
		if len(sources) == 0 {
			return nil, diagnostic, errors.New("선택한 문서에 인용할 본문이 없습니다")
		}
		if sources[0].ID != item.ID || sources[0].Version != item.Version {
			return nil, diagnostic, errAIStreamChanged
		}
		out = append(out, sources[0])
	}
	diagnostic.Scanned = len(selected)
	return out, diagnostic, nil
}
