package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"time"
)

const agentEnrollmentPolicyKey = "agent_enrollment_policy"

var errAgentEnrollmentPolicyConflict = errors.New(
	"agent enrollment policy changed concurrently",
)
var errAgentEnrollmentTokenNotConfigured = errors.New(
	"registration token is not configured",
)

type agentEnrollmentPolicy struct {
	Enabled         bool      `json:"enabled"`
	RequireToken    bool      `json:"require_token"`
	TokenHash       string    `json:"token_hash,omitempty"`
	NetworkMode     string    `json:"network_mode"`
	AllowedNetworks []string  `json:"allowed_networks"`
	TrustedProxies  []string  `json:"trusted_proxies"`
	Version         int       `json:"version"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by"`
}

func (policy agentEnrollmentPolicy) mode() string {
	if !policy.Enabled {
		return "disabled"
	}
	if policy.RequireToken {
		return "token"
	}
	return "open"
}

func (policy agentEnrollmentPolicy) publicValue() map[string]any {
	return map[string]any{
		"enabled":          policy.Enabled,
		"mode":             policy.mode(),
		"token_configured": policy.TokenHash != "",
		"network_mode":     policy.NetworkMode,
		"allowed_networks": policy.AllowedNetworks,
		"trusted_proxies":  policy.TrustedProxies,
		"version":          policy.Version,
		"updated_at":       policy.UpdatedAt,
		"updated_by":       policy.UpdatedBy,
		"source":           "database",
	}
}

func (s *Server) initialAgentEnrollmentPolicy() agentEnrollmentPolicy {
	policy := agentEnrollmentPolicy{
		Enabled:         s.agentEnrollmentEnabled,
		NetworkMode:     "any",
		AllowedNetworks: []string{},
		TrustedProxies:  []string{},
		Version:         1,
		UpdatedAt:       time.Now().UTC(),
		UpdatedBy:       "startup-environment",
	}
	if s.agentEnrollmentTokenRequired {
		policy.RequireToken = true
		policy.TokenHash = hex.EncodeToString(s.agentEnrollmentTokenHash[:])
	}
	return policy
}

func (s *Server) loadAgentEnrollmentPolicy(
	ctx context.Context,
) (agentEnrollmentPolicy, string, error) {
	var raw string
	err := s.database.DB().QueryRowContext(
		ctx,
		`SELECT value FROM server_metadata WHERE key=$1`,
		agentEnrollmentPolicyKey,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		initial := s.initialAgentEnrollmentPolicy()
		encoded, encodeErr := json.Marshal(initial)
		if encodeErr != nil {
			return agentEnrollmentPolicy{}, "", encodeErr
		}
		_, insertErr := s.database.DB().ExecContext(
			ctx,
			`INSERT INTO server_metadata(key,value,updated_at)
			 VALUES($1,$2,$3) ON CONFLICT(key) DO NOTHING`,
			agentEnrollmentPolicyKey,
			string(encoded),
			initial.UpdatedAt,
		)
		if insertErr != nil {
			return agentEnrollmentPolicy{}, "", fmt.Errorf(
				"initialize agent enrollment policy: %w",
				insertErr,
			)
		}
		err = s.database.DB().QueryRowContext(
			ctx,
			`SELECT value FROM server_metadata WHERE key=$1`,
			agentEnrollmentPolicyKey,
		).Scan(&raw)
	}
	if err != nil {
		return agentEnrollmentPolicy{}, "", fmt.Errorf(
			"load agent enrollment policy: %w",
			err,
		)
	}
	var policy agentEnrollmentPolicy
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return agentEnrollmentPolicy{}, "", fmt.Errorf(
			"decode agent enrollment policy: %w",
			err,
		)
	}
	if policy.NetworkMode == "" {
		// Policies written before network controls existed allowed every source.
		policy.NetworkMode = "any"
	}
	if policy.AllowedNetworks == nil {
		policy.AllowedNetworks = []string{}
	}
	if policy.TrustedProxies == nil {
		policy.TrustedProxies = []string{}
	}
	allowed, err := normalizeNetworkRules(policy.AllowedNetworks, 256)
	if err != nil {
		return agentEnrollmentPolicy{}, "", fmt.Errorf(
			"agent enrollment policy has invalid allowed networks: %w",
			err,
		)
	}
	proxies, err := normalizeNetworkRules(policy.TrustedProxies, 64)
	if err != nil {
		return agentEnrollmentPolicy{}, "", fmt.Errorf(
			"agent enrollment policy has invalid trusted proxies: %w",
			err,
		)
	}
	policy.AllowedNetworks = allowed
	policy.TrustedProxies = proxies
	if policy.NetworkMode != "any" && policy.NetworkMode != "allowlist" {
		return agentEnrollmentPolicy{}, "", errors.New(
			"agent enrollment policy has an invalid network mode",
		)
	}
	if policy.NetworkMode == "allowlist" && len(policy.AllowedNetworks) == 0 {
		return agentEnrollmentPolicy{}, "", errors.New(
			"agent enrollment policy allowlist is empty",
		)
	}
	if policy.Version < 1 {
		return agentEnrollmentPolicy{}, "", errors.New(
			"agent enrollment policy has an invalid version",
		)
	}
	if policy.TokenHash != "" {
		decoded, err := hex.DecodeString(policy.TokenHash)
		if err != nil || len(decoded) != sha256.Size {
			return agentEnrollmentPolicy{}, "", errors.New(
				"agent enrollment policy has an invalid token digest",
			)
		}
	}
	if policy.RequireToken && policy.TokenHash == "" {
		return agentEnrollmentPolicy{}, "", errors.New(
			"agent enrollment policy requires a missing token",
		)
	}
	return policy, raw, nil
}

