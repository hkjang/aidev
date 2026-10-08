package httpapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/diagnostics"
	"github.com/hkjang/invenqor/server/internal/ingest"
	"github.com/hkjang/invenqor/server/internal/storage"
)

const maxAgentEventBytes = 16 * 1024 * 1024

func (s *Server) autoEnrollAgent(
	response http.ResponseWriter,
	request *http.Request,
) {
	policy, _, err := s.loadAgentEnrollmentPolicy(request.Context())
	if err != nil {
		s.recordDiagnostic(request, diagnostics.Event{
			Level:     "error",
			Component: "agent_enrollment",
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
			Component: "agent_enrollment",
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
	clientKey := sourceIP.String()
	recordEnrollment := func(
		level string,
		code string,
		message string,
		agentID string,
		details map[string]any,
	) {
		if details == nil {
			details = map[string]any{}
		}
		details["policy_version"] = policy.Version
		details["network_mode"] = policy.NetworkMode
		details["enrollment_mode"] = policy.mode()
		details["agent_version"] = agentVersion(request.UserAgent())
		if level != "info" {
			details["remediation"] = enrollmentRemediation(code)
		}
		// An empty message means "whatever this code means", which is the table
		// the enrollment panel reads. Repeating the sentence at the call site is
		// how the two drifted apart.
		if message == "" {
			if summary := enrollmentSummary(code); summary != "" {
				message = summary
			} else {
				message = code
			}
		}
		s.recordDiagnostic(request, diagnostics.Event{
			Level:     level,
			Component: "agent_enrollment",
			EventCode: code,
			Message:   message,
			AgentID:   agentID,
			SourceIP:  clientKey,
			Details:   details,
		})
	}
	if !s.agentEnrollmentRateLimit.Allow(clientKey) {
		recordEnrollment(
			"warning",
			"AGENT_ENROLLMENT_RATE_LIMITED",
			"",
			"",
			nil,
		)
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
	if !verdict.Allowed {
		recordEnrollment(
			"warning",
			verdict.Code,
			enrollmentRejectionMessage(verdict.Code),
			"",
			map[string]any{"token_presented": verdict.TokenPresented},
		)
		writeAPIError(
			response, request, verdict.Status,
			verdict.Code, verdict.Message,
		)
		return
	}
	var input struct {
		AgentID    string `json:"agent_id"`
		Hostname   string `json:"hostname"`
		ClaimToken string `json:"claim_token"`
	}
	if err := decodeJSON(request, &input); err != nil {
		recordEnrollment(
			"warning",
			"INVALID_AGENT_REQUEST",
			"",
			"",
			map[string]any{"error": err.Error()},
		)
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_REQUEST", "The request body is invalid.",
		)
		return
	}
	if len(strings.TrimSpace(input.Hostname)) > 255 {
		recordEnrollment(
			"warning",
			"INVALID_AGENT_HOSTNAME",
			"",
			input.AgentID,
			nil,
		)
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT", "hostname must not exceed 255 characters.",
		)
		return
	}
	result, err := s.agentService.AutoEnroll(
		request.Context(),
		input.AgentID,
		input.Hostname,
		input.ClaimToken,
		sourceIP.String(),
	)
	switch {
	case err == nil:
		recordEnrollment(
			"info",
			"AGENT_ENROLLMENT_SUCCEEDED",
			"",
			input.AgentID,
			map[string]any{
				"hostname":         strings.TrimSpace(input.Hostname),
				"asset_state":      "discovered",
				"effective_source": clientKey,
			},
		)
		response.Header().Set("Cache-Control", "no-store")
		writeJSON(response, http.StatusCreated, result)
	case errors.Is(err, agents.ErrEnrollmentClaimMismatch):
		recordEnrollment(
			"warning",
			"AGENT_ALREADY_CLAIMED",
			"",
			input.AgentID,
			nil,
		)
		writeAPIError(
			response, request, http.StatusConflict,
			"AGENT_ALREADY_CLAIMED",
			"The agent identifier is already bound to another device claim.",
		)
	case errors.Is(err, agents.ErrBlocked):
		recordEnrollment(
			"warning",
			"AGENT_BLOCKED",
			"The blocked Agent attempted enrollment.",
			input.AgentID,
			nil,
		)
		writeAPIError(
			response, request, http.StatusForbidden,
			"AGENT_BLOCKED", "The agent is blocked.",
		)
	case errors.Is(err, agents.ErrInvalidEnrollment):
		recordEnrollment(
			"warning",
			"INVALID_AGENT_IDENTITY",
			"",
			input.AgentID,
			map[string]any{"error": err.Error()},
		)
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT", "The agent enrollment identity is invalid.",
		)
	default:
		recordEnrollment(
			"error",
			"AGENT_ENROLLMENT_FAILED",
			"",
			input.AgentID,
			map[string]any{"error": err.Error()},
		)
		s.internalError(response, request, err)
	}
}

func resolveEnrollmentSourceIP(
	request *http.Request,
	trustedProxies []string,
) (netip.Addr, error) {
	peer, err := parseRemoteAddress(request.RemoteAddr)
	if err != nil {
		return netip.Addr{}, err
	}
	if !networkRulesContain(trustedProxies, peer) {
		return peer, nil
	}
	forwarded := strings.TrimSpace(request.Header.Get("X-Forwarded-For"))
	if forwarded == "" {
		realIP := strings.TrimSpace(request.Header.Get("X-Real-IP"))
		if realIP == "" {
			return peer, nil
		}
		address, err := netip.ParseAddr(realIP)
		if err != nil {
			return netip.Addr{}, err
		}
		return address.Unmap(), nil
	}
	parts := strings.Split(forwarded, ",")
	chain := make([]netip.Addr, 0, len(parts))
	for _, part := range parts {
		address, err := netip.ParseAddr(strings.TrimSpace(part))
		if err != nil {
			return netip.Addr{}, err
		}
		chain = append(chain, address.Unmap())
	}
	for index := len(chain) - 1; index >= 0; index-- {
		if !networkRulesContain(trustedProxies, chain[index]) {
			return chain[index], nil
		}
	}
	return chain[0], nil
}

func parseRemoteAddress(value string) (netip.Addr, error) {
	if addressPort, err := netip.ParseAddrPort(value); err == nil {
		return addressPort.Addr().Unmap(), nil
	}
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		value = host
	}
	address, err := netip.ParseAddr(strings.Trim(value, "[]"))
	if err != nil {
		return netip.Addr{}, err
	}
	return address.Unmap(), nil
}

