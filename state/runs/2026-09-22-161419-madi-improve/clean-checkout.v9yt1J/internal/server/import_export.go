package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

type vaultDocument struct {
	ID          string   `json:"id"`
	File        string   `json:"file"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	Aliases     []string `json:"aliases"`
	ParentID    string   `json:"parent_id,omitempty"`
	Visibility  string   `json:"visibility"`
	Status      string   `json:"status"`
	Version     int64    `json:"source_version,omitempty"`
	Icon        string   `json:"icon"`
	Markdown    string   `json:"-"`
	NewID       string   `json:"-"`
	ParentNewID string   `json:"-"`
	Folder      bool     `json:"-"`
}

// Internal callers may export an explicit selection with a second actor's ACL
// intersection. This is not controlled by public query parameters.
type vaultExportSelectionKey struct{}
type vaultExportSelection struct {
	IDs               []string
	AdditionalActorID string
	MaxBytes          int64
	RawMarkdown       bool
}

type vaultAttachment struct {
	ID          string       `json:"id"`
	DocumentID  string       `json:"document_id"`
	File        string       `json:"file"`
	Name        string       `json:"name"`
	ContentType string       `json:"content_type"`
	Size        int64        `json:"size"`
	DiskPath    string       `json:"-"`
	NewID       string       `json:"-"`
	OwnerNewID  string       `json:"-"`
	Object      storedObject `json:"-"`
}

type vaultManifest struct {
	Format      string             `json:"format"`
	Version     int                `json:"version"`
	Documents   []*vaultDocument   `json:"documents"`
	Attachments []*vaultAttachment `json:"attachments"`
}

type contextVaultReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextVaultReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(b)
}

func safeVaultSegment(value string) string {
	var out strings.Builder
	for _, r := range value {
		if r < 32 || strings.ContainsRune(`/\<>:"|?*`, r) {
			out.WriteRune('_')
		} else {
			out.WriteRune(r)
		}
	}
	segment := strings.Trim(out.String(), " .")
	if segment == "" {
		segment = "문서"
	}
	runes := []rune(segment)
	if len(runes) > 90 {
		segment = string(runes[:90])
	}
	return segment
}

func vaultExportPaths(docs []*vaultDocument) error {
	lookup := map[string]*vaultDocument{}
	for _, d := range docs {
		lookup[d.ID] = d
	}
	used := map[string]bool{}
	visiting := map[string]bool{}
	var assign func(*vaultDocument) error
	assign = func(d *vaultDocument) error {
		if d.File != "" {
			return nil
		}
		if visiting[d.ID] {
			return errors.New("문서 트리에 순환 참조가 있습니다")
		}
		visiting[d.ID] = true
		folder := ""
		if parent := lookup[d.ParentID]; parent != nil {
			if err := assign(parent); err != nil {
				return err
			}
			folder = strings.TrimSuffix(parent.File, ".md") + "/"
		}
		base := folder + safeVaultSegment(d.Title)
		candidate := base + ".md"
		if used[strings.ToLower(candidate)] {
			candidate = base + "-" + d.ID[:8] + ".md"
		}
		for used[strings.ToLower(candidate)] {
			candidate = strings.TrimSuffix(candidate, ".md") + "_" + ".md"
		}
		d.File = candidate
		used[strings.ToLower(candidate)] = true
		delete(visiting, d.ID)
		return nil
	}
	for _, d := range docs {
		if err := assign(d); err != nil {
			return err
		}
	}
	return nil
}

var vaultWikiPattern = regexp.MustCompile(`(!?)\[\[([^\]\n]+)\]\]`)
var vaultMarkdownLink = regexp.MustCompile(`(!?\[[^\]\n]*\]\()([^\n)]*)(\))`)
var vaultReferenceDefinition = regexp.MustCompile(`(?m)^([ \t]{0,3}\[[^\]\n]+\]:\s*)(<[^>\n]*>|\S+)([^\n]*)`)

// Rewriting links must not change examples in fenced/inline code or YAML metadata.
func rewriteVaultContent(markdown string, rewrite func(string) string) string {
	lines := strings.SplitAfter(markdown, "\n")
	var out strings.Builder
	frontMatter := len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if frontMatter {
			out.WriteString(line)
			if index > 0 && trimmed == "---" {
				frontMatter = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := string(trimmed[0])
			count := 0
			for count < len(trimmed) && string(trimmed[count]) == marker {
				count++
			}
			if fence == "" {
				fence = strings.Repeat(marker, count)
			} else if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			out.WriteString(line)
			continue
		}
		if fence != "" || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			out.WriteString(line)
			continue
		}
		for rest := line; rest != ""; {
			start := strings.IndexByte(rest, '`')
			if start < 0 {
				out.WriteString(rewrite(rest))
				break
			}
			out.WriteString(rewrite(rest[:start]))
			count := 1
			for start+count < len(rest) && rest[start+count] == '`' {
				count++
			}
			marker := strings.Repeat("`", count)
			end := strings.Index(rest[start+count:], marker)
			if end < 0 {
				out.WriteString(rewrite(rest[start:]))
				break
			}
			end += start + 2*count
			out.WriteString(rest[start:end])
			rest = rest[end:]
		}
	}
	return out.String()
}

