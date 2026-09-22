package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type pluginBufferedWriter struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	overflow bool
}

func (w *pluginBufferedWriter) Header() http.Header { return w.header }
func (w *pluginBufferedWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *pluginBufferedWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	if w.body.Len()+len(b) > 4<<20 {
		w.overflow = true
		return 0, errors.New("플러그인 응답은 4MB까지 지원합니다")
	}
	return w.body.Write(b)
}

func (s *Server) pluginBridge(w http.ResponseWriter, r *http.Request) {
	if !s.pluginRequestAllowed(current(r).ID, r.PathValue("id")) {
		w.Header().Set("Retry-After", "60")
		apiError(w, 429, "플러그인 요청은 사용자·플러그인별 분당 120회까지 가능합니다")
		return
	}
	var in struct {
		WorkspaceID string         `json:"workspace_id"`
		Operation   string         `json:"operation"`
		Args        map[string]any `json:"args"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if decode(r, &in) != nil {
		apiError(w, 400, "플러그인 요청 형식 또는 2MB 크기 한도를 확인하세요")
		return
	}
	if in.Args == nil {
		in.Args = map[string]any{}
	}
	pluginID := r.PathValue("id")
	manifest, caps, e := s.pluginGrant(r, pluginID, in.WorkspaceID)
	if e != nil {
		apiError(w, 403, e.Error())
		return
	}
	required := map[string]string{"documents.list": "document:read", "documents.get": "document:read", "documents.create": "document:write", "documents.update": "document:write", "documents.delete": "document:write", "databases.list": "database:read", "databases.get": "database:read", "databases.query": "database:read", "databases.createRow": "database:write", "databases.updateRow": "database:write", "databases.deleteRow": "database:write", "ai.chat": "ai:execute", "storage.get": "storage:personal", "storage.set": "storage:personal", "storage.delete": "storage:personal", "ui.notify": "ui:notify", "ui.navigate": "ui:navigate", "file.import": "file:import", "file.export": "file:export"}[in.Operation]
	if required == "" || !pluginContains(caps, required) || ((strings.HasPrefix(required, "document:") || strings.HasPrefix(required, "database:") || required == "ai:execute") && !hasIntegrationScope(current(r), required)) {
		apiError(w, 403, "이 작업에 필요한 플러그인 권한이 승인되지 않았습니다")
		return
	}
	// Attenuate the existing identity: a plugin never inherits all browser-session
	// scopes or gains access to another workspace, even when its owner is an admin.
	p := *current(r)
	p.WorkspaceID = in.WorkspaceID
	p.PluginID = pluginID
	p.ScopeRestricted = true
	p.Scopes = []string{}
	for _, cap := range caps {
		if pluginContains(keyScopes, cap) && hasIntegrationScope(current(r), cap) {
			p.Scopes = append(p.Scopes, cap)
		}
	}
	ctx := context.WithValue(r.Context(), principalKey, &p)
	next := r.Clone(ctx)
	next.Header = r.Header.Clone()
	next.Header.Set("Content-Type", "application/json")
	next.URL = &url.URL{Path: "/api/v1/plugin-bridge"}
	next.SetPathValue("id", str(in.Args, "id"))
	next.SetPathValue("rowId", str(in.Args, "row_id"))
	args := in.Args
	args["workspace_id"] = in.WorkspaceID
	query := url.Values{"workspace_id": {in.WorkspaceID}, "limit": {"100"}}
	for _, key := range []string{"q", "tag"} {
		if value := str(args, key); value != "" {
			query.Set(key, value)
		}
	}
	next.URL.RawQuery = query.Encode()
	next.Body = io.NopCloser(bytes.NewReader(jsonValue(args)))
	var handler http.HandlerFunc
	switch in.Operation {
	case "documents.list":
		handler = s.pluginListDocuments
	case "documents.get":
		handler = s.getDocument
	case "documents.create":
		handler = s.createDocument
	case "documents.update":
		handler = s.updateDocument
	case "documents.delete":
		handler = s.deleteDocument
	case "databases.list":
		handler = s.listDatabases
	case "databases.get":
		handler = s.getDatabase
	case "databases.query":
		handler = s.queryAdvancedDatabase
	case "databases.createRow":
		handler = s.createRow
	case "databases.updateRow":
		handler = s.updateRow
	case "databases.deleteRow":
		handler = s.deleteRow
	case "ai.chat":
		if !pluginContains(caps, "document:read") {
			apiError(w, 403, "워크스페이스 AI에는 문서 조회 권한도 필요합니다")
			return
		}
		s.audit(r, "PLUGIN_EXECUTE", pluginID, map[string]any{"workspace_id": in.WorkspaceID, "operation": in.Operation})
		streamCtx, cancel := context.WithCancel(next.Context())
		defer cancel()
		next = next.WithContext(streamCtx)
		// A grant revoked during an ongoing stream also stops the upstream request.
		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-streamCtx.Done():
					return
				case <-ticker.C:
					_, active, e := s.pluginGrant(r, pluginID, in.WorkspaceID)
					if e != nil || !pluginContains(active, "ai:execute") || !pluginContains(active, "document:read") {
						cancel()
						return
					}
				}
			}
		}()
		s.aiChat(w, next)
		return
	case "storage.get", "storage.set", "storage.delete":
		s.pluginStorage(w, next, pluginID, in.WorkspaceID, in.Operation, args)
		return
	case "ui.notify":
		text := str(args, "message")
		if len(text) == 0 || len(text) > 1000 {
			apiError(w, 400, "알림은 1~1,000바이트의 텍스트여야 합니다")
			return
		}
		jsonResponse(w, 200, map[string]any{"message": text})
		return
	case "ui.navigate":
		target := str(args, "path")
		parts := strings.Split(strings.Trim(target, "/"), "/")
		allowed := target == "/app" || oneOf(target, "/app/search", "/app/graph", "/app/tasks", "/app/canvases", "/app/databases")
		if len(parts) == 3 && parts[0] == "app" && validID(parts[2]) {
			switch parts[1] {
			case "documents":
				allowed = s.canDocument(next.Context(), &p, parts[2], false)
			case "databases":
				allowed = s.canDatabase(next, parts[2], false)
			case "canvases":
				allowed = s.canCanvas(next, parts[2], false)
			}
		}
		if !allowed {
			apiError(w, 403, "접근 가능한 서비스 내부 화면으로만 이동할 수 있습니다")
			return
		}
		jsonResponse(w, 200, map[string]any{"path": target})
		return
	case "file.import", "file.export":
		kind := "importers"
		if in.Operation == "file.export" {
			kind = "exporters"
		}
		var contribution *pluginContribution
		for _, item := range manifest.Contributions[kind] {
			if item.ID == str(args, "contribution_id") {
				copy := item
				contribution = &copy
				break
			}
		}
		if contribution == nil {
			apiError(w, 403, "manifest에 등록된 가져오기·내보내기만 실행할 수 있습니다")
			return
		}
		name := str(args, "name")
		ext := strings.ToLower(path.Ext(name))
		if name == "" || len(name) > 160 || strings.ContainsAny(name, "/\\\x00\r\n") || !pluginContains(contribution.Extensions, ext) || !oneOf(ext, ".md", ".txt", ".json", ".csv", ".yaml", ".yml") {
			apiError(w, 400, "확장 기능에 등록된 안전한 문서 파일명이어야 합니다")
			return
		}
		jsonResponse(w, 200, map[string]any{"name": name, "allowed": true})
		return
	}
	if handler == nil {
		apiError(w, 400, "지원하지 않는 플러그인 작업입니다")
		return
	}
	s.audit(r, "PLUGIN_EXECUTE", pluginID, map[string]any{"workspace_id": in.WorkspaceID, "operation": in.Operation})
	buffer := &pluginBufferedWriter{header: make(http.Header)}
	buffer.header.Set("X-Request-ID", w.Header().Get("X-Request-ID"))
	handler(buffer, next)
	if buffer.overflow {
		apiError(w, 413, "플러그인 응답이 4MB를 초과했습니다. 검색 조건을 좁혀주세요")
		return
	}
	if buffer.status == 0 {
		buffer.status = 200
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(buffer.status)
	_, _ = w.Write(buffer.body.Bytes())
}

func (s *Server) pluginListDocuments(w http.ResponseWriter, r *http.Request) {
	p := current(r)
	query, tag := r.URL.Query().Get("q"), r.URL.Query().Get("tag")
	if len(query) > 1000 || len(tag) > 200 {
		apiError(w, 400, "검색어 또는 태그가 너무 깁니다")
		return
	}
	// SQL-bounded summaries avoid allocating hundreds of full documents before
	// the JSON response-size guard. Full Markdown requires an explicit get.
	items, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'workspace_id',d.workspace_id,'title',left(d.title,500),'snippet',left(d.markdown,2000),'tags',`+documentSummaryTags+`,'tags_truncated',CASE WHEN jsonb_typeof(d.tags)='array' THEN jsonb_array_length(d.tags)>jsonb_array_length(`+documentSummaryTags+`) ELSE true END,'version',d.version,'status',left(d.status,32),'visibility',left(d.visibility,32),'owner_id',d.owner_id,'created_at',d.created_at,'updated_at',d.updated_at,'can_write',madi_document_allowed($1,d.id,true) AND $5) FROM documents d WHERE d.workspace_id::text=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false) AND ($3='' OR d.search_vector@@websearch_to_tsquery('simple',$3) OR d.title ILIKE '%'||$3||'%' OR d.markdown ILIKE '%'||$3||'%') AND ($4='' OR d.tags ? $4) ORDER BY d.updated_at DESC LIMIT 100`, p.ID, p.WorkspaceID, query, tag, hasIntegrationScope(p, "document:write"))
	respond(w, items, e)
}

