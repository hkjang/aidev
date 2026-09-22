package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed workspace_agent.sql
var workspaceAgentSchema string

var errAgentChanged = errors.New("Agent 실행 중 설정·원본·권한이 변경되었거나 실행이 취소되었습니다. 현재 권한으로 새 실행을 시작하세요")
var agentToolNames = []string{"search_documents", "get_document", "query_database", "get_graph", "create_document", "update_document"}

type workspaceAgent struct {
	ID           string   `json:"id"`
	WorkspaceID  string   `json:"workspace_id"`
	Name         string   `json:"name"`
	Instructions string   `json:"instructions"`
	Enabled      bool     `json:"enabled"`
	Revision     int64    `json:"revision"`
	SpaceIDs     []string `json:"space_ids"`
	DocumentIDs  []string `json:"document_ids"`
	DatabaseIDs  []string `json:"database_ids"`
	Tools        []string `json:"tools"`
	MaxSteps     int      `json:"max_steps"`
	MaxTokens    int      `json:"max_tokens"`
}
type agentRun struct {
	SessionHash                                                                 string
	ID, AgentID, WorkspaceID, OwnerID, TokenID, Provider, Status, JobID, Prompt string
	TokenBound                                                                  bool
	Constraints                                                                 actorConstraints
	Revision                                                                    int64
	Step, OutputBytes                                                           int
	Messages                                                                    []agentMessage
}
type agentToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
type agentMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []agentToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}
type agentSource struct {
	Key, Kind, ResourceID, Fingerprint string
	Snapshot                           map[string]any
}

func (s *Server) migrateWorkspaceAgent(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, workspaceAgentSchema)
	return e
}

func agentConfig(ctx context.Context, q collaborationQuery, id string, lock bool) (workspaceAgent, error) {
	var a workspaceAgent
	query := `SELECT id::text,workspace_id::text,name,instructions,enabled,revision,space_ids::text[],document_ids::text[],database_ids::text[],tools,max_steps,max_tokens FROM workspace_agents WHERE id=$1`
	if lock {
		query += " FOR SHARE"
	}
	e := q.QueryRow(ctx, query, id).Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.Instructions, &a.Enabled, &a.Revision, &a.SpaceIDs, &a.DocumentIDs, &a.DatabaseIDs, &a.Tools, &a.MaxSteps, &a.MaxTokens)
	return a, e
}
func loadAgentRun(ctx context.Context, q collaborationQuery, id string, lock bool) (agentRun, error) {
	var r agentRun
	var constraints, messages []byte
	query := `SELECT id::text,agent_id::text,workspace_id::text,owner_id::text,COALESCE(token_id::text,''),token_bound,actor_constraints,agent_revision,provider_fingerprint,status,COALESCE(job_id::text,''),prompt,step,output_bytes,messages,session_hash FROM agent_runs WHERE id=$1`
	if lock {
		query += " FOR UPDATE"
	}
	e := q.QueryRow(ctx, query, id).Scan(&r.ID, &r.AgentID, &r.WorkspaceID, &r.OwnerID, &r.TokenID, &r.TokenBound, &constraints, &r.Revision, &r.Provider, &r.Status, &r.JobID, &r.Prompt, &r.Step, &r.OutputBytes, &messages, &r.SessionHash)
	if e == nil {
		e = json.Unmarshal(constraints, &r.Constraints)
	}
	if e == nil {
		e = json.Unmarshal(messages, &r.Messages)
	}
	return r, e
}
func agentProviderFingerprint(cfg map[string]any) string {
	values := map[string]any{}
	for _, k := range []string{"ai_enabled", "ai_base_url", "ai_model", "ai_api_key", "ai_max_tokens", "ai_system_prompt"} {
		values[k] = cfg[k]
	}
	return digest(string(jsonValue(values)))
}
func agentStrictJSON(raw []byte, target any) error {
	if len(raw) > 65536 {
		return errors.New("도구 입력은 64KB 이하여야 합니다")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(target); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("단일 JSON 객체가 필요합니다")
	}
	return nil
}
func agentValidateConfig(a workspaceAgent) error {
	if strings.TrimSpace(a.Name) == "" || len(a.Name) > 160 || len(a.Instructions) > 16000 || a.MaxSteps < 1 || a.MaxSteps > 24 || a.MaxTokens < 1 || a.MaxTokens > 262144 {
		return errors.New("이름·지시문·최대 단계(1~24)·토큰(1~262144)을 확인하세요")
	}
	for _, ids := range [][]string{a.SpaceIDs, a.DocumentIDs, a.DatabaseIDs} {
		if len(ids) > 100 {
			return errors.New("지식 범위는 종류별 100개까지 선택하세요")
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if !validID(id) || seen[id] {
				return errors.New("지식 범위 ID가 올바르지 않거나 중복입니다")
			}
			seen[id] = true
		}
	}
	seen := map[string]bool{}
	for _, tool := range a.Tools {
		if !slices.Contains(agentToolNames, tool) || seen[tool] {
			return errors.New("허용 도구를 확인하세요")
		}
		seen[tool] = true
	}
	if a.Enabled && (len(a.Tools) == 0 || len(a.SpaceIDs)+len(a.DocumentIDs)+len(a.DatabaseIDs) == 0) {
		return errors.New("지식 범위와 도구를 설정한 뒤 활성화하세요")
	}
	if a.Enabled && slices.Contains(a.Tools, "query_database") && len(a.DatabaseIDs) == 0 {
		return errors.New("DB 조회 도구에는 데이터베이스를 지정해야 합니다")
	}
	return nil
}

// Space selection includes descendants, but explicit document selection never
// implicitly shares children. Current hierarchical ACL is always intersected.
const agentDocScopeSQL = `(d.id=ANY($3::uuid[]) OR d.space_id IN (WITH RECURSIVE scope AS (SELECT id FROM spaces WHERE id=ANY($4::uuid[]) AND workspace_id=$2 UNION SELECT x.id FROM spaces x JOIN scope p ON x.parent_id=p.id WHERE x.workspace_id=$2) SELECT id FROM scope))`

func agentDocumentAllowed(ctx context.Context, q collaborationQuery, p *Principal, a workspaceAgent, id string, write bool) bool {
	if p == nil || !validID(id) || (!hasIntegrationScope(p, "document:read") && !write) || (write && !hasIntegrationScope(p, "document:write")) {
		return false
	}
	var ok bool
	e := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents d WHERE d.id=$5 AND d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,$6) AND `+agentDocScopeSQL+`)`, p.ID, a.WorkspaceID, a.DocumentIDs, a.SpaceIDs, id, write).Scan(&ok)
	return e == nil && ok
}
func agentEventTx(ctx context.Context, tx pgx.Tx, id, kind string, data any) error {
	raw := jsonValue(data)
	if len(raw) > 65536 {
		return errors.New("Agent 이벤트가 64KB 한도를 초과했습니다")
	}
	var total int
	e := tx.QueryRow(ctx, `UPDATE agent_runs SET output_bytes=output_bytes+$2,updated_at=now() WHERE id=$1 AND output_bytes+$2<=4194304 RETURNING output_bytes`, id, len(raw)).Scan(&total)
	if e != nil {
		return errors.New("Agent 실행 출력은 최대 4MB입니다")
	}
	_, e = tx.Exec(ctx, `INSERT INTO agent_events(run_id,event,data) VALUES($1,$2,$3)`, id, kind, raw)
	return e
}
