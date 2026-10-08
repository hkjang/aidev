package mcp

import (
	"strings"
	"testing"
)

// An agent reading over MCP is the same reader as the person at the canvas,
// arriving through a different door. Everything umm learned to say about a line
// that was decided against has to be said here too, or the agent acts on the
// option its owner already rejected.
func TestNoteToolsAdvertiseLinesOfThinking(t *testing.T) {
	descriptions := map[string]string{}
	for _, tool := range toolDefinitions() {
		name, _ := tool["name"].(string)
		description, _ := tool["description"].(string)
		descriptions[name] = description
	}

	for _, name := range []string{"list_notes", "search_notes"} {
		description, ok := descriptions[name]
		if !ok {
			t.Fatalf("%s is missing from the tool list", name)
		}
		if !strings.Contains(description, "note_lines") {
			t.Errorf("%s does not tell an agent that note_lines exists: %q", name, description)
		}
		if !strings.Contains(description, "abandoned") {
			t.Errorf("%s does not say what an abandoned line means: %q", name, description)
		}
	}

	lines, ok := descriptions["list_lines"]
	if !ok {
		t.Fatal("list_lines is missing from the tool list")
	}
	// The same honesty the contradiction and question tools carry: marked, not
	// detected, and an empty result means nothing is marked.
	for _, phrase := range []string{"never inferred", "empty result"} {
		if !strings.Contains(lines, phrase) {
			t.Errorf("list_lines does not say %q: %q", phrase, lines)
		}
	}
}
