package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var sqlSourceIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$#]{0,127}$`)

type SQLSourceFilter struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}
type SQLSourceOrder struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

// SQLSourcePlan is a deliberately closed query grammar, also suitable for an AI
// to propose. It cannot contain arbitrary SQL, functions, joins or procedures.
type SQLSourcePlan struct {
	Schema  string            `json:"schema"`
	Table   string            `json:"table"`
	Columns []string          `json:"columns"`
	Filters []SQLSourceFilter `json:"filters"`
	Order   []SQLSourceOrder  `json:"order"`
	Limit   int               `json:"limit"`
}

func sqlSourcePlan(raw any) (SQLSourcePlan, error) {
	var p SQLSourcePlan
	d := json.NewDecoder(strings.NewReader(string(jsonValue(raw))))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil {
		return p, errors.New("저장 쿼리 계획 형식을 확인하세요")
	}
	return p, nil
}
func buildSQLSourceQuery(kind string, p SQLSourcePlan, columns []sqlColumn, maxRows int) (string, []any, error) {
	fail := errors.New("쿼리는 허용된 테이블·컬럼과 제한된 SELECT 조건만 사용할 수 있습니다")
	if !sqlSourceIdentifier.MatchString(p.Schema) || !sqlSourceIdentifier.MatchString(p.Table) || len(p.Columns) == 0 || len(p.Columns) > 100 || len(p.Filters) > 30 || len(p.Order) > 10 || p.Limit < 1 || p.Limit > maxRows {
		return "", nil, fail
	}
	allowed := map[string]bool{}
	types := map[string]string{}
	for _, column := range columns {
		allowed[column.Name] = true
		types[column.Name] = strings.ToLower(column.Type)
	}
	valid := func(name string) bool { return sqlSourceIdentifier.MatchString(name) && allowed[name] }
	names := []string{}
	seen := map[string]bool{}
	for _, name := range p.Columns {
		if !valid(name) || seen[name] {
			return "", nil, fail
		}
		seen[name] = true
		columnType := types[name]
		if strings.Contains(columnType, "blob") || strings.Contains(columnType, "clob") || strings.Contains(columnType, "binary") || oneOf(columnType, "bytea", "image", "xml", "json", "jsonb", "array", "user-defined", "long", "long raw") {
			return "", nil, errors.New("바이너리·LOB·JSON·XML·배열·사용자 정의 타입은 메타데이터만 제공합니다. 작은 텍스트 조회용 원격 뷰를 등록하세요")
		}
		selected := sqlIdent(kind, name)
		if strings.Contains(columnType, "char") || strings.Contains(columnType, "text") || strings.Contains(columnType, "clob") {
			if kind != "oracle" {
				selected = "LEFT(" + selected + ",65537)"
			}
			selected += " AS " + sqlIdent(kind, name)
		}
		names = append(names, selected)
	}
	query := "SELECT "
	if kind == "mssql" {
		query += fmt.Sprintf("TOP (%d) ", p.Limit+1)
	}
	query += strings.Join(names, ",") + " FROM " + sqlIdent(kind, p.Schema) + "." + sqlIdent(kind, p.Table)
	conditions := []string{}
	args := []any{}
	for _, filter := range p.Filters {
		if !valid(filter.Column) {
			return "", nil, fail
		}
		name := sqlIdent(kind, filter.Column)
		if oneOf(filter.Operator, "is_null", "not_null") {
			op := " IS NULL"
			if filter.Operator == "not_null" {
				op = " IS NOT NULL"
			}
			conditions = append(conditions, name+op)
			continue
		}
		op := map[string]string{"eq": "=", "ne": "<>", "lt": "<", "lte": "<=", "gt": ">", "gte": ">=", "contains": "LIKE"}[filter.Operator]
		if op == "" {
			return "", nil, fail
		}
		value := filter.Value
		switch v := value.(type) {
		case string:
			if len(v) > 4096 {
				return "", nil, fail
			}
			if filter.Operator == "contains" {
				value = "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(v) + "%"
			}
		case bool, float64, json.Number:
			if filter.Operator == "contains" {
				return "", nil, fail
			}
		default:
			return "", nil, fail
		}
		args = append(args, value)
		condition := name + " " + op + " " + sqlParam(kind, len(args))
		if filter.Operator == "contains" {
			condition += " ESCAPE '!'"
		}
		conditions = append(conditions, condition)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	ordering := []string{}
	for _, order := range p.Order {
		if !valid(order.Column) || !oneOf(order.Direction, "asc", "desc") {
			return "", nil, fail
		}
		ordering = append(ordering, sqlIdent(kind, order.Column)+" "+strings.ToUpper(order.Direction))
	}
	if len(ordering) > 0 {
		query += " ORDER BY " + strings.Join(ordering, ",")
	}
	switch kind {
	case "oracle":
		query = fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM<=%d", query, p.Limit+1)
	case "mssql":
	default:
		query += fmt.Sprintf(" LIMIT %d", p.Limit+1)
	}
	return query, args, nil
}
func sqlSourceTableAllowed(c sqlSource, schema, table string) bool {
	return slices.Contains(listStrings(c.Config["tables"]), schema+"."+table)
}
func sqlSourceRows(ctx context.Context, tx *sql.Tx, query string, args []any, limit int) ([]map[string]any, bool, error) {
	rows, e := tx.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, false, errors.New("외부 SELECT 조회에 실패했습니다. 컬럼 형식·읽기 권한·시간 제한을 확인하세요")
	}
	defer rows.Close()
	columns, e := rows.Columns()
	if e != nil {
		return nil, false, e
	}
	out := []map[string]any{}
	size := 0
	for rows.Next() {
		if len(out) >= limit {
			return out, true, nil
		}
		values := make([]any, len(columns))
		targets := make([]any, len(values))
		for i := range values {
			targets[i] = &values[i]
		}
		if e = rows.Scan(targets...); e != nil {
			return nil, false, errors.New("외부 결과 형식을 읽을 수 없습니다")
		}
		row := map[string]any{}
		for i, v := range values {
			switch item := v.(type) {
			case []byte:
				if !utf8.Valid(item) || len(item) > 65536 {
					return nil, false, errors.New("결과 셀이 텍스트 64KB 제한을 초과하거나 바이너리입니다")
				}
				v = string(item)
			case string:
				if len(item) > 65536 {
					return nil, false, errors.New("결과 셀이 64KB 제한을 초과했습니다")
				}
			case time.Time:
				v = item.UTC().Format(time.RFC3339Nano)
			case nil, bool, int64, float64:
			default:
				v = fmt.Sprint(item)
			}
			row[columns[i]] = v
		}
		size += len(jsonValue(row))
		if size > 8<<20 {
			return nil, false, errors.New("조회 결과는 8MB 이하여야 합니다")
		}
		out = append(out, row)
	}
	if rows.Err() != nil {
		return nil, false, errors.New("외부 DB 결과 수신이 중단되었습니다")
	}
	return out, false, nil
}
