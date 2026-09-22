package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) registerSearchOperations() {
	s.handle("GET /api/v1/workspaces/{id}/search-dictionary", s.getSearchDictionary)
	s.handle("PUT /api/v1/workspaces/{id}/search-dictionary", s.putSearchDictionary)
	s.handle("POST /api/v1/workspaces/{id}/search-diagnostics", s.searchDiagnostics)
	s.handle("GET /api/v1/workspaces/{id}/search-evaluation/sample", s.searchEvaluationFixture)
	s.handle("POST /api/v1/workspaces/{id}/search-evaluation", s.evaluateSearch)
}
func (s *Server) getSearchDictionary(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "조직 용어 사전 관리 권한이 없습니다")
		return
	}
	revision, entries, e := searchDictionary(r.Context(), s.DB, wid)
	respond(w, map[string]any{"revision": revision, "entries": entries, "scope": "workspace_shared", "maximum_entries": 300, "maximum_aliases": 12, "normalization": "Unicode NFKC · 대소문자 · 공백 · 조직 별칭"}, e)
}
func (s *Server) putSearchDictionary(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	p := current(r)
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "조직 용어 사전 관리 권한이 없습니다")
		return
	}
	var in struct {
		Revision      int                     `json:"revision"`
		Entries       []searchDictionaryEntry `json:"entries"`
		ConfirmShared bool                    `json:"confirm_shared"`
	}
	if decode(r, &in) != nil || in.Revision < 0 || !in.ConfirmShared {
		apiError(w, 400, "현재 사전 버전과 워크스페이스 구성원 공유 확인이 필요합니다")
		return
	}
	entries, e := normalizeSearchDictionary(in.Entries)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var actor string
	cookie, cookieError := r.Cookie("madi_session")
	if cookieError != nil {
		apiError(w, 403, "현재 로그인 세션을 확인하세요")
		return
	}
	e = tx.QueryRow(r.Context(), `SELECT u.id::text FROM users u JOIN workspace_members m ON m.user_id=u.id JOIN sessions ss ON ss.user_id=u.id AND ss.token_hash=$3 AND ss.expires_at>now() WHERE u.id=$1 AND NOT u.disabled AND u.kind='user' AND u.role<>'viewer' AND m.workspace_id=$2 AND m.role IN ('owner','admin') FOR SHARE OF u,m,ss`, p.ID, wid, digest(cookie.Value)).Scan(&actor)
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			apiError(w, 403, "현재 조직 용어 사전 관리 권한이 없습니다")
		} else {
			respond(w, nil, e)
		}
		return
	}
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended('search-dictionary:'||$1,0))`, wid); e != nil {
		respond(w, nil, e)
		return
	}
	revision, _, e := searchDictionary(r.Context(), tx, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if revision != in.Revision {
		apiError(w, 409, "다른 관리자가 사전을 변경했습니다. 다시 불러온 뒤 확인하세요")
		return
	}
	protection, e := s.ProtectDocumentMetadataTx(r.Context(), tx, p, "", wid, entries)
	if e != nil {
		var denied ProtectionError
		if errors.As(e, &denied) {
			jsonResponse(w, 422, map[string]any{"error": "조직 용어에 정보 보호 정책 확인이 필요합니다", "protection": denied})
			return
		}
		respond(w, nil, e)
		return
	}
	if protection.Changed {
		raw, _ := json.Marshal(protection.Value)
		if e = json.Unmarshal(raw, &entries); e != nil {
			respond(w, nil, e)
			return
		}
		entries, e = normalizeSearchDictionary(entries)
		if e != nil {
			apiError(w, 422, "정보 보호 정제 후 용어가 중복되거나 유효하지 않습니다. 민감한 값을 제외하고 다시 저장하세요")
			return
		}
	}
	if _, e = tx.Exec(r.Context(), `INSERT INTO workspace_search_dictionary(workspace_id,revision,entries,updated_by) VALUES($1,1,$2,$3) ON CONFLICT(workspace_id) DO UPDATE SET revision=workspace_search_dictionary.revision+1,entries=excluded.entries,updated_by=excluded.updated_by,updated_at=now()`, wid, jsonValue(entries), p.ID); e != nil {
		respond(w, nil, e)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "SEARCH_DICTIONARY_UPDATE", wid, map[string]any{"revision": revision + 1, "entries": len(entries)})
	respond(w, map[string]any{"revision": revision + 1, "entries": entries, "scope": "workspace_shared", "protection_changed": protection.Changed}, nil)
}
