package httpapi

import (
	"strings"
	"testing"
)

// A language model writes JSON by prediction, not by a schema compiler. It sends
// numbers as strings, booleans as words, and occasionally an extra key. Every one
// of those used to be rejected as "arguments are invalid", which named nothing,
// so the model had nothing to correct and repeated the same call.
func TestArgumentsAcceptWhatAModelActuallySends(t *testing.T) {
	schema := objectSchema(map[string]any{
		"q":                    map[string]any{"type": "string"},
		"limit":                map[string]any{"type": "integer"},
		"offset":               map[string]any{"type": "integer"},
		"include_observations": map[string]any{"type": "boolean"},
	}, nil)

	cases := []struct {
		name      string
		raw       string
		limit     int
		observing bool
		query     string
	}{
		{"native types", `{"q":"web","limit":10,"include_observations":true}`, 10, true, "web"},
		// The three shapes that failed before.
		{"number as string", `{"limit":"10"}`, 10, false, ""},
		{"boolean as string", `{"include_observations":"true"}`, 50, true, ""},
		{"whole number as float", `{"limit":10.0}`, 10, false, ""},
		{"boolean as 1", `{"include_observations":1}`, 50, true, ""},
		{"absent arguments", ``, 50, false, ""},
		{"explicit null", `null`, 50, false, ""},
		{"null value for a field", `{"q":null,"limit":null}`, 50, false, ""},
	}
	for _, testCase := range cases {
		arguments := newMCPArguments("asset_search", []byte(testCase.raw), schema)
		query := arguments.String("q")
		limit := arguments.Int("limit", 50, 1, 100)
		observing := arguments.Bool("include_observations", false)
		if err := arguments.Err(); err != nil {
			t.Errorf("%s: %v", testCase.name, err)
			continue
		}
		if limit != testCase.limit || observing != testCase.observing || query != testCase.query {
			t.Errorf("%s: limit=%d observing=%v q=%q", testCase.name, limit, observing, query)
		}
	}
}

// When an argument genuinely cannot be read, the reply has to be usable: which
// parameter, what arrived, and what is accepted. Otherwise the caller can only
// guess, and a guessing caller retries the same mistake.
func TestArgumentErrorsNameTheParameterAndTheAcceptedValues(t *testing.T) {
	schema := objectSchema(map[string]any{
		"q":      map[string]any{"type": "string"},
		"limit":  map[string]any{"type": "integer"},
		"offset": map[string]any{"type": "integer"},
	}, nil)

	arguments := newMCPArguments("asset_search", []byte(`{"limit":"many"}`), schema)
	arguments.Int("limit", 50, 1, 100)
	err := arguments.Err()
	if err == nil {
		t.Fatal("an unreadable limit must be rejected")
	}
	for _, expected := range []string{`"limit"`, "between 1 and 100", `"many"`, "Accepted parameters"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("the error must contain %q: %s", expected, err)
		}
	}

	// Out of range is a different mistake from unreadable, and says so.
	arguments = newMCPArguments("asset_search", []byte(`{"limit":5000}`), schema)
	arguments.Int("limit", 50, 1, 100)
	if err := arguments.Err(); err == nil || !strings.Contains(err.Error(), "5000") {
		t.Fatalf("an out-of-range limit must name the value: %v", err)
	}

	// A misspelled parameter is the most recoverable mistake of all, if the
	// reply says what was meant.
	arguments = newMCPArguments("asset_search", []byte(`{"limitt":10}`), schema)
	err = arguments.Err()
	if err == nil || !strings.Contains(err.Error(), `did you mean "limit"`) {
		t.Fatalf("a near-miss parameter must be suggested: %v", err)
	}

	// Every problem at once, so one reply is enough to fix the call.
	arguments = newMCPArguments("asset_search", []byte(`{"limit":"x","nope":1}`), schema)
	arguments.Int("limit", 50, 1, 100)
	err = arguments.Err()
	if err == nil ||
		!strings.Contains(err.Error(), `"limit"`) ||
		!strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("all problems must be reported together: %v", err)
	}
}

func TestRequiredArgumentsAndEnumsExplainThemselves(t *testing.T) {
	schema := objectSchema(map[string]any{
		"asset_id": map[string]any{"type": "string"},
	}, []string{"asset_id"})

	arguments := newMCPArguments("asset_get", []byte(`{}`), schema)
	arguments.RequiredString("asset_id")
	if err := arguments.Err(); err == nil || !strings.Contains(err.Error(), `"asset_id" is required`) {
		t.Fatalf("a missing required argument must say so: %v", err)
	}

	if err := mustBeOneOf("runtime_state", "runing", "running", "stopped", "unknown"); err == nil ||
		!strings.Contains(err.Error(), "running, stopped, unknown") ||
		!strings.Contains(err.Error(), `"runing"`) {
		t.Fatalf("an enum rejection must list the accepted values: %v", err)
	}
	if err := mustBeOneOf("runtime_state", "", "running"); err != nil {
		t.Fatalf("an absent optional enum is not an error: %v", err)
	}
}
