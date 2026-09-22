package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) prepareDocumentSplitTx(r *http.Request, tx pgx.Tx, wid string) error {
	if _, ok := r.Context().Value(documentSplitContextKey{}).(documentSplitIntent); !ok {
		return nil
	}
	_, e := tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,22))", "documents/"+wid)
	return e
}
func (s *Server) validateDocumentSplitTx(r *http.Request, tx pgx.Tx, id string, old map[string]any, expected int, markdown string) error {
	intent, ok := r.Context().Value(documentSplitContextKey{}).(documentSplitIntent)
	if !ok {
		return nil
	}
	t := intent.Ticket
	if t.Actor != current(r).ID || t.SourceID != id || t.Version != expected || t.Expires <= time.Now().Unix() || digest(str(old, "markdown")) != t.SourceHash || markdown != intent.Replacement {
		return errors.New("문서 분리의 기준 원문과 확인 내용이 변경되었습니다")
	}
	if e := documentAccessActorTx(r, tx, str(old, "workspace_id"), true); e != nil {
		return e
	}
	if e := validateDocumentSplitRange(str(old, "markdown"), t.Start, t.End, intent.Selected); e != nil {
		return e
	}
	return s.treePlacementTx(r, tx, "documents", str(old, "workspace_id"), t.ChildID, id)
}
func (s *Server) finishDocumentSplitTx(r *http.Request, tx pgx.Tx, id string, old map[string]any, canonical string, version int) error {
	intent, ok := r.Context().Value(documentSplitContextKey{}).(documentSplitIntent)
	if !ok {
		return nil
	}
	t, p := intent.Ticket, current(r)
	if id != t.SourceID || version != t.Version+1 || !strings.Contains(canonical, "[["+t.ChildID+"|") {
		return errors.New("분리 참조를 정본에 안전하게 남길 수 없습니다")
	}
	wid, childMD := str(old, "workspace_id"), intent.Selected
	// A thematic break at the beginning of a fragment must not become YAML.
	if strings.HasPrefix(childMD, "---\n") || strings.HasPrefix(childMD, "---\r\n") {
		childMD = "\n" + childMD
	}
	protected, e := s.protectCanonicalDocumentTx(r.Context(), tx, p, t.ChildID, wid, t.Title, childMD, []string{}, []string{}, map[string]any{})
	if e != nil {
		return e
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO documents(id,workspace_id,parent_id,space_id,title,markdown,tags,aliases,icon,visibility,owner_id,block_metadata,status) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,'file','workspace',$9,$10,'draft')`, t.ChildID, wid, id, str(old, "space_id"), protected.Title, protected.Markdown, jsonValue(protected.Tags), jsonValue(protected.Aliases), p.ID, jsonValue(protected.Metadata))
	if e != nil {
		return e
	}
	var allowed bool
	if tx.QueryRow(r.Context(), "SELECT madi_document_allowed($1,$2,true)", p.ID, t.ChildID).Scan(&allowed) != nil || !allowed {
		return errors.New("분리한 하위 문서의 현재 작성 권한이 없습니다")
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", t.ChildID, p.ID)
	if e != nil {
		return e
	}
	// Explicit task IDs and their metadata move together; unassigned Markdown
	// todos are indexed from the child's unchanged Markdown by existing jobs.
	for _, task := range indexMarkdown(protected.Markdown).Tasks {
		if task.ID == "" {
			continue
		}
		if task.Ambiguous {
			return errors.New("중복된 할 일 ID는 분리할 수 없습니다")
		}
		_, e = tx.Exec(r.Context(), "UPDATE task_details SET document_id=$2 WHERE document_id=$1 AND task_id=$3", id, t.ChildID, task.ID)
		if e != nil {
			return e
		}
	}
	if e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.created", WorkspaceID: wid, ResourceID: t.ChildID, ResourceType: "document", After: map[string]any{"id": t.ChildID, "title": protected.Title, "status": "draft", "version": 1, "tags": protected.Tags, "split_from": id}}); e != nil {
		return e
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO document_split_receipts(user_id,request_id,source_id,child_id,source_version,child_version,payload_hash) VALUES($1,$2,$3,$4,$5,1,$6)", p.ID, intent.RequestID, id, t.ChildID, version, intent.PayloadHash)
	return e
}
