package server

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/reearth/ygo/crdt"
)

// lib0 ReadAny uses int64 for compact integers while HTTP JSON uses float64.
// Normalize XML attributes independently of the HTTP-only number helper.
func collaborationNumber(attrs map[string]any, key string, fallback int) int {
	switch value := attrs[key].(type) {
	case int:
		return value
	case int64:
		if value >= -1<<31 && value <= 1<<31-1 {
			return int(value)
		}
	case uint64:
		if value <= 1<<31-1 {
			return int(value)
		}
	case float64:
		if value >= -1<<31 && value <= 1<<31-1 {
			return int(value)
		}
	}
	return fallback
}

// This is deliberately a schema-aware projection, not an HTML stripper. Unknown
// schema nodes fail closed instead of silently dropping an author's content.
// Add new editor nodes to collaborationRenderExtension alongside their browser
// Markdown parser/renderer and round-trip fixture.
type collaborationNode struct {
	Type     string
	Attrs    map[string]any
	Children []*collaborationNode
	Text     string
	Marks    map[string]any
}

func collaborationMarkdown(root *crdt.YXmlFragment) (string, map[string]any, error) {
	count := 0
	blocks := []map[string]any{}
	blockIDs := map[string]bool{}
	var walk func(any, int) (*collaborationNode, error)
	walk = func(raw any, depth int) (*collaborationNode, error) {
		count++
		if count > 100000 || depth > 100 {
			return nil, collaborationError("document_limit", "문서 블록 수 또는 중첩 깊이를 초과했습니다")
		}
		switch item := raw.(type) {
		case *crdt.YXmlElement:
			if len(item.NodeName) > 128 {
				return nil, collaborationError("invalid_update", "블록 이름이 너무 깁니다")
			}
			n := &collaborationNode{Type: item.NodeName, Attrs: item.GetAttributeValues()}
			blockIndex := -1
			if rawID := n.Attrs["id"]; rawID != nil {
				if _, ok := rawID.(string); !ok {
					return nil, collaborationError("invalid_update", "블록 ID 형식을 확인하세요")
				}
			}
			if id, ok := n.Attrs["id"].(string); ok && id != "" {
				if len(id) > 128 {
					return nil, collaborationError("invalid_update", "블록 ID가 너무 깁니다")
				}
				if blockIDs[id] {
					return nil, collaborationError("invalid_update", "중복된 블록 ID가 있습니다")
				}
				blockIDs[id] = true
				if !oneOf(n.Type, "inlineMath", "footnoteReference", "hardBreak") {
					blockIndex = len(blocks)
					blocks = append(blocks, map[string]any{"id": id, "type": n.Type})
				}
			}
			for _, child := range item.Children() {
				v, e := walk(child, depth+1)
				if e != nil {
					return nil, e
				}
				n.Children = append(n.Children, v)
			}
			if blockIndex >= 0 {
				text := collaborationPlainText(n)
				runes := []rune(text)
				if len(runes) > 200 {
					text = string(runes[:200])
				}
				blocks[blockIndex]["text"] = text
			}
			return n, nil
		case *crdt.YXmlText:
			n := &collaborationNode{Type: "inline"}
			for _, d := range item.ToDelta() {
				text, ok := d.Insert.(string)
				if !ok {
					return nil, collaborationError("unsupported_node", "텍스트 내부 임베드는 지원되는 이미지/파일 블록으로 변환하세요")
				}
				if !utf8.ValidString(text) {
					return nil, collaborationError("invalid_update", "문자 인코딩을 확인하세요")
				}
				n.Children = append(n.Children, &collaborationNode{Type: "text", Text: text, Marks: d.Attributes})
			}
			return n, nil
		default:
			return nil, collaborationError("unsupported_node", "지원되지 않는 XML 편집 노드입니다")
		}
	}
	parts := []string{}
	for _, raw := range root.Children() {
		n, e := walk(raw, 0)
		if e != nil {
			return "", nil, e
		}
		part, e := collaborationRender(n)
		if e != nil {
			return "", nil, e
		}
		parts = append(parts, part)
	}
	metadata := map[string]any{"blocks": blocks}
	if len(jsonValue(metadata)) > 1<<20 {
		return "", nil, collaborationError("document_limit", "블록 메타데이터는 1MB 이하여야 합니다")
	}
	return strings.Join(parts, "\n\n"), metadata, nil
}

