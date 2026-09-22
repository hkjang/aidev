package server

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed document_access_requests.sql
var documentAccessRequestsSchema string

func (s *Server) migrateDocumentAccessRequests(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, documentAccessRequestsSchema)
	return e
}
func (s *Server) registerDocumentAccessRequests() {
	s.handle("POST /api/v1/documents/{id}/access-requests", s.createDocumentAccessRequest)
	s.handle("GET /api/v1/access-requests", s.listDocumentAccessRequests)
	s.handle("PUT /api/v1/access-requests/{id}", s.resolveDocumentAccessRequest)
}
func personalAccessRequest(p *Principal) bool {
	return p != nil && p.TokenID == "" && !p.ScopeRestricted && p.PluginID == "" && p.Kind == "user"
}
func (s *Server) createDocumentAccessRequest(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	if !personalAccessRequest(p) {
		apiError(w, 403, "개인 브라우저에서 접근 권한을 요청하세요")
		return
	}
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		Permission  string `json:"permission"`
		Reason      string `json:"reason"`
	}
	if decode(r, &in) != nil || !validID(r.PathValue("id")) || !validID(in.WorkspaceID) || !oneOf(in.Permission, "read", "write") || len(in.Reason) > 2000 {
		apiError(w, 400, "워크스페이스·요청 권한과 2KB 이하 사유를 확인하세요")
		return
	}
	if !s.canWorkspace(r.Context(), p, in.WorkspaceID, false) {
		apiError(w, 403, "현재 워크스페이스 멤버만 요청할 수 있습니다")
		return
	}
	if in.Permission == "write" && !s.canWorkspace(r.Context(), p, in.WorkspaceID, true) {
		apiError(w, 400, "현재 역할의 상한은 문서 읽기입니다")
		return
	}
	accepted := func() {
		jsonResponse(w, 202, map[string]any{"accepted": true, "message": "같은 워크스페이스에 요청 가능한 문서가 있으면 소유자에게 전달합니다. 문서의 존재나 제목은 공개하지 않습니다."})
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = documentAccessActorTx(r, tx, in.WorkspaceID, in.Permission == "write"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,196))", p.ID); e != nil {
		respond(w, nil, e)
		return
	}
	var recent, pending int
	var cooldown bool
	if e = tx.QueryRow(r.Context(), "SELECT count(*) FILTER (WHERE created_at>now()-interval '1 hour'),count(*) FILTER(WHERE status='pending'),coalesce(bool_or(requested_document_id=$2 AND workspace_id=$3 AND (status='pending' OR updated_at>now()-interval '10 minutes')),false) FROM document_access_requests WHERE requester_id=$1", p.ID, r.PathValue("id"), in.WorkspaceID).Scan(&recent, &pending, &cooldown); e != nil {
		respond(w, nil, e)
		return
	}
	if cooldown {
		accepted()
		return
	}
	if recent >= 20 || pending >= 20 {
		apiError(w, 429, "접근 요청은 시간당 20개·열린 요청 20개까지 보낼 수 있습니다")
		return
	}
	var owner string
	var already bool
	e = tx.QueryRow(r.Context(), "SELECT owner_id::text,madi_document_allowed($3,id,$4) FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL FOR SHARE", r.PathValue("id"), in.WorkspaceID, p.ID, in.Permission == "write").Scan(&owner, &already)
	var target any = r.PathValue("id")
	if errors.Is(e, pgx.ErrNoRows) || owner == p.ID || already {
		target = nil
		e = nil
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	protected, e := s.ProtectDocumentMetadataTx(r.Context(), tx, p, "", in.WorkspaceID, map[string]any{"reason": strings.TrimSpace(in.Reason)})
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	value, _ := protected.Value.(map[string]any)
	sealed, e := s.encrypt(str(value, "reason"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	tag, e := tx.Exec(r.Context(), `INSERT INTO document_access_requests(id,document_id,requested_document_id,workspace_id,requester_id,permission,reason_ciphertext) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(workspace_id,requested_document_id,requester_id) WHERE status='pending' DO NOTHING`, newID(), target, r.PathValue("id"), in.WorkspaceID, p.ID, in.Permission, sealed)
	if e == nil && tag.RowsAffected() == 1 && target != nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,'문서 접근 권한 요청이 도착했습니다. 접근 요청 메뉴에서 확인하세요.',$3)", newID(), owner, r.PathValue("id"))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	// Do not create a source-resource audit oracle for an unknown request target.
	s.audit(r, "DOCUMENT_ACCESS_REQUEST", in.WorkspaceID, map[string]any{"permission": in.Permission})
	accepted()
}
func (s *Server) listDocumentAccessRequests(w http.ResponseWriter, r *http.Request) {
	p, wid := current(r), r.URL.Query().Get("workspace_id")
	if !personalAccessRequest(p) || !s.canWorkspace(r.Context(), p, wid, false) {
		apiError(w, 403, "개인 브라우저의 현재 워크스페이스에서 확인하세요")
		return
	}
	mine, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',id,'document_id',requested_document_id,'permission',permission,'status',status,'revision',revision,'created_at',created_at,'updated_at',updated_at) FROM document_access_requests WHERE requester_id=$1 AND workspace_id=$2 ORDER BY created_at DESC LIMIT 100`, p.ID, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	incoming, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',a.id,'document_id',a.document_id,'document_title',d.title,'document_version',d.version,'visibility',d.visibility,'requester_name',u.name,'permission',a.permission,'status',a.status,'revision',a.revision,'created_at',a.created_at,'reason_ciphertext',a.reason_ciphertext) FROM document_access_requests a JOIN documents d ON d.id=a.document_id JOIN users u ON u.id=a.requester_id WHERE a.workspace_id=$2 AND d.owner_id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,true) ORDER BY a.created_at DESC LIMIT 100`, p.ID, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for _, item := range incoming {
		sealed := str(item, "reason_ciphertext")
		delete(item, "reason_ciphertext")
		reason, err := s.decrypt(sealed)
		if err != nil {
			item["reason"] = "사유를 복호화할 수 없습니다"
		} else {
			item["reason"] = reason
		}
	}
	jsonResponse(w, 200, map[string]any{"mine": mine, "incoming": incoming, "limit": 100, "notice": "문서 접근 권한 요청은 게시 검토·승인과 별개입니다. 내 요청 목록에서는 접근할 수 없는 문서의 제목을 공개하지 않습니다."})
}
