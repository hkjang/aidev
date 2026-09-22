package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
)

type migrationCSV struct {
	Header []string
	Rows   [][]string
}

func parseMigrationCSV(raw []byte) (migrationCSV, error) {
	out := migrationCSV{}
	if !utf8.Valid(raw) {
		return out, errors.New("CSV 파일은 UTF-8 인코딩이어야 합니다")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff")))
	header, e := reader.Read()
	if e != nil || len(header) == 0 || len(header) > 100 {
		return out, errors.New("CSV 첫 행에 1~100개 열 이름이 필요합니다")
	}
	seen := map[string]bool{}
	for i, h := range header {
		h = strings.TrimSpace(h)
		if h == "" || len(h) > 100 || seen[h] {
			return out, errors.New("CSV 열 이름은 비어 있거나 중복될 수 없습니다(최대 100바이트)")
		}
		header[i] = h
		seen[h] = true
	}
	out.Header = header
	for {
		row, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return out, errors.New("CSV 열 수와 따옴표 형식을 확인하세요")
		}
		if len(out.Rows) >= 5000 {
			return out, errors.New("CSV 가져오기는 한 번에 최대 5,000행입니다")
		}
		for _, v := range row {
			if len(v) > 64<<10 {
				return out, errors.New("CSV 셀은 64KB 이하여야 합니다")
			}
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// HTML is parsed as data, never executed or remotely fetched. Inline handlers,
// script/style/iframe content, unsafe links and tracking images are discarded.
func migrationHTML(raw []byte) (string, string, error) {
	return migrationHTMLWithAssets(raw, "", nil, nil)
}

// Only assets already present in the uploaded archive may become images. Remote
// sources, data URLs and executable HTML are never fetched or preserved.
func migrationHTMLWithAssets(raw []byte, filename string, files map[string][]byte, renamed map[string]string) (string, string, error) {
	if len(raw) > 4<<20 || !utf8.Valid(raw) {
		return "", "", errors.New("HTML 문서는 UTF-8 형식의 4MB 이하 파일이어야 합니다")
	}
	doc, e := html.Parse(bytes.NewReader(raw))
	if e != nil {
		return "", "", errors.New("HTML 문서를 읽을 수 없습니다")
	}
	// Bound the full tree before any helper traverses it (including <pre/title>).
	var check func(*html.Node, int) error
	count := 0
	check = func(n *html.Node, depth int) error {
		count++
		if count > 100000 || depth > 100 {
			return errors.New("HTML 구조가 너무 복잡합니다")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := check(c, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if e = check(doc, 0); e != nil {
		return "", "", e
	}
	var out strings.Builder
	title := ""
	nodes := 0
	var visit func(*html.Node, int) error
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
	visit = func(n *html.Node, depth int) error {
		nodes++
		if nodes > 100000 || depth > 100 {
			return errors.New("HTML 구조가 너무 복잡합니다")
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
			return nil
		}
		if n.Type != html.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if e := visit(c, depth+1); e != nil {
					return e
				}
			}
			return nil
		}
		switch n.Data {
		case "script", "style", "iframe", "object", "embed", "svg", "noscript", "template":
			return nil
		case "img":
			target := resolveVaultFile(files, filename, attr(n, "src"))
			if target != "" && oneOf(strings.ToLower(path.Ext(target)), ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif") {
				out.WriteString("![" + strings.NewReplacer("[", "\\[", "]", "\\]", "\n", " ").Replace(attr(n, "alt")) + "](<" + vaultEscapedPath(migrationRelativePath(filename, target)) + ">)")
			}
			return nil
		case "title":
			title = strings.TrimSpace(plain(n))
			return nil
		case "h1", "h2", "h3", "h4", "h5", "h6":
			out.WriteString("\n\n" + strings.Repeat("#", int(n.Data[1]-'0')) + " ")
		case "p", "div", "section", "article", "blockquote", "table", "ul", "ol":
			out.WriteString("\n\n")
		case "li":
			out.WriteString("\n- ")
		case "br":
			out.WriteByte('\n')
		case "strong", "b":
			out.WriteString("**")
		case "em", "i":
			out.WriteString("*")
		case "pre":
			out.WriteString("\n\n````\n" + strings.ReplaceAll(plain(n), "````", "` ` ` `") + "\n````\n\n")
			return nil
		case "code":
			out.WriteString("`" + strings.ReplaceAll(plain(n), "`", "'") + "`")
			return nil
		case "tr":
			out.WriteString("\n| ")
		case "a":
			destination := attr(n, "href")
			if target := resolveVaultFile(files, filename, destination); target != "" {
				if replacement := renamed[target]; replacement != "" {
					target = replacement
				}
				u, _ := url.Parse(destination)
				destination = vaultEscapedPath(migrationRelativePath(filename, target))
				if u != nil && u.Fragment != "" {
					destination += "#" + url.PathEscape(u.Fragment)
				}
			}
			u, e := url.Parse(destination)
			if e == nil && (u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto" || u.Scheme == "") && !strings.ContainsAny(destination, "\n\r\x00") {
				out.WriteString("[" + strings.ReplaceAll(plain(n), "]", "\\]") + "](<" + strings.ReplaceAll(destination, ">", "%3E") + ">)")
				return nil
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if e := visit(c, depth+1); e != nil {
				return e
			}
		}
		switch n.Data {
		case "strong", "b":
			out.WriteString("**")
		case "em", "i":
			out.WriteString("*")
		case "td", "th":
			out.WriteString(" | ")
		case "p", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
			out.WriteString("\n\n")
		}
		if out.Len() > 4<<20 {
			return errors.New("변환한 Markdown이 4MB를 초과합니다")
		}
		return nil
	}
	if e = visit(doc, 0); e != nil {
		return "", "", e
	}
	return title, strings.TrimSpace(out.String()), nil
}

func prepareMigrationInput(name, format string, raw []byte) (*vaultInput, *migrationCSV, error) {
	if len(raw) > 50<<20 {
		return nil, nil, errors.New("가져오기 파일은 50MB 이하여야 합니다")
	}
	switch format {
	case "csv":
		v, e := parseMigrationCSV(raw)
		return nil, &v, e
	case "html":
		if strings.EqualFold(path.Ext(name), ".zip") {
			return prepareMigrationInput(name, "notion", raw)
		}
		title, markdown, e := migrationHTML(raw)
		if e != nil {
			return nil, nil, e
		}
		if title == "" {
			title = strings.TrimSuffix(path.Base(name), path.Ext(name))
		}
		return &vaultInput{files: map[string][]byte{"import.md": []byte(migrationFrontMatter(title, markdown))}}, nil, nil
	case "json":
		var formatProbe struct {
			Format string `json:"format"`
		}
		if json.Unmarshal(raw, &formatProbe) == nil && formatProbe.Format != "" {
			input, e := parseJSONVault(raw)
			return input, nil, e
		}
		var envelope struct {
			Documents []map[string]any `json:"documents"`
		}
		var docs []map[string]any
		if json.Unmarshal(raw, &docs) != nil {
			if json.Unmarshal(raw, &envelope) != nil {
				return nil, nil, errors.New("JSON 문서 배열 또는 documents 배열 객체가 필요합니다")
			}
			docs = envelope.Documents
		}
		if len(docs) == 0 || len(docs) > 5000 {
			return nil, nil, errors.New("JSON 가져오기는 1~5,000개 문서입니다")
		}
		input := &vaultInput{files: map[string][]byte{}}
		for i, d := range docs {
			title := str(d, "title")
			markdown := str(d, "markdown")
			if title == "" || len(title) > 300 || len(markdown) > 4<<20 {
				return nil, nil, errors.New("JSON 문서에 title(300바이트 이하)과 markdown(4MB 이하)이 필요합니다")
			}
			for key := range d {
				if !oneOf(key, "title", "markdown", "tags", "aliases") {
					return nil, nil, errors.New("JSON 문서 속성은 title, markdown, tags, aliases만 지원합니다")
				}
			}
			content := migrationFrontMatter(title, markdown)
			if len(listStrings(d["tags"])) > 0 || len(listStrings(d["aliases"])) > 0 {
				tags, _ := json.Marshal(listStrings(d["tags"]))
				aliases, _ := json.Marshal(listStrings(d["aliases"]))
				content = strings.Replace(content, "---\n\n", "tags: "+string(tags)+"\naliases: "+string(aliases)+"\n---\n\n", 1)
			}
			input.files[fmt.Sprintf("%04d.md", i+1)] = []byte(content)
		}
		return input, nil, nil
	case "markdown", "obsidian", "notion":
		input, e := parseVaultInput(name, raw)
		if e != nil {
			return nil, nil, e
		}
		// Notion HTML exports can be supplied as ZIP as well as Markdown exports.
		if format == "notion" {
			input.databases = map[string]migrationCSV{}
			for file, data := range input.files {
				if strings.EqualFold(path.Ext(file), ".csv") {
					csv, e := parseMigrationCSV(data)
					if e != nil {
						return nil, nil, fmt.Errorf("%s: %w", file, e)
					}
					if len(input.databases) >= 20 {
						return nil, nil, errors.New("Notion CSV 데이터베이스는 한 번에 최대 20개입니다")
					}
					input.databases[file] = csv
				}
			}
			renamed := map[string]string{}
			for file := range input.files {
				if oneOf(strings.ToLower(path.Ext(file)), ".html", ".htm") {
					renamed[file] = strings.TrimSuffix(file, path.Ext(file)) + ".md"
				}
			}
			converted := map[string][]byte{}
			for file, data := range input.files {
				if oneOf(strings.ToLower(path.Ext(file)), ".html", ".htm") {
					title, md, e := migrationHTMLWithAssets(data, file, input.files, renamed)
					if e != nil {
						return nil, nil, e
					}
					if title == "" {
						title = strings.TrimSuffix(path.Base(file), path.Ext(file))
					}
					target := strings.TrimSuffix(file, path.Ext(file)) + ".md"
					if _, exists := input.files[target]; exists {
						return nil, nil, errors.New("Notion HTML/Markdown 변환 경로가 중복됩니다")
					}
					converted[target] = []byte(migrationFrontMatter(title, md))
				}
			}
			for file := range renamed {
				delete(input.files, file)
			}
			for file, data := range converted {
				input.files[file] = data
			}
			// Preserve original CSV as an attachment and create its database too.
			// A CSV-only export still needs a private source page to own files.
			if len(input.databases) > 0 {
				hasDoc := false
				for file := range input.files {
					if oneOf(strings.ToLower(path.Ext(file)), ".md", ".markdown") {
						hasDoc = true
					}
				}
				if !hasDoc {
					input.files["가져온 데이터베이스.md"] = []byte(migrationFrontMatter("가져온 데이터베이스", "Notion CSV 원본과 가져온 데이터베이스입니다."))
				}
			}
		}
		return input, nil, nil
	default:
		return nil, nil, errors.New("지원하지 않는 가져오기 형식입니다")
	}
}

func migrationRelativePath(from, to string) string {
	dir := strings.Split(path.Dir(from), "/")
	if path.Dir(from) == "." {
		dir = nil
	}
	target := strings.Split(to, "/")
	for len(dir) > 0 && len(target) > 0 && dir[0] == target[0] {
		dir, target = dir[1:], target[1:]
	}
	return strings.Repeat("../", len(dir)) + strings.Join(target, "/")
}
func migrationFrontMatter(title, markdown string) string {
	quoted, _ := json.Marshal(title)
	return "---\ntitle: " + string(quoted) + "\nvisibility: private\n---\n\n" + markdown
}
