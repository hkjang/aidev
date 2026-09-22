package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
	parser "golang.org/x/net/html"
)

type transferJSONFile struct {
	Path string `json:"path"`
	Data []byte `json:"data_base64"`
}
type transferJSONVault struct {
	Format   string             `json:"format"`
	Version  int                `json:"version"`
	Manifest vaultManifest      `json:"manifest"`
	Files    []transferJSONFile `json:"files"`
}

func (s *Server) buildExportArtifact(ctx context.Context, p *Principal, r exportRun, maxBytes int64) ([]byte, string, string, error) {
	if r.Format == "csv" {
		data, e := s.buildCSVExport(ctx, r, maxBytes)
		return data, "madi-database.csv", "text/csv; charset=utf-8", e
	}
	f, e := os.CreateTemp("", "madi-transfer-*.zip")
	if e != nil {
		return nil, "", "", e
	}
	defer f.Close()
	defer os.Remove(f.Name())
	ctx = context.WithValue(ctx, principalKey, p)
	ctx = context.WithValue(ctx, vaultExportSelectionKey{}, vaultExportSelection{IDs: r.IDs, MaxBytes: maxBytes, RawMarkdown: r.Format != "portable" && r.Format != "html"})
	req := httptest.NewRequest("GET", "/api/v1/export?workspace_id="+r.WorkspaceID, nil).WithContext(ctx)
	res := &gitSyncFileResponse{header: http.Header{}, status: 200, writer: &gitLimitWriter{ctx: ctx, out: f, remaining: maxBytes}}
	s.exportMarkdown(res, req)
	if res.err != nil {
		return nil, "", "", res.err
	}
	if res.status != 200 {
		return nil, "", "", errors.New("선택한 문서·첨부의 내보내기 권한 또는 크기를 확인하세요")
	}
	stat, e := f.Stat()
	if e != nil {
		return nil, "", "", e
	}
	if oneOf(r.Format, "markdown", "portable") {
		_, e = f.Seek(0, io.SeekStart)
		if e != nil {
			return nil, "", "", e
		}
		data, e := io.ReadAll(io.LimitReader(f, maxBytes+1))
		return data, "madi-" + r.Format + ".zip", "application/zip", e
	}
	archive, e := zip.NewReader(f, stat.Size())
	if e != nil {
		return nil, "", "", e
	}
	if len(archive.File) > 5000 {
		return nil, "", "", errors.New("한 번에 최대 5,000개 파일을 내보낼 수 있습니다")
	}
	files := map[string][]byte{}
	var total int64
	for _, file := range archive.File {
		if !safeVaultPath(file.Name) || file.UncompressedSize64 > uint64(maxBytes) {
			return nil, "", "", errors.New("내보내기 파일 크기 또는 경로가 올바르지 않습니다")
		}
		reader, e := file.Open()
		if e != nil {
			return nil, "", "", e
		}
		data, e := io.ReadAll(io.LimitReader(contextVaultReader{ctx, reader}, maxBytes+1))
		reader.Close()
		total += int64(len(data))
		if e != nil {
			return nil, "", "", e
		}
		if total > maxBytes {
			return nil, "", "", errors.New("내보내기 전체 파일 한도를 초과했습니다")
		}
		files[file.Name] = data
	}
	var manifest vaultManifest
	if json.Unmarshal(files["madi-manifest.json"], &manifest) != nil {
		return nil, "", "", errors.New("내보내기 manifest가 올바르지 않습니다")
	}
	delete(files, "madi-manifest.json")
	if r.Format == "json" {
		out := transferJSONVault{Format: "madi-json-vault", Version: 1, Manifest: manifest, Files: []transferJSONFile{}}
		names := []string{}
		for name := range files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			out.Files = append(out.Files, transferJSONFile{Path: name, Data: files[name]})
		}
		data, e := json.Marshal(out)
		if int64(len(data)) > maxBytes {
			return nil, "", "", errors.New("JSON base64 인코딩 후 크기 한도를 초과했습니다")
		}
		return data, "madi-vault.json", "application/json", e
	}
	data, e := buildHTMLExport(manifest, files, maxBytes)
	return data, "madi-html.zip", "application/zip", e
}

