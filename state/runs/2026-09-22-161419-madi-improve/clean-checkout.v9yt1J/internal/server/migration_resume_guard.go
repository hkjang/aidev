package server

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

var errMigrationChanged = errors.New("원본·대상·현재 권한 또는 정책이 바뀌었습니다. 새 미리보기를 확인하세요")

// Lock order is the actor, membership, credential, settings and space ancestry.
// The caller has already locked all existing target rows in stable UUID order.
func (s *Server) migrationActorTx(ctx context.Context, tx pgx.Tx, p *Principal, v migrationSession, needsCSV bool) error {
	if p == nil || p.ID != v.UserID || !hasIntegrationScope(p, "document:write") || needsCSV && !hasIntegrationScope(p, "database:write") {
		return errMigrationChanged
	}
	var role string
	if tx.QueryRow(ctx, "SELECT role FROM users WHERE id=$1 AND NOT disabled FOR SHARE", p.ID).Scan(&role) != nil || role == "viewer" {
		return errMigrationChanged
	}
	if tx.QueryRow(ctx, "SELECT role FROM workspace_members WHERE user_id=$1 AND workspace_id=$2 FOR SHARE", p.ID, v.WorkspaceID).Scan(&role) != nil || !oneOf(role, "owner", "admin", "editor") {
		return errMigrationChanged
	}
	var sessionHash, ip, tokenHash string
	if e := tx.QueryRow(ctx, "SELECT request_session_hash,request_ip,request_token_hash FROM migration_sessions WHERE id=$1", v.ID).Scan(&sessionHash, &ip, &tokenHash); e != nil {
		return e
	}
	deadline := v.ExpiresAt
	boundDeadline := func(expires time.Time) {
		if deadline.IsZero() || expires.Before(deadline) {
			deadline = expires
		}
	}
	if sessionHash != "" {
		var expires time.Time
		if tx.QueryRow(ctx, "SELECT expires_at FROM sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>clock_timestamp() FOR SHARE", p.ID, sessionHash).Scan(&expires) != nil {
			return errMigrationChanged
		}
		boundDeadline(expires)
	}
	needed := []string{"document:write"}
	if needsCSV {
		needed = append(needed, "database:write")
	}
	if p.TokenID != "" {
		var scopes, ips []string
		var hash string
		var expires time.Time
		if tx.QueryRow(ctx, "SELECT scopes,ip_allowlist,token_hash,expires_at FROM api_keys WHERE id=$1 AND user_id=$2 AND workspace_id=$3 AND revoked_at IS NULL AND expires_at>clock_timestamp() FOR SHARE", p.TokenID, p.ID, v.WorkspaceID).Scan(&scopes, &ips, &hash, &expires) != nil || hash != tokenHash || !integrationIPAllowed(ip, ips) {
			return errMigrationChanged
		}
		boundDeadline(expires)
		var raw []byte
		var cfg map[string]any
		if tx.QueryRow(ctx, "SELECT data FROM settings WHERE id=1 FOR SHARE").Scan(&raw) != nil || json.Unmarshal(raw, &cfg) != nil {
			return errMigrationChanged
		}
		allowed := settingStrings(cfg, "allowed_key_scopes", keyScopes)
		for _, scope := range needed {
			if !slices.Contains(scopes, scope) || !slices.Contains(allowed, scope) {
				return errMigrationChanged
			}
		}
	}
	if p.PluginID != "" {
		var raw, grant []byte
		var manifest pluginManifest
		var caps []string
		if tx.QueryRow(ctx, `SELECT p.manifest,w.capabilities FROM plugins p JOIN workspace_plugins w ON w.plugin_id=p.id WHERE p.id=$1 AND w.workspace_id=$2 AND p.enabled AND w.enabled FOR SHARE OF p,w`, p.PluginID, v.WorkspaceID).Scan(&raw, &grant) != nil || json.Unmarshal(raw, &manifest) != nil || json.Unmarshal(grant, &caps) != nil {
			return errMigrationChanged
		}
		for _, scope := range needed {
			if !slices.Contains(manifest.Capabilities, scope) || !slices.Contains(caps, scope) {
				return errMigrationChanged
			}
		}
	}
	if v.SpaceID != "" {
		rows, e := tx.Query(ctx, `WITH RECURSIVE c AS(SELECT id,parent_id,ARRAY[id] seen FROM spaces WHERE id=$1 UNION ALL SELECT s.id,s.parent_id,c.seen||s.id FROM spaces s JOIN c ON c.parent_id=s.id WHERE cardinality(c.seen)<20 AND NOT s.id=ANY(c.seen)) SELECT s.id::text FROM spaces s WHERE s.id IN(SELECT id FROM c) ORDER BY s.id FOR SHARE`, v.SpaceID)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if rows.Scan(&id) != nil {
				rows.Close()
				return errMigrationChanged
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		rows, e = tx.Query(ctx, "SELECT space_id FROM space_members WHERE user_id=$1 AND space_id=ANY($2::uuid[]) ORDER BY space_id FOR SHARE", p.ID, ids)
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
		var allowed bool
		if tx.QueryRow(ctx, "SELECT madi_space_allowed($1,$2,true)", p.ID, v.SpaceID).Scan(&allowed) != nil || !allowed {
			return errMigrationChanged
		}
	}
	// PostgreSQL now() is fixed at transaction start, and a credential read can
	// precede later lock waits. Validate the actual expiry again after every lock.
	if !deadline.IsZero() && !deadline.After(time.Now()) {
		return errMigrationChanged
	}
	return nil
}

func migrationBindingGuardTx(ctx context.Context, tx pgx.Tx, p *Principal, v migrationSession, item migrationSessionItem) error {
	var id, hash, sourceHash, kind string
	var version, revision int64
	e := tx.QueryRow(ctx, `SELECT target_id::text,target_hash,source_hash,kind,target_version,revision FROM migration_source_bindings WHERE workspace_id=$1 AND user_id=$2 AND source_key=$3 AND source_id=$4 FOR UPDATE`, v.WorkspaceID, v.UserID, v.SourceKey, item.SourceID).Scan(&id, &hash, &sourceHash, &kind, &version, &revision)
	if item.BindingRevision == 0 {
		if e == pgx.ErrNoRows {
			return nil
		}
		return errMigrationChanged
	}
	if e != nil || revision != item.BindingRevision || kind != item.Kind {
		return errMigrationChanged
	}
	currentVersion, currentHash, allowed, e := migrationTargetFingerprint(ctx, tx, p, kind, id, v.WorkspaceID)
	if e != nil || !allowed || currentVersion != item.ExpectedVersion || currentHash != item.TargetHash || currentVersion != version || currentHash != hash {
		return errMigrationChanged
	}
	return nil
}