func (s *Server) receiveAgentEvent(
	response http.ResponseWriter,
	request *http.Request,
) {
	diagnosticAgentID := strings.TrimSpace(
		request.Header.Get("X-Invenqor-Agent-Id"),
	)
	diagnosticSourceIP := s.agentDiagnosticSourceIP(request)
	recordEventFailure := func(code string, message string, err error) {
		details := map[string]any{
			"path":          request.URL.Path,
			"agent_version": agentVersion(request.UserAgent()),
			"remediation":   enrollmentRemediation(code),
		}
		if err != nil {
			details["error"] = err.Error()
		}
		s.recordDiagnostic(request, diagnostics.Event{
			Level:     "warning",
			Component: "agent_transport",
			EventCode: code,
			Message:   message,
			AgentID:   diagnosticAgentID,
			SourceIP:  diagnosticSourceIP,
			Details:   details,
		})
	}
	agent, err := s.authenticateAgent(request)
	switch {
	case errors.Is(err, agents.ErrUnauthorized):
		recordEventFailure(
			"AGENT_UNAUTHORIZED",
			"The Agent device credential was rejected.",
			nil,
		)
		writeAPIError(
			response, request, http.StatusUnauthorized,
			"AGENT_UNAUTHORIZED", "The agent credential is invalid.",
		)
		return
	case errors.Is(err, agents.ErrBlocked):
		recordEventFailure(
			"AGENT_BLOCKED",
			"A blocked Agent attempted to send inventory.",
			nil,
		)
		writeAPIError(
			response, request, http.StatusForbidden,
			"AGENT_BLOCKED", "The agent is blocked.",
		)
		return
	case err != nil:
		recordEventFailure(
			"AGENT_AUTHENTICATION_FAILED",
			"The server could not authenticate the Agent.",
			err,
		)
		context, cancel := context.WithTimeout(request.Context(), time.Second)
		defer cancel()
		if pingErr := s.database.Ping(context); pingErr != nil {
			writeAPIError(
				response, request, http.StatusServiceUnavailable,
				"AGENT_AUTH_UNAVAILABLE",
				"Agent authentication is temporarily unavailable.",
			)
			return
		}
		s.internalError(response, request, err)
		return
	}
	headerAgentID := strings.TrimSpace(
		request.Header.Get("X-Invenqor-Agent-Id"),
	)
	headerEventID := strings.TrimSpace(
		request.Header.Get("X-Invenqor-Event-Id"),
	)
	if headerAgentID == "" || headerEventID == "" {
		recordEventFailure(
			"MISSING_EVENT_IDENTITY",
			"The Agent request omitted identity headers.",
			nil,
		)
		writeAPIError(
			response, request, http.StatusBadRequest,
			"MISSING_EVENT_IDENTITY",
			"X-Invenqor-Agent-Id and X-Invenqor-Event-Id are required.",
		)
		return
	}
	if headerAgentID != agent.AgentID {
		recordEventFailure(
			"AGENT_ID_MISMATCH",
			"The authenticated Agent did not match the identity header.",
			nil,
		)
		writeAPIError(
			response, request, http.StatusForbidden,
			"AGENT_ID_MISMATCH",
			"The authenticated agent does not match the request header.",
		)
		return
	}
	if !s.agentRateLimit.Allow(agent.ID) {
		recordEventFailure(
			"AGENT_RATE_LIMITED",
			"The Agent event rate limit was exceeded.",
			nil,
		)
		response.Header().Set("Retry-After", "60")
		writeAPIError(
			response, request, http.StatusTooManyRequests,
			"AGENT_RATE_LIMITED", "The agent request rate limit was exceeded.",
		)
		return
	}
	request.Body = http.MaxBytesReader(
		response, request.Body, maxAgentEventBytes,
	)
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		code := "INVALID_EVENT"
		message := "The event body could not be read."
		if strings.Contains(err.Error(), "request body too large") {
			code = "EVENT_TOO_LARGE"
			message = "The event exceeds the maximum allowed size."
		}
		recordEventFailure(code, message, err)
		writeAPIError(response, request, http.StatusBadRequest, code, message)
		return
	}
	envelope, err := ingest.DecodeEnvelope(raw)
	if err != nil {
		s.logger.Info(
			"agent_event_rejected",
			"request_id", middleware.GetReqID(request.Context()),
			"agent_id", agent.AgentID,
			"reason", err.Error(),
		)
		recordEventFailure(
			"INVALID_EVENT",
			"The Agent event payload failed schema validation.",
			err,
		)
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_EVENT", err.Error(),
		)
		return
	}
	if envelope.AgentID != headerAgentID ||
		envelope.EventID != headerEventID {
		recordEventFailure(
			"EVENT_IDENTITY_MISMATCH",
			"The Agent event headers did not match its body.",
			nil,
		)
		writeAPIError(
			response, request, http.StatusBadRequest,
			"EVENT_IDENTITY_MISMATCH",
			"Event identity headers do not match the body.",
		)
		return
	}
	if s.database.Mode() == storage.ModePostgresDegraded {
		s.spoolAgentEvent(response, request, envelope, raw)
		return
	}
	result, err := s.ingestService.Process(
		request.Context(),
		agent,
		envelope,
		raw,
		agentVersion(request.UserAgent()),
	)
	switch {
	case err == nil:
		writeJSON(response, http.StatusOK, map[string]any{
			"accepted":       true,
			"duplicate":      result.Duplicate,
			"policy_version": result.PolicyVersion,
		})
	case errors.Is(err, ingest.ErrInProgress):
		recordEventFailure(
			"EVENT_IN_PROGRESS",
			"The Agent event is already being processed by a Server Pod.",
			err,
		)
		response.Header().Set("Retry-After", "1")
		writeAPIError(
			response, request, http.StatusConflict,
			"EVENT_IN_PROGRESS", "The event is already being processed.",
		)
	default:
		recordEventFailure(
			"AGENT_EVENT_PROCESSING_FAILED",
			"The server failed while processing the Agent event.",
			err,
		)
		context, cancel := context.WithTimeout(request.Context(), time.Second)
		pingErr := s.database.Ping(context)
		cancel()
		if pingErr != nil {
			s.spoolAgentEvent(response, request, envelope, raw)
			return
		}
		s.internalError(response, request, err)
	}
}

