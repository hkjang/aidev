package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

type distributionExport struct {
	ID, WorkspaceID, OwnerID, ExportID, KeyID, Receiver, Status, JobID string
	PolicyRevision                                                     int
	CreatedAt, ExpiresAt                                               time.Time
	Documents                                                          []distributionDocumentRef
}
type distributionDocumentRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

func (s *Server) readDistributionExport(ctx context.Context, id string) (distributionExport, error) {
	var v distributionExport
	var refs []byte
	e := s.DB.QueryRow(ctx, `SELECT id::text,workspace_id::text,owner_id::text,coalesce(export_id::text,''),signing_key_id::text,receiver_instance::text,status,coalesce(job_id::text,''),policy_revision,created_at,expires_at,document_refs FROM knowledge_distribution_exports WHERE id=$1`, id).Scan(&v.ID, &v.WorkspaceID, &v.OwnerID, &v.ExportID, &v.KeyID, &v.Receiver, &v.Status, &v.JobID, &v.PolicyRevision, &v.CreatedAt, &v.ExpiresAt, &refs)
	if e == nil {
		e = json.Unmarshal(refs, &v.Documents)
	}
	return v, e
}
func distributionSessionTx(ctx context.Context, tx pgx.Tx, run exportRun) error {
	var role string
	var deadline time.Time
	if run.TokenID != "" || run.TokenBound || run.SessionHash == "" || !run.Expires.After(time.Now()) {
		return errors.New("현재 개인 로그인으로 준비한 내보내기가 필요합니다")
	}
	if tx.QueryRow(ctx, `SELECT role FROM users WHERE id=$1 AND NOT disabled FOR SHARE`, run.OwnerID).Scan(&role) != nil || tx.QueryRow(ctx, `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 FOR SHARE`, run.WorkspaceID, run.OwnerID).Scan(&role) != nil {
		return errors.New("현재 계정·워크스페이스 접근 권한이 없습니다")
	}
	if tx.QueryRow(ctx, `SELECT expires_at FROM sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>clock_timestamp() FOR SHARE`, run.OwnerID, run.SessionHash).Scan(&deadline) != nil || !deadline.After(time.Now()) {
		return errors.New("내보내기를 요청한 로그인 세션이 만료되었습니다")
	}
	return nil
}

