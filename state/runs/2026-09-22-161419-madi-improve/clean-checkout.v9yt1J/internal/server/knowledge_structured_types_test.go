package server

import (
	"strings"
	"testing"
)

func TestStructuredDraftParserGroundingTypesAndBounds(t *testing.T) {
	props := []map[string]any{{"id": "count", "name": "수량", "type": "number"}, {"id": "state", "name": "상태", "type": "select", "options": []string{"운영", "검토"}}, {"id": "date", "name": "일자", "type": "date"}}
	selection := "GPU 수량은 2개입니다. 상태는 운영입니다. 일자는 미정입니다."
	raw := `{"fields":[{"property_id":"count","value":2,"quote":"2개"},{"property_id":"state","value":"삭제","quote":"상태는 운영"}]}`
	got, e := parseStructuredOutput(raw, selection, 10, props)
	if e != nil || len(got.Fields) != 3 {
		t.Fatal(got, e)
	}
	if !got.Fields[0].Valid || got.Fields[0].StartByte != 10+strings.Index(selection, "2개") || got.Fields[0].ContentHash != digest("2개") {
		t.Fatal(got.Fields[0])
	}
	if got.Fields[1].Valid || got.Fields[2].Valid || got.Fields[2].Value != nil {
		t.Fatal("bad select/missing field silently became a value", got)
	}
	for _, date := range []string{"2026-02-30", "2026-09-09T00:00:00Z"} {
		f := structuredValidateField(props[2], structuredProviderField{PropertyID: "date", Value: date, Quote: "일자는 미정"}, selection, 0)
		if f.Valid {
			t.Fatal("structured date not representable in date editor", f)
		}
	}
	for _, raw := range []string{`{"fields":[{"property_id":"count","value":2,"value":3,"quote":"2개"}]}`, `{"fields":[],"fields":[]}`, `{"fields":[{"property_id":"extra","value":true,"quote":"2개"}]}`, `{"fields":[{"property_id":"count","value":2},{"property_id":"count","value":3}]}`, "```json\n{}\n```", `{"fields":[]} hidden note`, `{"fields":[],"execute":"shell"}`} {
		if _, e := parseStructuredOutput(raw, selection, 0, props); e == nil {
			t.Fatal("ambiguous/unknown response accepted", raw)
		}
	}
	if _, _, _, e := structuredQuote("2개, 2개", 0, "2개", nil); e == nil {
		t.Fatal("ambiguous quote silently picked first")
	}
	if _, _, _, e := structuredQuote("aaa", 0, "aa", nil); e == nil {
		t.Fatal("overlapping ambiguous quote silently picked first")
	}
	start := len("2개, ")
	if from, to, _, e := structuredQuote("2개, 2개", 5, "2개", &start); e != nil || from != 5+start || to-from != len("2개") {
		t.Fatal(from, to, e)
	}
	start = 1
	if _, _, _, e := structuredQuote("한글", 0, "한", &start); e == nil {
		t.Fatal("split UTF-8 coordinate accepted")
	}
	if _, e := structuredProperties(append(props, map[string]any{"id": "run", "name": "실행", "type": "button"}), []string{"run"}); e == nil {
		t.Fatal("execution property sent to extraction")
	}
	if _, e := structuredProperties(props, []string{"count", "count"}); e == nil {
		t.Fatal("duplicate selected properties")
	}
	if _, e := parseStructuredOutput(strings.Repeat(" ", structuredOutputMax+1), selection, 0, props); e == nil {
		t.Fatal("oversize output")
	}
}