func (s *Server) agentDiagnosticSourceIP(request *http.Request) string {
	policy, _, err := s.loadAgentEnrollmentPolicy(request.Context())
	if err == nil {
		if address, resolveErr := resolveEnrollmentSourceIP(
			request,
			policy.TrustedProxies,
		); resolveErr == nil {
			return address.String()
		}
	}
	return clientIP(request)
}

func (s *Server) spoolAgentEvent(
	response http.ResponseWriter,
	request *http.Request,
	envelope ingest.Envelope,
	raw []byte,
) {
	if s.spool == nil {
		writeAPIError(
			response, request, http.StatusServiceUnavailable,
			"EVENT_SPOOL_UNAVAILABLE",
			"The durable event spool is not configured.",
		)
		return
	}
	duplicate, err := s.spool.Append(
		envelope.AgentID, envelope.EventID, raw,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusAccepted, map[string]any{
		"accepted":  true,
		"spooled":   true,
		"duplicate": duplicate,
	})
}

func (s *Server) authenticateAgent(
	request *http.Request,
) (agents.Agent, error) {
	if token := bearerToken(request); token != "" {
		return s.agentService.AuthenticateBearer(request.Context(), token)
	}
	if request.TLS == nil || len(request.TLS.PeerCertificates) == 0 {
		return agents.Agent{}, agents.ErrUnauthorized
	}
	sum := sha256.Sum256(request.TLS.PeerCertificates[0].Raw)
	return s.agentService.AuthenticateCertificate(
		request.Context(), hex.EncodeToString(sum[:]),
	)
}

