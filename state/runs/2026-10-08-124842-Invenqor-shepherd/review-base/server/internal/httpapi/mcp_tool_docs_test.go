package httpapi

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// The guide's tool table is what an integrator reads before writing a client,
// and it had already drifted: `software_inventory` accepts a `vendor` filter
// that the table never mentioned, so a client built from the documentation was
// missing a filter the server has always offered. This makes the table and the
// declared schemas one inventory, the way TestOpenAPIRouteCoverage does for the
// router and openapi.yaml.
func TestMCPToolTableMatchesDeclaredSchemas(t *testing.T) {
	documented := readMCPToolTable(t)
	if len(documented) != len(mcpTools) {
		t.Fatalf(
			"the guide documents %d tools, the server offers %d",
			len(documented), len(mcpTools),
		)
	}
	for _, tool := range mcpTools {
		inputs, listed := documented[tool.Name]
		if !listed {
			t.Fatalf("the guide's tool table omits %q", tool.Name)
		}
		declared := make([]string, 0)
		if properties, ok := tool.InputSchema["properties"].(map[string]any); ok {
			for name := range properties {
				declared = append(declared, name)
			}
		}
		sort.Strings(declared)
		sort.Strings(inputs)
		if strings.Join(declared, ",") != strings.Join(inputs, ",") {
			t.Fatalf(
				"%s inputs: guide=%v schema=%v",
				tool.Name, inputs, declared,
			)
		}
	}
}

var mcpToolTableRow = regexp.MustCompile(
	`^\|\s*` + "`" + `([a-z_]+)` + "`" + `\s*\|[^|]*\|([^|]*)\|`,
)

// readMCPToolTable returns each documented tool and the input names its row
// lists, read from the tool table in the API/MCP guide.
func readMCPToolTable(t *testing.T) map[string][]string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve MCP tool table test path")
	}
	guidePath := filepath.Clean(filepath.Join(
		filepath.Dir(sourceFile), "..", "..", "..", "docs", "API_MCP_GUIDE.md",
	))
	raw, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatalf("open %s: %v", guidePath, err)
	}
	known := make(map[string]struct{}, len(mcpTools))
	for _, tool := range mcpTools {
		known[tool.Name] = struct{}{}
	}
	documented := make(map[string][]string)
	for _, line := range strings.Split(string(raw), "\n") {
		match := mcpToolTableRow.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		// Prose elsewhere in the guide is not a table row, and only the tool
		// table names a tool in its first cell.
		if _, isTool := known[match[1]]; !isTool {
			continue
		}
		inputs := make([]string, 0)
		for _, name := range regexp.MustCompile("`([a-z_]+)`").
			FindAllStringSubmatch(match[2], -1) {
			inputs = append(inputs, name[1])
		}
		documented[match[1]] = inputs
	}
	return documented
}
