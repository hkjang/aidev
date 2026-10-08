package httpapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

const notificationSpaceExpression = `COALESCE(
	n.resource_space_id,
	CASE WHEN n.resource_type='space' THEN n.resource_id END
)`

const accessibleNotificationPredicate = `n.user_id=$1 AND (
	(n.resource_type='note' AND EXISTS(
		SELECT 1 FROM notes notification_note
		JOIN spaces notification_space ON notification_space.id=notification_note.space_id
		LEFT JOIN space_members notification_member
		  ON notification_member.space_id=notification_space.id AND notification_member.user_id=$1
		WHERE notification_note.id=n.resource_id AND notification_note.deleted_at IS NULL
		  AND (notification_space.owner_id=$1 OR notification_member.user_id=$1)
	)) OR
	(n.resource_type='dream' AND (
		n.resource_id IS NULL OR EXISTS(
			SELECT 1 FROM dream_notes notification_dream
			JOIN spaces notification_space ON notification_space.id=notification_dream.space_id
			LEFT JOIN space_members notification_member
			  ON notification_member.space_id=notification_space.id AND notification_member.user_id=$1
			WHERE notification_dream.dream_id=n.resource_id AND notification_dream.user_id=$1
			  AND (notification_space.owner_id=$1 OR notification_member.user_id=$1)
		)
	)) OR
	(n.resource_type NOT IN ('note','dream') AND (
		` + notificationSpaceExpression + ` IS NULL OR EXISTS(
			SELECT 1 FROM spaces sp
			LEFT JOIN space_members sm ON sm.space_id=sp.id AND sm.user_id=$1
			WHERE sp.id=` + notificationSpaceExpression + ` AND (sp.owner_id=$1 OR sm.user_id=$1)
		)
	))
)`

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	p := principal(r)
	offset, ok := decodeOffsetCursor(r.URL.Query().Get("cursor"))
	if !ok {
		writeError(w, 400, "알림 커서가 올바르지 않습니다.")
		return
	}
	limit := parsePageLimit(r, 30, 100)
	var unread int64
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT count(*) FROM notifications n WHERE `+accessibleNotificationPredicate+` AND n.read_at IS NULL`, p.User.ID).Scan(&unread); err != nil {
		writeError(w, 500, "읽지 않은 알림 수를 불러오지 못했습니다.")
		return
	}
	rows, err := s.Store.Pool.Query(r.Context(), `SELECT n.id,n.kind,n.title,n.body,n.resource_type,n.resource_id,n.resource_space_id,n.metadata,n.read_at,n.created_at FROM notifications n WHERE `+accessibleNotificationPredicate+` ORDER BY n.created_at DESC,n.id DESC LIMIT $2 OFFSET $3`, p.User.ID, limit+1, offset)
	if err != nil {
		writeError(w, 500, "알림을 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var kind, title, body, resourceType string
		var resourceID, resourceSpaceID *uuid.UUID
		var metadata map[string]any
		var readAt *time.Time
		var created time.Time
		if err := rows.Scan(&id, &kind, &title, &body, &resourceType, &resourceID, &resourceSpaceID, &metadata, &readAt, &created); err != nil {
			writeError(w, 500, "알림을 읽지 못했습니다.")
			return
		}
		items = append(items, map[string]any{"id": id, "kind": kind, "title": title, "body": body, "resourceType": resourceType, "resourceId": resourceID, "resourceSpaceId": resourceSpaceID, "metadata": metadata, "readAt": readAt, "createdAt": created})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "알림을 불러오지 못했습니다.")
		return
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = encodeOffsetCursor(offset + limit)
	}
	writeJSON(w, 200, map[string]any{"notifications": items, "unread": unread, "nextCursor": next})
}
func (s *Server) readNotification(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	id, ok := parseID(w, r, "notificationID")
	if !ok {
		return
	}
	p := principal(r)
	cmd, err := s.Store.Pool.Exec(r.Context(), `UPDATE notifications SET read_at=COALESCE(read_at,now()) WHERE id=$1 AND user_id=$2`, id, p.User.ID)
	if err != nil {
		writeError(w, 500, "알림을 읽음으로 표시하지 못했습니다.")
		return
	}
	if cmd.RowsAffected() == 0 {
		writeError(w, 404, "알림을 찾을 수 없습니다.")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
