package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed graph_ai.sql
var graphAISchema string
var graphAIKinds = []string{"relation", "duplicate", "topic", "entity", "gap"}
var errGraphAIChanged = errors.New("AI 그래프의 현재 문서·권한·공급자 동의·기능 정책 또는 실행 상태가 변경되었습니다. 현재 자료로 새 분석을 시작하세요")

type graphAIQuery interface {
	collaborationQuery
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type graphAIRun struct {
	ID, WorkspaceID, OwnerID, TokenID, SessionHash, Provider, Status, Error string
	TokenBound                                                              bool
	Constraints                                                             actorConstraints
	Kinds                                                                   []string
	CreatedAt, HeartbeatAt, ExpiresAt                                       time.Time
}
type graphAISource struct {
	DocumentID   string `json:"document_id"`
	Version      int    `json:"version"`
	DocumentHash string `json:"document_hash"`
	Start        int    `json:"start_byte"`
	End          int    `json:"end_byte"`
	ContentHash  string `json:"content_hash"`
}
type graphAIEvidence struct {
	DocumentID string    `json:"document_id"`
	Quote      string    `json:"quote,omitempty"`
	Citation   *aiSource `json:"citation,omitempty"`
}
type graphAICandidate struct {
	Kind         string            `json:"kind"`
	SourceID     string            `json:"source_id,omitempty"`
	TargetID     string            `json:"target_id,omitempty"`
	RelationType string            `json:"relation_type,omitempty"`
	Title        string            `json:"title,omitempty"`
	Description  string            `json:"description,omitempty"`
	EntityType   string            `json:"entity_type,omitempty"`
	Topic        string            `json:"topic,omitempty"`
	Reason       string            `json:"reason"`
	Evidence     []graphAIEvidence `json:"evidence"`
}
type graphAIAction struct {
	ID      string           `json:"id"`
	RunID   string           `json:"run_id"`
	Kind    string           `json:"kind"`
	Payload graphAICandidate `json:"payload"`
	Hash    string           `json:"action_hash"`
	Status  string           `json:"status"`
	Result  map[string]any   `json:"result"`
}

func (s *Server) migrateGraphAI(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, graphAISchema)
	return e
}
func loadGraphAIRun(ctx context.Context, q collaborationQuery, id string, lock bool) (graphAIRun, error) {
	var v graphAIRun
	var raw []byte
	query := `SELECT id::text,workspace_id::text,owner_id::text,coalesce(token_id::text,''),token_bound,actor_constraints,session_hash,provider_fingerprint,kinds,status,error,created_at,heartbeat_at,expires_at FROM graph_ai_runs WHERE id=$1`
	if lock {
		query += " FOR UPDATE"
	}
	e := q.QueryRow(ctx, query, id).Scan(&v.ID, &v.WorkspaceID, &v.OwnerID, &v.TokenID, &v.TokenBound, &raw, &v.SessionHash, &v.Provider, &v.Kinds, &v.Status, &v.Error, &v.CreatedAt, &v.HeartbeatAt, &v.ExpiresAt)
	if e == nil {
		e = json.Unmarshal(raw, &v.Constraints)
	}
	return v, e
}
func graphAICanonicalHash(title, markdown string) string {
	return digest(string(jsonValue(map[string]any{"title": title, "markdown": markdown})))
}
func graphAICookie(r *http.Request) string {
	if current(r).TokenID != "" {
		return ""
	}
	if c, e := r.Cookie("madi_session"); e == nil {
		return digest(c.Value)
	}
	return ""
}
func (s *Server) graphAIPrincipal(ctx context.Context, run graphAIRun) (*Principal, error) {
	if run.TokenBound && run.TokenID == "" {
		return nil, errGraphAIChanged
	}
	ctx = context.WithValue(ctx, jobContextKey{}, jobContext{ActorID: run.OwnerID, TokenID: run.TokenID, Constraints: run.Constraints})
	p, e := s.workerPrincipal(ctx, run.OwnerID, run.TokenID, run.WorkspaceID)
	if e != nil || !hasIntegrationScope(p, "ai:execute") || !hasIntegrationScope(p, "document:read") || !s.canFeature(ctx, p, run.WorkspaceID, "ai-graph") {
		return nil, errGraphAIChanged
	}
	if run.SessionHash != "" {
		var active bool
		if s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now())`, run.SessionHash, run.OwnerID).Scan(&active) != nil || !active {
			return nil, errGraphAIChanged
		}
	}
	return p, nil
}
func graphAISources(ctx context.Context, q graphAIQuery, id string) ([]graphAISource, error) {
	rows, e := q.Query(ctx, `SELECT document_id::text,document_version,document_hash,start_byte,end_byte,content_hash FROM graph_ai_sources WHERE run_id=$1 ORDER BY document_id`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	result := []graphAISource{}
	for rows.Next() {
		var v graphAISource
		if e = rows.Scan(&v.DocumentID, &v.Version, &v.DocumentHash, &v.Start, &v.End, &v.ContentHash); e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (s *Server) graphAIValidateSources(ctx context.Context, q graphAIQuery, p *Principal, run graphAIRun, writeID string) ([]aiSource, error) {
	sources, e := graphAISources(ctx, q, run.ID)
	if e != nil || len(sources) < 1 || len(sources) > 12 {
		return nil, errGraphAIChanged
	}
	result := []aiSource{}
	for _, src := range sources {
		var title, markdown string
		var version int
		query := `SELECT d.title,d.markdown,d.version FROM documents d WHERE d.id=$1 AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,$4)`
		if _, ok := q.(pgx.Tx); ok {
			query += " FOR SHARE OF d"
		}
		e = q.QueryRow(ctx, query, src.DocumentID, run.WorkspaceID, p.ID, src.DocumentID == writeID).Scan(&title, &markdown, &version)
		if e != nil || version != src.Version || graphAICanonicalHash(title, markdown) != src.DocumentHash || src.Start < 0 || src.End <= src.Start || src.End > len(markdown) || digest(markdown[src.Start:src.End]) != src.ContentHash {
			return nil, errGraphAIChanged
		}
		if tx, ok := q.(pgx.Tx); ok {
			if e = ragActorTx(ctx, tx, p, src.DocumentID, run.WorkspaceID, src.DocumentID == writeID); e != nil {
				return nil, errGraphAIChanged
			}
		}
		chunk := ragChunk{Start: src.Start, End: src.End, Content: markdown[src.Start:src.End], Hash: src.ContentHash, StartLine: 1, EndLine: 1}
		for _, c := range markdown[:src.Start] {
			if c == '\n' {
				chunk.StartLine++
			}
		}
		chunk.EndLine = chunk.StartLine
		for _, c := range chunk.Content {
			if c == '\n' {
				chunk.EndLine++
			}
		}
		result = append(result, sourceFromChunk(src.DocumentID, title, version, chunk))
	}
	return result, nil
}
func (s *Server) graphAIValidateTx(ctx context.Context, tx pgx.Tx, run graphAIRun, p *Principal, writeID string) error {
	if !featureAllowed(ctx, tx, p, run.WorkspaceID, "ai-graph") || (run.TokenBound && run.TokenID == "") {
		return errGraphAIChanged
	}
	if run.SessionHash != "" {
		var active bool
		if tx.QueryRow(ctx, `SELECT true FROM sessions WHERE token_hash=$1 AND user_id=$2 AND expires_at>now() FOR SHARE`, run.SessionHash, p.ID).Scan(&active) != nil {
			return errGraphAIChanged
		}
	}
	cfg, e := s.ragSettingsTx(ctx, tx, run.WorkspaceID)
	if e != nil || !boolean(cfg, "ai_enabled") || aiHistoryProvider(cfg) != run.Provider {
		return errGraphAIChanged
	}
	_, e = s.graphAIValidateSources(ctx, tx, p, run, writeID)
	return e
}
