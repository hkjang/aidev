package server

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
)

func (s *Server) enqueueTaskCompletions(ctx context.Context, tx pgx.Tx, wid, id, title, before, after string) error {
	type fallbackKey struct {
		Line int
		Text string
	}
	oldTasks := map[fallbackKey]markdownTask{}
	oldCount := map[fallbackKey]int{}
	nextCount := map[fallbackKey]int{}
	nextTasks := indexMarkdown(after).Tasks
	for _, task := range nextTasks {
		nextCount[fallbackKey{task.Line, task.Text}]++
	}
	byID := map[string]markdownTask{}
	for _, task := range indexMarkdown(before).Tasks {
		key := fallbackKey{task.Line, task.Text}
		oldTasks[key] = task
		oldCount[key]++
		if task.ID != "" && !task.Ambiguous {
			byID[task.ID] = task
		}
	}
	count := 0
	for _, next := range nextTasks {
		key := fallbackKey{next.Line, next.Text}
		old, found := oldTasks[key]
		found = found && oldCount[key] == 1 && nextCount[key] == 1
		stable := false
		if next.ID != "" {
			if previous, ok := byID[next.ID]; ok {
				old = previous
				found = true
				stable = true
			}
		}
		if !found || old.Done || !next.Done || next.Ambiguous || old.Ambiguous || (!stable && old.Text != next.Text) {
			continue
		}
		count++
		if count > 1000 {
			return errors.New("한 번에 최대 1000개의 할 일을 완료할 수 있습니다")
		}
		if e := s.enqueueEvent(ctx, tx, Event{Type: "task.completed", WorkspaceID: wid, ResourceID: id, After: map[string]any{"document_id": id, "title": title, "line": next.Line, "task_id": next.ID, "done": true}}); e != nil {
			return e
		}
	}
	return nil
}
func (s *Server) enqueueDatabaseEvent(r *http.Request, tx pgx.Tx, kind, id, rowID string, before map[string]any) error {
	var wid, space, name string
	if e := tx.QueryRow(r.Context(), "SELECT workspace_id::text,coalesce(space_id::text,''),name FROM databases WHERE id=$1", id).Scan(&wid, &space, &name); e != nil {
		return e
	}
	return s.enqueueEvent(r.Context(), tx, Event{Type: kind, WorkspaceID: wid, ResourceType: "database", ResourceID: id, Before: before, After: map[string]any{"database_id": id, "row_id": rowID, "title": name, "space_id": space}})
}
