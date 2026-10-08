package classify

import (
	"strings"
	"testing"
)

func categoryRule() Rule {
	return Rule{
		ID: "rule-type", Name: "category", Priority: 10, Enabled: true,
		Confidence: 1.0,
		Match:      Match{Categories: []string{"service"}},
		Assign:     Assign{Type: "service", RelateToHost: true, Relation: "runs_on"},
	}
}

func databaseRule() Rule {
	return Rule{
		ID: "rule-database", Name: "database", Priority: 40, Enabled: true,
		Confidence: 0.9,
		Match: Match{
			Categories:   []string{"service"},
			NamePatterns: []string{"postgres*", "*mysqld*"},
		},
		Assign: Assign{
			Type: "database", Tags: []string{"data-tier"},
			RelateToHost: true, Relation: "runs_on",
		},
	}
}

func productionRule() Rule {
	return Rule{
		ID: "rule-production", Name: "production naming", Priority: 60, Enabled: true,
		Confidence: 0.6,
		Match:      Match{NamePatterns: []string{"*prd*"}},
		Assign:     Assign{Environment: "production"},
	}
}

func criticalRule() Rule {
	return Rule{
		ID: "rule-critical", Name: "production data tier", Priority: 80, Enabled: true,
		Confidence: 0.7,
		Match: Match{
			Environments: []string{"production"},
			Types:        []string{"database"},
		},
		Assign: Assign{Criticality: "critical"},
	}
}

func TestRulesChainInPriorityOrder(t *testing.T) {
	rules := []Rule{criticalRule(), productionRule(), databaseRule(), categoryRule()}
	result := Apply(rules, Subject{
		Category: "service",
		Name:     "postgresql@14-prd01",
	})

	if result.Type != "database" {
		t.Fatalf("type = %q, want database", result.Type)
	}
	if result.Environment != "production" {
		t.Fatalf("environment = %q, want production", result.Environment)
	}
	// This is the point of the ordered pipeline: criticality depends on values
	// two earlier rules assigned.
	if result.Criticality != "critical" {
		t.Fatalf("criticality = %q, want critical", result.Criticality)
	}
	if len(result.Tags) != 1 || result.Tags[0] != "data-tier" {
		t.Fatalf("tags = %v", result.Tags)
	}
	if !result.RelateToHost || result.Relation != "runs_on" {
		t.Fatalf("relationship = %t %q", result.RelateToHost, result.Relation)
	}
	if strings.Join(result.AppliedRules, ",") !=
		"rule-type,rule-database,rule-production,rule-critical" {
		t.Fatalf("applied rules = %v", result.AppliedRules)
	}
	// Confidence is the weakest link, not the last rule to fire.
	if result.Confidence != 0.6 {
		t.Fatalf("confidence = %v, want the lowest contributing 0.6", result.Confidence)
	}
}

func TestManualFieldsAreNeverOverwritten(t *testing.T) {
	result := Apply(
		[]Rule{categoryRule(), productionRule(), databaseRule(), criticalRule()},
		Subject{
			Category:     "service",
			Name:         "postgres-prd-01",
			Criticality:  "medium",
			ManualFields: []string{"criticality"},
		},
	)
	if result.Criticality != "medium" {
		t.Fatalf("criticality = %q, want the operator's medium", result.Criticality)
	}
	found := false
	for _, field := range result.Skipped {
		if field == "criticality" {
			found = true
		}
	}
	if !found {
		t.Fatalf("skipped = %v, want criticality reported", result.Skipped)
	}
	// The rest of the classification must still apply.
	if result.Type != "database" || result.Environment != "production" {
		t.Fatalf("result = %+v", result)
	}
}

func TestDisabledRuleDoesNothing(t *testing.T) {
	rule := databaseRule()
	rule.Enabled = false
	result := Apply([]Rule{categoryRule(), rule}, Subject{
		Category: "service", Name: "postgres-01",
	})
	if result.Type != "service" {
		t.Fatalf("type = %q, want the category default", result.Type)
	}
	if len(result.Tags) != 0 {
		t.Fatalf("tags = %v", result.Tags)
	}
}

func TestUnmatchedSubjectKeepsItsExistingValues(t *testing.T) {
	result := Apply([]Rule{databaseRule(), criticalRule()}, Subject{
		Category: "software.package", Name: "fonts-noto", Type: "software",
		Environment: "other",
	})
	if result.Type != "software" || result.Environment != "other" {
		t.Fatalf("result = %+v", result)
	}
	if len(result.AppliedRules) != 0 || result.Confidence != 0 {
		t.Fatalf("nothing should have fired: %+v", result)
	}
}

