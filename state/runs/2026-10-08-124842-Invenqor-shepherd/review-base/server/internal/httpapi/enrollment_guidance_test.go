package httpapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// A recorder must not repeat a sentence the guidance table already holds.
//
// Every code carried its summary twice: once in enrollmentGuidance, which the
// enrollment panel reads, and once at the site that records the event, which
// the diagnostic log shows. Twelve matched and three had drifted, so the same
// failure read one way in one screen and another way in the other. Passing ""
// takes the table's wording; passing a sentence means this site knows something
// the table does not, which is true of exactly two - a blocked Agent enrolling
// and a blocked Agent sending inventory are different events sharing a code.
func TestRecordersDoNotRepeatTheGuidanceSummary(t *testing.T) {
	set := token.NewFileSet()
	parsed, err := parser.ParseFile(set, "agents.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	checked := 0
	ast.Inspect(parsed, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "recordEnrollment" || len(call.Args) < 3 {
			return true
		}
		code, codeOK := stringArgument(call.Args[1])
		message, messageOK := stringArgument(call.Args[2])
		if !codeOK || !messageOK || message == "" {
			return true
		}
		checked++
		if summary := enrollmentSummary(code); summary == message {
			t.Errorf(
				"agents.go:%d records %s with the sentence enrollmentGuidance "+
					"already holds; pass \"\" so the two cannot drift",
				set.Position(call.Pos()).Line, code,
			)
		}
		return true
	})

	if checked == 0 {
		t.Fatal("no recordEnrollment calls with a literal message were found, so this test proves nothing")
	}
}

func stringArgument(argument ast.Expr) (string, bool) {
	literal, ok := argument.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}
