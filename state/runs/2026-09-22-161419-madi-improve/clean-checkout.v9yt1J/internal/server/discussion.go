package server

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//go:embed discussion.sql
var discussionSchema string
var mentionPattern = regexp.MustCompile(`@\[[^\]\n]{1,100}\]\((user|team):([a-fA-F0-9-]{36})\)`)
var errDiscussionRecipients = errors.New("댓글 알림 대상은 한 번에 최대 100명입니다")

func (s *Server) migrateDiscussion(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, discussionSchema)
	return e
}
func (s *Server) registerDiscussion() {
	s.handle("PATCH /api/v1/documents/{id}/comments/{commentId}", s.discussionUpdateComment)
	s.handle("DELETE /api/v1/documents/{id}/comments/{commentId}", s.discussionDeleteComment)
	s.handle("POST /api/v1/documents/{id}/comments/{commentId}/reactions", s.discussionReaction)
	s.handle("GET /api/v1/teams", s.listTeams)
	s.handle("POST /api/v1/teams", s.createTeam)
	s.handle("PUT /api/v1/teams/{id}", s.updateTeam)
	s.handle("DELETE /api/v1/teams/{id}", s.deleteTeam)
	s.handle("GET /api/v1/teams/{id}/members", s.teamMembers)
	s.handle("PUT /api/v1/teams/{id}/members", s.setTeamMember)
	s.handle("POST /api/v1/notifications/read-all", func(w http.ResponseWriter, r *http.Request) {
		_, e := s.DB.Exec(r.Context(), "UPDATE notifications SET read_at=now() WHERE user_id=$1 AND read_at IS NULL", current(r).ID)
		respond(w, map[string]bool{"ok": true}, e)
	})
}
func (s *Server) canComment(r *http.Request, id string) bool {
	if !validID(id) || !hasIntegrationScope(current(r), "document:write") || !s.canDocument(r.Context(), current(r), id, false) {
		return false
	}
	var allowed bool
	e := s.DB.QueryRow(r.Context(), "SELECT deleted_at IS NULL AND madi_document_comment_allowed($1,id) FROM documents WHERE id=$2", current(r).ID, id).Scan(&allowed)
	return e == nil && allowed
}

const discussionSelect = `SELECT (to_jsonb(c)-'body'-'search_vector')||jsonb_build_object('body',CASE WHEN c.deleted_at IS NULL THEN c.body ELSE '' END,'user_name',u.name,'assigned_name',a.name,'resolved_name',z.name,'can_edit',c.user_id=$2 AND c.deleted_at IS NULL,'reactions',coalesce((SELECT jsonb_agg(x.value ORDER BY x.kind) FROM (SELECT cr.reaction kind,jsonb_build_object('reaction',cr.reaction,'count',count(*),'mine',bool_or(cr.user_id=$2)) value FROM comment_reactions cr WHERE cr.comment_id=c.id GROUP BY cr.reaction) x),'[]'),'anchor_current',CASE WHEN c.quote<>'' THEN position(c.quote in d.markdown)>0 WHEN c.block_id<>'' THEN EXISTS(SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(d.block_metadata->'blocks')='array' THEN d.block_metadata->'blocks' ELSE '[]'::jsonb END) b WHERE b->>'id'=c.block_id) ELSE true END) FROM comments c JOIN users u ON u.id=c.user_id JOIN documents d ON d.id=c.document_id LEFT JOIN users a ON a.id=c.assigned_to LEFT JOIN users z ON z.id=c.resolved_by`

