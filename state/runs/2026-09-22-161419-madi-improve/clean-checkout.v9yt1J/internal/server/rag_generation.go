package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

//go:embed rag_generation.sql
var ragGenerationSchema string

type ragGeneration struct {
	ID, WorkspaceID, Name, Status, Provider, CreatedBy, IndexJobID string
	Revision                                                       int64
	Dimensions                                                     int
	Config                                                         map[string]any
}

var ragGenerationProviderKeys = []string{"rag_embedding_base_url", "rag_embedding_model", "rag_embedding_api_key", "rag_embedding_dimensions", "rag_ca_pem", "rag_allow_http", "rag_rerank_enabled", "rag_rerank_base_url", "rag_rerank_model", "rag_rerank_api_key"}

func ragGenerationStoredConfig(cfg map[string]any) map[string]any {
	value := map[string]any{}
	for _, key := range ragSettingKeys() {
		if v, ok := cfg[key]; ok {
			value[key] = v
		}
	}
	return value
}

func (s *Server) ragGenerationTx(ctx context.Context, q collaborationQuery, id string, lock bool) (ragGeneration, error) {
	var g ragGeneration
	var cipher string
	sql := `SELECT id::text,workspace_id::text,name,status,provider_fingerprint,created_by::text,coalesce(index_job_id::text,''),revision,dimensions,config_cipher FROM rag_generations WHERE id=$1`
	if lock {
		sql += " FOR UPDATE"
	}
	e := q.QueryRow(ctx, sql, id).Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.Status, &g.Provider, &g.CreatedBy, &g.IndexJobID, &g.Revision, &g.Dimensions, &cipher)
	if e != nil {
		return g, e
	}
	plain, e := s.decrypt(cipher)
	if e != nil {
		return g, e
	}
	e = json.Unmarshal([]byte(plain), &g.Config)
	return g, e
}

func ragSelectedGeneration(ctx context.Context, q collaborationQuery, wid, requested string) (string, error) {
	if requested != "" {
		if !validID(requested) {
			return "", errRAGChanged
		}
		return requested, nil
	}
	var id string
	e := q.QueryRow(ctx, `SELECT coalesce(active_id::text,'') FROM rag_generation_state WHERE workspace_id=$1`, wid).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", nil
	}
	return id, e
}

// Provider credentials are immutable within a generation. Shadow builds use
// their explicitly consented snapshot; the active generation additionally must
// agree with effective workspace settings, so an ordinary settings change can
// never silently send content under an old generation's consent.
func (s *Server) ragGenerationConfig(ctx context.Context, q collaborationQuery, cfg map[string]any, wid, id string) (map[string]any, error) {
	if id == "" {
		return cfg, nil
	}
	g, e := s.ragGenerationTx(ctx, q, id, false)
	if e != nil || g.WorkspaceID != wid || g.Status == "disabled" {
		return nil, errRAGChanged
	}
	active, e := ragSelectedGeneration(ctx, q, wid, "")
	if e != nil {
		return nil, e
	}
	if active == id && (ragProviderFingerprint(cfg) != g.Provider || ragRerankFingerprint(cfg) != ragRerankFingerprint(g.Config) || boolean(cfg, "rag_rerank_enabled") != boolean(g.Config, "rag_rerank_enabled")) {
		return nil, errRAGChanged
	}
	enabled := boolean(cfg, "rag_enabled")
	out := map[string]any{}
	for key, value := range cfg {
		out[key] = value
	}
	for _, key := range ragGenerationProviderKeys {
		if value, ok := g.Config[key]; ok {
			out[key] = value
		}
	}
	out["rag_enabled"] = enabled && boolean(g.Config, "rag_enabled")
	out["rag_generation_dimensions"] = g.Dimensions
	if ragProviderFingerprint(out) != g.Provider {
		return nil, errRAGChanged
	}
	return out, nil
}
func (s *Server) ragGrantSettings(ctx context.Context, g ragIndexGrant) (map[string]any, error) {
	cfg, e := s.effectiveSettings(ctx, g.WorkspaceID)
	if e != nil {
		return nil, e
	}
	return s.ragGenerationConfig(ctx, s.DB, cfg, g.WorkspaceID, g.GenerationID)
}
func (s *Server) ragGrantSettingsTx(ctx context.Context, tx pgx.Tx, g ragIndexGrant) (map[string]any, error) {
	cfg, e := s.ragSettingsTx(ctx, tx, g.WorkspaceID)
	if e != nil {
		return nil, e
	}
	return s.ragGenerationConfig(ctx, tx, cfg, g.WorkspaceID, g.GenerationID)
}

func ragGenerationOperatorTx(ctx context.Context, tx pgx.Tx, r *http.Request, wid string) error {
	p := current(r)
	cookie, e := r.Cookie("madi_session")
	if e != nil || p == nil || p.TokenID != "" || p.ScopeRestricted {
		return errRAGChanged
	}
	var ok bool
	e = tx.QueryRow(ctx, `SELECT NOT u.disabled AND u.kind='user' AND u.role<>'viewer' AND m.role IN ('owner','admin') AND ss.expires_at>now() FROM users u JOIN workspace_members m ON m.user_id=u.id JOIN sessions ss ON ss.user_id=u.id AND ss.token_hash=$3 WHERE u.id=$1 AND m.workspace_id=$2 FOR SHARE OF u,m,ss`, p.ID, wid, digest(cookie.Value)).Scan(&ok)
	if e != nil || !ok {
		return errRAGChanged
	}
	return nil
}
