package storage_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// PostgreSQL has no LIKE for jsonb: `payload_json LIKE $1` fails with SQLSTATE
// 42883, "operator does not exist: jsonb ~~ unknown". On the SQLite start-up
// fallback the same column is TEXT and the same statement works, so every test
// in this repository passed while the query failed on every ingest in
// production. Casting to text is what both engines accept.
//
// This reads the shipped migrations for the columns PostgreSQL declares as JSONB
// and then checks the source for a text operator applied to one of them without
// a cast, so the next such statement is caught here rather than in a server log.
func TestNoTextOperatorIsAppliedToAJsonbColumnWithoutACast(t *testing.T) {
	root := filepath.Join("..", "..")
	jsonbColumns := map[string]bool{}
	migrations, err := filepath.Glob(filepath.Join(root, "migrations", "postgres", "*.sql"))
	if err != nil || len(migrations) == 0 {
		t.Fatalf("no PostgreSQL migrations found: %v", err)
	}
	declaration := regexp.MustCompile(`(?i)^\s*([a-z_]+)\s+JSONB\b`)
	for _, path := range migrations {
		text, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(text), "\n") {
			if match := declaration.FindStringSubmatch(line); match != nil {
				jsonbColumns[strings.ToLower(match[1])] = true
			}
		}
	}
	if len(jsonbColumns) == 0 {
		t.Fatal("no JSONB columns were found, so this test proves nothing")
	}

	// Operators PostgreSQL defines for text and not for jsonb.
	operators := []string{"LIKE", "ILIKE", "SIMILAR TO"}
	var findings []string
	err = filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		// Test files are scanned too. A test that applies a text operator to a
		// JSONB column fails only when the suite runs against PostgreSQL, and the
		// default is SQLite - so one sat in server_test.go until the PostgreSQL
		// run found it. This file is excluded because it names the operators.
		if strings.HasSuffix(path, "jsonb_operator_test.go") {
			return nil
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for number, line := range strings.Split(string(text), "\n") {
			upper := strings.ToUpper(line)
			for column := range jsonbColumns {
				for _, operator := range operators {
					// Qualified or bare, but not already wrapped in a cast.
					pattern := regexp.MustCompile(
						`(?i)(^|[^)\w.])(\w+\.)?` + regexp.QuoteMeta(column) + `\s+` + operator + `\b`,
					)
					if pattern.MatchString(line) && !strings.Contains(upper, "CAST(") {
						findings = append(findings, filepath.Base(path)+":"+
							itoa(number+1)+": "+column+" is JSONB and "+operator+
							" has no jsonb form; wrap it in CAST(... AS TEXT)")
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := ""
	for value > 0 {
		digits = string(rune('0'+value%10)) + digits
		value /= 10
	}
	return digits
}