func collaborationPlainText(n *collaborationNode) string {
	if n.Type == "text" {
		return n.Text
	}
	var out strings.Builder
	for _, child := range n.Children {
		out.WriteString(collaborationPlainText(child))
	}
	return out.String()
}

func collaborationChildren(n *collaborationNode, separator string) (string, error) {
	parts := make([]string, 0, len(n.Children))
	for _, child := range n.Children {
		part, e := collaborationRender(child)
		if e != nil {
			return "", e
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, separator), nil
}

func collaborationRender(n *collaborationNode) (string, error) {
	switch n.Type {
	case "text":
		return collaborationRenderText(n)
	case "inline":
		return collaborationChildren(n, "")
	case "paragraph":
		if str(n.Attrs, "textAlign") != "" {
			return collaborationHTML(n)
		}
		return collaborationChildren(n, "")
	case "heading":
		if str(n.Attrs, "textAlign") != "" {
			return collaborationHTML(n)
		}
		level := collaborationNumber(n.Attrs, "level", 1)
		if level < 1 || level > 6 {
			return "", collaborationError("unsupported_node", "제목 수준을 확인하세요")
		}
		body, e := collaborationChildren(n, "")
		return strings.Repeat("#", level) + " " + body, e
	case "blockquote":
		body, e := collaborationChildren(n, "\n\n")
		return "> " + strings.ReplaceAll(body, "\n", "\n> "), e
	case "bulletList", "orderedList", "taskList":
		parts := []string{}
		start := collaborationNumber(n.Attrs, "start", 1)
		for i, child := range n.Children {
			if child.Type != "listItem" && child.Type != "taskItem" {
				return "", collaborationError("unsupported_node", "목록 구조를 확인하세요")
			}
			body, e := collaborationChildren(child, "\n\n")
			if e != nil {
				return "", e
			}
			prefix := "- "
			if n.Type == "orderedList" {
				prefix = strconv.Itoa(start+i) + ". "
			}
			if n.Type == "taskList" {
				if boolean(child.Attrs, "checked") {
					prefix = "- [x] "
				} else {
					prefix = "- [ ] "
				}
			}
			parts = append(parts, prefix+strings.ReplaceAll(body, "\n", "\n"+strings.Repeat(" ", len(prefix))))
		}
		return strings.Join(parts, "\n"), nil
	case "listItem", "taskItem":
		return collaborationChildren(n, "\n\n")
	case "codeBlock":
		body := collaborationPlainText(n)
		language := str(n.Attrs, "language")
		if strings.ContainsAny(language, "\r\n`~") || len(language) > 128 {
			return "", collaborationError("unsupported_node", "코드 블록 언어를 확인하세요")
		}
		fence := strings.Repeat("`", max(3, collaborationLongestRun(body, '`')+1))
		return fence + language + "\n" + body + "\n" + fence, nil
	case "horizontalRule":
		return "---", nil
	case "hardBreak":
		return "  \n", nil
	case "image":
		src, e := collaborationURL(str(n.Attrs, "src"))
		if e != nil {
			return "", e
		}
		label := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "\n", " ").Replace(str(n.Attrs, "alt"))
		title := str(n.Attrs, "title")
		if title != "" {
			title = " \"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", " ").Replace(title) + "\""
		}
		return "![" + label + "](" + src + title + ")", nil
	case "table":
		return collaborationRenderTable(n)
	default:
		if rendered, handled, e := collaborationRenderExtension(n); handled {
			return rendered, e
		}
		return "", collaborationError("unsupported_node", fmt.Sprintf("공동 편집에서 지원하지 않는 블록입니다: %s", n.Type))
	}
}

