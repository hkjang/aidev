package server

import (
	"bytes"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

const splitSourceMaxBytes = 1 << 20
const splitSelectionMaxBytes = 256 << 10

// Independent parsing must retain the exact top-level CommonMark/GFM
// structure. A partial paragraph/list/table/fence cannot silently turn into a
// different block when it becomes its own document. This comparison is local;
// generated HTML is never sent to a browser, interpreted or fetched.
func validateDocumentSplitRange(markdown string, start, end int, selection string) error {
	if len(markdown) > splitSourceMaxBytes || len(selection) > splitSelectionMaxBytes {
		return errors.New("문서 분리는 원문1MiB·선택256KiB 이내에서 지원합니다. 더 큰 문서는 먼저 작은 범위로 정리하세요")
	}
	bad := errors.New("완전한 Markdown 문단·목록·표·코드 블록을 선택하세요. Front Matter·HTML 고급 블록 또는 중간 구조는 분리하지 않습니다")
	if len(markdown) > splitSourceMaxBytes || start < 0 || end <= start || end > len(markdown) || end-start > splitSelectionMaxBytes || !utf8.ValidString(markdown[:start]) || !utf8.ValidString(markdown[start:end]) || markdown[start:end] != selection || strings.TrimSpace(selection) == "" {
		return bad
	}
	if start > 0 && markdown[start-1] != '\n' || end < len(markdown) && markdown[end-1] != '\n' && markdown[end] != '\n' {
		return bad
	}
	source := markdownBodySource(markdown)
	if !bytes.Equal(source[start:end], []byte(selection)) {
		return bad
	}
	parser := goldmark.New(goldmark.WithExtensions(extension.GFM))
	node := parser.Parser().Parse(text.NewReader([]byte(selection)))
	htmlFound := false
	_ = ast.Walk(node, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if enter && (n.Kind() == ast.KindHTMLBlock || n.Kind() == ast.KindRawHTML) {
			htmlFound = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if htmlFound {
		return bad
	}
	render := func(value []byte) ([]byte, error) {
		var out bytes.Buffer
		e := parser.Convert(value, &out)
		return out.Bytes(), e
	}
	full, e := render(source)
	if e != nil {
		return bad
	}
	var fragments []byte
	for _, part := range [][]byte{source[:start], source[start:end], source[end:]} {
		rendered, e := render(part)
		if e != nil {
			return bad
		}
		fragments = append(fragments, rendered...)
	}
	if !bytes.Equal(full, fragments) {
		return bad
	}
	return nil
}

func documentSplitReplacement(markdown string, start, end int, childID, title string) string {
	// A label containing Wiki-Link syntax cannot add a second relation or close
	// the reference. The actual document title is retained in its own field.
	label := strings.NewReplacer("[", "（", "]", "）", "|", "·", "\n", " ", "\r", " ").Replace(title)
	return markdown[:start] + "\n\n[[" + childID + "|" + label + "]]\n\n" + markdown[end:]
}