func (s *Server) updateAgentEnrollmentPolicy(
	ctx context.Context,
	updatedBy string,
	mutate func(*agentEnrollmentPolicy) error,
) (agentEnrollmentPolicy, agentEnrollmentPolicy, error) {
	for range 5 {
		before, raw, err := s.loadAgentEnrollmentPolicy(ctx)
		if err != nil {
			return agentEnrollmentPolicy{}, agentEnrollmentPolicy{}, err
		}
		after := before
		if err := mutate(&after); err != nil {
			return before, agentEnrollmentPolicy{}, err
		}
		after.Version = before.Version + 1
		after.UpdatedAt = time.Now().UTC()
		after.UpdatedBy = updatedBy
		encoded, err := json.Marshal(after)
		if err != nil {
			return before, agentEnrollmentPolicy{}, err
		}
		result, err := s.database.DB().ExecContext(
			ctx,
			`UPDATE server_metadata SET value=$1,updated_at=$2
			 WHERE key=$3 AND value=$4`,
			string(encoded),
			after.UpdatedAt,
			agentEnrollmentPolicyKey,
			raw,
		)
		if err != nil {
			return before, agentEnrollmentPolicy{}, fmt.Errorf(
				"save agent enrollment policy: %w",
				err,
			)
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return before, agentEnrollmentPolicy{}, err
		}
		if changed == 1 {
			return before, after, nil
		}
	}
	return agentEnrollmentPolicy{}, agentEnrollmentPolicy{},
		errAgentEnrollmentPolicyConflict
}

func (s *Server) getAgentEnrollmentSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	policy, _, err := s.loadAgentEnrollmentPolicy(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, policy.publicValue())
}

