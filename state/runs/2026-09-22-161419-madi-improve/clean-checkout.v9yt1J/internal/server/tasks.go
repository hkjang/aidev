package server

import (
	"context"
	_ "embed"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"regexp"
	"strings"
	"time"
)

//go:embed tasks.sql
var tasksSchema string

func (s *Server) migrateTasks(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, tasksSchema)
	return e
}
func (s *Server) registerTasks() {
	s.handle("GET /api/v1/tasks/board", s.taskBoard)
	s.handle("PUT /api/v1/tasks/details", s.updateTaskDetails)
}

type taskMutationKey struct{}
type taskMutation struct{ ID, Assignee, Due, Status, Priority string }

var errTaskAssignee = errors.New("할 일 담당자는 현재 문서에 접근할 수 있어야 합니다")
var errTaskReviewDisabled = errors.New("검토 프로세스가 비활성화되었습니다. 현재 할 일을 다시 불러오세요")
var taskDue = regexp.MustCompile(`(?:^|\s)(\d{4}-\d{2}-\d{2})(?:$|\s)`)
var taskUser = regexp.MustCompile(`@\[([^\]]+)\]\(user:([0-9a-fA-F-]{36})\)|@([\p{L}\p{N}_.-]+)`)

func (s *Server) taskRows(r *http.Request) (map[string]any, error) {
	wid := r.URL.Query().Get("workspace_id")
	if current(r).WorkspaceID != "" {
		if wid != "" && wid != current(r).WorkspaceID {
			return nil, errors.New("키의 워크스페이스 범위를 벗어났습니다")
		}
		wid = current(r).WorkspaceID
	}
	if wid != "" && !s.canWorkspace(r.Context(), current(r), wid, false) {
		return nil, errors.New("워크스페이스 접근 권한이 없습니다")
	}
	did := r.URL.Query().Get("document_id")
	if did != "" && !validID(did) {
		return nil, errors.New("문서 ID를 확인하세요")
	}
	query, args := taskDocumentQuery(current(r).ID, wid, did)
	docs, e := s.rows(r.Context(), query, args...)
	if e != nil {
		return nil, e
	}
	total := 0
	if len(docs) > 0 {
		total = number(docs[0], "total_visible", 0)
		// A zero-payload result still carries safe visible-only statistics: for
		// example the newest source alone exceeds the 16MiB body budget.
		if str(docs[0], "id") == "" {
			docs = docs[:0]
		}
	}
	people, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',u.id,'name',u.name) FROM users u JOIN workspace_members m ON m.user_id=u.id WHERE ($1='' OR m.workspace_id::text=$1) AND NOT u.disabled AND EXISTS(SELECT 1 FROM workspace_members own WHERE own.workspace_id=m.workspace_id AND own.user_id=$2) GROUP BY u.id ORDER BY u.name LIMIT 10000`, wid, current(r).ID)
	if e != nil {
		return nil, e
	}
	names := map[string]string{}
	counts := map[string]int{}
	for _, p := range people {
		counts[str(p, "name")]++
		names[str(p, "name")] = str(p, "id")
	}
	out := []map[string]any{}
	for _, d := range docs {
		details := map[string]map[string]any{}
		if raw, ok := d["details"].([]any); ok {
			for _, v := range raw {
				m, _ := v.(map[string]any)
				details[str(m, "task_id")] = m
			}
		}
		for _, t := range indexMarkdown(str(d, "markdown")).Tasks {
			row := map[string]any{"document_id": d["id"], "title": d["title"], "text": t.Text, "line": t.Line, "version": d["version"], "task_id": t.ID, "ambiguous": t.Ambiguous, "done": t.Done, "status": "todo", "priority": "normal", "assignee_id": "", "assignee_name": "", "due_date": "", "can_write": boolean(d, "can_write") && hasIntegrationScope(current(r), "document:write"), "owner_id": d["owner_id"], "space_id": d["space_id"]}
			row["source_start"] = t.Start
			if m := details[t.ID]; m != nil && !t.Ambiguous {
				for _, key := range []string{"status", "priority", "assignee_id", "assignee_name", "assignee_available", "due_date"} {
					row[key] = m[key]
				}
			} else {
				if m := taskDue.FindStringSubmatch(t.Text); len(m) > 1 {
					if _, err := time.Parse(time.DateOnly, m[1]); err == nil {
						row["due_date"] = m[1]
					}
				}
				if m := taskUser.FindStringSubmatch(t.Text); len(m) > 0 {
					uid := m[2]
					if uid == "" && counts[m[3]] == 1 {
						uid = names[m[3]]
					}
					if uid != "" {
						for _, p := range people {
							if p["id"] == uid {
								row["assignee_id"] = uid
								row["assignee_name"] = p["name"]
								break
							}
						}
					}
				}
			}
			if t.Done {
				row["status"] = "done"
			} else if str(row, "status") == "done" {
				row["status"] = "todo"
			}
			if r.URL.Query().Get("task") != "" && r.URL.Query().Get("task") != t.ID {
				continue
			}
			out = append(out, row)
			if len(out) >= 10000 {
				break
			}
		}
		if len(out) >= 10000 {
			break
		}
	}
	return map[string]any{"items": out, "documents_scanned": len(docs), "total_documents": total,
		"total_documents_exact": total < 2001, "total_documents_is_lower_bound": total >= 2001,
		"truncated": total > len(docs) || len(out) >= 10000,
		"notice":    "현재 열람 가능한 최신 문서 최대 2,000개·본문 16MiB 범위입니다. total_documents_exact=false이면 total_documents는 전체 총수가 아니라 최소 문서 수입니다.",
		"limits":    map[string]any{"documents": 2000, "visible_count": 2001, "markdown_bytes": 16 << 20, "tasks": 10000}}, nil
}
func (s *Server) taskBoard(w http.ResponseWriter, r *http.Request) {
	if id := r.URL.Query().Get("document_id"); id != "" && !validID(id) {
		apiError(w, 400, "문서 ID를 확인하세요")
		return
	}
	wid := r.URL.Query().Get("workspace_id")
	if wid == "" {
		wid = current(r).WorkspaceID
	}
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	v, e := s.taskRows(r)
	respond(w, v, e)
}
func (s *Server) updateTaskDetails(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DocumentID string `json:"document_id"`
		Line       int    `json:"line"`
		Start      *int   `json:"source_start"`
		Version    int    `json:"version"`
		TaskID     string `json:"task_id"`
		Assignee   string `json:"assignee_id"`
		Due        string `json:"due_date"`
		Status     string `json:"status"`
		Priority   string `json:"priority"`
		Separate   bool   `json:"separate"`
	}
	if decode(r, &in) != nil || !oneOf(in.Status, "backlog", "todo", "doing", "review", "done") || !oneOf(in.Priority, "low", "normal", "high", "urgent") {
		apiError(w, 400, "할 일 상태와 우선순위를 확인하세요")
		return
	}
	if !s.canDocument(r.Context(), current(r), in.DocumentID, true) {
		apiError(w, 403, "할 일 수정 권한이 없습니다")
		return
	}
	if in.Assignee != "" && !validID(in.Assignee) {
		apiError(w, 400, "할 일 담당자를 확인하세요")
		return
	}
	if in.Due != "" {
		if _, e := time.Parse(time.DateOnly, in.Due); e != nil {
			apiError(w, 400, "마감일은 YYYY-MM-DD 형식이어야 합니다")
			return
		}
	}
	if in.Status == "review" {
		cfg, e := s.settings(r.Context())
		if e != nil {
			respond(w, nil, e)
			return
		}
		if !boolean(cfg, "approval_enabled") {
			apiError(w, 400, "관리자가 검토 프로세스를 켠 경우에만 검토 대기를 사용할 수 있습니다")
			return
		}
	}
	d, e := s.document(r, in.DocumentID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if in.Version != number(d, "version", 0) {
		apiError(w, 409, "문서가 변경되었습니다. 현재 할 일을 다시 불러오세요")
		return
	}
	var target *markdownTask
	for _, task := range indexMarkdown(str(d, "markdown")).Tasks {
		if task.Line == in.Line && (in.Start == nil || task.Start == *in.Start) {
			if target != nil {
				apiError(w, 409, "같은 줄의 할 일을 구분하려면 source_start가 필요합니다")
				return
			}
			copy := task
			target = &copy
		}
	}
	if target == nil || target.ID != in.TaskID || (target.Ambiguous && !in.Separate) {
		apiError(w, 409, "할 일 위치나 고유 참조가 변경되었습니다. 중복 참조는 별도 작업으로 분리하세요")
		return
	}
	id := target.ID
	if id == "" || in.Separate {
		id = newID()
	}
	ctx := context.WithValue(r.Context(), taskMutationKey{}, taskMutation{ID: id, Assignee: in.Assignee, Due: in.Due, Status: in.Status, Priority: in.Priority})
	if target.HTML != nil {
		md := updateHTMLTask(str(d, "markdown"), *target, id, in.Status == "done", in.Separate)
		s.saveDocument(w, r.WithContext(ctx), in.DocumentID, map[string]any{"version": in.Version, "markdown": md})
		return
	}
	lines := strings.Split(str(d, "markdown"), "\n")
	if target.Line < 0 || target.Line >= len(lines) || !taskPattern.MatchString(lines[target.Line]) {
		apiError(w, 409, "현재 형식의 체크리스트는 문서에서 직접 편집하세요")
		return
	}
	if target.ID == "" || in.Separate {
		if in.Separate {
			for n := target.Line; n <= target.EndLine && n < len(lines); n++ {
				if n > target.Line && taskPattern.MatchString(lines[n]) {
					break
				}
				lines[n] = strings.TrimRight(rewriteVaultContent(lines[n], func(value string) string { return taskReference.ReplaceAllString(value, "") }), " \t\r")
			}
		}
		lines[target.Line] += " [작업](/app/tasks?task=" + id + ")"
	}
	marker := strings.Index(lines[target.Line], "[")
	mark := " "
	if in.Status == "done" {
		mark = "x"
	}
	lines[target.Line] = lines[target.Line][:marker+1] + mark + lines[target.Line][marker+2:]
	s.saveDocument(w, r.WithContext(ctx), in.DocumentID, map[string]any{"version": in.Version, "markdown": strings.Join(lines, "\n")})
}
func (s *Server) saveTaskDetailsTx(r *http.Request, tx pgx.Tx, id, wid string) error {
	in, ok := r.Context().Value(taskMutationKey{}).(taskMutation)
	if !ok {
		return nil
	}
	// saveDocument already holds the shared settings lock; check the current
	// flag inside that transaction, not only the preflight request snapshot.
	if in.Status == "review" {
		var enabled bool
		if e := tx.QueryRow(r.Context(), "SELECT coalesce((data->>'approval_enabled')::boolean,false) FROM settings WHERE id=1").Scan(&enabled); e != nil {
			return e
		}
		if !enabled {
			return errTaskReviewDisabled
		}
	}
	if in.Assignee != "" {
		var allowed bool
		if e := tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND NOT disabled AND madi_document_allowed(id,$2,false))", in.Assignee, id).Scan(&allowed); e != nil {
			return e
		}
		if !allowed {
			return errTaskAssignee
		}
	}
	var previous string
	e := tx.QueryRow(r.Context(), "SELECT coalesce(assignee_id::text,'') FROM task_details WHERE document_id=$1 AND task_id=$2", id, in.ID).Scan(&previous)
	if e != nil && e != pgx.ErrNoRows {
		return e
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO task_details(document_id,task_id,assignee_id,due_date,status,priority,updated_by) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::date,$5,$6,$7) ON CONFLICT(document_id,task_id) DO UPDATE SET assignee_id=excluded.assignee_id,due_date=excluded.due_date,status=excluded.status,priority=excluded.priority,updated_by=excluded.updated_by,updated_at=now()`, id, in.ID, in.Assignee, in.Due, in.Status, in.Priority, current(r).ID)
	if e == nil && in.Assignee != "" && in.Assignee != previous {
		_, e = tx.Exec(r.Context(), "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,'담당할 할 일이 지정되었습니다.',$3)", newID(), in.Assignee, id)
	}
	return e
}