func (s *Server) provisionAgent(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		AgentID  string `json:"agent_id"`
		Hostname string `json:"hostname"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, 400, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	result, err := s.agentService.ProvisionBearer(
		request.Context(),
		input.AgentID,
		input.Hostname,
		principalFromContext(request.Context()).User.ID,
	)
	if err != nil {
		writeAPIError(response, request, 400, "INVALID_AGENT", err.Error())
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusCreated, result)
}

func (s *Server) rotateAgentToken(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		GraceSeconds int64 `json:"grace_seconds"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, 400, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	if input.GraceSeconds < 0 || input.GraceSeconds > 7*24*60*60 {
		writeAPIError(response, request, 400, "INVALID_GRACE_PERIOD", "Grace period must be between zero and seven days.")
		return
	}
	token, err := s.agentService.RotateBearer(
		request.Context(),
		chi.URLParam(request, "agentID"),
		time.Duration(input.GraceSeconds)*time.Second,
		principalFromContext(request.Context()).User.ID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(response, request, 404, "AGENT_NOT_FOUND", "The agent does not exist.")
		return
	}
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusOK, map[string]any{"token": token})
}

func (s *Server) blockAgent(
	response http.ResponseWriter,
	request *http.Request,
) {
	s.setAgentBlocked(response, request, true)
}

func (s *Server) unblockAgent(
	response http.ResponseWriter,
	request *http.Request,
) {
	s.setAgentBlocked(response, request, false)
}

