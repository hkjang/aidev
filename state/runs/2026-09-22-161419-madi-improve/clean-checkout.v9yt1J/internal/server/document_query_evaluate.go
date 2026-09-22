package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type documentQueryRow struct {
	Values     map[string]any `json:"values"`
	DocumentID string         `json:"document_id"`
	Version    int            `json:"version"`
	Line       int            `json:"line,omitempty"`
	Sources    []string       `json:"-"`
}

func documentQueryParameters(q documentQueryDefinition, raw map[string]json.RawMessage) (map[string]any, error) {
	bad := fmt.Errorf("조회 매개변수의 이름·유형·필수 값을 확인하세요")
	if len(raw) > 8 {
		return nil, bad
	}
	out := map[string]any{}
	for name := range raw {
		if _, ok := q.Parameters[name]; !ok {
			return nil, bad
		}
	}
	for name, p := range q.Parameters {
		v, ok := raw[name]
		if !ok {
			v = p.Default
		}
		if len(v) == 0 {
			if p.Required {
				return nil, bad
			}
			continue
		}
		if documentQueryCheckJSON(v) != nil {
			return nil, bad
		}
		value, e := documentQueryJSONValue(v)
		if e != nil || !documentQueryScalar(value, p.Type) {
			return nil, bad
		}
		out[name] = value
	}
	for _, f := range q.Filters {
		v, _ := documentQueryJSONValue(f.Value)
		if ref, ok := v.(map[string]any); ok {
			name, _ := ref["parameter"].(string)
			if _, ok = out[name]; !ok {
				return nil, bad
			}
		}
	}
	return out, nil
}
func documentQueryNumber(v any) (float64, bool) {
	var n float64
	switch v := v.(type) {
	case json.Number:
		var e error
		n, e = v.Float64()
		if e != nil {
			return 0, false
		}
	case float64:
		n = v
	case int:
		n = float64(v)
	case int64:
		n = float64(v)
	case uint64:
		n = float64(v)
	default:
		return 0, false
	}
	return n, !math.IsNaN(n) && !math.IsInf(n, 0) && math.Abs(n) <= 9007199254740991
}
func documentQueryCompare(a, b any, kind string) (int, bool) {
	if a == nil || b == nil {
		return 0, false
	}
	switch kind {
	case "number":
		x, ok := documentQueryNumber(a)
		y, yes := documentQueryNumber(b)
		if !ok || !yes {
			return 0, false
		}
		if x < y {
			return -1, true
		}
		if x > y {
			return 1, true
		}
		return 0, true
	case "date":
		x, ok := a.(string)
		y, yes := b.(string)
		if !ok || !yes {
			return 0, false
		}
		xt, e := documentQueryDate(x)
		yt, f := documentQueryDate(y)
		if e != nil || f != nil {
			return 0, false
		}
		return xt.Compare(yt), true
	case "boolean":
		x, ok := a.(bool)
		y, yes := b.(bool)
		if !ok || !yes {
			return 0, false
		}
		if x == y {
			return 0, true
		}
		if !x {
			return -1, true
		}
		return 1, true
	default:
		x, ok := a.(string)
		y, yes := b.(string)
		return strings.Compare(x, y), ok && yes
	}
}
func documentQueryMatches(q documentQueryDefinition, row map[string]any, params map[string]any) bool {
	for _, f := range q.Filters {
		actual := row[f.Field]
		if f.Op == "exists" {
			if actual == nil {
				return false
			}
			if v, ok := actual.(string); ok && v == "" {
				return false
			}
			continue
		}
		wanted, _ := documentQueryJSONValue(f.Value)
		if ref, ok := wanted.(map[string]any); ok {
			wanted = params[ref["parameter"].(string)]
		}
		kind := documentQueryFieldType(q, f.Field)
		if f.Op == "contains" {
			needle, _ := wanted.(string)
			matched := false
			if kind == "strings" {
				for _, v := range listStrings(actual) {
					if v == needle {
						matched = true
						break
					}
				}
			} else if value, ok := actual.(string); ok {
				matched = strings.Contains(value, needle)
			}
			if !matched {
				return false
			}
			continue
		}
		if f.Op == "in" {
			matched := false
			for _, v := range wanted.([]any) {
				if cmp, ok := documentQueryCompare(actual, v, kind); ok && cmp == 0 {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
			continue
		}
		cmp, ok := documentQueryCompare(actual, wanted, kind)
		if !ok {
			return false
		}
		if !(f.Op == "eq" && cmp == 0 || f.Op == "ne" && cmp != 0 || f.Op == "lt" && cmp < 0 || f.Op == "lte" && cmp <= 0 || f.Op == "gt" && cmp > 0 || f.Op == "gte" && cmp >= 0) {
			return false
		}
	}
	return true
}
func documentQuerySort(q documentQueryDefinition, rows []documentQueryRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		for _, o := range q.Order {
			a, b := rows[i].Values[o.Field], rows[j].Values[o.Field]
			if a == nil || b == nil {
				if a == nil && b != nil {
					return false
				}
				if a != nil && b == nil {
					return true
				}
				continue
			}
			n, ok := documentQueryCompare(a, b, documentQueryFieldType(q, o.Field))
			if ok && n != 0 {
				if o.Direction == "desc" {
					return n > 0
				}
				return n < 0
			}
		}
		return false
	})
}

// Only top-level scalar Front Matter properties are available. Aliases and
// nested YAML are not expanded; no arbitrary traversal or conversion occurs.
func documentQueryProperties(md string, declared map[string]string) (map[string]any, bool) {
	out := map[string]any{}
	if !strings.HasPrefix(md, "---\n") && !strings.HasPrefix(md, "---\r\n") {
		return out, false
	}
	md = strings.ReplaceAll(md, "\r\n", "\n")
	end := -1
	offset := 4
	for _, line := range strings.SplitAfter(md[4:], "\n") {
		if strings.TrimSpace(line) == "---" {
			end = offset - 4
			break
		}
		offset += len(line)
		if offset > 32772 {
			break
		}
	}
	if end < 0 || end > 32768 {
		return out, true
	}
	var root yaml.Node
	if yaml.NewDecoder(bytes.NewBufferString(md[4:4+end])).Decode(&root) != nil || len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return out, true
	}
	seen := map[string]bool{}
	bad := false
	m := root.Content[0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		if seen[k.Value] {
			return map[string]any{}, true
		}
		seen[k.Value] = true
		kind, ok := declared[k.Value]
		if !ok {
			continue
		}
		if k.Kind != yaml.ScalarNode || v.Kind != yaml.ScalarNode || v.Anchor != "" {
			bad = true
			continue
		}
		var value any
		switch kind {
		case "string":
			if v.Tag == "!!str" && len(v.Value) <= 500 {
				value = v.Value
			}
		case "boolean":
			if v.Tag == "!!bool" {
				var b bool
				if v.Decode(&b) == nil {
					value = b
				}
			}
		case "number":
			if oneOf(v.Tag, "!!int", "!!float") {
				var n float64
				if v.Decode(&n) == nil {
					if _, ok := documentQueryNumber(n); ok {
						value = n
					}
				}
			}
		case "date":
			if oneOf(v.Tag, "!!str", "!!timestamp") {
				if _, e := documentQueryDate(v.Value); e == nil {
					value = v.Value
				} else if v.Tag == "!!timestamp" {
					var stamp time.Time
					if v.Decode(&stamp) == nil {
						value = stamp.Format(time.RFC3339)
					}
				}
			}
		}
		if value == nil {
			bad = true
		} else {
			out["property."+k.Value] = value
		}
	}
	return out, bad
}
