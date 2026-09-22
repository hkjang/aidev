package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

const documentQueryDefinitionBytes = 5120
const documentQueryMaxDefinitions = 16
const documentQueryMaxCandidates = 2000
const documentQueryScanBytes = 8 << 20
const documentQueryMaxBodyBytes = 256 << 10
const documentQueryResultBytes = 1 << 20

type documentQueryColumn struct {
	Field string `json:"field"`
	Label string `json:"label,omitempty"`
}
type documentQueryFilter struct {
	Field string          `json:"field"`
	Op    string          `json:"op"`
	Value json.RawMessage `json:"value,omitempty"`
}
type documentQueryOrder struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
type documentQueryParameter struct {
	Type     string          `json:"type"`
	Label    string          `json:"label,omitempty"`
	Default  json.RawMessage `json:"default,omitempty"`
	Required bool            `json:"required,omitempty"`
}
type documentQueryDefinition struct {
	Version     int                               `json:"version"`
	Source      string                            `json:"source"`
	Columns     []documentQueryColumn             `json:"columns"`
	Filters     []documentQueryFilter             `json:"filters,omitempty"`
	Order       []documentQueryOrder              `json:"order_by,omitempty"`
	Properties  map[string]string                 `json:"properties,omitempty"`
	Parameters  map[string]documentQueryParameter `json:"parameters,omitempty"`
	SpaceID     string                            `json:"space_id,omitempty"`
	DocumentIDs []string                          `json:"document_ids,omitempty"`
	Direction   string                            `json:"direction,omitempty"`
	Depth       int                               `json:"depth,omitempty"`
	Limit       int                               `json:"limit"`
}
type documentQueryFence struct {
	Hash       string                   `json:"hash"`
	Source     string                   `json:"source"`
	Line       int                      `json:"line"`
	Definition *documentQueryDefinition `json:"definition,omitempty"`
	Error      string                   `json:"error,omitempty"`
}

var documentQueryName = regexp.MustCompile(`^[\p{L}\p{N}_][\p{L}\p{N}_-]{0,63}$`)
var documentQueryFields = map[string]map[string]string{
	"documents": {"id": "string", "title": "string", "status": "string", "tags": "strings", "owner_id": "string", "owner_name": "string", "created_at": "date", "updated_at": "date", "classification": "string", "kind": "string", "version": "number"},
	"tasks":     {"document_id": "string", "document_title": "string", "text": "string", "done": "boolean", "status": "string", "priority": "string", "assignee_id": "string", "assignee_name": "string", "due_date": "date", "version": "number", "line": "number"},
	"relations": {"source_id": "string", "source_title": "string", "target_id": "string", "target_title": "string", "relation_type": "string", "depth": "number"},
}

