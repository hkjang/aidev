package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
)

type workflowConfig struct {
	Enabled bool     `json:"enabled"`
	Actions []string `json:"actions"`
}

func (s *Server) createApproval(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "approvals:write") {
		return
	}
	var body struct {
		ResourceType string    `json:"resourceType"`
		ResourceID   uuid.UUID `json:"resourceId"`
		Action       string    `json:"action"`
		Comment      string    `json:"comment"`
	}
	if decodeJSON(w, r, &body) != nil || body.ResourceID == uuid.Nil || strings.TrimSpace(body.Action) == "" {
		writeError(w, 400, "검토 요청 형식이 올바르지 않습니다.")
		return
	}
	var cfg workflowConfig
	_ = s.Store.GetSetting(r.Context(), "workflow", &cfg)
	if !cfg.Enabled || !slices.Contains(cfg.Actions, body.Action) {
		writeJSON(w, 200, map[string]any{"required": false, "status": "approved", "message": "관리자가 이 작업에 검토 절차를 설정하지 않았습니다."})
		return
	}
	p := principal(r)
	var id uuid.UUID
	err := s.Store.Pool.QueryRow(r.Context(), `INSERT INTO approval_requests(requester_id,team_id,resource_type,resource_id,action,comment) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, p.User.ID, p.User.TeamID, body.ResourceType, body.ResourceID, body.Action, body.Comment).Scan(&id)
	if err != nil {
		writeError(w, 500, "검토 요청을 만들지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "approval.request", "approval", id.String(), map[string]any{"action": body.Action})
	writeJSON(w, 201, map[string]any{"required": true, "id": id, "status": "pending"})
}

func (s *Server) listApprovals(w http.ResponseWriter, r *http.Request) {
	p := principal(r)

	// The reviewer is being asked to allow something to happen to a particular
	// space, so the query says which one. Without it the page could only offer
	// "export · space", and approving an export without knowing what is being
	// exported is the one thing the reviewer is there to judge.
	//
	// LEFT JOIN because a request outlives its subject: a space deleted while a
	// request is still open must leave the request listed and reviewable, not
	// drop it from the page.
	const columns = `SELECT a.id,a.requester_id,a.team_id,a.resource_type,a.resource_id,a.action,a.status,
		       a.comment,a.reviewer_id,a.reviewed_at,a.created_at,u.display_name,COALESCE(sp.name,'')
		FROM approval_requests a
		JOIN users u ON u.id=a.requester_id
		LEFT JOIN spaces sp ON a.resource_type='space' AND sp.id=a.resource_id `

	// Pending first for the people who can act on it; a reviewer opens this page
	// to find what is waiting, not to read history.
	where, order := `WHERE a.requester_id=$1`, `ORDER BY a.created_at DESC`
	args := []any{p.User.ID}
	switch {
	case p.User.Role == "admin":
		where, order, args = ``, `ORDER BY (a.status='pending') DESC,a.created_at DESC`, nil
	case p.User.Role == "team_lead" && p.User.TeamID != nil:
		where = `WHERE a.team_id=$1 OR a.requester_id=$2`
		order = `ORDER BY (a.status='pending') DESC,a.created_at DESC`
		args = []any{p.User.TeamID, p.User.ID}
	}

	rows, err := s.Store.Pool.Query(r.Context(), columns+where+" "+order, args...)
	if err != nil {
		writeError(w, 500, "검토 요청을 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, requester, resource uuid.UUID
		var team, reviewer *uuid.UUID
		var resourceType, action, status, comment, name, resourceName string
		var reviewed *time.Time
		var created time.Time
		if rows.Scan(&id, &requester, &team, &resourceType, &resource, &action, &status, &comment, &reviewer, &reviewed, &created, &name, &resourceName) != nil {
			continue
		}
		out = append(out, map[string]any{"id": id, "requesterId": requester, "requesterName": name, "teamId": team, "resourceType": resourceType, "resourceId": resource, "resourceName": resourceName, "action": action, "status": status, "comment": comment, "reviewerId": reviewer, "reviewedAt": reviewed, "createdAt": created})
	}
	writeJSON(w, 200, map[string]any{"requests": out})
}

func (s *Server) decideApproval(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "requestID")
	if !ok {
		return
	}
	var body struct {
		Decision string `json:"decision"`
		Comment  string `json:"comment"`
	}
	if decodeJSON(w, r, &body) != nil || !slices.Contains([]string{"approved", "rejected"}, body.Decision) {
		writeError(w, 400, "승인 또는 반려를 선택해 주세요.")
		return
	}
	p := principal(r)
	if p.User.Role != "admin" && p.User.Role != "team_lead" {
		writeError(w, 403, "팀장 또는 관리자만 검토할 수 있습니다.")
		return
	}
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "검토 처리를 시작하지 못했습니다.")
		return
	}
	defer tx.Rollback(r.Context())
	var action string
	var resourceID, requesterID uuid.UUID
	var requestTeam *uuid.UUID
	var payloadRaw json.RawMessage
	err = tx.QueryRow(r.Context(), `SELECT action,resource_id,payload,requester_id,team_id FROM approval_requests WHERE id=$1 AND status='pending' FOR UPDATE`, id).Scan(&action, &resourceID, &payloadRaw, &requesterID, &requestTeam)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "처리할 검토 요청을 찾을 수 없습니다.")
		return
	}
	if err != nil {
		writeError(w, 500, "검토 요청을 처리하지 못했습니다.")
		return
	}
	if p.User.Role == "team_lead" {
		if p.User.TeamID == nil || requestTeam == nil || *requestTeam != *p.User.TeamID || requesterID == p.User.ID {
			writeError(w, 403, "자신의 요청 또는 다른 팀의 요청은 검토할 수 없습니다.")
			return
		}
	}
	var sharePayload struct {
		TargetUserID uuid.UUID `json:"targetUserId"`
		Permission   string    `json:"permission"`
	}
	if body.Decision == "approved" && action == "space_share" {
		if json.Unmarshal(payloadRaw, &sharePayload) != nil || sharePayload.TargetUserID == uuid.Nil || !slices.Contains([]string{"view", "edit", "manage"}, sharePayload.Permission) {
			writeError(w, 500, "승인 대상 공유 정보가 올바르지 않습니다.")
			return
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,$3) ON CONFLICT(space_id,user_id) DO UPDATE SET permission=EXCLUDED.permission`, resourceID, sharePayload.TargetUserID, sharePayload.Permission); err != nil {
			writeError(w, 500, "공간 공유 승인을 적용하지 못했습니다.")
			return
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,title,body,resource_type,resource_id,resource_space_id) VALUES($1,'space_shared','새 공간이 공유되었습니다','공유된 공간을 열어 생각을 함께 발전시켜 보세요.','space',$2,$2)`, sharePayload.TargetUserID, resourceID); err != nil {
			writeError(w, 500, "공간 공유 알림을 저장하지 못했습니다.")
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `UPDATE approval_requests SET status=$2,comment=CASE WHEN $3='' THEN comment ELSE $3 END,reviewer_id=$4,reviewed_at=now() WHERE id=$1`, id, body.Decision, body.Comment, p.User.ID); err != nil {
		writeError(w, 500, "검토 결과를 저장하지 못했습니다.")
		return
	}
	if body.Decision == "approved" && action == "space_share" {
		if err = s.Store.AppendSpaceEvent(r.Context(), tx, p.User.ID, resourceID, "member.updated", sharePayload.TargetUserID, sharePayload); err != nil {
			writeError(w, 500, "공유 변경 이벤트를 저장하지 못했습니다.")
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "검토 결과를 확정하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "approval."+body.Decision, "approval", id.String(), map[string]any{})
	writeJSON(w, 200, map[string]string{"status": body.Decision})
}
