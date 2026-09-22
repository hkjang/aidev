package server

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) getRAGIndex(w http.ResponseWriter, r *http.Request) {
	p, id := current(r), r.PathValue("id")
	if !s.canDocument(r.Context(), p, id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	var wid, title, visibility string
	var version int
	var write bool
	e := s.DB.QueryRow(r.Context(), `SELECT workspace_id::text,title,visibility,version,madi_document_allowed($2,id,true) FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,false)`, id, p.ID).Scan(&wid, &title, &visibility, &version, &write)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	var generation string
	if e == nil {
		generation, e = ragSelectedGeneration(r.Context(), s.DB, wid, r.URL.Query().Get("generation_id"))
	}
	if e == nil {
		cfg, e = s.ragGenerationConfig(r.Context(), s.DB, cfg, wid, generation)
	}
	if e != nil {
		ragIndexError(w, e)
		return
	}
	can := write && hasIntegrationScope(p, "ai:execute") && hasIntegrationScope(p, "document:write")
	out := map[string]any{"document_id": id, "document_version": version, "title": title, "visibility": visibility, "can_index": can, "enabled": boolean(cfg, "rag_enabled"), "provider": nil, "grant": nil, "index": nil, "notice": "색인은 문서 본문을 선택한 임베딩 공급자에 전송합니다. 비공개 문서도 예외가 아닙니다. 동의 철회는 madi의 파생 벡터와 후속 전송을 중단하며 이미 공급자에 전송된 내용은 회수할 수 없습니다."}
	out["generation_id"] = generation
	if can {
		out["provider"] = map[string]any{"configured": str(cfg, "rag_embedding_base_url") != "" && str(cfg, "rag_embedding_model") != "", "base_url": ragDisplayURL(str(cfg, "rag_embedding_base_url")), "model": str(cfg, "rag_embedding_model"), "fingerprint": ragProviderFingerprint(cfg), "rerank": map[string]any{"enabled": boolean(cfg, "rag_rerank_enabled"), "base_url": ragDisplayURL(str(cfg, "rag_rerank_base_url")), "model": str(cfg, "rag_rerank_model"), "fingerprint": ragRerankFingerprint(cfg)}}
	}
	g, e := ragGrantForGenerationTx(r.Context(), s.DB, id, generation, false)
	if errors.Is(e, pgx.ErrNoRows) {
		respond(w, out, nil)
		return
	}
	if e != nil {
		ragIndexError(w, e)
		return
	}
	changed := g.Provider != ragProviderFingerprint(cfg) || (g.Rerank != "" && g.Rerank != ragRerankFingerprint(cfg))
	if can {
		out["grant"] = map[string]any{"id": g.ID, "revision": g.Revision, "active": g.Active, "auto_reindex": g.Auto, "allow_rerank": g.Rerank != "", "provider_fingerprint": g.Provider, "rerank_fingerprint": g.Rerank, "requires_reconsent": changed, "actor_id": g.ActorID}
	}
	if g.JobID != "" {
		var status, indexStatus, errText string
		var total, indexed, dimensions, indexVersion int
		e = s.DB.QueryRow(r.Context(), `SELECT j.status,COALESCE(i.status,'building'),COALESCE(i.total_chunks,0),COALESCE(i.indexed_chunks,0),COALESCE(i.dimensions,0),COALESCE(i.document_version,$2),j.last_error FROM automation_jobs j LEFT JOIN rag_vector_indexes i ON i.id=j.id WHERE j.id=$1`, g.JobID, g.Version).Scan(&status, &indexStatus, &total, &indexed, &dimensions, &indexVersion, &errText)
		if e == nil {
			state := map[string]any{"status": status, "index_status": indexStatus, "document_version": indexVersion, "total_chunks": total, "indexed_chunks": indexed, "dimensions": dimensions, "can_retry": can && oneOf(status, "failed", "cancelled")}
			if can {
				state["job_id"] = g.JobID
				if errText != "" {
					state["error"] = "색인 작업이 완료되지 않았습니다. 현재 권한·공급자 설정을 확인한 뒤 다시 동의하여 요청하세요."
				}
			}
			out["index"] = state
		} else if !errors.Is(e, pgx.ErrNoRows) {
			ragIndexError(w, e)
			return
		}
	}
	respond(w, out, nil)
}

func (s *Server) createRAGIndex(w http.ResponseWriter, r *http.Request) {
	p, id := current(r), r.PathValue("id")
	if !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "ai:execute") || !s.canDocument(r.Context(), p, id, true) {
		apiError(w, 403, "문서 작성 및 AI 실행 권한이 모두 필요합니다")
		return
	}
	var in struct {
		Version     int    `json:"expected_version"`
		Provider    string `json:"provider_fingerprint"`
		Consent     bool   `json:"consent"`
		Auto        bool   `json:"auto_reindex"`
		AllowRerank bool   `json:"allow_rerank"`
		Rerank      string `json:"rerank_fingerprint"`
	}
	if decode(r, &in) != nil || !in.Consent || in.Version < 1 || len(in.Provider) != 64 || (in.AllowRerank && len(in.Rerank) != 64) {
		apiError(w, 400, "현재 문서 버전과 공급자를 확인하고 내용 전송에 명시적으로 동의하세요")
		return
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	defer tx.Rollback(ctx)
	var wid, markdown string
	var version int
	e = tx.QueryRow(ctx, `SELECT workspace_id::text,markdown,version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) FOR UPDATE`, id, p.ID).Scan(&wid, &markdown, &version)
	if e != nil || version != in.Version {
		ragIndexError(w, errRAGChanged)
		return
	}
	if e = ragActorTx(ctx, tx, p, id, wid, true); e != nil {
		ragIndexError(w, e)
		return
	}
	cfg, e := s.ragSettingsTx(ctx, tx, wid)
	var generation string
	if e == nil {
		generation, e = ragSelectedGeneration(ctx, tx, wid, r.URL.Query().Get("generation_id"))
	}
	if e == nil {
		cfg, e = s.ragGenerationConfig(ctx, tx, cfg, wid, generation)
	}
	if e != nil {
		ragIndexError(w, e)
		return
	}
	if !boolean(cfg, "rag_enabled") {
		apiError(w, 409, "검색 AI가 비활성화되었습니다")
		return
	}
	if validateRAGSettings(cfg) != nil || ragProviderFingerprint(cfg) != in.Provider || (in.AllowRerank && (!boolean(cfg, "rag_rerank_enabled") || ragRerankFingerprint(cfg) != in.Rerank)) {
		ragIndexError(w, errRAGChanged)
		return
	}
	chunks, e := chunkMarkdown(markdown, 6144, 512)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if len(chunks) == 0 {
		apiError(w, 400, "색인할 문서 본문이 없습니다")
		return
	}
	g, e := ragGrantForGenerationTx(ctx, tx, id, generation, true)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		ragIndexError(w, e)
		return
	}
	// Repeated clicks cannot enqueue an unbounded collection of overlapping jobs.
	if g.JobID != "" {
		var busy bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM automation_jobs WHERE id=$1 AND status IN ('pending','running') AND NOT cancel_requested)`, g.JobID).Scan(&busy)
		if e != nil {
			ragIndexError(w, e)
			return
		}
		if busy {
			apiError(w, 409, "이미 진행 중인 색인 작업이 있습니다. 취소 또는 완료 후 다시 요청하세요")
			return
		}
	}
	var pending int
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('madi:rag:actor:' || $1,0))`, p.ID); e != nil {
		ragIndexError(w, e)
		return
	}
	e = tx.QueryRow(ctx, `SELECT count(*) FROM automation_jobs WHERE actor_id=$1 AND kind='rag.index' AND status IN ('pending','running')`, p.ID).Scan(&pending)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	if pending >= 10 {
		apiError(w, 429, "동시에 요청할 수 있는 색인 작업은 10개입니다")
		return
	}
	rerank := ""
	if in.AllowRerank {
		rerank = in.Rerank
	}
	if g.ID == "" {
		g.ID = newID()
	}
	e = tx.QueryRow(ctx, `INSERT INTO rag_index_grants(id,document_id,workspace_id,actor_id,token_id,actor_constraints,provider_fingerprint,rerank_fingerprint,expected_version,auto_reindex,generation_id) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid) ON CONFLICT(document_id,generation_id) DO UPDATE SET actor_id=EXCLUDED.actor_id,token_id=EXCLUDED.token_id,actor_constraints=EXCLUDED.actor_constraints,revision=rag_index_grants.revision+1,active=true,auto_reindex=EXCLUDED.auto_reindex,provider_fingerprint=EXCLUDED.provider_fingerprint,rerank_fingerprint=EXCLUDED.rerank_fingerprint,expected_version=EXCLUDED.expected_version,updated_at=now() RETURNING revision`, g.ID, id, wid, p.ID, p.TokenID, jsonValue(constraintsFor(p)), in.Provider, rerank, version, in.Auto, generation).Scan(&g.Revision)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	jobID, e := s.EnqueueJob(ctx, tx, "rag.index", p.ID, wid, map[string]any{"grant_id": g.ID, "document_id": id, "document_version": version, "grant_revision": g.Revision, "provider_fingerprint": in.Provider})
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE automation_jobs SET timeout_seconds=1800,max_attempts=3,resource_id=$2 WHERE id=$1`, jobID, id)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE rag_index_grants SET last_job_id=$2 WHERE id=$1`, g.ID, jobID)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM rag_vector_indexes WHERE grant_id=$1`, g.ID)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM rag_reindex_queue WHERE document_id=$1 AND NULLIF($2,'')::uuid IS NOT DISTINCT FROM (SELECT active_id FROM rag_generation_state WHERE workspace_id=$3)`, id, generation, wid)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		ragIndexError(w, e)
		return
	}
	s.audit(r, "RAG_INDEX_CONSENT", id, map[string]any{"grant_revision": g.Revision, "provider_fingerprint": in.Provider, "allow_rerank": in.AllowRerank, "auto_reindex": in.Auto})
	jsonResponse(w, 202, map[string]any{"job_id": jobID, "grant_revision": g.Revision})
}