func splitVaultWiki(value string) (target, fragment, label string) {
	pieces := strings.SplitN(value, "|", 2)
	target = pieces[0]
	if len(pieces) == 2 {
		label = "|" + pieces[1]
	}
	if index := strings.IndexByte(target, '#'); index >= 0 {
		fragment = target[index:]
		target = target[:index]
	}
	return strings.TrimSpace(target), fragment, label
}

func rewriteVaultMarkdownLinks(md string, rewrite func(string) string) string {
	md = vaultMarkdownLink.ReplaceAllStringFunc(md, func(match string) string {
		parts := vaultMarkdownLink.FindStringSubmatch(match)
		inner := strings.TrimSpace(parts[2])
		destination, suffix := inner, ""
		if strings.HasPrefix(inner, "<") {
			if close := strings.Index(inner, ">"); close >= 0 {
				destination = inner[1:close]
				suffix = inner[close+1:]
			}
		}
		if destination == inner {
			for _, separator := range []string{` "`, ` '`} {
				if index := strings.Index(destination, separator); index >= 0 {
					suffix = destination[index:]
					destination = destination[:index]
					break
				}
			}
		}
		updated := rewrite(destination)
		if updated == destination {
			return match
		}
		return parts[1] + updated + suffix + parts[3]
	})
	return vaultReferenceDefinition.ReplaceAllStringFunc(md, func(match string) string {
		parts := vaultReferenceDefinition.FindStringSubmatch(match)
		destination := strings.TrimSuffix(strings.TrimPrefix(parts[2], "<"), ">")
		updated := rewrite(destination)
		if updated == destination {
			return match
		}
		return parts[1] + updated + parts[3]
	})
}

func vaultEscapedPath(value string) string { return (&url.URL{Path: value}).EscapedPath() }

