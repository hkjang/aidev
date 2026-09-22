package server

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge.sql
var knowledgeSchema string

func (s *Server) migrateKnowledge(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgeSchema)
	return e
}
func (s *Server) registerKnowledge() {
	s.handle("GET /api/v1/documents/{id}/knowledge", s.getDocumentKnowledge)
	s.handle("PUT /api/v1/documents/{id}/knowledge", s.updateDocumentKnowledge)
	s.handle("GET /api/v1/documents/{id}/knowledge/history", s.knowledgeHistory)
	s.handle("POST /api/v1/documents/{id}/reviewed", s.markDocumentReviewed)
	s.handle("GET /api/v1/knowledge/health", s.knowledgeHealth)
	s.handle("GET /api/v1/knowledge/analytics", s.knowledgeAnalytics)
}
func (s *Server) documentKnowledge(r *http.Request, id string) (map[string]any, error) {
	v, e := s.one(r.Context(), `SELECT jsonb_build_object('document_id',d.id,'workspace_id',d.workspace_id,'owner_id',d.owner_id,'owner_name',u.name,'owner_disabled',(u.disabled OR NOT madi_document_allowed(d.owner_id,d.id,true)),'version',d.version,'kind',coalesce(m.kind,'page'),'classification',coalesce(m.classification,sp.classification,'internal'),'reviewer_id',m.reviewer_id,'reviewer_name',ru.name,'maintainer_id',m.maintainer_id,'maintainer_name',mu.name,'review_period_days',m.review_period_days,'last_reviewed_at',m.last_reviewed_at,'legal_hold',coalesce(m.legal_hold,false),'retain_until',m.retain_until,'system_metadata',coalesce(m.system_metadata,'{}')) FROM documents d JOIN users u ON u.id=d.owner_id LEFT JOIN knowledge_document_meta m ON m.document_id=d.id LEFT JOIN spaces sp ON sp.id=d.space_id LEFT JOIN users ru ON ru.id=m.reviewer_id LEFT JOIN users mu ON mu.id=m.maintainer_id WHERE d.id=$1`, id)
	if e != nil {
		return nil, e
	}
	cfg, e := s.effectiveSettings(r.Context(), str(v, "workspace_id"))
	if e != nil {
		return nil, e
	}
	if v["review_period_days"] == nil {
		v["review_period_days"] = number(cfg, "review_period_days", 90)
	}
	v["can_manage"] = s.canDocument(r.Context(), current(r), id, true) && hasIntegrationScope(current(r), "document:write") && (current(r).ID == str(v, "owner_id") || s.workspaceAdmin(r, str(v, "workspace_id")))
	v["can_set_retention"] = s.workspaceAdmin(r, str(v, "workspace_id"))
	var status string
	if e = s.DB.QueryRow(r.Context(), "SELECT status FROM documents WHERE id=$1", id).Scan(&status); e != nil {
		return nil, e
	}
	v["status"] = status
	v["can_review"] = s.canDocument(r.Context(), current(r), id, true) && hasIntegrationScope(current(r), "document:write") && (boolean(v, "can_manage") || current(r).ID == str(v, "reviewer_id") || current(r).ID == str(v, "maintainer_id"))
	return v, nil
}
func (s *Server) getDocumentKnowledge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	v, e := s.documentKnowledge(r, id)
	if e == nil {
		var md string
		if e = s.DB.QueryRow(r.Context(), "SELECT markdown FROM documents WHERE id=$1", id).Scan(&md); e == nil {
			idx := indexMarkdown(md)
			v["outline"] = idx.Headings
			v["source_tags"] = idx.Tags
			v["source_links"] = idx.Links
		}
	}
	respond(w, v, e)
}
func (s *Server) updateDocumentKnowledge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "문서 관리 권한이 없습니다")
		return
	}
	old, e := s.documentKnowledge(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !boolean(old, "can_manage") {
		apiError(w, 403, "문서 소유자 또는 접근 가능한 워크스페이스 관리자만 변경할 수 있습니다")
		return
	}
	var in map[string]any
	if decode(r, &in) != nil {
		apiError(w, 400, "문서 운영 속성을 확인하세요")
		return
	}
	if number(in, "version", 0) != number(old, "version", -1) {
		apiError(w, 409, "문서가 변경되었습니다. 현재 버전을 다시 불러오세요")
		return
	}
	before := map[string]any{}
	for key, value := range old {
		before[key] = value
	}
	if status, ok := in["status"].(string); ok && status != str(old, "status") {
		if !oneOf(status, "draft", "published", "stale", "archived") {
			apiError(w, 400, "문서 상태를 확인하세요")
			return
		}
		cfg, e := s.effectiveSettings(r.Context(), str(old, "workspace_id"))
		if e != nil {
			respond(w, nil, e)
			return
		}
		if status == "published" && boolean(cfg, "approval_enabled") {
			apiError(w, 403, "관리자가 설정한 검토 승인 절차로 게시하세요")
			return
		}
		old["status"] = status
	}
	for _, key := range []string{"kind", "classification", "reviewer_id", "maintainer_id", "review_period_days", "legal_hold", "retain_until", "owner_id"} {
		if value, present := in[key]; present {
			old[key] = value
		}
	}
	if !oneOf(str(old, "kind"), "page", "note", "daily", "meeting", "decision", "runbook", "template", "inbox", "entity") || !oneOf(str(old, "classification"), "public", "internal", "confidential", "restricted") {
		apiError(w, 400, "문서 종류와 등급을 확인하세요")
		return
	}
	days := number(old, "review_period_days", 90)
	if days < 1 || days > 3650 {
		apiError(w, 400, "문서 검토 주기는 1~3650일입니다")
		return
	}
	if raw, ok := old["review_period_days"].(float64); ok && raw != float64(days) {
		apiError(w, 400, "검토 주기는 정수로 입력하세요")
		return
	}
	if _, ok := old["legal_hold"].(bool); !ok {
		apiError(w, 400, "법적 보존 여부를 확인하세요")
		return
	}
	if _, set := in["legal_hold"]; set && !s.workspaceAdmin(r, str(old, "workspace_id")) {
		apiError(w, 403, "법적 보존 설정은 워크스페이스 관리자 세션이 필요합니다")
		return
	}
	if _, set := in["retain_until"]; set && !s.workspaceAdmin(r, str(old, "workspace_id")) {
		apiError(w, 403, "보존 기한 설정은 워크스페이스 관리자 세션이 필요합니다")
		return
	}
	retention := str(old, "retain_until")
	if retention != "" {
		if _, e = time.Parse(time.RFC3339, retention); e != nil {
			apiError(w, 400, "보존 만료 시각은 RFC3339 형식이어야 합니다")
			return
		}
	}
	for _, key := range []string{"owner_id", "reviewer_id", "maintainer_id"} {
		uid := str(old, key)
		if uid == str(before, key) {
			continue
		}
		if uid == "" && key != "owner_id" {
			continue
		}
		if !validID(uid) {
			apiError(w, 400, "문서 담당자를 확인하세요")
			return
		}
		var allowed bool
		e = s.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE u.id=$1 AND NOT u.disabled AND m.workspace_id=$2 AND u.role<>'viewer' AND m.role IN ('owner','admin','editor'))`, uid, str(old, "workspace_id")).Scan(&allowed)
		if e != nil || !allowed {
			apiError(w, 400, "담당자는 활성 워크스페이스 작성자여야 합니다")
			return
		}
		if key == "owner_id" {
			var accessible bool
			_ = s.DB.QueryRow(r.Context(), `SELECT madi_space_allowed($1,d.space_id,true) AND (d.parent_id IS NULL OR madi_document_allowed($1,d.parent_id,false)) FROM documents d WHERE d.id=$2`, uid, id).Scan(&accessible)
			if !accessible {
				apiError(w, 400, "새 소유자는 상위 문서와 공간에 접근할 수 있어야 합니다")
				return
			}
		} else {
			var accessible bool
			_ = s.DB.QueryRow(r.Context(), "SELECT madi_document_allowed($1,$2,false)", uid, id).Scan(&accessible)
			if !accessible {
				apiError(w, 400, "검토자·유지관리자는 문서 열람 권한이 필요합니다")
				return
			}
		}
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	tag, e := tx.Exec(r.Context(), "UPDATE documents SET owner_id=$2,version=version+1,status=$5 WHERE id=$1 AND version=$3 AND deleted_at IS NULL AND madi_document_allowed($4,id,true)", id, str(old, "owner_id"), number(in, "version", 0), current(r).ID, str(old, "status"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "문서가 변경되었습니다. 현재 버전을 다시 불러오세요")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_document_meta(document_id,kind,classification,reviewer_id,maintainer_id,review_period_days,legal_hold,retain_until) VALUES($1,$2,$3,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,$6,$7,NULLIF($8,'')::timestamptz) ON CONFLICT(document_id) DO UPDATE SET kind=excluded.kind,classification=excluded.classification,reviewer_id=excluded.reviewer_id,maintainer_id=excluded.maintainer_id,review_period_days=excluded.review_period_days,legal_hold=excluded.legal_hold,retain_until=excluded.retain_until,updated_at=now()`, id, str(old, "kind"), str(old, "classification"), str(old, "reviewer_id"), str(old, "maintainer_id"), days, boolean(old, "legal_hold"), retention)
	for _, prefix := range []string{"owner", "reviewer", "maintainer"} {
		if e != nil {
			break
		}
		uid := str(old, prefix+"_id")
		if uid == str(before, prefix+"_id") {
			continue
		}
		old[prefix+"_name"] = nil
		if uid != "" {
			var name string
			e = tx.QueryRow(r.Context(), "SELECT name FROM users WHERE id=$1", uid).Scan(&name)
			old[prefix+"_name"] = name
		}
	}
	old["version"] = number(in, "version", 0) + 1
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO knowledge_document_history(id,document_id,user_id,document_version,action,before_data,after_data) VALUES($1,$2,$3,$4,'policy',$5,$6)", newID(), id, current(r).ID, number(in, "version", 0)+1, jsonValue(before), jsonValue(old))
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", id, current(r).ID)
	}
	if e == nil {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.updated", WorkspaceID: str(old, "workspace_id"), ResourceID: id, After: map[string]any{"id": id, "owner_id": old["owner_id"], "version": number(in, "version", 0) + 1}})
	}
	if e == nil && str(before, "status") != str(old, "status") {
		e = s.enqueueEvent(r.Context(), tx, Event{Type: "document.status_changed", WorkspaceID: str(old, "workspace_id"), ResourceID: id, Before: before, After: map[string]any{"id": id, "status": old["status"], "version": number(in, "version", 0) + 1}})
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_POLICY_CHANGE", id, in)
	// An ownership transfer can revoke the caller's private-document permission.
	if !s.canDocument(r.Context(), current(r), id, false) {
		jsonResponse(w, 200, map[string]any{"ok": true, "access_changed": true})
		return
	}
	v, e := s.documentKnowledge(r, id)
	respond(w, v, e)
}
func (s *Server) markDocumentReviewed(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, true) {
		apiError(w, 403, "문서 최신성 확인 권한이 없습니다")
		return
	}
	meta, e := s.documentKnowledge(r, id)
	if e != nil {
		respond(w, nil, e)
		return
	}
	p := current(r)
	if !boolean(meta, "can_manage") && p.ID != str(meta, "reviewer_id") && p.ID != str(meta, "maintainer_id") {
		apiError(w, 403, "소유자·검토자·유지관리자만 최신성을 확인할 수 있습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var version int
	e = tx.QueryRow(r.Context(), "SELECT version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,true) FOR UPDATE", id, current(r).ID).Scan(&version)
	if e == pgx.ErrNoRows {
		apiError(w, 404, "활성 문서에 접근할 수 없습니다")
		return
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO knowledge_document_meta(document_id,classification,last_reviewed_at) VALUES($1,$2,now()) ON CONFLICT(document_id) DO UPDATE SET last_reviewed_at=now(),updated_at=now()", id, str(meta, "classification"))
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "INSERT INTO knowledge_document_history(id,document_id,user_id,document_version,action,before_data,after_data) VALUES($1,$2,$3,$4,'reviewed',$5,jsonb_build_object('last_reviewed_at',now()))", newID(), id, current(r).ID, version, jsonValue(meta))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DOCUMENT_REVIEWED", id, nil)
	v, e := s.documentKnowledge(r, id)
	respond(w, v, e)
}

type knowledgeQuality struct {
	Score int            `json:"score"`
	Flags []string       `json:"flags"`
	Parts map[string]int `json:"parts"`
}

func documentQuality(d map[string]any, idx markdownIndex, broken []string, linked bool, now time.Time) knowledgeQuality {
	q := knowledgeQuality{Flags: []string{}, Parts: map[string]int{}}
	add := func(key string, score int) { q.Parts[key] = score; q.Score += score }
	length := len([]rune(idx.Plain))
	add("completeness", min(25, length/12))
	if length < 120 {
		q.Flags = append(q.Flags, "short")
	}
	add("structure", min(10, len(idx.Headings)*5))
	if len(idx.Links) > 0 && len(broken) == 0 {
		add("references", 15)
	} else {
		add("references", 0)
	}
	if len(listStrings(d["tags"]))+len(idx.Tags) > 0 {
		add("tags", 10)
	} else {
		add("tags", 0)
		q.Flags = append(q.Flags, "no_tags")
	}
	if !boolean(d, "owner_disabled") {
		add("ownership", 20)
	} else {
		add("ownership", 0)
		q.Flags = append(q.Flags, "missing_owner")
	}
	last, e := time.Parse(time.RFC3339Nano, str(d, "reviewed_at"))
	if e != nil {
		last, _ = time.Parse(time.RFC3339Nano, str(d, "updated_at"))
	}
	if now.Sub(last) <= time.Duration(number(d, "review_period_days", 90))*24*time.Hour {
		add("freshness", 20)
	} else {
		add("freshness", 0)
		q.Flags = append(q.Flags, "stale")
	}
	if !linked {
		q.Flags = append(q.Flags, "orphan")
	}
	if len(broken) > 0 {
		q.Flags = append(q.Flags, "broken_links")
	}
	return q
}
func (s *Server) knowledgeDocuments(r *http.Request) ([]map[string]any, error) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		return nil, pgx.ErrNoRows
	}
	cfg, e := s.effectiveSettings(r.Context(), wid)
	if e != nil {
		return nil, e
	}
	return s.rows(r.Context(), `WITH sampled AS (SELECT d.id,count(*) OVER() total_visible,sum(octet_length((to_jsonb(d)-'search_vector')::text)+2048) OVER(ORDER BY d.updated_at DESC,d.id) running_bytes FROM documents d WHERE d.workspace_id=$2 AND d.deleted_at IS NULL AND `+docACL+`) SELECT (to_jsonb(d)-'search_vector')||jsonb_build_object('total_visible',sampled.total_visible,'owner_disabled',(u.disabled OR NOT madi_document_allowed(d.owner_id,d.id,true)),'owner_name',u.name,'kind',coalesce(k.kind,'page'),'classification',coalesce(k.classification,sp.classification,'internal'),'reviewed_at',coalesce(k.last_reviewed_at,d.updated_at),'review_period_days',coalesce(k.review_period_days,$3),'legal_hold',coalesce(k.legal_hold,false)) FROM sampled JOIN documents d ON d.id=sampled.id JOIN users u ON u.id=d.owner_id LEFT JOIN knowledge_document_meta k ON k.document_id=d.id LEFT JOIN spaces sp ON sp.id=d.space_id WHERE sampled.running_bytes<=16777216 ORDER BY d.updated_at DESC,d.id LIMIT 2001`, current(r).ID, wid, number(cfg, "review_period_days", 90))
}
func (s *Server) knowledgeHealth(w http.ResponseWriter, r *http.Request) {
	docs, e := s.knowledgeDocuments(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	truncated := len(docs) > 2000 || (len(docs) > 0 && number(docs[0], "total_visible", 0) > len(docs))
	if len(docs) > 2000 {
		docs = docs[:2000]
	}
	indices := map[string]markdownIndex{}
	lookup := map[string]string{}
	hashes := map[string][]string{}
	linked := map[string]bool{}
	unresolved := map[string][]string{}
	for _, d := range docs {
		id := str(d, "id")
		indices[id] = indexMarkdown(str(d, "markdown"))
		for _, name := range documentNames(d) {
			lookup[name] = id
		}
		if strings.TrimSpace(str(d, "markdown")) != "" {
			sum := sha256.Sum256([]byte(strings.TrimSpace(str(d, "markdown"))))
			key := hex.EncodeToString(sum[:])
			hashes[key] = append(hashes[key], id)
		}
	}
	for _, d := range docs {
		id := str(d, "id")
		for _, target := range indices[id].Links {
			if other := lookup[target]; other != "" {
				linked[id] = true
				linked[other] = true
			} else {
				unresolved[id] = append(unresolved[id], target)
			}
		}
	}
	duplicate := map[string]bool{}
	for _, group := range hashes {
		if len(group) > 1 {
			for _, id := range group {
				duplicate[id] = true
			}
		}
	}
	counts := map[string]int{}
	out := []map[string]any{}
	score := 0
	for _, d := range docs {
		id := str(d, "id")
		q := documentQuality(d, indices[id], unresolved[id], linked[id], time.Now())
		if duplicate[id] {
			q.Flags = append(q.Flags, "duplicate")
		}
		for _, flag := range q.Flags {
			counts[flag]++
		}
		score += q.Score
		out = append(out, map[string]any{"id": id, "title": d["title"], "owner_name": d["owner_name"], "status": d["status"], "updated_at": d["updated_at"], "classification": d["classification"], "quality": q, "unresolved_links": unresolved[id], "legal_hold": d["legal_hold"]})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i]["quality"].(knowledgeQuality).Score < out[j]["quality"].(knowledgeQuality).Score
	})
	if len(docs) > 0 {
		score /= len(docs)
	}
	jsonResponse(w, 200, map[string]any{"score": score, "documents": out, "counts": counts, "sample_size": len(docs), "truncated": truncated, "method": "규칙 기반: 완성도25 + 구조10 + 참조15 + 태그10 + 소유권20 + 최신성20. 현재 열람 가능한 최근 2,000문서·본문 합계 16MiB 이내를 평가합니다. 미해결 링크에는 권한 밖 또는 표본 밖 문서도 포함될 수 있습니다."})
}
func (s *Server) knowledgeAnalytics(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.workspaceAdmin(r, wid) {
		apiError(w, 403, "지식 활용 분석은 워크스페이스 관리자 세션이 필요합니다")
		return
	}
	v, e := s.one(r.Context(), `SELECT jsonb_build_object('active_writers',count(DISTINCT user_id) FILTER(WHERE action IN ('DOCUMENT_CREATE','DOCUMENT_UPDATE')),'active_readers',count(DISTINCT user_id) FILTER(WHERE action='DOCUMENT_READ'),'reads',count(*) FILTER(WHERE action='DOCUMENT_READ'),'writes',count(*) FILTER(WHERE action IN ('DOCUMENT_CREATE','DOCUMENT_UPDATE'))) FROM audit_logs a WHERE created_at>now()-interval '30 days' AND EXISTS(SELECT 1 FROM documents d WHERE d.id::text=a.resource AND d.workspace_id=$2 AND `+docACL+`)`, current(r).ID, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	queries, e := s.rows(r.Context(), `SELECT jsonb_build_object('query',left(details->>'query',500),'count',count(*),'last_at',max(created_at)) FROM audit_logs WHERE action='SEARCH' AND resource=$1 AND created_at>now()-interval '30 days' AND details->>'results'='0' GROUP BY left(details->>'query',500) ORDER BY count(*) DESC LIMIT 30`, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	v["zero_result_searches"] = queries
	v["period_days"] = 30
	v["note"] = "문서 읽기·쓰기 지표는 관리자의 현재 열람 권한 내에서 집계합니다. 검색 공백은 워크스페이스 검색 로그입니다."
	jsonResponse(w, 200, v)
}

// A deletion must not bypass either a legal hold or an unexpired retention rule.
func (s *Server) documentHeld(ctx context.Context, id string) bool {
	var held bool
	e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM knowledge_document_meta WHERE document_id=$1 AND (legal_hold OR retain_until>now()))", id).Scan(&held)
	return e != nil || held
}

func (s *Server) knowledgeHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "문서에 접근할 수 없습니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT to_jsonb(h)||jsonb_build_object('user_name',u.name) FROM knowledge_document_history h JOIN users u ON u.id=h.user_id WHERE document_id=$1 ORDER BY h.created_at DESC,h.id DESC LIMIT 100", id)
	respond(w, v, e)
}
