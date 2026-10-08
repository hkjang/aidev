package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"net/netip"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/diagnostics"
	"github.com/hkjang/invenqor/server/internal/version"
)

// enrollmentVerdict is the single decision both /v1/agent/enroll and
// /v1/agent/preflight rely on, so a preflight answer can never disagree with
// what a real enrollment attempt would do.
type enrollmentVerdict struct {
	Allowed        bool
	Code           string
	Message        string
	Status         int
	NetworkAllowed bool
	TokenPresented bool
}

func evaluateEnrollment(
	policy agentEnrollmentPolicy,
	sourceIP netip.Addr,
	presentedToken string,
) enrollmentVerdict {
	presentedToken = strings.TrimSpace(presentedToken)
	verdict := enrollmentVerdict{
		NetworkAllowed: policy.NetworkMode != "allowlist" ||
			networkRulesContain(policy.AllowedNetworks, sourceIP),
		TokenPresented: presentedToken != "",
	}
	switch {
	case !policy.Enabled:
		verdict.Code = "AGENT_AUTO_ENROLLMENT_DISABLED"
		verdict.Message = "Automatic agent enrollment is not configured."
		verdict.Status = http.StatusForbidden
	case !verdict.NetworkAllowed:
		verdict.Code = "AGENT_SOURCE_NOT_ALLOWED"
		verdict.Message = "The Agent source IP is not permitted by the enrollment policy."
		verdict.Status = http.StatusForbidden
	case policy.RequireToken && !enrollmentTokenMatches(policy, presentedToken):
		verdict.Code = "AGENT_ENROLLMENT_UNAUTHORIZED"
		verdict.Message = "The fleet enrollment credential is invalid."
		verdict.Status = http.StatusUnauthorized
	default:
		verdict.Allowed = true
		verdict.Code = "AGENT_ENROLLMENT_READY"
		verdict.Message = "The Agent may enroll from this source."
		verdict.Status = http.StatusOK
	}
	return verdict
}

func enrollmentTokenMatches(
	policy agentEnrollmentPolicy,
	presentedToken string,
) bool {
	provided := sha256.Sum256([]byte(presentedToken))
	expected, err := hex.DecodeString(policy.TokenHash)
	if err != nil || len(expected) != sha256.Size {
		return false
	}
	return subtle.ConstantTimeCompare(provided[:], expected) == 1
}

// agentPreflight lets an Agent, or an operator holding nothing but curl,
// discover exactly why registration would fail before any state is created.
// It is deliberately readable without credentials because every value it
// reports is already observable from the enrollment endpoint's error codes.
func (s *Server) agentPreflight(
	response http.ResponseWriter,
	request *http.Request,
) {
	requestID := middleware.GetReqID(request.Context())
	policy, _, err := s.loadAgentEnrollmentPolicy(request.Context())
	if err != nil {
		s.recordDiagnostic(request, diagnostics.Event{
			Level:     "error",
			Component: "agent_preflight",
			EventCode: "AGENT_ENROLLMENT_POLICY_UNAVAILABLE",
			Message:   enrollmentSummary("AGENT_ENROLLMENT_POLICY_UNAVAILABLE"),
			Details:   map[string]any{"error": err.Error()},
		})
		writeAPIError(
			response, request, http.StatusServiceUnavailable,
			"AGENT_ENROLLMENT_POLICY_UNAVAILABLE",
			"The automatic enrollment policy is temporarily unavailable.",
		)
		return
	}
	sourceIP, err := resolveEnrollmentSourceIP(request, policy.TrustedProxies)
	if err != nil {
		s.recordDiagnostic(request, diagnostics.Event{
			Level:     "warning",
			Component: "agent_preflight",
			EventCode: "INVALID_AGENT_SOURCE_ADDRESS",
			Message:   enrollmentSummary("INVALID_AGENT_SOURCE_ADDRESS"),
			Details: map[string]any{
				"remote_address": request.RemoteAddr,
				"error":          err.Error(),
			},
		})
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT_SOURCE_ADDRESS",
			"The Agent source address could not be verified.",
		)
		return
	}
	if !s.agentEnrollmentRateLimit.Allow(sourceIP.String()) {
		response.Header().Set("Retry-After", "60")
		writeAPIError(
			response, request, http.StatusTooManyRequests,
			"AGENT_ENROLLMENT_RATE_LIMITED",
			"Too many agent enrollment attempts.",
		)
		return
	}
	verdict := evaluateEnrollment(
		policy,
		sourceIP,
		request.Header.Get("X-Invenqor-Enrollment-Token"),
	)
	credential := s.inspectAgentCredential(request)
	payload := map[string]any{
		"request_id":         requestID,
		"server_version":     version.Version,
		"instance_id":        s.diagnosticStore.InstanceID(),
		"database_mode":      s.database.Mode(),
		"observed_source_ip": sourceIP.String(),
		"enrollment": map[string]any{
			"enabled":         policy.Enabled,
			"mode":            policy.mode(),
			"token_required":  policy.RequireToken,
			"token_presented": verdict.TokenPresented,
			"network_mode":    policy.NetworkMode,
			"network_allowed": verdict.NetworkAllowed,
			"policy_version":  policy.Version,
			"would_enroll":    verdict.Allowed,
			"reason":          verdict.Code,
			"detail":          verdict.Message,
		},
		"credential": credential,
		"endpoints": map[string]any{
			"enroll":    "/v1/agent/enroll",
			"events":    "/v1/agent/events",
			"preflight": "/v1/agent/preflight",
		},
	}
	level := "info"
	code := "AGENT_PREFLIGHT_READY"
	if !verdict.Allowed || credential["state"] == "invalid" ||
		credential["state"] == "blocked" {
		level = "warning"
		code = "AGENT_PREFLIGHT_BLOCKED"
	}
	// The sentence for each code lives in enrollmentGuidance, so the log and
	// the enrollment panel cannot say different things about the same failure.
	message := enrollmentSummary(code)
	s.recordDiagnostic(request, diagnostics.Event{
		Level:     level,
		Component: "agent_preflight",
		EventCode: code,
		Message:   message,
		AgentID: strings.TrimSpace(
			request.Header.Get("X-Invenqor-Agent-Id"),
		),
		SourceIP: sourceIP.String(),
		Details: map[string]any{
			"would_enroll":     verdict.Allowed,
			"reason":           verdict.Code,
			"credential_state": credential["state"],
			"network_mode":     policy.NetworkMode,
			"enrollment_mode":  policy.mode(),
			"policy_version":   policy.Version,
			"user_agent":       agentVersion(request.UserAgent()),
		},
	})
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, payload)
}

// inspectAgentCredential reports whether a device credential still works
// without consuming it or storing an event, which is what makes it safe to
// call from a troubleshooting loop.
func (s *Server) inspectAgentCredential(
	request *http.Request,
) map[string]any {
	presented := bearerToken(request) != "" ||
		(request.TLS != nil && len(request.TLS.PeerCertificates) > 0)
	result := map[string]any{
		"presented": presented,
		"state":     "absent",
	}
	if !presented {
		return result
	}
	agent, err := s.authenticateAgent(request)
	switch {
	case err == nil:
		result["state"] = "valid"
		result["agent_id"] = agent.AgentID
		result["hostname"] = agent.Hostname
		result["auth_method"] = agent.AuthMethod
		result["status"] = agent.Status
	case errors.Is(err, agents.ErrBlocked):
		result["state"] = "blocked"
	case errors.Is(err, agents.ErrUnauthorized):
		result["state"] = "invalid"
	default:
		result["state"] = "unavailable"
		result["detail"] = "The server could not verify the credential."
	}
	return result
}
