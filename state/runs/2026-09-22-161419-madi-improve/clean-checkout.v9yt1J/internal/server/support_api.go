package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (s *Server) supportMeta(w http.ResponseWriter, r *http.Request) {
	cookie, e := supportCookie(r)
	if e != nil {
		apiError(w, 403, errSupportDenied.Error())
		return
	}
	minutes, e := supportAuthority(r.Context(), s.DB, current(r).ID, cookie)
	cfg, err := s.settings(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"enabled": boolean(cfg, "support_enabled"), "can_start": e == nil, "max_minutes": minutes, "notice": "관리자와 대상 사용자가 모두 현재 열람 가능한 정보만 보여주는 읽기 전용 진단입니다. 대상 사용자의 권한으로 로그인하지 않습니다. 다른 탭은 원래 본인의 권한을 유지합니다."})
}
func (s *Server) supportTargets(w http.ResponseWriter, r *http.Request) {
	cookie, e := supportCookie(r)
	if e != nil {
		apiError(w, 403, errSupportDenied.Error())
		return
	}
	if _, e = supportAuthority(r.Context(), s.DB, current(r).ID, cookie); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		apiError(w, 400, "검색어는 200바이트 이하여야 합니다")
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',u.id,'name',u.name) FROM users u WHERE u.id<>$1 AND NOT u.disabled AND u.kind='user' AND ($2='' OR u.name ILIKE '%'||$2||'%') AND EXISTS(SELECT 1 FROM workspace_members a JOIN workspace_members t ON t.workspace_id=a.workspace_id WHERE a.user_id=$1 AND t.user_id=u.id) ORDER BY u.name,u.id LIMIT 100`, current(r).ID, q)
	respond(w, rows, e)
}
func (s *Server) startSupport(w http.ResponseWriter, r *http.Request) {
	cookie, e := supportCookie(r)
	if e != nil {
		apiError(w, 403, errSupportDenied.Error())
		return
	}
	maxMinutes, e := supportAuthority(r.Context(), s.DB, current(r).ID, cookie)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var in struct {
		TargetID string `json:"target_id"`
		Reason   string `json:"reason"`
		Minutes  int    `json:"duration_minutes"`
	}
	if decode(r, &in) != nil || !validID(in.TargetID) || in.TargetID == current(r).ID || in.Minutes < 1 || in.Minutes > maxMinutes || utf8.RuneCountInString(strings.TrimSpace(in.Reason)) < 10 || utf8.RuneCountInString(in.Reason) > 500 || !utf8.ValidString(in.Reason) || strings.ContainsRune(in.Reason, 0) {
		apiError(w, 400, "다른 일반 사용자·사유 10~500자·허용된 지원 시간을 입력하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,916))`, current(r).ID); e != nil {
		respond(w, nil, e)
		return
	}
	currentMax, e := supportAuthority(r.Context(), tx, current(r).ID, cookie)
	if e != nil || in.Minutes > currentMax {
		apiError(w, 403, errSupportDenied.Error())
		return
	}
	var targetName string
	e = tx.QueryRow(r.Context(), `SELECT u.name FROM users u WHERE u.id=$2 AND NOT u.disabled AND u.kind='user' AND EXISTS(SELECT 1 FROM workspace_members a JOIN workspace_members t ON t.workspace_id=a.workspace_id WHERE a.user_id=$1 AND t.user_id=u.id) FOR SHARE OF u`, current(r).ID, in.TargetID).Scan(&targetName)
	if e != nil {
		apiError(w, 403, "공통 워크스페이스에 속한 활성 일반 사용자를 선택하세요")
		return
	}
	var active int
	if e = tx.QueryRow(r.Context(), `SELECT count(*) FROM support_sessions WHERE operator_id=$1 AND status='active' AND expires_at>now()`, current(r).ID).Scan(&active); e != nil {
		respond(w, nil, e)
		return
	}
	if active >= 3 {
		apiError(w, 429, "지원 진단은 동시에 3개까지 열 수 있습니다")
		return
	}
	reason, findings, mode, e := supportReasonTx(r.Context(), tx, strings.TrimSpace(in.Reason))
	if e != nil {
		if !WriteProtectionError(w, e) {
			respond(w, nil, e)
		}
		return
	}
	session := supportSession{ID: newID(), OperatorID: current(r).ID, TargetID: in.TargetID, TargetName: targetName, Reason: reason, Status: "active", SessionHash: cookie}
	e = tx.QueryRow(r.Context(), `INSERT INTO support_sessions(id,operator_id,target_id,session_hash,reason,expires_at) VALUES($1,$2,$3,$4,$5,now()+make_interval(mins=>$6)) RETURNING created_at,expires_at`, session.ID, session.OperatorID, session.TargetID, cookie, reason, in.Minutes).Scan(&session.CreatedAt, &session.ExpiresAt)
	if e == nil {
		e = supportAuditTx(r.Context(), tx, r, session, "SUPPORT_START", session.ID, "allowed", map[string]any{"expires_at": session.ExpiresAt, "protection_mode": mode, "findings": findings})
	}
	if e == nil && len(findings) > 0 {
		_, e = tx.Exec(r.Context(), `INSERT INTO protection_events(id,user_id,action,mode,findings) VALUES($1,$2,'support.reason',$3,$4)`, newID(), session.OperatorID, mode, jsonValue(findings))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 201, map[string]any{"session": session, "reason_masked": reason != strings.TrimSpace(in.Reason), "protection_mode": mode, "findings": findings})
}
func (s *Server) listSupport(w http.ResponseWriter, r *http.Request) {
	if _, e := supportCookie(r); e != nil {
		apiError(w, 403, errSupportDenied.Error())
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',x.id,'target_name',u.name,'reason',x.reason,'status',CASE WHEN x.status='active' AND x.expires_at<=now() THEN 'expired' ELSE x.status END,'created_at',x.created_at,'expires_at',x.expires_at) FROM support_sessions x JOIN users u ON u.id=x.target_id WHERE x.operator_id=$1 ORDER BY x.created_at DESC LIMIT 50`, current(r).ID)
	respond(w, rows, e)
}
func (s *Server) getSupport(w http.ResponseWriter, r *http.Request) {
	session, e := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if e != nil {
		s.supportDenied(w, r, session, "session")
		return
	}
	// A read-only heartbeat can pin every currently rendered resource. This does
	// not grant access: content endpoints independently enforce both principals.
	for _, kind := range []string{"document_ids", "database_ids"} {
		ids := strings.FieldsFunc(r.URL.Query().Get(kind), func(c rune) bool { return c == ',' })
		if len(ids) > 101 {
			apiError(w, 400, "한 번에 확인할 수 있는 진단 자료는 종류별 101개입니다")
			return
		}
		for _, id := range ids {
			allowed := validID(id)
			if kind == "document_ids" {
				allowed = allowed && supportDocumentAllowed(r.Context(), s.DB, session, id)
			} else {
				allowed = allowed && s.canDatabase(automationRequest(r.Context(), session.Operator, "GET", nil), id, false) && s.canDatabase(automationRequest(r.Context(), session.Target, "GET", nil), id, false)
			}
			if !allowed {
				s.supportDenied(w, r, session, id)
				return
			}
		}
	}
	if wid := r.URL.Query().Get("formula_workspace_id"); wid != "" && (!validID(wid) || !s.canFeature(r.Context(), session.Operator, wid, "database-formula") || !s.canFeature(r.Context(), session.Target, wid, "database-formula")) {
		s.supportDenied(w, r, session, wid)
		return
	}
	workspaces, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',w.id,'name',w.name,'target_role',t.role) FROM workspaces w JOIN workspace_members a ON a.workspace_id=w.id AND a.user_id=$1 JOIN workspace_members t ON t.workspace_id=w.id AND t.user_id=$2 ORDER BY w.name,w.id LIMIT 100`, session.OperatorID, session.TargetID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"session": session, "workspaces": workspaces, "mode": "read_only_acl_intersection", "restrictions": []string{"관리자와 대상 사용자의 현재 ACL 교집합", "타인의 개인 문서·개인 대화 이력 접근 우회 불가", "모든 쓰기·AI·외부 전송·첨부 다운로드 차단", "블록 실행·플러그인·외부 이미지 실행 없음", "대상 계정으로 로그인하지 않음; 다른 탭은 본인 권한"}})
}
func (s *Server) endSupport(w http.ResponseWriter, r *http.Request) {
	cookie, e := supportCookie(r)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	session, e := supportLoad(r.Context(), s.DB, r.PathValue("id"), current(r).ID, cookie)
	if e != nil {
		apiError(w, 404, "이 로그인에서 시작한 지원 진단을 찾을 수 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	tag, e := tx.Exec(r.Context(), `UPDATE support_sessions SET status='ended',ended_at=now() WHERE id=$1 AND status='active'`, session.ID)
	if e == nil && tag.RowsAffected() > 0 {
		e = supportAuditTx(r.Context(), tx, r, session, "SUPPORT_END", session.ID, "ended", nil)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"ended": true}, e)
}

func (s *Server) supportDocuments(w http.ResponseWriter, r *http.Request) {
	session, e := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if e != nil {
		s.supportDenied(w, r, session, "documents")
		return
	}
	wid := r.URL.Query().Get("workspace_id")
	if !supportWorkspaceAllowed(r.Context(), s.DB, session, wid) {
		s.supportDenied(w, r, session, wid)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 1000 {
		apiError(w, 400, "검색어는 1000바이트 이하여야 합니다")
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'version',d.version,'status',d.status,'visibility',d.visibility,'updated_at',d.updated_at) FROM documents d WHERE d.workspace_id=$3 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND madi_document_allowed($2,d.id,false) AND ($4='' OR d.search_vector@@websearch_to_tsquery('simple',$4) OR d.title ILIKE '%'||$4||'%' OR d.markdown ILIKE '%'||$4||'%') ORDER BY d.updated_at DESC,d.id LIMIT 100`, session.OperatorID, session.TargetID, wid, q)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = supportAccess(r.Context(), s.DB, r, session.ID); e != nil {
		s.supportDenied(w, r, session, wid)
		return
	}
	for _, row := range rows {
		if !supportDocumentAllowed(r.Context(), s.DB, session, str(row, "id")) {
			s.supportDenied(w, r, session, wid)
			return
		}
	}
	if e = s.supportAudit(r, session, "SUPPORT_DOCUMENT_LIST", wid, "allowed", map[string]any{"rows": len(rows), "search": q != ""}); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, rows)
}
func (s *Server) supportDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("document")
	session, e := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if e != nil || !supportDocumentAllowed(r.Context(), s.DB, session, id) {
		s.supportDenied(w, r, session, id)
		return
	}
	v, e := s.one(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'markdown',d.markdown,'version',d.version,'status',d.status,'visibility',d.visibility,'tags',d.tags,'updated_at',d.updated_at) FROM documents d WHERE id=$3 AND deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND madi_document_allowed($2,d.id,false)`, session.OperatorID, session.TargetID, id)
	if e != nil {
		s.supportDenied(w, r, session, id)
		return
	}
	if _, e = supportAccess(r.Context(), s.DB, r, session.ID); e != nil || !supportDocumentAllowed(r.Context(), s.DB, session, id) {
		s.supportDenied(w, r, session, id)
		return
	}
	if e = s.supportAudit(r, session, "SUPPORT_DOCUMENT_READ", id, "allowed", map[string]any{"version": v["version"]}); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, v)
}
func (s *Server) supportDatabases(w http.ResponseWriter, r *http.Request) {
	session, e := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if e != nil {
		s.supportDenied(w, r, session, "databases")
		return
	}
	wid := r.URL.Query().Get("workspace_id")
	if !supportWorkspaceAllowed(r.Context(), s.DB, session, wid) {
		s.supportDenied(w, r, session, wid)
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'name',d.name) FROM databases d WHERE d.workspace_id=$3 AND madi_space_allowed($1,d.space_id,false) AND madi_space_allowed($2,d.space_id,false) ORDER BY d.name,d.id LIMIT 100`, session.OperatorID, session.TargetID, wid)
	if e == nil {
		_, e = supportAccess(r.Context(), s.DB, r, session.ID)
	}
	if e == nil {
		e = s.supportAudit(r, session, "SUPPORT_DATABASE_LIST", wid, "allowed", map[string]any{"rows": len(rows)})
	}
	if e != nil {
		s.supportDenied(w, r, session, wid)
		return
	}
	jsonResponse(w, 200, rows)
}
func (s *Server) supportDatabase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("database")
	session, e := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if e != nil {
		s.supportDenied(w, r, session, id)
		return
	}
	actorRequest := automationRequest(r.Context(), session.Operator, "GET", nil)
	targetRequest := automationRequest(r.Context(), session.Target, "GET", nil)
	if !s.canDatabase(actorRequest, id, false) || !s.canDatabase(targetRequest, id, false) {
		s.supportDenied(w, r, session, id)
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			apiError(w, 400, "조회 행 수는 1~100 사이여야 합니다")
			return
		}
		limit = n
	}
	engine := newAdvancedEngine(s, actorRequest)
	engine.authorizeDatabase = func(id string) bool { return s.canDatabase(targetRequest, id, false) }
	db, e := engine.database(id)
	if e != nil {
		s.supportDenied(w, r, session, id)
		return
	}
	rows, e := s.rows(r.Context(), `SELECT to_jsonb(v) FROM database_rows v WHERE database_id=$1 ORDER BY id LIMIT $2`, id, limit)
	if e != nil {
		respond(w, nil, e)
		return
	}
	result := []map[string]any{}
	formulaAllowed := s.canFeature(r.Context(), session.Operator, db.workspaceID, "database-formula") && s.canFeature(r.Context(), session.Target, db.workspaceID, "database-formula")
	for _, row := range rows {
		engine.rows[id+":"+str(row, "id")] = row
		values, failures := map[string]any{}, map[string]any{}
		for _, property := range db.properties {
			key := str(property, "id")
			if oneOf(str(property, "type"), "button") {
				continue
			}
			if oneOf(str(property, "type"), "formula", "rollup") && !formulaAllowed {
				failures[key] = "현재 양측의 기능 정책 교집합에서 수식 계산이 꺼져 있습니다"
				continue
			}
			if str(property, "type") == "relation" && !engine.authorizeDatabase(str(property, "target_database_id")) {
				failures[key] = "관계 대상은 양측의 권한 교집합에 없습니다"
				continue
			}
			value, ex := engine.value(db, row, key, 0)
			if ex != nil {
				failures[key] = "참조·계산 결과를 현재 양측 권한으로 표시할 수 없습니다"
			} else {
				values[key] = value
			}
		}
		result = append(result, map[string]any{"id": row["id"], "values": values, "errors": failures})
	}
	if _, e = supportAccess(r.Context(), s.DB, r, session.ID); e != nil {
		s.supportDenied(w, r, session, id)
		return
	}
	dependencies := []string{}
	for target := range engine.databases {
		if !s.canDatabase(actorRequest, target, false) || !s.canDatabase(targetRequest, target, false) {
			s.supportDenied(w, r, session, id)
			return
		}
		dependencies = append(dependencies, target)
	}
	if formulaAllowed && (!s.canFeature(r.Context(), session.Operator, db.workspaceID, "database-formula") || !s.canFeature(r.Context(), session.Target, db.workspaceID, "database-formula")) {
		s.supportDenied(w, r, session, id)
		return
	}
	output := map[string]any{"id": id, "properties": db.properties, "rows": result, "limit": limit, "read_only": true, "dependency_database_ids": dependencies, "formula_workspace_id": ""}
	if formulaAllowed {
		output["formula_workspace_id"] = db.workspaceID
	}
	encoded, err := json.Marshal(output)
	if err != nil || len(encoded) > 4<<20 || len(dependencies) > 100 {
		apiError(w, 413, "진단 데이터가 4MiB 또는 참조 DB 100개를 초과했습니다. 조회 행 수를 줄여 주세요")
		return
	}
	if e = s.supportAudit(r, session, "SUPPORT_DATABASE_READ", id, "allowed", map[string]any{"rows": len(result), "dependencies": len(engine.databases)}); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, output)
}
func (s *Server) supportGraph(w http.ResponseWriter, r *http.Request) {
	session, e := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if e != nil {
		s.supportDenied(w, r, session, "graph")
		return
	}
	wid := r.URL.Query().Get("workspace_id")
	if !supportWorkspaceAllowed(r.Context(), s.DB, session, wid) {
		s.supportDenied(w, r, session, wid)
		return
	}
	docs, e := s.rows(r.Context(), `SELECT `+limitedGraphDocumentJSON+` FROM documents d WHERE d.workspace_id=$3 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND madi_document_allowed($2,d.id,false) ORDER BY d.id LIMIT 101`, session.OperatorID, session.TargetID, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	truncated := len(docs) > 100
	if truncated {
		docs = docs[:100]
	}
	lookup := map[string]string{}
	nodes, edges := []map[string]any{}, []map[string]any{}
	edgesTruncated := false
	for _, d := range docs {
		truncated = truncated || boolean(d, "source_truncated") || boolean(d, "aliases_truncated")
		lookup[str(d, "title")] = str(d, "id")
		for _, alias := range listStrings(d["aliases"]) {
			lookup[alias] = str(d, "id")
		}
		nodes = append(nodes, map[string]any{"id": d["id"], "title": d["title"]})
	}
	for _, d := range docs {
		for _, link := range wikiTargets(str(d, "markdown")) {
			if target := lookup[link]; target != "" {
				if len(edges) >= 500 {
					edgesTruncated = true
					break
				}
				edges = append(edges, map[string]any{"source": d["id"], "target": target})
			}
		}
		if edgesTruncated {
			break
		}
	}
	if _, e = supportAccess(r.Context(), s.DB, r, session.ID); e != nil {
		s.supportDenied(w, r, session, wid)
		return
	}
	for _, d := range docs {
		if !supportDocumentAllowed(r.Context(), s.DB, session, str(d, "id")) {
			s.supportDenied(w, r, session, wid)
			return
		}
	}
	if e = s.supportAudit(r, session, "SUPPORT_GRAPH_READ", wid, "allowed", map[string]any{"nodes": len(nodes), "edges": len(edges)}); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"nodes": nodes, "edges": edges, "limit": 100, "source_character_limit": 32000, "alias_limit": 32, "truncated": truncated || edgesTruncated})
}
func (s *Server) supportForbidden(w http.ResponseWriter, r *http.Request) {
	session, _ := supportAccess(r.Context(), s.DB, r, r.PathValue("id"))
	if validID(session.ID) {
		_ = s.supportAudit(r, session, "SUPPORT_FORBIDDEN", r.PathValue("rest"), "denied", map[string]any{"method": r.Method})
	}
	apiError(w, 403, "지원 진단에서는 쓰기·AI·외부 전송·첨부 다운로드·개인 설정을 사용할 수 없습니다")
}
func (s *Server) registerSupport() {
	s.admin("GET /api/v1/support/meta", s.supportMeta)
	s.admin("GET /api/v1/support/targets", s.supportTargets)
	s.admin("GET /api/v1/support/sessions", s.listSupport)
	s.admin("POST /api/v1/support/sessions", s.startSupport)
	s.admin("GET /api/v1/support/sessions/{id}", s.getSupport)
	s.admin("POST /api/v1/support/sessions/{id}/end", s.endSupport)
	s.admin("GET /api/v1/support/sessions/{id}/documents", s.supportDocuments)
	s.admin("GET /api/v1/support/sessions/{id}/documents/{document}", s.supportDocument)
	s.admin("GET /api/v1/support/sessions/{id}/databases", s.supportDatabases)
	s.admin("GET /api/v1/support/sessions/{id}/databases/{database}", s.supportDatabase)
	s.admin("GET /api/v1/support/sessions/{id}/graph", s.supportGraph)
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		s.admin(method+" /api/v1/support/sessions/{id}/{rest...}", s.supportForbidden)
	}
}
