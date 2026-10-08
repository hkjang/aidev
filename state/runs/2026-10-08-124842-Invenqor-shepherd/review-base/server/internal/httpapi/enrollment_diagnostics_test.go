package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestEnrollmentDiagnosticsExplainsWhyAnAgentIsMissing(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)

	succeeded := uuid.NewString()
	enroll := performJSON(
		t, server, http.MethodPost, "/v1/agent/enroll",
		map[string]string{
			"agent_id":    succeeded,
			"hostname":    "enrolled-host",
			"claim_token": "ivq_ec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		nil,
	)
	if enroll.Code != http.StatusCreated {
		t.Fatalf("enrollment status = %d body = %s", enroll.Code, enroll.Body.String())
	}

	// Now close the policy so the next attempt is rejected the way a real
	// misconfigured network would be.
	if _, _, err := server.updateAgentEnrollmentPolicy(
		context.Background(),
		"test",
		func(policy *agentEnrollmentPolicy) error {
			policy.NetworkMode = "allowlist"
			policy.AllowedNetworks = []string{"10.10.0.0/16"}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	rejected := performJSON(
		t, server, http.MethodPost, "/v1/agent/enroll",
		map[string]string{
			"agent_id":    uuid.NewString(),
			"hostname":    "blocked-host",
			"claim_token": "ivq_ec_abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		},
		nil,
	)
	if rejected.Code != http.StatusForbidden {
		t.Fatalf("rejected status = %d", rejected.Code)
	}

	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	response := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/diagnostics/enrollment?hours=24",
		nil, cookie, "",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("summary status = %d body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		WindowHours int `json:"window_hours"`
		Totals      struct {
			Succeeded int `json:"succeeded"`
			Rejected  int `json:"rejected"`
		} `json:"totals"`
		ByEventCode []struct {
			EventCode   string `json:"event_code"`
			Count       int    `json:"count"`
			Remediation string `json:"remediation"`
		} `json:"by_event_code"`
		Sources []struct {
			SourceIP      string `json:"source_ip"`
			Attempts      int    `json:"attempts"`
			Failures      int    `json:"failures"`
			LastEventCode string `json:"last_event_code"`
			Remediation   string `json:"remediation"`
		} `json:"sources"`
		AwaitingInventory []struct {
			AgentID  string `json:"agent_id"`
			Hostname string `json:"hostname"`
		} `json:"awaiting_inventory"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.WindowHours != 24 {
		t.Fatalf("window_hours = %d", payload.WindowHours)
	}
	if payload.Totals.Succeeded != 1 || payload.Totals.Rejected != 1 {
		t.Fatalf("totals = %+v", payload.Totals)
	}
	rejection := false
	for _, code := range payload.ByEventCode {
		if code.EventCode == "AGENT_SOURCE_NOT_ALLOWED" {
			rejection = true
			if code.Remediation == "" {
				t.Fatal("rejection summary carried no remediation")
			}
		}
	}
	if !rejection {
		t.Fatalf("by_event_code omitted the rejection: %s", response.Body.String())
	}
	// Failing sources sort first so the operator sees them without filtering.
	if len(payload.Sources) == 0 || payload.Sources[0].Failures == 0 {
		t.Fatalf("sources = %+v", payload.Sources)
	}
	if payload.Sources[0].SourceIP != "192.0.2.1" {
		t.Fatalf("source ip = %q", payload.Sources[0].SourceIP)
	}
	if payload.Sources[0].LastEventCode != "AGENT_SOURCE_NOT_ALLOWED" {
		t.Fatalf("last event code = %q", payload.Sources[0].LastEventCode)
	}
	// The Agent enrolled but never delivered inventory, which is the state an
	// operator most often mistakes for "the Agent never registered".
	found := false
	for _, agent := range payload.AwaitingInventory {
		if agent.AgentID == succeeded && agent.Hostname == "enrolled-host" {
			found = true
		}
	}
	if !found {
		t.Fatalf("awaiting_inventory omitted the enrolled agent: %+v", payload.AwaitingInventory)
	}
}

func TestEnrollmentGuidanceCoversEveryRejectionCodeTheServerEmits(t *testing.T) {
	// Any code that reaches an operator without a remedy is a dead end.
	emitted := []string{
		"AGENT_AUTO_ENROLLMENT_DISABLED",
		"AGENT_SOURCE_NOT_ALLOWED",
		"AGENT_ENROLLMENT_UNAUTHORIZED",
		"AGENT_ENROLLMENT_RATE_LIMITED",
		"AGENT_ALREADY_CLAIMED",
		"AGENT_BLOCKED",
		"INVALID_AGENT_IDENTITY",
		"INVALID_AGENT_SOURCE_ADDRESS",
		"INVALID_AGENT_HOSTNAME",
		"INVALID_AGENT_REQUEST",
		"AGENT_ENROLLMENT_POLICY_UNAVAILABLE",
		"AGENT_ENROLLMENT_FAILED",
		"AGENT_ENDPOINT_NOT_FOUND",
		"AGENT_ENDPOINT_METHOD_NOT_ALLOWED",
		"AGENT_UNAUTHORIZED",
	}
	for _, code := range emitted {
		if _, found := enrollmentGuidance[code]; !found {
			t.Errorf("no operator guidance for %s", code)
		}
	}
}
