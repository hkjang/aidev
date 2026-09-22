package server

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/yaml.v3"
)

func gitSyncContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func gitSyncSafeFile(name string) bool {
	if !safeVaultPath(name) || !utf8.ValidString(name) || strings.ContainsAny(name, "\r\n\t") || len(name) > 1000 || strings.Count(name, "/") > 30 {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if oneOf(strings.ToLower(part), ".git", ".gitmodules", ".gitattributes") {
			return false
		}
	}
	return true
}

type gitSyncFileResponse struct {
	header http.Header
	status int
	writer io.Writer
	err    error
}

func (w *gitSyncFileResponse) Header() http.Header    { return w.header }
func (w *gitSyncFileResponse) WriteHeader(status int) { w.status = status }
func (w *gitSyncFileResponse) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, e := w.writer.Write(p)
	if e != nil {
		w.err = e
	}
	return n, e
}

func (s *Server) gitSyncBuildExport(ctx context.Context, p *Principal, c gitSyncConnection, ids []string, maxBytes int64) (gitSyncSnapshot, error) {
	result := gitSyncSnapshot{Files: []gitSyncFile{}, SelectedDocumentIDs: append([]string{}, ids...), SourceVersions: map[string]int64{}}
	if len(ids) == 0 || len(ids) > 1000 {
		return result, errors.New("전송할 문서를 1~1000개 명시적으로 선택하세요")
	}
	if maxBytes < 1 || maxBytes > 100<<20 {
		return result, errGitSyncLimit
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !validID(id) || seen[id] {
			return result, errors.New("전송 문서 선택이 올바르지 않습니다")
		}
		seen[id] = true
	}
	f, e := os.CreateTemp("", "madi-git-export-*.zip")
	if e != nil {
		return result, e
	}
	defer f.Close()
	defer os.Remove(f.Name())
	ctx = context.WithValue(ctx, principalKey, p)
	ctx = context.WithValue(ctx, vaultExportSelectionKey{}, vaultExportSelection{IDs: ids, AdditionalActorID: c.OwnerID, MaxBytes: maxBytes})
	request := httptest.NewRequest("GET", "/api/v1/export?workspace_id="+c.WorkspaceID, nil).WithContext(ctx)
	response := &gitSyncFileResponse{header: http.Header{}, status: 200, writer: &gitLimitWriter{ctx: ctx, out: f, remaining: maxBytes}}
	s.exportMarkdown(response, request)
	if response.err != nil {
		return result, response.err
	}
	if response.status != 200 {
		return result, errors.New("선택 문서의 현재 권한·크기·저장소 상태를 확인하세요")
	}
	stat, e := f.Stat()
	if e != nil {
		return result, e
	}
	archive, e := zip.NewReader(f, stat.Size())
	if e != nil {
		return result, errors.New("Git 전송용 Markdown 내보내기를 읽지 못했습니다")
	}
	if len(archive.File) > 5000 {
		return result, errGitSyncLimit
	}
	contents := map[string][]byte{}
	var total int64
	for _, file := range archive.File {
		if !gitSyncSafeFile(file.Name) || file.UncompressedSize64 > uint64(gitSyncMaxObjectBytes) {
			return result, errors.New("Git 전송 파일 경로 또는 크기가 허용 범위를 벗어났습니다")
		}
		reader, e := file.Open()
		if e != nil {
			return result, e
		}
		data, e := io.ReadAll(io.LimitReader(contextVaultReader{ctx, reader}, gitSyncMaxObjectBytes+1))
		reader.Close()
		if e != nil {
			return result, e
		}
		total += int64(len(data))
		if len(data) > int(gitSyncMaxObjectBytes) || total > maxBytes {
			return result, errGitSyncLimit
		}
		contents[file.Name] = data
	}
	var manifest vaultManifest
	if json.Unmarshal(contents["madi-manifest.json"], &manifest) != nil || len(manifest.Documents) != len(ids) {
		return result, errors.New("Git 전송 문서 manifest를 확인할 수 없습니다")
	}
	docFiles := map[string]*vaultDocument{}
	attachmentFiles := map[string]*vaultAttachment{}
	for _, doc := range manifest.Documents {
		if !seen[doc.ID] || doc.Version < 1 {
			return result, errors.New("Git 전송 문서 버전을 확인할 수 없습니다")
		}
		result.SourceVersions[doc.ID] = doc.Version
		docFiles[doc.File] = doc
		data, ok := contents[doc.File]
		if !ok {
			return result, errors.New("Git 전송 문서 원문이 없습니다")
		}
		fm, e := parseFrontMatter(string(data))
		if e != nil {
			return result, errors.New("Git 전송 Front Matter를 확인하세요")
		}
		fm["id"], fm["title"], fm["tags"], fm["aliases"] = doc.ID, doc.Title, doc.Tags, doc.Aliases
		fm["visibility"], fm["status"], fm["madi_source_version"] = doc.Visibility, doc.Status, doc.Version
		header, e := yaml.Marshal(fm)
		if e != nil {
			return result, errors.New("Git 전송 Front Matter를 작성할 수 없습니다")
		}
		_, body := collaborationFrontMatter(string(data))
		canonical := []byte("---\n" + string(header) + "---\n" + body)
		total += int64(len(canonical) - len(data))
		if total > maxBytes || len(canonical) > 4<<20 {
			return result, errGitSyncLimit
		}
		contents[doc.File] = canonical
	}
	for _, attachment := range manifest.Attachments {
		attachmentFiles[attachment.File] = attachment
	}
	for name, data := range contents {
		// Git uses per-document Front Matter. A shared manifest would replace
		// metadata for documents intentionally omitted from this selection.
		if name == "madi-manifest.json" {
			continue
		}
		file := gitSyncFile{Path: path.Join(str(c.Config, "prefix"), name), Data: data, Hash: gitSyncContentHash(data)}
		if doc := docFiles[name]; doc != nil {
			file.DocumentID = doc.ID
			file.ContentType = "text/markdown"
		}
		if attachment := attachmentFiles[name]; attachment != nil {
			file.AttachmentID = attachment.ID
			file.DocumentID = attachment.DocumentID
			file.ContentType = attachment.ContentType
		}
		result.Files = append(result.Files, file)
	}
	mappings, e := s.gitSyncMappings(ctx, c.ID)
	if e != nil {
		return result, e
	}
	if e = gitSyncRemapExport(&result, mappings, str(c.Config, "prefix")); e != nil {
		return result, e
	}
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	return result, nil
}

