package httpapi

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/dream"
	"github.com/hkjang/umm/internal/store"
	"github.com/jackc/pgx/v5"
)

func encodeOffsetCursor(offset int) string {
	if offset <= 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte("offset:" + strconv.Itoa(offset)))
}

func decodeOffsetCursor(raw string) (int, bool) {
	if raw == "" {
		return 0, true
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || !strings.HasPrefix(string(decoded), "offset:") {
		return 0, false
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(string(decoded), "offset:"))
	return offset, err == nil && offset >= 0 && offset <= 1_000_000
}

func parsePageLimit(r *http.Request, fallback, maximum int) int {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = fallback
	}
	return min(limit, maximum)
}

func (s *Server) todayReview(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	p := principal(r)
	result, err := s.Store.TodayReview(r.Context(), p.User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "오늘의 리뷰를 준비하지 못했습니다.")
		return
	}
	redactTodayDreams(&result, hasScope(r, "dreams:read"))
	s.Store.Track(r.Context(), &p.User.ID, "today.opened", "user", &p.User.ID, map[string]any{"reviewCount": result.Counts["review"]})
	writeJSON(w, http.StatusOK, result)
}

func redactTodayDreams(result *store.TodayReview, allowed bool) {
	if allowed {
		return
	}
	result.Dreams = []store.ReviewDream{}
	if result.Counts != nil {
		result.Counts["dreams"] = 0
	}
}

func (s *Server) reviewNote(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	var body struct {
		SnoozeDays *int  `json:"snoozeDays"`
		Pinned     *bool `json:"pinned"`
		Complete   *bool `json:"complete"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, http.StatusBadRequest, "검토 설정 형식이 올바르지 않습니다.")
		return
	}
	if body.SnoozeDays != nil && (*body.SnoozeDays < 1 || *body.SnoozeDays > 365) {
		writeError(w, http.StatusBadRequest, "다시 볼 날짜는 1~365일 뒤로 지정해 주세요.")
		return
	}
	complete := body.Complete == nil || *body.Complete
	if !complete && (body.Pinned == nil || body.SnoozeDays != nil) {
		writeError(w, http.StatusBadRequest, "검토를 완료하지 않을 때는 고정 상태만 변경할 수 있습니다.")
		return
	}
	p := principal(r)
	item, err := s.Store.UpdateReview(r.Context(), p.User.ID, noteID, body.SnoozeDays, body.Pinned, complete)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "검토할 생각을 찾을 수 없습니다.")
			return
		}
		writeError(w, http.StatusInternalServerError, "검토 상태를 저장하지 못했습니다.")
		return
	}
	event := "review.pinned"
	if complete {
		event = "review.completed"
	}
	s.Store.Track(r.Context(), &p.User.ID, event, "note", &noteID, map[string]any{"snoozeDays": body.SnoozeDays, "pinned": body.Pinned})
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) onboardingProgress(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	result, err := s.Store.TodayReview(r.Context(), principal(r).User.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "시작 안내 상태를 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, result.Onboarding)
}

func (s *Server) completeOnboarding(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	p := principal(r)
	if err := s.Store.CompleteOnboarding(r.Context(), p.User.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "시작 안내 상태를 저장하지 못했습니다.")
		return
	}
	s.Store.Track(r.Context(), &p.User.ID, "onboarding.completed", "user", &p.User.ID, map[string]any{})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) noteBacklinks(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	links, err := s.Store.Backlinks(r.Context(), principal(r).User.ID, noteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "연결된 생각을 찾을 수 없습니다.")
			return
		}
		slog.Warn("backlink list failed", "note_id", noteID, "user_id", principal(r).User.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "연결된 생각을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backlinks": links})
}

func mentionedUsernameTokens(body string) []string {
	runes := []rune(body)
	seen := map[string]bool{}
	out := []string{}
	for index := 0; index < len(runes); index++ {
		if runes[index] != '@' || (index > 0 && !unicode.IsSpace(runes[index-1]) && !strings.ContainsRune("([{", runes[index-1])) {
			continue
		}
		end := index + 1
		for end < len(runes) && !unicode.IsSpace(runes[end]) {
			end++
		}
		name := strings.ToLower(string(runes[index+1 : end]))
		if name == "" {
			continue
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
		index = end - 1
	}
	return out
}

func commentCreateError(err error) (int, string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return http.StatusNotFound, "댓글을 남길 수 있는 생각을 찾지 못했습니다."
	case errors.Is(err, store.ErrInvalidParentComment):
		return http.StatusBadRequest, "답글을 연결할 댓글을 찾지 못했습니다."
	default:
		return http.StatusInternalServerError, "댓글을 저장하지 못했습니다."
	}
}

func commentMutationError(err error, forbiddenMessage, failureMessage string) (int, string) {
	if errors.Is(err, pgx.ErrNoRows) {
		return http.StatusForbidden, forbiddenMessage
	}
	return http.StatusInternalServerError, failureMessage
}

func writeCommentMutationError(w http.ResponseWriter, r *http.Request, status int, message string) {
	if status == http.StatusForbidden {
		writeProblem(w, r, status, "comment-mutation-forbidden", "댓글 변경 권한 없음", message, nil)
		return
	}
	writeError(w, status, message)
}

func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	comments, err := s.Store.ListComments(r.Context(), principal(r).User.ID, noteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "댓글을 볼 수 있는 생각을 찾지 못했습니다.")
			return
		}
		slog.Warn("comment list failed", "note_id", noteID, "user_id", principal(r).User.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "댓글을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	var body struct {
		Body     string     `json:"body"`
		ParentID *uuid.UUID `json:"parentId"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, http.StatusBadRequest, "댓글 형식이 올바르지 않습니다.")
		return
	}
	body.Body = strings.TrimSpace(body.Body)
	if utf8.RuneCountInString(body.Body) < 1 || utf8.RuneCountInString(body.Body) > 4000 {
		writeError(w, http.StatusBadRequest, "댓글은 1~4,000자로 입력해 주세요.")
		return
	}
	p := principal(r)
	comment, _, err := s.Store.CreateComment(r.Context(), p.User.ID, noteID, body.ParentID, body.Body, mentionedUsernameTokens(body.Body))
	if err != nil {
		status, message := commentCreateError(err)
		if status >= http.StatusInternalServerError {
			slog.Warn("comment create failed", "note_id", noteID, "user_id", p.User.ID, "error", err)
		}
		writeError(w, status, message)
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "comment.create", "comment", comment.ID.String(), map[string]any{"noteId": noteID})
	writeJSON(w, http.StatusCreated, comment)
}

