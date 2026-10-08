package querydsl

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	clausePattern = regexp.MustCompile(
		`(?i)^\s*([a-z_][a-z0-9_.]*)\s*(=|!=|<=|>=|<|>)\s*(?:"([^"]*)"|'([^']*)'|([a-z0-9_.:@/+ -]+))\s*$`,
	)
	safePath = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)
)

const (
	maxClauses = 20
	maxLength  = 4096
)

type Clause struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type Query struct {
	Clauses []Clause `json:"clauses"`
}

func Parse(input string) (Query, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Query{}, errors.New("query is empty")
	}
	if len(input) > maxLength {
		return Query{}, fmt.Errorf("query exceeds %d characters", maxLength)
	}
	parts, err := splitAND(input)
	if err != nil {
		return Query{}, err
	}
	if len(parts) > maxClauses {
		return Query{}, fmt.Errorf("query has more than %d clauses", maxClauses)
	}
	result := Query{Clauses: make([]Clause, 0, len(parts))}
	for _, part := range parts {
		match := clausePattern.FindStringSubmatch(part)
		if match == nil {
			return Query{}, fmt.Errorf("invalid clause %q", strings.TrimSpace(part))
		}
		field := normalizeField(match[1])
		if !allowedField(field) {
			return Query{}, fmt.Errorf("field %q is not allowed", field)
		}
		value := match[3]
		if value == "" {
			value = match[4]
		}
		if value == "" {
			value = strings.TrimSpace(match[5])
		}
		result.Clauses = append(result.Clauses, Clause{
			Field: field, Operator: match[2], Value: value,
		})
	}
	return result, nil
}

func (q Query) SQL(postgres bool) (string, []any, error) {
	if len(q.Clauses) == 0 {
		return "", nil, errors.New("query has no clauses")
	}
	conditions := []string{"deleted_at IS NULL"}
	args := make([]any, 0, len(q.Clauses))
	for _, clause := range q.Clauses {
		column, err := columnFor(clause.Field, postgres)
		if err != nil {
			return "", nil, err
		}
		if isAttributeInequality(clause) {
			args = append(args, clause.Value)
			conditions = append(conditions, fmt.Sprintf(
				"(%s IS NULL OR %s != $%d)", column, column, len(args),
			))
			continue
		}
		if number, ok := numericAttributeClause(clause); ok {
			args = append(args, number, clause.Value)
			conditions = append(conditions, numericAttributeCondition(
				clause.Field, clause.Operator, postgres,
				len(args)-1, len(args),
			))
			continue
		}
		value := any(clause.Value)
		if clause.Field == "confidence" {
			number, err := strconv.ParseFloat(clause.Value, 64)
			if err != nil {
				return "", nil, errors.New("confidence must be numeric")
			}
			value = number
		}
		if clause.Field == "last_seen_at" || clause.Field == "first_seen_at" {
			moment, err := parseTime(clause.Value)
			if err != nil {
				return "", nil, err
			}
			value = moment
		}
		if clause.Field == "id" {
			identifier, err := parseIdentifier(clause.Value)
			if err != nil {
				return "", nil, err
			}
			value = identifier
		}
		args = append(args, value)
		conditions = append(
			conditions,
			fmt.Sprintf("%s %s $%d", column, clause.Operator, len(args)),
		)
	}
	return strings.Join(conditions, " AND "), args, nil
}

// timeLayouts are the absolute forms a time clause may name, most specific
// first. A form without a zone is read as UTC, which is the zone every stored
// timestamp is written in.
var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02",
}

