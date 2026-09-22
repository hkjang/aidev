package server

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

func collaborationHTML(n *collaborationNode) (string, error) {
	var b strings.Builder
	for _, child := range n.Children {
		part, err := collaborationHTML(child)
		if err != nil {
			return "", err
		}
		b.WriteString(part)
	}
	body := b.String()
	if n.Type == "text" {
		body = html.EscapeString(n.Text)
		keys := make([]string, 0, len(n.Marks))
		for k := range n.Marks {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if n.Marks[key] == nil {
				continue
			}
			kind := strings.Split(key, "--")[0]
			if kind == "link" {
				a, ok := n.Marks[key].(map[string]any)
				if !ok {
					return "", collaborationError("invalid_update", "링크 형식을 확인하세요")
				}
				u := str(a, "href")
				if _, err := collaborationURL(u); err != nil {
					return "", err
				}
				body = `<a href="` + html.EscapeString(u) + `">` + body + `</a>`
			} else {
				tag := map[string]string{"bold": "strong", "italic": "em", "underline": "u", "strike": "s", "code": "code"}[kind]
				if tag == "" {
					return "", collaborationError("unsupported_node", "지원하지 않는 표 텍스트 서식입니다")
				}
				body = "<" + tag + ">" + body + "</" + tag + ">"
			}
		}
		return body, nil
	}
	if n.Type == "inline" {
		return body, nil
	}
	a := n.Attrs
	attr := func(key, value string) string { return " " + key + `="` + html.EscapeString(value) + `"` }
	tag := map[string]string{"paragraph": "p", "blockquote": "blockquote", "bulletList": "ul", "orderedList": "ol", "listItem": "li", "taskList": "ul", "taskItem": "li", "table": "table", "tableRow": "tr", "tableCell": "td", "tableHeader": "th", "details": "details", "detailsSummary": "summary", "detailsContent": "div", "columns": "div", "column": "div", "callout": "aside", "footnoteDefinition": "aside", "footnoteReference": "sup", "syncedEmbed": "div", "bookmark": "div", "inlineMath": "span", "blockMath": "div"}[n.Type]
	props := ""
	switch n.Type {
	case "heading":
		level := collaborationNumber(a, "level", 1)
		if level < 1 || level > 6 {
			return "", collaborationError("invalid_update", "제목 수준을 확인하세요")
		}
		tag = fmt.Sprintf("h%d", level)
	case "codeBlock":
		return `<pre><code class="language-` + html.EscapeString(str(a, "language")) + `">` + html.EscapeString(collaborationPlainText(n)) + `</code></pre>`, nil
	case "hardBreak":
		return "<br>", nil
	case "horizontalRule":
		return "<hr>", nil
	case "image":
		if _, err := collaborationURL(str(a, "src")); err != nil {
			return "", err
		}
		return "<img" + attr("src", str(a, "src")) + attr("alt", str(a, "alt")) + attr("title", str(a, "title")) + ">", nil
	case "tableCell", "tableHeader":
		for _, key := range []string{"colspan", "rowspan"} {
			v := collaborationNumber(a, key, 1)
			if v < 1 || v > 1000 {
				return "", collaborationError("invalid_update", "표 병합 범위를 확인하세요")
			}
			if v > 1 {
				props += attr(key, fmt.Sprint(v))
			}
		}
		if widths, ok := a["colwidth"].([]any); ok && len(widths) > 0 {
			values := []string{}
			for _, width := range widths {
				v := collaborationNumber(map[string]any{"v": width}, "v", 0)
				if v < 0 || v > 10000 {
					return "", collaborationError("invalid_update", "표 열 너비를 확인하세요")
				}
				values = append(values, fmt.Sprint(v))
			}
			props += attr("colwidth", strings.Join(values, ","))
		}
	case "orderedList":
		props += attr("start", fmt.Sprint(collaborationNumber(a, "start", 1)))
	case "taskList":
		props += attr("data-type", "taskList")
	case "taskItem":
		props += attr("data-type", "taskItem") + attr("data-checked", fmt.Sprint(boolean(a, "checked")))
	case "detailsContent":
		props += attr("data-type", "detailsContent")
	case "columns":
		props += attr("data-madi-columns", "") + attr("data-count", fmt.Sprint(collaborationNumber(a, "count", 2)))
	case "column":
		props += attr("data-madi-column", "")
	case "callout":
		props += attr("data-madi-callout", "") + attr("data-type", str(a, "type")) + attr("data-title", str(a, "title"))
	case "inlineMath", "blockMath":
		props += attr("data-madi-math", map[bool]string{true: "inline", false: "block"}[n.Type == "inlineMath"]) + attr("data-latex", str(a, "latex"))
	case "footnoteDefinition":
		props += attr("data-madi-footnote-definition", "") + attr("data-label", str(a, "label"))
	case "footnoteReference":
		props += attr("data-madi-footnote", "") + attr("data-label", str(a, "label"))
	case "syncedEmbed":
		props += attr("data-madi-embed", "") + attr("data-documentId", str(a, "documentId")) + attr("data-blockId", str(a, "blockId")) + attr("data-label", str(a, "label"))
	case "bookmark":
		props += attr("data-madi-bookmark", "") + attr("data-url", str(a, "url")) + attr("data-title", str(a, "title")) + attr("data-description", str(a, "description"))
	}
	if alignment := str(a, "textAlign"); alignment != "" {
		if !oneOf(alignment, "left", "center", "right", "justify") {
			return "", collaborationError("invalid_update", "문단 정렬을 확인하세요")
		}
		props += attr("style", "text-align:"+alignment)
	}
	if tag == "" {
		return "", collaborationError("unsupported_node", "HTML 표에서 지원하지 않는 블록입니다: "+n.Type)
	}
	return "<" + tag + props + ">" + body + "</" + tag + ">", nil
}

func collaborationTableNeedsHTML(n *collaborationNode) bool {
	if collaborationNumber(n.Attrs, "colspan", 1) > 1 || collaborationNumber(n.Attrs, "rowspan", 1) > 1 || str(n.Attrs, "textAlign") != "" {
		return true
	}
	if widths, ok := n.Attrs["colwidth"].([]any); ok && len(widths) > 0 {
		return true
	}
	if n.Type == "tableCell" || n.Type == "tableHeader" {
		if len(n.Children) != 1 || n.Children[0].Type != "paragraph" {
			return true
		}
	}
	for _, child := range n.Children {
		if collaborationTableNeedsHTML(child) {
			return true
		}
	}
	if n.Type == "table" {
		for i, row := range n.Children {
			for _, cell := range row.Children {
				if (i == 0) != (cell.Type == "tableHeader") {
					return true
				}
			}
		}
	}
	return false
}
