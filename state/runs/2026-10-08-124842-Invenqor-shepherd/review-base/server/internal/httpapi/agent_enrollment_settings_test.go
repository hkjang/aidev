package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
)

func TestEnrollmentNetworkPolicyAllowsCIDRAndRejectsSpoofedForwarding(
	t *testing.T,
) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	otherPod := testServer(t, runtime)

	saved := performAuthenticatedJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/admin/settings/agent-enrollment",
		map[string]any{
			"mode":             "open",
			"network_mode":     "allowlist",
			"allowed_networks": []string{"203.0.113.0/24", "10.20.30.40"},
			"trusted_proxies":  []string{"10.0.0.0/8"},
			"reason":           "managed enrollment networks",
		},
		cookie,
		csrf,
	)
	if saved.Code != http.StatusOK {
		t.Fatalf("save status = %d body = %s", saved.Code, saved.Body.String())
	}

	allowedID := uuid.NewString()
	allowed := enrollTestAgentFrom(
		t,
		otherPod,
		"10.1.2.3:42310",
		"203.0.113.44",
		allowedID,
	)
	if allowed.Code != http.StatusCreated {
		t.Fatalf(
			"trusted proxy enrollment status = %d body = %s",
			allowed.Code,
			allowed.Body.String(),
		)
	}
	var identifier, status string
	if err := runtime.DB().QueryRow(
		`SELECT i.identifier_value,a.status
		 FROM assets a
		 JOIN asset_identifiers i ON i.asset_id=a.id
		 WHERE a.asset_key=$1`,
		"agent:"+allowedID,
	).Scan(&identifier, &status); err != nil {
		t.Fatal(err)
	}
	if identifier != "203.0.113.44" || status != "discovered" {
		t.Fatalf("immediate asset identifier/status = %q/%q", identifier, status)
	}

	spoofed := enrollTestAgentFrom(
		t,
		otherPod,
		"198.51.100.10:42310",
		"203.0.113.45",
		uuid.NewString(),
	)
	if spoofed.Code != http.StatusForbidden ||
		!strings.Contains(spoofed.Body.String(), "AGENT_SOURCE_NOT_ALLOWED") {
		t.Fatalf(
			"spoofed forwarding status = %d body = %s",
			spoofed.Code,
			spoofed.Body.String(),
		)
	}

	exact := enrollTestAgentFrom(
		t,
		server,
		"10.20.30.40:55000",
		"",
		uuid.NewString(),
	)
	if exact.Code != http.StatusCreated {
		t.Fatalf(
			"exact IP enrollment status = %d body = %s",
			exact.Code,
			exact.Body.String(),
		)
	}
}

