package server

import (
	"strings"
	"testing"
)

func TestAdvancedFormulaLanguage(t *testing.T) {
	values := map[string]any{"가격": float64(1250), "수량": float64(3), "목록": []any{float64(2), float64(5)}, "이름": "마디"}
	cases := []struct {
		expression string
		want       any
	}{
		{`prop("가격") * prop("수량")`, float64(3750)},
		{`if(prop("수량") > 2, concat(prop("이름"), " 팀"), "기타")`, "마디 팀"},
		{`if(false, 1 / 0, round(1.235, 2))`, float64(1.24)},
		{`dateAdd("2026-09-08", 2, "weeks")`, "2026-09-22"},
		{`dateBetween("2026-09-10", "2026-09-08", "days")`, float64(2)},
		{`format("2026-09-08", "YYYY/MM/DD")`, "2026/09/08"},
		{`contains("마디 지식", "지식") && length("마디") == 2`, true},
		{`sum(prop("목록"), 3)`, float64(10)},
		{`!false && (3 + 4 * 2) == 11`, true},
		{`true || 1 / 0`, true},
		{`format(42)`, "42"},
	}
	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			ast, e := compileFormula(tc.expression)
			if e != nil {
				t.Fatal(e)
			}
			got, e := evaluateFormula(ast, func(ref string) (any, error) { return values[ref], nil }, &formulaBudget{steps: 1000}, 0)
			if e != nil {
				t.Fatal(e)
			}
			if !formulaEqual(got, tc.want) {
				t.Fatalf("got %#v want %#v", got, tc.want)
			}
		})
	}
}
func TestAdvancedFormulaRejectsUnsafeOrUnboundedInput(t *testing.T) {
	for _, expression := range []string{`fetch("http://example.com")`, `eval("1+1")`, `prop(concat("a","b"))`, `process.env.SECRET`, `"unterminated`, `1 +`, strings.Repeat("(", 40) + "1" + strings.Repeat(")", 40), strings.Repeat("1+", 3000) + "1"} {
		t.Run(expression[:min(len(expression), 40)], func(t *testing.T) {
			if _, e := compileFormula(expression); e == nil {
				t.Fatal("unsafe/invalid formula accepted")
			}
		})
	}
	for _, expression := range []string{`1 / 0`, `round(2, 99)`, `dateAdd("2026-01-01", 999999999, "years")`, `1e308 * 1e308`, `dateAdd("invalid", 1)`} {
		ast, e := compileFormula(expression)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = evaluateFormula(ast, func(string) (any, error) { return nil, nil }, &formulaBudget{steps: 1000}, 0); e == nil {
			t.Fatalf("invalid calculation accepted: %s", expression)
		}
	}
	ast, _ := compileFormula(`sum(1,2,3,4,5)`)
	if _, e := evaluateFormula(ast, func(string) (any, error) { return nil, nil }, &formulaBudget{steps: 2}, 0); e == nil {
		t.Fatal("budget ignored")
	}
}
func TestAdvancedPropertyCyclesAndTypedValues(t *testing.T) {
	props := []map[string]any{{"id": "a", "name": "A", "type": "formula", "expression": `prop("b") + 1`}, {"id": "b", "name": "B", "type": "formula", "expression": `prop("a") + 1`}}
	if e := validateAdvancedProperties(props); e == nil {
		t.Fatal("formula cycle accepted")
	}
	props[1] = map[string]any{"id": "b", "name": "B", "type": "number"}
	if e := validateAdvancedProperties(props); e != nil {
		t.Fatal(e)
	}
	props[0]["expression"] = `prop("missing")`
	if e := validateAdvancedProperties(props); e == nil {
		t.Fatal("missing property accepted")
	}
	for _, tc := range []struct {
		p map[string]any
		v any
	}{{map[string]any{"type": "relation"}, []any{"not-uuid"}}, {map[string]any{"type": "relation"}, "one"}, {map[string]any{"type": "progress"}, float64(101)}, {map[string]any{"type": "formula"}, float64(1)}, {map[string]any{"type": "rollup"}, float64(1)}} {
		if validateAdvancedValue(tc.p, tc.v) == nil {
			t.Fatalf("invalid value accepted: %v %v", tc.p, tc.v)
		}
	}
	if e := validateAdvancedValue(map[string]any{"type": "relation"}, []any{newID(), newID()}); e != nil {
		t.Fatal(e)
	}
	if e := validateAdvancedValue(map[string]any{"type": "progress"}, float64(66)); e != nil {
		t.Fatal(e)
	}
}
func TestAdvancedAggregationAndFilterSemantics(t *testing.T) {
	for _, tc := range []struct {
		op   string
		want any
	}{{"count", float64(3)}, {"sum", float64(12)}, {"avg", float64(4)}, {"min", float64(2)}, {"max", float64(6)}} {
		got, e := aggregateAdvancedValues(tc.op, []any{float64(2), float64(4), float64(6)})
		if e != nil || !formulaEqual(got, tc.want) {
			t.Fatalf("%s = %v, %v", tc.op, got, e)
		}
	}
	if advancedFilterMatches(float64(2), advancedFilter{Operator: "gt", Value: float64(10)}) {
		t.Fatal("numeric comparison used lexicographic order")
	}
	if !advancedFilterMatches([]any{}, advancedFilter{Operator: "is_empty"}) {
		t.Fatal("empty relation not recognized")
	}
	if !advancedFilterMatches("MADI 지식", advancedFilter{Operator: "contains", Value: "madi"}) {
		t.Fatal("contains not case insensitive")
	}
}
