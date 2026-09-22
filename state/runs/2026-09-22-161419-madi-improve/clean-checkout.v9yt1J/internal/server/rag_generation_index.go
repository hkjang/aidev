package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) queueRAGGenerationIndex(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 400, "색인 세대 ID를 확인하세요")
		return
	}
	g, e := s.ragGenerationTx(r.Context(), s.DB, id, false)
	if e != nil || !s.workspaceAdmin(r, g.WorkspaceID) {
		apiError(w, 403, "색인 세대 관리 권한이 없습니다")
		return
	}
	var in struct {
		Revision int64 `json:"revision"`
		Confirm  bool  `json:"confirm"`
	}
	if decode(r, &in) != nil || !in.Confirm || in.Revision < 1 {
		apiError(w, 400, "현재 세대 버전과 인덱스 생성 확인이 필요합니다")
		return
	}
	namespace, version, e := s.ragVectorExtension(r.Context())
	if e != nil || !ragVectorSupportsIterative(version) {
		apiError(w, 409, "운영자가 pgvector 0.8 이상을 먼저 설치해야 합니다. 자동 설치하지 않습니다")
		return
	}
	_ = namespace
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = ragGenerationOperatorTx(r.Context(), tx, r, g.WorkspaceID); e != nil {
		ragIndexError(w, e)
		return
	}
	g, e = s.ragGenerationTx(r.Context(), tx, id, true)
	if e != nil || g.Revision != in.Revision || g.Status == "disabled" || g.Dimensions < 1 || g.Dimensions > 2000 {
		apiError(w, 409, "ANN 세대는 현재 버전과 1~2000차원 설정이 필요합니다")
		return
	}
	var pending bool
	if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM automation_jobs WHERE id=NULLIF($1,'')::uuid AND status IN ('pending','running'))`, g.IndexJobID).Scan(&pending); e != nil {
		respond(w, nil, e)
		return
	}
	if pending {
		apiError(w, 409, "이미 인덱스를 준비하고 있습니다")
		return
	}
	cfg, e := s.ragSettingsTx(r.Context(), tx, g.WorkspaceID)
	if e != nil || !boolean(cfg, "rag_enabled") {
		ragIndexError(w, errRAGChanged)
		return
	}
	cookie, _ := r.Cookie("madi_session")
	job, e := s.EnqueueJob(r.Context(), tx, "rag.hnsw", current(r).ID, g.WorkspaceID, map[string]any{"generation_id": id, "generation_revision": g.Revision, "settings_fingerprint": ragGenerationSettingsFingerprint(cfg), "session_hash": digest(cookie.Value)})
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE rag_generations SET index_job_id=$2,updated_at=now() WHERE id=$1`, id, job)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET timeout_seconds=600 WHERE id=$1`, job)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RAG_HNSW_REQUEST", id, map[string]any{"job_id": job, "dimensions": g.Dimensions})
	jsonResponse(w, 202, map[string]any{"job_id": job, "generation_id": id, "index_name": ragHNSWIndexName(id)})
}

func ragGenerationSettingsFingerprint(cfg map[string]any) string {
	return digest(string(jsonValue(ragGenerationStoredConfig(cfg))))
}

func (s *Server) ragHNSWJobGuard(ctx context.Context, j Job) (ragGeneration, error) {
	g, e := s.ragGenerationTx(ctx, s.DB, str(j.Payload, "generation_id"), false)
	if e != nil || g.WorkspaceID != j.WorkspaceID || g.IndexJobID != j.ID || g.Status == "disabled" || g.Revision != int64(number(j.Payload, "generation_revision", 0)) || j.TokenID != "" || j.Constraints.PluginID != "" || j.Constraints.Restricted {
		return g, errRAGChanged
	}
	var allowed bool
	e = s.DB.QueryRow(ctx, `SELECT NOT u.disabled AND u.kind='user' AND u.role<>'viewer' AND m.role IN ('owner','admin') AND ss.expires_at>now() AND NOT j.cancel_requested AND j.status IN ('pending','running') FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$2 JOIN sessions ss ON ss.user_id=u.id AND ss.token_hash=$3 JOIN automation_jobs j ON j.id=$4 WHERE u.id=$1`, j.ActorID, j.WorkspaceID, str(j.Payload, "session_hash"), j.ID).Scan(&allowed)
	if e != nil || !allowed {
		return g, errRAGChanged
	}
	cfg, e := s.effectiveSettings(ctx, j.WorkspaceID)
	if e != nil || !boolean(cfg, "rag_enabled") || ragGenerationSettingsFingerprint(cfg) != str(j.Payload, "settings_fingerprint") {
		return g, errRAGChanged
	}
	return g, nil
}

func (s *Server) runRAGGenerationIndex(ctx context.Context, j Job) (map[string]any, error) {
	g, e := s.ragHNSWJobGuard(ctx, j)
	if e != nil {
		return nil, jobPermanent(errRAGChanged.Error())
	}
	namespace, version, e := s.ragVectorExtension(ctx)
	if e != nil || !ragVectorSupportsIterative(version) || g.Dimensions < 1 || g.Dimensions > 2000 {
		return nil, jobPermanent("pgvector 0.8 이상과 1~2000차원 세대가 필요합니다")
	}
	conn, e := s.DB.Acquire(ctx)
	if e != nil {
		return nil, e
	}
	defer conn.Release()
	// Only one concurrent CREATE INDEX per schema. A nonblocking advisory lock
	// keeps queued builders from occupying every pool slot needed by ACL guards.
	var locked bool
	if e = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended('madi:rag:hnsw:'||current_schema(),0))`).Scan(&locked); e != nil {
		return nil, e
	}
	if !locked {
		return nil, errors.New("다른 벡터 인덱스 생성 작업이 진행 중입니다")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := conn.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended('madi:rag:hnsw:'||current_schema(),0))`); err != nil {
			_ = conn.Conn().Close(cleanup)
		}
	}()
	var schema string
	if e = conn.QueryRow(ctx, `SELECT n.nspname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.oid='rag_vector_chunks'::regclass`).Scan(&schema); e != nil {
		return nil, e
	}
	name := ragHNSWIndexName(g.ID)
	var ready, owned bool
	e = conn.QueryRow(ctx, `SELECT i.indisvalid AND i.indisready,i.indrelid='rag_vector_chunks'::regclass AND am.amname='hnsw' FROM pg_class c JOIN pg_index i ON i.indexrelid=c.oid JOIN pg_am am ON am.oid=c.relam WHERE c.relnamespace=$1::regnamespace AND c.relname=$2`, schema, name).Scan(&ready, &owned)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	if e == nil && !owned {
		return nil, jobPermanent("같은 이름의 인덱스가 서비스 소유 형식과 다릅니다")
	}
	watched, stop, guardErr := ragWatch(ctx, func(check context.Context) error { _, e := s.ragHNSWJobGuard(check, j); return e })
	defer stop()
	if !ready {
		if owned {
			if _, e = conn.Exec(watched, `DROP INDEX CONCURRENTLY `+pgx.Identifier{schema, name}.Sanitize()); e != nil {
				return nil, e
			}
		}
		qualified := pgx.Identifier{namespace}.Sanitize()
		sql := `CREATE INDEX CONCURRENTLY ` + pgx.Identifier{name}.Sanitize() + ` ON ` + pgx.Identifier{schema, "rag_vector_chunks"}.Sanitize() + ` USING hnsw ((embedding::` + qualified + `.vector(` + strconv.Itoa(g.Dimensions) + `)) ` + qualified + `.vector_cosine_ops) WITH (m=16,ef_construction=100) WHERE generation_id='` + g.ID + `'::uuid AND cardinality(embedding)=` + strconv.Itoa(g.Dimensions)
		_, e = conn.Exec(watched, sql)
	}
	stop()
	if *guardErr != nil {
		return nil, jobPermanent(errRAGChanged.Error())
	}
	if e != nil {
		return nil, e
	}
	if _, e = s.ragHNSWJobGuard(ctx, j); e != nil {
		return nil, jobPermanent(errRAGChanged.Error())
	}
	return map[string]any{"generation_id": g.ID, "index_name": name, "dimensions": g.Dimensions, "extension_version": version, "ready": true}, nil
}
