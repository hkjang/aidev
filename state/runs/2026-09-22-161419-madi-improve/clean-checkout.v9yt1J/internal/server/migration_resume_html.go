package server

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func migrationBackticks(v string) string {
	longest, run := 0, 0
	for _, ch := range v {
		if ch == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", longest+1)
}

// The quality converter retains GFM structure while keeping the original HTML
// available for side-by-side review. It never fetches remote images or executes
// imported HTML. Unsupported visual semantics are counted, not silently claimed.
func migrationHTMLQuality(raw []byte, filename string, files map[string][]byte) (string, string, map[string]int, error) {
	title, _, e := migrationHTMLWithAssets(raw, filename, files, nil)
	if e != nil {
		return "", "", nil, e
	}
	doc, e := html.Parse(bytes.NewReader(raw))
	if e != nil {
		return "", "", nil, e
	}
	loss := map[string]int{"removed_executable": 0, "remote_images_omitted": 0, "merged_cells_flattened": 0, "style_attributes_omitted": 0, "interactive_elements_flattened": 0}
	attr := func(n *html.Node, key string) string {
		for _, a := range n.Attr {
			if a.Key == key {
				return a.Val
			}
		}
		return ""
	}
	var plain func(*html.Node) string
	plain = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var b strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			b.WriteString(plain(c))
		}
		return b.String()
	}
	var render func(*html.Node, int) string
	children := func(n *html.Node, depth int) string {
		var b strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			b.WriteString(render(c, depth))
		}
		return b.String()
	}
	render = func(n *html.Node, depth int) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		if n.Type != html.ElementNode {
			return children(n, depth)
		}
		if attr(n, "style") != "" {
			loss["style_attributes_omitted"]++
		}
		switch n.Data {
		case "head", "title", "meta", "link":
			return ""
		case "script", "style", "iframe", "object", "embed", "svg", "noscript", "template":
			loss["removed_executable"]++
			return ""
		case "img":
			target := resolveVaultFile(files, filename, attr(n, "src"))
			if target == "" {
				loss["remote_images_omitted"]++
				return ""
			}
			if !oneOf(strings.ToLower(path.Ext(target)), ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif") {
				loss["remote_images_omitted"]++
				return ""
			}
			label := strings.NewReplacer("[", "\\[", "]", "\\]", "\n", " ").Replace(attr(n, "alt"))
			return "![" + label + "](<" + vaultEscapedPath(migrationRelativePath(filename, target)) + ">)"
		case "a":
			label := children(n, depth)
			target := attr(n, "href")
			u, err := url.Parse(target)
			if err != nil || u.User != nil || u.Scheme != "" && !oneOf(strings.ToLower(u.Scheme), "http", "https", "mailto") {
				return label
			}
			return "[" + label + "](<" + strings.ReplaceAll(target, ">", "%3E") + ">)"
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return "\n\n" + strings.Repeat("#", int(n.Data[1]-'0')) + " " + strings.TrimSpace(children(n, depth)) + "\n\n"
		case "p", "div", "section", "article", "main", "header", "footer":
			return "\n\n" + strings.TrimSpace(children(n, depth)) + "\n\n"
		case "br":
			return "  \n"
		case "hr":
			return "\n\n---\n\n"
		case "strong", "b":
			return "**" + children(n, depth) + "**"
		case "em", "i":
			return "*" + children(n, depth) + "*"
		case "s", "del", "strike":
			return "~~" + children(n, depth) + "~~"
		case "pre":
			code := plain(n)
			fence := migrationBackticks(code)
			if len(fence) < 3 {
				fence = "```"
			}
			return "\n\n" + fence + "\n" + code + "\n" + fence + "\n\n"
		case "code":
			code := plain(n)
			marker := migrationBackticks(code)
			return marker + " " + code + " " + marker
		case "blockquote":
			body := strings.TrimSpace(children(n, depth))
			return "\n\n> " + strings.ReplaceAll(body, "\n", "\n> ") + "\n\n"
		case "ul", "ol":
			var b strings.Builder
			b.WriteString("\n")
			ordinal := 1
			if start, err := strconv.Atoi(attr(n, "start")); err == nil && start > 0 {
				ordinal = start
			}
			for li := n.FirstChild; li != nil; li = li.NextSibling {
				if li.Type != html.ElementNode || li.Data != "li" {
					continue
				}
				prefix := "- "
				if n.Data == "ol" {
					prefix = fmt.Sprint(ordinal) + ". "
					ordinal++
				}
				body := strings.TrimSpace(children(li, depth+1))
				lines := strings.Split(body, "\n")
				b.WriteString(strings.Repeat("  ", depth) + prefix + lines[0] + "\n")
				for _, line := range lines[1:] {
					if strings.TrimSpace(line) != "" {
						b.WriteString(strings.Repeat("  ", depth+1) + line + "\n")
					}
				}
			}
			return b.String() + "\n"
		case "input":
			if strings.EqualFold(attr(n, "type"), "checkbox") {
				for _, a := range n.Attr {
					if a.Key == "checked" {
						return "[x] "
					}
				}
				return "[ ] "
			}
			loss["interactive_elements_flattened"]++
			return ""
		case "button", "select", "textarea", "form":
			loss["interactive_elements_flattened"]++
			return children(n, depth)
		case "details":
			loss["interactive_elements_flattened"]++
			return "\n\n" + children(n, depth) + "\n\n"
		case "summary":
			return "**" + children(n, depth) + "**\n\n"
		case "table":
			rows := [][]string{}
			var collect func(*html.Node)
			collect = func(parent *html.Node) {
				for c := parent.FirstChild; c != nil; c = c.NextSibling {
					if c.Type != html.ElementNode {
						continue
					}
					if c.Data == "tr" {
						cells := []string{}
						for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
							if cell.Type != html.ElementNode || !oneOf(cell.Data, "th", "td") {
								continue
							}
							if attr(cell, "colspan") != "" && attr(cell, "colspan") != "1" || attr(cell, "rowspan") != "" && attr(cell, "rowspan") != "1" {
								loss["merged_cells_flattened"]++
							}
							value := strings.TrimSpace(children(cell, depth))
							value = strings.NewReplacer("|", "\\|", "\r", " ", "\n", " ").Replace(value)
							cells = append(cells, value)
						}
						if len(cells) > 0 {
							rows = append(rows, cells)
						}
					} else if c.Data != "table" {
						collect(c)
					}
				}
			}
			collect(n)
			if len(rows) == 0 {
				return ""
			}
			width := 0
			for _, row := range rows {
				width = max(width, len(row))
			}
			var b strings.Builder
			b.WriteString("\n\n")
			for i, row := range rows {
				for len(row) < width {
					row = append(row, "")
				}
				b.WriteString("| " + strings.Join(row, " | ") + " |\n")
				if i == 0 {
					b.WriteString("|" + strings.Repeat(" --- |", width) + "\n")
				}
			}
			return b.String() + "\n"
		}
		return children(n, depth)
	}
	body := strings.TrimSpace(render(doc, 0)) + "\n"
	return title, body, loss, nil
}