func documentQueryFieldType(q documentQueryDefinition, field string) string {
	if strings.HasPrefix(field, "property.") && q.Source == "documents" {
		return q.Properties[strings.TrimPrefix(field, "property.")]
	}
	return documentQueryFields[q.Source][field]
}
func documentQueryScalar(value any, kind string) bool {
	if value == nil {
		return false
	}
	switch kind {
	case "string":
		v, ok := value.(string)
		return ok && len(v) <= 500
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		v, ok := value.(json.Number)
		if !ok {
			return false
		}
		n, e := v.Float64()
		return e == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Abs(n) <= 9007199254740991
	case "date":
		v, ok := value.(string)
		if !ok {
			return false
		}
		_, e := documentQueryDate(v)
		return e == nil
	}
	return false
}
func documentQueryDate(value string) (time.Time, error) {
	if len(value) == 10 {
		return time.Parse("2006-01-02", value)
	}
	return time.Parse(time.RFC3339, value)
}
func documentQueryJSONValue(raw json.RawMessage) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// Reject ambiguous duplicate fields and deeply nested literal data before typed
// decoding. This grammar deliberately has no expressions, recursive operators,
// functions, arbitrary JSONPath, SQL fragments or executable values.
func documentQueryCheckJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 12 {
			return fmt.Errorf("조회 정의의 중첩 한도를 초과했습니다")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				key, e := decoder.Token()
				if e != nil {
					return e
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("중복 JSON 키는 허용하지 않습니다")
				}
				seen[name] = true
				if e = value(depth + 1); e != nil {
					return e
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if e := value(depth + 1); e != nil {
					return e
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("조회 JSON 형식이 올바르지 않습니다")
		}
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("조회 정의는 하나의 JSON 객체여야 합니다")
	}
	return nil
}
func parseDocumentQuery(raw string) (documentQueryDefinition, error) {
	var q documentQueryDefinition
	fail := func() (documentQueryDefinition, error) {
		return q, fmt.Errorf("조회 정의의 유형·필드·조건·한도를 확인하세요")
	}
	if len(raw) > documentQueryDefinitionBytes || documentQueryCheckJSON([]byte(raw)) != nil {
		return fail()
	}
	// encoding/json otherwise accepts case-insensitive struct field aliases.
	// Canonical DSL keys are exact; user property/parameter names remain case-sensitive.
	var shape map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &shape) != nil || shape == nil {
		return fail()
	}
	keys := func(value map[string]json.RawMessage, allowed ...string) bool {
		for key := range value {
			if !oneOf(key, allowed...) {
				return false
			}
		}
		return true
	}
	if !keys(shape, "version", "source", "columns", "filters", "order_by", "properties", "parameters", "space_id", "document_ids", "direction", "depth", "limit") {
		return fail()
	}
	for field, allowed := range map[string][]string{"columns": {"field", "label"}, "filters": {"field", "op", "value"}, "order_by": {"field", "direction"}} {
		if value, ok := shape[field]; ok {
			var items []map[string]json.RawMessage
			if json.Unmarshal(value, &items) != nil {
				return fail()
			}
			for _, item := range items {
				if !keys(item, allowed...) {
					return fail()
				}
			}
		}
	}
	if value, ok := shape["parameters"]; ok {
		var params map[string]map[string]json.RawMessage
		if json.Unmarshal(value, &params) != nil {
			return fail()
		}
		for _, param := range params {
			if !keys(param, "type", "label", "default", "required") {
				return fail()
			}
		}
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&q) != nil {
		return fail()
	}
	if q.Version != 1 || documentQueryFields[q.Source] == nil || len(q.Columns) < 1 || len(q.Columns) > 12 || len(q.Filters) > 8 || len(q.Order) > 2 || len(q.Properties) > 8 || len(q.Parameters) > 8 || q.Limit < 1 || q.Limit > 100 || len(q.DocumentIDs) > 50 || q.SpaceID != "" && !validID(q.SpaceID) {
		return fail()
	}
	if q.Source != "documents" && len(q.Properties) > 0 {
		return fail()
	}
	for name, kind := range q.Properties {
		if !documentQueryName.MatchString(name) || !oneOf(kind, "string", "number", "boolean", "date") {
			return fail()
		}
	}
	if q.Source == "relations" {
		if q.Direction == "" {
			q.Direction = "outgoing"
		}
		if q.Depth == 0 {
			q.Depth = 1
		}
		if !oneOf(q.Direction, "outgoing", "incoming") || q.Depth < 1 || q.Depth > 3 {
			return fail()
		}
	} else if q.Direction != "" || q.Depth != 0 {
		return fail()
	}
	ids := map[string]bool{}
	for i, id := range q.DocumentIDs {
		if !validID(id) || ids[strings.ToLower(id)] {
			return fail()
		}
		q.DocumentIDs[i] = strings.ToLower(id)
		ids[q.DocumentIDs[i]] = true
	}
	q.SpaceID = strings.ToLower(q.SpaceID)
	columns := map[string]bool{}
	for _, column := range q.Columns {
		if documentQueryFieldType(q, column.Field) == "" || columns[column.Field] || len(column.Label) > 200 {
			return fail()
		}
		columns[column.Field] = true
	}
	sorts := map[string]bool{}
	for _, order := range q.Order {
		kind := documentQueryFieldType(q, order.Field)
		if kind == "" || kind == "strings" || sorts[order.Field] || !oneOf(order.Direction, "asc", "desc") {
			return fail()
		}
		sorts[order.Field] = true
	}
	for name, param := range q.Parameters {
		if !documentQueryName.MatchString(name) || !oneOf(param.Type, "string", "number", "boolean", "date") || len(param.Label) > 200 {
			return fail()
		}
		if len(param.Default) > 0 {
			v, e := documentQueryJSONValue(param.Default)
			if e != nil || !documentQueryScalar(v, param.Type) {
				return fail()
			}
		}
	}
	for _, filter := range q.Filters {
		kind := documentQueryFieldType(q, filter.Field)
		if kind == "" || !oneOf(filter.Op, "eq", "ne", "contains", "in", "lt", "lte", "gt", "gte", "exists") {
			return fail()
		}
		if filter.Op == "exists" {
			if len(filter.Value) != 0 {
				return fail()
			}
			continue
		}
		if filter.Op == "contains" && !oneOf(kind, "string", "strings") {
			return fail()
		}
		if oneOf(filter.Op, "lt", "lte", "gt", "gte") && !oneOf(kind, "number", "date") {
			return fail()
		}
		if kind == "strings" && filter.Op != "contains" {
			return fail()
		}
		v, e := documentQueryJSONValue(filter.Value)
		if e != nil {
			return fail()
		}
		if ref, ok := v.(map[string]any); ok {
			key, ok := ref["parameter"].(string)
			param, defined := q.Parameters[key]
			want := kind
			if want == "strings" {
				want = "string"
			}
			if !ok || !defined || len(ref) != 1 || param.Type != want || filter.Op == "in" {
				return fail()
			}
			continue
		}
		if filter.Op == "in" {
			items, ok := v.([]any)
			if !ok || len(items) < 1 || len(items) > 20 {
				return fail()
			}
			for _, item := range items {
				if !documentQueryScalar(item, kind) {
					return fail()
				}
			}
		} else {
			if kind == "strings" {
				kind = "string"
			}
			if !documentQueryScalar(v, kind) {
				return fail()
			}
		}
	}
	return q, nil
}

