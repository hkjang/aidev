package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/realtime"
	"github.com/hkjang/umm/internal/store"
)

// spaceEvents streams the collaboration log. The reader is woken by the
// realtime hub's PostgreSQL LISTEN connection, so an idle space costs no
// queries at all. The ticker is only a safety net: it runs slowly while the
// listener is healthy and takes over at the old poll rate if it is not.
// spaceRewind answers what a space looked like at an instant.
//
// The material has been accumulating since revisions were introduced — every
// note update snapshots the state it replaces, positions included — and nothing
// had ever read it this way. NoteHistory answers for one note; this answers the
// question people actually ask, which is what the whole canvas looked like.
//
// Connections removed since cannot be drawn: a deletion event has only ever
// carried the edge's id, and the row is gone. The count travels with the
// answer so the screen can say so rather than showing a canvas that quietly
// omits them.
func (s *Server) spaceRewind(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	at := time.Now()
	if raw := strings.TrimSpace(r.URL.Query().Get("at")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			// Refused rather than quietly answered for now: someone who asked
			// about June and was shown today would not know they had been.
			writeError(w, 400, "시점을 RFC3339 형식으로 지정해 주세요.")
			return
		}
		at = parsed
	}

	p := principal(r)
	snapshot, err := s.Store.SpaceAt(r.Context(), p.User.ID, spaceID, at)
	if err != nil {
		if errors.Is(err, store.ErrSpaceNotVisible) {
			writeError(w, 404, "공간을 찾을 수 없습니다.")
			return
		}
		slog.Warn("space rewind failed", "space_id", spaceID, "user_id", p.User.ID, "error", err)
		writeError(w, 500, "그때의 공간을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, snapshot)
}

func (s *Server) spaceEvents(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	p := principal(r)
	if !s.Store.CanViewSpace(r.Context(), p.User.ID, spaceID) {
		writeError(w, 404, "공간을 찾을 수 없습니다.")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "실시간 전송을 지원하지 않습니다.")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	last, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)
	if queryLast, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64); queryLast > last {
		last = queryLast
	}
	if last == 0 && r.Header.Get("Last-Event-ID") == "" && r.URL.Query().Get("after") == "" {
		_ = s.Store.Pool.QueryRow(r.Context(), `SELECT COALESCE(max(sequence),0) FROM space_events WHERE space_id=$1`, spaceID).Scan(&last)
	}
	// Subscribing before the first drain closes the race where an event is
	// committed between reading the cursor above and registering here.
	var pushed <-chan struct{}
	if s.Events != nil {
		subscription := s.Events.Subscribe(spaceID)
		defer subscription.Close()
		pushed = subscription.C()
	}
	ticker := time.NewTicker(sseFallbackInterval(s.Events))
	heartbeat := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()
	fmt.Fprint(w, ": umm collaboration stream\n\n")
	flusher.Flush()
	// Each batch re-checks membership inside the same statement, so a stream
	// cannot keep delivering events after its reader loses access to the space.
	drain := func() (wrote bool, keepStreaming bool) {
		events, allowed, err := s.Store.SpaceEvents(r.Context(), p.User.ID, spaceID, last, 100)
		if err != nil {
			return false, true
		}
		if !allowed {
			return false, false
		}
		for _, item := range events {
			event, _ := json.Marshal(item)
			fmt.Fprintf(w, "id: %d\nevent: space-change\ndata: %s\n\n", item.Sequence, event)
			last = item.Sequence
			wrote = true
		}
		return wrote, true
	}
	// One batch is bounded, so keep draining until the query comes back empty
	// before parking on the next signal. Returns false once the reader has lost
	// access and the stream must end.
	drainAll := func() bool {
		for {
			wrote, keepStreaming := drain()
			if !keepStreaming {
				return false
			}
			flusher.Flush()
			if !wrote {
				return true
			}
		}
	}
	if !drainAll() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case <-pushed:
			if !drainAll() {
				return
			}
			// Listener state transitions use the same coalesced wake-up as
			// committed events. Refreshing here switches a disconnected stream
			// from the 30-second safety net to one-second polling immediately,
			// and restores the slow interval as soon as LISTEN reconnects.
			ticker.Reset(sseFallbackInterval(s.Events))
		case <-ticker.C:
			ticker.Reset(sseFallbackInterval(s.Events))
			if !drainAll() {
				return
			}
		}
	}
}

