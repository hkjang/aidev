package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) questionTx(r *http.Request) (pgx.Tx, error) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		return nil, e
	}
	if _, e = tx.Exec(r.Context(), `SET LOCAL statement_timeout='8s'; SET LOCAL lock_timeout='3s'`); e != nil {
		_ = tx.Rollback(r.Context())
		return nil, e
	}
	return tx, nil
}

func (s *Server) questionRecordTx(r *http.Request, tx pgx.Tx, id string, write bool) (map[string]any, map[string]map[string]any, []questionRef, error) {
	var raw []byte
	var refs []questionRef
	if !validID(id) {
		return nil, nil, nil, pgx.ErrNoRows
	}
	read := `SELECT to_jsonb(q) FROM knowledge_questions q JOIN documents d ON d.id=q.document_id WHERE q.id=$1 AND madi_question_allowed($2,q.id) AND ($3='' OR d.workspace_id::text=$3)`
	if e := tx.QueryRow(r.Context(), read, id, current(r).ID, current(r).WorkspaceID).Scan(&raw); e != nil {
		return nil, nil, nil, e
	}
	var first map[string]any
	if e := json.Unmarshal(raw, &first); e != nil {
		return nil, nil, nil, e
	}
	if e := json.Unmarshal(jsonValue(first["source_refs"]), &refs); e != nil {
		return nil, nil, nil, e
	}
	docs, e := s.questionDocumentsTx(r, tx, str(first, "document_id"), refs, write)
	if e != nil {
		return nil, nil, nil, e
	}
	lock := " FOR SHARE OF q"
	if write {
		lock = " FOR UPDATE OF q"
	}
	if e = tx.QueryRow(r.Context(), read+lock, id, current(r).ID, current(r).WorkspaceID).Scan(&raw); e != nil {
		return nil, nil, nil, e
	}
	var row map[string]any
	if e = json.Unmarshal(raw, &row); e != nil {
		return nil, nil, nil, e
	}
	if number(row, "revision", 0) != number(first, "revision", 0) {
		return nil, nil, nil, errors.New("문서와 질문이 변경되었습니다. 다시 조회하세요")
	}
	return row, docs, refs, nil
}
func (s *Server) questionViewTx(r *http.Request, tx pgx.Tx, row map[string]any, docs map[string]map[string]any, refs []questionRef, full bool) (map[string]any, error) {
	d := docs[str(row, "document_id")]
	wid := str(d, "workspace_id")
	plain, e := s.decrypt(str(row, "ciphertext"))
	if e != nil {
		return nil, e
	}
	var body questionBody
	if e = json.Unmarshal([]byte(plain), &body); e != nil {
		return nil, e
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, []any{body, d["title"], d["markdown"]}); e != nil {
		return nil, e
	}
	_, sourceFresh, e := s.questionRefsTx(r, tx, docs, refs, false)
	if e != nil {
		return nil, e
	}
	pub, approvalEnabled, e := s.questionPublishedTx(r, tx, d)
	if e != nil {
		return nil, e
	}
	ownerValid := s.questionOwnerTx(r, tx, str(row, "owner_id"), str(row, "document_id"), wid, refs) == nil
	fresh := sourceFresh && number(row, "answer_version", 0) == number(d, "version", 0) && str(row, "answer_hash") == digest(str(d, "markdown"))
	due := str(row, "review_due")
	overdue := due < questionToday()
	official := str(row, "state") == "confirmed" && fresh && !overdue && pub && ownerValid
	var canWrite bool
	if e = tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,true)`, current(r).ID, str(d, "id")).Scan(&canWrite); e != nil {
		return nil, e
	}
	canManage := personalAIHistory(current(r)) && canWrite && (current(r).ID == str(row, "owner_id") || current(r).ID == str(d, "owner_id"))
	canConfirm := personalAIHistory(current(r)) && canWrite && current(r).ID == str(row, "owner_id") && ownerValid
	sources := []map[string]any{}
	for _, ref := range refs {
		source := docs[ref.ID]
		if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, []any{source["title"], source["markdown"]}); e != nil {
			return nil, e
		}
		link := "/app/documents/" + ref.ID
		if ref.Attachment != nil {
			link = "/app/attachments/" + ref.Attachment.AttachmentID + "?extraction=" + ref.Attachment.ExtractionID + "&fragment=" + ref.Attachment.FragmentID
		}
		sources = append(sources, map[string]any{"id": ref.ID, "version": ref.Version, "hash": ref.Hash, "attachment": ref.Attachment, "title": source["title"], "current_version": source["version"], "url": link})
	}
	out := map[string]any{"id": row["id"], "document_id": row["document_id"], "document_title": d["title"], "workspace_id": wid, "owner_id": row["owner_id"], "question": body.Question, "reason": body.Reason, "revision": row["revision"], "state": row["state"], "origin": row["origin"], "review_due": row["review_due"], "answer_version": row["answer_version"], "current_version": d["version"], "answer_hash": row["answer_hash"], "fresh": fresh, "overdue": overdue, "owner_valid": ownerValid, "published_current": pub, "approval_enabled": approvalEnabled, "official": official, "can_manage": canManage, "confirmed_by": row["confirmed_by"], "confirmed_at": row["confirmed_at"], "updated_at": row["updated_at"], "sources": sources, "notice": "담당자 확인·원문 무결성·현재 근거·검토 기한·게시 승인은 별도 조건입니다. 공식 표시는 사실성의 자동 보증이 아닙니다. 외부 전달 사본은 회수되지 않습니다."}
	out["can_confirm"] = canConfirm
	if full {
		out["markdown"] = d["markdown"]
		events := []map[string]any{}
		rows, e := tx.Query(r.Context(), `SELECT revision,state,note_ciphertext,created_at,actor_id::text FROM knowledge_question_events WHERE question_id=$1 ORDER BY revision DESC LIMIT 50`, row["id"])
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var rev int
			var state, note, actor string
			var created any
			if e = rows.Scan(&rev, &state, &note, &created, &actor); e != nil {
				rows.Close()
				return nil, e
			}
			plain, e := s.decrypt(note)
			if e != nil {
				rows.Close()
				return nil, e
			}
			events = append(events, map[string]any{"revision": rev, "state": state, "note": plain, "created_at": created, "actor_id": actor})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, events); e != nil {
			return nil, e
		}
		out["events"] = events
	}
	return out, nil
}

func (s *Server) questionFinalTx(r *http.Request, tx pgx.Tx, wid, id string, write bool) error {
	scopes := []string{"document:read"}
	if write {
		scopes = append(scopes, "document:write")
	}
	if e := s.knowledgeActorTx(r, tx, wid, scopes...); e != nil {
		return e
	}
	var allowed bool
	if e := tx.QueryRow(r.Context(), `SELECT madi_question_allowed($1,$2)`, current(r).ID, id).Scan(&allowed); e != nil {
		return e
	}
	if !allowed {
		return errors.New("현재 질문과 근거 접근 권한이 변경되었습니다")
	}
	return nil
}
func (s *Server) getKnowledgeQuestion(w http.ResponseWriter, r *http.Request) {
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	row, docs, refs, e := s.questionRecordTx(r, tx, r.PathValue("id"), false)
	if e != nil {
		apiError(w, 404, "현재 접근 가능한 관리 질문이 없습니다")
		return
	}
	out, e := s.questionViewTx(r, tx, row, docs, refs, true)
	if e != nil {
		apiError(w, 409, "현재 원문·근거·정보 보호 정책으로 질문을 표시할 수 없습니다")
		return
	}
	if e = s.questionFinalTx(r, tx, str(out, "workspace_id"), str(row, "id"), false); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	jsonResponse(w, 200, out)
}
func (s *Server) listKnowledgeQuestions(w http.ResponseWriter, r *http.Request) {
	wid, after := r.URL.Query().Get("workspace_id"), r.URL.Query().Get("after")
	if !validID(wid) || after != "" && !validID(after) {
		apiError(w, 400, "워크스페이스와 다음 목록 위치를 확인하세요")
		return
	}
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT q.id::text FROM knowledge_questions q JOIN documents d ON d.id=q.document_id WHERE d.workspace_id=$1 AND madi_question_allowed($2,q.id) AND ($3='' OR q.id>NULLIF($3,'')::uuid) ORDER BY q.id LIMIT 21`, wid, current(r).ID, after)
	if e != nil {
		respond(w, nil, e)
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			respond(w, nil, e)
			return
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	more := len(ids) > 20
	ids = ids[:min(20, len(ids))]
	items := []map[string]any{}
	for _, id := range ids {
		row, docs, refs, e := s.questionRecordTx(r, tx, id, false)
		if e != nil {
			continue
		}
		out, e := s.questionViewTx(r, tx, row, docs, refs, false)
		if e != nil {
			continue
		}
		if s.questionFinalTx(r, tx, wid, id, false) != nil {
			apiError(w, 403, "현재 접근 권한이 변경되었습니다")
			return
		}
		items = append(items, out)
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	next := ""
	if more && len(ids) > 0 {
		next = ids[len(ids)-1]
	}
	jsonResponse(w, 200, map[string]any{"items": items, "next_after": next, "has_more": more, "notice": "현재 접근·정보 보호 조건에서 표시 가능한 질문만 제공합니다. 질문과 답변의 본문 검색은 답변 문서의 기존 통합 검색을 사용하세요."})
}
