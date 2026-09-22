package server

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type markdownHeading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	Line  int    `json:"line"`
}
type markdownTask struct {
	Text      string         `json:"text"`
	Done      bool           `json:"done"`
	Line      int            `json:"line"`
	EndLine   int            `json:"end_line"`
	ID        string         `json:"task_id"`
	Ambiguous bool           `json:"ambiguous"`
	Start     int            `json:"source_start"`
	HTML      *htmlTaskRange `json:"-"`
}
type markdownIndex struct {
	Links    []string
	Tags     []string
	Headings []markdownHeading
	Tasks    []markdownTask
	Plain    string
}

var inlineTagPattern = regexp.MustCompile(`(?:^|[\s(])#([\p{L}\p{N}_][\p{L}\p{N}_/-]*)`)
var taskLabelMarker = regexp.MustCompile(`^\[[ xX]\][\t ]*`)
var taskReference = regexp.MustCompile(`\[[^\[\]\n]*\]\(<?/app/tasks\?task=[0-9a-fA-F-]{36}>?\)`)

// Keep byte offsets intact while excluding YAML from Markdown semantics.
func markdownBodySource(markdown string) []byte {
	source := []byte(markdown)
	if !strings.HasPrefix(markdown, "---\n") && !strings.HasPrefix(markdown, "---\r\n") {
		return source
	}
	offset := 0
	for i, line := range strings.SplitAfter(markdown, "\n") {
		offset += len(line)
		if i > 0 && strings.TrimSpace(line) == "---" {
			for p := 0; p < offset; p++ {
				if source[p] != '\n' && source[p] != '\r' {
					source[p] = ' '
				}
			}
			break
		}
	}
	return source
}

// Use the same CommonMark/GFM structural rules for links, tags, outline and
// tasks. Examples in inline/fenced/indented code and raw HTML are not wiki links
// or tasks; nested list items retain their original source line coordinates.
func indexMarkdown(markdown string) markdownIndex {
	source := markdownBodySource(markdown)
	parsed := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(source))
	visible := bytes.Repeat([]byte{' '}, len(source))
	for i, b := range source {
		if b == '\n' || b == '\r' {
			visible[i] = b
		}
	}
	result := markdownIndex{Links: []string{}, Tags: []string{}, Headings: []markdownHeading{}, Tasks: []markdownTask{}}
	_ = ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.HTMLBlock:
			if n.Lines().Len() > 0 {
				start, end := n.Lines().At(0).Start, n.Lines().At(n.Lines().Len()-1).Stop
				if n.HasClosure() {
					end = n.ClosureLine.Stop
				}
				result.Tasks = append(result.Tasks, indexHTMLTasks(source, start, end)...)
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.RawHTML, *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if n.Segment.Start >= 0 && n.Segment.Stop <= len(source) {
				copy(visible[n.Segment.Start:n.Segment.Stop], source[n.Segment.Start:n.Segment.Stop])
			}
		case *ast.Heading:
			line := 0
			if n.Lines().Len() > 0 {
				line = bytes.Count(source[:n.Lines().At(0).Start], []byte{'\n'})
			}
			result.Headings = append(result.Headings, markdownHeading{Level: n.Level, Text: string(n.Text(source)), Line: line})
		case *extast.TaskCheckBox:
			parent := n.Parent()
			for parent != nil && parent.Type() == ast.TypeInline {
				parent = parent.Parent()
			}
			if parent == nil || parent.Lines().Len() == 0 {
				return ast.WalkContinue, nil
			}
			line := bytes.Count(source[:parent.Lines().At(0).Start], []byte{'\n'})
			label := strings.TrimSpace(string(parent.Text(source)))
			// TextBlock.Text retains source markers even after inline parsing.
			// Checkbox state is a separate field, never part of a task identity.
			label = taskLabelMarker.ReplaceAllString(label, "")
			ids := []string{}
			_ = ast.Walk(parent, func(child ast.Node, enter bool) (ast.WalkStatus, error) {
				if !enter {
					return ast.WalkContinue, nil
				}
				switch v := child.(type) {
				case *ast.CodeSpan, *ast.Image:
					return ast.WalkSkipChildren, nil
				case *ast.Link:
					if id, ok := strings.CutPrefix(string(v.Destination), "/app/tasks?task="); ok && validID(id) {
						ids = append(ids, strings.ToLower(id))
					}
				}
				return ast.WalkContinue, nil
			})
			task := markdownTask{Text: label, Done: n.IsChecked, Line: line, EndLine: bytes.Count(source[:parent.Lines().At(parent.Lines().Len()-1).Stop], []byte{'\n'})}
			task.Start = parent.Lines().At(0).Start
			if len(ids) > 0 {
				task.ID = ids[0]
				task.Ambiguous = len(ids) > 1
				task.Text = strings.TrimSpace(taskReference.ReplaceAllString(label, ""))
			}
			result.Tasks = append(result.Tasks, task)
		}
		return ast.WalkContinue, nil
	})
	seen := map[string]bool{}
	for _, position := range wikiPattern.FindAllSubmatchIndex(visible, -1) {
		escapes := 0
		for p := position[0] - 1; p >= 0 && source[p] == '\\'; p-- {
			escapes++
		}
		if escapes%2 != 0 {
			continue
		}
		value := string(source[position[2]:position[3]])
		target, _, _ := splitVaultWiki(value)
		target = strings.TrimSpace(target)
		if target != "" && !seen[target] {
			result.Links = append(result.Links, target)
			seen[target] = true
		}
	}
	seen = map[string]bool{}
	if fm, e := parseFrontMatter(markdown); e == nil {
		for _, tag := range listStrings(fm["tags"]) {
			if !seen[tag] {
				result.Tags = append(result.Tags, tag)
				seen[tag] = true
			}
		}
	}
	for _, match := range inlineTagPattern.FindAllSubmatch(visible, -1) {
		tag := string(match[1])
		if !seen[tag] {
			result.Tags = append(result.Tags, tag)
			seen[tag] = true
		}
	}
	result.Plain = strings.Join(strings.Fields(string(visible)), " ")
	sort.SliceStable(result.Tasks, func(i, j int) bool { return result.Tasks[i].Line < result.Tasks[j].Line })
	identities := map[string]int{}
	for _, task := range result.Tasks {
		if task.ID != "" {
			identities[task.ID]++
		}
	}
	for i := range result.Tasks {
		if result.Tasks[i].ID != "" && identities[result.Tasks[i].ID] > 1 {
			result.Tasks[i].Ambiguous = true
		}
	}
	return result
}
