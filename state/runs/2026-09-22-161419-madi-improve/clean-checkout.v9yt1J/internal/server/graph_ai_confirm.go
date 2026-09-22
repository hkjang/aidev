package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

var errGraphAIAlreadyApplied = errors.New("AI 후보가 이미 적용되었습니다")

func loadGraphAIAction(ctx context.Context, q collaborationQuery, id string, lock bool) (graphAIAction, error) {
	var action graphAIAction
	var raw, result []byte
	query := `SELECT id::text,run_id::text,kind,payload,action_hash,status,result FROM graph_ai_actions WHERE id=$1`
	if lock {
		query += " FOR UPDATE"
	}
	e := q.QueryRow(ctx, query, id).Scan(&action.ID, &action.RunID, &action.Kind, &raw, &action.Hash, &action.Status, &result)
	if e == nil {
		e = json.Unmarshal(raw, &action.Payload)
	}
	if e == nil {
		e = json.Unmarshal(result, &action.Result)
	}
	return action, e
}
func (s *Server) graphAIWriteAuthorityTx(ctx context.Context, tx pgx.Tx, p *Principal, run graphAIRun) error {
	if !hasIntegrationScope(p, "document:write") {
		return errGraphAIChanged
	}
	var allowed bool
	if tx.QueryRow(ctx, `SELECT NOT u.disabled AND u.role<>'viewer' AND m.role IN('owner','admin','editor') FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$2 WHERE u.id=$1 FOR SHARE OF u,m`, p.ID, run.WorkspaceID).Scan(&allowed) != nil || !allowed {
		return errGraphAIChanged
	}
	if p.TokenID != "" {
		var scopes []string
		if tx.QueryRow(ctx, `SELECT scopes FROM api_keys WHERE id=$1 AND user_id=$2 AND workspace_id=$3 AND revoked_at IS NULL AND expires_at>now() FOR SHARE`, p.TokenID, p.ID, run.WorkspaceID).Scan(&scopes) != nil || !slices.Contains(scopes, "document:write") {
			return errGraphAIChanged
		}
		cfg, e := s.ragSettingsTx(ctx, tx, run.WorkspaceID)
		if e != nil || !slices.Contains(settingStrings(cfg, "allowed_key_scopes", keyScopes), "document:write") {
			return errGraphAIChanged
		}
	}
	if p.PluginID != "" {
		if tx.QueryRow(ctx, `SELECT (x.manifest->'capabilities') ? 'document:write' AND w.capabilities ? 'document:write' FROM plugins x JOIN workspace_plugins w ON w.plugin_id=x.id WHERE x.id=$1 AND w.workspace_id=$2 AND x.enabled AND w.enabled FOR SHARE OF x,w`, p.PluginID, run.WorkspaceID).Scan(&allowed) != nil || !allowed {
			return errGraphAIChanged
		}
	}
	return nil
}
func (s *Server) graphAIConfirmTx(ctx context.Context, tx pgx.Tx, r *http.Request, expected graphAIAction, p *Principal) (graphAIRun, graphAIAction, error) {
	run, e := loadGraphAIRun(ctx, tx, expected.RunID, true)
	if e != nil || run.OwnerID != current(r).ID || run.Status != "ready" || time.Since(run.CreatedAt) > 24*time.Hour || (run.SessionHash != "" && run.SessionHash != graphAICookie(r)) {
		return run, expected, errGraphAIChanged
	}
	action, e := loadGraphAIAction(ctx, tx, expected.ID, true)
	if e != nil || action.RunID != run.ID || action.Hash != expected.Hash || digest(string(jsonValue(action.Payload))) != action.Hash {
		return run, action, errGraphAIChanged
	}
	if action.Status == "applied" {
		return run, action, errGraphAIAlreadyApplied
	}
	if action.Status != "proposed" {
		return run, action, errGraphAIChanged
	}
	if e = s.graphAIWriteAuthorityTx(ctx, tx, p, run); e != nil {
		return run, action, e
	}
	writeID := action.Payload.SourceID
	if e = s.graphAIValidateTx(ctx, tx, run, p, writeID); e != nil {
		return run, action, e
	}
	// The human's current cookie is independent of the original token/plug-in
	// authority and must remain live until this exact mutation commits.
	var active bool
	if tx.QueryRow(ctx, `SELECT true FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now() FOR SHARE`, graphAICookie(r), p.ID).Scan(&active) != nil {
		return run, action, errGraphAIChanged
	}
	return run, action, nil
}
func graphAIRecordAction(ctx context.Context, tx pgx.Tx, p *Principal, action graphAIAction, result map[string]any) error {
	tag, e := tx.Exec(ctx, `UPDATE graph_ai_actions SET status='applied',result=$3,decided_at=now() WHERE id=$1 AND action_hash=$2 AND status='proposed'`, action.ID, action.Hash, jsonValue(result))
	if e != nil {
		return e
	}
	if tag.RowsAffected() != 1 {
		return errGraphAIChanged
	}
	_, e = tx.Exec(ctx, `INSERT INTO audit_logs(id,user_id,action,resource,details) VALUES($1,$2,'GRAPH_AI_ACTION_APPLY',$3,$4)`, newID(), p.ID, action.ID, jsonValue(map[string]any{"run_id": action.RunID, "kind": action.Kind, "action_hash": action.Hash, "result": result, "automatic_apply": false}))
	return e
}
func (s *Server) confirmGraphAIAction(w http.ResponseWriter, r *http.Request) {
	if !personalAIHistory(current(r)) || current(r).Kind != "user" {
		apiError(w, 403, "실제 사용자 본인의 브라우저 로그인에서 후보 한 건을 확인하세요")
		return
	}
	var in struct {
		Hash    string `json:"action_hash"`
		Confirm bool   `json:"confirm"`
		Reject  bool   `json:"reject"`
	}
	if decode(r, &in) != nil || in.Confirm == in.Reject || len(in.Hash) != 64 || !validID(r.PathValue("id")) {
		apiError(w, 400, "고정된 후보 hash와 확인 또는 거절 중 하나가 필요합니다")
		return
	}
	action, e := loadGraphAIAction(r.Context(), s.DB, r.PathValue("id"), false)
	if e != nil {
		apiError(w, 404, "후보를 찾을 수 없습니다")
		return
	}
	run, e := loadGraphAIRun(r.Context(), s.DB, action.RunID, false)
	if e != nil || run.OwnerID != current(r).ID || in.Hash != action.Hash || (run.SessionHash != "" && run.SessionHash != graphAICookie(r)) {
		apiError(w, 404, "현재 본인의 로그인에서 확인할 후보가 아닙니다")
		return
	}
	if in.Reject {
		tag, e := s.DB.Exec(r.Context(), `UPDATE graph_ai_actions SET status='rejected',decided_at=now() WHERE id=$1 AND action_hash=$2 AND status='proposed'`, action.ID, action.Hash)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if tag.RowsAffected() == 0 && action.Status != "rejected" {
			apiError(w, 409, "이미 처리되거나 취소된 후보입니다")
			return
		}
		s.audit(r, "GRAPH_AI_ACTION_REJECT", action.ID, map[string]any{"run_id": run.ID})
		jsonResponse(w, 200, map[string]any{"id": action.ID, "status": "rejected"})
		return
	}
	p, e := s.graphAIPrincipal(r.Context(), run)
	if e != nil || !hasIntegrationScope(current(r), "document:write") || !s.canFeature(r.Context(), current(r), run.WorkspaceID, "ai-graph") {
		apiError(w, 403, errGraphAIChanged.Error())
		return
	}
	if _, e = s.graphAIValidateSources(r.Context(), s.DB, current(r), run, action.Payload.SourceID); e != nil {
		graphAIError(w, e)
		return
	}
	if action.Status == "applied" {
		if resultID := str(action.Result, "document_id"); resultID != "" && !s.canDocument(r.Context(), current(r), resultID, false) {
			apiError(w, 403, "적용 결과 문서의 현재 접근 권한이 없습니다")
			return
		}
		jsonResponse(w, 200, map[string]any{"id": action.ID, "status": "applied", "result": action.Result, "replayed": true})
		return
	}
	if action.Kind == "entity" || action.Kind == "gap" {
		s.graphAICreateFromCandidate(w, r, run, action, p)
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	protected, e := s.graphAIProtectCandidateTx(r.Context(), tx, p, run.WorkspaceID, action.Payload)
	if e != nil {
		graphAIError(w, e)
		return
	}
	if digest(string(jsonValue(protected))) != action.Hash {
		graphAIError(w, errGraphAIChanged)
		return
	}
	_, locked, e := s.graphAIConfirmTx(r.Context(), tx, r, action, p)
	if errors.Is(e, errGraphAIAlreadyApplied) {
		tx.Rollback(r.Context())
		jsonResponse(w, 200, map[string]any{"id": locked.ID, "status": "applied", "result": locked.Result, "replayed": true})
		return
	}
	if e != nil {
		graphAIError(w, e)
		return
	}
	result := map[string]any{"document_id": action.Payload.SourceID, "kind": action.Kind}
	if oneOf(action.Kind, "relation", "duplicate") {
		_, e = tx.Exec(r.Context(), `INSERT INTO document_relations(source_id,target_id,type,created_by) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, action.Payload.SourceID, action.Payload.TargetID, action.Payload.RelationType, p.ID)
		result["target_id"] = action.Payload.TargetID
		result["relation_type"] = action.Payload.RelationType
	} else if action.Kind == "topic" {
		// Only self-document inputs may become durable visible topics. A current
		// audience intersection would not protect against a future share change.
		var sourceCount int
		e = tx.QueryRow(r.Context(), `SELECT count(*) FROM graph_ai_sources WHERE run_id=$1`, run.ID).Scan(&sourceCount)
		if e != nil {
			graphAIError(w, e)
			return
		}
		if sourceCount != 1 {
			apiError(w, 409, "여러 입력 자료에서 추론한 주제는 이 문서에 저장할 수 없습니다. 해당 문서 하나만 선택하여 다시 분석하세요")
			return
		}
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_document_meta(document_id,classification,system_metadata) SELECT d.id,coalesce(sp.classification,'internal'),jsonb_build_object('ai_topics',jsonb_build_array($2::text)) FROM documents d LEFT JOIN spaces sp ON sp.id=d.space_id WHERE d.id=$1 ON CONFLICT(document_id) DO UPDATE SET system_metadata=jsonb_set(knowledge_document_meta.system_metadata,'{ai_topics}',(SELECT jsonb_agg(DISTINCT value) FROM jsonb_array_elements(CASE WHEN jsonb_typeof(knowledge_document_meta.system_metadata->'ai_topics')='array' THEN knowledge_document_meta.system_metadata->'ai_topics' ELSE '[]'::jsonb END || jsonb_build_array($2::text))),true),updated_at=now()`, action.Payload.SourceID, action.Payload.Topic)
		result["topic"] = action.Payload.Topic
	} else {
		graphAIError(w, errGraphAIChanged)
		return
	}
	if e == nil {
		minimal := map[string]any{"source_id": action.Payload.SourceID}
		if action.Kind == "topic" {
			minimal["topic"] = action.Payload.Topic
		} else {
			minimal["target_id"] = action.Payload.TargetID
			minimal["relation_type"] = action.Payload.RelationType
		}
		_, e = tx.Exec(r.Context(), `INSERT INTO graph_ai_annotations(id,document_id,action_id,kind,value,created_by) VALUES($1,$2,$3,$4,$5,$6)`, newID(), action.Payload.SourceID, action.ID, action.Kind, jsonValue(minimal), p.ID)
	}
	if e == nil {
		e = graphAIRecordAction(r.Context(), tx, p, action, result)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		graphAIError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"id": action.ID, "status": "applied", "result": result})
}
func (s *Server) graphAICreateFromCandidate(w http.ResponseWriter, r *http.Request, run graphAIRun, action graphAIAction, p *Principal) {
	ctx := withMutationResultGuard(r.Context(), func(ctx context.Context, tx pgx.Tx, created map[string]any) error {
		protected, e := s.graphAIProtectCandidateTx(ctx, tx, p, run.WorkspaceID, action.Payload)
		if e != nil {
			return e
		}
		if digest(string(jsonValue(protected))) != action.Hash {
			return errGraphAIChanged
		}
		_, locked, e := s.graphAIConfirmTx(ctx, tx, r, action, p)
		if e != nil {
			return e
		}
		id := str(created, "id")
		var private bool
		if tx.QueryRow(ctx, `SELECT visibility='private' AND owner_id=$2 AND workspace_id=$3 FROM documents WHERE id=$1`, id, p.ID, run.WorkspaceID).Scan(&private) != nil || !private {
			return errGraphAIChanged
		}
		for _, proof := range action.Payload.Evidence {
			_, e = tx.Exec(ctx, `INSERT INTO document_relations(source_id,target_id,type,created_by) VALUES($1,$2,'related',$3) ON CONFLICT DO NOTHING`, id, proof.DocumentID, p.ID)
			if e != nil {
				return e
			}
		}
		if action.Kind == "entity" {
			_, e = tx.Exec(ctx, `UPDATE knowledge_document_meta SET system_metadata=jsonb_set(system_metadata,'{entity_source}','"ai_confirmed"'::jsonb,true) WHERE document_id=$1`, id)
			if e != nil {
				return e
			}
		}
		var classificationRank int
		if e = tx.QueryRow(ctx, `SELECT coalesce(max(CASE coalesce(m.classification,sp.classification,'internal') WHEN 'restricted' THEN 3 WHEN 'confidential' THEN 2 WHEN 'internal' THEN 1 ELSE 0 END),1) FROM graph_ai_sources x JOIN documents d ON d.id=x.document_id LEFT JOIN knowledge_document_meta m ON m.document_id=d.id LEFT JOIN spaces sp ON sp.id=d.space_id WHERE x.run_id=$1`, run.ID).Scan(&classificationRank); e != nil {
			return e
		}
		classification := []string{"public", "internal", "confidential", "restricted"}[classificationRank]
		if _, e = tx.Exec(ctx, `INSERT INTO knowledge_document_meta(document_id,classification) VALUES($1,$2) ON CONFLICT(document_id) DO UPDATE SET classification=EXCLUDED.classification,updated_at=now()`, id, classification); e != nil {
			return e
		}
		return graphAIRecordAction(ctx, tx, p, locked, map[string]any{"document_id": id, "version": 1, "visibility": "private", "classification": classification, "kind": action.Kind})
	})
	body := map[string]any{"workspace_id": run.WorkspaceID, "title": action.Payload.Title, "visibility": "private"}
	if action.Kind == "entity" {
		body["description"] = action.Payload.Description
		body["entity_type"] = action.Payload.EntityType
	} else {
		body["markdown"] = "# " + action.Payload.Title + "\n\n" + action.Payload.Description
	}
	request := automationRequest(ctx, p, "POST", body)
	recorder := httptest.NewRecorder()
	if action.Kind == "entity" {
		s.createEnterpriseEntity(recorder, request)
	} else {
		s.createDocument(recorder, request)
	}
	// A concurrent second tab may have created a tentative row before its final
	// guard sees the first receipt. Its entire REST transaction is rolled back;
	// returning the committed receipt does not execute another mutation.
	currentAction, e := loadGraphAIAction(r.Context(), s.DB, action.ID, false)
	if e == nil && currentAction.Status == "applied" && currentAction.Hash == action.Hash {
		if id := str(currentAction.Result, "document_id"); !s.canDocument(r.Context(), current(r), id, false) {
			apiError(w, 403, "결과 문서에 현재 접근할 수 없습니다")
			return
		}
		jsonResponse(w, 200, map[string]any{"id": action.ID, "status": "applied", "result": currentAction.Result, "replayed": recorder.Code >= 300})
		return
	}
	if recorder.Code >= 300 {
		if recorder.Code == 422 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(422)
			_, _ = w.Write(recorder.Body.Bytes())
			return
		}
		graphAIError(w, errGraphAIChanged)
		return
	}
	graphAIError(w, errGraphAIChanged)
}
