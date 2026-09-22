package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_questions.sql
var knowledgeQuestionsSchema string

func (s *Server) migrateKnowledgeQuestions(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgeQuestionsSchema)
	return e
}
func (s *Server) registerKnowledgeQuestions() {
	s.handle("GET /api/v1/knowledge/questions", s.listKnowledgeQuestions)
	s.handle("POST /api/v1/knowledge/questions", s.createKnowledgeQuestion)
	s.handle("GET /api/v1/knowledge/questions/{id}", s.getKnowledgeQuestion)
	s.handle("PUT /api/v1/knowledge/questions/{id}", s.updateKnowledgeQuestion)
	s.handle("POST /api/v1/knowledge/questions/{id}/decision", s.decideKnowledgeQuestion)
	s.handle("GET /api/v1/ai/messages/{id}/question-preview", s.personalQuestionPreview)
	s.handle("POST /api/v1/ai/messages/{id}/question-draft", s.createPersonalQuestionDraft)
}

type questionRef struct {
	ID         string    `json:"id"`
	Version    int       `json:"version"`
	Hash       string    `json:"hash"`
	Attachment *aiSource `json:"attachment,omitempty"`
}
type questionBody struct {
	Question string `json:"question"`
	Reason   string `json:"reason"`
}
type questionInput struct {
	DocumentID string        `json:"document_id"`
	Version    int           `json:"version"`
	Revision   int           `json:"revision"`
	OwnerID    string        `json:"owner_id"`
	Question   string        `json:"question"`
	Reason     string        `json:"reason"`
	ReviewDue  string        `json:"review_due"`
	Sources    []questionRef `json:"sources"`
	MessageID  string        `json:"message_id"`
	Consent    bool          `json:"consent"`
}

func questionRefValid(ref questionRef) bool {
	return validID(ref.ID) && ref.Version > 0 && ref.Version < 2147483647 && (ref.Attachment == nil || ref.Attachment.ID == ref.ID && ref.Attachment.Version == ref.Version)
}
func questionRefKey(ref questionRef) string {
	if ref.Attachment != nil {
		return ref.ID + ":" + ref.Attachment.AttachmentID + ":" + ref.Attachment.FragmentID
	}
	return ref.ID
}
func validQuestionInput(in questionInput) bool {
	if !validID(in.DocumentID) || !validID(in.OwnerID) || in.Version < 1 || in.Version >= 2147483647 || !in.Consent || !validKnowledgeDate(in.ReviewDue) || strings.TrimSpace(in.Question) == "" || len(in.Question) > 2000 || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 4000 || len(in.Sources) < 1 || len(in.Sources) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, ref := range in.Sources {
		if !questionRefValid(ref) || seen[questionRefKey(ref)] {
			return false
		}
		seen[questionRefKey(ref)] = true
	}
	return in.MessageID == "" || validID(in.MessageID)
}

