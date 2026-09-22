package server

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// A links-only projection avoids the line/task bookkeeping cost when indexing
// a large Markdown document. Hidden code, HTML and YAML are never wiki edges.
func projectGraphReferences(markdown string) ([]string, []string) {
	source := markdownBodySource(markdown)
	parsed := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(source))
	visible := bytes.Repeat([]byte{' '}, len(source))
	_ = ast.Walk(parsed, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.HTMLBlock, *ast.CodeSpan, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.RawHTML, *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if n.Segment.Start >= 0 && n.Segment.Stop <= len(source) {
				copy(visible[n.Segment.Start:n.Segment.Stop], source[n.Segment.Start:n.Segment.Stop])
			}
		}
		return ast.WalkContinue, nil
	})
	links, tags := []string{}, []string{}
	seen := map[string]bool{}
	for _, pos := range wikiPattern.FindAllSubmatchIndex(visible, -1) {
		escapes := 0
		for i := pos[0] - 1; i >= 0 && source[i] == '\\'; i-- {
			escapes++
		}
		if escapes%2 != 0 {
			continue
		}
		target, _, _ := splitVaultWiki(string(source[pos[2]:pos[3]]))
		target = strings.TrimSpace(target)
		if target != "" && !seen[target] {
			links = append(links, target)
			seen[target] = true
		}
	}
	seen = map[string]bool{}
	if fm, e := parseFrontMatter(markdown); e == nil {
		for _, tag := range listStrings(fm["tags"]) {
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}
	for _, m := range inlineTagPattern.FindAllSubmatch(visible, -1) {
		tag := string(m[1])
		if !seen[tag] {
			tags = append(tags, tag)
			seen[tag] = true
		}
	}
	return links, tags
}
