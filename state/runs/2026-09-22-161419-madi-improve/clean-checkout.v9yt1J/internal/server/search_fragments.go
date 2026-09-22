package server

import (
	"bytes"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

type searchFragment struct {
	Kind             string
	Start, End, Line int
	Content          string
	Metadata         map[string]any
}

func projectSearchFragments(markdown string) []searchFragment {
	source := markdownBodySource(markdown)
	newlines := []int{}
	for i, b := range source {
		if b == '\n' {
			newlines = append(newlines, i)
		}
	}
	result := []searchFragment{}
	add := func(kind string, start, end int, meta map[string]any) {
		if start < 0 || end > len(source) || start >= end {
			return
		}
		// Bound each FTS fragment while retaining exact UTF-8 source offsets. Large
		// blocks remain searchable in full through their overlapping search_chunks.
		end = min(end, start+8192)
		for end < len(source) && !utf8.RuneStart(source[end]) {
			end--
		}
		content := string(source[start:end])
		if strings.TrimSpace(content) == "" {
			return
		}
		result = append(result, searchFragment{kind, start, end, sort.SearchInts(newlines, start) + 1, content, meta})
	}
	parsed := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(source))
	_ = ast.Walk(parsed, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter || n.Type() != ast.TypeBlock || n.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		start, end := n.Lines().At(0).Start, n.Lines().At(n.Lines().Len()-1).Stop
		switch v := n.(type) {
		case *ast.FencedCodeBlock:
			add("code", start, end, map[string]any{"language": string(v.Language(source))})
			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock:
			add("code", start, end, map[string]any{"language": ""})
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock:
			if v.HasClosure() {
				end = v.ClosureLine.Stop
			}
		}
		if n.Parent() == parsed {
			add("block", start, end, map[string]any{"block_type": n.Kind().String()})
		}
		return ast.WalkContinue, nil
	})
	for _, task := range indexMarkdown(markdown).Tasks {
		start, end := task.Start, len(source)
		if task.HTML != nil {
			start, end = task.HTML.start, task.HTML.end
		} else if start >= 0 && start < len(source) {
			if i := bytes.IndexByte(source[start:], '\n'); i >= 0 {
				end = start + i
			}
		}
		add("task", start, end, map[string]any{"task_id": task.ID, "done": task.Done, "label": task.Text, "ambiguous": task.Ambiguous, "source_start": task.Start})
	}
	return result
}
