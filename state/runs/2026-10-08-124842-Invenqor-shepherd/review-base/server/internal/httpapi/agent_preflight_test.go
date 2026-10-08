package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/google/uuid"
)

// getWithCookie issues an authenticated GET, for responses that are not JSON.
func getWithCookie(
	t *testing.T,
	server *Server,
	path string,
	cookie *http.Cookie,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func getJSON(
	t *testing.T,
	server *Server,
	path string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func decodePreflight(
	t *testing.T,
	response *httptest.ResponseRecorder,
) map[string]any {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("preflight status = %d body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode preflight: %v", err)
	}
	return payload
}

func TestPreflightReportsAnOpenPolicyAndTheObservedSourceAddress(t *testing.T) {
	server := testServer(t, newRuntime(t))
	payload := decodePreflight(t, getJSON(t, server, "/v1/agent/preflight", nil))

	if payload["observed_source_ip"] != "192.0.2.1" {
		t.Fatalf("observed_source_ip = %v", payload["observed_source_ip"])
	}
	enrollment := payload["enrollment"].(map[string]any)
	if enrollment["would_enroll"] != true {
		t.Fatalf("would_enroll = %v", enrollment["would_enroll"])
	}
	if enrollment["reason"] != "AGENT_ENROLLMENT_READY" {
		t.Fatalf("reason = %v", enrollment["reason"])
	}
	credential := payload["credential"].(map[string]any)
	if credential["state"] != "absent" {
		t.Fatalf("credential state = %v", credential["state"])
	}
	if payload["request_id"] == "" {
		t.Fatal("preflight omitted the request identifier used for correlation")
	}
}

func TestPreflightAgreesWithARejectedEnrollment(t *testing.T) {
	server := testServer(t, newRuntime(t))
	if _, _, err := server.updateAgentEnrollmentPolicy(
		context.Background(),
		"test",
		func(policy *agentEnrollmentPolicy) error {
			policy.Enabled = true
			policy.NetworkMode = "allowlist"
			policy.AllowedNetworks = []string{"10.10.0.0/16"}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}

	payload := decodePreflight(t, getJSON(t, server, "/v1/agent/preflight", nil))
	enrollment := payload["enrollment"].(map[string]any)
	if enrollment["would_enroll"] != false {
		t.Fatalf("would_enroll = %v", enrollment["would_enroll"])
	}
	if enrollment["reason"] != "AGENT_SOURCE_NOT_ALLOWED" {
		t.Fatalf("reason = %v", enrollment["reason"])
	}
	if enrollment["network_allowed"] != false {
		t.Fatalf("network_allowed = %v", enrollment["network_allowed"])
	}

	// The real endpoint must reject the same request for the same reason,
	// otherwise a preflight "OK" would send operators looking elsewhere.
	enroll := performJSON(
		t, server, http.MethodPost, "/v1/agent/enroll",
		map[string]string{
			"agent_id":    uuid.NewString(),
			"hostname":    "rejected-host",
			"claim_token": "ivq_ec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		nil,
	)
	if enroll.Code != http.StatusForbidden {
		t.Fatalf("enrollment status = %d body = %s", enroll.Code, enroll.Body.String())
	}
	var rejection struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(enroll.Body.Bytes(), &rejection); err != nil {
		t.Fatal(err)
	}
	if rejection.Error.Code != enrollment["reason"] {
		t.Fatalf(
			"enrollment code %q disagrees with preflight reason %v",
			rejection.Error.Code,
			enrollment["reason"],
		)
	}
}

func TestPreflightValidatesAStoredDeviceCredential(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	enroll := performJSON(
		t, server, http.MethodPost, "/v1/agent/enroll",
		map[string]string{
			"agent_id":    uuid.NewString(),
			"hostname":    "credentialed-host",
			"claim_token": "ivq_ec_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		nil,
	)
	if enroll.Code != http.StatusCreated {
		t.Fatalf("enrollment status = %d body = %s", enroll.Code, enroll.Body.String())
	}
	var enrolled struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(enroll.Body.Bytes(), &enrolled); err != nil {
		t.Fatal(err)
	}

	valid := decodePreflight(t, getJSON(t, server, "/v1/agent/preflight", map[string]string{
		"Authorization": "Bearer " + enrolled.Token,
	}))
	credential := valid["credential"].(map[string]any)
	if credential["state"] != "valid" {
		t.Fatalf("credential state = %v", credential["state"])
	}
	if credential["hostname"] != "credentialed-host" {
		t.Fatalf("credential hostname = %v", credential["hostname"])
	}

	invalid := decodePreflight(t, getJSON(t, server, "/v1/agent/preflight", map[string]string{
		"Authorization": "Bearer ivq_at_not-a-real-credential",
	}))
	if state := invalid["credential"].(map[string]any)["state"]; state != "invalid" {
		t.Fatalf("rejected credential state = %v", state)
	}
}

func TestAgentPathMistakesAnswerJSONAndAreRecorded(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)

	// A base URL with a stray path prefix must not receive the console SPA.
	strayPath := performJSON(
		t, server, http.MethodPost, "/invenqor/v1/agent/enroll",
		map[string]string{"agent_id": uuid.NewString()}, nil,
	)
	if strayPath.Code != http.StatusNotFound {
		t.Fatalf("stray path status = %d body = %s", strayPath.Code, strayPath.Body.String())
	}
	if got := strayPath.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("stray path content type = %q", got)
	}
	wrongMethod := getJSON(t, server, "/v1/agent/enroll", nil)
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d", wrongMethod.Code)
	}

	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	logs := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/admin/diagnostics/logs?component=agent_transport",
		nil, cookie, "",
	)
	if logs.Code != http.StatusOK {
		t.Fatalf("diagnostics status = %d body = %s", logs.Code, logs.Body.String())
	}
	body := logs.Body.String()
	for _, code := range []string{
		"AGENT_ENDPOINT_NOT_FOUND",
		"AGENT_ENDPOINT_METHOD_NOT_ALLOWED",
	} {
		if !strings.Contains(body, code) {
			t.Fatalf("diagnostics omitted %s: %s", code, body)
		}
	}
	// The remedy has to travel with the event; the code alone is not actionable.
	// Compared against the table rather than a copy of its wording, so this
	// keeps checking that the remedy travels rather than what it happens to say.
	remedy := enrollmentRemediation("AGENT_ENDPOINT_NOT_FOUND")
	if remedy == "" {
		t.Fatal("AGENT_ENDPOINT_NOT_FOUND has no remediation, so this proves nothing")
	}
	if !strings.Contains(body, remedy) {
		t.Fatalf("diagnostics omitted the remediation text: %s", body)
	}
}
