package store

import (
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyPoolDefaultsRespectsExplicitMaximum(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantMax int32
		wantMin int32
	}{
		{name: "one connection", dsn: "postgres://user:pass@localhost/db?pool_max_conns=1", wantMax: 1, wantMin: 1},
		{name: "three connections", dsn: "postgres://user:pass@localhost/db?pool_max_conns=3", wantMax: 3, wantMin: 2},
		{name: "explicit maximum above default", dsn: "postgres://user:pass@localhost/db?pool_max_conns=64", wantMax: 64, wantMin: 2},
		{name: "minimum capped to maximum", dsn: "postgres://user:pass@localhost/db?pool_max_conns=1&pool_min_conns=5", wantMax: 1, wantMin: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := pgxpool.ParseConfig(test.dsn)
			if err != nil {
				t.Fatal(err)
			}
			applyPoolDefaults(config)
			if config.MaxConns != test.wantMax || config.MinConns != test.wantMin {
				t.Fatalf("pool bounds = min:%d max:%d, want min:%d max:%d", config.MinConns, config.MaxConns, test.wantMin, test.wantMax)
			}
		})
	}
}

func TestApplyPoolDefaultsUsesBoundedAutomaticMaximum(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantMin int32
	}{
		{name: "plain DSN", dsn: "postgres://user:pass@localhost/db", wantMin: 2},
		{name: "oversized explicit minimum", dsn: "postgres://user:pass@localhost/db?pool_min_conns=40", wantMin: 16},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := pgxpool.ParseConfig(test.dsn)
			if err != nil {
				t.Fatal(err)
			}
			applyPoolDefaults(config)
			if config.MaxConns != 16 || config.MinConns != test.wantMin {
				t.Fatalf("automatic pool bounds = min:%d max:%d, want min:%d max:16", config.MinConns, config.MaxConns, test.wantMin)
			}
		})
	}
}

func TestNoteSearchPatternsSplitsAndEscapesTerms(t *testing.T) {
	got := noteSearchPatterns(`  한국어 검색 100% draft_name path\part `)
	want := []string{"%한국어%", "%검색%", `%100\%%`, `%draft\_name%`, `%path\\part%`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected search patterns: got %#v want %#v", got, want)
	}
}

func TestNoteSearchPatternsEmptyQuery(t *testing.T) {
	if got := noteSearchPatterns("  \t\n"); len(got) != 0 {
		t.Fatalf("empty query returned patterns: %#v", got)
	}
}

func TestLexicalScorePrefersExactBody(t *testing.T) {
	exact, _ := lexicalScore("bounded phrase", "", "bounded phrase", "")
	partial, _ := lexicalScore("bounded phrase", "", "prefix bounded phrase suffix", "")
	if exact <= partial {
		t.Fatalf("exact body score=%f, partial body score=%f", exact, partial)
	}
}