// sseFallbackInterval keeps the pre-v0.8 one second poll only while the
// PostgreSQL listener is unavailable. A healthy listener needs the timer purely
// as a backstop against a missed notification.
func sseFallbackInterval(hub *realtime.Hub) time.Duration {
	if hub != nil && hub.Listening() {
		return 30 * time.Second
	}
	return time.Second
}

func (s *Server) listSpaceMembers(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "spaces:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	p := principal(r)
	rows, err := s.Store.Pool.Query(r.Context(), `
		WITH authorized_space AS (
		  SELECT sp.id,sp.owner_id FROM spaces sp
		  LEFT JOIN space_members caller ON caller.space_id=sp.id AND caller.user_id=$2
		  WHERE sp.id=$1 AND (sp.owner_id=$2 OR caller.user_id=$2)
		)
		SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),'owner'::text
		FROM authorized_space asp JOIN users u ON u.id=asp.owner_id
		UNION ALL
		SELECT u.id,u.username,u.display_name,COALESCE(u.email,''),member.permission
		FROM authorized_space asp
		JOIN space_members member ON member.space_id=asp.id
		JOIN users u ON u.id=member.user_id
		ORDER BY 5,3`, spaceID, p.User.ID)
	if err != nil {
		writeError(w, 500, "공유 사용자를 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	members := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var username, name, email, permission string
		if err := rows.Scan(&id, &username, &name, &email, &permission); err != nil {
			writeError(w, 500, "공유 사용자를 읽지 못했습니다.")
			return
		}
		members = append(members, map[string]any{"id": id, "username": username, "displayName": name, "email": email, "permission": permission})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "공유 사용자를 불러오지 못했습니다.")
		return
	}
	if len(members) == 0 {
		writeError(w, 404, "공간을 찾을 수 없습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"members": members, "canManage": s.canManageSpace(r, p.User.ID, spaceID)})
}

func (s *Server) canManageSpace(r *http.Request, userID, spaceID uuid.UUID) bool {
	var ok bool
	_ = s.Store.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM spaces sp LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$1 WHERE sp.id=$2 AND (sp.owner_id=$1 OR sm.permission='manage'))`, userID, spaceID).Scan(&ok)
	return ok
}

func (s *Server) shareSpace(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	p := principal(r)
	if !s.canManageSpace(r, p.User.ID, spaceID) {
		writeError(w, 403, "공간 공유를 관리할 권한이 없습니다.")
		return
	}
	var body struct {
		Username   string `json:"username"`
		Permission string `json:"permission"`
	}
	if decodeJSON(w, r, &body) != nil || !slices.Contains([]string{"view", "edit", "manage"}, body.Permission) {
		writeError(w, 400, "공유 설정이 올바르지 않습니다.")
		return
	}
	var targetID uuid.UUID
	var targetName string
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT id,display_name FROM users WHERE username=$1 AND active`, strings.TrimSpace(body.Username)).Scan(&targetID, &targetName); err != nil {
		writeError(w, 404, "활성 사용자를 찾을 수 없습니다.")
		return
	}
	if targetID == p.User.ID {
		writeError(w, 400, "자기 자신을 공유 사용자로 추가할 수 없습니다.")
		return
	}
	payload := map[string]any{"targetUserId": targetID, "permission": body.Permission, "targetName": targetName}
	var cfg workflowConfig
	_ = s.Store.GetSetting(r.Context(), "workflow", &cfg)
	if cfg.Enabled && slices.Contains(cfg.Actions, "space_share") {
		raw, _ := json.Marshal(payload)
		var requestID uuid.UUID
		err := s.Store.Pool.QueryRow(r.Context(), `INSERT INTO approval_requests(requester_id,team_id,resource_type,resource_id,action,comment,payload) VALUES($1,$2,'space',$3,'space_share',$4,$5) RETURNING id`, p.User.ID, p.User.TeamID, spaceID, "공간 공유: "+targetName+" ("+body.Permission+")", raw).Scan(&requestID)
		if err != nil {
			writeError(w, 500, "공유 승인 요청을 만들지 못했습니다.")
			return
		}
		s.Store.Audit(r.Context(), &p.User.ID, "space.share.request", "approval", requestID.String(), payload)
		writeJSON(w, 202, map[string]any{"required": true, "requestId": requestID, "status": "pending"})
		return
	}
	if err := s.applySpaceShare(r, spaceID, targetID, body.Permission, payload); err != nil {
		writeError(w, 500, "공간을 공유하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "space.share", "space", spaceID.String(), payload)
	writeJSON(w, 201, map[string]any{"required": false, "status": "shared"})
}

func (s *Server) applySpaceShare(r *http.Request, spaceID, targetID uuid.UUID, permission string, payload any) error {
	actorID := principal(r).User.ID
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	command, err := tx.Exec(r.Context(), `
		INSERT INTO space_members(space_id,user_id,permission)
		SELECT $1,$2,$3 WHERE EXISTS(
		  SELECT 1 FROM spaces sp LEFT JOIN space_members actor_member ON actor_member.space_id=sp.id AND actor_member.user_id=$4
		  WHERE sp.id=$1 AND (sp.owner_id=$4 OR actor_member.permission='manage'))
		ON CONFLICT(space_id,user_id) DO UPDATE SET permission=EXCLUDED.permission`, spaceID, targetID, permission, actorID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("space share permission changed before commit")
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO notifications(user_id,kind,title,body,resource_type,resource_id,resource_space_id) VALUES($1,'space_shared','새 공간이 공유되었습니다','공유된 공간을 열어 생각을 함께 발전시켜 보세요.','space',$2,$2)`, targetID, spaceID); err != nil {
		return err
	}
	if err = s.Store.AppendSpaceEvent(r.Context(), tx, actorID, spaceID, "member.updated", targetID, payload); err != nil {
		return err
	}
	return tx.Commit(r.Context())
}

func (s *Server) removeSpaceMember(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	memberID, ok := parseID(w, r, "memberID")
	if !ok {
		return
	}
	p := principal(r)
	if !s.canManageSpace(r, p.User.ID, spaceID) {
		writeError(w, 403, "공간 공유를 관리할 권한이 없습니다.")
		return
	}
	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "공유 사용자를 제거하지 못했습니다.")
		return
	}
	defer tx.Rollback(r.Context())
	cmd, err := tx.Exec(r.Context(), `
		DELETE FROM space_members target USING spaces sp
		LEFT JOIN space_members actor_member ON actor_member.space_id=sp.id AND actor_member.user_id=$3
		WHERE target.space_id=$1 AND target.user_id=$2 AND sp.id=target.space_id
		  AND (sp.owner_id=$3 OR actor_member.permission='manage')`, spaceID, memberID, p.User.ID)
	if err != nil {
		writeError(w, 500, "공유 사용자를 제거하지 못했습니다.")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, 404, "공유 사용자를 찾을 수 없습니다.")
		return
	}
	if err = s.Store.AppendSpaceEvent(r.Context(), tx, p.User.ID, spaceID, "member.removed", memberID, map[string]any{}); err != nil {
		writeError(w, 500, "공유 변경 이벤트를 저장하지 못했습니다.")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "공유 사용자 제거를 확정하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "space.unshare", "space", spaceID.String(), map[string]any{"userId": memberID})
	w.WriteHeader(http.StatusNoContent)
}
