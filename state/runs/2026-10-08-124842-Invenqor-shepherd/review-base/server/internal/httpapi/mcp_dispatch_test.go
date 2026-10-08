package httpapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// Every tool offered must be one the dispatch can actually run.
//
// The dispatch is a switch on the tool name. A tool added to mcpTools without a
// case would have fallen through with no result and no error, and been reported
// as a successful empty answer - which a model reads as "there is no such
// asset" and repeats to whoever asked. An inventory question answered
// confidently and wrongly is worse than one that fails.
//
// There is a default now, so the runtime says so. This says it at build time,
// because the tool list is what a caller is shown and being shown a tool that
// cannot run is a defect on its own.
func TestEveryOfferedToolCanBeDispatched(t *testing.T) {
	set := token.NewFileSet()
	parsed, err := parser.ParseFile(set, "mcp.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	dispatched := map[string]bool{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, value := range clause.List {
			literal, isLiteral := value.(*ast.BasicLit)
			if !isLiteral || literal.Kind != token.STRING {
				continue
			}
			if name, unquoteErr := strconv.Unquote(literal.Value); unquoteErr == nil {
				dispatched[name] = true
			}
		}
		return true
	})

	if len(mcpTools) == 0 {
		t.Fatal("no tools are declared, so this test proves nothing")
	}
	var missing []string
	for _, tool := range mcpTools {
		if !dispatched[tool.Name] {
			missing = append(missing, tool.Name)
		}
	}
	if len(missing) > 0 {
		t.Errorf(
			"these tools are offered but have no case in the dispatch:\n  %s\n\n"+
				"A caller is shown them and gets an error, or worse an empty "+
				"answer, when they are called.",
			strings.Join(missing, "\n  "),
		)
	}
}

// Every tool must declare a scope, since the dispatch refuses a call whose
// principal lacks it - and an empty scope is one every principal holds.
func TestEveryOfferedToolDeclaresAScope(t *testing.T) {
	for _, tool := range mcpTools {
		if strings.TrimSpace(tool.Scope) == "" {
			t.Errorf("%s declares no scope, so every API key may call it", tool.Name)
		}
		if !strings.HasSuffix(tool.Scope, ".read") {
			t.Errorf(
				"%s declares %q; the server tells a model these tools are "+
					"read-only, so a tool that is not must not be offered here",
				tool.Name, tool.Scope,
			)
		}
	}
}
