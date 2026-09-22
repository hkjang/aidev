package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_structured.sql
var knowledgeStructuredSchema string

func (s *Server) migrateKnowledgeStructured(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgeStructuredSchema)
	return e
}
func (s *Server) registerKnowledgeStructured() {
	s.handle("GET /api/v1/documents/{id}/structured-context", s.getStructuredContext)
	s.handle("POST /api/v1/documents/{id}/structured-draft", s.generateStructuredDraft)
	s.handle("POST /api/v1/knowledge/structured-drafts", s.saveStructuredDraft)
	s.handle("GET /api/v1/knowledge/structured-drafts", s.listStructuredDrafts)
	s.handle("GET /api/v1/knowledge/structured-drafts/{id}", s.getStructuredDraft)
	s.handle("DELETE /api/v1/knowledge/structured-drafts/{id}", s.deleteStructuredDraft)
	s.handle("POST /api/v1/knowledge/structured-drafts/{id}/preview", s.previewStructuredRow)
	s.handle("POST /api/v1/knowledge/structured-drafts/{id}/apply", s.applyStructuredRow)
}

type structuredScope struct {
	DocumentID, DatabaseID, WorkspaceID, Markdown, Title, DatabaseName, SpaceID, Schema, Destination string
	Version                                                                                          int
	Properties                                                                                       []map[string]any
}
type structuredTicket struct {
	Kind        string            `json:"kind"`
	ID          string            `json:"id"`
	OwnerID     string            `json:"owner_id"`
	DocumentID  string            `json:"document_id"`
	DatabaseID  string            `json:"database_id"`
	WorkspaceID string            `json:"workspace_id"`
	Version     int               `json:"version"`
	SourceHash  string            `json:"source_hash"`
	Start       int               `json:"start_byte"`
	End         int               `json:"end_byte"`
	Schema      string            `json:"schema_hash"`
	Destination string            `json:"destination_hash"`
	Provider    string            `json:"provider_fingerprint"`
	Model       string            `json:"model"`
	Payload     structuredPayload `json:"payload"`
	Expires     int64             `json:"expires"`
}

func (s *Server) structuredTx(r *http.Request, writing bool) (pgx.Tx, error) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(r.Context(), `SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='8s'`)
	// These operations never update documents or spaces. Fence inherited ACL
	// changes only during the short save/preview/row-commit transaction, never AI.
	if e == nil && writing {
		_, e = tx.Exec(r.Context(), `LOCK TABLE documents,spaces,space_members,document_shares IN SHARE MODE`)
	}
	if e != nil {
		tx.Rollback(r.Context())
		return nil, e
	}
	return tx, nil
}
func (s *Server) structuredScopeTx(r *http.Request, tx pgx.Tx, docID, dbID string, write bool) (structuredScope, error) {
	v := structuredScope{DocumentID: docID, DatabaseID: dbID}
	changed := approvalProblem(404, "현재 접근할 수 있는 원문과 같은 워크스페이스의 데이터베이스를 선택하세요")
	if !validID(docID) || !validID(dbID) || current(r) == nil {
		return v, changed
	}
	e := tx.QueryRow(r.Context(), `SELECT workspace_id::text,version,title,markdown FROM documents WHERE id=$1 AND deleted_at IS NULL AND octet_length(markdown)<=1048576 AND madi_document_allowed($2,id,false) FOR SHARE`, docID, current(r).ID).Scan(&v.WorkspaceID, &v.Version, &v.Title, &v.Markdown)
	if e != nil {
		return v, changed
	}
	var raw []byte
	e = tx.QueryRow(r.Context(), `SELECT name,coalesce(space_id::text,''),properties FROM databases WHERE id=$1 AND workspace_id=$2 AND (space_id IS NULL OR madi_space_allowed($3,space_id,$4)) FOR SHARE`, dbID, v.WorkspaceID, current(r).ID, write).Scan(&v.DatabaseName, &v.SpaceID, &raw)
	if e != nil || len(raw) > 256<<10 || json.Unmarshal(raw, &v.Properties) != nil {
		return v, changed
	}
	v.Schema = digest(string(raw))
	v.Destination, e = accessPreviewFingerprint(r, tx, docID, v.WorkspaceID, documentAccessPlacement{Space: v.SpaceID, Visibility: "workspace"})
	if e != nil {
		return v, e
	}
	v.Destination = digest(v.Destination + ":" + dbID + ":" + v.SpaceID + ":" + v.Schema)
	scopes := []string{"document:read", "database:read"}
	if write {
		scopes = append(scopes, "database:write")
	}
	if e = s.knowledgeActorTx(r, tx, v.WorkspaceID, scopes...); e != nil {
		return v, approvalProblem(403, e.Error())
	}
	return v, nil
}
func (s *Server) structuredProviderTx(r *http.Request, tx pgx.Tx, v structuredScope) (map[string]any, error) {
	if e := s.knowledgeActorTx(r, tx, v.WorkspaceID, "document:read", "database:read", "database:write", "ai:execute"); e != nil {
		return nil, approvalProblem(403, e.Error())
	}
	cfg, e := s.ragSettingsTx(r.Context(), tx, v.WorkspaceID)
	if e != nil {
		return nil, e
	}
	_, endpointErr := aiEndpoint(str(cfg, "ai_base_url"))
	return map[string]any{"configured": boolean(cfg, "ai_enabled") && endpointErr == nil && str(cfg, "ai_model") != "", "base_url": ragDisplayURL(str(cfg, "ai_base_url")), "model": str(cfg, "ai_model"), "fingerprint": aiHistoryProvider(cfg), "max_tokens": number(cfg, "ai_max_tokens", 4096)}, nil
}
func (s *Server) getStructuredContext(w http.ResponseWriter, r *http.Request) {
	if !personalAIHistory(current(r)) {
		apiError(w, 403, "개인 로그인 화면에서 원문과 전송 대상을 확인하세요")
		return
	}
	tx, e := s.structuredTx(r, false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, e := s.structuredScopeTx(r, tx, r.PathValue("id"), r.URL.Query().Get("database_id"), true)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	provider, e := s.structuredProviderTx(r, tx, v)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	props := []map[string]any{}
	for _, p := range v.Properties {
		if structuredPropertySupported(p) {
			clean, e := structuredProperties(v.Properties, []string{str(p, "id")})
			if e == nil {
				props = append(props, clean[0])
			}
		}
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), v.WorkspaceID, map[string]any{"title": v.Title, "database": v.DatabaseName, "markdown": v.Markdown, "properties": props}); e != nil {
		approvalRespondError(w, e)
		return
	}
	if e = s.revalidateStructuredReadTx(r, tx, v, str(provider, "fingerprint")); e != nil {
		approvalRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"document_id": v.DocumentID, "database_id": v.DatabaseID, "workspace_id": v.WorkspaceID, "version": v.Version, "title": v.Title, "markdown": v.Markdown, "database_name": v.DatabaseName, "space_id": v.SpaceID, "schema_hash": v.Schema, "destination_hash": v.Destination, "properties": props, "provider": provider, "max_selection_bytes": structuredSelectionMax, "automatic_apply": false, "destination_notice": "새 행은 대상 데이터베이스를 현재 읽을 수 있는 사용자에게 보입니다. 원문의 비공개 권한은 복사된 값에 상속되지 않습니다."})
}

