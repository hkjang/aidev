package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

func (s *Server) gitSyncBuildPull(ctx context.Context, p *Principal, c gitSyncConnection, remote map[string]gitSyncFile, mappings map[string]gitSyncMapping, paths []string, limit int64) (gitSyncSnapshot, map[string]any, error) {
	snapshot := gitSyncSnapshot{Files: []gitSyncFile{}, SourceVersions: map[string]int64{}, SelectedDocumentIDs: []string{}}
	report := map[string]any{"files": []gitSyncChange{}, "has_conflicts": false, "selection_required": len(paths) == 0, "remote_visibility_warning": "원격 Git 권한·문서 ID·공개 범위는 상속하지 않습니다. 새 문서는 요청자 개인 초안으로 수신하며 원격 삭제는 자동 적용하지 않습니다. 첨부파일은 선택한 첫 Markdown 문서에 연결됩니다. 실패 전 저장된 항목은 이력에 남습니다"}
	if c.OwnerID != p.ID {
		return snapshot, report, errors.New("개인 문서를 자동 공유하지 않기 위해 Git 수신 연결의 실행 계정은 현재 요청자와 같아야 합니다")
	}
	selected := map[string]bool{}
	for _, name := range paths {
		if selected[name] || !gitSyncSafeFile(name) {
			return snapshot, report, errors.New("원격 선택 경로를 확인하세요")
		}
		if _, ok := remote[name]; !ok {
			return snapshot, report, errors.New("선택한 원격 파일이 없습니다")
		}
		selected[name] = true
	}
	ids := []string{}
	idSet := map[string]bool{}
	for name := range selected {
		m := mappings[name]
		if m.DocumentID != "" && !idSet[m.DocumentID] {
			if !s.canDocument(ctx, p, m.DocumentID, true) {
				return snapshot, report, errors.New("기존 수신 문서의 수정 권한을 확인하세요")
			}
			ids = append(ids, m.DocumentID)
			idSet[m.DocumentID] = true
		}
	}
	localFiles := map[string]gitSyncFile{}
	if len(ids) > 0 {
		local, e := s.gitSyncBuildExport(ctx, p, c, ids, limit)
		if e != nil {
			return snapshot, report, e
		}
		snapshot.SourceVersions = local.SourceVersions
		for _, f := range local.Files {
			key := f.DocumentID
			if f.AttachmentID != "" {
				key = f.AttachmentID
			} else {
				f.Hash = fmt.Sprintf("v:%d", local.SourceVersions[f.DocumentID])
			}
			localFiles[key] = f
		}
	}
	changes := []gitSyncChange{}
	names := []string{}
	for name := range remote {
		names = append(names, name)
	}
	sort.Strings(names)
	var total int64
	markdownCount := 0
	for _, name := range names {
		file := remote[name]
		change := gitSyncChange{Path: name, Size: len(file.Data), Action: "available"}
		if selected[name] {
			total += int64(len(file.Data))
			if total > limit {
				return snapshot, report, errGitSyncLimit
			}
			mapped, exists := mappings[name]
			file.DocumentID, file.AttachmentID = mapped.DocumentID, mapped.AttachmentID
			isMarkdown := oneOf(strings.ToLower(path.Ext(name)), ".md", ".markdown")
			if isMarkdown {
				markdownCount++
				if !utf8.Valid(file.Data) || len(file.Data) > 4<<20 {
					return snapshot, report, errors.New("수신 Markdown은 UTF-8 4MB 이하여야 합니다")
				}
				if _, e := parseFrontMatter(string(file.Data)); e != nil {
					return snapshot, report, errors.New("수신 Markdown Front Matter가 올바르지 않습니다")
				}
				file.ContentType = "text/markdown"
			} else {
				file.ContentType = "application/octet-stream"
			}
			change.Action = "create"
			change.DocumentID = file.DocumentID
			change.AttachmentID = file.AttachmentID
			if exists {
				key := mapped.DocumentID
				if mapped.AttachmentID != "" {
					key = mapped.AttachmentID
				}
				local, ok := localFiles[key]
				switch {
				case !ok:
					change.Action = "conflict"
					change.Reason = "기존 로컬 문서 또는 첨부파일이 없습니다"
				case mapped.LocalHash != local.Hash:
					change.Action = "conflict"
					change.Reason = "로컬 원본이 마지막 동기화 이후 변경되었습니다"
				case mapped.RemoteHash == file.Hash:
					change.Action = "unchanged"
				default:
					change.Action = "update"
				}
			}
			if change.Action == "conflict" {
				report["has_conflicts"] = true
			}
			snapshot.Files = append(snapshot.Files, file)
		}
		changes = append(changes, change)
	}
	if len(selected) > 0 && markdownCount == 0 {
		return snapshot, report, errors.New("첨부를 연결할 Markdown 문서를 하나 이상 선택하세요")
	}
	report["files"] = changes
	return snapshot, report, nil
}