func collaborationRenderText(n *collaborationNode) (string, error) {
	text := collaborationEscapeText(n.Text)
	keys := make([]string, 0, len(n.Marks))
	for k := range n.Marks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Inline code is literal; Markdown escaping inside it would alter the source.
	for _, key := range keys {
		if strings.Split(key, "--")[0] == "code" {
			fence := strings.Repeat("`", max(1, collaborationLongestRun(n.Text, '`')+1))
			text = n.Text
			if strings.HasPrefix(text, "`") || strings.HasSuffix(text, "`") || strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ") {
				text = " " + text + " "
			}
			text = fence + text + fence
		}
	}
	for _, key := range keys {
		if n.Marks[key] == nil {
			continue
		}
		switch strings.Split(key, "--")[0] {
		case "code":
		case "bold":
			text = "**" + text + "**"
		case "italic":
			text = "*" + text + "*"
		case "strike":
			text = "~~" + text + "~~"
		case "underline":
			text = "<u>" + text + "</u>"
		case "link":
			attrs, ok := n.Marks[key].(map[string]any)
			if !ok {
				return "", collaborationError("invalid_update", "링크 형식을 확인하세요")
			}
			href, e := collaborationURL(str(attrs, "href"))
			if e != nil {
				return "", e
			}
			text = "[" + text + "](" + href + ")"
		default:
			return "", collaborationError("unsupported_node", fmt.Sprintf("지원하지 않는 텍스트 서식입니다: %s", key))
		}
	}
	return text, nil
}

func collaborationLongestRun(s string, character byte) int {
	maxRun, run := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == character {
			run++
			maxRun = max(maxRun, run)
		} else {
			run = 0
		}
	}
	return maxRun
}

func collaborationEscapeText(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		// Preserve Markdown-first wiki links as actual links, including labels.
		if strings.HasPrefix(s[i:], "[[") {
			if end := strings.Index(s[i+2:], "]]"); end >= 0 && !strings.Contains(s[i+2:i+2+end], "\n") {
				out.WriteString(s[i : i+end+4])
				i += end + 3
				continue
			}
		}
		if strings.ContainsRune("\\`*_{}[]<>~", rune(s[i])) {
			out.WriteByte('\\')
		}
		if (i == 0 || s[i-1] == '\n') && strings.ContainsRune("#-+", rune(s[i])) {
			out.WriteByte('\\')
		}
		out.WriteByte(s[i])
	}
	return out.String()
}

func collaborationURL(value string) (string, error) {
	if len(value) > 16384 || strings.ContainsAny(value, "\r\n\x00") {
		return "", collaborationError("invalid_update", "링크 주소를 확인하세요")
	}
	u, e := url.Parse(value)
	if e != nil {
		return "", collaborationError("invalid_update", "링크 주소를 확인하세요")
	}
	if u.Scheme != "" && !oneOf(strings.ToLower(u.Scheme), "https", "http", "mailto", "tel") {
		return "", collaborationError("invalid_update", "허용되지 않는 링크 프로토콜입니다")
	}
	return "<" + strings.NewReplacer("<", "%3C", ">", "%3E", " ", "%20").Replace(value) + ">", nil
}

func collaborationRenderTable(n *collaborationNode) (string, error) {
	if collaborationTableNeedsHTML(n) {
		return collaborationHTML(n)
	}
	rows := [][]string{}
	columns := 0
	for _, row := range n.Children {
		if row.Type != "tableRow" {
			return "", collaborationError("unsupported_node", "표 행 구조를 확인하세요")
		}
		cells := []string{}
		for _, cell := range row.Children {
			if cell.Type != "tableCell" && cell.Type != "tableHeader" {
				return "", collaborationError("unsupported_node", "표 셀 구조를 확인하세요")
			}
			body, e := collaborationChildren(cell, "<br>")
			if e != nil {
				return "", e
			}
			cells = append(cells, strings.ReplaceAll(strings.ReplaceAll(body, "|", "\\|"), "\n", "<br>"))
		}
		columns = max(columns, len(cells))
		rows = append(rows, cells)
	}
	if columns == 0 {
		return "", nil
	}
	lines := []string{}
	for i, row := range rows {
		for len(row) < columns {
			row = append(row, "")
		}
		lines = append(lines, "| "+strings.Join(row, " | ")+" |")
		if i == 0 {
			lines = append(lines, "|"+strings.Repeat(" --- |", columns))
		}
	}
	return strings.Join(lines, "\n"), nil
}

