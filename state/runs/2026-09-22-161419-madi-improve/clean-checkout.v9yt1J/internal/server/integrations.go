package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

//go:embed integrations_schema.sql
var integrationSchema string

func (s *Server) migrateIntegrations(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, integrationSchema)
	return err
}

func (s *Server) registerIntegrations() {
	s.handle("GET /api/v1/keys", s.listKeys)
	s.handle("POST /api/v1/keys", s.createKey)
	s.handle("PUT /api/v1/keys/{id}", s.updateKey)
	s.handle("DELETE /api/v1/keys/{id}", s.revokeKey)
	s.handle("POST /api/v1/keys/{id}/rotate", s.rotateKey)
	s.mux.HandleFunc("GET /api/v1/auth/oidc/start", s.oidcStart)
	s.mux.HandleFunc("GET /api/v1/auth/oidc/callback", s.oidcCallback)
	s.handle("POST /api/v1/ai/chat", s.aiChat)
	s.handle("GET /api/v1/ai/actions", s.listAIActions)
	s.handle("POST /api/v1/mcp", s.mcp)
	s.handle("GET /api/v1/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "POST")
		apiError(w, http.StatusMethodNotAllowed, "madi MCP는 요청별 JSON 응답을 제공합니다. POST를 사용하세요.")
	})
	s.handle("POST /mcp", s.mcp)
	s.handle("GET /mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "POST")
		apiError(w, http.StatusMethodNotAllowed, "POST를 사용하세요.")
	})
}

func settingString(settings map[string]any, name string) string {
	v, _ := settings[name].(string)
	return strings.TrimSpace(v)
}

func settingBool(settings map[string]any, name string) bool {
	v, _ := settings[name].(bool)
	return v
}

func settingInt(settings map[string]any, name string, fallback int) int {
	switch v := settings[name].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		i, err := strconv.Atoi(string(v))
		if err == nil {
			return i
		}
	}
	return fallback
}

func settingStrings(settings map[string]any, name string, fallback []string) []string {
	if _, exists := settings[name]; !exists {
		return fallback
	}
	switch v := settings[name].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if value, ok := item.(string); ok {
				out = append(out, value)
			}
		}
		return out
	}
	return []string{} // Invalid policy types must fail closed, never grant default permissions.
}

var keyScopes = []string{"document:read", "document:write", "database:read", "database:write", "search:read", "ai:execute"}

// Provisioning is deliberately absent from default/personal/plugin scopes.
const scimProvisionScope = "identity:provision"