func TestPatternMatching(t *testing.T) {
	cases := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"postgres*", "postgresql@14", true},
		{"postgres*", "my-postgres", false},
		{"*prd*", "app-PRD-01", true},
		{"*prd*", "app-stg-01", false},
		{"*mysqld*", "mysqld.service", true},
		{"nginx", "nginx", true},
		{"nginx", "nginx.service", false},
		{"*", "anything", true},
		{"", "anything", false},
		{"a*b*c", "azzbzzc", true},
		{"a*b*c", "azzc", false},
	}
	for _, testCase := range cases {
		got := matchesAnyPattern([]string{testCase.pattern}, testCase.value)
		if got != testCase.want {
			t.Errorf("pattern %q against %q = %t, want %t",
				testCase.pattern, testCase.value, got, testCase.want)
		}
	}
}

func TestAttributePredicates(t *testing.T) {
	subject := Subject{
		Category: "system",
		Name:     "web-01",
		Attributes: map[string]any{
			"os_release":             map[string]any{"id": "ubuntu", "version_id": "24.04"},
			"agent_is_containerized": true,
		},
	}
	nested := Rule{
		ID: "nested", Enabled: true, Confidence: 1,
		Match:  Match{AttributeEquals: map[string]string{"os_release.id": "Ubuntu"}},
		Assign: Assign{OwnerDepartment: "linux-team"},
	}
	if result := Apply([]Rule{nested}, subject); result.OwnerDepartment != "linux-team" {
		t.Fatalf("nested attribute equality did not match: %+v", result)
	}
	boolean := Rule{
		ID: "boolean", Enabled: true, Confidence: 1,
		Match:  Match{AttributeEquals: map[string]string{"agent_is_containerized": "true"}},
		Assign: Assign{Tags: []string{"containerized"}},
	}
	if result := Apply([]Rule{boolean}, subject); len(result.Tags) != 1 {
		t.Fatalf("boolean attribute did not match: %+v", result)
	}
	missing := Rule{
		ID: "missing", Enabled: true, Confidence: 1,
		Match:  Match{AttributeEquals: map[string]string{"os_release.absent": "x"}},
		Assign: Assign{Tags: []string{"nope"}},
	}
	if result := Apply([]Rule{missing}, subject); len(result.Tags) != 0 {
		t.Fatalf("absent attribute must not match: %+v", result)
	}
	contains := Rule{
		ID: "contains", Enabled: true, Confidence: 1,
		Match:  Match{AttributeContains: map[string]string{"os_release.version_id": "24."}},
		Assign: Assign{Tags: []string{"lts"}},
	}
	if result := Apply([]Rule{contains}, subject); len(result.Tags) != 1 {
		t.Fatalf("substring attribute did not match: %+v", result)
	}
}

func TestApplyIsDeterministicRegardlessOfInputOrder(t *testing.T) {
	subject := Subject{Category: "service", Name: "postgres-prd-01"}
	forward := Apply(
		[]Rule{categoryRule(), databaseRule(), productionRule(), criticalRule()},
		subject,
	)
	backward := Apply(
		[]Rule{criticalRule(), productionRule(), databaseRule(), categoryRule()},
		subject,
	)
	if forward.Type != backward.Type ||
		forward.Criticality != backward.Criticality ||
		strings.Join(forward.AppliedRules, ",") != strings.Join(backward.AppliedRules, ",") {
		t.Fatalf("order changed the outcome:\n%+v\n%+v", forward, backward)
	}
}

func TestDecodeRuleColumns(t *testing.T) {
	match, err := DecodeMatch(`{"categories":["service"],"name_patterns":["nginx*"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(match.Categories) != 1 || match.NamePatterns[0] != "nginx*" {
		t.Fatalf("match = %+v", match)
	}
	assign, err := DecodeAssign(`{"type":"service","relate_to_host":true,"relation":"runs_on"}`)
	if err != nil {
		t.Fatal(err)
	}
	if assign.Type != "service" || !assign.RelateToHost || assign.Relation != "runs_on" {
		t.Fatalf("assign = %+v", assign)
	}
	if _, err := DecodeMatch(""); err != nil {
		t.Fatalf("an empty column must decode to an empty predicate: %v", err)
	}
}