// A remote path is a stable identity after first explicit synchronization.
// Renaming a title must not silently create a duplicate or delete another path.
func gitSyncRemapExport(snapshot *gitSyncSnapshot, mappings map[string]gitSyncMapping, prefix string) error {
	identities := map[string]string{}
	for name, m := range mappings {
		key := m.DocumentID
		if m.AttachmentID != "" {
			key = m.AttachmentID
		}
		if key == "" {
			continue
		}
		if previous := identities[key]; previous != "" && previous != name {
			return errors.New("동일 원본에 여러 Git 경로가 연결되어 수동 정리가 필요합니다")
		}
		identities[key] = name
	}
	remap := map[string]string{}
	used := map[string]bool{}
	for _, file := range snapshot.Files {
		key := file.DocumentID
		if file.AttachmentID != "" {
			key = file.AttachmentID
		}
		name := identities[key]
		if name == "" {
			name = file.Path
		}
		if !gitSyncSafeFile(name) || !strings.HasPrefix(name, prefix+"/") || used[strings.ToLower(name)] {
			return errors.New("Git 고정 경로 또는 대소문자 충돌을 확인하세요")
		}
		used[strings.ToLower(name)] = true
		remap[file.Path] = name
	}
	for i := range snapshot.Files {
		file := &snapshot.Files[i]
		oldPath, newPath := file.Path, remap[file.Path]
		if file.ContentType == "text/markdown" {
			md := rewriteVaultContent(string(file.Data), func(segment string) string {
				segment = rewriteVaultMarkdownLinks(segment, func(destination string) string {
					u, e := url.Parse(destination)
					if e != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") {
						return destination
					}
					target := path.Clean(path.Join(path.Dir(oldPath), u.Path))
					if moved := remap[target]; moved != "" {
						relative, e := filepath.Rel(filepath.FromSlash(path.Dir(newPath)), filepath.FromSlash(moved))
						if e == nil {
							u.Path = filepath.ToSlash(relative)
							u.RawPath = ""
							return u.String()
						}
					}
					return destination
				})
				return vaultWikiPattern.ReplaceAllStringFunc(segment, func(match string) string {
					parts := vaultWikiPattern.FindStringSubmatch(match)
					target, fragment, label := splitVaultWiki(parts[2])
					withExt := target
					if !strings.HasSuffix(strings.ToLower(withExt), ".md") {
						withExt += ".md"
					}
					if moved := remap[path.Join(prefix, withExt)]; moved != "" {
						return parts[1] + "[[" + strings.TrimSuffix(strings.TrimPrefix(moved, prefix+"/"), ".md") + fragment + label + "]]"
					}
					return match
				})
			})
			file.Data = []byte(md)
			file.Hash = gitSyncContentHash(file.Data)
		}
		file.Path = newPath
	}
	return nil
}

