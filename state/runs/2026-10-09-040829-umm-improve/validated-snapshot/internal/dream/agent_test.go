package dream

import (
	"strings"
	"testing"
)

// The boundary that makes this design acceptable is the tool list itself, not a
// runtime check. If a tool that changes anything ever appears here, a model can
// call it — so the list is asserted rather than assumed.
func TestAgentToolsAreReadOnly(t *testing.T) {
	writeWords := []string{"create", "update", "delete", "merge", "connect", "move", "write", "remove", "accept"}
	for _, tool := range agentTools() {
		name := strings.ToLower(tool.Function.Name)
		for _, word := range writeWords {
			if strings.Contains(name, word) {
				t.Errorf("tool %q looks like it changes something; the agent must only read", tool.Function.Name)
			}
		}
		if tool.Type != "function" {
			t.Errorf("tool %q has type %q", tool.Function.Name, tool.Type)
		}
		if len(tool.Function.Description) < 30 {
			t.Errorf("tool %q is described too thinly for a model to choose it well", tool.Function.Name)
		}
		if tool.Function.Parameters == nil {
			t.Errorf("tool %q has no parameter schema", tool.Function.Name)
		}
	}
}

// The descriptions carry the same honesty the endpoints do. A model told that an
// empty result means "nothing is marked" will not report it as "nothing exists".
func TestAgentToolsDescribeTheirLimits(t *testing.T) {
	for _, tool := range agentTools() {
		if tool.Function.Name == "search_thoughts" {
			continue
		}
		lower := strings.ToLower(tool.Function.Description)
		if !strings.Contains(lower, "marked") && !strings.Contains(lower, "recorded") {
			t.Errorf("tool %q does not tell the model these are marked rather than detected: %q",
				tool.Function.Name, tool.Function.Description)
		}
		if !strings.Contains(lower, "empty result") {
			t.Errorf("tool %q does not say what an empty result means", tool.Function.Name)
		}
	}
}

// A loop that cannot end is the failure mode of this design, and each step costs
// the person a model call against their daily quota.
func TestAgentStepBudgetIsBounded(t *testing.T) {
	if maxAgentSteps < 2 {
		t.Fatal("fewer than two steps leaves no room to look then answer")
	}
	if maxAgentSteps > 10 {
		t.Fatalf("a budget of %d model calls for one question is not a budget", maxAgentSteps)
	}
}

// An unknown tool name must produce something the model can recover from rather
// than a panic or an empty string it might treat as a real result.
func TestUnknownToolIsRefusedClearly(t *testing.T) {
	var service Service
	var call toolCall
	call.Function.Name = "delete_everything"
	output, summary, excluded := service.runAgentTool(nil, [16]byte{}, call)
	if !strings.Contains(output, "없습니다") {
		t.Errorf("an unknown tool returned %q, which a model could read as a result", output)
	}
	if summary == "" || excluded != 0 {
		t.Errorf("unknown tool summary=%q excluded=%d", summary, excluded)
	}
}
