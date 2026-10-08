package httpapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Every CSV column carrying text a person or an Agent chose must go through
// spreadsheetSafe.
//
// Excel and LibreOffice execute a cell whose text begins with =, +, - or @, and
// these files are served as attachments for exactly that. The reader is an
// administrator or an auditor, opening a file this product told them to
// download, with more access than whoever wrote the value.
//
// The two exports that exist were both written without it. A third would be
// written the same way, and nothing about the code would look wrong - which is
// why this reads the calls rather than trusting review.
func TestEveryCSVWriteEscapesTheColumnsAPersonCanChoose(t *testing.T) {
	// Columns whose value is a fixed vocabulary the server assigns - a status,
	// a type, a formatted number, a timestamp - cannot begin with a formula
	// character, so they are exempt.
	safeByConstruction := map[string]bool{
		"item.Type": true, "item.Status": true, "item.Environment": true,
		"item.Criticality": true, "item.Source": true,
		"record.ActorType": true, "record.Action": true, "record.Result": true,
		"record.ResourceType": true, "record.SourceIP": true,
		"record.RequestID": true, "occurred": true,
	}

	set := token.NewFileSet()
	checked := 0
	var offenders []string

	for _, name := range []string{"assets.go", "audit_search.go"} {
		parsed, err := parser.ParseFile(set, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Write" {
				return true
			}
			composite, ok := call.Args[0].(*ast.CompositeLit)
			if !ok {
				return true
			}
			for _, element := range composite.Elts {
				// A header row is a literal, and literals are ours.
				if literal, isLiteral := element.(*ast.BasicLit); isLiteral {
					_, _ = strconv.Unquote(literal.Value)
					continue
				}
				checked++
				text := expressionText(set, element)
				if strings.HasPrefix(text, "spreadsheetSafe(") ||
					safeByConstruction[text] ||
					strings.Contains(text, "FormatFloat") ||
					strings.Contains(text, ".String()") ||
					strings.Contains(text, "optionalText(") {
					continue
				}
				offenders = append(offenders, filepath.Base(name)+": "+text)
			}
			return true
		})
	}

	if checked == 0 {
		t.Fatal("no CSV columns were found, so this test proves nothing")
	}
	if len(offenders) > 0 {
		t.Errorf(
			"these CSV columns are written without spreadsheetSafe:\n  %s\n\n"+
				"If the value cannot begin with =, + or @, add it to the exempt "+
				"list with the reason; otherwise wrap it.",
			strings.Join(offenders, "\n  "),
		)
	}
}

// expressionText renders an expression the way it is written in the source.
func expressionText(set *token.FileSet, node ast.Expr) string {
	var builder strings.Builder
	var walk func(ast.Expr)
	walk = func(current ast.Expr) {
		switch typed := current.(type) {
		case *ast.Ident:
			builder.WriteString(typed.Name)
		case *ast.SelectorExpr:
			walk(typed.X)
			builder.WriteString("." + typed.Sel.Name)
		case *ast.CallExpr:
			walk(typed.Fun)
			builder.WriteString("(")
			for index, argument := range typed.Args {
				if index > 0 {
					builder.WriteString(", ")
				}
				walk(argument)
			}
			builder.WriteString(")")
		default:
			builder.WriteString("?")
		}
	}
	walk(node)
	return builder.String()
}