// Extension boundary for editor capabilities that retain Markdown-compatible
// source. Attributes are encoded, never injected as untrusted raw HTML.
func collaborationRenderExtension(n *collaborationNode) (string, bool, error) {
	switch n.Type {
	case "mermaid":
		return "```mermaid\n" + collaborationPlainText(n) + "\n```", true, nil
	case "inlineMath":
		if value := str(n.Attrs, "latex"); value == "" || strings.ContainsAny(value, "\r\n$") {
			body, e := collaborationHTML(n)
			return body, true, e
		}
		return "$" + str(n.Attrs, "latex") + "$", true, nil
	case "blockMath":
		if value := str(n.Attrs, "latex"); value == "" || strings.Contains("\n"+value+"\n", "\n$$\n") {
			body, e := collaborationHTML(n)
			return body, true, e
		}
		return "$$\n" + str(n.Attrs, "latex") + "\n$$", true, nil
	case "callout":
		kind := strings.ToLower(str(n.Attrs, "type"))
		if !oneOf(kind, "note", "tip", "important", "warning", "caution") {
			return "", true, collaborationError("invalid_update", "콜아웃 종류를 확인하세요")
		}
		title := str(n.Attrs, "title")
		if strings.ContainsAny(title, "\r\n") || len(title) > 500 {
			return "", true, collaborationError("invalid_update", "콜아웃 제목을 확인하세요")
		}
		if title != "" {
			title = " " + title
		}
		body, e := collaborationChildren(n, "\n\n")
		return "> [!" + strings.ToUpper(kind) + "]" + title + "\n> " + strings.ReplaceAll(body, "\n", "\n> "), true, e
	case "details":
		body, e := collaborationChildren(n, "\n\n")
		return ":::details\n\n" + body + "\n\n:::", true, e
	case "detailsSummary":
		body, e := collaborationChildren(n, "")
		return ":::detailsSummary\n\n" + body + "\n\n:::", true, e
	case "detailsContent":
		body, e := collaborationChildren(n, "\n\n")
		return ":::detailsContent\n\n" + body + "\n\n:::", true, e
	case "columns", "column":
		attrs := ""
		if n.Type == "columns" {
			count := collaborationNumber(n.Attrs, "count", 2)
			if count < 2 || count > 3 {
				return "", true, collaborationError("invalid_update", "열 레이아웃은 2~3열입니다")
			}
			attrs = fmt.Sprintf(" {count=\"%d\"}", count)
		}
		body, e := collaborationChildren(n, "\n\n")
		return ":::" + n.Type + attrs + "\n\n" + body + "\n\n:::", true, e
	case "footnoteReference", "footnoteDefinition":
		label := str(n.Attrs, "label")
		if label == "" || len(label) > 240 || strings.ContainsAny(label, "[]\r\n\t :") {
			return "", true, collaborationError("invalid_update", "각주 이름을 확인하세요")
		}
		if n.Type == "footnoteReference" {
			return "[^" + label + "]", true, nil
		}
		body, e := collaborationChildren(n, "\n\n")
		return "[^" + label + "]: " + strings.ReplaceAll(body, "\n", "\n    "), true, e
	case "syncedEmbed":
		id, block, label := str(n.Attrs, "documentId"), str(n.Attrs, "blockId"), str(n.Attrs, "label")
		if !validID(id) || len(block) > 128 || strings.ContainsAny(block, "[]|\r\n\t ") || len(label) > 500 || strings.ContainsAny(label, "]\r\n") {
			return "", true, collaborationError("invalid_update", "동기화 블록 참조를 확인하세요")
		}
		if block != "" {
			block = "#^" + block
		}
		if label != "" {
			label = "|" + label
		}
		return "![[" + id + block + label + "]]", true, nil
	case "bookmark":
		if _, e := collaborationURL(str(n.Attrs, "url")); e != nil {
			return "", true, e
		}
		attrs := []string{}
		encoder := strings.NewReplacer("&", "&amp;", "\"", "&quot;", "}", "&#125;", "\n", "&#10;", "\r", "&#13;")
		for _, key := range []string{"url", "title", "description"} {
			value := str(n.Attrs, key)
			if len(value) > 8000 {
				return "", true, collaborationError("invalid_update", "북마크 내용이 너무 깁니다")
			}
			attrs = append(attrs, key+"=\""+encoder.Replace(value)+"\"")
		}
		return ":::bookmark {" + strings.Join(attrs, " ") + "}\n\n\n\n:::", true, nil
	}
	return "", false, nil
}