func hasIntegrationScope(p *Principal, scope string) bool {
	if p == nil {
		return false
	}
	if p.TokenID == "" && !p.ScopeRestricted {
		return true
	}
	for _, s := range p.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Scope checking is separate from resource ACL checks: both must permit a request.
func integrationScopeAllowed(p *Principal, r *http.Request) bool {
	if p == nil {
		return false
	}
	if p.TokenID == "" && !p.ScopeRestricted {
		return true
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	write := r.Method != http.MethodGet && r.Method != http.MethodHead
	switch {
	case strings.HasPrefix(path, "/scim/v2/"):
		return p.TokenID != "" && p.Kind == "service" && !p.ScopeRestricted && hasIntegrationScope(p, scimProvisionScope)
	case path == "/mcp":
		return true // Each tool is checked and then dispatched through the REST auth stack.
	case path == "/knowledge-paths":
		return r.Method == http.MethodGet && hasIntegrationScope(p, "document:read")
	case strings.HasPrefix(path, "/knowledge-paths/"):
		parts := strings.Split(strings.Trim(path, "/"), "/")
		return len(parts) == 2 && validID(parts[1]) && r.Method == http.MethodGet && hasIntegrationScope(p, "document:read")
	case strings.HasPrefix(path, "/documents/") && (strings.HasSuffix(path, "/queries") || strings.HasSuffix(path, "/queries/execute") || strings.HasSuffix(path, "/queries/check")):
		return hasIntegrationScope(p, "document:read") && (r.Method == http.MethodGet || r.Method == http.MethodPost)
	case strings.HasPrefix(path, "/documents/") && (strings.HasSuffix(path, "/system-status") || strings.HasSuffix(path, "/system-status/reports")):
		return hasIntegrationScope(p, "document:read") && (!write || hasIntegrationScope(p, "document:write"))
	case strings.HasPrefix(path, "/attachment-extractions/"):
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/ai") {
			return hasIntegrationScope(p, "document:read") && hasIntegrationScope(p, "ai:execute")
		}
		return hasIntegrationScope(p, "document:read") && (!write || r.Method == http.MethodDelete && hasIntegrationScope(p, "document:write"))
	case strings.HasPrefix(path, "/attachments/") && strings.HasSuffix(path, "/extractions"):
		return r.Method == http.MethodPost && hasIntegrationScope(p, "document:read") && hasIntegrationScope(p, "document:write")
	case strings.HasPrefix(path, "/documents/") && strings.HasSuffix(path, "/ai-selection"):
		return hasIntegrationScope(p, "ai:execute") && hasIntegrationScope(p, "document:read")
	case path == "/knowledge/packages" || strings.HasPrefix(path, "/knowledge/packages/"):
		return hasIntegrationScope(p, "document:read")
	case path == "/knowledge/impact-check":
		return hasIntegrationScope(p, "document:read")
	case path == "/knowledge/time-search":
		return !write && hasIntegrationScope(p, "document:read")
	case path == "/knowledge/time-check":
		return hasIntegrationScope(p, "document:read")
	case path == "/knowledge/proposals" || strings.HasPrefix(path, "/knowledge/proposals/"):
		return hasIntegrationScope(p, "document:read") && (!write || hasIntegrationScope(p, "document:write"))
	case path == "/knowledge/questions" || strings.HasPrefix(path, "/knowledge/questions/"):
		return hasIntegrationScope(p, "document:read") && (!write || hasIntegrationScope(p, "document:write"))
	case path == "/knowledge/structured-drafts" || strings.HasPrefix(path, "/knowledge/structured-drafts/"):
		return !write && hasIntegrationScope(p, "document:read") && hasIntegrationScope(p, "database:read")
	case strings.HasPrefix(path, "/documents/") && (strings.HasSuffix(path, "/structured-context") || strings.HasSuffix(path, "/structured-draft")):
		return false
	case path == "/knowledge/conflicts" || strings.HasPrefix(path, "/knowledge/conflicts/") || strings.HasPrefix(path, "/knowledge/conflict-candidates/"):
		return !write && hasIntegrationScope(p, "document:read")
	case strings.HasPrefix(path, "/knowledge/impact-exceptions/"):
		return r.Method == http.MethodGet && hasIntegrationScope(p, "document:read")
	case path == "/knowledge/impact-reviews" || strings.HasPrefix(path, "/knowledge/impact-reviews/"):
		return hasIntegrationScope(p, "document:read") && (!write || hasIntegrationScope(p, "document:write"))
	case strings.HasPrefix(path, "/ai/"):
		return hasIntegrationScope(p, "ai:execute")
	case strings.HasPrefix(path, "/workspaces/") && strings.HasSuffix(path, "/agents"):
		return !write && hasIntegrationScope(p, "ai:execute")
	case strings.HasPrefix(path, "/agents/"):
		return strings.HasSuffix(path, "/runs") && (r.Method == http.MethodGet || r.Method == http.MethodPost) && hasIntegrationScope(p, "ai:execute")
	case strings.HasPrefix(path, "/agent-runs/"):
		return !strings.HasSuffix(path, "/confirm") && hasIntegrationScope(p, "ai:execute")
	case strings.Contains(path, "/graph-ai/"):
		return !strings.HasSuffix(path, "/confirm") && hasIntegrationScope(p, "ai:execute") && hasIntegrationScope(p, "document:read")
	case path == "/templates" || strings.HasPrefix(path, "/templates/"):
		if write {
			return hasIntegrationScope(p, "document:write")
		}
		return hasIntegrationScope(p, "document:read")
	case path == "/auth/me":
		return !write
	case path == "/search":
		return !write && (hasIntegrationScope(p, "search:read") || hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "database:read"))
	case path == "/exports" || strings.HasPrefix(path, "/exports/"):
		return hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "database:read")
	case path == "/migrations" || strings.HasPrefix(path, "/migrations/"):
		return hasIntegrationScope(p, "document:write")
	case path == "/search/index-status" || path == "/search/reindex":
		return hasIntegrationScope(p, "search:read") || hasIntegrationScope(p, "document:read")
	case path == "/knowledge/health":
		return !write && hasIntegrationScope(p, "document:read")
	case path == "/enterprise/search":
		return !write && (hasIntegrationScope(p, "search:read") || hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "database:read"))
	case path == "/enterprise/entities" || strings.HasPrefix(path, "/enterprise/entities/"):
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			return hasIntegrationScope(p, "document:write")
		}
		return !write && hasIntegrationScope(p, "document:read")
	case strings.HasPrefix(path, "/approvals/"):
		return !write && (hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "database:read"))
	case path == "/workspaces":
		return !write && (hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "database:read") || hasIntegrationScope(p, "search:read") || hasIntegrationScope(p, "ai:execute"))
	case strings.HasPrefix(path, "/spaces"):
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 3 && parts[0] == "spaces" && parts[2] == "documents" {
			return !write && hasIntegrationScope(p, "document:read")
		}
		return !write && (hasIntegrationScope(p, "document:read") || hasIntegrationScope(p, "database:read"))
	case strings.HasPrefix(path, "/canvases"), path == "/captures", strings.HasPrefix(path, "/captures/"):
		if write {
			return hasIntegrationScope(p, "document:write")
		}
		return hasIntegrationScope(p, "document:read")
	case strings.HasPrefix(path, "/databases"):
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/query") {
			return hasIntegrationScope(p, "database:read")
		}
		if write {
			return hasIntegrationScope(p, "database:write")
		}
		return hasIntegrationScope(p, "database:read")
	case path == "/data-sources" || strings.HasPrefix(path, "/data-sources/"):
		parts := strings.Split(strings.Trim(path, "/"), "/")
		queryExecute := r.Method == http.MethodPost && len(parts) == 5 && parts[0] == "data-sources" && validID(parts[1]) && parts[2] == "queries" && validID(parts[3]) && parts[4] == "execute"
		return (!write || queryExecute) && hasIntegrationScope(p, "database:read")
	case strings.HasPrefix(path, "/documents"), strings.HasPrefix(path, "/attachments"), path == "/graph", path == "/tasks", strings.HasPrefix(path, "/tasks/"), path == "/export", path == "/import":
		if write {
			return hasIntegrationScope(p, "document:write")
		}
		return hasIntegrationScope(p, "document:read")
	default:
		return false
	}
}
