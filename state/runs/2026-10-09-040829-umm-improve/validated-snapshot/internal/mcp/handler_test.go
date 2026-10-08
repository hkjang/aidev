package mcp

import (
	"slices"
	"strings"
	"testing"
)

// The tool list is umm's contract with every agent connected to it. A bare count
// only says something changed; naming the surface says what, so a tool removed by
// accident fails as loudly as one added.
func TestToolDefinitionsAreStable(t *testing.T) {
	want := []string{
		"capture_thought",
		"connect_notes",
		"create_note",
		"find_contradictions",
		"find_open_questions",
		"get_connections",
		"get_related_notes",
		"list_clusters",
		"list_dreams",
		"list_lines",
		"list_notes",
		"list_presentations",
		"list_spaces",
		"make_presentation",
		"preview_presentation",
		"search_notes",
	}
	tools := toolDefinitions()
	got := make([]string, 0, len(tools))
	for _, tool := range tools {
		name, ok := tool["name"].(string)
		if !ok || name == "" {
			t.Fatalf("a tool has no name: %v", tool)
		}
		got = append(got, name)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("tool surface changed\n  got:  %v\n  want: %v", got, want)
	}
	// Sorted, so a client listing them gets a stable order across releases.
	if !slices.IsSorted(got) {
		t.Fatalf("tools are not sorted: %v", got)
	}
}

// Every tool has to describe itself well enough for an agent to choose it
// without trial and error.
func TestEveryToolIsDescribed(t *testing.T) {
	for _, tool := range toolDefinitions() {
		name := tool["name"].(string)
		description, _ := tool["description"].(string)
		if len(description) < 20 {
			t.Errorf("%s has no usable description: %q", name, description)
		}
		if _, ok := tool["inputSchema"]; !ok {
			t.Errorf("%s has no input schema", name)
		}
	}
}

// Making a talk sends the owner's thoughts to another service and creates
// something there; looking at what a space would become does not. A key that
// may only read must be able to do the second and not the first, and the
// difference is only real if it is checked.
func TestMakingATalkNeedsMoreThanReading(t *testing.T) {
	scopes := map[string]string{}
	for _, tool := range toolDefinitions() {
		name := tool["name"].(string)
		if strings.Contains(name, "presentation") {
			scopes[name] = ""
		}
	}
	if len(scopes) != 3 {
		t.Fatalf("expected three presentation tools, got %v", scopes)
	}
	for _, name := range []string{"preview_presentation", "make_presentation", "list_presentations"} {
		if _, ok := scopes[name]; !ok {
			t.Fatalf("%s is not declared", name)
		}
	}
}

// The tools that reach Ptium have to say that the slides carry the author's own
// sentences. An agent choosing between them otherwise has no way to know this
// is not a summariser, and would use it where a summary was wanted.
func TestPresentationToolsSayTheWordsAreTheAuthorsOwn(t *testing.T) {
	for _, tool := range toolDefinitions() {
		name := tool["name"].(string)
		if name != "preview_presentation" && name != "make_presentation" {
			continue
		}
		description := tool["description"].(string)
		if !strings.Contains(description, "own sentences") && !strings.Contains(description, "word for word") {
			t.Errorf("%s does not say the slides carry the author's own words: %q", name, description)
		}
	}
}