func gitSyncCleanInboundMarkdown(file gitSyncFile) (string, string, error) {
	fm, e := parseFrontMatter(string(file.Data))
	if e != nil {
		return "", "", e
	}
	title := strings.TrimSpace(str(fm, "title"))
	if title == "" {
		title = strings.TrimSuffix(path.Base(file.Path), path.Ext(file.Path))
	}
	if len(title) > 500 {
		return "", "", errors.New("원격 문서 제목은 500바이트 이하여야 합니다")
	}
	for _, key := range []string{"id", "owner", "owner_id", "workspace", "workspace_id", "space_id", "parent_id", "visibility", "status", "madi_source_version", "permissions", "shared_with"} {
		delete(fm, key)
	}
	fm["title"] = title
	header, e := yaml.Marshal(fm)
	if e != nil {
		return "", "", e
	}
	_, body := collaborationFrontMatter(string(file.Data))
	return title, "---\n" + string(header) + "---\n" + body, nil
}

func (s *Server) gitSyncApplyPull(ctx context.Context, j Job, run gitSyncRun, c gitSyncConnection, p *Principal, snapshot gitSyncSnapshot) (map[string]any, error) {
	if p.ID != c.OwnerID {
		return nil, jobPermanent("수신 실행 계정은 현재 요청자여야 합니다")
	}
	// Local writes are checkpointed per immutable file, unlike remote
	// receive-pack. Every content mutation and mapping commits in one transaction.
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	if e = s.gitSyncGuardTx(ctx, tx, run, c, p, true); e == nil {
		tag, err := tx.Exec(ctx, `UPDATE git_sync_runs SET status='running',revision=revision+1,updated_at=now() WHERE id=$1 AND status='queued'`, run.ID)
		e = err
		if e == nil && tag.RowsAffected() != 1 {
			e = jobPermanent("Git 수신이 취소되거나 이미 실행되었습니다")
		}
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	_ = tx.Rollback(ctx)
	if e != nil {
		return nil, e
	}
	docs := []gitSyncFile{}
	attachments := []gitSyncFile{}
	for _, f := range snapshot.Files {
		if f.ContentType == "text/markdown" {
			docs = append(docs, f)
		} else {
			attachments = append(attachments, f)
		}
	}
	if len(docs) == 0 {
		return nil, jobPermanent("수신할 Markdown 문서가 없습니다")
	}
	docIDs := map[string]string{}
	links := map[string]string{}
	mappings, e := s.gitSyncMappings(ctx, c.ID)
	if e != nil {
		return nil, e
	}
	effectIndex := 0
	apply := func(file gitSyncFile, attachment bool, payload map[string]any, data []byte) (map[string]any, error) {
		index := effectIndex
		effectIndex++
		var old []byte
		e := s.DB.QueryRow(ctx, "SELECT result FROM automation_effects WHERE job_id=$1 AND action_index=$2", j.ID, index).Scan(&old)
		if e == nil {
			var result map[string]any
			if json.Unmarshal(old, &result) != nil {
				return nil, errors.New("Git 수신 체크포인트를 읽을 수 없습니다")
			}
			return result, nil
		}
		if e != pgx.ErrNoRows {
			return nil, e
		}
		fresh, e := gitSyncScanRun(s.DB.QueryRow(ctx, "SELECT "+gitSyncRunSelect+" FROM git_sync_runs WHERE id=$1", run.ID))
		if e != nil {
			return nil, e
		}
		mutationCtx := context.WithValue(ctx, automationEffectKey{}, automationEffectContext{JobID: j.ID, Index: index})
		mutationCtx = withMutationResultGuard(mutationCtx, func(guardCtx context.Context, tx pgx.Tx, result map[string]any) error {
			var status string
			if e := tx.QueryRow(guardCtx, "SELECT status FROM git_sync_runs WHERE id=$1 FOR UPDATE", run.ID).Scan(&status); e != nil || status != "running" {
				return jobPermanent("Git 수신이 취소되거나 변경되었습니다")
			}
			id := str(result, "id")
			docID := id
			if attachment {
				docID = str(result, "document_id")
			} else {
				fresh.SourceVersions[id] = int64(number(result, "version", 1))
			}
			if e := s.gitSyncGuardTx(guardCtx, tx, fresh, c, p, true); e != nil {
				return e
			}
			attachmentID := ""
			localHash := fmt.Sprintf("v:%d", fresh.SourceVersions[docID])
			if attachment {
				attachmentID = id
				localHash = str(result, "checksum_sha256")
			}
			_, e := tx.Exec(guardCtx, `INSERT INTO git_sync_mappings(connection_id,path,document_id,attachment_id,local_hash,remote_hash) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6) ON CONFLICT(connection_id,path) DO UPDATE SET document_id=EXCLUDED.document_id,attachment_id=EXCLUDED.attachment_id,local_hash=EXCLUDED.local_hash,remote_hash=EXCLUDED.remote_hash,updated_at=now()`, c.ID, file.Path, docID, attachmentID, localHash, file.Hash)
			if e != nil {
				return e
			}
			_, e = tx.Exec(guardCtx, "UPDATE git_sync_runs SET source_versions=$2,updated_at=now() WHERE id=$1", run.ID, jsonValue(fresh.SourceVersions))
			return e
		})
		recorder := httptest.NewRecorder()
		request := automationRequest(mutationCtx, p, "POST", payload)
		if attachment {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, e := writer.CreateFormFile("file", path.Base(file.Path))
			if e != nil {
				return nil, e
			}
			if _, e = part.Write(data); e != nil {
				return nil, e
			}
			if e = writer.Close(); e != nil {
				return nil, e
			}
			request = httptest.NewRequest("POST", "/api/v1/attachments?document_id="+str(payload, "document_id"), &body).WithContext(context.WithValue(mutationCtx, principalKey, p))
			request.Header.Set("Content-Type", writer.FormDataContentType())
			s.uploadAttachment(recorder, request)
		} else if file.DocumentID == "" {
			s.createDocument(recorder, request)
		} else {
			payload["version"] = int(fresh.SourceVersions[file.DocumentID])
			s.saveDocument(recorder, request, file.DocumentID, payload)
		}
		return automationResponse(recorder)
	}
	for _, file := range docs {
		if m, ok := mappings[file.Path]; ok && m.RemoteHash == file.Hash && m.DocumentID != "" {
			docIDs[file.Path] = m.DocumentID
			links[file.Path] = "/app/documents/" + m.DocumentID
			continue
		}
		title, md, e := gitSyncCleanInboundMarkdown(file)
		if e != nil {
			return nil, e
		}
		payload := map[string]any{"title": title, "markdown": md, "status": "draft"}
		if file.DocumentID == "" {
			payload["workspace_id"], payload["space_id"], payload["visibility"] = c.WorkspaceID, c.SpaceID, "private"
		}
		result, e := apply(file, false, payload, nil)
		if e != nil {
			return nil, e
		}
		id := str(result, "id")
		docIDs[file.Path] = id
		links[file.Path] = "/app/documents/" + id
	}
	for _, file := range attachments {
		if m, ok := mappings[file.Path]; ok && m.RemoteHash == file.Hash && m.AttachmentID != "" {
			links[file.Path] = "/api/v1/attachments/" + m.AttachmentID
			continue
		}
		docID := file.DocumentID
		if docID == "" {
			docID = docIDs[docs[0].Path]
		}
		result, e := apply(file, true, map[string]any{"document_id": docID}, file.Data)
		if e != nil {
			return nil, e
		}
		links[file.Path] = "/api/v1/attachments/" + str(result, "id")
	}
	for _, file := range docs {
		title, md, e := gitSyncCleanInboundMarkdown(file)
		if e != nil {
			return nil, e
		}
		rewritten := rewriteVaultContent(md, func(line string) string {
			return rewriteVaultMarkdownLinks(line, func(destination string) string {
				u, e := url.Parse(destination)
				if e != nil || u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") {
					return destination
				}
				target := path.Clean(path.Join(path.Dir(file.Path), u.Path))
				if replacement := links[target]; replacement != "" {
					if u.Fragment != "" {
						replacement += "#" + u.Fragment
					}
					return replacement
				}
				return destination
			})
		})
		if rewritten != md {
			file.DocumentID = docIDs[file.Path]
			var existing string
			if e := s.DB.QueryRow(ctx, "SELECT markdown FROM documents WHERE id=$1 AND madi_document_allowed($2,id,true)", file.DocumentID, p.ID).Scan(&existing); e != nil {
				return nil, e
			}
			if existing == rewritten {
				continue
			}
			if _, e = apply(file, false, map[string]any{"title": title, "markdown": rewritten, "status": "draft"}, nil); e != nil {
				return nil, e
			}
		}
	}
	// Each mutation already recorded its canonical version/checksum baseline in
	// the same transaction. Completion only marks this bounded batch finished.
	ids := []string{}
	for _, id := range docIDs {
		ids = append(ids, id)
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	tag, e := tx.Exec(ctx, "UPDATE git_sync_runs SET status='succeeded',report=report||$2::jsonb,revision=revision+1,updated_at=now() WHERE id=$1 AND status='running'", run.ID, jsonValue(map[string]any{"received_documents": len(ids), "received_attachments": len(attachments)}))
	if e == nil && tag.RowsAffected() != 1 {
		e = errors.New("Git 수신 완료 전에 작업이 취소되거나 변경되었습니다")
	}
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE git_sync_connections SET last_remote_commit=$2 WHERE id=$1", c.ID, run.RemoteCommit)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	return map[string]any{"run_id": run.ID, "status": "succeeded", "documents": len(ids)}, e
}