// Source resources precede policies/credentials in the lock order, as in normal
// document saves and approvals. This is a short snapshot check, not a network
// call or a permission cache. Exact file hashes supplement the export revision.
func (s *Server) distributionSourcesTx(ctx context.Context, tx pgx.Tx, p *Principal, run exportRun, m *distributionManifest, check bool, files map[string][]byte) error {
	if run.Format != "markdown" || run.Status != "ready" || len(run.IDs) < 1 || len(run.IDs) > distributionMaxFiles {
		return errors.New("완료된 원문 Markdown 내보내기를 선택하세요")
	}
	rows, e := tx.Query(ctx, `SELECT to_jsonb(d)-'search_vector' FROM documents d WHERE d.id=ANY($1::uuid[]) ORDER BY d.id FOR UPDATE`, run.IDs)
	if e != nil {
		return e
	}
	docs := map[string]map[string]any{}
	for rows.Next() {
		var raw []byte
		var d map[string]any
		if e = rows.Scan(&raw); e != nil {
			break
		}
		if e = json.Unmarshal(raw, &d); e != nil {
			break
		}
		docs[str(d, "id")] = d
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	if len(docs) != len(run.IDs) {
		return errors.New("선택 문서의 현재 상태를 확인할 수 없습니다")
	}
	rows, e = tx.Query(ctx, `SELECT id::text,document_id::text,name,size,checksum_sha256,content_type FROM attachments WHERE document_id=ANY($1::uuid[]) ORDER BY id FOR SHARE`, run.IDs)
	if e != nil {
		return e
	}
	type attachment struct {
		parent, name, hash, contentType string
		size                            int64
	}
	attachments := map[string]attachment{}
	for rows.Next() {
		var id string
		var a attachment
		if e = rows.Scan(&id, &a.parent, &a.name, &a.size, &a.hash, &a.contentType); e != nil {
			break
		}
		attachments[id] = a
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	if len(m.Files) != len(docs)+len(attachments) {
		return errors.New("원본의 문서·첨부 목록이 변경되었습니다")
	}
	var raw []byte
	cfg := defaultSettings()
	if e = tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&raw); e != nil {
		return e
	}
	if e = json.Unmarshal(raw, &cfg); e != nil {
		return e
	}
	var protectionRevision int
	if e = tx.QueryRow(ctx, `SELECT revision FROM protection_settings WHERE id=1 FOR SHARE`).Scan(&protectionRevision); e != nil {
		return e
	}
	if check && m.ProtectionRevision != protectionRevision {
		return errors.New("정보 보호 정책이 변경됐습니다. 현재 정책으로 새 배포를 준비하세요")
	}
	m.ProtectionRevision = protectionRevision
	var policyClock int64
	if e = tx.QueryRow(ctx, `SELECT revision FROM approval_policy_clock WHERE id=1 FOR SHARE`).Scan(&policyClock); e != nil {
		return e
	}
	for i := range m.Files {
		f := &m.Files[i]
		if f.Kind == "document" {
			d, ok := docs[f.SourceID]
			if !ok || str(d, "workspace_id") != run.WorkspaceID || d["deleted_at"] != nil || str(d, "status") != "published" || number(d, "version", 0) != f.SourceVersion || digest(str(d, "markdown")) != f.SHA256 || int64(len(str(d, "markdown"))) != f.Bytes {
				return errors.New("게시된 원문의 정확한 버전만 반출할 수 있습니다. 원문이 바뀌면 다시 준비하세요")
			}
			var allowed bool
			if e = tx.QueryRow(ctx, `SELECT madi_document_allowed($1,$2,false)`, p.ID, f.SourceID).Scan(&allowed); e != nil || !allowed {
				return errors.New("선택 문서의 현재 접근 권한이 변경되었습니다")
			}
			meta := map[string]any{"title": d["title"], "tags": d["tags"], "aliases": d["aliases"], "icon": d["icon"]}
			parent := str(d, "parent_id")
			if docs[parent] == nil {
				parent = ""
			}
			if f.MetadataHash != digest(string(jsonValue(meta))) || f.ParentSourceID != parent {
				return errors.New("원문의 속성·위치가 변경되었습니다")
			}
			approval := distributionApproval{Required: boolean(cfg, "approval_enabled")}
			if approval.Required {
				approval.ResourceHash = approvalSnapshotHash(approvalDocumentSnapshot(d))
				e = tx.QueryRow(ctx, `SELECT id::text FROM approval_requests WHERE resource_kind='document' AND resource_id=$1 AND status='approved' AND resource_version=$2 AND resource_hash=$3 AND policy_revision=$4 AND completed_at IS NOT NULL ORDER BY completed_at DESC LIMIT 1 FOR SHARE`, f.SourceID, f.SourceVersion-1, approval.ResourceHash, policyClock).Scan(&approval.RequestID)
				if e != nil {
					return errors.New("현재 승인 정책으로 승인된 정확한 게시 버전이 필요합니다. 검토 절차를 다시 완료하세요")
				}
			}
			if check && f.Approval != approval {
				return errors.New("원본의 승인 근거가 변경되었습니다")
			}
			f.Approval = approval
			if e = s.checkEvidenceProtection(ctx, tx, p, run.WorkspaceID, []any{d["title"], d["markdown"], d["tags"], d["aliases"]}); e != nil {
				return errors.New("현재 정보 보호 정책에서 원문을 반출할 수 없습니다")
			}
		} else {
			a, ok := attachments[f.SourceID]
			if !ok || a.parent != f.ParentSourceID || a.name != f.Metadata.Title || a.hash != f.SHA256 || a.size != f.Bytes {
				return errors.New("첨부의 소유 문서·이름·크기·SHA256이 변경됐거나 원본 해시가 없습니다")
			}
			if !check {
				body, ok := files[f.Path]
				if !ok {
					return errors.New("검사할 첨부 원본이 없습니다")
				}
				result, err := s.ProtectAttachmentTx(ctx, tx, p, a.parent, run.WorkspaceID, a.name, a.contentType, body)
				if err != nil || result.Changed {
					return errors.New("현재 정보 보호 정책에서 첨부를 원문 그대로 반출할 수 없습니다")
				}
			}
		}
	}
	var enabled bool
	var rev int64
	if e = tx.QueryRow(ctx, `SELECT enabled,revision FROM export_settings WHERE id FOR SHARE`).Scan(&enabled, &rev); e != nil || !enabled || rev != run.Revision {
		return errors.New("현재 내보내기 정책이 변경되었습니다")
	}
	var status, fp string
	if e = tx.QueryRow(ctx, `SELECT status,source_fingerprint FROM export_runs WHERE id=$1 FOR SHARE`, run.ID).Scan(&status, &fp); e != nil || status != "ready" || fp != run.Fingerprint {
		return errors.New("원본 내보내기가 취소되거나 변경되었습니다")
	}
	return distributionSessionTx(ctx, tx, run)
}

func distributionSourceArchive(ctx context.Context, data []byte, m *distributionManifest) (map[string][]byte, error) {
	invalid := errors.New("원문 내보내기 ZIP의 목록·파일 크기를 확인하세요")
	if len(data) > distributionMaxBytes+(2<<20) {
		return nil, invalid
	}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil || len(z.File) > distributionMaxFiles+1 {
		return nil, invalid
	}
	files := map[string][]byte{}
	var total int64
	for _, f := range z.File {
		_, exists := files[f.Name]
		if exists || !safeVaultPath(f.Name) || !f.Mode().IsRegular() || f.UncompressedSize64 > distributionMaxBytes {
			return nil, invalid
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		body, e := io.ReadAll(io.LimitReader(contextVaultReader{ctx, r}, distributionMaxBytes+1))
		r.Close()
		total += int64(len(body))
		if e != nil || total > distributionMaxBytes+distributionMaxManifest {
			return nil, invalid
		}
		files[f.Name] = body
	}
	var vault vaultManifest
	if json.Unmarshal(files["madi-manifest.json"], &vault) != nil || vault.Format != "madi-vault" || vault.Version != 1 {
		return nil, invalid
	}
	delete(files, "madi-manifest.json")
	m.Files = []distributionFile{}
	for _, d := range vault.Documents {
		if d == nil {
			return nil, invalid
		}
		body, ok := files[d.File]
		if !ok {
			return nil, invalid
		}
		meta := distributionMetadata{d.Title, append([]string{}, d.Tags...), append([]string{}, d.Aliases...), d.Icon}
		m.Files = append(m.Files, distributionFile{SourceID: d.ID, Path: d.File, Kind: "document", Bytes: int64(len(body)), SHA256: digest(string(body)), ParentSourceID: d.ParentID, SourceVersion: int(d.Version), Metadata: meta, MetadataHash: digest(string(jsonValue(meta.value())))})
	}
	for _, a := range vault.Attachments {
		if a == nil {
			return nil, invalid
		}
		body, ok := files[a.File]
		if !ok {
			return nil, invalid
		}
		meta := distributionMetadata{Title: a.Name, Tags: []string{}, Aliases: []string{}}
		m.Files = append(m.Files, distributionFile{SourceID: a.ID, Path: a.File, Kind: "attachment", Bytes: int64(len(body)), SHA256: digest(string(body)), ParentSourceID: a.DocumentID, Metadata: meta, MetadataHash: digest(string(jsonValue(meta.value())))})
	}
	if len(files) != len(m.Files) {
		return nil, invalid
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	if _, e = parseDistributionManifest(jsonValue(m)); e != nil {
		return nil, e
	}
	return files, nil
}
func buildDistributionZIP(ctx context.Context, m distributionManifest, signature string, files map[string][]byte) ([]byte, error) {
	var b bytes.Buffer
	out := zip.NewWriter(&b)
	for _, f := range m.Files {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		w, e := out.Create(f.Path)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(files[f.Path]); e != nil {
			return nil, e
		}
	}
	for _, f := range []struct {
		name string
		body []byte
	}{{distributionManifestPath, jsonValue(m)}, {distributionSignaturePath, []byte(signature)}} {
		w, e := out.Create(f.name)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(f.body); e != nil {
			return nil, e
		}
	}
	if e := out.Close(); e != nil {
		return nil, e
	}
	if b.Len() > distributionMaxBytes+distributionMaxManifest+(2<<20) {
		return nil, errors.New("최종 서명 ZIP 크기 한도를 초과했습니다")
	}
	return b.Bytes(), nil
}
func (s *Server) distributionSigningTx(ctx context.Context, tx pgx.Tx, v distributionExport) (distributionPolicy, string, error) {
	p, e := distributionPolicyTx(ctx, tx, false)
	if e != nil {
		return p, "", e
	}
	if !p.Enabled || p.Revision != v.PolicyRevision || !v.ExpiresAt.After(time.Now()) || v.ExpiresAt.Sub(v.CreatedAt) > time.Duration(p.MaxValidDays)*24*time.Hour {
		return p, "", errors.New("배포 정책·유효기간이 변경되었습니다")
	}
	var cipher string
	e = tx.QueryRow(ctx, `SELECT private_ciphertext FROM knowledge_distribution_keys WHERE id=$1 AND kind='signing' AND source_instance=$2 AND revoked_at IS NULL FOR SHARE`, v.KeyID, p.InstanceID).Scan(&cipher)
	if e != nil {
		return p, "", errors.New("현재 서명 키가 철회됐거나 없습니다")
	}
	return p, cipher, nil
}
func (s *Server) executeDistributionExport(ctx context.Context, j Job) (result map[string]any, err error) {
	v, e := s.readDistributionExport(ctx, str(j.Payload, "distribution_id"))
	if e != nil {
		return nil, jobPermanent("배포 작업이 없습니다")
	}
	if v.Status == "ready" {
		return map[string]any{"id": v.ID, "ready": true}, nil
	}
	if v.Status != "queued" {
		return nil, jobPermanent("배포 작업이 취소됐습니다")
	}
	defer func() {
		if err != nil {
			_, _ = s.DB.Exec(context.WithoutCancel(ctx), `UPDATE knowledge_distribution_exports SET status='failed' WHERE id=$1 AND status='queued'`, v.ID)
		}
	}()
	run, e := s.readExport(ctx, v.ExportID)
	if e != nil || run.OwnerID != v.OwnerID || run.WorkspaceID != v.WorkspaceID {
		return nil, jobPermanent("원본 내보내기가 없습니다")
	}
	p, _, e := s.validateExport(ctx, run)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	var sealed []byte
	e = s.DB.QueryRow(ctx, `SELECT b.ciphertext FROM export_artifact_blobs b JOIN export_runs e ON e.id=b.run_id WHERE b.run_id=$1 AND e.artifact_bytes<=$2`, run.ID, distributionMaxBytes+(2<<20)).Scan(&sealed)
	if e != nil {
		return nil, jobPermanent("원본 결과가 만료됐거나 배포 크기 한도를 초과합니다")
	}
	plain, e := s.decrypt(string(sealed))
	if e != nil {
		return nil, e
	}
	m := distributionManifest{Format: "madi-distribution-v1", BundleID: v.ID, SourceWorkspace: v.WorkspaceID, ReceiverInstance: v.Receiver, KeyID: v.KeyID, ProtectionRevision: 1, CreatedAt: v.CreatedAt.Unix(), ExpiresAt: v.ExpiresAt.Unix()}
	var policy distributionPolicy
	var cipher string
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	policy, cipher, e = s.distributionSigningTx(ctx, tx, v)
	tx.Rollback(ctx)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	m.SourceInstance = policy.InstanceID
	files, e := distributionSourceArchive(ctx, []byte(plain), &m)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	expected := map[string]int{}
	for _, d := range v.Documents {
		expected[d.ID] = d.Version
	}
	if len(expected) != len(run.IDs) {
		return nil, jobPermanent("명시적으로 확인한 원문 버전 목록이 없습니다")
	}
	for _, f := range m.Files {
		if f.Kind == "document" && expected[f.SourceID] != f.SourceVersion {
			return nil, jobPermanent("반출 확인 후 원문 버전이 변경됐습니다. 다시 비교하고 확인하세요")
		}
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	e = s.distributionSourcesTx(ctx, tx, p, run, &m, false, files)
	tx.Rollback(ctx)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	awaiting, e := s.prepareDistributionReview(ctx, p, run, v, &m)
	if e != nil {
		return nil, jobPermanent(e.Error())
	}
	if awaiting {
		return map[string]any{"id": v.ID, "status": "awaiting_review", "signed": false}, nil
	}
	secret, e := s.decrypt(cipher)
	if e != nil {
		return nil, e
	}
	private, e := base64.StdEncoding.Strict().DecodeString(secret)
	if e != nil || len(private) != ed25519.PrivateKeySize {
		return nil, errors.New("서명 키를 해독할 수 없습니다")
	}
	raw := jsonValue(m)
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw))
	data, e := buildDistributionZIP(ctx, m, signature, files)
	if e != nil {
		return nil, e
	}
	artifact, e := s.encrypt(string(data))
	if e != nil {
		return nil, e
	}
	manifestCipher, e := s.encrypt(string(raw))
	if e != nil {
		return nil, e
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	// Same per-owner quota lock as creation, before any source resource locks.
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,9342))`, v.OwnerID); e != nil {
		return nil, e
	}
	if e = s.distributionSourcesTx(ctx, tx, p, run, &m, true, nil); e != nil {
		return nil, jobPermanent(e.Error())
	}
	if _, _, e = s.distributionSigningTx(ctx, tx, v); e != nil {
		return nil, jobPermanent(e.Error())
	}
	if e = s.distributionBundleApprovedTx(ctx, tx, v, &m, true); e != nil {
		return nil, jobPermanent(e.Error())
	}
	var status string
	if e = tx.QueryRow(ctx, `SELECT status FROM knowledge_distribution_exports WHERE id=$1 FOR UPDATE`, v.ID).Scan(&status); e != nil {
		return nil, e
	}
	if status != "queued" {
		return nil, jobPermanent("배포 요청이 취소됐습니다")
	}
	var used int64
	if e = tx.QueryRow(ctx, `SELECT coalesce(sum(artifact_bytes),0) FROM knowledge_distribution_exports WHERE owner_id=$1 AND status='ready' AND expires_at>clock_timestamp()`, v.OwnerID).Scan(&used); e != nil {
		return nil, e
	}
	if used+int64(len(data)) > 200<<20 {
		return nil, jobPermanent("개인 배포 결과200MiB 한도입니다. 기존 결과를 폐기하세요")
	}
	if !v.ExpiresAt.After(time.Now()) || !run.Expires.After(time.Now()) {
		return nil, jobPermanent("반출 확인 전에 유효기간이 지났습니다")
	}
	_, e = tx.Exec(ctx, `INSERT INTO knowledge_distribution_artifacts(export_id,ciphertext) VALUES($1,$2)`, v.ID, []byte(artifact))
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE knowledge_distribution_exports SET status='ready',manifest_ciphertext=$2,signature=$3,manifest_hash=$4,artifact_bytes=$5 WHERE id=$1`, v.ID, manifestCipher, signature, digest(string(raw)), len(data))
	}
	if e == nil {
		e = distributionSessionTx(ctx, tx, run)
	}
	if e == nil && !v.ExpiresAt.After(time.Now()) {
		e = errors.New("반출 확정 전에 유효기간이 지났습니다")
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	return map[string]any{"id": v.ID, "files": len(m.Files), "bytes": len(data)}, e
}

