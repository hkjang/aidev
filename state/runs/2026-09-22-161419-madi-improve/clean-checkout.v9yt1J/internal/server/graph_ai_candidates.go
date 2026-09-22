package server

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// Model-authored quotes exist only while validating a reply. Durable evidence
// contains exact source coordinates and hashes, never copied source bodies.
func graphAIParseCandidates(raw string, kinds []string, sources []aiSource) ([]graphAICandidate, error) {
	var in struct {
		Candidates []graphAICandidate `json:"candidates"`
	}
	if e := agentStrictJSON([]byte(raw), &in); e != nil || len(in.Candidates) > 30 || in.Candidates == nil {
		return nil, errors.New("AI는 candidates 배열을 가진 JSON 객체와 최대 30개 후보를 반환해야 합니다")
	}
	byID := map[string]aiSource{}
	for _, src := range sources {
		byID[src.ID] = src
	}
	seen := map[string]bool{}
	for i := range in.Candidates {
		v := &in.Candidates[i]
		if !slices.Contains(kinds, v.Kind) || !slices.Contains(graphAIKinds, v.Kind) || strings.TrimSpace(v.Reason) == "" || len(v.Reason) > 2000 || len(v.Title) > 300 || len(v.Description) > 8000 || len(v.Topic) > 120 || len(v.Evidence) < 1 || len(v.Evidence) > 6 {
			return nil, errors.New("AI 후보 종류·길이·근거가 허용 범위를 벗어났습니다")
		}
		for _, text := range []string{v.Reason, v.Title, v.Description, v.Topic} {
			if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
				return nil, errors.New("AI 후보 텍스트가 올바르지 않습니다")
			}
		}
		switch v.Kind {
		case "relation", "duplicate":
			if v.SourceID == v.TargetID || byID[v.SourceID].ID == "" || byID[v.TargetID].ID == "" || v.Title != "" || v.Description != "" || v.Topic != "" || v.EntityType != "" || !oneOf(v.RelationType, "related", "reference") || (v.Kind == "duplicate" && v.RelationType != "related") {
				return nil, errors.New("관계 후보는 실제 선택한 두 문서와 허용된 관계 종류만 참조할 수 있습니다")
			}
		case "topic":
			if byID[v.SourceID].ID == "" || strings.TrimSpace(v.Topic) == "" || v.TargetID != "" || v.Title != "" || v.Description != "" || v.EntityType != "" || v.RelationType != "" {
				return nil, errors.New("주제 후보는 선택 문서와 간결한 주제만 지정해야 합니다")
			}
		case "entity", "gap":
			if strings.TrimSpace(v.Title) == "" || strings.TrimSpace(v.Description) == "" || v.SourceID != "" || v.TargetID != "" || v.Topic != "" || v.RelationType != "" || (v.Kind == "entity" && !slices.Contains(enterpriseEntityKinds, v.EntityType)) || (v.Kind == "gap" && v.EntityType != "") {
				return nil, errors.New("엔터티·보완 문서 후보의 제목·설명·종류를 확인하세요")
			}
		}
		evidenceIDs := map[string]bool{}
		for j := range v.Evidence {
			evidence := &v.Evidence[j]
			src, ok := byID[evidence.DocumentID]
			if !ok || evidence.Citation != nil || len(evidence.Quote) < 2 || len(evidence.Quote) > 500 || !utf8.ValidString(evidence.Quote) {
				return nil, errors.New("AI 근거는 선택 문서의 실제 원문이어야 합니다")
			}
			offset := strings.Index(src.Markdown, evidence.Quote)
			if offset < 0 {
				return nil, errors.New("AI 인용이 제공된 원문과 정확히 일치하지 않습니다")
			}
			start := src.StartByte + offset
			line := src.StartLine + strings.Count(src.Markdown[:offset], "\n")
			citation := sourceFromChunk(src.ID, "", src.Version, ragChunk{Start: start, End: start + len(evidence.Quote), Content: evidence.Quote, Hash: digest(evidence.Quote), StartLine: line, EndLine: line + strings.Count(evidence.Quote, "\n")})
			citation.Markdown = ""
			evidence.Citation = &citation
			evidence.Quote = ""
			evidenceIDs[src.ID] = true
		}
		if v.SourceID != "" && !evidenceIDs[v.SourceID] || v.TargetID != "" && !evidenceIDs[v.TargetID] {
			return nil, errors.New("관계·주제의 각 대상 문서에 실제 근거가 필요합니다")
		}
		hash := digest(string(jsonValue(v)))
		if seen[hash] {
			return nil, errors.New("중복된 AI 후보를 반환했습니다")
		}
		seen[hash] = true
	}
	return in.Candidates, nil
}
func (s *Server) graphAIProtectCandidateTx(ctx context.Context, tx pgx.Tx, p *Principal, wid string, candidate graphAICandidate) (graphAICandidate, error) {
	text := map[string]any{"title": candidate.Title, "description": candidate.Description, "reason": candidate.Reason, "topic": candidate.Topic}
	protected, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", wid, text)
	if e != nil {
		return candidate, e
	}
	value, ok := protected.Value.(map[string]any)
	if !ok {
		return candidate, errors.New("AI 후보 정보보호 결과를 확인할 수 없습니다")
	}
	candidate.Title = str(value, "title")
	candidate.Description = str(value, "description")
	candidate.Reason = str(value, "reason")
	candidate.Topic = str(value, "topic")
	return candidate, nil
}
