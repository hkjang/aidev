package server

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) registerTaskCalendar() {
	s.handle("GET /api/v1/tasks/calendar", s.taskCalendar)
	s.handle("POST /api/v1/tasks/calendar/events", s.saveCalendarEvent)
	s.handle("PUT /api/v1/tasks/calendar/events/{id}", s.saveCalendarEvent)
	s.handle("DELETE /api/v1/tasks/calendar/events/{id}", s.deleteCalendarEvent)
}
func (s *Server) taskCalendar(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	did := r.URL.Query().Get("document_id")
	if did != "" && !validID(did) {
		apiError(w, 400, "문서 ID를 확인하세요")
		return
	}
	month, e := time.Parse("2006-01", r.URL.Query().Get("month"))
	if e != nil {
		apiError(w, 400, "달력 월은 YYYY-MM 형식이어야 합니다")
		return
	}
	start, end := month.Format(time.DateOnly), month.AddDate(0, 1, 0).Format(time.DateOnly)
	events, e := s.rows(r.Context(), `SELECT to_jsonb(c)||jsonb_build_object('key','event:'||c.id::text,'start',c.start_date,'end',c.end_date,'can_write',c.owner_id=$1) FROM calendar_events c WHERE c.workspace_id=$2 AND c.start_date<$4::date AND c.end_date>=$3::date AND ($5='' OR c.document_id=NULLIF($5,'')::uuid) AND (c.visibility='workspace' OR c.owner_id=$1) AND (c.document_id IS NULL OR EXISTS(SELECT 1 FROM documents d WHERE d.id=c.document_id AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false))) ORDER BY c.start_date,c.id LIMIT 2001`, current(r).ID, wid, start, end, did)
	if e != nil {
		respond(w, nil, e)
		return
	}
	truncated := len(events) > 2000
	if truncated {
		events = events[:2000]
	}
	board, e := s.taskRows(r)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if boolean(board, "truncated") {
		truncated = true
	}
	if tasks, ok := board["items"].([]map[string]any); ok {
		for _, task := range tasks {
			day := str(task, "due_date")
			if day >= start && day < end {
				events = append(events, map[string]any{"key": "task:" + str(task, "document_id") + ":" + str(task, "task_id") + ":" + strings.TrimSpace(string(jsonValue(task["source_start"]))), "kind": "task", "title": task["text"], "start": day, "end": day, "document_id": task["document_id"], "task": task, "done": task["done"], "url": "/app/tasks?task=" + str(task, "task_id")})
			}
		}
	}
	dbRows := []map[string]any{}
	// Calendar aggregation must not broaden a document-only integration grant.
	if did == "" && hasIntegrationScope(current(r), "database:read") {
		dbRows, e = s.rows(r.Context(), `SELECT jsonb_build_object('key','database:'||b.id::text||':'||r.id::text||':'||(p->>'id'),'kind','database','title',b.name||' · '||coalesce(nullif(r.values->>(b.properties->0->>'id'),''),r.id::text),'start',left(r.values->>(p->>'id'),10),'end',left(r.values->>(p->>'id'),10),'database_id',b.id,'row_id',r.id,'property',p->>'name','url','/app/databases/'||b.id::text||'?view=calendar&row='||r.id::text) FROM databases b JOIN database_rows r ON r.database_id=b.id CROSS JOIN LATERAL jsonb_array_elements(b.properties) p WHERE b.workspace_id=$2 AND madi_space_allowed($1,b.space_id,false) AND p->>'type'='date' AND left(r.values->>(p->>'id'),10)>=$3 AND left(r.values->>(p->>'id'),10)<$4 ORDER BY r.created_at DESC LIMIT 2001`, current(r).ID, wid, start, end)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if len(dbRows) > 2000 {
		truncated = true
		dbRows = dbRows[:2000]
	}
	events = append(events, dbRows...)
	var docs []map[string]any
	if did != "" {
		// A document filter expresses an existing relation, not a request to
		// associate independent meetings or database dates with this source.
		docs, e = s.rows(r.Context(), `SELECT jsonb_build_object('id',d.id,'title',d.title,'markdown',d.markdown,'kind',coalesce(k.kind,'page')) FROM documents d LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.id=$3::uuid AND d.workspace_id=$2::uuid AND d.deleted_at IS NULL AND octet_length(d.markdown)<=16777216 AND `+docACL, current(r).ID, wid, did)
	} else {
		docs, e = s.knowledgeDocuments(r)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if len(docs) > 2000 {
		truncated = true
		docs = docs[:2000]
	}
	if len(docs) > 0 && number(docs[0], "total_visible", 0) > len(docs) {
		truncated = true
	}
	for _, d := range docs {
		fm, err := parseFrontMatter(str(d, "markdown"))
		if err != nil {
			continue
		}
		kind := str(d, "kind")
		day := str(fm, "date")
		if day == "" {
			if v, ok := fm["date"].(time.Time); ok {
				day = v.Format(time.DateOnly)
			}
		}
		if day == "" && kind == "daily" {
			day = str(d, "title")
		}
		if day == "" && len(str(d, "title")) == 10 {
			if _, err := time.Parse(time.DateOnly, str(d, "title")); err == nil {
				day = str(d, "title")
				kind = "daily"
			}
		}
		if len(day) >= 10 {
			day = day[:10]
		}
		if _, err := time.Parse(time.DateOnly, day); err == nil && day >= start && day < end {
			events = append(events, map[string]any{"key": "document:" + str(d, "id"), "kind": kind, "title": d["title"], "start": day, "end": day, "document_id": d["id"], "url": "/app/documents/" + str(d, "id")})
		}
	}
	jsonResponse(w, 200, map[string]any{"events": events, "month": month.Format("2006-01"), "truncated": truncated})
}
func (s *Server) saveCalendarEvent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Workspace  string `json:"workspace_id"`
		Title      string `json:"title"`
		Kind       string `json:"kind"`
		Document   string `json:"document_id"`
		Start      string `json:"start_date"`
		End        string `json:"end_date"`
		Visibility string `json:"visibility"`
		Version    int    `json:"version"`
	}
	if decode(r, &in) != nil || !oneOf(in.Kind, "meeting", "milestone") || !oneOf(in.Visibility, "private", "workspace") || strings.TrimSpace(in.Title) == "" || len(in.Title) > 500 {
		apiError(w, 400, "일정 제목·종류·공유 범위를 확인하세요")
		return
	}
	if !s.canWorkspace(r.Context(), current(r), in.Workspace, true) {
		apiError(w, 403, "일정 작성 권한이 없습니다")
		return
	}
	first, e1 := time.Parse(time.DateOnly, in.Start)
	last, e2 := time.Parse(time.DateOnly, in.End)
	if e1 != nil || e2 != nil || last.Before(first) || last.Sub(first) > 366*24*time.Hour {
		apiError(w, 400, "일정 날짜는 YYYY-MM-DD, 기간은 366일 이내여야 합니다")
		return
	}
	if in.Document != "" && !validID(in.Document) {
		apiError(w, 400, "연결할 문서를 확인하세요")
		return
	}
	id := r.PathValue("id")
	creating := id == ""
	if creating {
		id = newID()
	} else if !validID(id) {
		apiError(w, 404, "일정을 찾을 수 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if in.Document != "" {
		var allowed bool
		e = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false))", in.Document, in.Workspace, current(r).ID).Scan(&allowed)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if !allowed {
			apiError(w, 400, "같은 워크스페이스에서 접근 가능한 문서를 연결하세요")
			return
		}
	}
	if creating {
		_, e = tx.Exec(r.Context(), "INSERT INTO calendar_events(id,workspace_id,owner_id,document_id,title,kind,start_date,end_date,visibility) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7::date,$8::date,$9)", id, in.Workspace, current(r).ID, in.Document, strings.TrimSpace(in.Title), in.Kind, in.Start, in.End, in.Visibility)
	} else {
		tag, err := tx.Exec(r.Context(), "UPDATE calendar_events SET title=$2,document_id=NULLIF($3,'')::uuid,kind=$4,start_date=$5::date,end_date=$6::date,visibility=$7,version=version+1,updated_at=now() WHERE id=$1 AND workspace_id=$8 AND owner_id=$9 AND version=$10", id, strings.TrimSpace(in.Title), in.Document, in.Kind, in.Start, in.End, in.Visibility, in.Workspace, current(r).ID, in.Version)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			apiError(w, 409, "일정이 변경되었거나 소유자 권한이 없습니다")
			return
		}
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "CALENDAR_EVENT_SAVE", id, map[string]any{"kind": in.Kind, "workspace_id": in.Workspace})
	v, e := s.one(r.Context(), "SELECT to_jsonb(c) FROM calendar_events c WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) deleteCalendarEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		apiError(w, 404, "일정을 찾을 수 없습니다")
		return
	}
	v, e := s.one(r.Context(), "SELECT to_jsonb(c) FROM calendar_events c WHERE id=$1 AND owner_id=$2", id, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if !s.canWorkspace(r.Context(), current(r), str(v, "workspace_id"), true) {
		apiError(w, 403, "일정 삭제 권한이 없습니다")
		return
	}
	_, e = s.DB.Exec(r.Context(), "DELETE FROM calendar_events WHERE id=$1 AND owner_id=$2", id, current(r).ID)
	if e == nil {
		s.audit(r, "CALENDAR_EVENT_DELETE", id, nil)
	}
	respond(w, map[string]bool{"ok": true}, e)
}
