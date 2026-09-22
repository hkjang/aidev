package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed ai_history.sql
var aiHistorySchema string

func (s *Server) migrateAIHistory(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, aiHistorySchema)
	return e
}
func (s *Server) registerAIHistory() {
	s.handle("POST /api/v1/ai/conversations", s.saveAIConversation)
	s.handle("GET /api/v1/ai/conversations", s.listAIConversations)
	s.handle("GET /api/v1/ai/conversations/{id}", s.getAIConversation)
	s.handle("DELETE /api/v1/ai/conversations/{id}", s.deleteAIConversation)
}

type aiHistoryGrant struct {
	DocumentID string `json:"document_id"`
	GrantID    string `json:"grant_id"`
	Revision   int64  `json:"revision"`
}
type aiHistoryTicket struct {
	Kind                string           `json:"kind"`
	ID                  string           `json:"id"`
	UserID              string           `json:"user_id"`
	WorkspaceID         string           `json:"workspace_id"`
	ConversationID      string           `json:"conversation_id"`
	ConversationVersion int              `json:"conversation_version"`
	Question            string           `json:"question"`
	Answer              string           `json:"answer"`
	Action              string           `json:"action"`
	Sources             []aiSource       `json:"sources"`
	Grants              []aiHistoryGrant `json:"grants"`
	Provider            string           `json:"provider"`
	Model               string           `json:"model"`
	Expires             int64            `json:"expires"`
}

func personalAIHistory(p *Principal) bool {
	return p != nil && p.TokenID == "" && p.PluginID == "" && !p.ScopeRestricted && p.Kind != "service"
}
func aiHistoryProvider(cfg map[string]any) string {
	fields := map[string]any{}
	for _, key := range append([]string{"ai_enabled", "ai_base_url", "ai_api_key", "ai_model", "ai_max_tokens", "ai_system_prompt"}, ragSettingKeys()...) {
		fields[key] = cfg[key]
	}
	return digest(string(jsonValue(fields)))
}
func (s *Server) sealAIHistory(p *Principal, wid, question, answer, action string, sources []aiSource, cfg map[string]any, history ...aiHistoryContext) (string, error) {
	if !personalAIHistory(p) || !validID(wid) || answer == "" {
		return "", nil
	}
	t := aiHistoryTicket{Kind: "madi-ai-history-v1", ID: newID(), UserID: p.ID, WorkspaceID: wid, Question: question, Answer: answer, Action: action, Sources: sources, Grants: []aiHistoryGrant{}, Provider: aiHistoryProvider(cfg), Model: str(cfg, "ai_model"), Expires: time.Now().Add(time.Hour).Unix()}
	if len(history) > 0 {
		t.ConversationID = history[0].ID
		t.ConversationVersion = history[0].Version
	}
	for _, src := range sources {
		if src.RAGGrantID != "" {
			t.Grants = append(t.Grants, aiHistoryGrant{src.ID, src.RAGGrantID, src.RAGGrantRevision})
		}
	}
	encoded := jsonValue(t)
	if len(encoded) > 5<<20 {
		return "", errors.New("개인 기록의 직렬화 크기 제한(5MiB)을 초과했습니다. 더 짧은 답변을 생성해 저장하세요")
	}
	return s.encrypt(string(encoded))
}

func (s *Server) openAIHistory(raw string, p *Principal) (aiHistoryTicket, error) {
	var t aiHistoryTicket
	if !personalAIHistory(p) || len(raw) > 7<<20 {
		return t, errors.New("개인 브라우저 세션의 유효한 저장 요청이 필요합니다")
	}
	plain, e := s.decrypt(raw)
	if e != nil || json.Unmarshal([]byte(plain), &t) != nil || t.Kind != "madi-ai-history-v1" || t.UserID != p.ID || !validID(t.ID) || !validID(t.WorkspaceID) || t.Expires < time.Now().Unix() || t.Expires > time.Now().Add(time.Hour+time.Minute).Unix() || len(t.Answer) > 4<<20 || len(t.Question) > 128000 || len(t.Sources) > 64 {
		return t, errors.New("저장 티켓이 만료되었거나 현재 사용자의 응답이 아닙니다. 답변을 다시 생성하세요")
	}
	for i := range t.Sources {
		for _, g := range t.Grants {
			if g.DocumentID == t.Sources[i].ID {
				t.Sources[i].RAGGrantID = g.GrantID
				t.Sources[i].RAGGrantRevision = g.Revision
			}
		}
	}
	return t, nil
}