func (s *Server) buildCSVExport(ctx context.Context, r exportRun, maxBytes int64) ([]byte, error) {
	db, e := s.one(ctx, `SELECT to_jsonb(d) FROM databases d WHERE id=$1`, r.DatabaseID)
	if e != nil {
		return nil, e
	}
	rows, e := s.rows(ctx, `SELECT to_jsonb(r) FROM database_rows r WHERE database_id=$1 ORDER BY created_at,id LIMIT 10001`, r.DatabaseID)
	if e != nil {
		return nil, e
	}
	if len(rows) > 10000 {
		return nil, errors.New("CSV는 최대 10,000행입니다")
	}
	props, _ := db["properties"].([]any)
	var b bytes.Buffer
	b.WriteString("\ufeff")
	writer := csv.NewWriter(&b)
	header := []string{}
	for _, raw := range props {
		p, _ := raw.(map[string]any)
		header = append(header, csvSafeCell(str(p, "name")))
	}
	if e = writer.Write(header); e != nil {
		return nil, e
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		values, _ := row["values"].(map[string]any)
		line := []string{}
		for _, raw := range props {
			p, _ := raw.(map[string]any)
			v := values[str(p, "id")]
			cell := ""
			switch x := v.(type) {
			case nil:
			case string:
				cell = x
			default:
				cell = string(jsonValue(x))
			}
			line = append(line, csvSafeCell(cell))
		}
		if e = writer.Write(line); e != nil {
			return nil, e
		}
		writer.Flush()
		if int64(b.Len()) > maxBytes {
			return nil, errors.New("CSV 크기 한도를 초과했습니다")
		}
	}
	writer.Flush()
	return b.Bytes(), writer.Error()
}
func csvSafeCell(s string) string {
	trimmed := strings.TrimLeft(s, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + s
	}
	return s
}

func buildHTMLExport(manifest vaultManifest, files map[string][]byte, maxBytes int64) ([]byte, error) {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	var expanded int64
	write := func(name string, data []byte) error {
		expanded += int64(len(data))
		if expanded > maxBytes {
			return errors.New("HTML 펼친 파일 합계가 내보내기 한도를 초과했습니다")
		}
		w, e := zw.Create(name)
		if e == nil {
			_, e = w.Write(data)
		}
		if int64(b.Len()) > maxBytes {
			return errors.New("HTML 내보내기 크기 한도를 초과했습니다")
		}
		return e
	}
	renderer := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(rendererhtml.WithUnsafe()))
	docLookup := map[string]*vaultDocument{}
	for _, d := range manifest.Documents {
		docLookup[strings.TrimSuffix(d.File, ".md")] = d
		docLookup[d.ID] = d
		docLookup[d.Title] = d
	}
	var index strings.Builder
	index.WriteString("<h1>madi 지식 보관함</h1><p>이 파일은 내보낸 시점의 읽기 전용 사본입니다. 원본 권한·만료·삭제는 이미 내려받은 사본에 적용되지 않습니다.</p><ul>")
	for _, d := range manifest.Documents {
		target := strings.TrimSuffix(d.File, path.Ext(d.File)) + ".html"
		md := string(files[d.File])
		_, md = collaborationFrontMatter(md)
		md = rewriteVaultContent(md, func(segment string) string {
			return vaultWikiPattern.ReplaceAllStringFunc(segment, func(match string) string {
				parts := vaultWikiPattern.FindStringSubmatch(match)
				name, fragment, label := splitVaultWiki(parts[2])
				linked := docLookup[name]
				if linked == nil {
					return match
				}
				if label == "" {
					label = linked.Title
				} else {
					label = strings.TrimPrefix(label, "|")
				}
				dest := migrationRelativePath(target, strings.TrimSuffix(linked.File, path.Ext(linked.File))+".html")
				return "[" + strings.ReplaceAll(label, "]", "\\]") + "](<" + vaultEscapedPath(dest) + fragment + ">)"
			})
		})
		md = rewriteVaultContent(md, func(segment string) string {
			return rewriteVaultMarkdownLinks(segment, func(dest string) string {
				if strings.HasSuffix(dest, ".md") {
					return strings.TrimSuffix(dest, ".md") + ".html"
				}
				return dest
			})
		})
		var body bytes.Buffer
		if e := renderer.Convert([]byte(md), &body); e != nil {
			return nil, e
		}
		// The allowlist sanitizer preserves editor HTML tables while removing all
		// scripts, active embeds, handlers and nonlocal image sources. CSP is defense in depth.
		safeBody, e := sanitizeExportHTML(body.String(), target, files)
		if e != nil {
			return nil, e
		}
		content := htmlExportPage(d.Title, safeBody)
		if e := write(target, []byte(content)); e != nil {
			return nil, e
		}
		index.WriteString(`<li><a href="` + html.EscapeString(vaultEscapedPath(target)) + `">` + html.EscapeString(d.Title) + `</a></li>`)
	}
	for _, a := range manifest.Attachments {
		if data, ok := files[a.File]; ok {
			if e := write(a.File, data); e != nil {
				return nil, e
			}
		}
	}
	index.WriteString("</ul>")
	if e := write("index.html", []byte(htmlExportPage("madi 지식 보관함", index.String()))); e != nil {
		return nil, e
	}
	if e := zw.Close(); e != nil {
		return nil, e
	}
	if int64(b.Len()) > maxBytes {
		return nil, errors.New("HTML 내보내기 크기 한도를 초과했습니다")
	}
	return b.Bytes(), nil
}

