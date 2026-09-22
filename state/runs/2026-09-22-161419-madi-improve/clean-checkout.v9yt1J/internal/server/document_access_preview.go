package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type documentAccessPlacement struct {
	Parent     string `json:"parent_id"`
	Space      string `json:"space_id"`
	Visibility string `json:"visibility"`
}
type documentAccessPreviewTicket struct {
	Kind        string                  `json:"kind"`
	Actor       string                  `json:"actor"`
	Document    string                  `json:"document"`
	Version     int                     `json:"version"`
	Placement   documentAccessPlacement `json:"placement"`
	Fingerprint string                  `json:"fingerprint"`
	Expires     int64                   `json:"expires"`
}

func (s *Server) registerDocumentAccessPreview() {
	s.handle("POST /api/v1/documents/{id}/access-preview", s.documentAccessPreview)
}
func accessPlacement(in map[string]any, old map[string]any) (documentAccessPlacement, error) {
	p := documentAccessPlacement{str(old, "parent_id"), str(old, "space_id"), str(old, "visibility")}
	for name, dest := range map[string]*string{"parent_id": &p.Parent, "space_id": &p.Space, "visibility": &p.Visibility} {
		if v, ok := in[name]; ok {
			if v == nil && name != "visibility" {
				*dest = ""
			} else if text, ok := v.(string); ok {
				*dest = text
			} else {
				return p, errors.New("공유 범위와 이동 대상 형식을 확인하세요")
			}
		}
	}
	if p.Parent != "" && !validID(p.Parent) || p.Space != "" && !validID(p.Space) || !oneOf(p.Visibility, "private", "selected", "workspace") {
		return p, errors.New("공유 범위와 이동 대상 형식을 확인하세요")
	}
	return p, nil
}

// This is metadata only. No source body, secret, hidden descendant count or
// projected list of users is returned to the browser. Hash covers the old/new
// ancestor policies, their direct shares and current workspace role ceilings.
const accessPreviewFingerprintSQL = `WITH RECURSIVE dc AS (
 SELECT d.id,d.parent_id,d.space_id,ARRAY[d.id] path FROM documents d WHERE d.id=$1 OR d.id=NULLIF($2,'')::uuid
 UNION ALL SELECT d.id,d.parent_id,d.space_id,c.path||d.id FROM documents d JOIN dc c ON d.id=c.parent_id WHERE NOT d.id=ANY(c.path) AND cardinality(c.path)<21
), sc AS (
 SELECT s.id,s.parent_id,ARRAY[s.id] path FROM spaces s WHERE s.id=NULLIF($3,'')::uuid OR s.id IN(SELECT space_id FROM dc)
 UNION ALL SELECT s.id,s.parent_id,c.path||s.id FROM spaces s JOIN sc c ON s.id=c.parent_id WHERE NOT s.id=ANY(c.path) AND cardinality(c.path)<21
), snapshot AS (SELECT jsonb_build_object(
 'documents',(SELECT coalesce(jsonb_agg(jsonb_build_array(d.id,d.workspace_id,d.parent_id,d.space_id,d.visibility,d.owner_id,d.version,d.deleted_at) ORDER BY d.id),'[]') FROM documents d WHERE d.id IN(SELECT id FROM dc)),
 'shares',(SELECT coalesce(jsonb_agg(jsonb_build_array(document_id,user_id,permission) ORDER BY document_id,user_id),'[]') FROM document_shares WHERE document_id IN(SELECT id FROM dc)),
 'spaces',(SELECT coalesce(jsonb_agg(jsonb_build_array(s.id,s.workspace_id,s.parent_id,s.visibility,s.classification) ORDER BY s.id),'[]') FROM spaces s WHERE s.id IN(SELECT id FROM sc)),
 'space_members',(SELECT coalesce(jsonb_agg(jsonb_build_array(space_id,user_id,role) ORDER BY space_id,user_id),'[]') FROM space_members WHERE space_id IN(SELECT id FROM sc)),
 'members',(SELECT coalesce(jsonb_agg(jsonb_build_array(m.user_id,m.role,u.role,u.kind,u.disabled) ORDER BY m.user_id),'[]') FROM workspace_members m JOIN users u ON u.id=m.user_id WHERE m.workspace_id=$4)
 ) value)
 SELECT CASE WHEN octet_length(value::text)<=4194304 THEN value::text ELSE NULL END FROM snapshot`

func accessPreviewFingerprint(r *http.Request, q databaseQuerier, id, wid string, p documentAccessPlacement) (string, error) {
	var raw string
	if e := q.QueryRow(r.Context(), accessPreviewFingerprintSQL, id, p.Parent, p.Space, wid).Scan(&raw); e != nil {
		return "", errors.New("공유 정책의 확인 범위를 읽을 수 없습니다. 현재 상태를 다시 확인하세요")
	}
	return digest(raw), nil
}