// parseTime resolves a time clause's value to an instant. Only the relative
// "now - 24h" form was resolved before and anything else was handed to the
// database as text, which was wrong in both storage modes. PostgreSQL compares
// TIMESTAMPTZ, so a value it could not read as a timestamp failed the statement
// and the operator got HTTP 500 with no hint of the typo. The SQLite fallback
// stores the column as text written from a Go time, so "2026-01-01T00:00:00Z"
// was compared byte by byte against "2026-01-01 00:00:00 +0000 UTC" - no error,
// just the wrong rows.
func parseTime(value string) (time.Time, error) {
	text := strings.TrimSpace(value)
	if strings.EqualFold(text, "now") {
		return time.Now().UTC(), nil
	}
	if rest, found := cutRelativeNow(text); found {
		duration, err := time.ParseDuration(strings.TrimSpace(rest))
		if err != nil {
			return time.Time{}, errors.New("relative time duration is invalid")
		}
		return time.Now().UTC().Add(-duration), nil
	}
	for _, layout := range timeLayouts {
		if moment, err := time.Parse(layout, text); err == nil {
			return moment.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf(
		"time value %q is neither %s nor a timestamp such as "+
			`"2026-01-31T09:00:00Z"`,
		text, `"now - 24h"`,
	)
}

// parseIdentifier resolves an id clause's value to the one spelling the column
// holds. The value used to be handed to the database as the caller typed it,
// and the id column is a UUID: PostgreSQL failed the statement on anything it
// could not read as one, so a mistyped identifier came back as HTTP 500 with
// no hint of the value, and the SQLite fallback compared it as text and
// answered 200 with an empty list as though the asset did not exist. The two
// modes also disagreed on a UUID that is merely spelled differently -
// PostgreSQL reads an upper-cased or undashed one as the same value and the
// fallback's text comparison does not - so the spelling is folded here to the
// canonical form every asset is stored under.
func parseIdentifier(value string) (string, error) {
	identifier, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf(
			"id value %q is not an asset UUID such as "+
				`"0d0f4d64-9d5b-4f4a-9a1e-3a6f2c8b7e10"`,
			value,
		)
	}
	return identifier.String(), nil
}

func cutRelativeNow(text string) (string, bool) {
	if len(text) < 4 || !strings.EqualFold(text[:3], "now") {
		return "", false
	}
	rest := strings.TrimSpace(text[3:])
	if !strings.HasPrefix(rest, "-") {
		return "", false
	}
	return strings.TrimPrefix(rest, "-"), true
}

// Field describes one queryable field for the console's reference panel. The
// grammar was only discoverable by trial and error: an operator had to guess a
// field name, read "field ... is not allowed", and guess again.
type Field struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// fields is the single source for both the parser's allowlist and the published
// reference, so the two cannot disagree.
var fields = []Field{
	{"name", "text", "자산 이름", `name = "web-01"`},
	{"asset_key", "text", "수집 원천이 부여한 고유 키", `asset_key = "host:web-01"`},
	{
		"id", "uuid", "자산 UUID. 대소문자와 하이픈 표기는 구분하지 않습니다",
		`id = "0d0f4d64-9d5b-4f4a-9a1e-3a6f2c8b7e10"`,
	},
	{"type", "text", "자산 유형", `type = "host"`},
	{"status", "text", "수명주기 상태", `status = "active"`},
	{"environment", "text", "분류가 판정한 운영 환경", `environment = "production"`},
	{"criticality", "text", "분류가 판정한 중요도", `criticality = "critical"`},
	{"owner_department", "text", "담당 부서", `owner_department = "플랫폼"`},
	{"location", "text", "위치", `location = "IDC-1"`},
	{"source", "text", "수집 원천", `source = "agent"`},
	{"confidence", "number", "분류 확신도 0~1", "confidence >= 0.8"},
	{
		"first_seen_at", "time",
		`최초 확인 시각. "now - 168h" 또는 "2026-01-31T09:00:00Z"`,
		`first_seen_at >= "now - 168h"`,
	},
	{
		"last_seen_at", "time",
		`최근 확인 시각. "now - 24h" 또는 "2026-01-31T09:00:00Z"`,
		`last_seen_at < "now - 24h"`,
	},
	{
		"attributes.*", "path",
		"수집 속성 경로. 대소문자를 구분하며, 숫자로 저장된 값은 " +
			"<, <=, >, >= 비교에서 숫자로 비교합니다. != 는 해당 속성이 " +
			"없는 자산도 포함합니다. 예: attributes.os_name",
		`attributes.os_name = "Ubuntu"`,
	},
}

// Grammar is everything the console needs to explain the query language.
type Grammar struct {
	Fields      []Field  `json:"fields"`
	Operators   []string `json:"operators"`
	Combinator  string   `json:"combinator"`
	MaxClauses  int      `json:"max_clauses"`
	MaxLength   int      `json:"max_length"`
	RelativeNow string   `json:"relative_now"`
}

func Describe() Grammar {
	return Grammar{
		Fields:      fields,
		Operators:   []string{"=", "!=", "<", "<=", ">", ">="},
		Combinator:  "AND",
		MaxClauses:  maxClauses,
		MaxLength:   maxLength,
		RelativeNow: `"now - 24h"`,
	}
}

const attributePrefix = "attributes."

// normalizeField folds a field name to the spelling the grammar uses. A column
// is named case-insensitively, but everything after "attributes." is a key in
// the stored JSON document and JSON keys are case sensitive. Folding the whole
// field lowercased that key too, so "attributes.assetTag" asked the document
// for "assettag", a key nobody wrote, and the query answered with no rows and
// no error - and /query/validate had already called the expression valid.
func normalizeField(field string) string {
	if len(field) > len(attributePrefix) &&
		strings.EqualFold(field[:len(attributePrefix)], attributePrefix) {
		return attributePrefix + field[len(attributePrefix):]
	}
	return strings.ToLower(field)
}

func allowedField(field string) bool {
	for _, candidate := range fields {
		if candidate.Name == field {
			return true
		}
	}
	return strings.HasPrefix(field, attributePrefix) &&
		safePath.MatchString(strings.TrimPrefix(field, attributePrefix))
}

func columnFor(field string, postgres bool) (string, error) {
	if !allowedField(field) {
		return "", fmt.Errorf("field %q is not allowed", field)
	}
	if !strings.HasPrefix(field, attributePrefix) {
		return field, nil
	}
	text, _, _ := attributeExpressions(field, postgres)
	return text, nil
}

// attributeExpressions returns the three things an attributes.* path is asked
// for: the extracted value as text, the same value as a number, and the JSON
// type of the stored value.
//
// The text form has to render a stored value the same way in both storage
// modes, because every clause but an ordering comparison compares it with the
// text the caller typed. PostgreSQL's #>> already does that. SQLite's
// json_extract instead hands back the value with its SQL type, and a
// comparison there converts nothing: an INTEGER is never equal to a TEXT
// parameter, and a JSON boolean arrives as 1 or 0 rather than as 'true' or
// 'false'. So "attributes.cpu_count = 8" answered HTTP 200 with an empty list
// on a value an agent reports as a JSON number - and its mirror image,
// "attributes.cpu_count != 8", kept every asset including the ones holding 8.
func attributeExpressions(
	field string, postgres bool,
) (text, number, jsonType string) {
	path := strings.Split(strings.TrimPrefix(field, attributePrefix), ".")
	if postgres {
		braced := "'{" + strings.Join(path, ",") + "}'"
		extract := "attributes_json #>> " + braced
		return extract, "(" + extract + ")::double precision",
			"jsonb_typeof(attributes_json #> " + braced + ")"
	}
	dotted := "'$." + strings.Join(path, ".") + "'"
	extract := "json_extract(attributes_json, " + dotted + ")"
	kind := "json_type(attributes_json, " + dotted + ")"
	// CAST leaves SQL NULL alone, so a path no asset reported still extracts
	// NULL and the inequality clause below still recognises it.
	return fmt.Sprintf(
		"(CASE %s WHEN 'true' THEN 'true' WHEN 'false' THEN 'false' "+
			"ELSE CAST(%s AS TEXT) END)",
		kind, extract,
	), extract, kind
}

// isAttributeInequality reports whether a clause asks which assets an attribute
// does not describe. An attribute path is extracted from the stored document,
// and an asset that never reported the key extracts SQL NULL, where "!=" is
// unknown rather than true. The asset was therefore dropped from a clause it
// plainly satisfies: attributes.env != "prod" answered without every asset
// whose env was never collected - the unlabelled ones an operator asking what
// is not production most needs to see - with HTTP 200 and nothing saying so.
// Every queryable column is NOT NULL, so only an attribute path is affected.
func isAttributeInequality(clause Clause) bool {
	return clause.Operator == "!=" &&
		strings.HasPrefix(clause.Field, attributePrefix)
}

// numericAttributeClause reports the number an ordering comparison on an
// attribute path is asking about. An attribute is extracted from the document
// as text, so "attributes.memory_bytes >= 2000000000" compared digit strings:
// "16000000000" sorts before "2000000000" because '1' < '2', and a host with
// 16 GB was reported as having less memory than the bound. Equality is left
// alone - it is already right for text, and reading "1.10" as a number there
// would stop an attribute holding a version from matching itself.
func numericAttributeClause(clause Clause) (float64, bool) {
	if !strings.HasPrefix(clause.Field, attributePrefix) {
		return 0, false
	}
	switch clause.Operator {
	case "<", "<=", ">", ">=":
	default:
		return 0, false
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(clause.Value), 64)
	if err != nil {
		return 0, false
	}
	return number, true
}

// numericAttributeCondition compares a stored JSON number as a number and
// anything else as text, so the fix cannot take rows away from a clause that
// orders text today, such as attributes.os_version > "20.04" over "22.04".
func numericAttributeCondition(
	field string,
	operator string,
	postgres bool,
	numberArg int,
	textArg int,
) string {
	text, number, jsonType := attributeExpressions(field, postgres)
	isNumber := jsonType + " IN ('integer','real')"
	if postgres {
		isNumber = jsonType + " = 'number'"
	}
	return fmt.Sprintf(
		"(CASE WHEN %s THEN %s %s $%d ELSE %s %s $%d END)",
		isNumber, number, operator, numberArg, text, operator, textArg,
	)
}

func splitAND(input string) ([]string, error) {
	parts := make([]string, 0)
	start := 0
	var quote rune
	runes := []rune(input)
	for index := 0; index < len(runes); index++ {
		character := runes[index]
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			continue
		}
		if character == '"' || character == '\'' {
			quote = character
			continue
		}
		if index+3 <= len(runes) &&
			strings.EqualFold(string(runes[index:index+3]), "AND") &&
			(index == 0 || isSpace(runes[index-1])) &&
			(index+3 == len(runes) || isSpace(runes[index+3])) {
			parts = append(parts, string(runes[start:index]))
			start = index + 3
			index += 2
		}
	}
	if quote != 0 {
		return nil, errors.New("query contains an unterminated string")
	}
	parts = append(parts, string(runes[start:]))
	return parts, nil
}

func isSpace(character rune) bool {
	return character == ' ' || character == '\t' ||
		character == '\r' || character == '\n'
}
