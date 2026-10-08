package storage

import "testing"

// Every LIKE built from typed search text used to hand the engine the text as a
// pattern. These are the three characters that made the pattern mean something
// other than the text.
func TestLikePatternEscapesTheCharactersThatAreNotLiteral(t *testing.T) {
	for _, testCase := range []struct {
		value string
		want  string
	}{
		{"db_prod", "db!_prod"},
		{"100%", "100!%"},
		{`C:\Program`, `C:\Program`},
		// The escape character itself has to survive being searched for.
		{"wow!", "wow!!"},
		{"api_key.create", "api!_key.create"},
		{"plain", "plain"},
	} {
		if got := LikePattern(testCase.value); got != testCase.want {
			t.Errorf("LikePattern(%q) = %q, want %q",
				testCase.value, got, testCase.want)
		}
	}
	if got := LikeContains("db_prod"); got != "%db!_prod%" {
		t.Errorf("LikeContains() = %q", got)
	}
}

// The clause is concatenated into statements assembled with fmt.Sprintf, where
// a stray verb would corrupt the SQL rather than fail loudly.
func TestLikeEscapeClauseCarriesNoFormattingVerb(t *testing.T) {
	for _, character := range LikeEscapeClause {
		if character == '%' {
			t.Fatalf("LikeEscapeClause = %q contains a formatting verb",
				LikeEscapeClause)
		}
	}
}
