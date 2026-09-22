package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) updateKnowledgeQuestion(w http.ResponseWriter, r *http.Request) {
	var in questionInput
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "담당자의 로그인 화면에서 질문을 정리하세요")
		return
	}
	if decode(r, &in) != nil || !validQuestionInput(in) || in.Revision < 1 || in.Revision >= 2147483647 || in.MessageID != "" {
		apiError(w, 400, "현재 질문·답변 버전, 담당자·근거·검토일과 동의를 확인하세요")
		return
	}
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	// Lock the union before the question row, avoiding order inversion when
	// references change concurrently in another tab.
	var existing []byte
	var old map[string]any
	var oldRefs []questionRef
	e = tx.QueryRow(r.Context(), `SELECT to_jsonb(q) FROM knowledge_questions q WHERE id=$1 AND madi_question_allowed($2,id)`, r.PathValue("id"), current(r).ID).Scan(&existing)
	if e != nil || json.Unmarshal(existing, &old) != nil || json.Unmarshal(jsonValue(old["source_refs"]), &oldRefs) != nil || str(old, "document_id") != in.DocumentID {
		apiError(w, 404, "현재 담당자가 관리할 수 있는 질문이 없습니다")
		return
	}
	allRefs := append(append([]questionRef{}, oldRefs...), in.Sources...)
	docs, e := s.questionDocumentsTx(r, tx, in.DocumentID, allRefs, true)
	if e != nil {
		apiError(w, 404, e.Error())
		return
	}
	var rev int
	var owner string
	e = tx.QueryRow(r.Context(), `SELECT revision,owner_id::text FROM knowledge_questions WHERE id=$1 FOR UPDATE`, r.PathValue("id")).Scan(&rev, &owner)
	if e != nil || rev != in.Revision || rev != number(old, "revision", 0) || (owner != current(r).ID && str(docs[in.DocumentID], "owner_id") != current(r).ID) {
		apiError(w, 409, "질문이 변경되었습니다. 현재 버전으로 다시 확인하세요")
		return
	}
	d := docs[in.DocumentID]
	wid := str(d, "workspace_id")
	if number(d, "version", 0) != in.Version {
		apiError(w, 409, "답변 원문이 변경되었습니다")
		return
	}
	refs, _, e := s.questionRefsTx(r, tx, docs, in.Sources, true)
	if e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if e = s.questionOwnerTx(r, tx, in.OwnerID, in.DocumentID, wid, refs); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	body := questionBody{strings.TrimSpace(in.Question), strings.TrimSpace(in.Reason)}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, body); e != nil {
		apiError(w, 422, e.Error())
		return
	}
	sealed, e := s.encrypt(string(jsonValue(body)))
	if e != nil {
		respond(w, nil, e)
		return
	}
	note, e := s.encrypt(body.Reason)
	if e != nil {
		respond(w, nil, e)
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE knowledge_questions SET owner_id=$2,ciphertext=$3,source_refs=$4,answer_version=$5,answer_hash=$6,review_due=$7,state='proposed',confirmed_by=NULL,confirmed_at=NULL,revision=revision+1,updated_at=now() WHERE id=$1`, r.PathValue("id"), in.OwnerID, sealed, jsonValue(refs), in.Version, digest(str(d, "markdown")), in.ReviewDue)
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_question_events(question_id,actor_id,revision,state,note_ciphertext) VALUES($1,$2,$3,'proposed',$4)`, r.PathValue("id"), current(r).ID, rev+1, note)
	}
	if e == nil {
		e = s.questionFinalTx(r, tx, wid, r.PathValue("id"), true)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "KNOWLEDGE_QUESTION_UPDATE", r.PathValue("id"), map[string]any{"revision": rev + 1, "source_count": len(refs)})
	jsonResponse(w, 200, map[string]any{"id": r.PathValue("id"), "revision": rev + 1, "state": "proposed"})
}
func (s *Server) decideKnowledgeQuestion(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision int    `json:"revision"`
		State    string `json:"state"`
		Note     string `json:"note"`
		Consent  bool   `json:"consent"`
	}
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "담당자의 로그인 화면에서 확인하세요")
		return
	}
	if decode(r, &in) != nil || in.Revision < 1 || in.Revision >= 2147483647 || !oneOf(in.State, "confirmed", "archived") || len(strings.TrimSpace(in.Note)) == 0 || len(in.Note) > 4000 || !in.Consent {
		apiError(w, 400, "현재 버전·처리 상태·확인 이유와 동의가 필요합니다")
		return
	}
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	row, docs, refs, e := s.questionRecordTx(r, tx, r.PathValue("id"), true)
	if e != nil || str(row, "owner_id") != current(r).ID {
		apiError(w, 404, "현재 담당자가 관리할 수 있는 질문이 없습니다")
		return
	}
	if number(row, "revision", 0) != in.Revision {
		apiError(w, 409, "질문이 변경되었습니다. 다시 확인하세요")
		return
	}
	view, e := s.questionViewTx(r, tx, row, docs, refs, true)
	if e != nil {
		apiError(w, 409, "현재 근거·보호 정책으로 처리할 수 없습니다")
		return
	}
	if !boolean(view, "can_manage") {
		apiError(w, 403, "현재 담당자의 권한이 변경되었습니다")
		return
	}
	if in.State == "confirmed" && (!boolean(view, "fresh") || boolean(view, "overdue") || !boolean(view, "published_current")) {
		apiError(w, 409, "최신 근거·검토일과 게시 상태를 확인하세요. 승인 설정이 있으면 답변 문서의 현재 게시 승인을 먼저 완료하세요")
		return
	}
	if str(row, "state") == in.State {
		if e = s.questionFinalTx(r, tx, str(view, "workspace_id"), str(row, "id"), true); e != nil {
			apiError(w, 403, e.Error())
			return
		}
		jsonResponse(w, 200, map[string]any{"id": row["id"], "revision": row["revision"], "state": row["state"]})
		return
	}
	wid := str(view, "workspace_id")
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, in.Note); e != nil {
		apiError(w, 422, e.Error())
		return
	}
	note, e := s.encrypt(strings.TrimSpace(in.Note))
	if e != nil {
		respond(w, nil, e)
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE knowledge_questions SET state=$2,revision=revision+1,confirmed_by=CASE WHEN $2='confirmed' THEN $3::uuid ELSE NULL END,confirmed_at=CASE WHEN $2='confirmed' THEN now() ELSE NULL END,updated_at=now() WHERE id=$1`, row["id"], in.State, current(r).ID)
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_question_events(question_id,actor_id,revision,state,note_ciphertext) VALUES($1,$2,$3,$4,$5)`, row["id"], current(r).ID, in.Revision+1, in.State, note)
	}
	if e == nil {
		e = s.questionFinalTx(r, tx, wid, str(row, "id"), true)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "KNOWLEDGE_QUESTION_CONFIRM", str(row, "id"), map[string]any{"revision": in.Revision + 1, "state": in.State})
	jsonResponse(w, 200, map[string]any{"id": row["id"], "revision": in.Revision + 1, "state": in.State})
}
