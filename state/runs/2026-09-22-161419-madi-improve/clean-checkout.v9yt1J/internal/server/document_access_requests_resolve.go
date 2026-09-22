package server

import (
	"net/http"
)

func (s *Server) resolveDocumentAccessRequest(w http.ResponseWriter, r *http.Request) {
	p, id := current(r), r.PathValue("id")
	if !personalAccessRequest(p) || !validID(id) {
		apiError(w, 403, "개인 브라우저의 접근 요청만 처리할 수 있습니다")
		return
	}
	var in struct {
		Revision        int    `json:"revision"`
		Action          string `json:"action"`
		ExpectedVersion int    `json:"expected_document_version"`
	}
	if decode(r, &in) != nil || in.Revision < 1 || !oneOf(in.Action, "grant", "reject", "cancel") {
		apiError(w, 400, "현재 요청 버전과 처리 동작을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if in.Action == "cancel" {
		var wid string
		if tx.QueryRow(r.Context(), "SELECT workspace_id::text FROM document_access_requests WHERE id=$1 AND requester_id=$2", id, p.ID).Scan(&wid) != nil {
			apiError(w, 409, "현재 요청을 찾을 수 없습니다")
			return
		}
		if e = documentAccessActorTx(r, tx, wid, false); e != nil {
			apiError(w, 403, e.Error())
			return
		}
		tag, e := tx.Exec(r.Context(), "UPDATE document_access_requests SET status='cancelled',revision=revision+1,updated_at=now(),resolved_by=$2 WHERE id=$1 AND requester_id=$2 AND revision=$3 AND status='pending'", id, p.ID, in.Revision)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if tag.RowsAffected() != 1 {
			apiError(w, 409, "요청이 변경되었거나 취소할 수 없습니다")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			respond(w, nil, e)
			return
		}
		jsonResponse(w, 200, map[string]any{"ok": true, "status": "cancelled", "revision": in.Revision + 1})
		return
	}
	if in.ExpectedVersion < 1 {
		apiError(w, 400, "확인한 문서 버전이 필요합니다")
		return
	}
	var docID string
	e = tx.QueryRow(r.Context(), `SELECT a.document_id::text FROM document_access_requests a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.owner_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,true)`, id, p.ID).Scan(&docID)
	if e != nil {
		apiError(w, 403, "현재 소유자의 문서 접근 권한이 필요합니다")
		return
	}
	var doc map[string]any
	e = tx.QueryRow(r.Context(), `SELECT to_jsonb(d)-'search_vector' FROM documents d WHERE id=$1 AND owner_id=$2 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) FOR UPDATE`, docID, p.ID).Scan(&doc)
	if e != nil {
		apiError(w, 403, "문서 소유권이나 접근 권한이 변경되었습니다")
		return
	}
	if number(doc, "version", 0) != in.ExpectedVersion {
		apiError(w, 409, "문서의 공유 상태 또는 버전이 변경되었습니다. 다시 확인하세요")
		return
	}
	if e = documentAccessActorTx(r, tx, str(doc, "workspace_id"), true); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var requester, permission, status string
	var revision int
	e = tx.QueryRow(r.Context(), "SELECT requester_id::text,permission,status,revision FROM document_access_requests WHERE id=$1 AND document_id=$2 FOR UPDATE", id, docID).Scan(&requester, &permission, &status, &revision)
	if e != nil || status != "pending" || revision != in.Revision {
		apiError(w, 409, "요청 상태가 변경되었습니다")
		return
	}
	state := "rejected"
	version := in.ExpectedVersion
	if in.Action == "grant" {
		var allowed bool
		e = tx.QueryRow(r.Context(), `SELECT NOT u.disabled AND u.kind='user' AND ($3='read' OR (u.role<>'viewer' AND m.role IN ('owner','admin','editor'))) FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$1 AND m.workspace_id=$2 FOR SHARE OF u,m`, requester, str(doc, "workspace_id"), permission).Scan(&allowed)
		if e != nil || !allowed {
			apiError(w, 409, "요청자의 현재 워크스페이스 역할이 요청 권한을 허용하지 않습니다")
			return
		}
		visibility := str(doc, "visibility")
		if visibility == "private" {
			var dormant bool
			if e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM document_shares WHERE document_id=$1)", docID).Scan(&dormant); e != nil {
				respond(w, nil, e)
				return
			}
			if dormant {
				apiError(w, 409, "비활성 개별 공유가 남아 있습니다. 기존 공유 설정을 먼저 확인하세요. 다른 사용자의 권한을 묵시적으로 다시 활성화하지 않습니다")
				return
			}
			visibility = "selected"
		}
		_, e = tx.Exec(r.Context(), "INSERT INTO document_shares(document_id,user_id,permission) VALUES($1,$2,$3) ON CONFLICT(document_id,user_id) DO UPDATE SET permission=EXCLUDED.permission", docID, requester, permission)
		if e != nil {
			respond(w, nil, e)
			return
		}
		publication, e := approvalSaveStatusTx(r.Context(), tx, docID, doc, map[string]any{"visibility": visibility}, str(doc, "status"))
		if e != nil {
			approvalRespondError(w, e)
			return
		}
		_, e = tx.Exec(r.Context(), "UPDATE documents SET visibility=$2,status=$3,version=version+1,updated_at=now() WHERE id=$1", docID, visibility, publication)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if tx.QueryRow(r.Context(), "SELECT madi_document_allowed($1,$2,$3)", requester, docID, permission == "write").Scan(&allowed) != nil || !allowed {
			apiError(w, 409, "상위 문서 또는 공간의 접근 제한이 남아 있습니다. 해당 권한을 먼저 확인하세요. 이번 요청은 적용하지 않았습니다")
			return
		}
		_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", docID, p.ID)
		if e == nil {
			e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.updated", WorkspaceID: str(doc, "workspace_id"), ResourceID: docID, ResourceType: "document", Before: doc, After: map[string]any{"id": docID, "version": in.ExpectedVersion + 1, "visibility": visibility, "permission_changed": true}})
		}
		if e != nil {
			respond(w, nil, e)
			return
		}
		state = "granted"
		version++
	}
	_, e = tx.Exec(r.Context(), "UPDATE document_access_requests SET status=$2,revision=revision+1,updated_at=now(),resolved_by=$3 WHERE id=$1", id, state, p.ID)
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO notifications(id,user_id,title) VALUES($1,$2,'문서 접근 권한 요청이 처리되었습니다. 접근 요청 메뉴에서 결과를 확인하세요.')", newID(), requester)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_ACCESS_RESOLVE", docID, map[string]any{"state": state, "permission": permission, "requester_id": requester, "version": version})
	jsonResponse(w, 200, map[string]any{"ok": true, "status": state, "revision": in.Revision + 1, "document_version": version})
}
