package config

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var environmentName = regexp.MustCompile(`INVENQOR_[A-Z0-9_]+`)

// Guidance shown to an operator must name a variable that exists.
//
// The KEYCLOAK_SECRET_UNREADABLE diagnostic told operators to "set
// INVENQOR_MASTER_KEY to the same value on every instance". There is no such
// variable - the real one is INVENQOR_MASTER_KEY_FILE and it takes a path.
// Someone following that instruction sees no change and has no reason to doubt
// it, which is worse than being told nothing.
//
// The rule that catches it: a name written inside a sentence must also appear
// somewhere on its own, as the literal passed to whatever reads it. A name that
// only ever appears inside prose is a name nothing reads.
func TestEnvironmentVariablesNamedInGuidanceAreRead(t *testing.T) {
	root := repositoryRoot(t)
	declared := map[string]bool{}
	mentions := map[string][]string{}

	forEachStringLiteral(t, root, func(file string, line int, value string) {
		names := environmentName.FindAllString(value, -1)
		if len(names) == 0 {
			return
		}
		// The literal is exactly one variable name: this is a read site.
		if len(names) == 1 && strings.TrimSpace(value) == names[0] {
			declared[names[0]] = true
			return
		}
		for _, name := range names {
			mentions[name] = append(
				mentions[name],
				filepath.Base(file)+":"+strconv.Itoa(line),
			)
		}
	})

	if len(declared) == 0 {
		t.Fatal("no environment variables were found, so this test proves nothing")
	}
	for name, where := range mentions {
		if !declared[name] {
			t.Errorf(
				"%s is named in guidance at %s but nothing reads it;\n"+
					"an operator following that instruction would see no effect",
				name, strings.Join(where, ", "),
			)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	// This package sits at server/internal/config.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func forEachGoFile(
	t *testing.T,
	root string,
	visit func(path string, set *token.FileSet, parsed *ast.File),
) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(root, "internal", "*", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	more, err := filepath.Glob(filepath.Join(root, "cmd", "*", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range append(files, more...) {
		if strings.HasSuffix(path, "guidance_test.go") {
			continue
		}
		set := token.NewFileSet()
		parsed, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		visit(path, set, parsed)
	}
}

func forEachStringLiteral(
	t *testing.T,
	root string,
	visit func(file string, line int, value string),
) {
	t.Helper()
	forEachGoFile(t, root, func(path string, set *token.FileSet, parsed *ast.File) {
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			visit(path, set.Position(literal.Pos()).Line, value)
			return true
		})
	})
}

func TestKoreanGuidanceDoesNotTrailIntoEnglish(t *testing.T) {
	root := repositoryRoot(t)
	hangul := regexp.MustCompile(`[가-힣]`)
	englishClause := regexp.MustCompile(`\b[a-z]+ [a-z]+ [a-z]+\b`)

	var offenders []string
	// Concatenations, not individual literals. The two that shipped were
	// "설정 > ... 바꾸거나, " + "or provision the device manually...", where each
	// half is unremarkable on its own: the Korean one has no English clause and
	// the English one has no Hangul. Checking them separately reproduces the
	// blind spot that caused the defect - the first version of this test did
	// exactly that and passed on both.
	forEachJoinedString(t, root, func(file string, line int, value string) {
		if !hangul.MatchString(value) {
			return
		}
		if match := englishClause.FindString(value); match != "" {
			offenders = append(offenders, fmt.Sprintf(
				"%s:%d: %q trails into %q",
				filepath.Base(file), line, value, match,
			))
		}
	})
	if len(offenders) > 0 {
		t.Errorf(
			"these read as neither Korean nor English:\n%s",
			strings.Join(offenders, "\n"),
		)
	}
}

// forEachJoinedString visits every string expression with adjacent literals
// joined by + folded together, so a sentence split across several literals is
// seen as the sentence a reader gets.
func forEachJoinedString(
	t *testing.T,
	root string,
	visit func(file string, line int, value string),
) {
	t.Helper()
	var fold func(node ast.Expr) (string, bool)
	fold = func(node ast.Expr) (string, bool) {
		switch typed := node.(type) {
		case *ast.BasicLit:
			if typed.Kind != token.STRING {
				return "", false
			}
			value, err := strconv.Unquote(typed.Value)
			return value, err == nil
		case *ast.BinaryExpr:
			if typed.Op != token.ADD {
				return "", false
			}
			left, leftOK := fold(typed.X)
			right, rightOK := fold(typed.Y)
			if !leftOK || !rightOK {
				return "", false
			}
			return left + right, true
		case *ast.ParenExpr:
			return fold(typed.X)
		}
		return "", false
	}

	forEachGoFile(t, root, func(path string, set *token.FileSet, parsed *ast.File) {
		ast.Inspect(parsed, func(node ast.Node) bool {
			expression, ok := node.(ast.Expr)
			if !ok {
				return true
			}
			value, folded := fold(expression)
			if !folded {
				return true
			}
			visit(path, set.Position(expression.Pos()).Line, value)
			// The parts are covered by the whole.
			return false
		})
	})
}