func (s *Server) createDistributionExport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExportID  string                    `json:"export_id"`
		KeyID     string                    `json:"signing_key_id"`
		Receiver  string                    `json:"receiver_instance"`
		ValidDays int                       `json:"valid_days"`
		Consent   bool                      `json:"consent"`
		Documents []distributionDocumentRef `json:"documents"`
	}
	if decode(r, &in) != nil || !validID(in.ExportID) || !validID(in.KeyID) || !validID(in.Receiver) || in.ValidDays < 1 || in.ValidDays > 365 || !in.Consent || len(in.Documents) < 1 || len(in.Documents) > distributionMaxFiles {
		apiError(w, 400, "완료된 내보내기·서명 키·수신망 ID·유효기간과 반출 확인이 필요합니다")
		return
	}
	run, e := s.readExport(r.Context(), in.ExportID)
	cookie, ce := r.Cookie("madi_session")
	if e != nil || ce != nil || current(r).TokenID != "" || current(r).ScopeRestricted || current(r).PluginID != "" || run.OwnerID != current(r).ID || run.TokenBound || run.SessionHash != digest(cookie.Value) || run.Status != "ready" || run.Format != "markdown" {
		apiError(w, 404, "현재 로그인에서 준비한 원문 내보내기 결과가 없습니다")
		return
	}
	if _, _, e = s.validateExport(r.Context(), run); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	refs := map[string]int{}
	for _, d := range in.Documents {
		if !validID(d.ID) || d.Version < 1 || d.Version > 2147483647 || refs[d.ID] != 0 {
			apiError(w, 400, "중복 없는 문서 ID와 확인한 버전이 필요합니다")
			return
		}
		refs[d.ID] = d.Version
	}
	if len(refs) != len(run.IDs) {
		apiError(w, 400, "확인한 문서와 원문 내보내기 범위가 다릅니다")
		return
	}
	for _, id := range run.IDs {
		if refs[id] == 0 {
			apiError(w, 400, "확인한 문서와 원문 내보내기 범위가 다릅니다")
			return
		}
	}
	v := distributionExport{ID: newID(), WorkspaceID: run.WorkspaceID, OwnerID: run.OwnerID, ExportID: run.ID, KeyID: in.KeyID, Receiver: in.Receiver, CreatedAt: time.Now().UTC().Truncate(time.Second), Status: "queued", Documents: in.Documents}
	v.ExpiresAt = v.CreatedAt.Add(time.Duration(in.ValidDays) * 24 * time.Hour)
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtextextended($1,9342))`, v.OwnerID); e != nil {
		respond(w, nil, e)
		return
	}
	p, e := distributionPolicyTx(r.Context(), tx, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	v.PolicyRevision = p.Revision
	if _, _, e = s.distributionSigningTx(r.Context(), tx, v); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	if e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var count int
	var used int64
	if e = tx.QueryRow(r.Context(), `SELECT count(*) FILTER(WHERE status IN ('queued','awaiting_review')),coalesce(sum(artifact_bytes) FILTER(WHERE status='ready' AND expires_at>clock_timestamp()),0) FROM knowledge_distribution_exports WHERE owner_id=$1`, v.OwnerID).Scan(&count, &used); e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 5 || used >= 200<<20 {
		apiError(w, 409, "개인 대기5개·준비 결과200MiB 한도를 확인하세요")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_distribution_exports(id,workspace_id,owner_id,export_id,signing_key_id,receiver_instance,policy_revision,created_at,expires_at,document_refs) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, v.ID, v.WorkspaceID, v.OwnerID, v.ExportID, v.KeyID, v.Receiver, v.PolicyRevision, v.CreatedAt, v.ExpiresAt, jsonValue(v.Documents))
	if e == nil {
		v.JobID, e = s.EnqueueJob(r.Context(), tx, "knowledge.distribution", v.OwnerID, v.WorkspaceID, map[string]any{"distribution_id": v.ID})
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE knowledge_distribution_exports SET job_id=$2 WHERE id=$1`, v.ID, v.JobID)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET timeout_seconds=300,max_attempts=1 WHERE id=$1`, v.JobID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DISTRIBUTION_EXPORT_REQUEST", v.ID, map[string]any{"receiver_instance": v.Receiver, "key_id": v.KeyID})
	jsonResponse(w, 202, map[string]any{"id": v.ID, "job_id": v.JobID, "status": "queued"})
}
func (s *Server) listDistributionExports(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if _, e := s.packagePrincipal(r, wid); e != nil {
		apiError(w, 403, "현재 워크스페이스 조회 권한이 필요합니다")
		return
	}
	v, e := s.rows(r.Context(), `SELECT to_jsonb(d)-'manifest_ciphertext'-'signature'-'document_refs' || jsonb_build_object('job_status',(SELECT status FROM automation_jobs WHERE id=d.job_id)) FROM knowledge_distribution_exports d WHERE workspace_id=$1 AND owner_id=$2 ORDER BY created_at DESC LIMIT 100`, wid, current(r).ID)
	respond(w, v, e)
}
func (s *Server) revokeDistributionExport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || !validID(r.PathValue("id")) || in.Confirmation != "REVOKE" {
		apiError(w, 400, "배포 결과 폐기를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var wid string
	e = tx.QueryRow(r.Context(), `SELECT workspace_id::text FROM knowledge_distribution_exports WHERE id=$1 AND owner_id=$2`, r.PathValue("id"), current(r).ID).Scan(&wid)
	if e != nil {
		apiError(w, 404, "본인 배포 결과가 없습니다")
		return
	}
	if e = s.knowledgeActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE knowledge_distribution_exports SET status='revoked' WHERE id=$1`, r.PathValue("id"))
	if e == nil {
		_, e = tx.Exec(r.Context(), `DELETE FROM knowledge_distribution_artifacts WHERE export_id=$1`, r.PathValue("id"))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e == nil {
		s.audit(r, "DISTRIBUTION_EXPORT_REVOKE", r.PathValue("id"), map[string]any{"delivered_copies_recalled": false})
	}
	respond(w, map[string]any{"revoked": true, "notice": distributionNotice}, e)
}
func (s *Server) distributionDownloadCheck(r *http.Request, v distributionExport) (distributionManifest, error) {
	var m distributionManifest
	cookie, e := r.Cookie("madi_session")
	if e != nil || current(r).TokenID != "" || current(r).ScopeRestricted || current(r).PluginID != "" || v.OwnerID != current(r).ID || v.Status != "ready" {
		return m, errors.New("현재 로그인에서 다운로드할 결과가 없습니다")
	}
	run, e := s.readExport(r.Context(), v.ExportID)
	if e != nil || run.SessionHash != digest(cookie.Value) {
		return m, errors.New("원본 내보내기 세션이 변경되었습니다")
	}
	if _, _, e = s.validateExport(r.Context(), run); e != nil {
		return m, e
	}
	var sealed, signature string
	e = s.DB.QueryRow(r.Context(), `SELECT manifest_ciphertext,signature FROM knowledge_distribution_exports WHERE id=$1`, v.ID).Scan(&sealed, &signature)
	if e != nil {
		return m, e
	}
	raw, e := s.decrypt(sealed)
	if e != nil {
		return m, e
	}
	m, e = parseDistributionManifest([]byte(raw))
	if e != nil {
		return m, e
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		return m, e
	}
	defer tx.Rollback(r.Context())
	if e = s.distributionSourcesTx(r.Context(), tx, current(r), run, &m, true, nil); e != nil {
		return m, e
	}
	if _, _, e = s.distributionSigningTx(r.Context(), tx, v); e != nil {
		return m, e
	}
	if e = s.distributionBundleApprovedTx(r.Context(), tx, v, &m, true); e != nil {
		return m, e
	}
	var public, status string
	if e = tx.QueryRow(r.Context(), `SELECT public_key FROM knowledge_distribution_keys WHERE id=$1 FOR SHARE`, v.KeyID).Scan(&public); e != nil {
		return m, e
	}
	if e = verifyDistributionSignature([]byte(raw), signature, public); e != nil {
		return m, e
	}
	if e = tx.QueryRow(r.Context(), `SELECT status FROM knowledge_distribution_exports WHERE id=$1 FOR SHARE`, v.ID).Scan(&status); e != nil || status != "ready" {
		return m, errors.New("배포 결과가 폐기됐습니다")
	}
	if e = distributionSessionTx(r.Context(), tx, run); e != nil {
		return m, e
	}
	if !v.ExpiresAt.After(time.Now()) {
		return m, errors.New("배포 결과 유효기간이 지났습니다")
	}
	return m, nil
}
func (s *Server) downloadDistributionExport(w http.ResponseWriter, r *http.Request) {
	if !validID(r.PathValue("id")) {
		apiError(w, 404, "배포 결과가 없습니다")
		return
	}
	v, e := s.readDistributionExport(r.Context(), r.PathValue("id"))
	if e != nil {
		apiError(w, 404, "배포 결과가 없습니다")
		return
	}
	if _, e = s.distributionDownloadCheck(r, v); e != nil {
		apiError(w, 409, e.Error())
		return
	}
	var sealed []byte
	if e = s.DB.QueryRow(r.Context(), `SELECT ciphertext FROM knowledge_distribution_artifacts WHERE export_id=$1`, v.ID).Scan(&sealed); e != nil {
		apiError(w, 410, "배포 결과가 만료됐거나 폐기됐습니다")
		return
	}
	plain, e := s.decrypt(string(sealed))
	if e != nil {
		respond(w, nil, e)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="madi-knowledge-%s.zip"`, v.ID))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", fmt.Sprint(len(plain)))
	checked := time.Time{}
	for offset := 0; offset < len(plain); {
		if time.Since(checked) > time.Second {
			latest, err := s.readDistributionExport(r.Context(), v.ID)
			if err != nil {
				return
			}
			if _, err = s.distributionDownloadCheck(r, latest); err != nil {
				return
			}
			checked = time.Now()
		}
		end := min(offset+(64<<10), len(plain))
		if _, e = io.WriteString(w, plain[offset:end]); e != nil {
			return
		}
		offset = end
	}
	_, _ = s.DB.Exec(r.Context(), `UPDATE knowledge_distribution_exports SET downloads=downloads+1 WHERE id=$1`, v.ID)
	s.audit(r, "DISTRIBUTION_EXPORT_DOWNLOAD", v.ID, map[string]any{"bytes": len(plain), "receiver_instance": v.Receiver})
}
