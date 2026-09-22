package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed rag_index.sql
var ragIndexSchema string

type ragIndexGrant struct {
	ID, DocumentID, WorkspaceID, ActorID, TokenID string
	GenerationID                                  string
	Constraints                                   actorConstraints
	Revision                                      int64
	Active, Auto                                  bool
	Provider, Rerank                              string
	Version                                       int
	JobID                                         string
}

func (s *Server) migrateRAGIndex(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, ragIndexSchema)
	if e == nil {
		_, e = s.DB.Exec(ctx, ragGenerationSchema)
	}
	return e
}
func (s *Server) registerRAGIndex() {
	s.registerRAGGenerations()
	s.handle("GET /api/v1/documents/{id}/rag-index", s.getRAGIndex)
	s.handle("POST /api/v1/documents/{id}/rag-index", s.createRAGIndex)
	s.handle("DELETE /api/v1/documents/{id}/rag-index", s.revokeRAGIndex)
	s.handle("POST /api/v1/documents/{id}/rag-index/cancel", s.cancelRAGIndex)
	s.RegisterJobHandler("rag.index", s.runRAGIndex)
}

func ragRerankProvider(cfg map[string]any) ragProvider {
	return ragProvider{BaseURL: str(cfg, "rag_rerank_base_url"), Model: str(cfg, "rag_rerank_model"), APIKey: str(cfg, "rag_rerank_api_key"), CA: str(cfg, "rag_ca_pem"), AllowHTTP: boolean(cfg, "rag_allow_http")}
}
func ragRerankFingerprint(cfg map[string]any) string {
	return digest(string(jsonValue(ragRerankProvider(cfg))))
}

// Provider query strings may contain gateway credentials. Consent identifies
// the real endpoint/model with an opaque fingerprint, never by leaking secrets.
func ragDisplayURL(raw string) string {
	u, e := url.Parse(raw)
	if e != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	return u.String()
}

// Callers holding a transaction must not borrow a second pool connection.
func (s *Server) ragSettingsTx(ctx context.Context, tx pgx.Tx, wid string) (map[string]any, error) {
	var raw []byte
	if e := tx.QueryRow(ctx, "SELECT data FROM settings WHERE id=1 FOR SHARE").Scan(&raw); e != nil {
		return nil, e
	}
	cfg, e := s.decodeSettings(raw)
	if e != nil || wid == "" {
		return cfg, e
	}
	e = tx.QueryRow(ctx, "SELECT data FROM workspace_settings WHERE workspace_id=$1 FOR SHARE", wid).Scan(&raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return cfg, nil
	}
	if e != nil {
		return nil, e
	}
	var over map[string]any
	if e = json.Unmarshal(raw, &over); e != nil {
		return nil, e
	}
	for _, pair := range [][2]string{{"ai_base_url", "ai_api_key"}, {"rag_embedding_base_url", "rag_embedding_api_key"}, {"rag_rerank_base_url", "rag_rerank_api_key"}} {
		if _, ok := over[pair[0]]; ok && str(over, pair[0]) != str(cfg, pair[0]) {
			cfg[pair[1]] = ""
		}
	}
	for key, value := range over {
		if isWorkspaceSecret(key) {
			v, e := s.decrypt(str(over, key))
			if e != nil {
				return nil, e
			}
			cfg[key] = v
		} else {
			cfg[key] = value
		}
	}
	return cfg, nil
}

func ragGrantTx(ctx context.Context, q collaborationQuery, id string, lock bool) (ragIndexGrant, error) {
	return ragLoadGrant(ctx, q, `document_id=$1 AND generation_id IS NOT DISTINCT FROM (SELECT active_id FROM rag_generation_state WHERE workspace_id=g.workspace_id)`, []any{id}, lock)
}
func ragGrantByIDTx(ctx context.Context, q collaborationQuery, id string, lock bool) (ragIndexGrant, error) {
	return ragLoadGrant(ctx, q, `id=$1`, []any{id}, lock)
}
func ragGrantForGenerationTx(ctx context.Context, q collaborationQuery, id, generation string, lock bool) (ragIndexGrant, error) {
	return ragLoadGrant(ctx, q, `document_id=$1 AND generation_id IS NOT DISTINCT FROM NULLIF($2,'')::uuid`, []any{id, generation}, lock)
}
func ragLoadGrant(ctx context.Context, q collaborationQuery, predicate string, args []any, lock bool) (ragIndexGrant, error) {
	var g ragIndexGrant
	var raw []byte
	sql := `SELECT id::text,document_id::text,workspace_id::text,actor_id::text,COALESCE(token_id::text,''),actor_constraints,revision,active,auto_reindex,provider_fingerprint,rerank_fingerprint,expected_version,COALESCE(last_job_id::text,''),COALESCE(generation_id::text,'') FROM rag_index_grants g WHERE ` + predicate
	if lock {
		sql += " FOR UPDATE"
	}
	e := q.QueryRow(ctx, sql, args...).Scan(&g.ID, &g.DocumentID, &g.WorkspaceID, &g.ActorID, &g.TokenID, &raw, &g.Revision, &g.Active, &g.Auto, &g.Provider, &g.Rerank, &g.Version, &g.JobID, &g.GenerationID)
	if e == nil {
		e = json.Unmarshal(raw, &g.Constraints)
	}
	return g, e
}

