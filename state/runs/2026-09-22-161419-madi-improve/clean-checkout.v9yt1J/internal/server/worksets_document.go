package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

func (s *Server) registerWorksetDocuments() {
	s.handle("GET /api/v1/documents/{id}/passport", s.documentPassport)
	s.handle("POST /api/v1/documents/{id}/cleanup-preview", s.documentCleanupPreview)
}
func (s *Server) worksetDocumentTx(r *http.Request, tx pgx.Tx, write bool) (map[string]any, error) {
	if !personalAccessRequest(current(r)) || !validID(r.PathValue("id")) {
		return nil, pgx.ErrNoRows
	}
	var raw []byte
	e := tx.QueryRow(r.Context(), `SELECT jsonb_build_object('id',d.id,'workspace_id',d.workspace_id,'title',d.title,'version',d.version,'status',d.status,'visibility',d.visibility,'owner_name',u.name,'created_at',d.created_at,'updated_at',d.updated_at,'kind',coalesce(k.kind,'page'),'classification',coalesce(k.classification,'internal'),'last_reviewed_at',k.last_reviewed_at,'review_period_days',k.review_period_days,'can_write',madi_document_allowed($2,d.id,true),'tags',`+documentSummaryTags+`,'stale',d.updated_at<now()-interval '90 days') FROM documents d JOIN users u ON u.id=d.owner_id LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,$3) FOR SHARE OF d`, r.PathValue("id"), current(r).ID, write).Scan(&raw)
	if e != nil {
		return nil, e
	}
	var out map[string]any
	if e = json.Unmarshal(raw, &out); e != nil {
		return nil, e
	}
	if e = documentAccessActorTx(r, tx, str(out, "workspace_id"), write); e != nil {
		return nil, e
	}
	return out, nil
}
func (s *Server) documentPassport(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	doc, e := s.worksetDocumentTx(r, tx, false)
	if e != nil {
		apiError(w, 404, "현재 접근 가능한 문서가 없습니다")
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT jsonb_build_object('version',document_version,'from',valid_from::text,'until',valid_until::text) FROM knowledge_validity_periods WHERE document_id=$1 ORDER BY ordinal LIMIT 100`, r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	periods := []map[string]any{}
	for rows.Next() {
		var raw []byte
		var item map[string]any
		if e = rows.Scan(&raw); e != nil {
			break
		}
		if e = json.Unmarshal(raw, &item); e != nil {
			break
		}
		periods = append(periods, item)
	}
	rows.Close()
	if e == nil {
		e = rows.Err()
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	doc["validity_periods"] = periods
	doc["notice"] = "문서 여권은 현재 소유·버전·분류·유효기간의 사실 목록입니다. 내용의 정확성 인증이나 게시 승인, 실제 열람자 범위의 보증이 아니며 상위·공간 권한이 함께 적용됩니다."
	jsonResponse(w, 200, doc)
}

var cleanupWord = regexp.MustCompile(`[\p{L}][\p{L}\p{N}_-]{1,23}`)

func cleanupCandidates(title, md string, tags []string) ([]string, []string) {
	idx := indexMarkdown(md)
	titles := []string{title}
	seenTitle := map[string]bool{title: true}
	for _, h := range idx.Headings {
		candidate := strings.TrimSpace(truncateAIRunes(h.Text, 120))
		if candidate != "" && !seenTitle[candidate] {
			titles = append(titles, candidate)
			seenTitle[candidate] = true
		}
		if len(titles) >= 4 {
			break
		}
	}
	scores := map[string]int{}
	for _, tag := range idx.Tags {
		scores[tag] += 20
	}
	for _, h := range idx.Headings {
		for _, word := range cleanupWord.FindAllString(h.Text, -1) {
			scores[word] += 2
		}
	}
	stop := map[string]bool{"문서": true, "제목": true, "목차": true, "개요": true, "참고": true, "the": true, "and": true, "this": true, "that": true}
	for _, word := range cleanupWord.FindAllString(truncateAIRunes(idx.Plain, 20000), -1) {
		if !stop[strings.ToLower(word)] {
			scores[word]++
		}
	}
	words := []string{}
	for word := range scores {
		if !stop[strings.ToLower(word)] && len(word) <= 200 {
			words = append(words, word)
		}
	}
	sort.Slice(words, func(i, j int) bool {
		if scores[words[i]] == scores[words[j]] {
			return words[i] < words[j]
		}
		return scores[words[i]] > scores[words[j]]
	})
	result := []string{}
	seen := map[string]bool{}
	for _, word := range append(tags, words...) {
		if !seen[word] && word != "" {
			result = append(result, word)
			seen[word] = true
		}
		if len(result) >= 32 {
			break
		}
	}
	return titles, result
}
func cleanupFrontMatterTags(md string, tags []string) (string, bool, error) {
	if !strings.HasPrefix(md, "---\n") && !strings.HasPrefix(md, "---\r\n") {
		return md, false, nil
	}
	offset, start, end := 0, 0, 0
	for i, line := range strings.SplitAfter(md, "\n") {
		offset += len(line)
		if i == 0 {
			start = offset
			continue
		}
		if strings.TrimSpace(line) == "---" {
			end = offset - len(line)
			break
		}
	}
	if end == 0 {
		return "", false, errors.New("Front Matter 경계를 확인하세요")
	}
	var document yaml.Node
	if e := yaml.Unmarshal([]byte(md[start:end]), &document); e != nil {
		return "", false, e
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return md, false, nil
	}
	root := document.Content[0]
	index := -1
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "tags" {
			index = i + 1
			break
		}
	}
	if index < 0 {
		return md, false, nil
	}
	var value yaml.Node
	if e := value.Encode(tags); e != nil {
		return "", false, e
	}
	value.HeadComment = root.Content[index].HeadComment
	value.LineComment = root.Content[index].LineComment
	value.FootComment = root.Content[index].FootComment
	root.Content[index] = &value
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if e := encoder.Encode(&document); e != nil {
		return "", false, e
	}
	_ = encoder.Close()
	return md[:start] + buf.String() + md[end:], true, nil
}
func (s *Server) documentCleanupPreview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int      `json:"expected_version"`
		Title   *string  `json:"title"`
		Tags    []string `json:"tags"`
	}
	if decode(r, &in) != nil || in.Version < 1 || len(in.Tags) > 32 {
		apiError(w, 400, "현재 버전과 최대32개 태그를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	doc, e := s.worksetDocumentTx(r, tx, true)
	if e != nil {
		apiError(w, 404, "현재 편집 가능한 문서가 없습니다")
		return
	}
	if number(doc, "version", 0) != in.Version {
		apiError(w, 409, "문서가 변경되었습니다. 현재 원문에서 후보를 다시 확인하세요")
		return
	}
	var md string
	var tagsRaw []byte
	if e = tx.QueryRow(r.Context(), `SELECT markdown,tags FROM documents WHERE id=$1 AND octet_length(markdown)<=1048576 AND octet_length(tags::text)<=32768`, r.PathValue("id")).Scan(&md, &tagsRaw); errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 413, "정리 미리보기는 원문1MiB·태그32KiB까지 지원합니다")
		return
	} else if e != nil {
		respond(w, nil, e)
		return
	}
	tags := []string{}
	if e = json.Unmarshal(tagsRaw, &tags); e != nil {
		respond(w, nil, e)
		return
	}
	titles, candidates := cleanupCandidates(str(doc, "title"), md, tags)
	if len(tags) > 32 {
		apiError(w, 413, "정리 후보는 기존 태그가 32개 이하인 문서에서 지원합니다")
		return
	}
	metadata, e := s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), r.PathValue("id"), str(doc, "workspace_id"), map[string]any{"titles": titles, "tags": candidates})
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	safe, _ := metadata.Value.(map[string]any)
	titles, candidates = listStrings(safe["titles"]), listStrings(safe["tags"])
	chosenTitle := str(doc, "title")
	if in.Title != nil {
		chosenTitle = strings.TrimSpace(*in.Title)
		if !oneOf(chosenTitle, titles...) {
			apiError(w, 400, "현재 본문에서 제안한 제목을 선택하세요")
			return
		}
	}
	chosenTags := tags
	if in.Tags != nil {
		chosenTags = []string{}
		seen := map[string]bool{}
		for _, tag := range in.Tags {
			if !oneOf(tag, candidates...) {
				apiError(w, 400, "현재 본문에서 제안한 태그를 선택하세요")
				return
			}
			if !seen[tag] {
				chosenTags = append(chosenTags, tag)
				seen[tag] = true
			}
		}
	}
	proposed, formatted := md, false
	if in.Tags != nil && string(jsonValue(tags)) != string(jsonValue(chosenTags)) {
		proposed, formatted, e = cleanupFrontMatterTags(md, chosenTags)
	}
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	protected, e := s.protectCanonicalDocumentTx(r.Context(), tx, current(r), r.PathValue("id"), str(doc, "workspace_id"), chosenTitle, proposed, chosenTags, []string{}, map[string]any{})
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	jsonResponse(w, 200, map[string]any{"document_id": r.PathValue("id"), "version": in.Version, "before": map[string]any{"title": doc["title"], "tags": tags, "markdown": md}, "after": map[string]any{"title": protected.Title, "tags": protected.Tags, "markdown": protected.Markdown}, "title_candidates": safe["titles"], "tag_candidates": safe["tags"], "front_matter_reformatted": formatted, "notice": "현재 본문의 제목·명시 태그·단어 빈도에서 만든 규칙 기반 후보입니다. AI나 외부 서비스로 보내지 않으며 정확성을 보장하지 않습니다. 적용은 별도 선택과 기존 문서 저장 CAS·민감정보·게시 정책을 거칩니다."})
}
