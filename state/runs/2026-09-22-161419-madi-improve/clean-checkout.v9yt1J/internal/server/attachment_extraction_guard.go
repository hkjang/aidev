package server

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

var errExtractionChanged = errors.New("첨부 원본·현재 권한·세션 또는 추출 정책이 바뀌었습니다. 새로 확인하세요")

func (s *Server) extractionActorTx(ctx context.Context, tx pgx.Tx, p *Principal, v extractionRun) error {
	if p == nil || p.ID != v.ActorID || p.PluginID != "" || !hasIntegrationScope(p, "document:read") || !hasIntegrationScope(p, "document:write") || p.WorkspaceID != "" && p.WorkspaceID != v.WorkspaceID {
		return errExtractionChanged
	}
	var role, member string
	var expires time.Time
	if tx.QueryRow(ctx, "SELECT role FROM users WHERE id=$1 AND NOT disabled FOR SHARE", p.ID).Scan(&role) != nil || role == "viewer" {
		return errExtractionChanged
	}
	if tx.QueryRow(ctx, "SELECT role FROM workspace_members WHERE user_id=$1 AND workspace_id=$2 FOR SHARE", p.ID, v.WorkspaceID).Scan(&member) != nil || !oneOf(member, "owner", "admin", "editor") {
		return errExtractionChanged
	}
	if p.TokenID == "" {
		if v.SessionHash == "" || tx.QueryRow(ctx, "SELECT expires_at FROM sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>clock_timestamp() FOR SHARE", p.ID, v.SessionHash).Scan(&expires) != nil {
			return errExtractionChanged
		}
	} else {
		var hash string
		var scopes, ips []string
		if tx.QueryRow(ctx, `SELECT token_hash,scopes,ip_allowlist,expires_at FROM api_keys WHERE id=$1 AND user_id=$2 AND workspace_id=$3 AND revoked_at IS NULL AND expires_at>clock_timestamp() FOR SHARE`, p.TokenID, p.ID, v.WorkspaceID).Scan(&hash, &scopes, &ips, &expires) != nil || hash != v.TokenHash || !integrationIPAllowed(v.RequestIP, ips) {
			return errExtractionChanged
		}
		var raw []byte
		var config map[string]any
		if tx.QueryRow(ctx, "SELECT data FROM settings WHERE id=1 FOR SHARE").Scan(&raw) != nil || json.Unmarshal(raw, &config) != nil {
			return errExtractionChanged
		}
		allowed := settingStrings(config, "allowed_key_scopes", keyScopes)
		for _, scope := range []string{"document:read", "document:write"} {
			if !slices.Contains(scopes, scope) || !slices.Contains(allowed, scope) {
				return errExtractionChanged
			}
		}
	}
	if !expires.After(time.Now()) {
		return errExtractionChanged
	}
	return nil
}

// The final projection commit holds these short ACL fences; native tools and
// object-store reads never execute while database locks are held.
func (s *Server) extractionGuardTx(ctx context.Context, tx pgx.Tx, p *Principal, v extractionRun) error {
	if _, e := tx.Exec(ctx, `SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s'; LOCK TABLE spaces,space_members,document_shares IN SHARE MODE`); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,22))", "documents/"+v.WorkspaceID); e != nil {
		return e
	}
	rows, e := tx.Query(ctx, `WITH RECURSIVE ancestry AS(SELECT id,parent_id,ARRAY[id] seen FROM documents WHERE id=$1 UNION ALL SELECT d.id,d.parent_id,a.seen||d.id FROM documents d JOIN ancestry a ON d.id=a.parent_id WHERE cardinality(a.seen)<20 AND NOT d.id=ANY(a.seen)) SELECT id FROM documents WHERE id IN(SELECT id FROM ancestry) ORDER BY id FOR SHARE`, v.DocumentID)
	if e != nil {
		return e
	}
	for rows.Next() {
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	var version int
	var object storedObject
	var documentID, wid string
	if e := tx.QueryRow(ctx, `SELECT d.id::text,d.workspace_id::text,d.version,a.path,coalesce(a.storage_provider_id::text,''),a.object_key,a.checksum_sha256,a.size FROM attachments a JOIN documents d ON d.id=a.document_id WHERE a.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,true) FOR SHARE OF d,a`, v.AttachmentID, p.ID).Scan(&documentID, &wid, &version, &object.Path, &object.ProviderID, &object.Key, &object.Checksum, &object.Size); e != nil {
		return errExtractionChanged
	}
	if documentID != v.DocumentID || wid != v.WorkspaceID || version != v.DocumentVersion || object != v.Object || object.Checksum != v.Checksum {
		return errExtractionChanged
	}
	policy, e := extractionPolicyQuery(ctx, tx, true)
	if e != nil || !boolean(policy.Data, "enabled") || policy.Revision != v.PolicyRevision || len(v.OCRPages) > 0 && !boolean(policy.Data, "ocr_enabled") {
		return errExtractionChanged
	}
	return s.extractionActorTx(ctx, tx, p, v)
}
