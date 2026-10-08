package diagnostics

import (
	"context"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

func TestStoreFiltersSharedDiagnosticsAndRedactsSecrets(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	store := NewStore(runtime.DB())
	if err := store.Record(context.Background(), Event{
		Level:     "error",
		Component: "agent_enrollment",
		EventCode: "AGENT_ENROLLMENT_FAILED",
		Message:   "failed with ivq_at_should-never-appear",
		RequestID: "request-123",
		AgentID:   "agent-123",
		SourceIP:  "10.20.30.40",
		Details: map[string]any{
			"error": "postgres://user:password@database/invenqor",
			"token": "ivq_et_should-never-appear",
		},
	}); err != nil {
		t.Fatal(err)
	}
	items, total, facets, err := store.List(context.Background(), Filter{
		Level: "error",
		Query: "request-123",
		Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || total != 1 || len(facets.Instances) != 1 {
		t.Fatalf("items/total/instances = %d/%d/%d",
			len(items), total, len(facets.Instances))
	}
	// The console builds its component filter from these, so a component the
	// server records must appear without anyone editing the console.
	if len(facets.Components) != 1 || facets.Components[0] != "agent_enrollment" {
		t.Fatalf("facets.Components = %v", facets.Components)
	}
	if len(facets.EventCodes) != 1 {
		t.Fatalf("facets.EventCodes = %v", facets.EventCodes)
	}
	if items[0].Message != "failed with [REDACTED]" ||
		items[0].Details["token"] != "[REDACTED]" ||
		items[0].Details["error"] !=
			"postgres://user:[REDACTED]@database/invenqor" {
		t.Fatalf("diagnostic was not redacted: %#v", items[0])
	}
}

func TestSanitizeRedactsAPIKeysAndCompleteBearerCredentials(t *testing.T) {
	apiKey := "ivq_sk_prefix_0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	value := "key=" + apiKey + " Authorization: Bearer opaque_token.with-parts=="
	sanitized := Sanitize(value)
	if strings.Contains(sanitized, apiKey) ||
		strings.Contains(sanitized, "opaque_token.with-parts==") {
		t.Fatalf("Sanitize() leaked a credential: %q", sanitized)
	}
	if strings.Count(sanitized, "[REDACTED]") != 2 {
		t.Fatalf("Sanitize() = %q, want both credentials redacted", sanitized)
	}
}
