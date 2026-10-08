package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hkjang/umm/internal/auth"
)

// specPathPattern matches a path key in the OpenAPI document. Paths are the only
// two-space indented keys under `paths:` that start with a slash, so a full YAML
// parser — and the dependency it would add — is not needed to find them.
var specPathPattern = regexp.MustCompile(`(?m)^  (/\S*):\s*$`)

// specMethodPattern matches the HTTP verbs declared under a path.
var specMethodPattern = regexp.MustCompile(`(?m)^    (get|post|put|patch|delete):\s*$`)

// chiParamPattern rewrites chi's {noteID} into the OpenAPI {noteId} convention.
var chiParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

// documentedOperations reads the operations the specification promises.
func documentedOperations(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read OpenAPI document: %v", err)
	}
	document := string(raw)
	pathMatches := specPathPattern.FindAllStringSubmatchIndex(document, -1)
	operations := map[string]bool{}
	for index, match := range pathMatches {
		path := document[match[2]:match[3]]
		end := len(document)
		if index+1 < len(pathMatches) {
			end = pathMatches[index+1][0]
		}
		for _, method := range specMethodPattern.FindAllStringSubmatch(document[match[1]:end], -1) {
			operations[strings.ToUpper(method[1])+" "+path] = true
		}
	}
	return operations
}

// routedOperations walks the live router, which is the only description of the
// API that cannot drift from the running service.
func routedOperations(t *testing.T) map[string]bool {
	t.Helper()
	server := &Server{Auth: &auth.Service{}, OIDC: &auth.OIDCService{}}
	operations := map[string]bool{}
	err := chi.Walk(server.router(), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = strings.TrimSuffix(route, "/*")
		if !strings.HasPrefix(route, "/api/v1/") {
			return nil
		}
		route = strings.TrimPrefix(route, "/api/v1")
		// chi names its parameters spaceID; the document uses the OpenAPI
		// convention spaceId. Only that suffix differs.
		route = chiParamPattern.ReplaceAllStringFunc(route, func(param string) string {
			name := strings.Trim(param, "{}")
			return "{" + strings.TrimSuffix(name, "ID") + strings.Repeat("Id", boolToInt(strings.HasSuffix(name, "ID"))) + "}"
		})
		operations[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	return operations
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// TestOpenAPIMatchesRoutes keeps docs/openapi.yaml honest. A documented endpoint
// that no longer exists misleads every integrator, and a new endpoint that never
// reaches the document is invisible to them.
func TestOpenAPIMatchesRoutes(t *testing.T) {
	documented := documentedOperations(t)
	routed := routedOperations(t)
	if len(documented) == 0 || len(routed) == 0 {
		t.Fatalf("expected operations on both sides, got %d documented and %d routed", len(documented), len(routed))
	}
	var missing, extra []string
	for _, operation := range sortedKeys(routed) {
		if !documented[operation] {
			missing = append(missing, operation)
		}
	}
	for _, operation := range sortedKeys(documented) {
		if !routed[operation] {
			extra = append(extra, operation)
		}
	}
	if len(missing) > 0 {
		t.Errorf("routes missing from docs/openapi.yaml:\n  %s", strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("documented operations that no longer exist:\n  %s", strings.Join(extra, "\n  "))
	}
}
