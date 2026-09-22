package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const structuredSelectionMax = 32 << 10
const structuredOutputMax = 64 << 10

type structuredProviderField struct {
	PropertyID string `json:"property_id"`
	Value      any    `json:"value"`
	Quote      string `json:"quote"`
	StartByte  *int   `json:"start_byte,omitempty"`
}
type structuredField struct {
	PropertyID  string `json:"property_id"`
	Value       any    `json:"value"`
	Quote       string `json:"quote"`
	StartByte   int    `json:"start_byte"`
	EndByte     int    `json:"end_byte"`
	ContentHash string `json:"content_hash"`
	Issue       string `json:"issue"`
	Valid       bool   `json:"valid"`
	HumanEdited bool   `json:"human_edited"`
}
type structuredPayload struct {
	Format      string            `json:"format"`
	Fields      []structuredField `json:"fields"`
	PropertyIDs []string          `json:"property_ids"`
}

func structuredPropertySupported(p map[string]any) bool {
	return oneOf(str(p, "type"), "text", "number", "checkbox", "select", "status", "multi_select", "multiselect", "date", "url", "email", "phone", "progress")
}
func structuredProperties(props []map[string]any, ids []string) ([]map[string]any, error) {
	if len(ids) < 1 || len(ids) > 32 {
		return nil, errors.New("구조화할 속성은1~32개를 명시적으로 선택하세요")
	}
	selected := []map[string]any{}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			return nil, errors.New("속성 ID를 중복 없이 선택하세요")
		}
		seen[id] = true
		var property map[string]any
		for _, p := range props {
			if str(p, "id") == id {
				property = p
				break
			}
		}
		if property == nil || !structuredPropertySupported(property) {
			return nil, errors.New("선택한 속성이 없거나 계산·자동화·사용자·관계 속성입니다. 일반 입력 속성만 선택하세요")
		}
		// Only the selected property's value schema is sent to the provider. Other
		// database properties, formulas, relations, rows and tools remain excluded.
		clean := map[string]any{"id": id, "name": str(property, "name"), "type": str(property, "type")}
		if oneOf(str(property, "type"), "select", "status", "multi_select", "multiselect") {
			clean["options"] = property["options"]
		}
		selected = append(selected, clean)
	}
	return selected, nil
}

// Reject duplicate JSON keys before typed decoding. Different consumers must
// never disagree about a field's value or quoted evidence. Output is bounded
// before this walk; depth and token caps bound unusual provider responses too.
func structuredUniqueJSON(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tokens := 0
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 12 {
			return errors.New("구조화 JSON 중첩 한도 초과")
		}
		tokens++
		if tokens > 4096 {
			return errors.New("구조화 JSON 항목 한도 초과")
		}
		token, e := dec.Token()
		if e != nil {
			return e
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			keys := map[string]bool{}
			for dec.More() {
				key, e := dec.Token()
				if e != nil {
					return e
				}
				name, ok := key.(string)
				if !ok || keys[name] {
					return errors.New("중복 JSON 필드는 허용하지 않습니다")
				}
				keys[name] = true
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
			end, e := dec.Token()
			if e != nil || end != json.Delim('}') {
				return errors.New("JSON 객체 닫기 오류")
			}
		case '[':
			for dec.More() {
				if e = walk(depth + 1); e != nil {
					return e
				}
			}
			end, e := dec.Token()
			if e != nil || end != json.Delim(']') {
				return errors.New("JSON 배열 닫기 오류")
			}
		default:
			return errors.New("JSON 구조를 확인하세요")
		}
		return nil
	}
	if e := walk(0); e != nil {
		return e
	}
	if _, e := dec.Token(); e != io.EOF {
		return errors.New("JSON 뒤에 추가 내용을 반환할 수 없습니다")
	}
	return nil
}