// A policy or provider row lock may have waited after an earlier authorization
// check. Re-read provider first, then the current source/destination and actor,
// before exposing a context or sending another AI event. Missing workspace
// overrides can be inserted while a transaction holds only global settings.
func (s *Server) revalidateStructuredReadTx(r *http.Request, tx pgx.Tx, baseline structuredScope, fingerprint string) error {
	provider, e := s.structuredProviderTx(r, tx, baseline)
	if e != nil {
		return e
	}
	fresh, e := s.structuredScopeTx(r, tx, baseline.DocumentID, baseline.DatabaseID, true)
	if e != nil {
		return e
	}
	if str(provider, "fingerprint") != fingerprint || fresh.Version != baseline.Version || fresh.Markdown != baseline.Markdown || fresh.Title != baseline.Title || fresh.DatabaseName != baseline.DatabaseName || fresh.Schema != baseline.Schema || fresh.Destination != baseline.Destination || fresh.SpaceID != baseline.SpaceID {
		return errStructuredChanged
	}
	return nil
}

func structuredTicketValid(t structuredTicket, p *Principal) bool {
	return p != nil && t.Kind == "madi-structured-draft-v1" && t.OwnerID == p.ID && validID(t.ID) && validID(t.DocumentID) && validID(t.DatabaseID) && validID(t.WorkspaceID) && t.Version > 0 && t.Version < 2147483647 && t.Start >= 0 && t.End > t.Start && t.End-t.Start <= structuredSelectionMax && len(t.SourceHash) == 64 && len(t.Schema) == 64 && len(t.Destination) == 64 && len(t.Provider) == 64 && t.Expires > time.Now().Unix() && len(t.Payload.Fields) > 0 && len(t.Payload.Fields) <= 32
}
func structuredMatches(t structuredTicket, v structuredScope) bool {
	return t.DocumentID == v.DocumentID && t.DatabaseID == v.DatabaseID && t.WorkspaceID == v.WorkspaceID && t.Version == v.Version && t.SourceHash == digest(v.Markdown) && t.End <= len(v.Markdown) && t.Schema == v.Schema && t.Destination == v.Destination
}

func structuredCurrentACLTx(r *http.Request, tx pgx.Tx, v structuredScope) error {
	var allowed bool
	e := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM documents d JOIN databases b ON b.id=$2 AND b.workspace_id=d.workspace_id WHERE d.id=$1 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) AND (b.space_id IS NULL OR madi_space_allowed($3,b.space_id,false)))`, v.DocumentID, v.DatabaseID, current(r).ID).Scan(&allowed)
	if e != nil || !allowed {
		return approvalProblem(404, "현재 원문과 데이터베이스 접근 범위를 다시 확인하세요")
	}
	return nil
}

var errStructuredChanged = approvalProblem(409, "원문·속성·공유 대상·현재 권한이 변경됐습니다. 현재 상태에서 다시 비교하세요")