func gitSyncRemoteFiles(ctx context.Context, repo gitSyncRepository, prefix string, maxBytes int64) (map[string]gitSyncFile, error) {
	files := map[string]gitSyncFile{}
	if repo.Head.IsZero() {
		return files, nil
	}
	commit, e := object.GetCommit(repo.Store, repo.Head)
	if e != nil {
		return nil, e
	}
	root, e := commit.Tree()
	if e != nil {
		return nil, e
	}
	subtree, e := root.Tree(prefix)
	if errors.Is(e, object.ErrDirectoryNotFound) {
		return files, nil
	}
	if e != nil {
		return nil, errors.New("Git 전용 폴더가 디렉터리가 아닙니다")
	}
	var total int64
	var visit func(*object.Tree, string, int) error
	visit = func(tree *object.Tree, base string, depth int) error {
		if depth > 30 {
			return errGitSyncLimit
		}
		for _, entry := range tree.Entries {
			if e := ctx.Err(); e != nil {
				return e
			}
			name := base + "/" + entry.Name
			if strings.Contains(entry.Name, "/") || !gitSyncSafeFile(name) {
				return errors.New("Git 전용 폴더에 안전하지 않은 경로가 있습니다")
			}
			if entry.Mode == filemode.Dir {
				child, e := object.GetTree(repo.Store, entry.Hash)
				if e != nil {
					return e
				}
				if e = visit(child, name, depth+1); e != nil {
					return e
				}
				continue
			}
			if entry.Mode != filemode.Regular {
				return errors.New("Git 가져오기는 일반 파일만 허용하며 심볼릭 링크·서브모듈·실행 파일은 제외합니다")
			}
			blob, e := object.GetBlob(repo.Store, entry.Hash)
			if e != nil {
				return e
			}
			if blob.Size > gitSyncMaxObjectBytes {
				return errGitSyncLimit
			}
			total += blob.Size
			if total > maxBytes || len(files) >= 5000 {
				return errGitSyncLimit
			}
			r, e := blob.Reader()
			if e != nil {
				return e
			}
			data, e := io.ReadAll(io.LimitReader(contextVaultReader{ctx, r}, gitSyncMaxObjectBytes+1))
			r.Close()
			if e != nil {
				return e
			}
			if int64(len(data)) != blob.Size {
				return errors.New("Git 파일 크기가 선언과 다릅니다")
			}
			files[name] = gitSyncFile{Path: name, Data: data, Hash: gitSyncContentHash(data)}
		}
		return nil
	}
	if e = visit(subtree, prefix, 0); e != nil {
		return nil, e
	}
	return files, nil
}
