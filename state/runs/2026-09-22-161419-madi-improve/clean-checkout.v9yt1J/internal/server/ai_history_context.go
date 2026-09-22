package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type aiHistoryContext struct {
	ID       string
	Version  int
	Messages []map[string]string
	Sources  []aiSource
}

// Selecting a saved conversation is explicit consent to resend its bounded
// context to the same provider. A stale source/provider never silently moves
// historical private text to a changed integration.
func (s *Server) loadAIHistoryContext(r *http.Request, id string, version int, wid string, cfg map[string]any) (aiHistoryContext, error) {
	h := aiHistoryContext{Messages: []map[string]string{}, Sources: []aiSource{}}
	if id == "" {
		if version != 0 {
			return h, errors.New("대화 식별자와 버전을 함께 지정하세요")
		}
		return h, nil
	}
	p := current(r)
	if !personalAIHistory(p) || !validID(id) || version < 1 {
		return h, errors.New("현재 본인의 대화 기록과 버전이 필요합니다")
	}
	var raw []byte
	e := s.DB.QueryRow(r.Context(), `SELECT jsonb_build_object('version',c.version,'messages',(SELECT jsonb_agg(to_jsonb(m) ORDER BY ordinal) FROM ai_messages m WHERE m.conversation_id=c.id)) FROM ai_conversations c WHERE c.id=$1 AND c.owner_id=$2 AND c.workspace_id=$3 AND c.version=$4 AND madi_ai_conversation_allowed($2,c.id) AND (SELECT coalesce(sum(octet_length(m.question)+octet_length(m.answer)),0) FROM ai_messages m WHERE m.conversation_id=c.id)<=131072 AND (SELECT count(*) FROM ai_messages m WHERE m.conversation_id=c.id)<=20`, id, p.ID, wid, version).Scan(&raw)
	if e != nil {
		return h, errors.New("대화가 변경되었거나 현재 권한/컨텍스트 제한(20개 답변·128KiB)을 확인해야 합니다. 새 대화로 시작하세요")
	}
	var data struct {
		Version  int `json:"version"`
		Messages []struct {
			Question string           `json:"question"`
			Answer   string           `json:"answer"`
			Sources  []aiSource       `json:"sources"`
			Grants   []aiHistoryGrant `json:"grant_refs"`
			Provider string           `json:"provider_fingerprint"`
		} `json:"messages"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return h, errors.New("저장한 대화 형식이 올바르지 않습니다")
	}
	h.ID, h.Version = id, version
	for i, message := range data.Messages {
		if message.Provider != aiHistoryProvider(cfg) {
			return h, errors.New("이 기록을 생성한 AI 공급자 설정이 변경되었습니다. 과거 대화를 자동 전송하지 않고 새 대화를 시작하세요")
		}
		for j := range message.Sources {
			for _, g := range message.Grants {
				if g.DocumentID == message.Sources[j].ID {
					message.Sources[j].RAGGrantID = g.GrantID
					message.Sources[j].RAGGrantRevision = g.Revision
				}
			}
		}
		h.Sources = mergeAIHistorySources(h.Sources, message.Sources)
		if len(h.Sources) > 64 {
			return h, errors.New("과거 참조가 64개를 초과했습니다. 새 대화로 시작하세요")
		}
		h.Messages = append(h.Messages, map[string]string{"role": "user", "content": fmt.Sprintf("과거 대화 %d의 질문 (아래 출처 번호는 이 과거 응답에만 해당):\n%s", i+1, message.Question)}, map[string]string{"role": "assistant", "content": message.Answer})
	}
	if s.validateAIStream(r, p, wid, h.Sources, cfg) != nil {
		return h, errors.New("과거 대화의 참조 버전·권한·색인 동의가 변경되었습니다. 현재 자료로 새 대화를 시작하세요")
	}
	budget := 96 << 10
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		return h, e
	}
	defer tx.Rollback(r.Context())
	for i := range h.Sources {
		src := &h.Sources[i]
		if src.StartByte < 0 || src.EndByte < src.StartByte || src.EndByte-src.StartByte > 8192 {
			return h, errors.New("과거 참조 범위를 확인할 수 없습니다")
		}
		var fragment []byte
		if src.AttachmentID != "" {
			var value attachmentCitationCurrent
			value, e = s.attachmentCitationTx(r.Context(), tx, p, *src, true)
			src.Title, fragment = value.Title, []byte(value.Text)
		} else {
			e = tx.QueryRow(r.Context(), `SELECT title,substring(convert_to(markdown,'UTF8') from $4 for $5) FROM documents WHERE id=$1 AND workspace_id=$2 AND version=$3 AND deleted_at IS NULL AND madi_document_allowed($6,id,false)`, src.ID, wid, src.Version, src.StartByte+1, src.EndByte-src.StartByte, p.ID).Scan(&src.Title, &fragment)
		}
		if e != nil || (src.ContentHash != "" && digest(string(fragment)) != src.ContentHash) {
			return h, errors.New("과거 참조의 원문 해시·권한이 변경되었습니다. 새 대화를 시작하세요")
		}
		budget -= len(fragment)
		if budget < 0 {
			return h, errors.New("과거 출처 컨텍스트가 96KiB를 초과했습니다. 새 대화를 시작하세요")
		}
		src.Markdown = string(fragment)
	}
	return h, nil
}
func mergeAIHistorySources(first, second []aiSource) []aiSource {
	result := []aiSource{}
	seen := map[string]bool{}
	for _, list := range [][]aiSource{first, second} {
		for _, source := range list {
			key := fmt.Sprintf("%s:%d:%d:%d:%s:%s:%d:%s:%s:%s:%d", source.ID, source.Version, source.StartByte, source.EndByte, source.ContentHash, source.RAGGrantID, source.RAGGrantRevision, source.AttachmentID, source.ExtractionID, source.FragmentID, source.ExtractionRevision)
			if !seen[key] {
				seen[key] = true
				result = append(result, source)
			}
		}
	}
	return result
}