func documentQueryFences(markdown string) ([]documentQueryFence, bool) {
	if strings.HasPrefix(markdown, "---\n") || strings.HasPrefix(markdown, "---\r\n") {
		closed := false
		for i, line := range strings.Split(markdown, "\n") {
			if i > 0 && strings.TrimSpace(line) == "---" {
				closed = true
				break
			}
		}
		if !closed {
			return []documentQueryFence{}, false
		}
	}
	source := markdownBodySource(markdown)
	parsed := goldmark.New().Parser().Parse(text.NewReader(source))
	out := []documentQueryFence{}
	limited := false
	_ = ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := node.(*ast.HTMLBlock); ok {
			return ast.WalkSkipChildren, nil
		}
		fence, ok := node.(*ast.FencedCodeBlock)
		if !ok || string(fence.Language(source)) != "madi-query" {
			return ast.WalkContinue, nil
		}
		if len(out) >= documentQueryMaxDefinitions {
			limited = true
			return ast.WalkStop, nil
		}
		var raw strings.Builder
		line := 0
		for i := 0; i < fence.Lines().Len(); i++ {
			segment := fence.Lines().At(i)
			if i == 0 {
				line = bytes.Count(source[:segment.Start], []byte{'\n'})
			}
			if raw.Len()+segment.Len() > documentQueryDefinitionBytes+2 {
				out = append(out, documentQueryFence{Line: line, Error: "조회 정의는 5 KiB 이하여야 합니다"})
				return ast.WalkSkipChildren, nil
			}
			raw.Write(segment.Value(source))
		}
		value := strings.TrimSpace(raw.String())
		entry := documentQueryFence{Hash: digest(value), Source: value, Line: line}
		q, e := parseDocumentQuery(value)
		if e != nil {
			entry.Error = e.Error()
		} else {
			entry.Definition = &q
		}
		out = append(out, entry)
		return ast.WalkSkipChildren, nil
	})
	return out, limited
}