func (s *Server) updateAgentEnrollmentSettings(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Mode            string    `json:"mode"`
		NetworkMode     *string   `json:"network_mode"`
		AllowedNetworks *[]string `json:"allowed_networks"`
		TrustedProxies  *[]string `json:"trusted_proxies"`
		Reason          string    `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT_ENROLLMENT_POLICY",
			"The request body is invalid.",
		)
		return
	}
	input.Mode = strings.ToLower(strings.TrimSpace(input.Mode))
	if input.Mode != "disabled" && input.Mode != "open" &&
		input.Mode != "token" {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT_ENROLLMENT_MODE",
			"mode must be disabled, open, or token.",
		)
		return
	}
	principal := principalFromContext(request.Context())
	before, after, err := s.updateAgentEnrollmentPolicy(
		request.Context(),
		principal.User.ID,
		func(policy *agentEnrollmentPolicy) error {
			switch input.Mode {
			case "disabled":
				policy.Enabled = false
			case "open":
				policy.Enabled = true
				policy.RequireToken = false
			case "token":
				if policy.TokenHash == "" {
					return errAgentEnrollmentTokenNotConfigured
				}
				policy.Enabled = true
				policy.RequireToken = true
			}
			if input.NetworkMode != nil {
				policy.NetworkMode = strings.ToLower(strings.TrimSpace(
					*input.NetworkMode,
				))
			}
			if input.AllowedNetworks != nil {
				normalized, err := normalizeNetworkRules(
					*input.AllowedNetworks,
					256,
				)
				if err != nil {
					return fmt.Errorf("allowed_networks: %w", err)
				}
				policy.AllowedNetworks = normalized
			}
			if input.TrustedProxies != nil {
				normalized, err := normalizeNetworkRules(
					*input.TrustedProxies,
					64,
				)
				if err != nil {
					return fmt.Errorf("trusted_proxies: %w", err)
				}
				policy.TrustedProxies = normalized
			}
			if policy.NetworkMode != "any" &&
				policy.NetworkMode != "allowlist" {
				return errors.New(
					"network_mode must be any or allowlist",
				)
			}
			if policy.NetworkMode == "allowlist" &&
				len(policy.AllowedNetworks) == 0 {
				return errors.New(
					"allowed_networks must contain at least one IP or CIDR",
				)
			}
			return nil
		},
	)
	if err != nil {
		if errors.Is(err, errAgentEnrollmentTokenNotConfigured) {
			writeAPIError(
				response, request, http.StatusConflict,
				"AGENT_ENROLLMENT_TOKEN_REQUIRED",
				"Issue a registration token before enabling protected mode.",
			)
			return
		}
		if errors.Is(err, errAgentEnrollmentPolicyConflict) {
			writeAPIError(
				response, request, http.StatusConflict,
				"AGENT_ENROLLMENT_POLICY_CONFLICT",
				"The policy changed concurrently. Reload and try again.",
			)
			return
		}
		if strings.Contains(err.Error(), "network_mode") ||
			strings.Contains(err.Error(), "allowed_networks") ||
			strings.Contains(err.Error(), "trusted_proxies") {
			writeAPIError(
				response, request, http.StatusBadRequest,
				"INVALID_AGENT_ENROLLMENT_NETWORK_POLICY",
				err.Error(),
			)
			return
		}
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"agent.enrollment_policy.update",
		"agent_enrollment_policy",
		agentEnrollmentPolicyKey,
		before.publicValue(),
		after.publicValue(),
		input.Reason,
	)
	writeJSON(response, http.StatusOK, after.publicValue())
}

func normalizeNetworkRules(values []string, limit int) ([]string, error) {
	if len(values) > limit {
		return nil, fmt.Errorf("at most %d entries are allowed", limit)
	}
	unique := make(map[string]struct{}, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, "/") {
			prefix, err := netip.ParsePrefix(value)
			if err != nil {
				return nil, fmt.Errorf(
					"entry %d (%q) is not a valid CIDR",
					index+1,
					value,
				)
			}
			if prefix.Addr().Is4In6() {
				prefix = netip.PrefixFrom(
					prefix.Addr().Unmap(),
					prefix.Bits()-96,
				)
			}
			unique[prefix.Masked().String()] = struct{}{}
			continue
		}
		address, err := netip.ParseAddr(value)
		if err != nil {
			return nil, fmt.Errorf(
				"entry %d (%q) is not a valid IP address",
				index+1,
				value,
			)
		}
		unique[address.Unmap().String()] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for value := range unique {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func networkRulesContain(rules []string, address netip.Addr) bool {
	address = address.Unmap()
	for _, rule := range rules {
		if strings.Contains(rule, "/") {
			prefix, err := netip.ParsePrefix(rule)
			if err == nil && prefix.Contains(address) {
				return true
			}
			continue
		}
		candidate, err := netip.ParseAddr(rule)
		if err == nil && candidate.Unmap() == address {
			return true
		}
	}
	return false
}

func (s *Server) issueAgentEnrollmentToken(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Reason string `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT_ENROLLMENT_TOKEN_REQUEST",
			"The request body is invalid.",
		)
		return
	}
	token, err := newAgentEnrollmentToken()
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	sum := sha256.Sum256([]byte(token))
	principal := principalFromContext(request.Context())
	before, after, err := s.updateAgentEnrollmentPolicy(
		request.Context(),
		principal.User.ID,
		func(policy *agentEnrollmentPolicy) error {
			policy.Enabled = true
			policy.RequireToken = true
			policy.TokenHash = hex.EncodeToString(sum[:])
			return nil
		},
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"agent.enrollment_token.issue",
		"agent_enrollment_policy",
		agentEnrollmentPolicyKey,
		before.publicValue(),
		after.publicValue(),
		input.Reason,
	)
	payload := after.publicValue()
	payload["registration_token"] = token
	payload["shown_once"] = true
	response.Header().Set("Cache-Control", "no-store")
	writeJSON(response, http.StatusCreated, payload)
}

func (s *Server) deleteAgentEnrollmentToken(
	response http.ResponseWriter,
	request *http.Request,
) {
	var input struct {
		Reason string `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(
			response, request, http.StatusBadRequest,
			"INVALID_AGENT_ENROLLMENT_TOKEN_REQUEST",
			"The request body is invalid.",
		)
		return
	}
	principal := principalFromContext(request.Context())
	before, after, err := s.updateAgentEnrollmentPolicy(
		request.Context(),
		principal.User.ID,
		func(policy *agentEnrollmentPolicy) error {
			policy.TokenHash = ""
			policy.RequireToken = false
			return nil
		},
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"agent.enrollment_token.delete",
		"agent_enrollment_policy",
		agentEnrollmentPolicyKey,
		before.publicValue(),
		after.publicValue(),
		input.Reason,
	)
	writeJSON(response, http.StatusOK, after.publicValue())
}

func newAgentEnrollmentToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate registration token: %w", err)
	}
	return "ivq_et_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}
