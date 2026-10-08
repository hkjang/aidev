package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Spaces, seen from the outside.
//
// An administrator could see how many spaces existed and nothing else — not who
// owned them, not who could reach them. That is fine until somebody leaves.
// Deactivating an account does not touch what it owns, so their spaces stay
// theirs, reachable by whoever was shared in and by nobody else, with no way to
// hand them on. The count on the metrics screen said everything was fine.
func (s *Server) adminSpaces(w http.ResponseWriter, r *http.Request) {
	offset, ok := decodeOffsetCursor(r.URL.Query().Get("cursor"))
	if !ok {
		writeError(w, 400, "공간 목록 커서가 올바르지 않습니다.")
		return
	}
	limit := parsePageLimit(r, 50, 200)

	where := []string{}
	args := []any{}
	query := r.URL.Query()
	if owner := strings.TrimSpace(query.Get("owner")); owner != "" {
		args = append(args, owner)
		where = append(where, "u.username = $1::citext")
	}
	// The question this exists for: what belongs to someone who has left.
	if query.Get("ownerInactive") == "true" {
		where = append(where, "NOT u.active")
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, limit+1, offset)

	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT sp.id, sp.name, sp.is_inbox, sp.created_at,
		       u.id, u.username, u.active,
		       (SELECT count(*) FROM space_members m WHERE m.space_id=sp.id),
		       (SELECT count(*) FROM notes n WHERE n.space_id=sp.id AND n.deleted_at IS NULL)
		FROM spaces sp JOIN users u ON u.id=sp.owner_id`+clause+`
		ORDER BY NOT u.active DESC, sp.created_at DESC, sp.id DESC
		`+fmt.Sprintf("LIMIT $%d OFFSET $%d", len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, 500, "공간 목록을 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, ownerID uuid.UUID
		var name, ownerName string
		var isInbox, ownerActive bool
		var createdAt any
		var members, notes int64
		if err := rows.Scan(&id, &name, &isInbox, &createdAt, &ownerID, &ownerName, &ownerActive, &members, &notes); err != nil {
			writeError(w, 500, "공간 목록을 읽지 못했습니다.")
			return
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "isInbox": isInbox, "createdAt": createdAt,
			"ownerId": ownerID, "owner": ownerName, "ownerActive": ownerActive,
			"members": members, "notes": notes,
		})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "공간 목록을 불러오지 못했습니다.")
		return
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = encodeOffsetCursor(offset + limit)
	}
	writeJSON(w, 200, map[string]any{"spaces": out, "nextCursor": next})
}

// adminSpaceMembers is who can reach one space, and how.
func (s *Server) adminSpaceMembers(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT u.id, u.username, u.active, 'owner'::text FROM spaces sp JOIN users u ON u.id=sp.owner_id WHERE sp.id=$1
		UNION ALL
		SELECT u.id, u.username, u.active, m.permission FROM space_members m JOIN users u ON u.id=m.user_id WHERE m.space_id=$1
		ORDER BY 4, 2`, spaceID)
	if err != nil {
		writeError(w, 500, "공간 참여자를 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var username, permission string
		var active bool
		if err := rows.Scan(&id, &username, &active, &permission); err != nil {
			writeError(w, 500, "공간 참여자를 읽지 못했습니다.")
			return
		}
		out = append(out, map[string]any{"id": id, "username": username, "active": active, "permission": permission})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "공간 참여자를 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"members": out})
}

// transferSpaceOwner hands a space to someone who is still here.
//
// The person who leaves keeps their access when they are still active — an
// administrator moving a space should not quietly take it away from whoever was
// using it. When the old owner is deactivated they get nothing, which is the
// case this exists for.
//
// An inbox is refused. Every person has exactly one, it is where their
// unfiled thoughts land, and handing it to someone else would give away a
// private drawer and leave the owner's next capture with nowhere to go.
func (s *Server) transferSpaceOwner(w http.ResponseWriter, r *http.Request) {
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	var body struct {
		UserID uuid.UUID `json:"userId"`
	}
	if decodeJSON(w, r, &body) != nil || body.UserID == uuid.Nil {
		writeError(w, 400, "넘겨받을 사용자를 지정해 주세요.")
		return
	}
	p := principal(r)

	tx, err := s.Store.Pool.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "공간 소유자를 바꾸지 못했습니다.")
		return
	}
	defer tx.Rollback(r.Context())

	var previous uuid.UUID
	var isInbox bool
	err = tx.QueryRow(r.Context(), `SELECT owner_id, is_inbox FROM spaces WHERE id=$1 FOR UPDATE`, spaceID).Scan(&previous, &isInbox)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "공간을 찾을 수 없습니다.")
		return
	}
	if err != nil {
		writeError(w, 500, "공간을 읽지 못했습니다.")
		return
	}
	if isInbox {
		writeError(w, 400, "수집함은 넘길 수 없습니다. 사람마다 하나씩 있는 개인 공간입니다.")
		return
	}
	if previous == body.UserID {
		writeError(w, 400, "이미 이 사람의 공간입니다.")
		return
	}

	var newOwnerActive bool
	err = tx.QueryRow(r.Context(), `SELECT active FROM users WHERE id=$1`, body.UserID).Scan(&newOwnerActive)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "넘겨받을 사용자를 찾을 수 없습니다.")
		return
	}
	if err != nil {
		writeError(w, 500, "사용자를 읽지 못했습니다.")
		return
	}
	if !newOwnerActive {
		writeError(w, 400, "비활성 사용자에게는 넘길 수 없습니다. 공간이 다시 갈 곳을 잃습니다.")
		return
	}

	if _, err = tx.Exec(r.Context(), `UPDATE spaces SET owner_id=$2, updated_at=now() WHERE id=$1`, spaceID, body.UserID); err != nil {
		writeError(w, 500, "공간 소유자를 바꾸지 못했습니다.")
		return
	}
	// Ownership is not a membership row, so the new owner's old one would sit
	// there saying something weaker than the truth.
	if _, err = tx.Exec(r.Context(), `DELETE FROM space_members WHERE space_id=$1 AND user_id=$2`, spaceID, body.UserID); err != nil {
		writeError(w, 500, "이전 참여 기록을 정리하지 못했습니다.")
		return
	}
	keptPrevious := false
	var previousActive bool
	if err = tx.QueryRow(r.Context(), `SELECT active FROM users WHERE id=$1`, previous).Scan(&previousActive); err == nil && previousActive {
		if _, err = tx.Exec(r.Context(),
			`INSERT INTO space_members(space_id,user_id,permission) VALUES($1,$2,'manage')
			 ON CONFLICT(space_id,user_id) DO UPDATE SET permission='manage'`, spaceID, previous); err != nil {
			writeError(w, 500, "이전 소유자의 접근 권한을 남기지 못했습니다.")
			return
		}
		keptPrevious = true
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "공간 소유자 변경을 확정하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "space.transfer", "space", spaceID.String(),
		map[string]any{"from": previous.String(), "to": body.UserID.String(), "previousKeptAccess": keptPrevious})
	writeJSON(w, 200, map[string]any{"ownerId": body.UserID, "previousKeptAccess": keptPrevious})
}
