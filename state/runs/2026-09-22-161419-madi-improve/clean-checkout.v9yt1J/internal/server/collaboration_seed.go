package server

import (
	"bytes"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	nethtml "golang.org/x/net/html"
)

type collaborationComparisonWriter struct{ bytes.Buffer }

func (w *collaborationComparisonWriter) Write(value []byte) (int, error) {
	if w.Len()+len(value) > 8<<20 {
		return 0, errors.New("comparison size limit")
	}
	return w.Buffer.Write(value)
}
func (w *collaborationComparisonWriter) WriteString(value string) (int, error) {
	return w.Write([]byte(value))
}
func (w *collaborationComparisonWriter) WriteByte(value byte) error {
	_, e := w.Write([]byte{value})
	return e
}

// Compare CommonMark/GFM rendering only inside the server; this HTML is never
// sent to a browser. WithUnsafe is essential here: omitting raw HTML would make
// two different imported HTML blocks look deceptively equal. Non-equivalent
// custom syntax fails closed to the source editor instead of rewriting it.
func collaborationSeedEquivalent(original, projected string) bool {
	if strings.TrimRight(original, "\r\n") == strings.TrimRight(projected, "\r\n") {
		return true
	}
	parser := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(html.WithUnsafe()))
	var a, b collaborationComparisonWriter
	if parser.Convert([]byte(original), &a) != nil || parser.Convert([]byte(projected), &b) != nil {
		return false
	}
	if bytes.Equal(a.Bytes(), b.Bytes()) {
		return true
	}
	// HTML import may reorder attributes or insert a tbody. Compare parsed DOM
	// structure while retaining all actual attributes, text and comments.
	x, ok := collaborationCanonicalHTML(a.String())
	if !ok {
		return false
	}
	y, ok := collaborationCanonicalHTML(b.String())
	return ok && x == y
}

func collaborationCanonicalHTML(source string) (string, bool) {
	doc, e := nethtml.Parse(strings.NewReader(source))
	if e != nil {
		return "", false
	}
	var out strings.Builder
	count := 0
	var visit func(*nethtml.Node, int) bool
	visit = func(n *nethtml.Node, depth int) bool {
		count++
		if count > 100000 || depth > 100 {
			return false
		}
		if n.Type == nethtml.TextNode && strings.TrimSpace(n.Data) == "" && n.Parent != nil && oneOf(n.Parent.Data, "html", "head", "body", "table", "tbody", "thead", "tfoot", "tr") {
			return true
		}
		// JSON escaping creates an unambiguous comparison token stream.
		out.Write(jsonValue([]any{int(n.Type), n.Data, n.Namespace}))
		attrs := append([]nethtml.Attribute{}, n.Attr...)
		if n.Type == nethtml.ElementNode && n.Data == "table" {
			widths, ok := collaborationTableColumnWidths(n)
			if !ok {
				return false
			}
			out.Write(jsonValue(widths))
		}
		if n.Type == nethtml.ElementNode && oneOf(n.Data, "td", "th") {
			filtered := []nethtml.Attribute{}
			for _, a := range attrs {
				if a.Key != "colwidth" {
					filtered = append(filtered, a)
				}
			}
			attrs = filtered
		}
		sort.Slice(attrs, func(i, j int) bool {
			if attrs[i].Namespace == attrs[j].Namespace {
				return attrs[i].Key < attrs[j].Key
			}
			return attrs[i].Namespace < attrs[j].Namespace
		})
		out.Write(jsonValue(attrs))
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !visit(c, depth+1) {
				return false
			}
		}
		out.WriteByte(']')
		return true
	}
	if !visit(doc, 0) {
		return "", false
	}
	return out.String(), true
}

// TipTap propagates a header's colwidth to cells in subsequent rows. That is
// derived table geometry, not a user edit; compare the effective column widths
// and retain every original attribute in Markdown until the first real edit.
func collaborationTableColumnWidths(table *nethtml.Node) ([]int, bool) {
	rows := []*nethtml.Node{}
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == nethtml.ElementNode && c.Data == "table" {
				continue
			}
			if c.Type == nethtml.ElementNode && c.Data == "tr" {
				rows = append(rows, c)
			} else {
				walk(c)
			}
		}
	}
	walk(table)
	if len(rows) > 10000 {
		return nil, false
	}
	widths, remaining := make([]int, 1000), make([]int, 1000)
	maxColumn := 0
	for _, row := range rows {
		column := 0
		for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
			if cell.Type != nethtml.ElementNode || !oneOf(cell.Data, "td", "th") {
				continue
			}
			for column < 1000 && remaining[column] > 0 {
				column++
			}
			attrs := map[string]string{}
			for _, a := range cell.Attr {
				attrs[a.Key] = a.Val
			}
			span, height := 1, 1
			var e error
			if attrs["colspan"] != "" {
				span, e = strconv.Atoi(attrs["colspan"])
				if e != nil {
					return nil, false
				}
			}
			if attrs["rowspan"] != "" {
				height, e = strconv.Atoi(attrs["rowspan"])
				if e != nil {
					return nil, false
				}
			}
			if span < 1 || height < 1 || height > 1000 || column+span > 1000 {
				return nil, false
			}
			for i := column; i < column+span; i++ {
				if remaining[i] > 0 {
					return nil, false
				}
				remaining[i] = height
			}
			if raw, ok := attrs["colwidth"]; ok {
				parts := strings.Split(raw, ",")
				if len(parts) != span {
					return nil, false
				}
				for i, part := range parts {
					v, e := strconv.Atoi(strings.TrimSpace(part))
					if e != nil || v < 0 || v > 10000 {
						return nil, false
					}
					if widths[column+i] != 0 && v != 0 && widths[column+i] != v {
						return nil, false
					}
					if v != 0 {
						widths[column+i] = v
					}
				}
			}
			column += span
			maxColumn = max(maxColumn, column)
		}
		for i := range remaining {
			if remaining[i] > 0 {
				remaining[i]--
			}
		}
	}
	return widths[:maxColumn], true
}
