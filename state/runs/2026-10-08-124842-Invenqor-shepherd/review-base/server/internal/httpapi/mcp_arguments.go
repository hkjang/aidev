package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Reading tool arguments from a language model is not the same as reading them
// from a program, and decoding them as if it were is why tool calls kept failing.
//
// Two things went wrong. Arguments were decoded straight into Go structs with
// DisallowUnknownFields, so a model that sent `"limit": "50"` - a number as a
// string, which every model does sometimes - or one extra key got a flat
// rejection. And the rejection said only "asset_search arguments are invalid",
// which names neither the parameter nor the problem, so the model had nothing to
// correct and simply tried the same call again.
//
// So arguments are read leniently where the meaning is unambiguous, and every
// refusal names the parameter, what arrived, and what is accepted.
type mcpArguments struct {
	tool     string
	values   map[string]any
	accepted []string
	problems []string
}

// newMCPArguments reads the arguments object and checks its keys against the
// tool's own schema, so the accepted names come from the same declaration the
// client was given rather than a second list that can drift from it.
func newMCPArguments(
	tool string,
	raw json.RawMessage,
	schema map[string]any,
) *mcpArguments {
	arguments := &mcpArguments{tool: tool, values: map[string]any{}}
	if properties, ok := schema["properties"].(map[string]any); ok {
		for name := range properties {
			arguments.accepted = append(arguments.accepted, name)
		}
		sort.Strings(arguments.accepted)
	}
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return arguments
	}
	// UseNumber keeps large integers exact and lets a float-shaped whole number
	// still be read as an integer.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&arguments.values); err != nil {
		arguments.problems = append(arguments.problems, fmt.Sprintf(
			"the arguments must be a JSON object; %v", err,
		))
		return arguments
	}
	for name := range arguments.values {
		if arguments.isAccepted(name) {
			continue
		}
		problem := fmt.Sprintf("unknown parameter %q", name)
		if suggestion := closestName(name, arguments.accepted); suggestion != "" {
			problem += fmt.Sprintf("; did you mean %q?", suggestion)
		}
		arguments.problems = append(arguments.problems, problem)
	}
	return arguments
}

func (a *mcpArguments) isAccepted(name string) bool {
	for _, candidate := range a.accepted {
		if candidate == name {
			return true
		}
	}
	return false
}

func (a *mcpArguments) reject(name string, value any, expectation string) {
	a.problems = append(a.problems, fmt.Sprintf(
		"%q must be %s, received %s", name, expectation, describeValue(value),
	))
}

// String reads a string parameter, accepting a number or boolean written as one.
func (a *mcpArguments) String(name string) string {
	value, present := a.values[name]
	if !present || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case bool:
		return strconv.FormatBool(typed)
	default:
		a.reject(name, value, "a string")
		return ""
	}
}

func (a *mcpArguments) RequiredString(name string) string {
	value := a.String(name)
	if value == "" {
		if _, present := a.values[name]; !present {
			a.problems = append(a.problems, fmt.Sprintf("%q is required", name))
		} else {
			a.reject(name, a.values[name], "a non-empty string")
		}
	}
	return value
}

// Int reads an integer, accepting the string and whole-float forms a model
// produces, and holds it to the range the schema advertises.
func (a *mcpArguments) Int(name string, fallback, minimum, maximum int) int {
	value, present := a.values[name]
	if !present || value == nil {
		return fallback
	}
	expectation := fmt.Sprintf("an integer between %d and %d", minimum, maximum)
	var number int
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			// A whole number written as 50.0 is still fifty.
			asFloat, floatErr := typed.Float64()
			if floatErr != nil || asFloat != float64(int64(asFloat)) {
				a.reject(name, value, expectation)
				return fallback
			}
			parsed = int64(asFloat)
		}
		number = int(parsed)
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return fallback
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			a.reject(name, value, expectation)
			return fallback
		}
		number = parsed
	default:
		a.reject(name, value, expectation)
		return fallback
	}
	if number < minimum || number > maximum {
		a.reject(name, value, expectation)
		return fallback
	}
	return number
}

// Bool reads a boolean, accepting the string and 0/1 forms a model produces.
func (a *mcpArguments) Bool(name string, fallback bool) bool {
	value, present := a.values[name]
	if !present || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(strings.ToLower(typed)))
		if err != nil {
			a.reject(name, value, "true or false")
			return fallback
		}
		return parsed
	case json.Number:
		switch typed.String() {
		case "1":
			return true
		case "0":
			return false
		}
	}
	a.reject(name, value, "true or false")
	return fallback
}

// Err reports every problem at once, with the accepted parameters, so one reply
// is enough for the caller to correct the call instead of discovering the
// mistakes one round trip at a time.
func (a *mcpArguments) Err() error {
	if len(a.problems) == 0 {
		return nil
	}
	message := a.tool + ": " + strings.Join(a.problems, "; ")
	if len(a.accepted) > 0 {
		message += ". Accepted parameters: " + strings.Join(a.accepted, ", ")
	}
	return fmt.Errorf("%s", message)
}

// describeValue renders what actually arrived, with its JSON type, because "must
// be an integer" alone does not tell the caller what it sent.
func describeValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		if len(typed) > 40 {
			typed = typed[:40] + "…"
		}
		return strconv.Quote(typed) + " (string)"
	case bool:
		return strconv.FormatBool(typed) + " (boolean)"
	case json.Number:
		return typed.String() + " (number)"
	case []any:
		return fmt.Sprintf("an array of %d item(s)", len(typed))
	case map[string]any:
		return "an object"
	default:
		return fmt.Sprintf("%v", typed)
	}
}

// closestName finds the accepted parameter a misspelling most likely meant, so
// the reply can suggest it. Edit distance of two or less, which covers a
// transposition, a dropped letter and a doubled one.
func closestName(name string, accepted []string) string {
	best := ""
	bestDistance := 3
	lowered := strings.ToLower(name)
	for _, candidate := range accepted {
		distance := editDistance(lowered, strings.ToLower(candidate))
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}

func editDistance(left, right string) int {
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}
	for i := 1; i <= len(left); i++ {
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			current[j] = minimumOf(current[j-1]+1, previous[j]+1, previous[j-1]+cost)
		}
		copy(previous, current)
	}
	return previous[len(right)]
}

func minimumOf(values ...int) int {
	smallest := values[0]
	for _, value := range values[1:] {
		if value < smallest {
			smallest = value
		}
	}
	return smallest
}
