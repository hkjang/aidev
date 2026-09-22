package server

import (
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (s *Server) deleteRAGGeneration(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 400, "색인 세대 ID를 확인하세요")
		return
	}
	g, e := s.ragGenerationTx(r.Context(), s.DB, id, false)
	if e != nil || !s.workspaceAdmin(r, g.WorkspaceID) {
		apiError(w, 403, "색인 세대 삭제 권한이 없습니다")
		return
	}
	var in struct {
		Revision int64 `json:"revision"`
		Confirm  bool  `json:"confirm"`
	}
	if decode(r, &in) != nil || !in.Confirm || in.Revision < 1 {
		apiError(w, 400, "현재 세대 버전과 파생 색인·동의 삭제 확인이 필요합니다")
		return
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	if e = ragGenerationOperatorTx(ctx, tx, r, g.WorkspaceID); e != nil {
		ragIndexError(w, e)
		return
	}
	if _, e = tx.Exec(ctx, `SELECT set_config('lock_timeout','500ms',true),set_config('statement_timeout','5s',true)`); e != nil {
		respond(w, nil, e)
		return
	}
	g, e = s.ragGenerationTx(ctx, tx, id, true)
	if e != nil || g.Revision != in.Revision {
		ragIndexError(w, errRAGChanged)
		return
	}
	var active bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM rag_generation_state WHERE active_id=$1)`, id).Scan(&active); e != nil || active {
		apiError(w, 409, "현재 검색 중인 세대는 삭제할 수 없습니다. 다른 세대를 검증하고 전환하세요")
		return
	}
	var running bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM automation_jobs WHERE (id=NULLIF($1,'')::uuid OR id IN (SELECT last_job_id FROM rag_index_grants WHERE generation_id=$2)) AND status IN ('pending','running'))`, g.IndexJobID, g.ID).Scan(&running); e != nil || running {
		apiError(w, 409, "진행 중인 세대 작업을 먼저 취소하고 완료 상태를 확인하세요")
		return
	}
	// Only the exact service-derived index for this immutable UUID is a target.
	// A short lock timeout fails cleanly instead of blocking active search/edits.
	var schema string
	e = tx.QueryRow(ctx, `SELECT n.nspname FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_am am ON am.oid=c.relam WHERE c.relname=$1 AND i.indrelid='rag_vector_chunks'::regclass AND am.amname='hnsw'`, ragHNSWIndexName(g.ID)).Scan(&schema)
	if e != nil && e != pgx.ErrNoRows {
		respond(w, nil, e)
		return
	}
	if e == nil {
		if _, e = tx.Exec(ctx, `DROP INDEX `+pgx.Identifier{schema, ragHNSWIndexName(g.ID)}.Sanitize()); e != nil {
			apiError(w, 409, "다른 작업이 인덱스를 사용 중입니다. 잠시 후 다시 삭제하세요")
			return
		}
	}
	if _, e = tx.Exec(ctx, `DELETE FROM rag_generations WHERE id=$1`, g.ID); e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RAG_GENERATION_DELETE", g.ID, map[string]any{"revision": g.Revision})
	jsonResponse(w, 200, map[string]any{"deleted": true, "documents_preserved": true, "notice": "이 세대의 공급자 설정·동의·검증 영수증·파생 벡터와 HNSW 인덱스만 삭제했습니다. 원본 문서는 유지됩니다. 세대를 다시 만들려면 새 동의와 색인이 필요합니다."})
}
