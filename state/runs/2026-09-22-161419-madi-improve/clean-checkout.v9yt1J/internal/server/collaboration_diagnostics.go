package server

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func (s *Server) collaborationDiagnostics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) || !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 진단 조회 권한이 없습니다")
		return
	}
	value, err := s.one(r.Context(), `SELECT jsonb_build_object('document_id',d.id,'version',d.version,'epoch',coalesce(c.epoch::text,''),'sequence',coalesce(c.sequence,0),'snapshot_sequence',coalesce(c.snapshot_sequence,0),'snapshot_bytes',coalesce(octet_length(c.state),0),'snapshot_at',c.snapshot_at,'updated_at',c.updated_at,'reset_reason',coalesce(c.reset_reason,''),'pending_updates',(SELECT count(*) FROM collaboration_updates x WHERE x.document_id=d.id),'pending_bytes',(SELECT coalesce(sum(octet_length(x.update_data)),0) FROM collaboration_updates x WHERE x.document_id=d.id),'active_connections',(SELECT count(*) FROM collaboration_presence p WHERE p.document_id=d.id AND p.touched_at>now()-interval '30 seconds'),'can_compact',madi_document_allowed($2,d.id,true) AND (d.owner_id=$2 OR EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=d.workspace_id AND m.user_id=$2 AND m.role IN ('owner','admin')))) FROM documents d LEFT JOIN document_collaboration c ON c.document_id=d.id WHERE d.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)`, id, current(r).ID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted {
		value["can_compact"] = false
	}
	value["limits"] = map[string]any{"checkpoint_updates": collaborationCheckpointUpdates, "checkpoint_bytes": collaborationCheckpointBytes, "checkpoint_seconds": 60, "automatic_compact_bytes": collaborationCompactBytes, "maximum_update_bytes": collaborationMaxUpdate, "authority_check_seconds": 1, "catchup_seconds": 5}
	value["history_mode"] = "immutable_versions_grouped_activity"
	value["transport"] = "postgres_notify_durable_catchup"
	value["listener_status"] = "idle"
	s.collaborationMu.Lock()
	rt := s.collaboration
	s.collaborationMu.Unlock()
	if rt != nil {
		value["listener_status"] = "reconnecting"
		if rt.connected.Load() {
			value["listener_status"] = "connected"
		}
		value["listener_reconnects"] = rt.reconnects.Load()
		// Global room traffic must not reveal activity in another private document.
		if current(r).Role == "admin" && current(r).TokenID == "" && !current(r).ScopeRestricted {
			value["cache_weight_bytes"] = rt.cacheBytes.Load()
			value["cache_hits"] = rt.cacheHits.Load()
			value["cache_misses"] = rt.cacheMisses.Load()
		}
	}
	respond(w, value, nil)
}

func (s *Server) compactCollaboration(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	id := r.PathValue("id")
	cookie, e := r.Cookie("madi_session")
	if !validID(id) || e != nil || p.TokenID != "" || p.ScopeRestricted {
		apiError(w, 403, "편집 이력 압축은 문서 소유자 또는 워크스페이스 관리자의 로그인 세션이 필요합니다")
		return
	}
	var input struct {
		ExpectedVersion int    `json:"expected_version"`
		ExpectedEpoch   string `json:"expected_epoch"`
		Confirm         bool   `json:"confirm"`
	}
	if decode(r, &input) != nil || input.ExpectedVersion < 1 || !validID(input.ExpectedEpoch) || !input.Confirm {
		apiError(w, 400, "현재 버전·편집 기준과 압축 확인이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var version int
	var epoch, markdown string
	var allowed bool
	e = tx.QueryRow(r.Context(), `SELECT d.version,c.epoch::text,d.markdown,(d.owner_id=u.id OR m.role IN ('owner','admin')) FROM documents d JOIN document_collaboration c ON c.document_id=d.id JOIN workspace_members m ON m.workspace_id=d.workspace_id AND m.user_id=$2 JOIN users u ON u.id=m.user_id JOIN sessions s ON s.user_id=u.id AND s.token_hash=$3 AND s.expires_at>clock_timestamp() WHERE d.id=$1 AND d.deleted_at IS NULL AND NOT u.disabled AND u.kind='user' AND madi_document_allowed(u.id,d.id,true) FOR UPDATE OF d FOR SHARE OF u,m,s`, id, p.ID, digest(cookie.Value)).Scan(&version, &epoch, &markdown, &allowed)
	if e == pgx.ErrNoRows || e == nil && !allowed {
		apiError(w, 403, "현재 문서 압축 권한이 없습니다")
		return
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if version != input.ExpectedVersion || epoch != input.ExpectedEpoch {
		apiError(w, 409, "문서 또는 편집 기준이 변경되었습니다. 진단을 다시 불러오세요")
		return
	}
	newEpoch := newID()
	if e = collaborationResetTx(r.Context(), tx, id, newEpoch, markdown, version, "manual_compaction"); e != nil {
		respond(w, nil, e)
		return
	}
	// Existing actor/member/session and document locks protect this operation,
	// but wall-clock expiry must be checked again after reset/journal lock waits.
	if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>clock_timestamp()) AND madi_document_allowed($2,$3,true)`, digest(cookie.Value), p.ID, id).Scan(&allowed); e != nil {
		respond(w, nil, e)
		return
	}
	if !allowed {
		apiError(w, 403, "현재 문서 압축 권한 또는 세션이 만료되었습니다")
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "COLLABORATION_COMPACT", id, map[string]any{"version": version})
	respond(w, map[string]any{"epoch": newEpoch, "version": version, "markdown_unchanged": true, "reconnect_required": true}, nil)
}

func (s *Server) collaborationHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) || !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 이력 조회 권한이 없습니다")
		return
	}
	before := 2147483647
	if raw := r.URL.Query().Get("before"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 || n > 2147483647 {
			apiError(w, 400, "이력 페이지 버전을 확인하세요")
			return
		}
		before = n
	}
	value, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',g.id,'first_version',g.first_version,'last_version',g.last_version,'started_at',g.started_at,'updated_at',g.updated_at,'user_name',u.name,'changes',g.last_version-g.first_version+1) FROM collaboration_history_groups g JOIN users u ON u.id=g.actor_id JOIN documents d ON d.id=g.document_id WHERE g.document_id=$1 AND g.last_version<$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) ORDER BY g.last_version DESC LIMIT 50`, id, before, current(r).ID)
	respond(w, value, e)
}