func sanitizeExportHTML(body, filename string, files map[string][]byte) (string, error) {
	doc, e := parser.Parse(strings.NewReader(body))
	if e != nil {
		return "", e
	}
	count := 0
	var bound func(*parser.Node, int) error
	bound = func(n *parser.Node, depth int) error {
		count++
		if count > 100000 || depth > 100 {
			return errors.New("HTML 구조 한도를 초과했습니다")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if e := bound(c, depth+1); e != nil {
				return e
			}
		}
		return nil
	}
	if e = bound(doc, 0); e != nil {
		return "", e
	}
	var visit func(*parser.Node)
	visit = func(n *parser.Node) {
		for child := n.FirstChild; child != nil; {
			next := child.NextSibling
			remove := false
			if child.Type == parser.ElementNode {
				if oneOf(child.Data, "script", "style", "iframe", "object", "embed", "svg", "math", "noscript", "template", "link", "meta", "base", "form", "button", "textarea", "select", "option", "audio", "video", "source") {
					n.RemoveChild(child)
					child = next
					continue
				}
				if !oneOf(child.Data, "html", "head", "body", "div", "span", "section", "article", "p", "br", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "blockquote", "pre", "code", "strong", "b", "em", "i", "s", "del", "u", "mark", "sub", "sup", "a", "img", "table", "thead", "tbody", "tfoot", "tr", "td", "th", "caption", "details", "summary", "input") {
					visit(child)
					for c := child.FirstChild; c != nil; {
						after := c.NextSibling
						child.RemoveChild(c)
						n.InsertBefore(c, child)
						c = after
					}
					n.RemoveChild(child)
					child = next
					continue
				}
				attrs := []parser.Attribute{}
				for _, a := range child.Attr {
					if a.Namespace != "" {
						continue
					}
					if oneOf(a.Key, "title", "id") || child.Data == "a" && a.Key == "href" || child.Data == "img" && oneOf(a.Key, "src", "alt") || oneOf(child.Data, "td", "th") && oneOf(a.Key, "colspan", "rowspan") || child.Data == "input" && oneOf(a.Key, "type", "checked") {
						if oneOf(a.Key, "colspan", "rowspan") {
							v, e := strconv.Atoi(a.Val)
							if e != nil || v < 1 || v > 100 {
								continue
							}
						}
						attrs = append(attrs, a)
					}
				}
				child.Attr = attrs
				if child.Data == "input" {
					checkbox := false
					for _, a := range child.Attr {
						if a.Key == "type" && a.Val == "checkbox" {
							checkbox = true
						}
					}
					if !checkbox {
						n.RemoveChild(child)
						child = next
						continue
					}
					child.Attr = append(child.Attr, parser.Attribute{Key: "disabled", Val: ""})
				}
			}
			if child.Type == parser.ElementNode && child.Data == "img" {
				var src string
				for _, a := range child.Attr {
					if a.Key == "src" {
						src = a.Val
					}
				}
				u, e := url.Parse(src)
				target := ""
				if e == nil && u.Scheme == "" && u.Host == "" && !strings.HasPrefix(u.Path, "/") {
					target = resolveVaultFile(files, filename, src)
				}
				if e == nil && u.Scheme == "" && u.Host == "" && strings.HasPrefix(u.Path, "/api/v1/attachments/") {
					id := strings.TrimPrefix(u.Path, "/api/v1/attachments/")
					if validID(id) {
						for name := range files {
							if strings.HasPrefix(name, "attachments/"+id+"/") {
								target = name
								break
							}
						}
					}
				}
				remove = target == "" || !oneOf(strings.ToLower(path.Ext(target)), ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif")
				if !remove {
					for i, a := range child.Attr {
						if a.Key == "src" {
							child.Attr[i].Val = vaultEscapedPath(migrationRelativePath(filename, target))
						}
					}
				}
			}
			if child.Type == parser.ElementNode && child.Data == "a" {
				for i, a := range child.Attr {
					if a.Key == "href" {
						u, e := url.Parse(a.Val)
						if e != nil || !oneOf(strings.ToLower(u.Scheme), "", "http", "https", "mailto") || u.Host != "" && u.Scheme == "" {
							child.Attr[i].Val = "#"
						}
					}
				}
				child.Attr = append(child.Attr, parser.Attribute{Key: "rel", Val: "noopener noreferrer"})
			}
			if remove {
				n.RemoveChild(child)
			} else {
				visit(child)
			}
			child = next
		}
	}
	visit(doc)
	var bodyNode *parser.Node
	var find func(*parser.Node)
	find = func(n *parser.Node) {
		if n.Type == parser.ElementNode && n.Data == "body" {
			bodyNode = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	if bodyNode == nil {
		return "", errors.New("HTML 본문을 만들지 못했습니다")
	}
	var b bytes.Buffer
	for c := bodyNode.FirstChild; c != nil; c = c.NextSibling {
		if e = parser.Render(&b, c); e != nil {
			return "", e
		}
	}
	return b.String(), nil
}
func htmlExportPage(title, body string) string {
	return `<!doctype html><html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex,nofollow"><meta name="referrer" content="no-referrer"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src 'self' file:; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'"><title>` + html.EscapeString(title) + ` · madi</title><style>body{font:18px/1.8 system-ui,sans-serif;color:#183d36;background:#f8faf8;margin:auto;padding:32px 20px;max-width:960px;overflow-wrap:anywhere}img{max-width:100%;height:auto}pre{overflow:auto;background:#e9efec;padding:20px;border-radius:12px}table{border-collapse:collapse;width:100%;display:block;overflow:auto}td,th{border:1px solid #cadbd4;padding:10px}a{color:#176b54}blockquote{border-left:4px solid #90b5a5;margin-left:0;padding-left:20px}@media print{body{background:white;font-size:12pt}pre{white-space:pre-wrap}}</style></head><body><main>` + body + `</main></body></html>`
}

func parseJSONVault(raw []byte) (*vaultInput, error) {
	var v transferJSONVault
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&v) != nil || v.Format != "madi-json-vault" || v.Version != 1 || v.Manifest.Format != "madi-vault" || v.Manifest.Version != 1 {
		return nil, errors.New("지원 형식은 madi-json-vault 버전 1입니다")
	}
	if len(v.Files) == 0 || len(v.Files) > 5000 || len(v.Manifest.Documents) == 0 || len(v.Manifest.Documents) > 1000 {
		return nil, errors.New("JSON vault 문서·파일 개수 제한을 확인하세요")
	}
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	used := map[string]bool{}
	var total int64
	for _, f := range v.Files {
		if !safeVaultPath(f.Path) || used[strings.ToLower(f.Path)] || f.Path == "madi-manifest.json" {
			return nil, errors.New("JSON vault 파일 경로가 중복되거나 안전하지 않습니다")
		}
		used[strings.ToLower(f.Path)] = true
		total += int64(len(f.Data))
		if total > 100<<20 || len(f.Data) > 50<<20 {
			return nil, errors.New("JSON vault 펼친 크기 제한을 초과했습니다")
		}
		w, e := zw.Create(f.Path)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(f.Data); e != nil {
			return nil, e
		}
	}
	for _, d := range v.Manifest.Documents {
		if d == nil {
			return nil, errors.New("JSON 문서 항목이 비어 있습니다")
		}
		d.Visibility = "private"
		d.Status = "draft"
		d.Version = 0
	}
	w, e := zw.Create("madi-manifest.json")
	if e != nil {
		return nil, e
	}
	if e = json.NewEncoder(w).Encode(v.Manifest); e != nil {
		return nil, e
	}
	if e = zw.Close(); e != nil {
		return nil, e
	}
	return parseVaultInput("madi-json-vault.zip", b.Bytes())
}
