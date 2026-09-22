package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
)

func (s *Server) agentRunPrincipal(ctx context.Context, r agentRun) (*Principal, error) {
	if r.TokenBound && r.TokenID == "" {
		return nil, errAgentChanged
	}
	ctx = context.WithValue(ctx, jobContextKey{}, jobContext{ActorID: r.OwnerID, TokenID: r.TokenID, Constraints: r.Constraints})
	p, e := s.workerPrincipal(ctx, r.OwnerID, r.TokenID, r.WorkspaceID)
	if e != nil || !hasIntegrationScope(p, "ai:execute") || !s.canFeature(ctx, p, r.WorkspaceID, "workspace-agents") {
		return nil, errAgentChanged
	}
	if r.SessionHash != "" {
		var active bool
		if s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now())`, r.SessionHash, r.OwnerID).Scan(&active) != nil || !active {
			return nil, errAgentChanged
		}
	}
	return p, nil
}
func (s *Server) agentGuard(ctx context.Context, id string) (agentRun, workspaceAgent, *Principal, map[string]any, error) {
	r, e := loadAgentRun(ctx, s.DB, id, false)
	if e != nil {
		return r, workspaceAgent{}, nil, nil, e
	}
	a, e := agentConfig(ctx, s.DB, r.AgentID, false)
	if e != nil || !a.Enabled || a.Revision != r.Revision || !oneOf(r.Status, "pending", "running", "awaiting_confirmation") {
		return r, a, nil, nil, errAgentChanged
	}
	p, e := s.agentRunPrincipal(ctx, r)
	if e != nil {
		return r, a, nil, nil, e
	}
	cfg, e := s.effectiveSettings(ctx, r.WorkspaceID)
	if e != nil || !boolean(cfg, "ai_enabled") || agentProviderFingerprint(cfg) != r.Provider {
		return r, a, p, cfg, errAgentChanged
	}
	if e = s.validateAgentSources(ctx, p, a, id, true); e != nil {
		return r, a, p, cfg, e
	}
	return r, a, p, cfg, nil
}
func (s *Server) agentActorTx(ctx context.Context, tx pgx.Tx, p *Principal, r agentRun, scopes ...string) error {
	if p == nil || r.TokenBound && r.TokenID == "" {
		return errAgentChanged
	}
	if !featureAllowed(ctx, tx, p, r.WorkspaceID, "workspace-agents") {
		return errAgentChanged
	}
	for _, scope := range append(scopes, "ai:execute") {
		if !hasIntegrationScope(p, scope) {
			return errAgentChanged
		}
	}
	var active bool
	if tx.QueryRow(ctx, `SELECT NOT u.disabled FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$2 WHERE u.id=$1 FOR SHARE OF u,m`, p.ID, r.WorkspaceID).Scan(&active) != nil || !active {
		return errAgentChanged
	}
	if r.SessionHash != "" {
		if tx.QueryRow(ctx, `SELECT true FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now() FOR SHARE`, r.SessionHash, p.ID).Scan(&active) != nil {
			return errAgentChanged
		}
	}
	needed := append(slices.Clone(scopes), "ai:execute")
	if r.TokenID != "" {
		var current []string
		if tx.QueryRow(ctx, `SELECT scopes FROM api_keys WHERE id=$1 AND user_id=$2 AND workspace_id=$3 AND revoked_at IS NULL AND expires_at>now() FOR SHARE`, r.TokenID, p.ID, r.WorkspaceID).Scan(&current) != nil {
			return errAgentChanged
		}
		var raw []byte
		if tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&raw) != nil {
			return errAgentChanged
		}
		var settings map[string]any
		if json.Unmarshal(raw, &settings) != nil {
			return errAgentChanged
		}
		allowed := settingStrings(settings, "allowed_key_scopes", keyScopes)
		for _, scope := range needed {
			if !slices.Contains(current, scope) || !slices.Contains(allowed, scope) {
				return errAgentChanged
			}
		}
	}
	if p.PluginID != "" {
		var manifestRaw, grantsRaw []byte
		if tx.QueryRow(ctx, `SELECT p.manifest,w.capabilities FROM plugins p JOIN workspace_plugins w ON w.plugin_id=p.id WHERE p.id=$1 AND w.workspace_id=$2 AND p.enabled AND w.enabled FOR SHARE OF p,w`, p.PluginID, r.WorkspaceID).Scan(&manifestRaw, &grantsRaw) != nil {
			return errAgentChanged
		}
		var manifest pluginManifest
		var grants []string
		if json.Unmarshal(manifestRaw, &manifest) != nil || json.Unmarshal(grantsRaw, &grants) != nil {
			return errAgentChanged
		}
		for _, scope := range needed {
			if !slices.Contains(grants, scope) || !slices.Contains(manifest.Capabilities, scope) {
				return errAgentChanged
			}
		}
	}
	return nil
}
func (s *Server) agentRunTx(ctx context.Context, tx pgx.Tx, r agentRun, p *Principal) (workspaceAgent, error) {
	a, e := agentConfig(ctx, tx, r.AgentID, true)
	if e != nil || !a.Enabled || a.Revision != r.Revision {
		return a, errAgentChanged
	}
	current, e := loadAgentRun(ctx, tx, r.ID, true)
	if e != nil || current.Revision != r.Revision || current.JobID != r.JobID || !oneOf(current.Status, "pending", "running", "awaiting_confirmation") {
		return a, errAgentChanged
	}
	if e = s.agentActorTx(ctx, tx, p, r); e != nil {
		return a, e
	}
	cfg, e := s.ragSettingsTx(ctx, tx, r.WorkspaceID)
	if e != nil || !boolean(cfg, "ai_enabled") || agentProviderFingerprint(cfg) != r.Provider {
		return a, errAgentChanged
	}
	return a, nil
}

// Same-transaction final source backstop. Query/formula dependencies are pinned
// by actual input rows and schemas, not by trusting a model-provided checksum.
func (s *Server) agentSourceTx(ctx context.Context, tx pgx.Tx, p *Principal, a workspaceAgent, runID, ownUpdatedDoc string, oldVersion int) error {
	rows, e := tx.Query(ctx, `SELECT kind,resource_id::text,snapshot,fingerprint FROM agent_sources WHERE run_id=$1 ORDER BY source_key`, runID)
	if e != nil {
		return e
	}
	sources := []agentSource{}
	for rows.Next() {
		var src agentSource
		var raw []byte
		if e = rows.Scan(&src.Kind, &src.ResourceID, &raw, &src.Fingerprint); e != nil {
			break
		}
		if e = json.Unmarshal(raw, &src.Snapshot); e != nil {
			break
		}
		sources = append(sources, src)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return e
	}
	for _, src := range sources {
		if src.Kind == "document" {
			if !agentDocumentAllowed(ctx, tx, p, a, src.ResourceID, false) {
				return errAgentChanged
			}
			var version int
			if tx.QueryRow(ctx, `SELECT version FROM documents WHERE id=$1 AND deleted_at IS NULL FOR SHARE`, src.ResourceID).Scan(&version) != nil {
				return errAgentChanged
			}
			expected := src.Fingerprint
			if src.ResourceID == ownUpdatedDoc && fmt.Sprint(oldVersion) == expected {
				expected = fmt.Sprint(oldVersion + 1)
			}
			if fmt.Sprint(version) != expected {
				return errAgentChanged
			}
			continue
		}
		if !hasIntegrationScope(p, "database:read") {
			return errAgentChanged
		}
		var deps []map[string]any
		if json.Unmarshal(jsonValue(src.Snapshot["dependencies"]), &deps) != nil || len(deps) == 0 {
			return errAgentChanged
		}
		for _, dep := range deps {
			var raw []byte
			if str(dep, "kind") == "database" {
				if !slices.Contains(a.DatabaseIDs, str(dep, "id")) {
					return errAgentChanged
				}
				var ok bool
				if tx.QueryRow(ctx, `SELECT madi_space_allowed($1,d.space_id,false) AND EXISTS(SELECT 1 FROM workspace_members m WHERE m.user_id=$1 AND m.workspace_id=d.workspace_id) FROM databases d WHERE id=$2 AND workspace_id=$3 FOR SHARE OF d`, p.ID, str(dep, "id"), a.WorkspaceID).Scan(&ok) != nil || !ok {
					return errAgentChanged
				}
				e = tx.QueryRow(ctx, `SELECT jsonb_build_object('workspace_id',workspace_id::text,'space_id',coalesce(space_id::text,''),'properties',properties) FROM databases WHERE id=$1`, str(dep, "id")).Scan(&raw)
			} else {
				e = tx.QueryRow(ctx, `SELECT to_jsonb(v) FROM database_rows v WHERE id=$1 AND database_id=$2 FOR SHARE`, str(dep, "id"), str(dep, "database_id")).Scan(&raw)
			}
			if e != nil {
				return errAgentChanged
			}
			var value any
			if json.Unmarshal(raw, &value) != nil || digest(string(jsonValue(value))) != str(dep, "fingerprint") {
				return errAgentChanged
			}
		}
	}
	return nil
}
func agentPublicError(e error) string {
	if errors.Is(e, errAgentChanged) {
		return e.Error()
	}
	var permanent permanentJobError
	if errors.As(e, &permanent) {
		return permanent.Error()
	}
	return "Agent 실행을 완료하지 못했습니다. 공급자 설정과 실행 이력을 확인하세요"
}