func (s *Server) saveAIConversation(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	if !personalAIHistory(p) {
		apiError(w, 403, "개인 대화 기록은 브라우저에서 본인만 저장할 수 있습니다")
		return
	}
	var in struct {
		Ticket  string `json:"ticket"`
		Consent bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !in.Consent {
		apiError(w, 400, "질문과 완료 답변의 개인 기록 저장에 동의하세요")
		return
	}
	t, e := s.openAIHistory(in.Ticket, p)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	cfg, e := s.effectiveSettings(r.Context(), t.WorkspaceID)
	if e != nil || aiHistoryProvider(cfg) != t.Provider || s.validateAIStream(r, p, t.WorkspaceID, t.Sources, cfg) != nil {
		apiError(w, 409, "참조 자료·권한·AI 설정이 변경되었습니다. 현재 자료로 답변을 다시 생성하세요")
		return
	}
	ctx := r.Context()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	// Serialize identical tickets, including the first conversation creation.
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,187))", t.ID); e != nil {
		respond(w, nil, e)
		return
	}
	var previous string
	e = tx.QueryRow(ctx, `SELECT m.conversation_id::text FROM ai_messages m JOIN ai_conversations c ON c.id=m.conversation_id WHERE m.id=$1 AND c.owner_id=$2 AND madi_ai_conversation_allowed($2,c.id)`, t.ID, p.ID).Scan(&previous)
	if e == nil {
		_ = tx.Rollback(ctx)
		jsonResponse(w, 200, map[string]any{"id": previous, "message_id": t.ID, "replayed": true})
		return
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		respond(w, nil, e)
		return
	}
	ordered := append([]aiSource{}, t.Sources...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	for _, src := range ordered {
		var version int
		var fragment []byte
		if src.StartByte < 0 || src.EndByte < src.StartByte || src.EndByte-src.StartByte > 8192 {
			apiError(w, 409, "참조 조각 범위가 올바르지 않습니다")
			return
		}
		if src.AttachmentID != "" {
			var value attachmentCitationCurrent
			value, e = s.attachmentCitationTx(ctx, tx, p, src, true)
			version, fragment = value.Version, []byte(value.Text)
		} else {
			e = tx.QueryRow(ctx, `SELECT version,substring(convert_to(markdown,'UTF8') from $4 for $5) FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false) FOR SHARE`, src.ID, t.WorkspaceID, p.ID, src.StartByte+1, src.EndByte-src.StartByte).Scan(&version, &fragment)
		}
		if e != nil || version != src.Version || (src.ContentHash != "" && digest(string(fragment)) != src.ContentHash) || ragActorTx(ctx, tx, p, src.ID, t.WorkspaceID, false) != nil {
			apiError(w, 409, "참조 문서 또는 권한이 변경되어 저장하지 않았습니다")
			return
		}
	}
	for _, src := range ordered {
		if src.RAGGrantID == "" {
			continue
		}
		g, err := ragGrantTx(ctx, tx, src.ID, true)
		if err != nil || !g.Active || g.ID != src.RAGGrantID || g.Revision != src.RAGGrantRevision || g.Version != src.Version || g.Provider != ragProviderFingerprint(cfg) {
			apiError(w, 409, "참조 자료의 AI 색인 동의가 변경되었습니다")
			return
		}
		actor := &Principal{ID: g.ActorID, TokenID: g.TokenID, WorkspaceID: g.WorkspaceID, PluginID: g.Constraints.PluginID, ScopeRestricted: g.Constraints.Restricted || g.TokenID != "", Scopes: g.Constraints.Scopes}
		if ragActorTx(ctx, tx, actor, src.ID, t.WorkspaceID, true) != nil {
			apiError(w, 409, "색인 실행자의 권한이 변경되었습니다")
			return
		}
	}
	fresh, e := s.ragSettingsTx(ctx, tx, t.WorkspaceID)
	if e != nil || aiHistoryProvider(fresh) != t.Provider {
		apiError(w, 409, "AI 설정이 변경되었습니다")
		return
	}
	var active bool
	cookie, cookieError := r.Cookie("madi_session")
	if cookieError != nil {
		apiError(w, 403, "현재 로그인 세션이 필요합니다")
		return
	}
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id JOIN sessions s ON s.user_id=u.id WHERE u.id=$1 AND NOT u.disabled AND m.workspace_id=$2 AND s.token_hash=$3 AND s.expires_at>now())`, p.ID, t.WorkspaceID, digest(cookie.Value)).Scan(&active)
	if e != nil || !active {
		apiError(w, 403, "현재 사용자·워크스페이스·세션 권한이 필요합니다")
		return
	}
	protected, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", t.WorkspaceID, map[string]any{"question": t.Question, "answer": t.Answer})
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	canonical := protected.Value.(map[string]any)
	t.Question, t.Answer = str(canonical, "question"), str(canonical, "answer")
	if len(t.Question) > 128000 || len(t.Answer) > 4<<20 {
		apiError(w, 422, "보호 처리 후 대화 기록 크기 제한을 초과했습니다")
		return
	}
	// Source titles can contain now-sensitive legacy text. Keep immutable
	// citation coordinates, and resolve the current title through the guarded
	// citation viewer instead of copying titles into another persistent store.
	for i := range t.Sources {
		t.Sources[i].Title = "참고 문서 " + strconv.Itoa(i+1)
	}
	id := t.ConversationID
	ordinal := 1
	if id != "" {
		var currentVersion, count, totalBytes int
		e = tx.QueryRow(ctx, `SELECT version FROM ai_conversations WHERE id=$1 AND owner_id=$2 AND workspace_id=$3 AND madi_ai_conversation_allowed($2,id) FOR UPDATE`, id, p.ID, t.WorkspaceID).Scan(&currentVersion)
		if e != nil || currentVersion != t.ConversationVersion {
			apiError(w, 409, "다른 탭에서 대화가 변경되었습니다. 현재 기록에서 다시 질문하세요")
			return
		}
		e = tx.QueryRow(ctx, `SELECT count(*),coalesce(sum(octet_length(question)+octet_length(answer)),0) FROM ai_messages WHERE conversation_id=$1`, id).Scan(&count, &totalBytes)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if count >= 20 || totalBytes+len(t.Question)+len(t.Answer) > 16<<20 {
			apiError(w, 409, "대화 저장 한도(20개 답변·16MiB)에 도달했습니다. 새 대화를 시작하세요")
			return
		}
		ordinal = currentVersion + 1
		_, e = tx.Exec(ctx, `UPDATE ai_conversations SET version=version+1,updated_at=now() WHERE id=$1`, id)
	} else {
		id = newID()
	}
	title := string([]rune(t.Question)[:min(100, len([]rune(t.Question)))])
	if t.ConversationID == "" {
		_, e = tx.Exec(ctx, `INSERT INTO ai_conversations(id,workspace_id,owner_id,title) VALUES($1,$2,$3,$4)`, id, t.WorkspaceID, p.ID, title)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `INSERT INTO ai_messages(id,conversation_id,ordinal,question,answer,action,sources,grant_refs,provider_fingerprint,model) VALUES($1,$2,$10,$3,$4,$5,$6,$7,$8,$9)`, t.ID, id, t.Question, t.Answer, t.Action, jsonValue(t.Sources), jsonValue(t.Grants), t.Provider, t.Model, ordinal)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "AI_HISTORY_SAVE", id, map[string]any{"source_count": len(t.Sources)})
	jsonResponse(w, 201, map[string]any{"id": id, "message_id": t.ID, "version": ordinal, "protection_changed": protected.Changed})
}

func (s *Server) listAIConversations(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	q := r.URL.Query()
	wid := q.Get("workspace_id")
	if !personalAIHistory(p) || !s.canWorkspace(r.Context(), p, wid, false) {
		apiError(w, 403, "개인 대화 기록 접근 권한이 없습니다")
		return
	}
	term := strings.TrimSpace(q.Get("q"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if len(term) > 500 || offset < 0 || offset > 10000 {
		apiError(w, 400, "검색 조건을 확인하세요")
		return
	}
	pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(term) + "%"
	rows, e := s.rows(r.Context(), `SELECT to_jsonb(c)||jsonb_build_object('message_count',(SELECT count(*) FROM ai_messages m WHERE m.conversation_id=c.id)) FROM ai_conversations c WHERE c.workspace_id=$2 AND c.owner_id=$1 AND madi_ai_conversation_allowed($1,c.id) AND ($3='' OR c.title ILIKE $4 OR EXISTS(SELECT 1 FROM ai_messages m WHERE m.conversation_id=c.id AND (m.question ILIKE $4 OR m.answer ILIKE $4))) ORDER BY c.updated_at DESC,c.id LIMIT 41 OFFSET $5`, p.ID, wid, term, pattern, offset)
	if e != nil {
		respond(w, nil, e)
		return
	}
	more := len(rows) > 40
	if more {
		rows = rows[:40]
	}
	jsonResponse(w, 200, map[string]any{"items": rows, "has_more": more, "next_offset": offset + len(rows)})
}
func (s *Server) getAIConversation(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	id := r.PathValue("id")
	if !personalAIHistory(p) || !validID(id) {
		apiError(w, 404, "개인 대화 기록을 찾을 수 없습니다")
		return
	}
	v, e := s.one(r.Context(), `SELECT to_jsonb(c)||jsonb_build_object('messages',coalesce((SELECT jsonb_agg((to_jsonb(m)-'grant_refs'-'provider_fingerprint') ORDER BY m.ordinal) FROM ai_messages m WHERE m.conversation_id=c.id),'[]'::jsonb)) FROM ai_conversations c WHERE c.id=$1 AND c.owner_id=$2 AND madi_ai_conversation_allowed($2,c.id) AND ($3='' OR c.workspace_id::text=$3)`, id, p.ID, r.URL.Query().Get("workspace_id"))
	if errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 404, "개인 대화 기록을 찾을 수 없거나 참조 문서 권한이 변경되었습니다")
		return
	}
	respond(w, v, e)
}
func (s *Server) deleteAIConversation(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	id := r.PathValue("id")
	var in struct {
		Confirmation string `json:"confirmation"`
		Version      int    `json:"version"`
	}
	if !personalAIHistory(p) || !validID(id) {
		apiError(w, 403, "본인의 대화 기록만 삭제할 수 있습니다")
		return
	}
	if decode(r, &in) != nil || in.Confirmation != "DELETE" || in.Version < 1 {
		apiError(w, 400, "대화 버전과 삭제 확인이 필요합니다")
		return
	}
	// The owner may discard a now-inaccessible record without reading its body.
	result, e := s.DB.Exec(r.Context(), `DELETE FROM ai_conversations WHERE id=$1 AND owner_id=$2 AND version=$3`, id, p.ID, in.Version)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if result.RowsAffected() != 1 {
		apiError(w, 409, "기록이 변경되었거나 이미 삭제되었습니다")
		return
	}
	s.audit(r, "AI_HISTORY_DELETE", id, nil)
	jsonResponse(w, 200, map[string]any{"deleted": true})
}