func (s *Server) documentAccessPreview(w http.ResponseWriter, r *http.Request) {
	p, id := current(r), r.PathValue("id")
	if !personalAccessRequest(p) || !s.canDocument(r.Context(), p, id, true) {
		apiError(w, 403, "개인 브라우저의 현재 문서 소유자만 공유·이동을 미리 확인할 수 있습니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "공유·이동 입력값을 확인하세요")
		return
	}
	doc, e := s.document(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if str(doc, "owner_id") != p.ID || doc["deleted_at"] != nil {
		apiError(w, 403, "현재 문서 소유권이 필요합니다")
		return
	}
	expected := number(in, "expected_version", 0)
	if expected < 1 || expected != number(doc, "version", 0) {
		apiError(w, 409, "문서 버전이 변경되었습니다")
		return
	}
	placement, e := accessPlacement(in, doc)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	wid := str(doc, "workspace_id")
	if !s.validParent(r, id, wid, placement.Parent) || placement.Parent != "" && !s.canDocument(r.Context(), p, placement.Parent, true) || !s.canSpace(r.Context(), p, wid, placement.Space, true) {
		apiError(w, 403, "현재 작성 가능한 상위 문서·공간을 선택하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = documentAccessActorTx(r, tx, wid, true); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	fingerprint, e := accessPreviewFingerprint(r, tx, id, wid, placement)
	if e != nil {
		apiError(w, 409, e.Error())
		return
	}
	parentName, spaceName := "워크스페이스 최상위", "공간 지정 없음"
	if placement.Parent != "" {
		if e = tx.QueryRow(r.Context(), "SELECT title FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,true)", placement.Parent, wid, p.ID).Scan(&parentName); e != nil {
			apiError(w, 403, "상위 문서 권한이 변경되었습니다")
			return
		}
	}
	if placement.Space != "" {
		if e = tx.QueryRow(r.Context(), "SELECT name FROM spaces WHERE id=$1 AND workspace_id=$2 AND madi_space_allowed($3,id,true)", placement.Space, wid, p.ID).Scan(&spaceName); e != nil {
			apiError(w, 403, "공간 권한이 변경되었습니다")
			return
		}
	}
	token, e := s.encrypt(string(jsonValue(documentAccessPreviewTicket{"madi-access-preview-v1", p.ID, id, expected, placement, fingerprint, time.Now().Add(10 * time.Minute).Unix()})))
	if e != nil {
		respond(w, nil, e)
		return
	}
	oldVisibility := str(doc, "visibility")
	expansion := oldVisibility == "private" && placement.Visibility != "private" || oldVisibility == "selected" && placement.Visibility == "workspace"
	moving := placement.Parent != str(doc, "parent_id") || placement.Space != str(doc, "space_id")
	if moving && placement.Visibility != "private" {
		expansion = true
	}
	warnings := []string{"내부 문서 링크는 로그인과 현재 권한을 확인하는 주소입니다. 이 변경은 외부 공개 링크를 생성하거나 문서를 게시하지 않습니다.", "상위 문서·공간·워크스페이스의 현재 권한 상한이 모두 적용됩니다. 실제 열람 인원이나 숨겨진 하위 문서 수를 추정하지 않습니다."}
	if expansion {
		warnings = append(warnings, "열람 범위가 넓어질 수 있습니다. 이동으로 이전 상위 제한이 제거되거나 기존 개별 공유 대상이 다시 접근할 수 있습니다.")
	}
	if moving {
		warnings = append(warnings, "문서 ID와 내부 링크는 유지됩니다. 하위 문서도 바뀐 상위 권한의 영향을 받을 수 있으므로 관련 소유자와 확인하세요.")
	}
	jsonResponse(w, 200, map[string]any{"document_id": id, "version": expected, "ticket": token, "expires_in": 600, "before": documentAccessPlacement{str(doc, "parent_id"), str(doc, "space_id"), oldVisibility}, "after": placement, "destination": map[string]any{"parent_name": parentName, "space_name": spaceName}, "potential_expansion": expansion, "requires_confirmation": expansion || moving || oldVisibility != placement.Visibility, "warnings": warnings})
}

// Called after the source document FOR UPDATE/CAS lock, before its UPDATE.
// The final UPDATE also re-evaluates destination ACL in its SQL predicate.
func (s *Server) validateDocumentAccessChangeTx(r *http.Request, tx pgx.Tx, id string, old, in map[string]any, parent, space, visibility string) error {
	placement := documentAccessPlacement{parent, space, visibility}
	changed := placement.Parent != str(old, "parent_id") || placement.Space != str(old, "space_id") || placement.Visibility != str(old, "visibility")
	if !changed && in["access_preview_ticket"] == nil {
		return nil
	}
	p, wid := current(r), str(old, "workspace_id")
	denied := errors.New("문서·목적지·현재 공유 권한이 변경되었습니다. 범위 변경을 다시 확인하세요")
	if p == nil || p.ID != str(old, "owner_id") {
		return denied
	}
	if placement.Parent != "" {
		var target string
		if tx.QueryRow(r.Context(), "SELECT id::text FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,true) FOR SHARE", placement.Parent, wid, p.ID).Scan(&target) != nil {
			return denied
		}
	}
	if placement.Space != "" {
		var target string
		if tx.QueryRow(r.Context(), "SELECT id::text FROM spaces WHERE id=$1 AND workspace_id=$2 AND madi_space_allowed($3,id,true) FOR SHARE", placement.Space, wid, p.ID).Scan(&target) != nil {
			return denied
		}
	}
	value, supplied := in["access_preview_ticket"]
	if !supplied {
		return nil
	}
	if !personalAccessRequest(p) || documentAccessActorTx(r, tx, wid, true) != nil {
		return denied
	}
	token, ok := value.(string)
	if !ok || len(token) > 8192 {
		return denied
	}
	raw, e := s.decrypt(token)
	var ticket documentAccessPreviewTicket
	if e != nil || json.Unmarshal([]byte(raw), &ticket) != nil || ticket.Kind != "madi-access-preview-v1" || ticket.Actor != p.ID || ticket.Document != id || ticket.Version != number(in, "version", 0) || ticket.Expires <= time.Now().Unix() || ticket.Placement != placement {
		return denied
	}
	fingerprint, e := accessPreviewFingerprint(r, tx, id, wid, placement)
	if e != nil || ticket.Fingerprint != fingerprint {
		return denied
	}
	return nil
}
