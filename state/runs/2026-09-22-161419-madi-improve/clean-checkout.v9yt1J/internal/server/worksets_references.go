package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func validateWorksetContext(item worksetItem) error {
	c := item.Context
	if c == nil || len(jsonValue(c)) > 32768 {
		return errWorksetContext
	}
	allowed := map[string]bool{}
	switch item.Kind {
	case "document":
		for _, k := range []string{"version", "line", "selection_start", "selection_end", "scroll_y", "mode", "range"} {
			allowed[k] = true
		}
	case "task":
		for _, k := range []string{"version", "task_id", "line"} {
			allowed[k] = true
		}
	case "database":
		allowed["view_id"] = true
		allowed["view"] = true
	default:
		return errWorksetContext
	}
	for k, v := range c {
		if !allowed[k] {
			return errWorksetContext
		}
		switch k {
		case "range":
			value, ok := v.(map[string]any)
			if !ok || len(value) != 4 || number(c, "version", 0) < 1 {
				return errWorksetContext
			}
			for _, key := range []string{"start", "end"} {
				items, ok := value[key].([]any)
				if !ok || len(items) > 32 {
					return errWorksetContext
				}
				for _, item := range items {
					n, ok := item.(float64)
					if !ok || n < 0 || n > 100000 || math.Trunc(n) != n {
						return errWorksetContext
					}
				}
			}
			for _, key := range []string{"startOffset", "endOffset"} {
				n, ok := value[key].(float64)
				if !ok || n < 0 || n > 10000000 || math.Trunc(n) != n {
					return errWorksetContext
				}
			}
		case "mode":
			if text, ok := v.(string); !ok || !oneOf(text, "read", "edit") {
				return errWorksetContext
			}
		case "task_id", "view_id":
			if text, ok := v.(string); !ok || !validID(text) {
				return errWorksetContext
			}
		case "view":
			if _, ok := v.(map[string]any); !ok {
				return errWorksetContext
			}
		default:
			n, ok := v.(float64)
			limit := float64(10000000)
			if k == "version" {
				limit = 2147483647
			}
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || n < 0 || n > limit {
				return errWorksetContext
			}
			if k == "version" && (n < 1 || n > 2147483647) {
				return errWorksetContext
			}
		}
	}
	if item.Kind == "task" && !validID(str(c, "task_id")) {
		return errWorksetContext
	}
	if item.Kind == "document" {
		_, start := c["selection_start"]
		_, end := c["selection_end"]
		if start != end || start && (number(c, "selection_end", 0) < number(c, "selection_start", 0) || number(c, "version", 0) < 1) {
			return errWorksetContext
		}
		if (c["line"] != nil || c["scroll_y"] != nil) && number(c, "version", 0) < 1 {
			return errWorksetContext
		}
	}
	return nil
}
func (s *Server) resolveWorksetItem(r *http.Request, tx pgx.Tx, wid string, item worksetItem) (map[string]any, error) {
	out := map[string]any{"kind": item.Kind, "resource_id": item.ResourceID, "available": false}
	var raw []byte
	if item.Kind == "database" {
		e := tx.QueryRow(r.Context(), `SELECT jsonb_build_object('title',left(d.name,500),'updated_at',d.created_at,'can_write',madi_space_allowed($2,d.space_id,true)) FROM databases d WHERE d.id=$1 AND d.workspace_id=$3 AND madi_space_allowed($2,d.space_id,false)`, item.ResourceID, current(r).ID, wid).Scan(&raw)
		if errors.Is(e, pgx.ErrNoRows) {
			return out, nil
		}
		if e != nil {
			return nil, e
		}
		var data map[string]any
		if e = json.Unmarshal(raw, &data); e != nil {
			return nil, e
		}
		for k, v := range data {
			out[k] = v
		}
		out["available"] = true
		out["url"] = "/app/databases/" + item.ResourceID
		if e = s.validateWorksetDatabaseContext(r, tx, wid, item); e != nil {
			out["context_changed"] = true
			out["context_notice"] = "원래 보기 또는 속성에 접근할 수 없어 기본 표로 엽니다"
			return out, nil
		}
		query := url.Values{}
		config, hasConfig := item.Context["view"].(map[string]any)
		if saved := str(item.Context, "view_id"); saved != "" {
			query.Set("saved_view", saved)
			if !hasConfig {
				if e = tx.QueryRow(r.Context(), "SELECT data FROM database_views WHERE id=$1 AND database_id=$2 AND(owner_id=$3 OR visibility='workspace')", saved, item.ResourceID, current(r).ID).Scan(&raw); e != nil {
					return nil, e
				}
				if e = json.Unmarshal(raw, &config); e != nil {
					return nil, e
				}
			}
		}
		if config != nil {
			for k, v := range config {
				switch k {
				case "view":
					query.Set(k, fmt.Sprint(v))
				case "board_property_id":
					query.Set("board_property", fmt.Sprint(v))
				case "date_property_id":
					query.Set("date_property", fmt.Sprint(v))
				default:
					query.Set(k, string(jsonValue(v)))
				}
			}
		}
		if len(query) > 0 {
			out["url"] = out["url"].(string) + "?" + query.Encode()
		}
		return out, nil
	}
	e := tx.QueryRow(r.Context(), `SELECT jsonb_build_object('title',left(d.title,500),'version',d.version,'updated_at',d.updated_at,'visibility',d.visibility,'status',d.status,'can_write',madi_document_allowed($2,d.id,true),'tags',`+documentSummaryTags+`,'stale',d.updated_at<now()-interval '90 days') FROM documents d WHERE d.id=$1 AND d.workspace_id=$3 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false)`, item.ResourceID, current(r).ID, wid).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return nil, e
	}
	var data map[string]any
	if e = json.Unmarshal(raw, &data); e != nil {
		return nil, e
	}
	for k, v := range data {
		out[k] = v
	}
	out["available"] = true
	if item.Kind == "task" {
		var md string
		if e = tx.QueryRow(r.Context(), `SELECT markdown FROM documents WHERE id=$1 AND madi_document_allowed($2,id,false) AND octet_length(markdown)<=1048576`, item.ResourceID, current(r).ID).Scan(&md); errors.Is(e, pgx.ErrNoRows) {
			return map[string]any{"kind": item.Kind, "resource_id": item.ResourceID, "available": false}, nil
		} else if e != nil {
			return nil, e
		}
		found := false
		for _, task := range indexMarkdown(md).Tasks {
			if task.ID == str(item.Context, "task_id") && !task.Ambiguous {
				found = true
				out["task_text"] = truncateAIRunes(task.Text, 160)
				out["done"] = task.Done
				out["url"] = "/app/tasks?document_id=" + item.ResourceID + "&task=" + task.ID
				break
			}
		}
		if !found {
			return map[string]any{"kind": item.Kind, "resource_id": item.ResourceID, "available": false}, nil
		}
		return out, nil
	}
	query := url.Values{"mode": []string{"read"}}
	if str(item.Context, "mode") == "edit" {
		query.Set("mode", "edit")
	}
	if number(item.Context, "version", 0) > 0 && number(item.Context, "version", 0) != number(out, "version", 0) {
		out["context_changed"] = true
		out["context_notice"] = "문서 버전이 바뀌어 오래된 줄·선택·스크롤을 복원하지 않습니다"
	} else {
		if line := number(item.Context, "line", 0); line > 0 {
			query.Set("line", strconv.Itoa(line))
		}
		out["restore_context"] = item.Context
	}
	out["url"] = "/app/documents/" + item.ResourceID + "?" + query.Encode()
	return out, nil
}
func (s *Server) validateWorksetDatabaseContext(r *http.Request, tx pgx.Tx, wid string, item worksetItem) error {
	var properties []map[string]any
	var raw []byte
	if e := tx.QueryRow(r.Context(), `SELECT d.properties FROM databases d WHERE d.id=$1 AND d.workspace_id=$3 AND madi_space_allowed($2,d.space_id,false) FOR SHARE`, item.ResourceID, current(r).ID, wid).Scan(&raw); e != nil {
		return errWorksetContext
	}
	if json.Unmarshal(raw, &properties) != nil {
		return errWorksetContext
	}
	if viewID := str(item.Context, "view_id"); viewID != "" {
		if e := tx.QueryRow(r.Context(), `SELECT data FROM database_views WHERE id=$1 AND database_id=$2 AND(owner_id=$3 OR visibility='workspace') FOR SHARE`, viewID, item.ResourceID, current(r).ID).Scan(&raw); e != nil {
			return errWorksetContext
		}
		var data map[string]any
		if json.Unmarshal(raw, &data) != nil || validateDatabaseViewData(data, properties) != nil {
			return errWorksetContext
		}
	}
	if config, ok := item.Context["view"].(map[string]any); ok {
		return validateDatabaseViewData(config, properties)
	}
	return nil
}
