package httpapi

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/hkjang/invenqor/server/internal/storage"
)

// TestOpenAPIRouteCoverage makes the router and its published contract one
// inventory. Adding, removing, or changing an HTTP operation now requires the
// OpenAPI description to move in the same change.
func TestOpenAPIRouteCoverage(t *testing.T) {
	database, err := storage.Open(context.Background(), storage.Options{
		SQLitePath: filepath.Join(t.TempDir(), "invenqor.db"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	server := testServer(t, database)
	registered := make(map[string]struct{})
	err = chi.Walk(server.router, func(
		method string,
		route string,
		_ http.Handler,
		_ ...func(http.Handler) http.Handler,
	) error {
		if documentedHTTPMethod(method) {
			registered[canonicalOperation(method, route)] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	documented, operationIDs := readOpenAPIOperations(t)
	missing := setDifference(registered, documented)
	extra := setDifference(documented, registered)
	if len(missing) != 0 || len(extra) != 0 {
		t.Fatalf(
			"OpenAPI route coverage differs (registered=%d documented=%d)\nmissing from OpenAPI:\n%s\nnot registered by chi:\n%s",
			len(registered),
			len(documented),
			strings.Join(missing, "\n"),
			strings.Join(extra, "\n"),
		)
	}
	if len(operationIDs) != len(documented) {
		t.Fatalf(
			"OpenAPI operationId coverage differs (operations=%d unique operationIds=%d)",
			len(documented),
			len(operationIDs),
		)
	}
	t.Logf(
		"OpenAPI route coverage complete: registered=%d documented=%d missing=0 extra=0 unique_operation_ids=%d",
		len(registered),
		len(documented),
		len(operationIDs),
	)
}

func readOpenAPIOperations(t *testing.T) (map[string]struct{}, map[string]string) {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve OpenAPI coverage test path")
	}
	specPath := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", "..", "openapi.yaml"))
	file, err := os.Open(specPath)
	if err != nil {
		t.Fatalf("open %s: %v", specPath, err)
	}
	defer file.Close()

	operations := make(map[string]struct{})
	operationIDs := make(map[string]string)
	operationForID := make(map[string]string)
	currentPath := ""
	currentOperation := ""
	inPaths := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		indent := leadingSpaces(line)
		if trimmed == "paths:" && indent == 0 {
			inPaths = true
			continue
		}
		if !inPaths {
			continue
		}
		if indent == 0 && strings.HasSuffix(trimmed, ":") {
			break
		}
		if indent == 2 && strings.HasPrefix(trimmed, "/") && strings.HasSuffix(trimmed, ":") {
			currentPath = strings.TrimSuffix(trimmed, ":")
			currentOperation = ""
			continue
		}
		if indent == 4 && strings.HasSuffix(trimmed, ":") {
			method := strings.ToUpper(strings.TrimSuffix(trimmed, ":"))
			if documentedHTTPMethod(method) {
				currentOperation = canonicalOperation(method, currentPath)
				if _, exists := operations[currentOperation]; exists {
					t.Fatalf("duplicate OpenAPI operation %s", currentOperation)
				}
				operations[currentOperation] = struct{}{}
			} else {
				currentOperation = ""
			}
			continue
		}
		if indent == 6 && strings.HasPrefix(trimmed, "operationId:") && currentOperation != "" {
			operationID := strings.TrimSpace(strings.TrimPrefix(trimmed, "operationId:"))
			if operationID == "" {
				t.Fatalf("empty operationId for %s", currentOperation)
			}
			if previous, exists := operationForID[operationID]; exists {
				t.Fatalf("duplicate operationId %q for %s and %s", operationID, previous, currentOperation)
			}
			operationForID[operationID] = currentOperation
			operationIDs[operationID] = currentOperation
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read %s: %v", specPath, err)
	}
	for operation := range operations {
		found := false
		for _, candidate := range operationIDs {
			if candidate == operation {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("OpenAPI operation %s has no operationId", operation)
		}
	}
	return operations, operationIDs
}

func documentedHTTPMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete:
		return true
	default:
		return false
	}
}

func canonicalOperation(method string, path string) string {
	var normalized strings.Builder
	for index := 0; index < len(path); {
		if path[index] != '{' {
			normalized.WriteByte(path[index])
			index++
			continue
		}
		closing := strings.IndexByte(path[index:], '}')
		if closing < 0 {
			normalized.WriteString(path[index:])
			break
		}
		normalized.WriteString("{}")
		index += closing + 1
	}
	return fmt.Sprintf("%s %s", strings.ToUpper(method), normalized.String())
}

func setDifference(left map[string]struct{}, right map[string]struct{}) []string {
	result := make([]string, 0)
	for value := range left {
		if _, exists := right[value]; !exists {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func leadingSpaces(value string) int {
	return len(value) - len(strings.TrimLeftFunc(value, unicode.IsSpace))
}
