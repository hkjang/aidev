// Package classify turns collected inventory into business context.
//
// The design constraints come from CMDB practice rather than from what is easy:
//
//   - Explainable. Every assignment names the rule that made it, so an operator
//     can answer "why is this host critical" without reading code. A model that
//     cannot be explained cannot be trusted with a change-management decision.
//   - Deterministic and ordered. Rules run by priority and later rules may match
//     on what earlier ones assigned, which is how "production databases are
//     critical" is expressed without a second engine.
//   - Human values win. A field an operator has set is never overwritten by a
//     later automatic pass; the golden record belongs to the person.
//   - Re-runnable. The same rule set applied to the same inventory yields the
//     same result, so a rule change can be replayed over existing assets.
package classify

import (
	"encoding/json"
	"sort"
	"strings"
)

// Match is the predicate half of a rule. An empty field means "do not care";
// every populated field must match, and within a field any member may match.
type Match struct {
	Categories []string `json:"categories,omitempty"`
	// NamePatterns match the whole name with '*' wildcards. Good for software
	// families ("postgres*"), wrong for short environment codes: "*stg*" also
	// matches "po-stg-resql".
	NamePatterns []string `json:"name_patterns,omitempty"`
	// NameTokens match a delimiter-separated token of the name exactly, which is
	// how site naming conventions actually work ("app-prd-01" carries the token
	// "prd"). Anchoring to token boundaries is what keeps an environment rule
	// from firing on an unrelated product name.
	NameTokens        []string          `json:"name_tokens,omitempty"`
	Types             []string          `json:"types,omitempty"`
	Environments      []string          `json:"environments,omitempty"`
	AttributeEquals   map[string]string `json:"attribute_equals,omitempty"`
	AttributeContains map[string]string `json:"attribute_contains,omitempty"`
}