// A short transactional backstop for races after the outer current-token and
// plugin checks. The saved grant never escalates an actor's current permissions.
func ragActorTx(ctx context.Context, tx pgx.Tx, p *Principal, id, wid string, write bool) error {
	if p == nil || !validID(id) || !hasIntegrationScope(p, "ai:execute") || (write && !hasIntegrationScope(p, "document:write")) || (!write && !hasIntegrationScope(p, "document:read")) {
		return errRAGChanged
	}
	var allowed bool
	e := tx.QueryRow(ctx, `SELECT NOT u.disabled AND madi_document_allowed(u.id,d.id,$3) AND ($4='' OR d.workspace_id::text=$4) FROM users u,documents d WHERE u.id=$1 AND d.id=$2 AND d.workspace_id=$5 AND d.deleted_at IS NULL FOR SHARE OF u`, p.ID, id, write, p.WorkspaceID, wid).Scan(&allowed)
	if e != nil || !allowed {
		return errRAGChanged
	}
	if p.TokenID != "" {
		var scopes []string
		e = tx.QueryRow(ctx, `SELECT scopes FROM api_keys WHERE id=$1 AND user_id=$2 AND workspace_id=$3 AND revoked_at IS NULL AND expires_at>now() FOR SHARE`, p.TokenID, p.ID, wid).Scan(&scopes)
		needed := "document:read"
		if write {
			needed = "document:write"
		}
		if e != nil || !slices.Contains(scopes, needed) || !slices.Contains(scopes, "ai:execute") {
			return errRAGChanged
		}
		var raw []byte
		if tx.QueryRow(ctx, `SELECT data FROM settings WHERE id=1 FOR SHARE`).Scan(&raw) != nil {
			return errRAGChanged
		}
		var settings map[string]any
		if json.Unmarshal(raw, &settings) != nil {
			return errRAGChanged
		}
		allowed := settingStrings(settings, "allowed_key_scopes", keyScopes)
		if !slices.Contains(allowed, needed) || !slices.Contains(allowed, "ai:execute") {
			return errRAGChanged
		}
	}
	if p.PluginID != "" {
		var raw, grants []byte
		e = tx.QueryRow(ctx, `SELECT x.manifest,w.capabilities FROM plugins x JOIN workspace_plugins w ON w.plugin_id=x.id WHERE x.id=$1 AND w.workspace_id=$2 AND x.enabled AND w.enabled FOR SHARE OF x,w`, p.PluginID, wid).Scan(&raw, &grants)
		var manifest pluginManifest
		var caps []string
		if e != nil || json.Unmarshal(raw, &manifest) != nil || json.Unmarshal(grants, &caps) != nil {
			return errRAGChanged
		}
		needed := "document:read"
		if write {
			needed = "document:write"
		}
		for _, scope := range []string{needed, "ai:execute"} {
			if !slices.Contains(caps, scope) || !slices.Contains(manifest.Capabilities, scope) {
				return errRAGChanged
			}
		}
	}
	return nil
}

func (s *Server) ragCurrentActor(ctx context.Context, g ragIndexGrant) (*Principal, error) {
	ctx = context.WithValue(ctx, jobContextKey{}, jobContext{ActorID: g.ActorID, TokenID: g.TokenID, Constraints: g.Constraints})
	p, e := s.workerPrincipal(ctx, g.ActorID, g.TokenID, g.WorkspaceID)
	if e != nil || !hasIntegrationScope(p, "document:write") || !hasIntegrationScope(p, "ai:execute") || !s.canDocument(ctx, p, g.DocumentID, true) {
		return nil, errRAGChanged
	}
	return p, nil
}

func ragIndexError(w http.ResponseWriter, e error) {
	var unavailable ragGenerationUnavailable
	if errors.As(e, &unavailable) {
		apiError(w, 409, unavailable.Error())
		return
	}
	if errors.Is(e, errRAGChanged) || errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 409, errRAGChanged.Error())
		return
	}
	respond(w, nil, e)
}

// Context-bound monitor cancels an in-flight request when its grant is revoked.
// Already transmitted bytes cannot be recalled; no later batch/output is used.
func ragWatch(ctx context.Context, check func(context.Context) error) (context.Context, func(), *error) {
	child, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	var result error
	go func() {
		defer close(done)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-child.Done():
				return
			case <-tick.C:
				if e := check(child); e != nil {
					result = e
					cancel()
					return
				}
			}
		}
	}()
	return child, func() { cancel(); <-done }, &result
}
