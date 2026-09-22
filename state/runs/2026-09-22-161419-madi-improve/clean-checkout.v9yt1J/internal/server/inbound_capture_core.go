package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed inbound_capture.sql
var inboundCaptureSchema string

func (s *Server) migrateInboundCaptures(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, inboundCaptureSchema)
	return e
}

type inboundChannel struct {
	ID, UserID, WorkspaceID, Name, Kind string
	Config, Secrets, Checkpoint         map[string]any
	Enabled                             bool
	Revision                            int
}

func (s *Server) inboundChannel(ctx context.Context, id string) (inboundChannel, error) {
	c := inboundChannel{}
	var config, checkpoint []byte
	var encrypted string
	e := s.DB.QueryRow(ctx, "SELECT id::text,user_id::text,workspace_id::text,name,kind,config,secret_ciphertext,enabled,revision,checkpoint FROM inbound_capture_channels WHERE id=$1", id).Scan(&c.ID, &c.UserID, &c.WorkspaceID, &c.Name, &c.Kind, &config, &encrypted, &c.Enabled, &c.Revision, &checkpoint)
	if e != nil {
		return c, e
	}
	if json.Unmarshal(config, &c.Config) != nil || json.Unmarshal(checkpoint, &c.Checkpoint) != nil {
		return c, errors.New("수집 채널 설정을 읽지 못했습니다")
	}
	plain, e := s.decrypt(encrypted)
	if e != nil || json.Unmarshal([]byte(plain), &c.Secrets) != nil {
		return c, errors.New("수집 채널 비밀을 해독하지 못했습니다")
	}
	return c, nil
}
func inboundChannelPublic(c inboundChannel) map[string]any {
	return map[string]any{"id": c.ID, "workspace_id": c.WorkspaceID, "name": c.Name, "kind": c.Kind, "config": c.Config, "enabled": c.Enabled, "revision": c.Revision, "checkpoint": c.Checkpoint, "secrets": map[string]bool{"username_configured": str(c.Secrets, "username") != "", "password_configured": str(c.Secrets, "password") != "", "signing_secret_configured": str(c.Secrets, "signing_secret") != ""}}
}
func (s *Server) inboundOwner(ctx context.Context, c inboundChannel) (*Principal, error) {
	p, e := s.workerPrincipal(ctx, c.UserID, "", c.WorkspaceID)
	if e != nil || p.Kind != "user" || !s.canWorkspace(ctx, p, c.WorkspaceID, true) {
		return nil, jobPermanent("수집 소유자의 현재 문서 작성 권한이 없습니다")
	}
	return p, nil
}
func inboundCurrentTx(ctx context.Context, tx pgx.Tx, c inboundChannel) bool {
	var allowed bool
	e := tx.QueryRow(ctx, `SELECT c.enabled AND c.revision=$2 AND NOT u.disabled AND u.kind='user' AND u.role<>'viewer' AND m.role IN ('owner','admin','editor') AND CASE WHEN c.kind='imap' THEN f.imap_enabled ELSE f.hooks_enabled END FROM inbound_capture_channels c JOIN users u ON u.id=c.user_id JOIN workspace_members m ON m.user_id=c.user_id AND m.workspace_id=c.workspace_id JOIN inbound_capture_settings f ON f.id=1 WHERE c.id=$1 FOR SHARE OF c,u,m,f`, c.ID, c.Revision).Scan(&allowed)
	return e == nil && allowed
}
func validateInboundContent(c *inboundContent) error {
	c.Title = strings.TrimSpace(c.Title)
	if c.Title == "" {
		c.Title = "외부에서 수집한 메모"
	}
	if len([]rune(c.Title)) > 150 || len(c.Markdown) > 4<<20 || len(c.Attachments) > 20 || len(c.MessageID) > 998 || strings.ContainsAny(c.Title, "\r\n") {
		return errors.New("수집 제목·본문·첨부 제한을 확인하세요")
	}
	total := len(c.Markdown)
	for i := range c.Attachments {
		a := &c.Attachments[i]
		a.Name = inboundFilename(a.Name)
		if a.Type == "" {
			a.Type = "application/octet-stream"
		}
		if len(a.Type) > 200 || strings.ContainsAny(a.Type, "\r\n") {
			return errors.New("첨부 MIME 형식을 확인하세요")
		}
		total += len(a.Data)
	}
	if total > inboundCaptureMax {
		return errors.New("수집 본문과 첨부의 합은 10MB 이하여야 합니다")
	}
	return nil
}
func (s *Server) stageInboundContent(ctx context.Context, c inboundChannel, key string, content inboundContent) (string, bool, error) {
	if e := validateInboundContent(&content); e != nil {
		return "", false, e
	}
	if len(key) > 1100 || key == "" {
		return "", false, errors.New("수집 메시지 고유 키를 확인하세요")
	}
	p, e := s.inboundOwner(ctx, c)
	if e != nil {
		return "", false, e
	}
	ctx = context.WithValue(ctx, principalKey, p)
	// Message identifiers participate in deduplication, not user-visible content.
	// Retain a digest so an address-shaped Message-ID does not become plaintext PII.
	if content.MessageID != "" {
		content.MessageID = digest(content.MessageID)
	}
	raw := jsonValue(content)
	hash := digest(string(raw))
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", false, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,93))", c.UserID); e != nil {
		return "", false, e
	}
	if !inboundCurrentTx(ctx, tx, c) {
		return "", false, jobPermanent("수집 채널 정책이나 소유자 권한이 변경되었습니다")
	}
	var existing, oldHash string
	e = tx.QueryRow(ctx, `SELECT id::text,payload_hash FROM inbound_capture_messages WHERE channel_id=$1 AND (message_key=$2 OR (message_id<>'' AND message_id=$3))`, c.ID, key, content.MessageID).Scan(&existing, &oldHash)
	if e == nil {
		if oldHash != hash {
			return "", false, errors.New("같은 메시지 ID의 내용이 달라 수집하지 않았습니다")
		}
		return existing, true, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return "", false, e
	}
	if c.Kind == "hmac" {
		var recent int
		if e = tx.QueryRow(ctx, "SELECT count(*) FROM inbound_capture_messages WHERE channel_id=$1 AND created_at>now()-interval '1 minute'", c.ID).Scan(&recent); e != nil {
			return "", false, e
		}
		if recent >= 60 {
			return "", false, inboundRateLimit{}
		}
	}
	var bytesPending, count int
	e = tx.QueryRow(ctx, "SELECT coalesce(sum(m.payload_size),0),count(*) FROM inbound_capture_messages m JOIN inbound_capture_channels c ON c.id=m.channel_id WHERE c.user_id=$1 AND m.status='pending'", c.UserID).Scan(&bytesPending, &count)
	if e != nil {
		return "", false, e
	}
	if bytesPending+len(raw) > 100<<20 || count >= 100 {
		return "", false, errors.New("개인 수집 대기 한도 100MB 또는 100건을 초과했습니다")
	}
	id := newID()
	protected, e := s.ProtectDocumentTx(ctx, tx, p, "", c.WorkspaceID, content.Title, content.Markdown)
	if e != nil {
		return "", false, e
	}
	content.Title, content.Markdown = protected.Title, protected.Markdown
	metadata, e := s.ProtectDocumentMetadataTx(ctx, tx, p, "", c.WorkspaceID, content.Source)
	if e != nil {
		return "", false, e
	}
	if metadata.Changed {
		content.Source = metadata.Value.(map[string]any)
	}
	for i, attachment := range content.Attachments {
		file, e := s.ProtectAttachmentTx(ctx, tx, p, "", c.WorkspaceID, attachment.Name, attachment.Type, attachment.Data)
		if e != nil {
			return "", false, e
		}
		content.Attachments[i].Name = file.Name
		content.Attachments[i].Data = file.Data
	}
	// Mask before the encrypted staging payload is persisted, not only after
	// the future worker materializes a document. The receipt keeps input hash.
	if e = validateInboundContent(&content); e != nil {
		return "", false, e
	}
	raw = jsonValue(content)
	if bytesPending+len(raw) > 100<<20 {
		return "", false, errors.New("정제 후 개인 수집 대기 한도 100MB를 초과했습니다")
	}
	cipher, e := s.encrypt(string(raw))
	if e != nil {
		return "", false, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO inbound_capture_messages(id,channel_id,message_key,message_id,payload_hash,payload_ciphertext,payload_size,channel_revision,title) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, c.ID, key, content.MessageID, hash, cipher, len(raw), c.Revision, content.Title)
	if e != nil {
		return "", false, e
	}
	jid, e := s.EnqueueJob(ctx, tx, "capture.ingest", c.UserID, c.WorkspaceID, map[string]any{"message_id": id})
	if e != nil {
		return "", false, e
	}
	_, e = tx.Exec(ctx, "UPDATE inbound_capture_messages SET job_id=$2 WHERE id=$1", id, jid)
	if e != nil {
		return "", false, e
	}
	return id, false, tx.Commit(ctx)
}
func (s *Server) ingestInboundCapture(ctx context.Context, j Job) (map[string]any, error) {
	mid := str(j.Payload, "message_id")
	var cid string
	e := s.DB.QueryRow(ctx, "SELECT channel_id::text FROM inbound_capture_messages WHERE id=$1 AND job_id=$2", mid, j.ID).Scan(&cid)
	if e != nil {
		return nil, jobPermanent("수집 작업 원본을 찾을 수 없습니다")
	}
	c, e := s.inboundChannel(ctx, cid)
	if e != nil {
		return nil, jobPermanent("수집 채널을 찾을 수 없습니다")
	}
	p, e := s.inboundOwner(ctx, c)
	if e != nil {
		return nil, e
	}
	if p.ID != j.OwnerID {
		return nil, jobPermanent("수집 소유자가 다릅니다")
	}
	provider, e := s.resolveStorage(ctx, c.WorkspaceID)
	if e != nil {
		return nil, e
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	// Use the same tree lock order as ordinary create/move, before message rows.
	request := (&http.Request{}).WithContext(context.WithValue(ctx, principalKey, p))
	did := newID()
	if e = s.treePlacementTx(request, tx, "documents", c.WorkspaceID, did, ""); e != nil {
		return nil, e
	}
	var cipher, hash, status, existing string
	var revision int
	e = tx.QueryRow(ctx, "SELECT payload_ciphertext,payload_hash,status,coalesce(document_id::text,''),channel_revision FROM inbound_capture_messages WHERE id=$1 FOR UPDATE", mid).Scan(&cipher, &hash, &status, &existing, &revision)
	if e != nil {
		return nil, e
	}
	if status == "completed" {
		return map[string]any{"document_id": existing, "deduplicated": true}, nil
	}
	if status != "pending" {
		return nil, jobPermanent("수집이 취소되거나 거부되었습니다")
	}
	if revision != c.Revision || !inboundCurrentTx(ctx, tx, c) {
		_, e = tx.Exec(ctx, "UPDATE inbound_capture_messages SET status='cancelled',message='수집 채널 또는 현재 권한이 변경되었습니다',completed_at=now() WHERE id=$1", mid)
		if e == nil {
			e = tx.Commit(ctx)
		}
		if e != nil {
			return nil, e
		}
		return map[string]any{"cancelled": true}, nil
	}
	plain, e := s.decrypt(cipher)
	if e != nil {
		return nil, jobPermanent("수집 원본을 해독하지 못했습니다")
	}
	content := inboundContent{}
	if json.Unmarshal([]byte(plain), &content) != nil || validateInboundContent(&content) != nil {
		return nil, jobPermanent("수집 원본 형식이 잘못되었습니다")
	}
	type file struct {
		id         string
		attachment inboundAttachment
		object     storedObject
	}
	files := []file{}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = tx.Rollback(ctx)
		for _, f := range files {
			check, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			var exists bool
			if err := s.DB.QueryRow(check, "SELECT EXISTS(SELECT 1 FROM attachments WHERE id=$1)", f.id).Scan(&exists); err == nil && !exists {
				_ = s.cleanupStoredObject(check, f.object)
			}
			cancel()
		}
	}()
	markdown := content.Markdown
	for _, attachment := range content.Attachments {
		id := newID()
		protectedFile, protectionError := s.ProtectAttachmentTx(ctx, tx, p, did, c.WorkspaceID, attachment.Name, attachment.Type, attachment.Data)
		if protectionError != nil {
			return nil, protectionError
		}
		attachment.Name, attachment.Data = protectedFile.Name, protectedFile.Data
		object, e := s.putStoredObject(ctx, provider, "attachments/"+id, bytes.NewReader(attachment.Data), inboundCaptureMax, attachment.Type)
		files = append(files, file{id, attachment, object})
		if e != nil {
			return nil, errors.New("수집 첨부파일을 저장하지 못했습니다")
		}
		label := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]").Replace(attachment.Name)
		markdown += "\n\n[" + label + "](/api/v1/attachments/" + id + ")"
	}
	if len(markdown) > 4<<20 {
		return nil, jobPermanent("첨부 링크를 포함한 문서가 4MB를 초과했습니다")
	}
	protected, e := s.ProtectDocumentTx(ctx, tx, p, did, c.WorkspaceID, content.Title, markdown)
	if e != nil {
		return nil, e
	}
	content.Title, markdown = protected.Title, protected.Markdown
	_, e = tx.Exec(ctx, "INSERT INTO documents(id,workspace_id,title,markdown,visibility,owner_id,tags,aliases,block_metadata) VALUES($1,$2,$3,$4,'private',$5,'[]','[]','{}')", did, c.WorkspaceID, content.Title, markdown, p.ID)
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,owner_id,block_metadata FROM documents WHERE id=$1", did)
	if e != nil {
		return nil, e
	}
	source := content.Source
	if source == nil {
		source = map[string]any{}
	}
	source["origin"] = c.Kind
	source["capture_channel_id"] = c.ID
	protectedSource, e := s.ProtectDocumentMetadataTx(ctx, tx, p, did, c.WorkspaceID, source)
	if e != nil {
		return nil, e
	}
	if protectedSource.Changed {
		source = protectedSource.Value.(map[string]any)
	}
	_, e = tx.Exec(ctx, "INSERT INTO knowledge_document_meta(document_id,kind,system_metadata) VALUES($1,'inbox',$2)", did, jsonValue(source))
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "INSERT INTO capture_receipts(user_id,request_id,workspace_id,document_id,payload_hash) VALUES($1,$2,$3,$4,$5)", p.ID, mid, c.WorkspaceID, did, hash)
	if e != nil {
		return nil, e
	}
	for _, f := range files {
		_, e = tx.Exec(ctx, "INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path,storage_provider_id,object_key,checksum_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10)", f.id, did, p.ID, f.attachment.Name, f.attachment.Type, f.object.Size, f.object.Path, f.object.ProviderID, f.object.Key, f.object.Checksum)
		if e != nil {
			return nil, e
		}
		_, e = tx.Exec(ctx, "INSERT INTO attachment_receipts(user_id,request_id,document_id,attachment_id,payload_hash) VALUES($1,$2,$3,$2,$4)", p.ID, f.id, did, digest(did+"\n"+f.attachment.Name+"\n"+f.object.Checksum))
		if e != nil {
			return nil, e
		}
	}
	// Private by construction; automation sees current owner ACL, never an admin bypass.
	if e = s.enqueueEvent(ctx, tx, Event{Type: "document.created", WorkspaceID: c.WorkspaceID, ActorID: p.ID, ResourceID: did, After: map[string]any{"id": did, "title": content.Title, "visibility": "private", "status": "draft", "version": 1}}); e != nil {
		return nil, e
	}
	_, e = tx.Exec(ctx, "UPDATE inbound_capture_messages SET document_id=$2,status='completed',message='',completed_at=now() WHERE id=$1", mid, did)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	committed = true
	s.audit(request, "INBOUND_CAPTURE", did, map[string]any{"channel_id": c.ID, "attachments": len(files)})
	return map[string]any{"document_id": did, "attachments": len(files)}, nil
}
