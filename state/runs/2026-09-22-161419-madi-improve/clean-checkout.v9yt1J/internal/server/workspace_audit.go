package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func (s *Server) migrateWorkspaceAudit(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, `CREATE INDEX IF NOT EXISTS audit_resource_created_idx ON audit_logs(resource,created_at DESC); CREATE INDEX IF NOT EXISTS audit_action_created_idx ON audit_logs(action,created_at DESC)`)
	return err
}
func (s *Server) registerWorkspaceAudit() {
	s.handle("GET /api/v1/workspaces/{id}/audit", s.workspaceAudit)
}

type auditCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

var auditActionPattern = regexp.MustCompile(`^[A-Z][A-Z_0-9]{0,63}$`)

func (s *Server) workspaceAudit(w http.ResponseWriter, r *http.Request) {
	wid := r.PathValue("id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "팀 감사로그는 현재 워크스페이스 소유자·관리자만 확인할 수 있습니다")
		return
	}
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 365 {
			apiError(w, 400, "감사 조회 기간은 1~365일입니다")
			return
		}
		days = n
	}
	action := r.URL.Query().Get("action")
	if action != "" && !auditActionPattern.MatchString(action) {
		apiError(w, 400, "감사 동작 필터를 확인하세요")
		return
	}
	cursor := auditCursor{time.Now().Add(time.Second), "ffffffff-ffff-ffff-ffff-ffffffffffff"}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		data, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || len(data) > 200 || json.Unmarshal(data, &cursor) != nil || !validID(cursor.ID) || cursor.Time.IsZero() {
			apiError(w, 400, "감사 페이지 커서를 확인하세요")
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := s.rows(ctx, workspaceAuditQuery, current(r).ID, wid, days, action, cursor.Time, cursor.ID)
	if err != nil {
		apiError(w, 503, "감사 기록을 조회하지 못했습니다. 기간을 줄이거나 잠시 후 다시 시도하세요")
		return
	}
	next := ""
	if len(items) > 100 {
		items = items[:100]
		last := items[len(items)-1]
		stamp, err := time.Parse(time.RFC3339Nano, str(last, "created_at"))
		if err == nil {
			next = base64.RawURLEncoding.EncodeToString(jsonValue(auditCursor{stamp, str(last, "id")}))
		}
	}
	jsonResponse(w, 200, map[string]any{"items": items, "next_cursor": next, "days": days, "notice": "현재 접근 가능한 문서·첨부·데이터베이스·공간·템플릿·캔버스 및 팀 설정·저장소·키 관리 동작을 표시합니다. 다른 사람의 개인 문서와 접근 불가·이미 없어진 원본 기록은 제외합니다. 본문·이전/이후 원문·검색어·IP·비밀 설정은 표시하지 않습니다."})
}