func TestAdminCanManageAgentEnrollmentPolicyAcrossServerInstances(
	t *testing.T,
) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	otherPod := testServer(t, runtime)

	settings := performAuthenticatedJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/admin/settings/agent-enrollment",
		map[string]any{},
		cookie,
		csrf,
	)
	if settings.Code != http.StatusOK ||
		!strings.Contains(settings.Body.String(), `"mode":"open"`) {
		t.Fatalf(
			"initial policy status = %d body = %s",
			settings.Code,
			settings.Body.String(),
		)
	}

	disabled := performAuthenticatedJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/admin/settings/agent-enrollment",
		map[string]string{"mode": "disabled", "reason": "maintenance"},
		cookie,
		csrf,
	)
	if disabled.Code != http.StatusOK {
		t.Fatalf(
			"disable status = %d body = %s",
			disabled.Code,
			disabled.Body.String(),
		)
	}
	disabledEnrollment := enrollTestAgent(t, otherPod, "", "disabled")
	if disabledEnrollment.Code != http.StatusForbidden {
		t.Fatalf(
			"other pod disabled enrollment status = %d body = %s",
			disabledEnrollment.Code,
			disabledEnrollment.Body.String(),
		)
	}

	open := performAuthenticatedJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/admin/settings/agent-enrollment",
		map[string]string{"mode": "open", "reason": "zero touch"},
		cookie,
		csrf,
	)
	if open.Code != http.StatusOK {
		t.Fatalf("open status = %d body = %s", open.Code, open.Body.String())
	}
	openEnrollment := enrollTestAgent(t, otherPod, "", "open")
	if openEnrollment.Code != http.StatusCreated {
		t.Fatalf(
			"other pod open enrollment status = %d body = %s",
			openEnrollment.Code,
			openEnrollment.Body.String(),
		)
	}

	issued := performAuthenticatedJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/admin/settings/agent-enrollment/token",
		map[string]string{"reason": "protect enrollment"},
		cookie,
		csrf,
	)
	if issued.Code != http.StatusCreated {
		t.Fatalf(
			"issue token status = %d body = %s",
			issued.Code,
			issued.Body.String(),
		)
	}
	var issuedPayload struct {
		RegistrationToken string `json:"registration_token"`
		Mode              string `json:"mode"`
		TokenConfigured   bool   `json:"token_configured"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &issuedPayload); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(issuedPayload.RegistrationToken, "ivq_et_") ||
		issuedPayload.Mode != "token" || !issuedPayload.TokenConfigured {
		t.Fatalf("unexpected token payload: %#v", issuedPayload)
	}
	var storedPolicy string
	if err := runtime.DB().QueryRow(
		`SELECT value FROM server_metadata WHERE key=$1`,
		agentEnrollmentPolicyKey,
	).Scan(&storedPolicy); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedPolicy, issuedPayload.RegistrationToken) {
		t.Fatal("registration token plaintext was stored in the database")
	}
	withoutToken := enrollTestAgent(t, otherPod, "", "protected-no-token")
	if withoutToken.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d", withoutToken.Code)
	}
	withToken := enrollTestAgent(
		t,
		otherPod,
		issuedPayload.RegistrationToken,
		"protected-token",
	)
	if withToken.Code != http.StatusCreated {
		t.Fatalf(
			"protected enrollment status = %d body = %s",
			withToken.Code,
			withToken.Body.String(),
		)
	}

	rotated := performAuthenticatedJSON(
		t,
		otherPod,
		http.MethodPost,
		"/api/v1/admin/settings/agent-enrollment/token",
		map[string]string{"reason": "scheduled rotation"},
		cookie,
		csrf,
	)
	if rotated.Code != http.StatusCreated {
		t.Fatalf(
			"rotate token status = %d body = %s",
			rotated.Code,
			rotated.Body.String(),
		)
	}
	var rotatedPayload struct {
		RegistrationToken string `json:"registration_token"`
	}
	if err := json.Unmarshal(rotated.Body.Bytes(), &rotatedPayload); err != nil {
		t.Fatal(err)
	}
	if rotatedPayload.RegistrationToken == issuedPayload.RegistrationToken {
		t.Fatal("token rotation returned the previous token")
	}
	if response := enrollTestAgent(
		t,
		server,
		issuedPayload.RegistrationToken,
		"old-token",
	); response.Code != http.StatusUnauthorized {
		t.Fatalf("old token status after rotation = %d", response.Code)
	}
	if response := enrollTestAgent(
		t,
		server,
		rotatedPayload.RegistrationToken,
		"new-token",
	); response.Code != http.StatusCreated {
		t.Fatalf("new token status after rotation = %d", response.Code)
	}

	deleted := performAuthenticatedJSON(
		t,
		server,
		http.MethodDelete,
		"/api/v1/admin/settings/agent-enrollment/token",
		map[string]string{"reason": "return to zero touch"},
		cookie,
		csrf,
	)
	if deleted.Code != http.StatusOK ||
		!strings.Contains(deleted.Body.String(), `"mode":"open"`) {
		t.Fatalf(
			"delete token status = %d body = %s",
			deleted.Code,
			deleted.Body.String(),
		)
	}
	if response := enrollTestAgent(t, otherPod, "", "open-after-delete"); response.Code != http.StatusCreated {
		t.Fatalf(
			"open enrollment after token deletion status = %d body = %s",
			response.Code,
			response.Body.String(),
		)
	}
}

func enrollTestAgent(
	t *testing.T,
	server *Server,
	registrationToken string,
	hostname string,
) *httptest.ResponseRecorder {
	t.Helper()
	headers := map[string]string{}
	if registrationToken != "" {
		headers["X-Invenqor-Enrollment-Token"] = registrationToken
	}
	return performJSON(
		t,
		server,
		http.MethodPost,
		"/v1/agent/enroll",
		map[string]string{
			"agent_id":    uuid.NewString(),
			"hostname":    hostname,
			"claim_token": "ivq_ec_" + strings.Repeat("a", 64),
		},
		headers,
	)
}

func enrollTestAgentFrom(
	t *testing.T,
	server *Server,
	remoteAddress string,
	forwardedFor string,
	agentID string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"agent_id":    agentID,
		"hostname":    "network-enrolled-host",
		"claim_token": "ivq_ec_" + strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/agent/enroll",
		bytes.NewReader(body),
	)
	request.RemoteAddr = remoteAddress
	request.Header.Set("Content-Type", "application/json")
	if forwardedFor != "" {
		request.Header.Set("X-Forwarded-For", forwardedFor)
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}
