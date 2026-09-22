package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const formulaMaxSource = 4096
const formulaMaxNodes = 1000
const formulaMaxDepth = 32
const formulaMaxOutput = 65536

type formulaToken struct {
	kind, text string
	value      any
}
type formulaNode struct {
	kind, text string
	value      any
	children   []*formulaNode
}
type formulaParser struct {
	tokens     []formulaToken
	pos, nodes int
}
type formulaBudget struct{ steps int }

func lexFormula(source string) ([]formulaToken, error) {
	if len(source) == 0 || len(source) > formulaMaxSource {
		return nil, errors.New("수식은 1~4096바이트로 입력하세요")
	}
	tokens := []formulaToken{}
	for i := 0; i < len(source); {
		r, size := utf8.DecodeRuneInString(source[i:])
		if r == utf8.RuneError && size == 1 {
			return nil, errors.New("수식의 문자 인코딩을 확인하세요")
		}
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		if len(tokens) >= formulaMaxNodes {
			return nil, errors.New("수식이 너무 복잡합니다")
		}
		c := source[i]
		if c == '"' || c == '\'' {
			quote := c
			i++
			var b strings.Builder
			closed := false
			for i < len(source) {
				if source[i] == quote {
					i++
					closed = true
					break
				}
				if source[i] == '\\' {
					i++
					if i >= len(source) {
						break
					}
					switch source[i] {
					case 'n':
						b.WriteByte('\n')
					case 't':
						b.WriteByte('\t')
					case 'r':
						b.WriteByte('\r')
					case '\\', '"', '\'':
						b.WriteByte(source[i])
					default:
						return nil, errors.New("문자열 이스케이프를 확인하세요")
					}
					i++
					continue
				}
				b.WriteByte(source[i])
				i++
			}
			if !closed {
				return nil, errors.New("문자열의 닫는 따옴표가 없습니다")
			}
			tokens = append(tokens, formulaToken{kind: "literal", value: b.String()})
			continue
		}
		if c >= '0' && c <= '9' || c == '.' && i+1 < len(source) && source[i+1] >= '0' && source[i+1] <= '9' {
			start := i
			i++
			for i < len(source) && ((source[i] >= '0' && source[i] <= '9') || source[i] == '.') {
				i++
			}
			if i < len(source) && (source[i] == 'e' || source[i] == 'E') {
				i++
				if i < len(source) && (source[i] == '+' || source[i] == '-') {
					i++
				}
				for i < len(source) && source[i] >= '0' && source[i] <= '9' {
					i++
				}
			}
			number, e := strconv.ParseFloat(source[start:i], 64)
			if e != nil || math.IsNaN(number) || math.IsInf(number, 0) {
				return nil, errors.New("올바른 유한 숫자를 입력하세요")
			}
			tokens = append(tokens, formulaToken{kind: "literal", value: number})
			continue
		}
		if unicode.IsLetter(r) || r == '_' {
			start := i
			i += size
			for i < len(source) {
				rr, ss := utf8.DecodeRuneInString(source[i:])
				if !unicode.IsLetter(rr) && !unicode.IsDigit(rr) && rr != '_' {
					break
				}
				i += ss
			}
			word := source[start:i]
			switch word {
			case "true":
				tokens = append(tokens, formulaToken{kind: "literal", value: true})
			case "false":
				tokens = append(tokens, formulaToken{kind: "literal", value: false})
			case "null":
				tokens = append(tokens, formulaToken{kind: "literal", value: nil})
			default:
				tokens = append(tokens, formulaToken{kind: "identifier", text: word})
			}
			continue
		}
		if i+1 < len(source) && oneOf(source[i:i+2], "==", "!=", "<=", ">=", "&&", "||") {
			tokens = append(tokens, formulaToken{kind: "operator", text: source[i : i+2]})
			i += 2
			continue
		}
		if strings.ContainsRune("+-*/%!<>() ,", r) {
			kind := "operator"
			if oneOf(string(c), "(", ")", ",") {
				kind = string(c)
			}
			tokens = append(tokens, formulaToken{kind: kind, text: string(c)})
			i++
			continue
		}
		return nil, fmt.Errorf("수식에서 지원하지 않는 문자: %q", r)
	}
	tokens = append(tokens, formulaToken{kind: "eof"})
	return tokens, nil
}
func formulaPrecedence(op string) int {
	switch op {
	case "||":
		return 1
	case "&&":
		return 2
	case "==", "!=":
		return 3
	case "<", ">", "<=", ">=":
		return 4
	case "+", "-":
		return 5
	case "*", "/", "%":
		return 6
	}
	return 0
}
func (p *formulaParser) parse(min, depth int) (*formulaNode, error) {
	if depth > formulaMaxDepth {
		return nil, errors.New("수식 중첩은 32단계까지 지원합니다")
	}
	p.nodes++
	if p.nodes > formulaMaxNodes {
		return nil, errors.New("수식이 너무 복잡합니다")
	}
	tok := p.tokens[p.pos]
	p.pos++
	var node *formulaNode
	switch tok.kind {
	case "literal":
		node = &formulaNode{kind: "literal", value: tok.value}
	case "operator":
		if !oneOf(tok.text, "+", "-", "!") {
			return nil, errors.New("피연산자가 필요합니다")
		}
		child, e := p.parse(7, depth+1)
		if e != nil {
			return nil, e
		}
		node = &formulaNode{kind: "unary", text: tok.text, children: []*formulaNode{child}}
	case "(":
		inner, e := p.parse(1, depth+1)
		if e != nil {
			return nil, e
		}
		if p.tokens[p.pos].kind != ")" {
			return nil, errors.New("닫는 괄호가 없습니다")
		}
		p.pos++
		node = inner
	case "identifier":
		if p.tokens[p.pos].kind != "(" {
			return nil, fmt.Errorf("속성은 prop(\"속성 이름\")으로 참조하세요: %s", tok.text)
		}
		p.pos++
		args := []*formulaNode{}
		if p.tokens[p.pos].kind != ")" {
			for {
				arg, e := p.parse(1, depth+1)
				if e != nil {
					return nil, e
				}
				args = append(args, arg)
				if len(args) > 30 {
					return nil, errors.New("함수 인수는 30개까지 지원합니다")
				}
				if p.tokens[p.pos].kind != "," {
					break
				}
				p.pos++
			}
		}
		if p.tokens[p.pos].kind != ")" {
			return nil, errors.New("함수의 닫는 괄호가 없습니다")
		}
		p.pos++
		if !oneOf(tok.text, "prop", "if", "concat", "dateAdd", "dateBetween", "format", "contains", "length", "round", "sum") {
			return nil, fmt.Errorf("지원하지 않는 함수: %s", tok.text)
		}
		if tok.text == "prop" && (len(args) != 1 || args[0].kind != "literal") {
			return nil, errors.New("prop 함수에는 속성 이름 또는 ID 문자열 하나를 입력하세요")
		}
		if tok.text == "prop" {
			if _, ok := args[0].value.(string); !ok {
				return nil, errors.New("prop 인수는 문자열이어야 합니다")
			}
		}
		node = &formulaNode{kind: "call", text: tok.text, children: args}
	default:
		return nil, errors.New("수식의 값 또는 함수를 확인하세요")
	}
	for p.pos < len(p.tokens) {
		op := p.tokens[p.pos]
		precedence := formulaPrecedence(op.text)
		if op.kind != "operator" || precedence < min || precedence == 0 {
			break
		}
		p.pos++
		right, e := p.parse(precedence+1, depth+1)
		if e != nil {
			return nil, e
		}
		node = &formulaNode{kind: "binary", text: op.text, children: []*formulaNode{node, right}}
	}
	return node, nil
}
func compileFormula(source string) (*formulaNode, error) {
	tokens, e := lexFormula(source)
	if e != nil {
		return nil, e
	}
	parser := formulaParser{tokens: tokens}
	node, e := parser.parse(1, 0)
	if e != nil {
		return nil, e
	}
	if parser.tokens[parser.pos].kind != "eof" {
		return nil, errors.New("수식 뒤에 처리할 수 없는 내용이 있습니다")
	}
	return node, nil
}
func formulaDependencies(node *formulaNode) []string {
	out := []string{}
	var walk func(*formulaNode)
	walk = func(n *formulaNode) {
		if n.kind == "call" && n.text == "prop" {
			out = append(out, n.children[0].value.(string))
		}
		for _, child := range n.children {
			walk(child)
		}
	}
	walk(node)
	return out
}
func formulaNumber(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		if !math.IsNaN(x) && !math.IsInf(x, 0) {
			return x, nil
		}
	case int:
		return float64(x), nil
	case json.Number:
		f, e := x.Float64()
		if e == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
			return f, nil
		}
	case nil:
		return 0, nil
	}
	return 0, errors.New("숫자 값이 필요합니다")
}
func formulaString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.(bool); ok {
		if b {
			return "true"
		}
		return "false"
	}
	if n, e := formulaNumber(v); e == nil {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	b, _ := json.Marshal(v)
	return string(b)
}
func formulaTruth(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	}
	return true
}
func formulaEqual(a, b any) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(aa) == string(bb)
}
func formulaDate(v any) (time.Time, error) {
	s, ok := v.(string)
	if !ok {
		return time.Time{}, errors.New("날짜 문자열이 필요합니다")
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		if t, e := time.Parse(layout, s); e == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("날짜는 YYYY-MM-DD 또는 ISO 8601 형식이어야 합니다")
}
func evaluateFormula(node *formulaNode, resolve func(string) (any, error), budget *formulaBudget, depth int) (any, error) {
	budget.steps--
	if budget.steps < 0 {
		return nil, errors.New("수식 계산 한도를 초과했습니다")
	}
	if depth > formulaMaxDepth {
		return nil, errors.New("수식 중첩 한도를 초과했습니다")
	}
	eval := func(n *formulaNode) (any, error) { return evaluateFormula(n, resolve, budget, depth+1) }
	switch node.kind {
	case "literal":
		return node.value, nil
	case "unary":
		v, e := eval(node.children[0])
		if e != nil {
			return nil, e
		}
		if node.text == "!" {
			return !formulaTruth(v), nil
		}
		n, e := formulaNumber(v)
		if e != nil {
			return nil, e
		}
		if node.text == "-" {
			n = -n
		}
		return n, nil
	case "binary":
		a, e := eval(node.children[0])
		if e != nil {
			return nil, e
		}
		if node.text == "&&" && !formulaTruth(a) {
			return false, nil
		}
		if node.text == "||" && formulaTruth(a) {
			return true, nil
		}
		b, e := eval(node.children[1])
		if e != nil {
			return nil, e
		}
		switch node.text {
		case "==":
			return formulaEqual(a, b), nil
		case "!=":
			return !formulaEqual(a, b), nil
		case "&&":
			return formulaTruth(b), nil
		case "||":
			return formulaTruth(b), nil
		}
		if node.text == "+" {
			if _, ok := a.(string); ok {
				out := formulaString(a) + formulaString(b)
				if len(out) > formulaMaxOutput {
					return nil, errors.New("수식 출력이 너무 큽니다")
				}
				return out, nil
			}
		}
		if oneOf(node.text, "<", ">", "<=", ">=") {
			if aa, ok := a.(string); ok {
				bb, ok := b.(string)
				if !ok {
					return nil, errors.New("비교할 값의 유형이 다릅니다")
				}
				c := strings.Compare(aa, bb)
				switch node.text {
				case "<":
					return c < 0, nil
				case ">":
					return c > 0, nil
				case "<=":
					return c <= 0, nil
				default:
					return c >= 0, nil
				}
			}
		}
		aa, e := formulaNumber(a)
		if e != nil {
			return nil, e
		}
		bb, e := formulaNumber(b)
		if e != nil {
			return nil, e
		}
		var result float64
		switch node.text {
		case "+":
			result = aa + bb
		case "-":
			result = aa - bb
		case "*":
			result = aa * bb
		case "/":
			if bb == 0 {
				return nil, errors.New("0으로 나눌 수 없습니다")
			}
			result = aa / bb
		case "%":
			if bb == 0 {
				return nil, errors.New("0으로 나눌 수 없습니다")
			}
			result = math.Mod(aa, bb)
		case "<":
			return aa < bb, nil
		case ">":
			return aa > bb, nil
		case "<=":
			return aa <= bb, nil
		case ">=":
			return aa >= bb, nil
		}
		if math.IsInf(result, 0) || math.IsNaN(result) {
			return nil, errors.New("숫자 계산 범위를 초과했습니다")
		}
		return result, nil
	case "call":
		if node.text == "prop" {
			return resolve(node.children[0].value.(string))
		}
		if node.text == "if" {
			if len(node.children) != 3 {
				return nil, errors.New("if 함수에는 조건, 참일 때, 거짓일 때의 세 인수가 필요합니다")
			}
			condition, e := eval(node.children[0])
			if e != nil {
				return nil, e
			}
			if formulaTruth(condition) {
				return eval(node.children[1])
			}
			return eval(node.children[2])
		}
		args := make([]any, len(node.children))
		for i, child := range node.children {
			v, e := eval(child)
			if e != nil {
				return nil, e
			}
			args[i] = v
		}
		arity := func(min, max int) error {
			if len(args) < min || len(args) > max {
				return fmt.Errorf("%s 함수는 %d~%d개 인수가 필요합니다", node.text, min, max)
			}
			return nil
		}
		switch node.text {
		case "concat":
			var b strings.Builder
			for _, v := range args {
				b.WriteString(formulaString(v))
				if b.Len() > formulaMaxOutput {
					return nil, errors.New("수식 출력이 너무 큽니다")
				}
			}
			return b.String(), nil
		case "sum":
			sum := float64(0)
			var add func(any) error
			add = func(v any) error {
				budget.steps--
				if budget.steps < 0 {
					return errors.New("수식 계산 한도를 초과했습니다")
				}
				if a, ok := v.([]any); ok {
					for _, item := range a {
						if e := add(item); e != nil {
							return e
						}
					}
					return nil
				}
				n, e := formulaNumber(v)
				if e != nil {
					return e
				}
				sum += n
				return nil
			}
			for _, v := range args {
				if e := add(v); e != nil {
					return nil, e
				}
			}
			if math.IsInf(sum, 0) || math.IsNaN(sum) {
				return nil, errors.New("합계 범위를 초과했습니다")
			}
			return sum, nil
		case "contains":
			if e := arity(2, 2); e != nil {
				return nil, e
			}
			if a, ok := args[0].([]any); ok {
				for _, item := range a {
					budget.steps--
					if budget.steps < 0 {
						return nil, errors.New("수식 계산 한도를 초과했습니다")
					}
					if formulaEqual(item, args[1]) {
						return true, nil
					}
				}
				return false, nil
			}
			return strings.Contains(formulaString(args[0]), formulaString(args[1])), nil
		case "length":
			if e := arity(1, 1); e != nil {
				return nil, e
			}
			if a, ok := args[0].([]any); ok {
				return float64(len(a)), nil
			}
			return float64(utf8.RuneCountInString(formulaString(args[0]))), nil
		case "round":
			if e := arity(1, 2); e != nil {
				return nil, e
			}
			n, e := formulaNumber(args[0])
			if e != nil {
				return nil, e
			}
			digits := float64(0)
			if len(args) > 1 {
				digits, e = formulaNumber(args[1])
				if e != nil {
					return nil, e
				}
			}
			if digits < 0 || digits > 12 || digits != math.Trunc(digits) {
				return nil, errors.New("반올림 자릿수는 0~12 정수여야 합니다")
			}
			factor := math.Pow10(int(digits))
			out := math.Round(n*factor) / factor
			if math.IsInf(out, 0) {
				return nil, errors.New("반올림 범위를 초과했습니다")
			}
			return out, nil
		case "format":
			if e := arity(1, 2); e != nil {
				return nil, e
			}
			if len(args) == 1 {
				return formulaString(args[0]), nil
			}
			d, e := formulaDate(args[0])
			if e != nil {
				return nil, e
			}
			layout := strings.NewReplacer("YYYY", "2006", "MM", "01", "DD", "02", "HH", "15", "mm", "04", "ss", "05").Replace(formulaString(args[1]))
			if len(layout) > 200 {
				return nil, errors.New("날짜 형식이 너무 깁니다")
			}
			return d.Format(layout), nil
		case "dateAdd":
			if e := arity(2, 3); e != nil {
				return nil, e
			}
			d, e := formulaDate(args[0])
			if e != nil {
				return nil, e
			}
			amount, e := formulaNumber(args[1])
			if e != nil {
				return nil, e
			}
			if math.Abs(amount) > 365000 || amount != math.Trunc(amount) {
				return nil, errors.New("날짜 변경량은 ±365000 이내의 정수여야 합니다")
			}
			unit := "days"
			if len(args) > 2 {
				unit = formulaString(args[2])
			}
			switch unit {
			case "day", "days":
				d = d.AddDate(0, 0, int(amount))
			case "week", "weeks":
				d = d.AddDate(0, 0, int(amount)*7)
			case "month", "months":
				d = d.AddDate(0, int(amount), 0)
			case "year", "years":
				d = d.AddDate(int(amount), 0, 0)
			case "hour", "hours":
				d = d.Add(time.Duration(amount) * time.Hour)
			case "minute", "minutes":
				d = d.Add(time.Duration(amount) * time.Minute)
			default:
				return nil, errors.New("지원하는 날짜 단위: days, weeks, months, years, hours, minutes")
			}
			if d.Year() < 1 || d.Year() > 9999 {
				return nil, errors.New("날짜 범위를 초과했습니다")
			}
			if len(formulaString(args[0])) == 10 && !oneOf(unit, "hours", "hour", "minutes", "minute") {
				return d.Format("2006-01-02"), nil
			}
			return d.Format(time.RFC3339), nil
		case "dateBetween":
			if e := arity(2, 3); e != nil {
				return nil, e
			}
			a, e := formulaDate(args[0])
			if e != nil {
				return nil, e
			}
			b, e := formulaDate(args[1])
			if e != nil {
				return nil, e
			}
			unit := "days"
			if len(args) > 2 {
				unit = formulaString(args[2])
			}
			seconds := float64(a.Unix() - b.Unix())
			switch unit {
			case "day", "days":
				return math.Trunc(seconds / 86400), nil
			case "week", "weeks":
				return math.Trunc(seconds / (86400 * 7)), nil
			case "hour", "hours":
				return math.Trunc(seconds / 3600), nil
			case "minute", "minutes":
				return math.Trunc(seconds / 60), nil
			case "month", "months":
				return float64((a.Year()-b.Year())*12 + int(a.Month()) - int(b.Month())), nil
			case "year", "years":
				return float64(a.Year() - b.Year()), nil
			default:
				return nil, errors.New("날짜 단위를 확인하세요")
			}
		}
	}
	return nil, errors.New("수식을 계산할 수 없습니다")
}