func (s *Server) discussionComment(r *http.Request, id string) (map[string]any, error) {
	return s.one(r.Context(), discussionSelect+" WHERE c.id=$1", id, current(r).ID)
}
func (s *Server) discussionComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 접근 권한이 없습니다")
		return
	}
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 500 {
			apiError(w, 400, "댓글 조회 개수는 1~500입니다")
			return
		}
		limit = v
	}
	after := r.URL.Query().Get("after")
	if after != "" && !validID(after) {
		apiError(w, 400, "댓글 페이지 위치를 확인하세요")
		return
	}
	out, e := s.rows(r.Context(), discussionSelect+" WHERE c.document_id=$1 AND ($3='' OR (c.created_at,c.id)>(SELECT created_at,id FROM comments WHERE id=NULLIF($3,'')::uuid AND document_id=$1)) ORDER BY c.created_at,c.id LIMIT $4", id, current(r).ID, after, limit)
	canEdit := s.canComment(r, id)
	for _, v := range out {
		v["can_edit"] = boolean(v, "can_edit") && canEdit
	}
	respond(w, out, e)
}
func (s *Server) discussionRecipient(ctx context.Context, tx pgx.Tx, id, uid string) bool {
	var allowed bool
	e := tx.QueryRow(ctx, "SELECT madi_document_allowed($1,$2,false)", uid, id).Scan(&allowed)
	return e == nil && allowed
}
func (s *Server) notifyDiscussion(r *http.Request, tx pgx.Tx, did, cid, body, parent, assigned string) error {
	recipients := map[string]bool{}
	var owner string
	if e := tx.QueryRow(r.Context(), "SELECT owner_id::text FROM documents WHERE id=$1", did).Scan(&owner); e != nil {
		return e
	}
	recipients[owner] = true
	if parent != "" {
		var uid string
		if e := tx.QueryRow(r.Context(), "SELECT user_id::text FROM comments WHERE id=$1", parent).Scan(&uid); e != nil {
			return e
		}
		recipients[uid] = true
	}
	if assigned != "" {
		recipients[assigned] = true
	}
	for _, match := range mentionPattern.FindAllStringSubmatch(body, -1) {
		if !validID(match[2]) {
			continue
		}
		if match[1] == "user" {
			recipients[match[2]] = true
		} else {
			rows, e := tx.Query(r.Context(), "SELECT m.user_id::text FROM team_members m JOIN teams t ON t.id=m.team_id JOIN documents d ON d.workspace_id=t.workspace_id WHERE t.id=$1 AND d.id=$2 LIMIT 101", match[2], did)
			if e != nil {
				return e
			}
			for rows.Next() {
				var uid string
				if e = rows.Scan(&uid); e != nil {
					break
				}
				recipients[uid] = true
			}
			if e == nil {
				e = rows.Err()
			}
			rows.Close()
			if e != nil {
				return e
			}
		}
	}
	if len(recipients) > 100 {
		return errDiscussionRecipients
	}
	for uid := range recipients {
		if uid == current(r).ID || !s.discussionRecipient(r.Context(), tx, did, uid) {
			continue
		}
		_, e := tx.Exec(r.Context(), "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,$3,$4)", newID(), uid, current(r).Name+"님이 문서 토론에서 언급하거나 댓글을 남겼습니다", did)
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *Server) discussionAddComment(w http.ResponseWriter, r *http.Request) {
	did := r.PathValue("id")
	if !s.canComment(r, did) {
		apiError(w, 403, "댓글 작성 권한이 없습니다")
		return
	}
	var in struct {
		Body     string `json:"body"`
		Parent   string `json:"parent_id"`
		Block    string `json:"block_id"`
		Quote    string `json:"quote"`
		Assigned string `json:"assigned_to"`
		Version  int    `json:"document_version"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.Body) == "" || len(in.Body) > 10000 || len(in.Quote) > 4000 || (in.Parent != "" && !validID(in.Parent)) || (in.Block != "" && !validID(in.Block)) || (in.Assigned != "" && !validID(in.Assigned)) {
		apiError(w, 400, "댓글 내용·참조·담당자를 확인하세요 (본문 10,000바이트 이내)")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var wid, title, md string
	var version int
	e = tx.QueryRow(r.Context(), "SELECT workspace_id::text,title,markdown,version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_comment_allowed($2,id) FOR SHARE", did, current(r).ID).Scan(&wid, &title, &md, &version)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if in.Block != "" || in.Quote != "" {
		if in.Version != version {
			apiError(w, 409, "참조 문서가 변경되었습니다. 인용 위치를 다시 선택하세요")
			return
		}
		if in.Quote != "" && !strings.Contains(md, in.Quote) {
			apiError(w, 400, "인용 문장을 문서 원문에서 찾을 수 없습니다")
			return
		}
		if in.Block != "" {
			var found bool
			e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM documents d,jsonb_array_elements(CASE WHEN jsonb_typeof(d.block_metadata->'blocks')='array' THEN d.block_metadata->'blocks' ELSE '[]'::jsonb END) b WHERE d.id=$1 AND b->>'id'=$2)", did, in.Block).Scan(&found)
			if e != nil || !found {
				apiError(w, 400, "현재 문서의 블록 ID를 선택하세요")
				return
			}
		}
	}
	if in.Parent != "" {
		var depth int
		e = tx.QueryRow(r.Context(), `WITH RECURSIVE chain AS (SELECT id,parent_id,1 depth FROM comments WHERE id=$1 AND document_id=$2 AND deleted_at IS NULL UNION ALL SELECT c.id,c.parent_id,ch.depth+1 FROM comments c JOIN chain ch ON c.id=ch.parent_id WHERE ch.depth<6) SELECT coalesce(max(depth),0) FROM chain`, in.Parent, did).Scan(&depth)
		if e != nil || depth == 0 || depth >= 5 {
			apiError(w, 400, "답글은 같은 문서에서 5단계까지 작성할 수 있습니다")
			return
		}
	}
	if in.Assigned != "" && !s.discussionRecipient(r.Context(), tx, did, in.Assigned) {
		apiError(w, 400, "담당자는 문서를 열람할 수 있어야 합니다")
		return
	}
	cid := newID()
	_, e = tx.Exec(r.Context(), "INSERT INTO comments(id,document_id,user_id,body,parent_id,block_id,quote,document_version,assigned_to) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,NULLIF($9,'')::uuid)", cid, did, current(r).ID, in.Body, in.Parent, in.Block, in.Quote, version, in.Assigned)
	if e == nil {
		e = s.notifyDiscussion(r, tx, did, cid, in.Body, in.Parent, in.Assigned)
	}
	if errors.Is(e, errDiscussionRecipients) {
		apiError(w, 400, e.Error())
		return
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "comment.created", WorkspaceID: wid, ResourceID: did, After: map[string]any{"id": cid, "document_id": did, "title": title}})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "COMMENT_CREATE", did, map[string]any{"comment_id": cid, "parent_id": in.Parent, "block_id": in.Block})
	v, e := s.discussionComment(r, cid)
	respond(w, v, e)
}
func (s *Server) discussionUpdateComment(w http.ResponseWriter, r *http.Request) {
	did, cid := r.PathValue("id"), r.PathValue("commentId")
	if !s.canComment(r, did) {
		apiError(w, 403, "댓글 수정 권한이 없습니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "댓글 변경 값을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var author, owner, body, assigned string
	var resolved bool
	e = tx.QueryRow(r.Context(), "SELECT c.user_id::text,d.owner_id::text,c.body,coalesce(c.assigned_to::text,''),c.resolved_at IS NOT NULL FROM comments c JOIN documents d ON d.id=c.document_id WHERE c.id=$1 AND c.document_id=$2 AND c.deleted_at IS NULL AND d.deleted_at IS NULL AND madi_document_comment_allowed($3,d.id) FOR UPDATE OF c", cid, did, current(r).ID).Scan(&author, &owner, &body, &assigned, &resolved)
	if e != nil {
		respond(w, nil, e)
		return
	}
	previousAssigned := assigned
	if value, ok := in["body"]; ok {
		v, ok := value.(string)
		if !ok || strings.TrimSpace(v) == "" || len(v) > 10000 {
			apiError(w, 400, "댓글 본문을 확인하세요")
			return
		}
		if author != current(r).ID {
			apiError(w, 403, "작성자만 댓글 본문을 수정할 수 있습니다")
			return
		}
		body = v
	}
	if value, ok := in["resolved"]; ok {
		v, ok := value.(bool)
		if !ok {
			apiError(w, 400, "해결 여부를 확인하세요")
			return
		}
		if current(r).ID != author && current(r).ID != owner && current(r).ID != assigned {
			apiError(w, 403, "작성자·문서 소유자·담당자만 해결 상태를 바꿀 수 있습니다")
			return
		}
		resolved = v
	}
	if value, ok := in["assigned_to"]; ok {
		v, ok := value.(string)
		if !ok || (v != "" && !validID(v)) {
			apiError(w, 400, "담당자를 확인하세요")
			return
		}
		if author != current(r).ID && owner != current(r).ID {
			apiError(w, 403, "작성자·문서 소유자만 담당자를 변경할 수 있습니다")
			return
		}
		if v != "" && !s.discussionRecipient(r.Context(), tx, did, v) {
			apiError(w, 400, "담당자는 문서를 열람할 수 있어야 합니다")
			return
		}
		assigned = v
	}
	_, e = tx.Exec(r.Context(), "UPDATE comments SET body=$2,assigned_to=NULLIF($3,'')::uuid,resolved_at=CASE WHEN $4 THEN coalesce(resolved_at,now()) ELSE NULL END,resolved_by=CASE WHEN $4 THEN coalesce(resolved_by,$5::uuid) ELSE NULL END,edited_at=now() WHERE id=$1", cid, body, assigned, resolved, current(r).ID)
	if e == nil && assigned != "" && assigned != previousAssigned {
		e = s.notifyDiscussion(r, tx, did, cid, "", "", assigned)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "COMMENT_UPDATE", did, map[string]any{"comment_id": cid, "resolved": resolved, "assigned_to": assigned})
	v, e := s.discussionComment(r, cid)
	respond(w, v, e)
}
func (s *Server) discussionDeleteComment(w http.ResponseWriter, r *http.Request) {
	did, cid := r.PathValue("id"), r.PathValue("commentId")
	if !s.canComment(r, did) {
		apiError(w, 403, "댓글 삭제 권한이 없습니다")
		return
	}
	tag, e := s.DB.Exec(r.Context(), "UPDATE comments SET deleted_at=now(),body='',quote='',block_id='' WHERE id=$1 AND document_id=$2 AND user_id=$3 AND deleted_at IS NULL", cid, did, current(r).ID)
	if e == nil && tag.RowsAffected() == 0 {
		apiError(w, 404, "작성한 댓글을 찾을 수 없습니다")
		return
	}
	if e == nil {
		s.audit(r, "COMMENT_DELETE", did, map[string]any{"comment_id": cid})
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) discussionReaction(w http.ResponseWriter, r *http.Request) {
	did, cid := r.PathValue("id"), r.PathValue("commentId")
	if !s.canComment(r, did) {
		apiError(w, 403, "댓글 반응 권한이 없습니다")
		return
	}
	var in struct {
		Reaction string `json:"reaction"`
		Remove   bool   `json:"remove"`
	}
	if decode(r, &in) != nil || !oneOf(in.Reaction, "like", "thanks", "idea", "check", "heart") {
		apiError(w, 400, "지원하는 댓글 반응을 선택하세요")
		return
	}
	var exists bool
	e := s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM comments WHERE id=$1 AND document_id=$2 AND deleted_at IS NULL)", cid, did).Scan(&exists)
	if e != nil || !exists {
		apiError(w, 404, "댓글을 찾을 수 없습니다")
		return
	}
	if in.Remove {
		_, e = s.DB.Exec(r.Context(), "DELETE FROM comment_reactions WHERE comment_id=$1 AND user_id=$2 AND reaction=$3", cid, current(r).ID, in.Reaction)
	} else {
		_, e = s.DB.Exec(r.Context(), "INSERT INTO comment_reactions(comment_id,user_id,reaction) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", cid, current(r).ID, in.Reaction)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	v, e := s.discussionComment(r, cid)
	respond(w, v, e)
}

func (s *Server) listTeams(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(t)||jsonb_build_object('member_count',(SELECT count(*) FROM team_members m WHERE m.team_id=t.id)) FROM teams t WHERE workspace_id=$1 ORDER BY name", wid)
	respond(w, v, e)
}
func (s *Server) createTeam(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		Name        string `json:"name"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 {
		apiError(w, 400, "팀 이름을 확인하세요")
		return
	}
	if !s.workspaceAdmin(r, in.WorkspaceID) {
		apiError(w, 403, "워크스페이스 관리자만 팀을 만들 수 있습니다")
		return
	}
	id := newID()
	_, e := s.DB.Exec(r.Context(), "INSERT INTO teams(id,workspace_id,name) VALUES($1,$2,$3)", id, in.WorkspaceID, strings.TrimSpace(in.Name))
	if teamNameConflict(w, e) {
		return
	}
	if e == nil {
		s.audit(r, "TEAM_CREATE", id, nil)
	}
	respond(w, map[string]any{"id": id, "name": strings.TrimSpace(in.Name)}, e)
}
func (s *Server) teamWorkspace(r *http.Request) (string, error) {
	var wid string
	e := s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM teams WHERE id=$1", r.PathValue("id")).Scan(&wid)
	return wid, e
}
func (s *Server) updateTeam(w http.ResponseWriter, r *http.Request) {
	wid, e := s.teamWorkspace(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "팀 관리 권한이 없습니다")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if decode(r, &in) != nil || strings.TrimSpace(in.Name) == "" || len(in.Name) > 200 {
		apiError(w, 400, "팀 이름을 확인하세요")
		return
	}
	_, e = s.DB.Exec(r.Context(), "UPDATE teams SET name=$2 WHERE id=$1", r.PathValue("id"), strings.TrimSpace(in.Name))
	if teamNameConflict(w, e) {
		return
	}
	if e == nil {
		s.audit(r, "TEAM_UPDATE", r.PathValue("id"), nil)
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) deleteTeam(w http.ResponseWriter, r *http.Request) {
	wid, e := s.teamWorkspace(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "팀 관리 권한이 없습니다")
		return
	}
	_, e = s.DB.Exec(r.Context(), "DELETE FROM teams WHERE id=$1", r.PathValue("id"))
	if e == nil {
		s.audit(r, "TEAM_DELETE", r.PathValue("id"), nil)
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func teamNameConflict(w http.ResponseWriter, e error) bool {
	var p *pgconn.PgError
	if errors.As(e, &p) && p.Code == "23505" {
		apiError(w, 409, "같은 이름의 팀이 이미 있습니다")
		return true
	}
	return false
}
func (s *Server) teamMembers(w http.ResponseWriter, r *http.Request) {
	wid, e := s.teamWorkspace(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',u.id,'name',u.name,'email',u.email) FROM team_members m JOIN users u ON u.id=m.user_id JOIN workspace_members wm ON wm.user_id=u.id AND wm.workspace_id=$2 WHERE m.team_id=$1 AND NOT u.disabled ORDER BY u.name", r.PathValue("id"), wid)
	respond(w, v, e)
}
func (s *Server) setTeamMember(w http.ResponseWriter, r *http.Request) {
	wid, e := s.teamWorkspace(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "팀 관리 권한이 없습니다")
		return
	}
	var in struct {
		UserID string `json:"user_id"`
		Remove bool   `json:"remove"`
	}
	if decode(r, &in) != nil || !validID(in.UserID) {
		apiError(w, 400, "팀 멤버를 선택하세요")
		return
	}
	if in.Remove {
		_, e = s.DB.Exec(r.Context(), "DELETE FROM team_members WHERE team_id=$1 AND user_id=$2", r.PathValue("id"), in.UserID)
	} else {
		var tag int64
		result, err := s.DB.Exec(r.Context(), "INSERT INTO team_members(team_id,user_id) SELECT $1,u.id FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$2 AND m.workspace_id=$3 AND NOT u.disabled ON CONFLICT DO NOTHING", r.PathValue("id"), in.UserID, wid)
		e = err
		if e == nil {
			tag = result.RowsAffected()
		}
		if e == nil && tag == 0 {
			var exists bool
			e = s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id=$1 AND user_id=$2)", r.PathValue("id"), in.UserID).Scan(&exists)
			if e == nil && !exists {
				apiError(w, 400, "활성 워크스페이스 멤버만 추가할 수 있습니다")
				return
			}
		}
	}
	if e == nil {
		s.audit(r, "TEAM_MEMBERS_CHANGE", r.PathValue("id"), map[string]any{"user_id": in.UserID, "remove": in.Remove})
	}
	respond(w, map[string]bool{"ok": true}, e)
}
