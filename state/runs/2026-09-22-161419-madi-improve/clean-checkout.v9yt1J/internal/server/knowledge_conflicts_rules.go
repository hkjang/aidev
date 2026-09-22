package server

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

const conflictRuleVersion = "bounded-literal-v1"

type conflictSource struct {
	ID, Title, Markdown string
	Version             int
	Tags                []string
}
type conflictStatement struct {
	SourceID, Heading, Rule, Key, Excerpt string
	Version, Start, End, Line             int
	Values                                []string
}
type conflictRuleCandidate struct {
	Left, Right conflictStatement
	Topic       string
}
type conflictRuleDiagnostics struct {
	Documents, Lines, Statements int
	Truncated                    bool
}

var conflictNumber = regexp.MustCompile(`([-+]?\d+(?:,\d{3})*(?:\.\d+)?)[\t ]*(KiB/s|MiB/s|GiB/s|KB/s|MB/s|GB/s|KiB|MiB|GiB|TiB|KB|MB|GB|TB|ms|초|분|시간|일|주|개월|년|%|회|개|명|대|건)`)
var conflictPolicy = regexp.MustCompile(`허용|금지|필수|선택`)

func conflictFold(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, value)
}
func conflictStatements(ctx context.Context, source conflictSource) ([]conflictStatement, conflictRuleDiagnostics, error) {
	stats := conflictRuleDiagnostics{Documents: 1}
	if len(source.Markdown) > 1<<20 {
		return nil, stats, errors.New("규칙 검사는 문서당1MiB 이하입니다")
	}
	body := markdownBodySource(source.Markdown)
	visible := bytes.Repeat([]byte{' '}, len(body))
	for i, b := range body {
		if b == '\n' || b == '\r' {
			visible[i] = b
		}
	}
	tree := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(body))
	headings := map[int]string{}
	e := ast.Walk(tree, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		if err := ctx.Err(); err != nil {
			return ast.WalkStop, err
		}
		switch n := node.(type) {
		case *ast.Blockquote, *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML, *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			copy(visible[n.Segment.Start:n.Segment.Stop], body[n.Segment.Start:n.Segment.Stop])
		case *ast.Heading:
			if n.Lines().Len() > 0 {
				headings[bytes.Count(body[:n.Lines().At(0).Start], []byte{'\n'})+1] = strings.TrimSpace(string(n.Text(body)))
			}
		}
		return ast.WalkContinue, nil
	})
	if e != nil {
		return nil, stats, e
	}
	result := []conflictStatement{}
	offset, line, heading := 0, 0, source.Title
	for _, segment := range bytes.SplitAfter(visible, []byte{'\n'}) {
		line++
		if line > 2000 {
			stats.Truncated = true
			break
		}
		stats.Lines++
		if err := ctx.Err(); err != nil {
			return nil, stats, err
		}
		start, end := offset, offset+len(segment)
		offset = end
		if h, ok := headings[line]; ok {
			heading = h
			continue
		}
		value := strings.TrimSpace(string(segment))
		if len(value) > 2048 || len(value) < 4 {
			continue
		}
		lower := strings.ToLower(value)
		if strings.HasPrefix(lower, "예시") || strings.HasPrefix(lower, "예제") || strings.HasPrefix(lower, "example") || strings.ContainsAny(value, "\"“”‘’") {
			continue
		}
		base := conflictStatement{SourceID: source.ID, Version: source.Version, Heading: heading, Start: start, End: end, Line: line, Excerpt: source.Markdown[start:end]}
		matches := conflictNumber.FindAllStringSubmatchIndex(value, -1)
		if len(matches) > 0 && len(matches) <= 4 {
			keys := strings.Builder{}
			previous := 0
			values := []string{}
			valid := true
			for _, m := range matches {
				keys.WriteString(value[previous:m[0]])
				keys.WriteString("NUMBER")
				previous = m[1]
				n, err := strconv.ParseFloat(strings.ReplaceAll(value[m[2]:m[3]], ",", ""), 64)
				if err != nil {
					valid = false
					break
				}
				values = append(values, strconv.FormatFloat(n, 'g', -1, 64)+strings.ToLower(value[m[4]:m[5]]))
			}
			keys.WriteString(value[previous:])
			key := conflictFold(keys.String())
			if valid && len([]rune(key)) >= 10 && len([]rune(key)) <= 256 {
				item := base
				item.Rule = "quantity_difference"
				item.Key = key
				item.Values = values
				result = append(result, item)
			}
		}
		policies := conflictPolicy.FindAllStringIndex(value, -1)
		if len(policies) == 1 && !strings.Contains(value, "아니") && !strings.Contains(value, "않") && !strings.Contains(value, "예외") {
			m := policies[0]
			word := value[m[0]:m[1]]
			key := conflictFold(value[:m[0]] + "POLICY" + value[m[1]:])
			if len([]rune(key)) >= 10 && len([]rune(key)) <= 256 {
				item := base
				item.Rule = "permission_difference"
				if word == "필수" || word == "선택" {
					item.Rule = "requirement_difference"
				}
				item.Key = key
				item.Values = []string{word}
				result = append(result, item)
			}
		}
		if len(result) >= 200 {
			stats.Truncated = true
			break
		}
	}
	stats.Statements = len(result)
	return result, stats, nil
}
func detectKnowledgeConflicts(ctx context.Context, sources []conflictSource) ([]conflictRuleCandidate, conflictRuleDiagnostics, error) {
	stats := conflictRuleDiagnostics{}
	if len(sources) < 2 || len(sources) > 32 {
		return nil, stats, errors.New("비교할 문서를2~32개 선택하세요")
	}
	total := 0
	for _, source := range sources {
		total += len(source.Markdown)
	}
	if total > 8<<20 {
		return nil, stats, errors.New("선택한 원문 합계는8MiB 이하여야 합니다")
	}
	sorted := slices.Clone(sources)
	slices.SortFunc(sorted, func(a, b conflictSource) int { return strings.Compare(a.ID, b.ID) })
	byID := map[string]conflictSource{}
	buckets := map[string][]conflictStatement{}
	out := []conflictRuleCandidate{}
	seen := map[string]bool{}
	for _, source := range sorted {
		byID[source.ID] = source
		statements, st, e := conflictStatements(ctx, source)
		if e != nil {
			return nil, stats, e
		}
		stats.Documents += st.Documents
		stats.Lines += st.Lines
		stats.Statements += st.Statements
		stats.Truncated = stats.Truncated || st.Truncated
		for _, statement := range statements {
			key := statement.Rule + ":" + statement.Key
			for _, prior := range buckets[key] {
				if e := ctx.Err(); e != nil {
					return nil, stats, e
				}
				if prior.SourceID == statement.SourceID || slices.Equal(prior.Values, statement.Values) {
					continue
				}
				topic := ""
				if conflictFold(prior.Heading) != "" && conflictFold(prior.Heading) == conflictFold(statement.Heading) {
					topic = statement.Heading
				} else {
					for _, tag := range source.Tags {
						if conflictFold(tag) == "" {
							continue
						}
						for _, other := range byID[prior.SourceID].Tags {
							if conflictFold(tag) == conflictFold(other) {
								topic = "#" + tag
								break
							}
						}
						if topic != "" {
							break
						}
					}
				}
				if topic == "" {
					continue
				}
				signature := prior.SourceID + ":" + statement.SourceID + ":" + key + ":" + strings.Join(prior.Values, ",") + ":" + strings.Join(statement.Values, ",")
				if seen[signature] {
					continue
				}
				seen[signature] = true
				out = append(out, conflictRuleCandidate{Left: prior, Right: statement, Topic: topic})
				if len(out) >= 64 {
					stats.Truncated = true
					return out, stats, nil
				}
			}
			if len(buckets[key]) < 256 {
				buckets[key] = append(buckets[key], statement)
			} else {
				stats.Truncated = true
			}
		}
	}
	return out, stats, nil
}