func (s *Server) setAgentBlocked(
	response http.ResponseWriter,
	request *http.Request,
	blocked bool,
) {
	err := s.agentService.SetBlocked(
		request.Context(),
		chi.URLParam(request, "agentID"),
		blocked,
		principalFromContext(request.Context()).User.ID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(response, request, 404, "AGENT_NOT_FOUND", "The agent does not exist.")
		return
	}
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"blocked": blocked})
}

func (s *Server) registerAgentCertificate(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Fingerprint string `json:"fingerprint"`
		ExpiresAt   string `json:"expires_at"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, 400, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	var expiresAt *time.Time
	if input.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.ExpiresAt)
		if err != nil {
			writeAPIError(response, request, 400, "INVALID_EXPIRY", "expires_at must be RFC 3339.")
			return
		}
		expiresAt = &parsed
	}
	err := s.agentService.RegisterCertificate(
		request.Context(),
		chi.URLParam(request, "agentID"),
		input.Fingerprint,
		expiresAt,
		principalFromContext(request.Context()).User.ID,
	)
	if err != nil {
		writeAPIError(response, request, 400, "INVALID_CERTIFICATE", err.Error())
		return
	}
	writeJSON(response, http.StatusCreated, map[string]any{"registered": true})
}

func (s *Server) listAgents(
	response http.ResponseWriter,
	request *http.Request,
) {
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT id, agent_id, hostname, status, version, os_name,
		 architecture, auth_method, policy_version, last_seen_at,
		 last_inventory_at
		 FROM agents ORDER BY hostname, agent_id`,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer rows.Close()
	result := make([]agents.Agent, 0)
	for rows.Next() {
		var agent agents.Agent
		if err := rows.Scan(
			&agent.ID, &agent.AgentID, &agent.Hostname, &agent.Status,
			&agent.Version, &agent.OSName, &agent.Architecture,
			&agent.AuthMethod, &agent.PolicyVersion, &agent.LastSeenAt,
			&agent.LastInventoryAt,
		); err != nil {
			s.internalError(response, request, err)
			return
		}
		result = append(result, agent)
	}
	writeJSON(response, http.StatusOK, map[string]any{"agents": result})
}

func agentVersion(userAgent string) string {
	const prefix = "invenqor-agent/"
	for _, item := range strings.Fields(userAgent) {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}
	return ""
}

type rateWindow struct {
	start time.Time
	count int
}

type agentRateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	// entries only ever grew. The enrollment limiter is keyed by source address
	// on endpoints that need no credentials, so every address that ever reached
	// the Server kept a record in a long-running process - and one host holding
	// an IPv6 /64 can spend addresses faster than an operator can notice the
	// memory going. A counter is meaningless once its window has passed, so the
	// expired ones are swept out rather than kept for an address that may never
	// return.
	entries map[string]rateWindow
	sweptAt time.Time
	// now is a field so the sweep can be exercised without a test sleeping
	// through a real window.
	now func() time.Time
}

func newAgentRateLimiter(limit int, window time.Duration) *agentRateLimiter {
	return &agentRateLimiter{
		limit: limit, window: window, entries: make(map[string]rateWindow),
		now: time.Now,
	}
}

func (l *agentRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	entry := l.entries[key]
	if entry.start.IsZero() || now.Sub(entry.start) >= l.window {
		l.entries[key] = rateWindow{start: now, count: 1}
		return true
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

// sweep drops the windows that have already passed. It runs at most once per
// window so that a burst of distinct keys pays one pass over the map between
// them rather than one pass each.
func (l *agentRateLimiter) sweep(now time.Time) {
	if !l.sweptAt.IsZero() && now.Sub(l.sweptAt) < l.window {
		return
	}
	l.sweptAt = now
	for key, entry := range l.entries {
		if now.Sub(entry.start) >= l.window {
			delete(l.entries, key)
		}
	}
}

// Marshal helper is intentionally unused by handlers but gives integration
// callers one canonical representation for replaying an accepted event.
func CanonicalAgentEvent(envelope ingest.Envelope) ([]byte, error) {
	bytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal agent event: %w", err)
	}
	return bytes, nil
}
