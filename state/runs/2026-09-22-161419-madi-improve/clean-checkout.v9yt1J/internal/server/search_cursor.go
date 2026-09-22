package server

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"strings"
	"time"
)

type searchCursor struct {
	Version    int            `json:"version"`
	Actor      string         `json:"actor"`
	Workspace  string         `json:"workspace"`
	QueryHash  string         `json:"query_hash"`
	Dictionary int            `json:"dictionary"`
	AsOf       time.Time      `json:"as_of"`
	Sort       string         `json:"sort"`
	After      map[string]any `json:"after"`
}

var searchFilterKeys = []string{"q", "type", "space_id", "author_id", "tag", "status", "from", "to", "has_attachment", "sort", "group", "limit"}

func searchFilterFingerprint(params url.Values) string {
	filtered := url.Values{}
	for _, key := range searchFilterKeys {
		if v := params.Get(key); v != "" {
			filtered.Set(key, v)
		}
	}
	return digest(filtered.Encode())
}

func (s *Server) readSearchCursor(raw string, p *Principal, wid string, params url.Values, revision int) (searchCursor, error) {
	result := searchCursor{Version: 1, Actor: p.ID, Workspace: wid, QueryHash: searchFilterFingerprint(params), Dictionary: revision, AsOf: time.Now().UTC(), Sort: params.Get("sort"), After: map[string]any{}}
	if raw == "" {
		return result, nil
	}
	denied := errors.New("검색 기준이 달라졌거나 페이지가 만료되었습니다. 첫 페이지에서 다시 검색하세요")
	if len(raw) > 8192 || !strings.HasPrefix(raw, "enc:") {
		return result, denied
	}
	plain, e := s.decrypt(raw)
	if e != nil {
		return result, denied
	}
	var previous searchCursor
	if json.Unmarshal([]byte(plain), &previous) != nil || previous.Version != 1 || previous.Actor != p.ID || previous.Workspace != wid || previous.QueryHash != result.QueryHash || previous.Dictionary != revision || previous.Sort != result.Sort || previous.AsOf.Before(time.Now().Add(-15*time.Minute)) || previous.AsOf.After(time.Now().Add(time.Minute)) {
		return result, denied
	}
	score, ok := previous.After["score"].(float64)
	if !ok || math.IsNaN(score) || math.IsInf(score, 0) || len(str(previous.After, "id")) > 512 || len(str(previous.After, "title")) > 4096 || !oneOf(str(previous.After, "kind"), universalSearchKinds...) {
		return result, denied
	}
	if _, e = time.Parse(time.RFC3339Nano, str(previous.After, "updated_at")); e != nil {
		return result, denied
	}
	return previous, nil
}

func (s *Server) nextSearchCursor(cursor searchCursor, last map[string]any) (string, error) {
	cursor.After = map[string]any{"score": last["score"], "updated_at": last["updated_at"], "title": last["title"], "kind": last["kind"], "id": last["id"]}
	raw, e := json.Marshal(cursor)
	if e != nil {
		return "", e
	}
	return s.encrypt(string(raw))
}

func searchCursorPredicate(sort string) string {
	const score = "($22::jsonb->>'score')::float8"
	const stamp = "($22::jsonb->>'updated_at')::timestamptz"
	const ties = "(h.kind,h.id)>($22::jsonb->>'kind',$22::jsonb->>'id')"
	var after string
	switch sort {
	case "newest":
		after = "h.updated_at<" + stamp + " OR (h.updated_at=" + stamp + " AND (h.score<" + score + " OR (h.score=" + score + " AND " + ties + ")))"
	case "oldest":
		after = "h.updated_at>" + stamp + " OR (h.updated_at=" + stamp + " AND (h.score<" + score + " OR (h.score=" + score + " AND " + ties + ")))"
	case "title":
		after = "(h.title,h.kind,h.id)>($22::jsonb->>'title',$22::jsonb->>'kind',$22::jsonb->>'id')"
	default:
		after = "h.score<" + score + " OR (h.score=" + score + " AND (h.updated_at<" + stamp + " OR (h.updated_at=" + stamp + " AND " + ties + ")))"
	}
	return " WHERE (NOT ($22::jsonb ? 'id') OR (" + after + "))"
}

// Group before pagination, not just within a page of raw fragment hits.
func groupedSearchSQL(sql string) string {
	const grouped = `), grouped_ranked AS (
 SELECT h.*,CASE WHEN document_id<>'' THEN 'document:'||document_id ELSE kind||':'||id END group_key,
 row_number() OVER(PARTITION BY CASE WHEN document_id<>'' THEN 'document:'||document_id ELSE kind||':'||id END ORDER BY (kind='document') DESC,score DESC,updated_at DESC,kind,id) AS head_rank,
 row_number() OVER(PARTITION BY CASE WHEN document_id<>'' THEN 'document:'||document_id ELSE kind||':'||id END ORDER BY score DESC,updated_at DESC,kind,id) AS hit_rank,
 max(score) OVER(PARTITION BY CASE WHEN document_id<>'' THEN 'document:'||document_id ELSE kind||':'||id END) AS top_score,
 count(*) OVER(PARTITION BY CASE WHEN document_id<>'' THEN 'document:'||document_id ELSE kind||':'||id END) AS hit_count
 FROM hits h
), grouped_previews AS (
 SELECT group_key,jsonb_agg(jsonb_build_object('kind',kind,'snippet',snippet,'url',url,'metadata',metadata) ORDER BY hit_rank) AS matches FROM grouped_ranked WHERE hit_rank<=3 GROUP BY group_key
), grouped_hits AS (
 SELECT h.kind,h.id,h.document_id,h.title,h.snippet,h.url,h.updated_at,h.top_score AS score,h.metadata||jsonb_build_object('matches',g.matches,'matched_count',h.hit_count) AS metadata FROM grouped_ranked h JOIN grouped_previews g USING(group_key) WHERE h.head_rank=1
)
SELECT CASE WHEN h.kind`
	sql = strings.Replace(sql, ")\nSELECT CASE WHEN h.kind", grouped, 1)
	sql = strings.Replace(sql, "'metadata',jsonb_build_object('version'", "'metadata',h.metadata||jsonb_build_object('version'", 1)
	return strings.Replace(sql, "ELSE to_jsonb(h) END FROM hits h", "ELSE to_jsonb(h) END FROM grouped_hits h", 1)
}