// Acquire all documents in UUID order before question/approval/protection locks.
// This is bounded reference validation; it never runs a provider or grants access.
func (s *Server) questionDocumentsTx(r *http.Request, tx pgx.Tx, answer string, refs []questionRef, write bool) (map[string]map[string]any, error) {
	ids := []string{answer}
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}
	sort.Strings(ids)
	docs := map[string]map[string]any{}
	wid := ""
	bytes := 0
	for _, id := range ids {
		if docs[id] != nil {
			continue
		}
		var raw []byte
		e := tx.QueryRow(r.Context(), `SELECT to_jsonb(d) FROM documents d WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,$3) AND ($4='' OR workspace_id::text=$4) FOR SHARE`, id, current(r).ID, write && id == answer, current(r).WorkspaceID).Scan(&raw)
		if e != nil {
			return nil, errors.New("현재 접근 가능한 답변과 근거를 확인하세요")
		}
		bytes += len(raw)
		if bytes > 8<<20 {
			return nil, errors.New("답변과 근거 검사는 JSON 합계8MiB까지입니다. 더 작은 문서 단위로 근거를 정리하세요")
		}
		var d map[string]any
		if json.Unmarshal(raw, &d) != nil {
			return nil, errors.New("문서 형식을 확인할 수 없습니다")
		}
		if wid == "" {
			wid = str(d, "workspace_id")
		}
		if str(d, "workspace_id") != wid {
			return nil, errors.New("같은 워크스페이스의 근거를 선택하세요")
		}
		docs[id] = d
	}
	return docs, nil
}
func (s *Server) questionRefsTx(r *http.Request, tx pgx.Tx, docs map[string]map[string]any, refs []questionRef, strict bool) ([]questionRef, bool, error) {
	canonical := []questionRef{}
	fresh := true
	for _, ref := range refs {
		d := docs[ref.ID]
		if d == nil {
			return nil, false, errors.New("현재 근거 접근 권한이 변경되었습니다")
		}
		match := ref.Version == number(d, "version", 0) && (ref.Hash == "" || ref.Hash == digest(str(d, "markdown")))
		if strict && !match {
			return nil, false, errors.New("근거 문서 버전이 변경되었습니다. 최신 근거로 다시 확인하세요")
		}
		if ref.Attachment != nil {
			if strict {
				q, e := s.packageAttachmentTx(r.Context(), tx, current(r), packageAttachmentInput{Source: *ref.Attachment})
				if e != nil {
					return nil, false, e
				}
				x := q.aiSource
				x.Markdown = ""
				x.Title = ""
				x.URL = ""
				x.CitationURL = ""
				ref.Attachment = &x
			} else {
				v, e := s.attachmentCitationTx(r.Context(), tx, current(r), *ref.Attachment, false)
				if e != nil {
					return nil, false, e
				}
				match = match && v.Fresh
			}
		}
		if strict {
			ref.Hash = digest(str(d, "markdown"))
		}
		fresh = fresh && match
		canonical = append(canonical, ref)
	}
	return canonical, fresh, nil
}
func (s *Server) questionOwnerTx(r *http.Request, tx pgx.Tx, owner, answer, wid string, refs []questionRef) error {
	var allowed bool
	e := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$1 AND NOT u.disabled AND u.kind<>'service' AND u.role<>'viewer' AND m.workspace_id=$2 AND m.role IN('owner','admin','editor') AND madi_document_allowed(u.id,$3,true) AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset($4::jsonb) ref(id uuid) WHERE NOT madi_document_allowed(u.id,ref.id,false)))`, owner, wid, answer, jsonValue(refs)).Scan(&allowed)
	if e != nil || !allowed {
		return errors.New("담당자는 답변 수정과 모든 근거 조회가 가능한 현재 구성원이어야 합니다")
	}
	return nil
}

// Reuse the canonical document publication decision. A question never publishes
// its answer or bypasses the configured independent approval workflow.
func (s *Server) questionPublishedTx(r *http.Request, tx pgx.Tx, d map[string]any) (bool, bool, error) {
	var cfgRaw []byte
	var cfg map[string]any
	var clock int64
	if e := tx.QueryRow(r.Context(), `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&cfgRaw); e != nil {
		return false, false, e
	}
	if e := json.Unmarshal(cfgRaw, &cfg); e != nil {
		return false, false, e
	}
	enabled := boolean(cfg, "approval_enabled")
	if str(d, "status") != "published" {
		return false, enabled, nil
	}
	if !enabled {
		return true, false, nil
	}
	if e := tx.QueryRow(r.Context(), `SELECT revision FROM approval_policy_clock WHERE id=1 FOR SHARE`).Scan(&clock); e != nil {
		return false, true, e
	}
	var approved bool
	e := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM approval_requests WHERE resource_kind='document' AND resource_id=$1 AND status='approved' AND resource_version=$2 AND resource_hash=$3 AND policy_revision=$4 AND completed_at IS NOT NULL)`, str(d, "id"), number(d, "version", 0)-1, approvalSnapshotHash(approvalDocumentSnapshot(d)), clock).Scan(&approved)
	return approved, true, e
}

func (s *Server) createKnowledgeQuestion(w http.ResponseWriter, r *http.Request) {
	var in questionInput
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "관리 질문 등록은 사람의 로그인 화면에서 가능합니다")
		return
	}
	if decode(r, &in) != nil || !validQuestionInput(in) {
		apiError(w, 400, "답변 버전·담당자·질문·검토일·1~32개 근거와 공유 동의를 확인하세요")
		return
	}
	tx, e := s.questionTx(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	docs, e := s.questionDocumentsTx(r, tx, in.DocumentID, in.Sources, true)
	if e != nil {
		apiError(w, 404, e.Error())
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
	origin, originHash := "human", ""
	if in.MessageID != "" {
		if str(d, "visibility") != "private" || str(d, "owner_id") != current(r).ID {
			apiError(w, 409, "개인 질문은 본인 소유 비공개 답변 초안에 먼저 연결하세요. 팀 공유와 게시를 별도로 확인하세요")
			return
		}
		preview, err := s.personalQuestionTx(r, tx, in.MessageID)
		if err != nil || str(preview, "workspace_id") != wid || str(preview, "question") != in.Question {
			apiError(w, 409, "본인의 현재 개인 질문과 근거를 다시 확인하세요")
			return
		}
		var expected []questionRef
		_ = json.Unmarshal(jsonValue(preview["sources"]), &expected)
		if questionRefFingerprint(refs) != questionRefFingerprint(expected) {
			apiError(w, 409, "개인 답변의 근거를 빠뜨리거나 다른 버전으로 승격할 수 없습니다")
			return
		}
		origin = "personal_ai_question"
		originHash = digest(string(jsonValue(map[string]any{"message_id": in.MessageID, "question": in.Question, "answer": preview["answer"]})))
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
	var id string
	e = tx.QueryRow(r.Context(), `INSERT INTO knowledge_questions(document_id,owner_id,creator_id,ciphertext,source_refs,answer_version,answer_hash,review_due,origin,origin_hash) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(document_id) DO NOTHING RETURNING id::text`, in.DocumentID, in.OwnerID, current(r).ID, sealed, jsonValue(refs), in.Version, digest(str(d, "markdown")), in.ReviewDue, origin, originHash).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 409, "이 답변 문서에 연결된 질문이 이미 있습니다")
		return
	}
	if e == nil {
		e = s.questionFinalTx(r, tx, wid, id, true)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "KNOWLEDGE_QUESTION_CREATE", id, map[string]any{"document_id": in.DocumentID, "source_count": len(refs), "origin": origin})
	jsonResponse(w, 201, map[string]any{"id": id, "revision": 1, "state": "proposed"})
}

func questionRefFingerprint(refs []questionRef) string {
	values := []string{}
	for _, ref := range refs { // provenance comparison ignores server display URLs and document hash supplied by a client
		value := questionRefKey(ref) + ":" + string(jsonValue(ref.Version))
		if ref.Attachment != nil {
			a := ref.Attachment
			value += ":" + a.AttachmentChecksum + ":" + a.ExtractionID + ":" + string(jsonValue(a.ExtractionRevision)) + ":" + a.FragmentHash + ":" + a.ContentHash + ":" + string(jsonValue([]int{a.StartByte, a.EndByte}))
		}
		values = append(values, value)
	}
	sort.Strings(values)
	return digest(string(jsonValue(values)))
}

func questionToday() string { return time.Now().UTC().Format("2006-01-02") }
