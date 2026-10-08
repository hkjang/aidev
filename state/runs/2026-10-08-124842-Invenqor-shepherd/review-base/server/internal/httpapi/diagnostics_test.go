package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/hkjang/invenqor/server/internal/diagnostics"
)

func TestDiagnosticLogsAreSharedAcrossServerInstances(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	first := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, first, runtime)
	second := testServer(t, runtime)
	if err := first.diagnosticStore.Record(
		context.Background(),
		diagnostics.Event{
			Level:     "error",
			Component: "agent_transport",
			EventCode: "AGENT_EVENT_PROCESSING_FAILED",
			Message:   "The Agent event could not be processed.",
			RequestID: "shared-request-id",
			AgentID:   "agent-42",
			SourceIP:  "10.20.30.40",
		},
	); err != nil {
		t.Fatal(err)
	}
	response := performAuthenticatedJSON(
		t,
		second,
		http.MethodGet,
		"/api/v1/admin/diagnostics/logs?q=shared-request-id&level=error",
		nil,
		cookie,
		csrf,
	)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "shared-request-id") ||
		!strings.Contains(response.Body.String(), "agent-42") {
		t.Fatalf(
			"diagnostic status = %d body = %s",
			response.Code,
			response.Body.String(),
		)
	}
}