func structuredQuote(selection string, offset int, quote string, start *int) (int, int, string, error) {
	if quote == "" || len(quote) > 4096 || !utf8.ValidString(quote) || strings.ContainsRune(quote, '\x00') {
		return -1, -1, "", errors.New("근거는 선택 원문의 UTF-8 1~4096바이트 구간이어야 합니다")
	}
	position := -1
	if start != nil {
		position = *start
		if position < 0 || position > len(selection) || len(quote) > len(selection)-position || !utf8.ValidString(selection[:position]) || selection[position:position+len(quote)] != quote {
			return -1, -1, "", errors.New("제안한 근거 좌표가 원문과 일치하지 않습니다")
		}
	} else {
		if strings.Index(selection, quote) < 0 || strings.Index(selection, quote) != strings.LastIndex(selection, quote) {
			return -1, -1, "", errors.New("원문에 근거가 없거나 여러 번 나타납니다. 정확한 구간을 다시 선택하세요")
		}
		position = strings.Index(selection, quote)
	}
	return offset + position, offset + position + len(quote), digest(quote), nil
}
func structuredValidateField(p map[string]any, f structuredProviderField, selection string, offset int) structuredField {
	field := structuredField{PropertyID: f.PropertyID, Value: f.Value, Quote: f.Quote, StartByte: -1, EndByte: -1}
	var issues []string
	start, end, hash, e := structuredQuote(selection, offset, f.Quote, f.StartByte)
	if e != nil {
		issues = append(issues, e.Error())
	} else {
		field.StartByte, field.EndByte, field.ContentHash = start, end, hash
	}
	if f.Value == nil {
		issues = append(issues, "값을 확인할 수 없습니다. 자동으로 빈 값이나 기본값을 채우지 않았습니다.")
	}
	if len(jsonValue(f.Value)) > 8192 {
		issues = append(issues, "한 속성 값은 JSON8KiB 이하여야 합니다")
	}
	if e = validateValuesForProperties([]map[string]any{p}, map[string]any{f.PropertyID: f.Value}); e != nil {
		issues = append(issues, e.Error())
	}
	if str(p, "type") == "date" {
		date, ok := f.Value.(string)
		if !ok || !validKnowledgeDate(date) {
			issues = append(issues, "이 구조화 흐름의 날짜는 실제 존재하는 YYYY-MM-DD 형식이어야 합니다")
		}
	}
	field.Valid = len(issues) == 0
	field.Issue = strings.Join(issues, " · ")
	return field
}
func parseStructuredOutput(raw, selection string, offset int, props []map[string]any) (structuredPayload, error) {
	out := structuredPayload{Format: "madi-structured-draft-v1", Fields: []structuredField{}, PropertyIDs: []string{}}
	if len(raw) > structuredOutputMax || !utf8.ValidString(raw) || len(selection) > structuredSelectionMax || !utf8.ValidString(selection) || offset < 0 {
		return out, errors.New("선택 원문32KiB·구조화 응답64KiB·UTF-8 한도를 확인하세요")
	}
	if e := structuredUniqueJSON([]byte(raw)); e != nil {
		return out, e
	}
	var response struct {
		Fields []structuredProviderField `json:"fields"`
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if e := dec.Decode(&response); e != nil {
		return out, errors.New("AI는 fields 배열만 포함한 JSON을 반환해야 합니다")
	}
	if len(response.Fields) > 32 {
		return out, errors.New("제안 속성은32개 이하여야 합니다")
	}
	lookup := map[string]map[string]any{}
	found := map[string]bool{}
	for _, p := range props {
		lookup[str(p, "id")] = p
		out.PropertyIDs = append(out.PropertyIDs, str(p, "id"))
	}
	for _, f := range response.Fields {
		p := lookup[f.PropertyID]
		if p == nil || found[f.PropertyID] {
			return out, errors.New("선택하지 않은 속성 또는 중복 속성을 반환했습니다")
		}
		found[f.PropertyID] = true
		out.Fields = append(out.Fields, structuredValidateField(p, f, selection, offset))
	}
	for _, p := range props {
		if !found[str(p, "id")] {
			out.Fields = append(out.Fields, structuredField{PropertyID: str(p, "id"), StartByte: -1, EndByte: -1, Issue: "AI가 이 속성 값을 제안하지 않았습니다. 근거를 직접 확인하거나 반영 대상에서 제외하세요."})
		}
	}
	if len(out.Fields) == 0 {
		return out, errors.New("구조화할 속성이 없습니다")
	}
	if len(jsonValue(out)) > 256<<10 {
		return out, fmt.Errorf("근거와 검토 결과는 JSON256KiB 이하여야 합니다")
	}
	return out, nil
}
