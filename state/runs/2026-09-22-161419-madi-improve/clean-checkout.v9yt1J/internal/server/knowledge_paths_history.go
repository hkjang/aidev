package server

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) registerKnowledgePaths() {
	s.RegisterApprovalAdapter("learning_step", s.learningApprovalAdapter())
	s.handle("GET /api/v1/knowledge-paths", s.listKnowledgePaths)
	s.handle("POST /api/v1/knowledge-paths", s.saveKnowledgePath)
	s.handle("GET /api/v1/knowledge-paths/{id}", s.getKnowledgePath)
	s.handle("PUT /api/v1/knowledge-paths/{id}", s.saveKnowledgePath)
	s.handle("GET /api/v1/knowledge-paths/{id}/history", s.getLearningHistory)
	s.handle("POST /api/v1/knowledge-paths/{id}/steps/{step}/confirm", s.confirmLearningStep)
	s.handle("GET /api/v1/learning-progress/{id}/review", s.getLearningReview)
}
func (s *Server) getLearningHistory(w http.ResponseWriter, r *http.Request) {
	if !learningCookie(r) {
		apiError(w, 403, "개인 실습 이력은 본인의 로그인 화면에서 확인하세요")
		return
	}
	tx, e := s.learningTx(r, true)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	path, e := knowledgePathReadTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT e.id::text,coalesce(e.step_id::text,''),e.action,e.metadata,e.proof,e.created_at,
 coalesce(madi_document_allowed($2,st.document_id,false) AND (st.kind<>'review' OR (a.id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset(coalesce(a.snapshot->'preceding_records','[]'::jsonb)) ref(document_id uuid) WHERE NOT madi_document_allowed($2,ref.document_id,false)))),false)
 FROM knowledge_path_events e LEFT JOIN knowledge_path_steps st ON st.id=e.step_id
 LEFT JOIN approval_requests a ON a.id=NULLIF(e.metadata->>'approval_id','')::uuid
 WHERE e.path_id=$1 AND e.user_id=$2 ORDER BY e.created_at DESC,e.id DESC LIMIT 201`, path.ID, current(r).ID)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	out := []map[string]any{}
	for rows.Next() {
		var id, step, action, cipher string
		var raw []byte
		var at time.Time
		var allowed bool
		if e = rows.Scan(&id, &step, &action, &raw, &cipher, &at, &allowed); e != nil {
			break
		}
		item := map[string]any{"id": id, "action": action, "created_at": at, "available": allowed}
		if allowed {
			var metadata any
			if e = json.Unmarshal(raw, &metadata); e != nil {
				break
			}
			item["step_id"], item["metadata"] = step, metadata
			if cipher != "" {
				plain, err := s.decrypt(cipher)
				if err != nil {
					e = err
					break
				}
				item["proof"] = plain
			}
		}
		out = append(out, item)
	}
	rows.Close()
	if e == nil {
		e = rows.Err()
	}
	// Do protection reads after closing rows, never nest queries on busy pgx.
	if e == nil {
		for _, item := range out {
			if plain, ok := item["proof"].(string); ok {
				if err := s.checkEvidenceProtection(r.Context(), tx, current(r), path.WorkspaceID, map[string]any{"proof": plain}); err != nil {
					delete(item, "proof")
					item["proof_hidden"] = true
				}
			}
		}
	}
	more := len(out) > 200
	if more {
		out = out[:200]
	}
	if e == nil {
		e = s.learningActorTx(r, tx, path.WorkspaceID, "document:read")
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		learningRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"items": out, "has_more": more, "notice": "최근 200개 개인 이력입니다. 이전 기록은 현재 완료가 아니며, 근거 접근 권한을 잃으면 실습 원문을 표시하지 않습니다."})
}

func knowledgePathMCPTools() []mcpTool {
	items := []mcpTool{
		{Name: "list_knowledge_paths", Description: "현재 접근 가능한 역할별 지식 경로의 목록만 조회합니다. 역할 이름은 권한이 아니며 완료나 실행을 수행하지 않습니다.", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"workspace_id": map[string]any{"type": "string"}}, "required": []string{"workspace_id"}, "additionalProperties": false}},
		{Name: "get_knowledge_path", Description: "경로의 순서·현재 접근 가능한 문서 참조·개인 진행 메타데이터를 조회합니다. 실습 원문을 반환하거나 단계를 완료하지 않습니다.", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"path_id": map[string]any{"type": "string"}}, "required": []string{"path_id"}, "additionalProperties": false}},
	}
	for i := range items {
		items[i].Scope = "document:read"
		items[i].Annotations = map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false}
	}
	return items
}