func (s *Server) exportMarkdown(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	p := current(r)
	if !s.canWorkspace(r.Context(), p, wid, false) {
		apiError(w, 403, "워크스페이스 내보내기 권한이 없습니다")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	selection, _ := r.Context().Value(vaultExportSelectionKey{}).(vaultExportSelection)
	rows, err := tx.Query(r.Context(), `SELECT d.id::text,d.title,d.markdown,d.tags,d.aliases,COALESCE(d.parent_id::text,''),d.visibility,d.status,d.icon,d.version FROM documents d WHERE `+docACL+` AND d.workspace_id=$2 AND d.deleted_at IS NULL AND ($3::uuid[] IS NULL OR d.id=ANY($3::uuid[])) AND ($4='' OR madi_document_allowed(NULLIF($4,'')::uuid,d.id,false)) ORDER BY d.created_at,d.id`, p.ID, wid, selection.IDs, selection.AdditionalActorID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	manifest := vaultManifest{Format: "madi-vault", Version: 1, Documents: []*vaultDocument{}, Attachments: []*vaultAttachment{}}
	var exportSize int64
	for rows.Next() {
		d := &vaultDocument{}
		var tags, aliases []byte
		err = rows.Scan(&d.ID, &d.Title, &d.Markdown, &tags, &aliases, &d.ParentID, &d.Visibility, &d.Status, &d.Icon, &d.Version)
		if err != nil {
			break
		}
		exportSize += int64(len(d.Markdown) + len(d.Title) + len(tags) + len(aliases))
		if selection.MaxBytes > 0 && exportSize > selection.MaxBytes {
			rows.Close()
			apiError(w, 413, "선택 문서의 내보내기 크기 한도를 초과했습니다")
			return
		}
		if err = json.Unmarshal(tags, &d.Tags); err != nil {
			break
		}
		if err = json.Unmarshal(aliases, &d.Aliases); err != nil {
			break
		}
		manifest.Documents = append(manifest.Documents, d)
		if selection.MaxBytes > 0 && len(manifest.Documents) > 1000 {
			rows.Close()
			apiError(w, 413, "한 번에 문서 최대 1,000개를 내보낼 수 있습니다")
			return
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	if selection.IDs != nil && len(manifest.Documents) != len(selection.IDs) {
		apiError(w, 403, "선택 문서의 현재 내보내기 권한을 모두 확인할 수 없습니다")
		return
	}
	if err = vaultExportPaths(manifest.Documents); err != nil {
		apiError(w, 409, err.Error())
		return
	}
	lookup := map[string]*vaultDocument{}
	byID := map[string]*vaultDocument{}
	for _, d := range manifest.Documents {
		byID[d.ID] = d
		for _, name := range append([]string{d.ID, d.Title}, d.Aliases...) {
			if lookup[name] == nil {
				lookup[name] = d
			}
		}
	}
	attachmentRows, err := tx.Query(r.Context(), `SELECT a.id::text,a.document_id::text,a.name,a.content_type,a.size,a.path,coalesce(a.storage_provider_id::text,''),a.object_key,a.checksum_sha256 FROM attachments a JOIN documents d ON d.id=a.document_id WHERE `+docACL+` AND d.workspace_id=$2 AND d.deleted_at IS NULL AND ($3::uuid[] IS NULL OR d.id=ANY($3::uuid[])) AND ($4='' OR madi_document_allowed(NULLIF($4,'')::uuid,d.id,false)) ORDER BY a.id`, p.ID, wid, selection.IDs, selection.AdditionalActorID)
	if err != nil {
		respond(w, nil, err)
		return
	}
	for attachmentRows.Next() {
		a := &vaultAttachment{}
		if err = attachmentRows.Scan(&a.ID, &a.DocumentID, &a.Name, &a.ContentType, &a.Size, &a.DiskPath, &a.Object.ProviderID, &a.Object.Key, &a.Object.Checksum); err != nil {
			break
		}
		a.Object.Path = a.DiskPath
		a.Object.Size = a.Size
		exportSize += a.Size
		if selection.MaxBytes > 0 && exportSize > selection.MaxBytes {
			attachmentRows.Close()
			apiError(w, 413, "선택 문서와 첨부의 내보내기 크기 한도를 초과했습니다")
			return
		}
		a.File = "attachments/" + a.ID + "/" + safeVaultSegment(a.Name)
		manifest.Attachments = append(manifest.Attachments, a)
		if selection.MaxBytes > 0 && len(manifest.Documents)+len(manifest.Attachments)+1 > 5000 {
			attachmentRows.Close()
			apiError(w, 413, "한 번에 파일 최대 5,000개를 내보낼 수 있습니다")
			return
		}
	}
	if err == nil {
		err = attachmentRows.Err()
	}
	attachmentRows.Close()
	if err != nil {
		respond(w, nil, err)
		return
	}
	if selection.MaxBytes > 0 && exportSize > selection.MaxBytes {
		apiError(w, 413, "선택 문서와 첨부의 내보내기 크기 한도를 초과했습니다")
		return
	}
	attachments := map[string]*vaultAttachment{}
	for _, a := range manifest.Attachments {
		attachments["/api/v1/attachments/"+a.ID] = a
	}
	tmp, err := os.CreateTemp("", "madi-export-*.zip")
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	zw := zip.NewWriter(tmp)
	for _, d := range manifest.Documents {
		if err = r.Context().Err(); err != nil {
			break
		}
		md := rewriteVaultContent(d.Markdown, func(segment string) string {
			return vaultWikiPattern.ReplaceAllStringFunc(segment, func(match string) string {
				parts := vaultWikiPattern.FindStringSubmatch(match)
				target, fragment, label := splitVaultWiki(parts[2])
				linked := lookup[target]
				if linked == nil {
					return match
				}
				exportTarget := strings.TrimSuffix(linked.File, ".md")
				if label == "" && exportTarget != target {
					label = "|" + linked.Title
				}
				return parts[1] + "[[" + exportTarget + fragment + label + "]]"
			})
		})
		md = rewriteVaultContent(md, func(segment string) string {
			return rewriteVaultMarkdownLinks(segment, func(destination string) string {
				if a := attachments[destination]; a != nil {
					relative, e := filepath.Rel(filepath.FromSlash(path.Dir(d.File)), filepath.FromSlash(a.File))
					if e == nil {
						return vaultEscapedPath(filepath.ToSlash(relative))
					}
				}
				parsed, _ := url.Parse(destination)
				internalPath := destination
				if parsed != nil {
					internalPath = parsed.Path
				}
				if linked := byID[strings.TrimPrefix(internalPath, "/app/documents/")]; linked != nil {
					relative, e := filepath.Rel(filepath.FromSlash(path.Dir(d.File)), filepath.FromSlash(linked.File))
					if e == nil {
						fragment := ""
						if parsed != nil && parsed.Fragment != "" {
							fragment = "#" + url.PathEscape(parsed.Fragment)
						}
						return vaultEscapedPath(filepath.ToSlash(relative)) + fragment
					}
				}
				return destination
			})
		})
		if selection.RawMarkdown {
			md = d.Markdown
		}
		var f io.Writer
		f, err = zw.Create(d.File)
		if err == nil {
			_, err = io.WriteString(f, md)
		}
		if err != nil {
			break
		}
		if byID[d.ParentID] == nil {
			d.ParentID = ""
		}
	}
	if err == nil {
		for _, a := range manifest.Attachments {
			var src *os.File
			src, err = s.materializeObject(r.Context(), a.Object, 50<<20)
			if err != nil {
				break
			}
			var f io.Writer
			f, err = zw.Create(a.File)
			if err == nil {
				_, err = io.Copy(f, contextVaultReader{r.Context(), src})
			}
			src.Close()
			os.Remove(src.Name())
			if err != nil {
				break
			}
		}
	}
	if err == nil {
		var f io.Writer
		f, err = zw.Create("madi-manifest.json")
		if err == nil {
			err = json.NewEncoder(f).Encode(manifest)
		}
	}
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	if _, err = tmp.Seek(0, 0); err != nil {
		respond(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="madi-workspace.zip"`)
	s.audit(r, "WORKSPACE_EXPORT", wid, map[string]any{"documents": len(manifest.Documents), "attachments": len(manifest.Attachments)})
	_, _ = io.Copy(w, contextVaultReader{r.Context(), tmp})
}

type vaultInput struct {
	files       map[string][]byte
	manifest    *vaultManifest
	directories []string
	databases   map[string]migrationCSV
}

func safeVaultPath(value string) bool {
	if value == "" || len(value) > 4096 || strings.Count(value, "/") > 64 || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") || path.Clean(value) != strings.TrimSuffix(value, "/") {
		return false
	}
	for _, segment := range strings.Split(strings.TrimSuffix(value, "/"), "/") {
		if segment == ".." || segment == "." || segment == "" || strings.Contains(segment, ":") {
			return false
		}
	}
	return true
}

func parseVaultInput(filename string, raw []byte) (*vaultInput, error) {
	input := &vaultInput{files: map[string][]byte{}}
	ext := strings.ToLower(path.Ext(filename))
	if ext == ".md" || ext == ".markdown" {
		if len(raw) > 4<<20 {
			return nil, errors.New("개별 문서는 4MB 이하여야 합니다")
		}
		input.files[path.Base(strings.ReplaceAll(filename, "\\", "/"))] = raw
		return input, nil
	}
	if ext != ".zip" {
		return nil, errors.New(".md, .markdown, .zip 형식을 지원합니다")
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, errors.New("올바른 ZIP 파일이 아닙니다")
	}
	if len(zr.File) > 5000 {
		return nil, errors.New("ZIP 항목은 5000개 이하여야 합니다")
	}
	var total uint64
	seen := map[string]bool{}
	for _, f := range zr.File {
		if !safeVaultPath(f.Name) || f.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("안전하지 않은 ZIP 경로: %s", f.Name)
		}
		key := strings.ToLower(strings.TrimSuffix(f.Name, "/"))
		if seen[key] {
			return nil, fmt.Errorf("중복 ZIP 경로: %s", f.Name)
		}
		seen[key] = true
		if f.FileInfo().IsDir() {
			if !oneOf(strings.Split(f.Name, "/")[0], ".obsidian", ".git", "__MACOSX") {
				input.directories = append(input.directories, strings.TrimSuffix(f.Name, "/"))
			}
			continue
		}
		if f.UncompressedSize64 > 100<<20-total {
			return nil, errors.New("압축 해제 크기 합계는 100MB 이하여야 합니다")
		}
		total += f.UncompressedSize64
		limit := int64(50 << 20)
		if oneOf(strings.ToLower(path.Ext(f.Name)), ".md", ".markdown") || f.Name == "madi-manifest.json" {
			limit = 4 << 20
		}
		if f.UncompressedSize64 > uint64(limit) {
			return nil, fmt.Errorf("파일 크기 제한을 초과했습니다: %s", f.Name)
		}
		rc, e := f.Open()
		if e != nil {
			return nil, e
		}
		data, e := io.ReadAll(io.LimitReader(rc, limit+1))
		rc.Close()
		if e != nil || int64(len(data)) > limit {
			return nil, fmt.Errorf("파일 압축을 해제하지 못했습니다: %s", f.Name)
		}
		if strings.HasPrefix(f.Name, ".obsidian/") || strings.HasPrefix(f.Name, ".git/") || strings.HasPrefix(f.Name, "__MACOSX/") || path.Base(f.Name) == ".DS_Store" {
			continue
		}
		input.files[f.Name] = data
	}
	if data, exists := input.files["madi-manifest.json"]; exists {
		manifest := &vaultManifest{}
		if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
			err = json.Unmarshal(data, &manifest.Documents)
		} else {
			err = json.Unmarshal(data, manifest)
			if err == nil && (manifest.Format != "madi-vault" || manifest.Version != 1) {
				err = errors.New("지원하지 않는 manifest 버전")
			}
		}
		if err != nil {
			return nil, errors.New("madi manifest 형식을 확인하세요")
		}
		input.manifest = manifest
		delete(input.files, "madi-manifest.json")
	}
	return input, nil
}

func resolveVaultFile(files map[string][]byte, documentFile, target string) string {
	if u, err := url.Parse(target); err == nil {
		if u.IsAbs() || u.Host != "" {
			return ""
		}
		target = u.Path
	} else {
		return ""
	}
	if strings.Contains(target, "\\") || strings.ContainsRune(target, '\x00') {
		return ""
	}
	if target == "" {
		return ""
	}
	for _, candidate := range []string{path.Clean(path.Join(path.Dir(documentFile), target)), strings.TrimPrefix(path.Clean(target), "/")} {
		if !safeVaultPath(candidate) {
			continue
		}
		if _, exists := files[candidate]; exists {
			return candidate
		}
		for _, ext := range []string{".md", ".markdown"} {
			if _, exists := files[candidate+ext]; exists {
				return candidate + ext
			}
		}
	}
	var found string
	for name := range files {
		if path.Base(name) == target || strings.TrimSuffix(path.Base(name), path.Ext(name)) == target {
			if found != "" {
				return ""
			}
			found = name
		}
	}
	return found
}

func prepareVaultDocuments(input *vaultInput) ([]*vaultDocument, map[string]*vaultDocument, error) {
	metadata := map[string]*vaultDocument{}
	oldIDs := map[string]*vaultDocument{}
	if input.manifest != nil {
		for _, d := range input.manifest.Documents {
			if d == nil || !safeVaultPath(d.File) || !oneOf(strings.ToLower(path.Ext(d.File)), ".md", ".markdown") || metadata[d.File] != nil || d.ID == "" || oldIDs[d.ID] != nil {
				return nil, nil, errors.New("manifest 문서 ID 또는 파일 경로가 올바르지 않습니다")
			}
			if _, exists := input.files[d.File]; !exists {
				return nil, nil, fmt.Errorf("manifest 문서 파일이 없습니다: %s", d.File)
			}
			metadata[d.File] = d
			oldIDs[d.ID] = d
		}
	}
	names := []string{}
	for name := range input.files {
		if oneOf(strings.ToLower(path.Ext(name)), ".md", ".markdown") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	docs := []*vaultDocument{}
	byFile := map[string]*vaultDocument{}
	for _, name := range names {
		md := string(input.files[name])
		fm, err := parseFrontMatter(md)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}
		d := metadata[name]
		if d == nil {
			d = &vaultDocument{Title: strings.TrimSuffix(path.Base(name), path.Ext(name)), Visibility: "workspace", Status: "draft", Icon: "file"}
			if title := str(fm, "title"); strings.TrimSpace(title) != "" {
				d.Title = title
			}
			if visibility := str(fm, "visibility"); visibility != "" {
				d.Visibility = visibility
			}
		}
		d.File = name
		d.NewID = newID()
		d.Markdown = md
		if d.Tags == nil {
			d.Tags = listStrings(fm["tags"])
		}
		if d.Aliases == nil {
			d.Aliases = listStrings(fm["aliases"])
		}
		if d.Visibility == "" {
			d.Visibility = "workspace"
		}
		if d.Status == "" {
			d.Status = "draft"
		}
		if d.Icon == "" {
			d.Icon = "file"
		}
		if strings.TrimSpace(d.Title) == "" || len(d.Title) > 500 || !oneOf(d.Visibility, "workspace", "private", "selected") || !oneOf(d.Status, "draft", "review", "published", "rejected", "stale", "archived") || len(d.Icon) > 64 {
			return nil, nil, fmt.Errorf("문서 속성이 올바르지 않습니다: %s", name)
		}
		docs = append(docs, d)
		byFile[name] = d
	}
	if len(docs) == 0 {
		return nil, nil, errors.New("가져올 Markdown 문서가 없습니다")
	}
	folders := map[string]*vaultDocument{}
	var folderFor func(string) (*vaultDocument, error)
	folderFor = func(folder string) (*vaultDocument, error) {
		if folder == "." || folder == "" {
			return nil, nil
		}
		if d := folders[folder]; d != nil {
			return d, nil
		}
		for _, ext := range []string{".md", ".markdown"} {
			if d := byFile[folder+ext]; d != nil {
				folders[folder] = d
				return d, nil
			}
		}
		parent, err := folderFor(path.Dir(folder))
		if err != nil {
			return nil, err
		}
		d := &vaultDocument{NewID: newID(), File: folder + "/.madi-folder.md", Title: path.Base(folder), Tags: []string{}, Aliases: []string{}, Icon: "folder", Visibility: "workspace", Status: "draft", Folder: true}
		if len(d.Title) > 500 {
			return nil, errors.New("폴더 이름이 너무 깁니다")
		}
		if parent != nil {
			d.ParentNewID = parent.NewID
		}
		folders[folder] = d
		docs = append(docs, d)
		return d, nil
	}
	actual := append([]*vaultDocument(nil), docs...)
	for _, d := range actual {
		if d.ParentID != "" {
			parent := oldIDs[d.ParentID]
			if parent == nil || parent.NewID == "" {
				return nil, nil, fmt.Errorf("manifest 상위 문서가 없습니다: %s", d.File)
			}
			d.ParentNewID = parent.NewID
		} else if metadata[d.File] == nil {
			parent, err := folderFor(path.Dir(d.File))
			if err != nil {
				return nil, nil, err
			}
			if parent != nil {
				d.ParentNewID = parent.NewID
			}
		}
	}
	if input.manifest == nil {
		for _, folder := range input.directories {
			if _, err := folderFor(folder); err != nil {
				return nil, nil, err
			}
		}
	}
	byID := map[string]*vaultDocument{}
	for _, d := range docs {
		byID[d.NewID] = d
	}
	for _, d := range docs {
		seen := map[string]bool{}
		for current := d; current != nil; current = byID[current.ParentNewID] {
			if seen[current.NewID] {
				return nil, nil, errors.New("manifest 문서 트리에 순환 참조가 있습니다")
			}
			seen[current.NewID] = true
		}
	}
	return docs, byFile, nil
}

func (s *Server) importMarkdown(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, true) {
		apiError(w, 403, "워크스페이스 가져오기 권한이 없습니다")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		apiError(w, 400, "최대 50MB의 Markdown 또는 ZIP 파일을 선택하세요")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		apiError(w, 400, "파일을 선택하세요")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (50<<20)+1))
	if err != nil || len(raw) > 50<<20 {
		apiError(w, 400, "파일을 읽을 수 없거나 50MB를 초과했습니다")
		return
	}
	input, err := parseVaultInput(header.Filename, raw)
	if err != nil {
		apiError(w, 400, err.Error())
		return
	}
	s.importVaultInput(w, r, input)
}

func (s *Server) importVaultInput(w http.ResponseWriter, r *http.Request, input *vaultInput) {
	wid := r.URL.Query().Get("workspace_id")
	space := r.URL.Query().Get("space_id")
	if !hasIntegrationScope(current(r), "document:write") || !s.canSpace(r.Context(), current(r), wid, space, true) {
		apiError(w, 403, "대상 공간의 가져오기 권한이 없습니다")
		return
	}
	docs, byFile, err := prepareVaultDocuments(input)
	if err != nil {
		apiError(w, 400, err.Error())
		return
	}
	settings, err := s.settings(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	byOldID := map[string]*vaultDocument{}
	titleCount := map[string]int{}
	for _, d := range docs {
		if d.ID != "" {
			byOldID[d.ID] = d
		}
		titleCount[d.Title]++
	}
	assets := map[string][]byte{}
	for name, data := range input.files {
		if byFile[name] == nil {
			assets[name] = data
		}
	}
	attachments := []*vaultAttachment{}
	attachmentByFile := map[string]*vaultAttachment{}
	perDocument := map[string]*vaultAttachment{}
	if input.manifest != nil {
		for _, a := range input.manifest.Attachments {
			if a == nil || !safeVaultPath(a.File) || attachmentByFile[a.File] != nil || byOldID[a.DocumentID] == nil {
				returnErrorVault(w, "manifest 첨부파일 정보를 확인하세요")
				return
			}
			if _, exists := assets[a.File]; !exists {
				returnErrorVault(w, "manifest 첨부파일이 없습니다: "+a.File)
				return
			}
			a.NewID = newID()
			a.OwnerNewID = byOldID[a.DocumentID].NewID
			a.Name = path.Base(strings.ReplaceAll(a.Name, "\\", "/"))
			if a.Name == "." || a.Name == "" {
				a.Name = path.Base(a.File)
			}
			attachmentByFile[a.File] = a
			attachments = append(attachments, a)
		}
	}
	usedAssets := map[string]bool{}
	attach := func(d *vaultDocument, filename string) string {
		usedAssets[filename] = true
		a := attachmentByFile[filename]
		if a == nil {
			key := d.NewID + "/" + filename
			a = perDocument[key]
			if a == nil {
				a = &vaultAttachment{NewID: newID(), OwnerNewID: d.NewID, File: filename, Name: path.Base(filename)}
				perDocument[key] = a
				attachments = append(attachments, a)
			}
		}
		return "/api/v1/attachments/" + a.NewID
	}
	oldDocumentFiles := map[string]string{}
	oldAttachmentFiles := map[string]string{}
	if input.manifest != nil {
		for _, d := range input.manifest.Documents {
			oldDocumentFiles[d.ID] = d.File
		}
		for _, a := range input.manifest.Attachments {
			oldAttachmentFiles[a.ID] = a.File
		}
	}
	for _, d := range docs {
		if d.Folder {
			continue
		}
		d.Markdown = rewriteVaultContent(d.Markdown, func(segment string) string {
			return vaultWikiPattern.ReplaceAllStringFunc(segment, func(match string) string {
				parts := vaultWikiPattern.FindStringSubmatch(match)
				target, fragment, label := splitVaultWiki(parts[2])
				filename := resolveVaultFile(input.files, d.File, target)
				if mapped := oldDocumentFiles[target]; mapped != "" {
					filename = mapped
				}
				if linked := byFile[filename]; linked != nil {
					newTarget := linked.Title
					if titleCount[newTarget] > 1 {
						newTarget = linked.NewID
						if label == "" {
							label = "|" + linked.Title
						}
					}
					return parts[1] + "[[" + newTarget + fragment + label + "]]"
				}
				if _, exists := assets[filename]; exists {
					link := attach(d, filename)
					name := path.Base(filename)
					prefix := ""
					if parts[1] == "!" && strings.HasPrefix(mime.TypeByExtension(strings.ToLower(path.Ext(filename))), "image/") {
						prefix = "!"
					}
					return prefix + "[" + strings.ReplaceAll(name, "]", "\\]") + "](" + link + ")"
				}
				return match
			})
		})
		d.Markdown = rewriteVaultContent(d.Markdown, func(segment string) string {
			return rewriteVaultMarkdownLinks(segment, func(destination string) string {
				filename := resolveVaultFile(input.files, d.File, destination)
				if parsed, e := url.Parse(destination); e == nil && parsed.Scheme == "" && parsed.Host == "" {
					if mapped := oldAttachmentFiles[strings.TrimPrefix(parsed.Path, "/api/v1/attachments/")]; mapped != "" && strings.HasPrefix(parsed.Path, "/api/v1/attachments/") {
						filename = mapped
					}
					if mapped := oldDocumentFiles[strings.TrimPrefix(parsed.Path, "/app/documents/")]; mapped != "" && strings.HasPrefix(parsed.Path, "/app/documents/") {
						filename = mapped
					}
				}
				if _, exists := assets[filename]; exists {
					return attach(d, filename)
				}
				if linked := byFile[filename]; linked != nil {
					fragment := ""
					if parsed, e := url.Parse(destination); e == nil && parsed.Fragment != "" {
						fragment = "#" + url.PathEscape(parsed.Fragment)
					}
					return "/app/documents/" + linked.NewID + fragment
				}
				return destination
			})
		})
		if len(d.Markdown) > 4<<20 {
			returnErrorVault(w, "링크 변환 후 문서 크기가 4MB를 초과했습니다: "+d.Title)
			return
		}
	}
	var unattached *vaultDocument
	for filename := range assets {
		if usedAssets[filename] || attachmentByFile[filename] != nil {
			continue
		}
		if unattached == nil {
			unattached = &vaultDocument{NewID: newID(), Title: "가져온 첨부파일", Tags: []string{}, Aliases: []string{}, Icon: "folder", Visibility: "private", Status: "draft", Folder: true}
			docs = append(docs, unattached)
		}
		link := attach(unattached, filename)
		unattached.Markdown += "- [" + strings.ReplaceAll(path.Base(filename), "]", "\\]") + "](" + link + ")\n"
	}
	// ACL traversal deliberately supports at most 20 ancestors. Reject imports
	// that would otherwise report success but create unreachable deep documents.
	byNewID := map[string]*vaultDocument{}
	for _, d := range docs {
		byNewID[d.NewID] = d
	}
	for _, d := range docs {
		seen := map[string]bool{}
		current := d
		depth := 0
		for current != nil {
			if seen[current.NewID] || depth >= 20 {
				apiError(w, 400, "가져올 문서 트리는 순환 없이 최대 20단계까지 지원합니다")
				return
			}
			seen[current.NewID] = true
			depth++
			current = byNewID[current.ParentNewID]
		}
	}
	committed := false
	provider, err := s.resolveStorage(r.Context(), wid)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if len(attachments) > 0 {
		defer func() {
			if !committed {
				for _, a := range attachments {
					_ = s.cleanupStoredObject(r.Context(), a.Object)
				}
			}
		}()
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	for _, d := range docs {
		protected, protectionError := s.ProtectDocumentTx(r.Context(), tx, current(r), d.NewID, wid, d.Title, d.Markdown)
		if protectionError != nil {
			if !WriteProtectionError(w, protectionError) {
				respond(w, nil, protectionError)
			}
			return
		}
		d.Title, d.Markdown = protected.Title, protected.Markdown
		metadata, protectionError := s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), d.NewID, wid, map[string]any{"tags": d.Tags, "aliases": d.Aliases})
		if protectionError != nil {
			if !WriteProtectionError(w, protectionError) {
				respond(w, nil, protectionError)
			}
			return
		}
		if metadata.Changed {
			values := metadata.Value.(map[string]any)
			d.Tags = listStrings(values["tags"])
			d.Aliases = listStrings(values["aliases"])
		}
		status := d.Status
		if boolean(settings, "approval_enabled") && oneOf(status, "published", "review", "rejected") {
			status = "draft"
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO documents(id,workspace_id,title,markdown,tags,aliases,icon,status,visibility,owner_id,space_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid)`, d.NewID, wid, d.Title, d.Markdown, jsonValue(d.Tags), jsonValue(d.Aliases), d.Icon, status, d.Visibility, current(r).ID, space)
		if err != nil {
			respond(w, nil, err)
			return
		}
	}
	for _, d := range docs {
		if d.ParentNewID != "" {
			if _, err = tx.Exec(r.Context(), `UPDATE documents SET parent_id=$2 WHERE id=$1`, d.NewID, d.ParentNewID); err != nil {
				respond(w, nil, err)
				return
			}
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id) SELECT id,version,title,markdown,tags,owner_id FROM documents WHERE id=$1`, d.NewID); err != nil {
			respond(w, nil, err)
			return
		}
		if err = s.enqueueEvent(r.Context(), tx, Event{Type: "document.created", WorkspaceID: wid, ResourceID: d.NewID, After: map[string]any{"id": d.NewID, "title": d.Title, "tags": d.Tags}}); err != nil {
			respond(w, nil, err)
			return
		}
	}
	for _, a := range attachments {
		if err = r.Context().Err(); err != nil {
			return
		}
		a.Size = int64(len(assets[a.File]))
		a.ContentType = mime.TypeByExtension(strings.ToLower(path.Ext(a.Name)))
		if a.ContentType == "" {
			a.ContentType = "application/octet-stream"
		}
		protectedFile, protectionError := s.ProtectAttachmentTx(r.Context(), tx, current(r), a.OwnerNewID, wid, a.Name, a.ContentType, assets[a.File])
		if protectionError != nil {
			if !WriteProtectionError(w, protectionError) {
				respond(w, nil, protectionError)
			}
			return
		}
		a.Name = protectedFile.Name
		a.Size = int64(len(protectedFile.Data))
		a.Object, err = s.putStoredObject(r.Context(), provider, "attachments/"+a.NewID, bytes.NewReader(protectedFile.Data), 50<<20, a.ContentType)
		if err != nil {
			respond(w, nil, err)
			return
		}
		a.DiskPath = a.Object.Path
		_, err = tx.Exec(r.Context(), `INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path,storage_provider_id,object_key,checksum_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10)`, a.NewID, a.OwnerNewID, current(r).ID, a.Name, a.ContentType, a.Size, a.DiskPath, a.Object.ProviderID, a.Object.Key, a.Object.Checksum)
		if err != nil {
			respond(w, nil, err)
			return
		}
	}
	folders := 0
	for _, d := range docs {
		if d.Folder {
			folders++
		}
	}
	report := map[string]any{"imported": len(docs) - folders, "folders": folders, "attachments": len(attachments), "errors": []string{}}
	if len(input.databases) > 0 {
		if !hasIntegrationScope(current(r), "database:write") {
			apiError(w, 403, "Notion CSV 데이터베이스 가져오기 권한이 없습니다")
			return
		}
		ids := []string{}
		count := 0
		for name, data := range input.databases {
			id, e := s.importCSVTx(r.Context(), tx, current(r), wid, r.URL.Query().Get("space_id"), strings.TrimSuffix(path.Base(name), path.Ext(name)), data)
			if e != nil {
				respond(w, nil, e)
				return
			}
			ids = append(ids, id)
			count += len(data.Rows)
		}
		report["databases"] = len(ids)
		report["database_ids"] = ids
		report["rows"] = count
	}
	if err = s.recordMigrationEffect(r.Context(), tx, report); err != nil {
		respond(w, nil, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		respond(w, nil, err)
		return
	}
	committed = true
	s.audit(r, "WORKSPACE_IMPORT", wid, map[string]any{"imported": len(docs) - folders, "folders": folders, "attachments": len(attachments)})
	jsonResponse(w, 200, report)
}

func returnErrorVault(w http.ResponseWriter, message string) { apiError(w, 400, message) }
