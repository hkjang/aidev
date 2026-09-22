package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func connectorQuery(ref string, q url.Values) string { return ref + "?" + q.Encode() }
func connectorNext(c connectorConfig, current, next string) (string, error) {
	if next == "" {
		return "", nil
	}
	base, e := connectorURL(c.BaseURL, current)
	if e != nil {
		return "", e
	}
	ref, e := url.Parse(next)
	if e != nil {
		return "", errors.New("페이지네이션 주소 형식이 올바르지 않습니다")
	}
	next = base.ResolveReference(ref).String()
	_, e = connectorURL(c.BaseURL, next)
	return next, e
}
func connectorSourceURL(value string) string {
	u, e := url.Parse(value)
	if e != nil || !oneOf(u.Scheme, "https", "http") || u.Hostname() == "" || u.User != nil || len(value) > 2048 {
		return ""
	}
	return u.String()
}
func connectorPlainItem(id, title, md, source string) (connectorItem, error) {
	if id == "" || len(id) > 1000 || strings.ContainsAny(id, "\x00\n\r") {
		return connectorItem{}, errors.New("원격 문서의 고유 ID가 없거나 너무 깁니다")
	}
	if !utf8.ValidString(md) || len(md) > 4<<20 {
		return connectorItem{}, errors.New("원격 문서는 UTF-8 형식의 4MB 이하여야 합니다")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = id
	}
	return connectorItem{ID: id, Title: connectorTrim(title, 300), Markdown: md, URL: connectorSourceURL(source)}, nil
}
func (s *Server) fetchConnectorPage(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	if len(cursor) > 16384 {
		return connectorPage{}, errors.New("커넥터 체크포인트가 너무 큽니다")
	}
	switch c.Kind {
	case "github":
		return s.fetchGitHubConnector(ctx, c, cursor)
	case "gitlab":
		return s.fetchGitLabConnector(ctx, c, cursor)
	case "jira":
		return s.fetchJiraConnector(ctx, c, cursor)
	case "confluence":
		return s.fetchConfluenceConnector(ctx, c, cursor)
	case "drive":
		return s.fetchDriveConnector(ctx, c, cursor)
	case "sharepoint":
		return s.fetchSharePointConnector(ctx, c, cursor)
	case "rest":
		return s.fetchRESTConnector(ctx, c, cursor)
	}
	return connectorPage{}, errors.New("지원하지 않는 커넥터입니다")
}
func (s *Server) fetchGitLabConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	if cursor == "" {
		cursor = "1"
	}
	n, e := strconv.Atoi(cursor)
	if e != nil || n < 1 || n > 100000 {
		return page, errors.New("GitLab 페이지 번호를 확인하세요")
	}
	ref := connectorQuery("projects/"+url.PathEscape(str(c.Config, "project_id"))+"/wikis", url.Values{"with_content": {"1"}, "per_page": {"100"}, "page": {cursor}})
	raw, headers, e := s.connectorRead(ctx, c, ref)
	if e != nil {
		return page, e
	}
	var entries []map[string]any
	if json.Unmarshal(raw, &entries) != nil {
		return page, errors.New("GitLab Wiki 응답 형식을 확인하세요")
	}
	for _, v := range entries {
		if str(v, "format") != "" && str(v, "format") != "markdown" {
			page.Warnings = append(page.Warnings, "Markdown이 아닌 GitLab Wiki를 원문으로 보존합니다: "+str(v, "title"))
		}
		item, e := connectorPlainItem(str(v, "slug"), str(v, "title"), str(v, "content"), str(v, "web_url"))
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	page.Next = headers.Get("X-Next-Page")
	if page.Next == "" && len(entries) == 100 {
		page.Next = strconv.Itoa(n + 1)
	}
	return page, nil
}
func (s *Server) fetchJiraConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	q := url.Values{"jql": {str(c.Config, "jql")}, "maxResults": {"100"}, "fields": {"summary,description,updated"}}
	ref := "search/jql"
	server := str(c.Config, "variant") == "server"
	if server {
		ref = "search"
		if cursor == "" {
			cursor = "0"
		}
		if n, e := strconv.Atoi(cursor); e != nil || n < 0 {
			return page, errors.New("Jira 시작 위치가 올바르지 않습니다")
		}
		q.Set("startAt", cursor)
	} else if cursor != "" {
		q.Set("nextPageToken", cursor)
	}
	raw, _, e := s.connectorRead(ctx, c, connectorQuery(ref, q))
	if e != nil {
		return page, e
	}
	v, e := connectorJSON(raw)
	if e != nil {
		return page, e
	}
	for _, entry := range connectorArray(v["issues"]) {
		issue, _ := entry.(map[string]any)
		fields, _ := issue["fields"].(map[string]any)
		id := str(issue, "key")
		item, e := connectorPlainItem(id, id+" · "+str(fields, "summary"), connectorBody(fields["description"]), str(issue, "self"))
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	if server {
		start, _ := strconv.Atoi(cursor)
		total, _ := strconv.Atoi(connectorString(v["total"]))
		if start+len(page.Items) < total {
			page.Next = strconv.Itoa(start + len(page.Items))
		}
	} else {
		page.Next = str(v, "nextPageToken")
	}
	return page, nil
}
func (s *Server) fetchConfluenceConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	ref := cursor
	legacy := str(c.Config, "variant") == "server"
	if ref == "" {
		if legacy {
			ref = connectorQuery("content", url.Values{"type": {"page"}, "spaceKey": {str(c.Config, "remote_space")}, "limit": {"100"}, "expand": {"body.storage,version"}})
		} else {
			q := url.Values{"limit": {"100"}, "body-format": {"storage"}}
			if remote := str(c.Config, "remote_space"); remote != "" {
				q.Set("space-id", remote)
			}
			ref = connectorQuery("pages", q)
		}
	}
	raw, _, e := s.connectorRead(ctx, c, ref)
	if e != nil {
		return page, e
	}
	v, e := connectorJSON(raw)
	if e != nil {
		return page, e
	}
	for _, entry := range connectorArray(v["results"]) {
		record, _ := entry.(map[string]any)
		htmlText := connectorString(connectorValue(record, "body.storage.value"))
		_, md, e := migrationHTML([]byte(htmlText))
		if e != nil {
			return page, e
		}
		source := str(record, "webUrl")
		if source == "" {
			source = connectorString(connectorValue(record, "_links.webui"))
			if source != "" {
				u, _ := connectorURL(c.BaseURL, source)
				if u != nil {
					source = u.String()
				}
			}
		}
		item, e := connectorPlainItem(connectorString(record["id"]), str(record, "title"), md, source)
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	page.Next, e = connectorNext(c, ref, connectorString(connectorValue(v, "_links.next")))
	return page, e
}
func (s *Server) fetchDriveConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	q := url.Values{"q": {"trashed=false"}, "pageSize": {"100"}, "fields": {"nextPageToken,incompleteSearch,files(id,name,mimeType,webViewLink,modifiedTime)"}, "supportsAllDrives": {"true"}, "includeItemsFromAllDrives": {"true"}}
	if query := str(c.Config, "query"); query != "" {
		q.Set("q", "trashed=false and ("+query+")")
	}
	if cursor != "" {
		q.Set("pageToken", cursor)
	}
	raw, _, e := s.connectorRead(ctx, c, connectorQuery("files", q))
	if e != nil {
		return page, e
	}
	v, e := connectorJSON(raw)
	if e != nil {
		return page, e
	}
	if boolean(v, "incompleteSearch") {
		return page, errors.New("Drive 검색이 불완전합니다. 쿼리 범위를 특정 드라이브/폴더로 좁히세요")
	}
	for _, entry := range connectorArray(v["files"]) {
		file, _ := entry.(map[string]any)
		id := str(file, "id")
		mime := str(file, "mimeType")
		if mime == "application/vnd.google-apps.folder" || mime == "application/vnd.google-apps.shortcut" {
			continue
		}
		content := ""
		ref := ""
		switch mime {
		case "application/vnd.google-apps.document":
			ref = connectorQuery("files/"+url.PathEscape(id)+"/export", url.Values{"mimeType": {"text/plain"}})
		case "text/plain", "text/markdown", "text/html", "text/csv", "application/json":
			ref = connectorQuery("files/"+url.PathEscape(id), url.Values{"alt": {"media"}})
		default:
			content = "# " + str(file, "name") + "\n\n파일 유형: " + mime + "\n\n원본 시스템에서 여는 첨부 자료입니다. 바이너리 파일 본문은 텍스트로 변환하지 않습니다."
			page.Warnings = append(page.Warnings, "바이너리 파일은 메타데이터 페이지로 가져옵니다: "+str(file, "name"))
		}
		if ref != "" {
			body, _, e := s.connectorRead(ctx, c, ref)
			if e != nil {
				return page, e
			}
			if mime == "text/html" {
				_, content, e = migrationHTML(body)
				if e != nil {
					return page, e
				}
			} else {
				content = string(body)
			}
		}
		item, e := connectorPlainItem(id, str(file, "name"), content, str(file, "webViewLink"))
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	page.Next = str(v, "nextPageToken")
	return page, nil
}
func (s *Server) fetchSharePointConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	ref := cursor
	if ref == "" {
		ref = connectorQuery("sites/"+url.PathEscape(str(c.Config, "site_id"))+"/pages/microsoft.graph.sitePage", url.Values{"$top": {"100"}, "$expand": {"canvasLayout"}})
	}
	raw, _, e := s.connectorRead(ctx, c, ref)
	if e != nil {
		return page, e
	}
	v, e := connectorJSON(raw)
	if e != nil {
		return page, e
	}
	for _, entry := range connectorArray(v["value"]) {
		record, _ := entry.(map[string]any)
		var content strings.Builder
		var walk func(any)
		walk = func(value any) {
			switch node := value.(type) {
			case map[string]any:
				if fragment := str(node, "innerHtml"); fragment != "" {
					content.WriteString(fragment)
					content.WriteByte('\n')
				}
				keys := make([]string, 0, len(node))
				for key := range node {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					if key != "innerHtml" {
						walk(node[key])
					}
				}
			case []any:
				for _, child := range node {
					walk(child)
				}
			}
		}
		walk(record["canvasLayout"])
		_, markdown, e := migrationHTML([]byte(content.String()))
		if e != nil {
			return page, e
		}
		item, e := connectorPlainItem(str(record, "id"), str(record, "title"), markdown, str(record, "webUrl"))
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	page.Next, e = connectorNext(c, ref, str(v, "@odata.nextLink"))
	return page, e
}
func (s *Server) fetchRESTConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	ref := cursor
	if ref == "" {
		ref = str(c.Config, "list_path")
	}
	raw, _, e := s.connectorRead(ctx, c, ref)
	if e != nil {
		return page, e
	}
	var v any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if decoder.Decode(&v) != nil {
		return page, errors.New("REST JSON 응답을 읽을 수 없습니다")
	}
	items, ok := connectorValue(v, str(c.Config, "items_path")).([]any)
	if !ok {
		return page, errors.New("REST 항목 경로가 JSON 배열을 가리켜야 합니다")
	}
	if len(items) > 1000 {
		return page, errors.New("REST API는 페이지당 1,000개 이하 항목을 반환해야 합니다")
	}
	for _, record := range items {
		item, e := connectorPlainItem(connectorString(connectorValue(record, str(c.Config, "id_path"))), connectorString(connectorValue(record, str(c.Config, "title_path"))), connectorString(connectorValue(record, str(c.Config, "content_path"))), connectorString(connectorValue(record, str(c.Config, "url_path"))))
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	if next := str(c.Config, "next_path"); next != "" {
		page.Next, e = connectorNext(c, ref, connectorString(connectorValue(v, next)))
	}
	return page, e
}
func (s *Server) fetchGitHubConnector(ctx context.Context, c connectorConfig, cursor string) (connectorPage, error) {
	page := connectorPage{}
	offset := 0
	pinned := ""
	if cursor != "" {
		parts := strings.Split(cursor, ":")
		if len(parts) != 2 || len(parts[0]) < 40 || len(parts[0]) > 64 {
			return page, errors.New("GitHub 트리 체크포인트를 확인하세요")
		}
		for _, c := range parts[0] {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return page, errors.New("GitHub 트리 체크포인트를 확인하세요")
			}
		}
		pinned = parts[0]
		var e error
		offset, e = strconv.Atoi(parts[1])
		if e != nil || offset < 0 || offset > 100000 {
			return page, errors.New("GitHub 체크포인트를 확인하세요")
		}
	}
	repo := strings.Split(str(c.Config, "repository"), "/")
	if len(repo) != 2 {
		return page, errors.New("GitHub 저장소는 owner/repository 형식입니다")
	}
	prefix := "repos/" + url.PathEscape(repo[0]) + "/" + url.PathEscape(repo[1]) + "/git/"
	ref := str(c.Config, "ref")
	if ref == "" {
		ref = "HEAD"
	}
	if pinned != "" {
		ref = pinned
	}
	raw, _, e := s.connectorRead(ctx, c, prefix+"trees/"+url.PathEscape(ref)+"?recursive=1")
	if e != nil {
		return page, e
	}
	tree, e := connectorJSON(raw)
	if e != nil {
		return page, e
	}
	if boolean(tree, "truncated") {
		return page, errors.New("GitHub 트리가 잘렸습니다. 완전한 트리를 반환하는 하위 tree SHA를 ref로 지정하세요")
	}
	files := []map[string]any{}
	filter := str(c.Config, "path_prefix")
	for _, entry := range connectorArray(tree["tree"]) {
		file, _ := entry.(map[string]any)
		name := str(file, "path")
		if str(file, "type") == "blob" && strings.HasPrefix(name, filter) && oneOf(strings.ToLower(path.Ext(name)), ".md", ".markdown", ".txt", ".html", ".htm") {
			files = append(files, file)
		}
	}
	if offset > len(files) {
		return page, errors.New("GitHub 트리가 바뀌었습니다. 처음부터 다시 가져오세요")
	}
	stop := min(len(files), offset+100)
	for _, file := range files[offset:stop] {
		raw, _, e := s.connectorRead(ctx, c, prefix+"blobs/"+url.PathEscape(str(file, "sha")))
		if e != nil {
			return page, e
		}
		blob, e := connectorJSON(raw)
		if e != nil {
			return page, e
		}
		if str(blob, "encoding") != "base64" {
			return page, errors.New("GitHub 파일 인코딩은 base64여야 합니다")
		}
		content, e := base64.StdEncoding.DecodeString(strings.ReplaceAll(str(blob, "content"), "\n", ""))
		if e != nil {
			return page, errors.New("GitHub 파일 내용을 해독하지 못했습니다")
		}
		name := str(file, "path")
		md := string(content)
		if oneOf(strings.ToLower(path.Ext(name)), ".html", ".htm") {
			_, md, e = migrationHTML(content)
			if e != nil {
				return page, e
			}
		}
		item, e := connectorPlainItem(name, strings.TrimSuffix(path.Base(name), path.Ext(name)), md, fmt.Sprintf("%s/repos/%s/%s/contents/%s", strings.TrimSuffix(c.BaseURL, "/"), url.PathEscape(repo[0]), url.PathEscape(repo[1]), url.PathEscape(name)))
		if e != nil {
			return page, e
		}
		page.Items = append(page.Items, item)
	}
	if stop < len(files) {
		sha := str(tree, "sha")
		if len(sha) < 40 || len(sha) > 64 {
			return page, errors.New("GitHub 응답에 고정할 트리 SHA가 없습니다")
		}
		page.Next = sha + ":" + strconv.Itoa(stop)
	}
	return page, nil
}
