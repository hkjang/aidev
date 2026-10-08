package diagnostics

import (
	"context"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

// The server log search is how an operator finds one enrolment failure among
// everything every Pod recorded, and the event codes it searches are
// underscored throughout. The typed text went into a LIKE unescaped, so "_"
// matched any character and the answer quietly included unrelated events.
func TestDiagnosticSearchTreatsAnUnderscoreAsItself(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	store := NewStore(runtime.DB())
	ctx := context.Background()
	for _, code := range []string{"agent_enrollment_failed", "agentXenrollment"} {
		if err := store.Record(ctx, Event{
			Level:     "error",
			Component: "agent_enrollment",
			EventCode: code,
			Message:   "recorded for the search test",
		}); err != nil {
			t.Fatal(err)
		}
	}

	items, total, _, err := store.List(ctx, Filter{
		Query: "agent_enrollment_failed",
		Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 ||
		items[0].EventCode != "agent_enrollment_failed" {
		t.Fatalf("searching for an event code found total=%d items=%+v",
			total, items)
	}

	// A percent sign is text an operator can reasonably paste out of a message;
	// as a pattern it returned the whole log.
	if _, everything, _, err := store.List(ctx, Filter{
		Query: "%",
		Limit: 10,
	}); err != nil {
		t.Fatal(err)
	} else if everything != 0 {
		t.Fatalf("a search for a percent sign matched %d events", everything)
	}
}