func (s *Server) resolveComment(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	commentID, ok := parseID(w, r, "commentID")
	if !ok {
		return
	}
	var body struct {
		Resolved bool `json:"resolved"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, http.StatusBadRequest, "댓글 상태 형식이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	comment, _, err := s.Store.ResolveComment(r.Context(), p.User.ID, commentID, body.Resolved)
	if err != nil {
		status, message := commentMutationError(err, "댓글 상태를 변경할 권한이 없습니다.", "댓글 상태를 저장하지 못했습니다.")
		if status >= http.StatusInternalServerError {
			slog.Warn("comment resolution failed", "comment_id", commentID, "user_id", p.User.ID, "error", err)
		}
		writeCommentMutationError(w, r, status, message)
		return
	}
	writeJSON(w, http.StatusOK, comment)
}

func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	commentID, ok := parseID(w, r, "commentID")
	if !ok {
		return
	}
	p := principal(r)
	_, err := s.Store.DeleteComment(r.Context(), p.User.ID, commentID)
	if err != nil {
		status, message := commentMutationError(err, "댓글을 삭제할 권한이 없습니다.", "댓글을 삭제하지 못했습니다.")
		if status >= http.StatusInternalServerError {
			slog.Warn("comment deletion failed", "comment_id", commentID, "user_id", p.User.ID, "error", err)
		}
		writeCommentMutationError(w, r, status, message)
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "comment.delete", "comment", commentID.String(), map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}

func parseOptionalTime(value string) (*time.Time, bool) {
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func parseOptionalUUID(value string) (*uuid.UUID, bool) {
	if value == "" {
		return nil, true
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func noteIDFromPath(r *http.Request) string { return chi.URLParam(r, "noteID") }

var _ = store.Comment{}

// morningBrief reports what accumulated while the person was away.
//
// The window defaults to a day, which is the cadence Dream runs on. It is a
// parameter because "since I last looked" is not always yesterday — someone
// returning from a week away wants the week.
func (s *Server) morningBrief(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	hours := 24
	if raw := r.URL.Query().Get("hours"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 24*90 {
			writeError(w, http.StatusBadRequest, "기간은 1시간에서 90일 사이여야 합니다.")
			return
		}
		hours = parsed
	}
	p := principal(r)
	brief, err := s.Store.MorningBrief(r.Context(), p.User.ID, time.Now().Add(-time.Duration(hours)*time.Hour))
	if err != nil {
		slog.Warn("morning brief failed", "user_id", p.User.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "간밤의 요약을 준비하지 못했습니다.")
		return
	}
	// Dreams are a separate capability, so a key without that scope sees the
	// rest of the brief rather than nothing.
	if !hasScope(r, "dreams:read") {
		brief.Dreams = []store.BriefGroup{}
	}
	writeJSON(w, http.StatusOK, brief)
}

// contradictions lists the disagreements recorded in a person's spaces.
//
// Recorded, not detected: umm does not read two notes and conclude they
// conflict. An empty list means nobody has marked any, which is why the
// interface shows nothing at all rather than a zero.
func (s *Server) contradictions(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	var spaceID *uuid.UUID
	if raw := r.URL.Query().Get("spaceId"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "공간 ID가 올바르지 않습니다.")
			return
		}
		spaceID = &parsed
	}
	p := principal(r)
	items, err := s.Store.Contradictions(r.Context(), p.User.ID, spaceID)
	if err != nil {
		slog.Warn("contradictions failed", "user_id", p.User.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "상충하는 생각을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contradictions": items})
}

// openQuestions lists questions nobody has answered.
//
// Open means nobody has drawn an answer, not that umm looked for one. An empty
// result is not "everything is answered", and the interface shows nothing rather
// than a zero for the same reason contradictions do.
func (s *Server) openQuestions(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	var spaceID *uuid.UUID
	if raw := r.URL.Query().Get("spaceId"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "공간 ID가 올바르지 않습니다.")
			return
		}
		spaceID = &parsed
	}
	p := principal(r)
	items, err := s.Store.OpenQuestions(r.Context(), p.User.ID, spaceID)
	if err != nil {
		slog.Warn("open questions failed", "user_id", p.User.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "열린 질문을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": items})
}

// askMemory answers a question from the person's own thoughts.
//
// The answer is built only from what they already wrote, and the sources come
// back with it so a claim with no citation is visible as one. When nothing
// matched, umm says so instead of calling a model that would fill the gap
// convincingly.
func (s *Server) askMemory(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "ai:assist") {
		return
	}
	var body struct {
		Question string `json:"question"`
	}
	if decodeJSON(w, r, &body) != nil || strings.TrimSpace(body.Question) == "" {
		writeError(w, http.StatusBadRequest, "질문을 입력해 주세요.")
		return
	}
	p := principal(r)
	result, err := s.Dreams.Ask(r.Context(), p.User.ID, body.Question)
	if err != nil {
		if errors.Is(err, dream.ErrNothingToAnswerFrom) {
			// Not an error condition for the person — an answer about their memory.
			writeJSON(w, http.StatusOK, map[string]any{
				"answer":   "",
				"sources":  []any{},
				"excluded": result.Excluded,
				"grounded": false,
			})
			return
		}
		if errors.Is(err, dream.ErrChatModelNotConfigured) {
			writeProblem(w, r, http.StatusPreconditionFailed, "chat-model-not-configured",
				"채팅 모델이 설정되지 않았습니다",
				"관리자 → AI Gateway에서 모델을 설정하면 기억에 질문할 수 있습니다. 임베딩 모델과는 별개입니다.", nil)
			return
		}
		if writeAIQuotaProblem(w, r, err) {
			return
		}
		slog.Warn("ask failed", "user_id", p.User.ID, "error", err)
		writeError(w, http.StatusBadGateway, "기억에서 답을 찾지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"answer":   result.Answer,
		"sources":  result.Sources,
		"excluded": result.Excluded,
		"model":    result.Model,
		"grounded": true,
	})
}

// runThoughtAgent lets an assistant look around someone's memory before
// answering, instead of answering from a single retrieval.
//
// It reads and never writes: the tools it can call are all lookups, so a
// proposal comes back as text for the person to act on. The steps come back
// with the answer for the same reason the sources do on /ai/ask — an answer
// whose path you cannot see is one you have to take on faith.
func (s *Server) runThoughtAgent(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "ai:assist") {
		return
	}
	var body struct {
		Task string `json:"task"`
	}
	if decodeJSON(w, r, &body) != nil || strings.TrimSpace(body.Task) == "" {
		writeError(w, http.StatusBadRequest, "무엇을 살펴볼지 입력해 주세요.")
		return
	}
	p := principal(r)
	result, err := s.Dreams.RunAgent(r.Context(), p.User.ID, body.Task)
	if err != nil {
		if errors.Is(err, dream.ErrAgentTaskInvalid) {
			writeError(w, http.StatusBadRequest, "요청이 비어 있거나 너무 깁니다. 500자 이내로 적어 주세요.")
			return
		}
		if errors.Is(err, dream.ErrChatModelNotConfigured) {
			writeProblem(w, r, http.StatusPreconditionFailed, "chat-model-not-configured",
				"채팅 모델이 설정되지 않았습니다",
				"관리자 → AI Gateway에서 모델을 설정하면 조력자를 쓸 수 있습니다. 임베딩 모델과는 별개입니다.", nil)
			return
		}
		if writeAIQuotaProblem(w, r, err) {
			return
		}
		slog.Warn("thought agent failed", "user_id", p.User.ID, "error", err)
		writeError(w, http.StatusBadGateway, "조력자가 답하지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"answer":    result.Answer,
		"steps":     result.Steps,
		"excluded":  result.Excluded,
		"model":     result.Model,
		"truncated": result.Truncated,
	})
}
