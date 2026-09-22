package server

import (
	"bytes"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type ragChunk struct {
	Index     int    `json:"index"`
	Start     int    `json:"start_byte"`
	End       int    `json:"end_byte"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Heading   string `json:"heading"`
	Content   string `json:"content"`
	Hash      string `json:"hash"`
}

// Store exact source slices and positions. The server can later verify every
// citation against the same document version, not a model-invented quotation.
// Bound by UTF-8 bytes, not an assumed token/character ratio for Korean or code.
func chunkMarkdown(markdown string, maxBytes, overlapBytes int) ([]ragChunk, error) {
	if !utf8.ValidString(markdown) || maxBytes < 256 || maxBytes > 8192 || overlapBytes < 0 || overlapBytes >= maxBytes/2 {
		return nil, errors.New("문서 인코딩 또는 검색 조각 크기를 확인하세요")
	}
	if len(markdown) > 4<<20 {
		return nil, errors.New("검색 대상 문서는 4MiB 이하여야 합니다")
	}
	source := []byte(markdown)
	masked := markdownBodySource(markdown)
	start := len(masked) - len(bytes.TrimLeft(masked, " \t\r\n"))
	if start == len(source) {
		return []ragChunk{}, nil
	}
	// Keep indentation on the first content line: it can mean a code block.
	start = bytes.LastIndexByte(source[:start], '\n') + 1
	newlines := []int{}
	for i, b := range source {
		if b == '\n' {
			newlines = append(newlines, i)
		}
	}
	// Parse headings only. Full document indexing also extracts tasks/links and
	// must not run for each RAG projection merely to discover section titles.
	headings := []markdownHeading{}
	parsed := goldmark.New().Parser().Parse(text.NewReader(masked))
	_ = ast.Walk(parsed, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok && h.Lines().Len() > 0 {
			headings = append(headings, markdownHeading{Level: h.Level, Text: string(h.Text(masked)), Line: sort.SearchInts(newlines, h.Lines().At(0).Start)})
		}
		return ast.WalkContinue, nil
	})
	headingIndex := -1
	out := []ragChunk{}
	for start < len(source) {
		end := min(len(source), start+maxBytes)
		for end < len(source) && !utf8.RuneStart(source[end]) {
			end--
		}
		if end < len(source) {
			lower := start + (end-start)/2
			if n := bytes.LastIndex(source[lower:end], []byte("\n\n")); n >= 0 {
				end = lower + n + 2
			} else if n = bytes.LastIndexByte(source[lower:end], '\n'); n >= 0 {
				end = lower + n + 1
			}
		}
		part := string(source[start:end])
		if strings.TrimSpace(part) != "" {
			first := sort.SearchInts(newlines, start) + 1
			last := sort.SearchInts(newlines, end) + 1
			if end > start && source[end-1] == '\n' {
				last--
			}
			for headingIndex+1 < len(headings) && headings[headingIndex+1].Line+1 <= first {
				headingIndex++
			}
			heading := ""
			if headingIndex >= 0 {
				heading = headings[headingIndex].Text
			}
			out = append(out, ragChunk{Index: len(out), Start: start, End: end, StartLine: first, EndLine: last, Heading: heading, Content: part, Hash: digest(part)})
		}
		if end == len(source) {
			break
		}
		next := max(start+1, end-overlapBytes)
		for next < end && !utf8.RuneStart(source[next]) {
			next++
		}
		start = next
	}
	return out, nil
}