// Assign is the consequence half. Only the populated fields are applied.
type Assign struct {
	Type            string   `json:"type,omitempty"`
	Environment     string   `json:"environment,omitempty"`
	Criticality     string   `json:"criticality,omitempty"`
	OwnerDepartment string   `json:"owner_department,omitempty"`
	Location        string   `json:"location,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	// RelateToHost asks the relationship pass to link this asset to the host it
	// was collected from. The rule set is where that belongs: whether a package
	// deserves an edge is a curation decision, not a code decision.
	RelateToHost bool   `json:"relate_to_host,omitempty"`
	Relation     string `json:"relation,omitempty"`
}

type Rule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Priority    int     `json:"priority"`
	Enabled     bool    `json:"enabled"`
	SystemRule  bool    `json:"system_rule"`
	Confidence  float64 `json:"confidence"`
	Match       Match   `json:"match"`
	Assign      Assign  `json:"assign"`
}

// Subject is the asset being classified, as the engine sees it.
type Subject struct {
	Category    string
	Name        string
	Type        string
	Environment string
	Criticality string
	// Attributes is the collector payload, addressed with dotted paths.
	Attributes map[string]any
	// ManualFields are the fields a person has set; the engine leaves them alone.
	ManualFields []string
}

// Result is what the engine decided, with the trail that produced it.
type Result struct {
	Type            string
	Environment     string
	Criticality     string
	OwnerDepartment string
	Location        string
	Tags            []string
	RelateToHost    bool
	Relation        string
	// AppliedRules lists the rule identifiers in the order they fired.
	AppliedRules []string
	// Confidence is the lowest confidence among the rules that set a field, so a
	// single weak guess never presents itself as a certainty.
	Confidence float64
	// Skipped names the fields a rule wanted but a person already owns.
	Skipped []string
}

// Sort orders a rule set the way the engine will run it. Ties break on name so
// two deployments with the same rules always classify identically.
func Sort(rules []Rule) {
	sort.SliceStable(rules, func(left, right int) bool {
		if rules[left].Priority != rules[right].Priority {
			return rules[left].Priority < rules[right].Priority
		}
		return rules[left].Name < rules[right].Name
	})
}

// Apply runs the rule set over one subject.
func Apply(rules []Rule, subject Subject) Result {
	ordered := make([]Rule, len(rules))
	copy(ordered, rules)
	Sort(ordered)

	manual := map[string]bool{}
	for _, field := range subject.ManualFields {
		manual[field] = true
	}
	result := Result{
		Type:         subject.Type,
		Environment:  subject.Environment,
		Criticality:  subject.Criticality,
		Tags:         []string{},
		AppliedRules: []string{},
		Skipped:      []string{},
		Confidence:   0,
	}
	// Track the running view so a later rule sees earlier assignments.
	running := subject
	lowest := 0.0
	assigned := false

	for _, rule := range ordered {
		if !rule.Enabled || !matches(rule.Match, running) {
			continue
		}
		fired := false
		set := func(field string, value string, target *string) {
			if value == "" {
				return
			}
			if manual[field] {
				result.Skipped = appendUnique(result.Skipped, field)
				return
			}
			if *target != value {
				*target = value
			}
			fired = true
		}
		set("type", rule.Assign.Type, &result.Type)
		set("environment", rule.Assign.Environment, &result.Environment)
		set("criticality", rule.Assign.Criticality, &result.Criticality)
		set("owner_department", rule.Assign.OwnerDepartment, &result.OwnerDepartment)
		set("location", rule.Assign.Location, &result.Location)
		for _, tag := range rule.Assign.Tags {
			if tag == "" {
				continue
			}
			result.Tags = appendUnique(result.Tags, tag)
			fired = true
		}
		if rule.Assign.RelateToHost {
			result.RelateToHost = true
			if rule.Assign.Relation != "" {
				result.Relation = rule.Assign.Relation
			}
			fired = true
		}
		if !fired {
			continue
		}
		result.AppliedRules = append(result.AppliedRules, rule.ID)
		running.Type = result.Type
		running.Environment = result.Environment
		running.Criticality = result.Criticality
		if !assigned || rule.Confidence < lowest {
			lowest = rule.Confidence
		}
		assigned = true
	}
	if assigned {
		result.Confidence = lowest
	}
	sort.Strings(result.Tags)
	return result
}

func matches(match Match, subject Subject) bool {
	if len(match.Categories) > 0 && !containsFold(match.Categories, subject.Category) {
		return false
	}
	if len(match.Types) > 0 && !containsFold(match.Types, subject.Type) {
		return false
	}
	if len(match.Environments) > 0 &&
		!containsFold(match.Environments, subject.Environment) {
		return false
	}
	if len(match.NamePatterns) > 0 && !matchesAnyPattern(match.NamePatterns, subject.Name) {
		return false
	}
	if len(match.NameTokens) > 0 && !matchesAnyToken(match.NameTokens, subject.Name) {
		return false
	}
	for path, expected := range match.AttributeEquals {
		if !strings.EqualFold(attributeString(subject.Attributes, path), expected) {
			return false
		}
	}
	for path, needle := range match.AttributeContains {
		haystack := strings.ToLower(attributeString(subject.Attributes, path))
		if !strings.Contains(haystack, strings.ToLower(needle)) {
			return false
		}
	}
	return true
}

// matchesAnyToken splits the name on the separators that appear in host and unit
// naming conventions and compares whole tokens.
func matchesAnyToken(tokens []string, value string) bool {
	fields := strings.FieldsFunc(strings.ToLower(value), func(char rune) bool {
		return char == '-' || char == '_' || char == '.' || char == '@' ||
			char == ':' || char == '/' || char == ' ' ||
			(char >= '0' && char <= '9')
	})
	for _, token := range tokens {
		candidate := strings.ToLower(strings.TrimSpace(token))
		if candidate == "" {
			continue
		}
		for _, field := range fields {
			if field == candidate {
				return true
			}
		}
	}
	return false
}

// matchesAnyPattern supports the '*' wildcard only. A full regular expression
// engine in a rule column is a denial-of-service waiting to be configured, and
// site naming conventions never need more than a wildcard.
func matchesAnyPattern(patterns []string, value string) bool {
	subject := strings.ToLower(strings.TrimSpace(value))
	for _, pattern := range patterns {
		if matchPattern(strings.ToLower(strings.TrimSpace(pattern)), subject) {
			return true
		}
	}
	return false
}

func matchPattern(pattern string, value string) bool {
	if pattern == "" {
		return false
	}
	segments := strings.Split(pattern, "*")
	if len(segments) == 1 {
		return pattern == value
	}
	remaining := value
	for index, segment := range segments {
		if segment == "" {
			continue
		}
		switch {
		case index == 0:
			if !strings.HasPrefix(remaining, segment) {
				return false
			}
			remaining = remaining[len(segment):]
		case index == len(segments)-1:
			return strings.HasSuffix(remaining, segment)
		default:
			position := strings.Index(remaining, segment)
			if position < 0 {
				return false
			}
			remaining = remaining[position+len(segment):]
		}
	}
	return true
}

func attributeString(attributes map[string]any, path string) string {
	var current any = attributes
	for _, key := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = object[key]
		if !ok {
			return ""
		}
	}
	switch typed := current.(type) {
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	default:
		return ""
	}
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}

// DecodeMatch and DecodeAssign read the rule columns, tolerating the empty
// object a hand-edited row may contain.
func DecodeMatch(raw string) (Match, error) {
	var match Match
	if strings.TrimSpace(raw) == "" {
		return match, nil
	}
	return match, json.Unmarshal([]byte(raw), &match)
}

func DecodeAssign(raw string) (Assign, error) {
	var assign Assign
	if strings.TrimSpace(raw) == "" {
		return assign, nil
	}
	return assign, json.Unmarshal([]byte(raw), &assign)
}
