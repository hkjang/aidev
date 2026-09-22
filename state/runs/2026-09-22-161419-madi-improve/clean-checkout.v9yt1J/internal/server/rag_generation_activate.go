package server

import (
	"encoding/json"
	"math"
	"net/http"
)

func (s *Server) activateRAGGeneration(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "색인 세대 전환 권한이 없습니다")
		return
	}
	var in struct {
		ValidationID    string `json:"validation_id"`
		StateRevision   int64  `json:"state_revision"`
		Mode            string `json:"mode"`
		Confirm         bool   `json:"confirm"`
		ConfirmCoverage bool   `json:"confirm_coverage"`
	}
	if decode(r, &in) != nil || !in.Confirm || !in.ConfirmCoverage || !validID(in.ValidationID) || in.StateRevision < 1 || !oneOf(in.Mode, "exact", "ann", "verify") {
		apiError(w, 400, "검증 영수증·현재 상태 버전·모드와 전환 후 검색 범위 변경 확인이 필요합니다")
		return
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if e = ragGenerationOperatorTx(ctx, tx, r, wid); e != nil {
		ragIndexError(w, e)
		return
	}
	var generation, sessionHash, fingerprint string
	var generationRevision, stateRevision int64
	var cohortRaw, reportRaw []byte
	var valid bool
	e = tx.QueryRow(ctx, `SELECT generation_id::text,session_hash,generation_revision,state_revision,settings_fingerprint,cohort,report,expires_at>now() AND consumed_at IS NULL FROM rag_generation_validations WHERE id=$1 AND workspace_id=$2 AND actor_id=$3 FOR UPDATE`, in.ValidationID, wid, current(r).ID).Scan(&generation, &sessionHash, &generationRevision, &stateRevision, &fingerprint, &cohortRaw, &reportRaw, &valid)
	cookie, _ := r.Cookie("madi_session")
	if e != nil || !valid || stateRevision != in.StateRevision || cookie == nil || sessionHash != digest(cookie.Value) {
		ragIndexError(w, errRAGChanged)
		return
	}
	cohort, report, e := ragDecodeValidation(cohortRaw, reportRaw)
	if e != nil || report.Mode != in.Mode || report.Truncated || math.IsNaN(report.Recall) || math.IsInf(report.Recall, 0) || report.Recall < 0.9 || report.Recall > 1 || report.ExactCount < 1 || (in.Mode != "exact" && !report.PlannedIndexUsed) {
		apiError(w, 409, "현재 모드의 잘리지 않은 검증 결과와 Recall@k 0.9 이상이 필요합니다")
		return
	}
	g, e := s.ragGenerationTx(ctx, tx, generation, true)
	if e != nil || g.WorkspaceID != wid || g.Status == "disabled" || g.Revision != generationRevision {
		ragIndexError(w, errRAGChanged)
		return
	}
	if _, e = s.ragGenerationCohortTx(ctx, tx, r, g, cohort, true); e != nil {
		ragIndexError(w, e)
		return
	}
	cfg, e := s.ragSettingsTx(ctx, tx, wid)
	if e != nil || !boolean(cfg, "rag_enabled") || ragGenerationSettingsFingerprint(cfg) != fingerprint {
		ragIndexError(w, errRAGChanged)
		return
	}
	var previous string
	var revision int64
	if e = tx.QueryRow(ctx, `SELECT coalesce(active_id::text,''),revision FROM rag_generation_state WHERE workspace_id=$1 FOR UPDATE`, wid).Scan(&previous, &revision); e != nil || revision != in.StateRevision {
		ragIndexError(w, errRAGChanged)
		return
	}
	if in.Mode != "exact" {
		var ready bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_am am ON am.oid=c.relam WHERE c.relname=$1 AND i.indrelid='rag_vector_chunks'::regclass AND i.indisvalid AND i.indisready AND am.amname='hnsw')`, ragHNSWIndexName(g.ID)).Scan(&ready); e != nil || !ready {
			apiError(w, 409, "검증한 HNSW 인덱스가 더 이상 준비되어 있지 않습니다")
			return
		}
	}
	// Provider settings and the serving pointer change atomically. No request
	// observes a new provider paired with the old generation's document consent.
	if _, e = tx.Exec(ctx, `INSERT INTO workspace_settings(workspace_id) VALUES($1) ON CONFLICT DO NOTHING`, wid); e != nil {
		respond(w, nil, e)
		return
	}
	var raw []byte
	var settingsVersion int
	if e = tx.QueryRow(ctx, `SELECT data,version FROM workspace_settings WHERE workspace_id=$1 FOR UPDATE`, wid).Scan(&raw, &settingsVersion); e != nil {
		respond(w, nil, e)
		return
	}
	var stored map[string]any
	if e = json.Unmarshal(raw, &stored); e != nil {
		respond(w, nil, e)
		return
	}
	next := map[string]any{}
	for key, value := range cfg {
		next[key] = value
	}
	for _, key := range ragGenerationProviderKeys {
		value := g.Config[key]
		next[key] = value
		if isWorkspaceSecret(key) {
			value, e = s.encrypt(str(g.Config, key))
			if e != nil {
				respond(w, nil, e)
				return
			}
		}
		stored[key] = value
	}
	next["rag_vector_mode"] = in.Mode
	stored["rag_vector_mode"] = in.Mode
	if in.Mode != "exact" {
		next["rag_backend"] = "pgvector"
		stored["rag_backend"] = "pgvector"
	}
	if e = validateRAGSettings(next); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if ragProviderFingerprint(next) != g.Provider {
		ragIndexError(w, errRAGChanged)
		return
	}
	_, e = tx.Exec(ctx, `INSERT INTO workspace_settings_history(id,workspace_id,user_id,version,data) VALUES($1,$2,$3,$4,$5)`, newID(), wid, current(r).ID, settingsVersion, raw)
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE workspace_settings SET data=$2,version=version+1,updated_at=now() WHERE workspace_id=$1`, wid, jsonValue(stored))
	}
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE rag_generation_state SET active_id=$2,revision=revision+1,mode=$3,updated_at=now() WHERE workspace_id=$1 AND revision=$4`, wid, g.ID, in.Mode, in.StateRevision)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE rag_generations SET status=CASE WHEN id=$2 THEN 'active' ELSE 'retired' END,updated_at=now() WHERE workspace_id=$1 AND (id=$2 OR status='active')`, wid, g.ID)
	}
	if e == nil && previous != g.ID {
		_, e = tx.Exec(ctx, `UPDATE automation_jobs SET cancel_requested=true,updated_at=now() WHERE id IN (SELECT last_job_id FROM rag_index_grants WHERE generation_id=NULLIF($1,'')::uuid) AND status IN ('pending','running')`, previous)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE rag_generation_validations SET consumed_at=now() WHERE id=$1`, in.ValidationID)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RAG_GENERATION_ACTIVATE", g.ID, map[string]any{"previous_generation_id": previous, "state_revision": revision + 1, "mode": in.Mode, "validation_id": in.ValidationID, "validated_documents": len(cohort)})
	jsonResponse(w, 200, map[string]any{"active_id": g.ID, "state_revision": revision + 1, "mode": in.Mode, "previous_id": previous, "validated_documents": len(cohort), "warning": "전환 세대에서 현재 문서 버전·권한·색인 동의가 일치하는 자료만 검색합니다. 이전 세대로 돌아갈 때도 새 검증과 명시 전환이 필요합니다."})
}
