package server

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type rankedAISource struct {
	Source aiSource
	Score  float64
}

func sourceFromChunk(id, title string, version int, c ragChunk) aiSource {
	identity := fmt.Sprintf("%s:%d:%d:%d:%s", id, version, c.Start, c.End, c.Hash)
	return aiSource{ID: id, Title: title, Version: version, Markdown: c.Content, CitationID: digest(identity), StartByte: c.Start, EndByte: c.End, StartLine: c.StartLine, EndLine: c.EndLine, ContentHash: c.Hash, URL: fmt.Sprintf("/app/documents/%s?line=%d", id, c.StartLine), CitationURL: fmt.Sprintf("/documents/%s/citation?version=%d&start=%d&end=%d&hash=%s", id, version, c.Start, c.End, c.Hash)}
}

// Keyword retrieval works without any embedding provider. Fresh source fallback
// is intentional while the derived local index is catching up; old projection
// content is never substituted for a changed document version.
func (s *Server) keywordAISources(r *http.Request, p *Principal, documentID, workspaceID, prompt string) ([]aiSource, error) {
	cfg, e := s.effectiveSettings(r.Context(), workspaceID)
	if e != nil {
		return nil, e
	}
	words := aiQueryWord.FindAllString(prompt, 24)
	query := strings.Join(words, " OR ")
	patterns := []string{}
	for _, word := range words {
		patterns = append(patterns, "%"+strings.ReplaceAll(word, "_", "\\_")+"%")
	}
	rows, e := s.DB.Query(r.Context(), `SELECT d.id::text,d.title,d.markdown,d.version FROM documents d WHERE ($1='' OR d.workspace_id::text=$1) AND ($5='' OR d.id::text=$5) AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false) AND ($5<>'' OR d.search_vector@@websearch_to_tsquery('simple',$3) OR d.title ILIKE ANY($4::text[]) OR d.markdown ILIKE ANY($4::text[]) OR d.tags::text ILIKE ANY($4::text[])) ORDER BY CASE WHEN d.title ILIKE ANY($4::text[]) THEN 4 ELSE 0 END+ts_rank_cd(d.search_vector,websearch_to_tsquery('simple',$3)) DESC,d.updated_at DESC LIMIT 16`, workspaceID, p.ID, query, patterns, documentID)
	if e != nil {
		return nil, e
	}
	documents := []aiSource{}
	bytes := 0
	for rows.Next() {
		var source aiSource
		if e = rows.Scan(&source.ID, &source.Title, &source.Markdown, &source.Version); e != nil {
			rows.Close()
			return nil, e
		}
		bytes += len(source.Markdown)
		if bytes > 16<<20 {
			break
		}
		documents = append(documents, source)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	ranked := []rankedAISource{}
	for _, d := range documents {
		chunks, e := chunkMarkdown(d.Markdown, 6144, 512)
		if e != nil {
			return nil, e
		}
		seen := map[string]bool{}
		for _, c := range chunks {
			if seen[c.Hash] {
				continue
			}
			seen[c.Hash] = true
			body, heading, title := strings.ToLower(c.Content), strings.ToLower(c.Heading), strings.ToLower(d.Title)
			score := 0.0
			for _, word := range words {
				word = strings.ToLower(word)
				if strings.Contains(title, word) {
					score += 4
				}
				if strings.Contains(heading, word) {
					score += 2
				}
				score += float64(min(4, strings.Count(body, word)))
			}
			ranked = append(ranked, rankedAISource{sourceFromChunk(d.ID, d.Title, d.Version, c), score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
	top := number(cfg, "rag_top_k", 8)
	out := []aiSource{}
	budget := 96 << 10
	for _, candidate := range ranked {
		if len(out) >= top {
			break
		}
		if len(candidate.Source.Markdown) > budget {
			continue
		}
		budget -= len(candidate.Source.Markdown)
		out = append(out, candidate.Source)
	}
	return out, nil
}

func (s *Server) getCitation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p := current(r)
	if !s.canDocument(r.Context(), p, id, false) {
		apiError(w, 404, "출처 문서에 접근할 수 없습니다")
		return
	}
	q := r.URL.Query()
	values := map[string]int{}
	for _, key := range []string{"version", "start", "end"} {
		v, e := strconv.Atoi(q.Get(key))
		if e != nil || v < 0 {
			apiError(w, 400, "인용 원문의 버전과 범위를 확인하세요")
			return
		}
		values[key] = v
	}
	start, end := values["start"], values["end"]
	if end <= start || end-start > 8192 || len(q.Get("hash")) != 64 {
		apiError(w, 400, "인용 원문 범위를 확인하세요")
		return
	}
	var title, markdown string
	var version int
	e := s.DB.QueryRow(r.Context(), `SELECT title,markdown,version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,false) AND ($3='' OR workspace_id::text=$3)`, id, p.ID, p.WorkspaceID).Scan(&title, &markdown, &version)
	if e != nil {
		apiError(w, 404, "출처 문서에 접근할 수 없습니다")
		return
	}
	if version != values["version"] {
		apiError(w, 409, "출처 문서가 변경되었습니다. 현재 내용으로 AI 답변을 다시 생성하세요")
		return
	}
	if end > len(markdown) || !utf8.ValidString(markdown[start:end]) || digest(markdown[start:end]) != q.Get("hash") {
		apiError(w, 409, "인용 범위가 원문과 일치하지 않습니다")
		return
	}
	first := strings.Count(markdown[:start], "\n") + 1
	last := first + strings.Count(markdown[start:end], "\n")
	if strings.HasSuffix(markdown[start:end], "\n") {
		last--
	}
	source := sourceFromChunk(id, title, version, ragChunk{Start: start, End: end, StartLine: first, EndLine: last, Hash: q.Get("hash"), Content: markdown[start:end]})
	out := map[string]any{"source": source, "markdown": source.Markdown, "verified": true, "notice": "현재 문서 버전·원문 범위·접근 권한이 일치합니다. AI 답변 자체의 사실성 보증은 아닙니다."}
	respond(w, out, nil)
}

var errRAGChanged = errors.New("검색 AI 처리 중 색인·공급자·문서 권한이 변경되었습니다. 현재 상태로 다시 요청하세요")
