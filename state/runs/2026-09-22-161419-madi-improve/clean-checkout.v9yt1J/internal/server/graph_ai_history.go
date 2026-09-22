package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) expireGraphAI(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, `UPDATE graph_ai_runs SET status='cancelled',finished_at=now(),error='전경 분석 연결이 종료되거나 확인 시간이 만료되었습니다. 자동 재전송하지 않습니다.' WHERE status='running' AND (heartbeat_at<now()-interval '5 seconds' OR expires_at<=now())`)
	return e
}
func graphAIActions(ctx context.Context, q graphAIQuery, id string) ([]graphAIAction, error) {
	rows, e := q.Query(ctx, `SELECT id::text,run_id::text,kind,payload,action_hash,status,result FROM graph_ai_actions WHERE run_id=$1 ORDER BY created_at,id LIMIT 30`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []graphAIAction{}
	for rows.Next() {
		var x graphAIAction
		var raw, receipt []byte
		if e = rows.Scan(&x.ID, &x.RunID, &x.Kind, &raw, &x.Hash, &x.Status, &receipt); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(raw, &x.Payload); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(receipt, &x.Result); e != nil {
			return nil, e
		}
		result = append(result, x)
	}
	return result, rows.Err()
}
func (s *Server) graphAIView(r *http.Request, id string) (map[string]any, error) {
	if !validID(id) || !hasIntegrationScope(current(r), "ai:execute") || !hasIntegrationScope(current(r), "document:read") {
		return nil, errGraphAIChanged
	}
	run, e := loadGraphAIRun(r.Context(), s.DB, id, false)
	if e != nil || run.OwnerID != current(r).ID || !s.canWorkspace(r.Context(), current(r), run.WorkspaceID, false) {
		return nil, errGraphAIChanged
	}
	sources, e := s.graphAIValidateSources(r.Context(), s.DB, current(r), run, "")
	if e != nil {
		return nil, e
	}
	actions, e := graphAIActions(r.Context(), s.DB, id)
	if e != nil {
		return nil, e
	}
	actor, actorErr := s.graphAIPrincipal(r.Context(), run)
	cfg, cfgErr := s.effectiveSettings(r.Context(), run.WorkspaceID)
	canApply := run.Status == "ready" && actorErr == nil && cfgErr == nil && aiHistoryProvider(cfg) == run.Provider && personalAIHistory(current(r)) && hasIntegrationScope(current(r), "document:write") && hasIntegrationScope(actor, "document:write") && s.canWorkspace(r.Context(), current(r), run.WorkspaceID, true) && s.canWorkspace(r.Context(), actor, run.WorkspaceID, true) && (run.SessionHash == "" || run.SessionHash == graphAICookie(r)) && time.Since(run.CreatedAt) < 24*time.Hour
	// Results are ordinary private documents after confirmation. Their current
	// ACL still applies if the original owner subsequently shares/transfers them.
	for i := range actions {
		if resultID := str(actions[i].Result, "document_id"); resultID != "" && !s.canDocument(r.Context(), current(r), resultID, false) {
			actions[i].Result = map[string]any{"access_revoked": true}
		}
	}
	if !s.canFeature(r.Context(), current(r), run.WorkspaceID, "ai-graph") {
		actions = []graphAIAction{}
		canApply = false
	}
	return map[string]any{"id": run.ID, "workspace_id": run.WorkspaceID, "status": run.Status, "error": run.Error, "created_at": run.CreatedAt, "kinds": run.Kinds, "actions": actions, "sources": sources, "can_apply": canApply, "automatic_apply": false, "scope": "provided_document_ranges_only", "generation_deadline": run.ExpiresAt}, nil
}
func (s *Server) getGraphAIRun(w http.ResponseWriter, r *http.Request) {
	if e := s.expireGraphAI(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	v, e := s.graphAIView(r, r.PathValue("id"))
	if e != nil {
		apiError(w, 404, "현재 본인의 권한으로 분석 기록을 열 수 없습니다")
		return
	}
	jsonResponse(w, 200, v)
}
func (s *Server) listGraphAIRuns(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !hasIntegrationScope(current(r), "ai:execute") || !hasIntegrationScope(current(r), "document:read") || !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "현재 워크스페이스의 AI 분석 조회 권한이 없습니다")
		return
	}
	if e := s.expireGraphAI(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',g.id,'status',g.status,'error',g.error,'created_at',g.created_at,'kinds',g.kinds) FROM graph_ai_runs g WHERE g.workspace_id=$1 AND g.owner_id=$2 AND EXISTS(SELECT 1 FROM graph_ai_sources x WHERE x.run_id=g.id) AND NOT EXISTS(SELECT 1 FROM graph_ai_sources x LEFT JOIN documents d ON d.id=x.document_id WHERE x.run_id=g.id AND (d.id IS NULL OR d.deleted_at IS NOT NULL OR d.version<>x.document_version OR NOT madi_document_allowed($2,d.id,false))) ORDER BY g.created_at DESC LIMIT 50`, wid, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, rows)
}
func (s *Server) cancelGraphAIRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) || !hasIntegrationScope(current(r), "ai:execute") {
		apiError(w, 403, "본인의 분석만 취소할 수 있습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	run, e := loadGraphAIRun(r.Context(), tx, id, true)
	if e != nil || run.OwnerID != current(r).ID || (current(r).WorkspaceID != "" && current(r).WorkspaceID != run.WorkspaceID) {
		apiError(w, 404, "본인의 분석을 찾을 수 없습니다")
		return
	}
	if oneOf(run.Status, "running", "ready") {
		_, e = tx.Exec(r.Context(), `UPDATE graph_ai_runs SET status='cancelled',error='사용자가 분석과 남은 후보를 취소했습니다.',finished_at=now() WHERE id=$1`, id)
		if e == nil {
			_, e = tx.Exec(r.Context(), `UPDATE graph_ai_actions SET status='cancelled',decided_at=now() WHERE run_id=$1 AND status='proposed'`, id)
		}
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"cancelled": true, "applied_actions_unchanged": true}, e)
}
func (s *Server) deleteGraphAIRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) || !hasIntegrationScope(current(r), "ai:execute") {
		apiError(w, 403, "본인의 분석 기록만 삭제할 수 있습니다")
		return
	}
	tag, e := s.DB.Exec(r.Context(), `DELETE FROM graph_ai_runs WHERE id=$1 AND owner_id=$2 AND ($3='' OR workspace_id::text=$3)`, id, current(r).ID, current(r).WorkspaceID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 404, "본인의 분석을 찾을 수 없습니다")
		return
	}
	s.audit(r, "GRAPH_AI_HISTORY_DELETE", id, map[string]any{"applied_resources_unchanged": true})
	jsonResponse(w, 200, map[string]any{"deleted": true, "applied_resources_unchanged": true})
}