func (s *Server) pluginRequestAllowed(userID, pluginID string) bool {
	s.pluginLimiterMu.Lock()
	defer s.pluginLimiterMu.Unlock()
	if s.pluginRequests == nil {
		s.pluginRequests = map[string]attempt{}
	}
	if !pluginIDPattern.MatchString(pluginID) {
		return false
	}
	now := time.Now()
	key := "plugin|" + userID + "|" + pluginID
	if len(s.pluginRequests) >= 10000 {
		for name, value := range s.pluginRequests {
			if now.After(value.until) {
				delete(s.pluginRequests, name)
			}
		}
	}
	value := s.pluginRequests[key]
	if now.After(value.until) {
		value = attempt{until: now.Add(time.Minute)}
	}
	if value.count >= 120 || len(s.pluginRequests) >= 10000 && value.count == 0 {
		return false
	}
	value.count++
	s.pluginRequests[key] = value
	return true
}

func (s *Server) pluginStorage(w http.ResponseWriter, r *http.Request, pluginID, wid, operation string, args map[string]any) {
	key := str(args, "key")
	if key == "" || len(key) > 120 || strings.ContainsAny(key, "\x00\r\n") {
		apiError(w, 400, "저장 키는 1~120바이트여야 합니다")
		return
	}
	uid := current(r).ID
	if operation == "storage.get" {
		var raw []byte
		e := s.DB.QueryRow(r.Context(), "SELECT value FROM plugin_storage WHERE plugin_id=$1 AND workspace_id=$2 AND user_id=$3 AND key=$4", pluginID, wid, uid, key).Scan(&raw)
		if errors.Is(e, pgx.ErrNoRows) {
			jsonResponse(w, 200, map[string]any{"value": nil})
			return
		}
		if e != nil {
			respond(w, nil, e)
			return
		}
		jsonResponse(w, 200, map[string]any{"value": json.RawMessage(raw)})
		return
	}
	if operation == "storage.delete" {
		_, e := s.DB.Exec(r.Context(), "DELETE FROM plugin_storage WHERE plugin_id=$1 AND workspace_id=$2 AND user_id=$3 AND key=$4", pluginID, wid, uid, key)
		respond(w, map[string]any{"ok": true}, e)
		return
	}
	raw := jsonValue(args["value"])
	if len(raw) > 65536 {
		apiError(w, 400, "개인 플러그인 값은 64KB 이하여야 합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	_, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", pluginID+":"+wid+":"+uid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	var count int
	var existing bool
	e = tx.QueryRow(r.Context(), "SELECT count(*),coalesce(bool_or(key=$4),false) FROM plugin_storage WHERE plugin_id=$1 AND workspace_id=$2 AND user_id=$3", pluginID, wid, uid, key).Scan(&count, &existing)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 100 && !existing {
		apiError(w, 400, "개인 플러그인 저장 키는 100개까지 지원합니다")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO plugin_storage(plugin_id,workspace_id,user_id,key,value) VALUES($1,$2,$3,$4,$5) ON CONFLICT(plugin_id,workspace_id,user_id,key) DO UPDATE SET value=excluded.value,updated_at=now()`, pluginID, wid, uid, key, raw)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"ok": true}, e)
}