func (s *Server) revokeRAGIndex(w http.ResponseWriter, r *http.Request) { s.stopRAGIndex(w, r, true) }
func (s *Server) cancelRAGIndex(w http.ResponseWriter, r *http.Request) { s.stopRAGIndex(w, r, false) }
func (s *Server) stopRAGIndex(w http.ResponseWriter, r *http.Request, revoke bool) {
	p, id := current(r), r.PathValue("id")
	if !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "ai:execute") || !s.canDocument(r.Context(), p, id, true) {
		apiError(w, 403, "문서 작성 및 AI 실행 권한이 모두 필요합니다")
		return
	}
	var in struct {
		Revision int64  `json:"grant_revision"`
		JobID    string `json:"job_id"`
	}
	if decode(r, &in) != nil || (revoke && in.Revision < 1) || (!revoke && !validID(in.JobID)) {
		apiError(w, 400, "동의 버전 또는 작업 ID를 확인하세요")
		return
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	defer tx.Rollback(ctx)
	var wid string
	e = tx.QueryRow(ctx, `SELECT workspace_id::text FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) FOR SHARE`, id, p.ID).Scan(&wid)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	if e = ragActorTx(ctx, tx, p, id, wid, true); e != nil {
		ragIndexError(w, e)
		return
	}
	generation, e := ragSelectedGeneration(ctx, tx, wid, r.URL.Query().Get("generation_id"))
	if e != nil {
		ragIndexError(w, e)
		return
	}
	g, e := ragGrantForGenerationTx(ctx, tx, id, generation, true)
	if e != nil {
		ragIndexError(w, e)
		return
	}
	if (revoke && g.Revision != in.Revision) || (!revoke && g.JobID != in.JobID) {
		ragIndexError(w, errRAGChanged)
		return
	}
	_, e = tx.Exec(ctx, `UPDATE automation_jobs SET cancel_requested=true,status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END,updated_at=now() WHERE (id IN (SELECT id FROM rag_vector_indexes WHERE grant_id=$1) OR id=NULLIF($2,'')::uuid) AND status IN ('pending','running')`, g.ID, g.JobID)
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE rag_index_grants SET active=CASE WHEN $2 THEN false ELSE active END,auto_reindex=false,revision=revision+1,updated_at=now() WHERE id=$1`, g.ID, revoke)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM rag_reindex_queue WHERE document_id=$1 AND NULLIF($2,'')::uuid IS NOT DISTINCT FROM (SELECT active_id FROM rag_generation_state WHERE workspace_id=$3)`, id, generation, wid)
	}
	if e == nil && revoke {
		_, e = tx.Exec(ctx, `DELETE FROM rag_vector_indexes WHERE grant_id=$1`, g.ID)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		ragIndexError(w, e)
		return
	}
	action := "RAG_INDEX_CANCEL"
	if revoke {
		action = "RAG_INDEX_REVOKE"
	}
	s.audit(r, action, id, map[string]any{"revision": g.Revision})
	respond(w, map[string]any{"revoked": revoke, "cancelled": true}, nil)
}