const workspaceAuditQuery = `WITH candidate AS (
 SELECT a.*,CASE WHEN a.resource~'^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$' THEN a.resource::uuid END resource_uuid
 FROM audit_logs a WHERE a.created_at>now()-($3::int*interval '1 day') AND (a.created_at,a.id)<($5::timestamptz,$6::uuid) AND ($4='' OR a.action=$4)
), resolved AS (
 SELECT a.*,u.name actor_name,
 CASE WHEN d.id IS NOT NULL THEN 'document' WHEN db.id IS NOT NULL THEN 'database' WHEN sp.id IS NOT NULL THEN 'space' WHEN t.id IS NOT NULL THEN 'template' WHEN cv.id IS NOT NULL THEN 'canvas' WHEN k.id IS NOT NULL THEN 'key' WHEN st.id IS NOT NULL THEN 'storage' WHEN ws.id IS NOT NULL THEN 'workspace' END resource_kind,
 CASE WHEN d.id IS NOT NULL THEN d.id WHEN db.id IS NOT NULL THEN db.id WHEN sp.id IS NOT NULL THEN sp.id WHEN t.id IS NOT NULL THEN t.id WHEN cv.id IS NOT NULL THEN cv.id WHEN k.id IS NOT NULL THEN k.id WHEN st.id IS NOT NULL THEN st.id WHEN ws.id IS NOT NULL THEN ws.id END target_id,
 coalesce(d.title,db.name,sp.name,t.name,cv.title,k.name,st.name,ws.name) resource_title
 FROM candidate a LEFT JOIN users u ON u.id=a.user_id
 LEFT JOIN attachments f ON f.id=a.resource_uuid AND a.action LIKE 'FILE_%'
 LEFT JOIN documents d ON d.id=CASE WHEN f.id IS NOT NULL THEN f.document_id ELSE a.resource_uuid END
  AND (f.id IS NOT NULL OR a.action LIKE 'DOCUMENT_%' OR a.action LIKE 'COMMENT_%' OR a.action IN ('SHARE_CREATE','CAPTURE_CLASSIFY','COLLABORATION_CONNECT','AI_QUERY','AI_ERROR'))
  AND d.workspace_id=$2 AND madi_document_allowed($1,d.id,false)
 LEFT JOIN database_rows dr ON dr.id=a.resource_uuid AND (a.action LIKE 'DATABASE_ROW_%' OR a.action IN ('DATABASE_AI_ERROR','DATABASE_AI_GENERATE','DATABASE_BUTTON_EXECUTE'))
 LEFT JOIN databases db ON db.id=CASE WHEN dr.id IS NOT NULL THEN dr.database_id ELSE a.resource_uuid END AND a.action LIKE 'DATABASE_%'
  AND db.workspace_id=$2 AND (db.space_id IS NULL OR madi_space_allowed($1,db.space_id,false))
 LEFT JOIN spaces sp ON sp.id=a.resource_uuid AND a.action LIKE 'SPACE_%' AND sp.workspace_id=$2 AND madi_space_allowed($1,sp.id,false)
 LEFT JOIN document_templates t ON t.id=a.resource_uuid AND a.action LIKE 'TEMPLATE_%' AND t.workspace_id=$2 AND madi_template_allowed($1,t.id,false)
 LEFT JOIN canvases cv ON cv.id=a.resource_uuid AND a.action LIKE 'CANVAS_%' AND cv.workspace_id=$2 AND madi_canvas_allowed($1,cv.id,false)
 LEFT JOIN api_keys k ON k.id=a.resource_uuid AND a.action LIKE 'KEY_%' AND k.workspace_id=$2
 LEFT JOIN storage_providers st ON st.id=a.resource_uuid AND a.action LIKE 'STORAGE_%' AND st.workspace_id=$2
 LEFT JOIN workspaces ws ON ws.id=a.resource_uuid AND ws.id=$2 AND (a.action LIKE 'WORKSPACE_%' OR a.action IN ('PERMISSION_CHANGE','STORAGE_ASSIGNMENT','RAG_CONNECTION_TEST'))
 WHERE EXISTS(SELECT 1 FROM users actor JOIN workspace_members m ON m.user_id=actor.id WHERE actor.id=$1 AND NOT actor.disabled AND actor.role<>'viewer' AND m.workspace_id=$2 AND m.role IN ('owner','admin'))
)
SELECT jsonb_build_object('id',id,'created_at',created_at,'action',action,'actor_name',actor_name,'resource_kind',resource_kind,'resource_id',target_id,'resource_title',resource_title,'changes',jsonb_strip_nulls(jsonb_build_object(
 'version',CASE WHEN jsonb_typeof(details->'version')='number' THEN details->'version' END,
 'enabled',CASE WHEN jsonb_typeof(details->'enabled')='boolean' THEN details->'enabled' END,
 'restored',CASE WHEN jsonb_typeof(details->'restored')='boolean' THEN details->'restored' END,
 'rows',CASE WHEN jsonb_typeof(details->'rows')='number' THEN details->'rows' END,
 'status',CASE WHEN details->>'status' IN ('draft','published','stale','archived','review','rejected','approved','pending','running','succeeded','failed','cancelled') THEN details->'status' END
 ))) FROM resolved WHERE resource_kind IS NOT NULL ORDER BY created_at DESC,id DESC LIMIT 101`
