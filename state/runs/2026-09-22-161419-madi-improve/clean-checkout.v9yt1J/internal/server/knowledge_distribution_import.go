package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) trustedDistributionTx(ctx context.Context, tx pgx.Tx, raw []byte, signature string) (distributionManifest, string, error) {
	m, e := parseDistributionManifest(raw)
	if e != nil {
		return m, "", e
	}
	p, e := distributionPolicyTx(ctx, tx, false)
	if e != nil {
		return m, "", e
	}
	if !p.Enabled || p.InstanceID != m.ReceiverInstance {
		return m, "", errors.New("현재 반입 정책이 비활성이거나 수신망 식별자가 다릅니다")
	}
	var keyID, public string
	e = tx.QueryRow(ctx, `SELECT id::text,public_key FROM knowledge_distribution_keys WHERE kind='trusted' AND source_instance=$1 AND source_key_id=$2 AND revoked_at IS NULL FOR SHARE`, m.SourceInstance, m.KeyID).Scan(&keyID, &public)
	if e != nil {
		return m, "", errors.New("현재 신뢰 목록에 유효한 반출망 공개키가 없습니다")
	}
	if e = verifyDistributionSignature(raw, signature, public); e != nil {
		return m, "", e
	}
	if e = validateReceivedDistributionApproval(m); e != nil {
		return m, "", e
	}
	if e = distributionCurrentTime(m, time.Now(), p.MaxValidDays); e != nil {
		return m, "", e
	}
	return m, keyID, nil
}
func (s *Server) bindSignedMigration(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Manifest  string `json:"manifest_base64"`
		Signature string `json:"signature"`
		Consent   bool   `json:"consent"`
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted || current(r).PluginID != "" {
		apiError(w, 403, "서명 반입 준비는 개인 로그인으로 명시 확인하세요")
		return
	}
	if decode(r, &in) != nil || !validID(r.PathValue("id")) || !in.Consent || len(in.Manifest) > ((distributionMaxManifest+2)/3)*4 || len(in.Signature) > 128 {
		apiError(w, 400, "서명 매니페스트와 비공개 이관 확인이 필요합니다")
		return
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(in.Manifest)
	if e != nil {
		apiError(w, 400, "매니페스트 base64 형식을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, e := scanMigrationSession(tx.QueryRow(r.Context(), `SELECT `+migrationSessionSelect+` FROM migration_sessions WHERE id=$1 FOR UPDATE`, r.PathValue("id")))
	if e != nil || !s.migrationSessionAllowed(r.Context(), current(r), v) || v.Status != "uploading" || !v.ExpiresAt.After(time.Now()) {
		apiError(w, 404, "현재 준비 가능한 개인 이관 세션이 없습니다")
		return
	}
	// Existing binding is immutable, including after uploads have started.
	var oldID, oldHash, oldSignature string
	e = tx.QueryRow(r.Context(), `SELECT id::text,manifest_hash,signature FROM knowledge_distribution_imports WHERE session_id=$1`, v.ID).Scan(&oldID, &oldHash, &oldSignature)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		respond(w, nil, e)
		return
	}
	if oldID != "" && (oldHash != digest(string(raw)) || oldSignature != in.Signature) {
		apiError(w, 409, "이관 세션에 이미 다른 서명 원본이 결합됐습니다")
		return
	}
	if oldID == "" && v.ItemCount != 0 {
		apiError(w, 409, "파일을 올리기 전의 빈 세션에서 서명 원본을 먼저 확인하세요")
		return
	}
	m, keyID, e := s.trustedDistributionTx(r.Context(), tx, raw, in.Signature)
	if e != nil {
		apiError(w, 422, e.Error())
		return
	}
	if v.SourceKey != m.sourceKey() || v.Format != "markdown" {
		apiError(w, 409, "서명 원본 식별자와 Markdown 이관 형식이 일치해야 합니다")
		return
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), v.WorkspaceID, m); e != nil {
		apiError(w, 422, "현재 정보 보호 정책에서 서명 메타데이터를 준비할 수 없습니다")
		return
	}
	if e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read", "document:write"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if !v.ExpiresAt.After(time.Now()) || m.ExpiresAt <= time.Now().Unix() {
		apiError(w, 409, "반입 준비 유효기간이 지났습니다")
		return
	}
	id := oldID
	if id == "" {
		id = newID()
		sealed, err := s.encrypt(string(raw))
		if err != nil {
			respond(w, nil, err)
			return
		}
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_distribution_imports(id,session_id,workspace_id,owner_id,trusted_key_id,bundle_id,source_instance,manifest_hash,manifest_ciphertext,signature,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,to_timestamp($11))`, id, v.ID, v.WorkspaceID, v.UserID, keyID, m.BundleID, m.SourceInstance, digest(string(raw)), sealed, in.Signature, m.ExpiresAt)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DISTRIBUTION_IMPORT_VERIFY", id, map[string]any{"session_id": v.ID, "manifest_hash": digest(string(raw)), "trusted_key_id": keyID})
	jsonResponse(w, 200, map[string]any{"id": id, "session_id": v.ID, "signature_valid": true, "files_verified": false, "manifest": m, "notice": distributionNotice + " 실제 파일 검증·현재 신뢰 재검사는 이관 준비와 최종 반영 단계에서 수행됩니다."})
}

// Both hooks run inside the existing migration publication transaction, after
// the migration session/target resource locks. An absent receipt means ordinary
// import; source_key alone never asserts verified origin.
func (s *Server) validateSignedMigrationTx(ctx context.Context, tx pgx.Tx, v migrationSession) error {
	var cipher, signature, keyID string
	e := tx.QueryRow(ctx, `SELECT manifest_ciphertext,signature,trusted_key_id::text FROM knowledge_distribution_imports WHERE session_id=$1 AND owner_id=$2 AND workspace_id=$3`, v.ID, v.UserID, v.WorkspaceID).Scan(&cipher, &signature, &keyID)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	raw, e := s.decrypt(cipher)
	if e != nil {
		return e
	}
	m, currentKey, e := s.trustedDistributionTx(ctx, tx, []byte(raw), signature)
	if e != nil {
		return e
	}
	if currentKey != keyID || v.SourceKey != m.sourceKey() {
		return errors.New("반입 신뢰 키 또는 원본 결합이 변경되었습니다")
	}
	want := map[string]distributionFile{}
	for _, f := range m.Files {
		want[f.SourceID] = f
	}
	rows, e := tx.Query(ctx, `SELECT source_id,source_hash,file_path,kind,source_bytes,parent_source_id,metadata->>'source_metadata_hash' FROM migration_session_items WHERE session_id=$1 ORDER BY source_id`, v.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, hash, name, kind, parent string
		var metadata *string
		var size int64
		if e = rows.Scan(&id, &hash, &name, &kind, &size, &parent, &metadata); e != nil {
			return e
		}
		if kind == "folder" && strings.HasPrefix(id, "@madi-folder:") {
			continue
		}
		f, ok := want[id]
		if !ok || f.SHA256 != hash || f.Path != name || f.Kind != kind || f.Bytes != size || f.ParentSourceID != parent || metadata == nil || *metadata != f.MetadataHash {
			return errors.New("이관 항목이 서명된 원본·메타데이터와 다릅니다. 서명 원본부터 다시 준비하세요")
		}
		count++
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if count != len(m.Files) {
		return errors.New("서명에 선언된 모든 파일을 준비한 뒤 반영하세요. 제외는 명시적인 건너뛰기로 기록하세요")
	}
	return nil
}
func (s *Server) finishSignedMigrationTx(ctx context.Context, tx pgx.Tx, v migrationSession) error {
	if e := s.validateSignedMigrationTx(ctx, tx, v); e != nil {
		return e
	}
	var id, cipher string
	e := tx.QueryRow(ctx, `SELECT id::text,manifest_ciphertext FROM knowledge_distribution_imports WHERE session_id=$1 FOR UPDATE`, v.ID).Scan(&id, &cipher)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	raw, e := s.decrypt(cipher)
	if e != nil {
		return e
	}
	m, e := parseDistributionManifest([]byte(raw))
	if e != nil {
		return e
	}
	versions := map[string]int{}
	for _, f := range m.Files {
		versions[f.SourceID] = f.SourceVersion
	}
	rows, e := tx.Query(ctx, `SELECT i.source_id,i.kind,i.target_id::text,coalesce(b.target_version,0),coalesce(b.target_hash,''),coalesce(i.metadata->>'decision','apply'),i.disposition FROM migration_session_items i LEFT JOIN migration_source_bindings b ON b.workspace_id=$2 AND b.user_id=$3 AND b.source_key=$4 AND b.source_id=i.source_id WHERE i.session_id=$1 AND i.kind<>'folder' ORDER BY i.source_id`, v.ID, v.WorkspaceID, v.UserID, v.SourceKey)
	if e != nil {
		return e
	}
	mapping := []map[string]any{}
	for rows.Next() {
		var source, kind, target, hash, decision, disposition string
		var version int64
		if e = rows.Scan(&source, &kind, &target, &version, &hash, &decision, &disposition); e != nil {
			break
		}
		item := map[string]any{"source_id": source, "source_version": versions[source], "kind": kind, "decision": decision, "disposition": disposition}
		if decision != "skip" {
			item["target_id"] = target
			item["target_version"] = version
			item["target_hash"] = hash
		}
		mapping = append(mapping, item)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	// One source file can become independent attachments on several imported
	// documents. Preserve every claimed copy, not just the primary inbox object.
	refs, e := tx.Query(ctx, `SELECT i.source_id,o.attachment_id::text,o.document_id::text,a.checksum_sha256,md5(jsonb_build_array(a.name,a.content_type,a.size,a.checksum_sha256,a.object_key,a.storage_provider_id)::text) FROM migration_session_objects o JOIN migration_session_items i ON i.id=o.item_id JOIN attachments a ON a.id=o.attachment_id AND a.document_id=o.document_id WHERE i.session_id=$1 AND o.status='claimed' ORDER BY i.source_id,o.document_id LIMIT 100001`, v.ID)
	if e != nil {
		return e
	}
	count := 0
	for refs.Next() {
		var source, target, document, checksum, hash string
		if e = refs.Scan(&source, &target, &document, &checksum, &hash); e != nil {
			break
		}
		count++
		if count > 100000 {
			e = errors.New("반입 첨부 사본 대응표 한도100,000개를 초과했습니다")
			break
		}
		mapping = append(mapping, map[string]any{"source_id": source, "source_version": versions[source], "kind": "attachment", "decision": "apply", "disposition": "reference_copy", "target_id": target, "target_document_id": document, "target_version": 1, "target_hash": hash, "target_content_sha256": checksum})
	}
	if e == nil {
		e = refs.Err()
	}
	refs.Close()
	if e != nil {
		return e
	}
	if m.ExpiresAt <= time.Now().Unix() {
		return errors.New("반입 확정 전에 서명 유효기간이 지났습니다")
	}
	_, e = tx.Exec(ctx, `UPDATE knowledge_distribution_imports SET imported_at=clock_timestamp(),mapping=$2 WHERE id=$1 AND imported_at IS NULL`, id, jsonValue(mapping))
	return e
}
func (s *Server) getSignedMigration(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var id, hash string
	var expires time.Time
	var imported *time.Time
	var raw []byte
	e = tx.QueryRow(r.Context(), `SELECT id::text,manifest_hash,expires_at,imported_at,mapping FROM knowledge_distribution_imports WHERE session_id=$1 AND owner_id=$2`, v.ID, current(r).ID).Scan(&id, &hash, &expires, &imported, &raw)
	if e != nil {
		apiError(w, 404, "이관 세션에 서명 원본 기록이 없습니다")
		return
	}
	var mappings []map[string]any
	if e = json.Unmarshal(raw, &mappings); e != nil {
		respond(w, nil, e)
		return
	}
	// IDs/versions of formerly imported targets are not an ACL bypass. Return
	// only mappings whose current target remains accessible to this actor.
	visible := []map[string]any{}
	for _, m := range mappings {
		target := str(m, "target_id")
		if target == "" {
			visible = append(visible, m)
			continue
		}
		_, _, allowed, err := migrationTargetFingerprint(r.Context(), tx, current(r), str(m, "kind"), target, v.WorkspaceID)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			respond(w, nil, err)
			return
		}
		if allowed {
			visible = append(visible, m)
		}
	}
	if e = s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read", "document:write"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	respond(w, map[string]any{"id": id, "manifest_hash": hash, "expires_at": expires, "imported_at": imported, "mapping": visible, "notice": distributionNotice + " 이력은 당시 반입 사실이며 현재 신뢰 키 유효성을 인증하지 않습니다."}, nil)
}
