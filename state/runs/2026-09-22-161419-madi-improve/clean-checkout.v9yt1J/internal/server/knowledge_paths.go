package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_paths.sql
var knowledgePathsSchema string

type knowledgePath struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	SpaceID     string    `json:"space_id"`
	OwnerID     string    `json:"owner_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	RoleLabels  []string  `json:"role_labels"`
	Visibility  string    `json:"visibility"`
	Archived    bool      `json:"archived"`
	Revision    int64     `json:"revision"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type knowledgePathStep struct {
	ID          string `json:"id"`
	DocumentID  string `json:"document_id"`
	Ordinal     int    `json:"ordinal"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Instruction string `json:"instruction"`
}
type knowledgePathInput struct {
	WorkspaceID   string              `json:"workspace_id"`
	SpaceID       string              `json:"space_id"`
	Title         string              `json:"title"`
	Description   string              `json:"description"`
	RoleLabels    []string            `json:"role_labels"`
	Visibility    string              `json:"visibility"`
	Archived      bool                `json:"archived"`
	Revision      int64               `json:"revision"`
	ConfirmShared bool                `json:"confirm_shared"`
	Steps         []knowledgePathStep `json:"steps"`
}

const knowledgePathSelect = `p.id::text,p.workspace_id::text,coalesce(p.space_id::text,''),p.owner_id::text,p.title,p.description,p.role_labels,p.visibility,p.archived,p.revision,p.updated_at`

func (s *Server) migrateKnowledgePaths(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgePathsSchema)
	return e
}
func scanKnowledgePath(row pgx.Row) (knowledgePath, error) {
	var p knowledgePath
	e := row.Scan(&p.ID, &p.WorkspaceID, &p.SpaceID, &p.OwnerID, &p.Title, &p.Description, &p.RoleLabels, &p.Visibility, &p.Archived, &p.Revision, &p.UpdatedAt)
	if p.RoleLabels == nil {
		p.RoleLabels = []string{}
	}
	return p, e
}
func validKnowledgePathInput(in *knowledgePathInput) bool {
	if in.RoleLabels == nil {
		in.RoleLabels = []string{}
	}
	in.Title = strings.TrimSpace(in.Title)
	if !validID(in.WorkspaceID) || (in.SpaceID != "" && !validID(in.SpaceID)) || in.Title == "" || len(in.Title) > 250 || len(in.Description) > 4000 || !utf8.ValidString(in.Title+in.Description) || strings.ContainsRune(in.Title+in.Description, 0) || !oneOf(in.Visibility, "private", "workspace") || len(in.Steps) < 1 || len(in.Steps) > 100 || len(in.RoleLabels) > 20 {
		return false
	}
	seen := map[string]bool{}
	for _, label := range in.RoleLabels {
		if strings.TrimSpace(label) == "" || len(label) > 100 || !utf8.ValidString(label) || strings.ContainsRune(label, 0) || seen[label] {
			return false
		}
		seen[label] = true
	}
	seen = map[string]bool{}
	for i := range in.Steps {
		step := &in.Steps[i]
		step.Ordinal = i
		step.Title = strings.TrimSpace(step.Title)
		if !validID(step.DocumentID) || !oneOf(step.Kind, "read", "practice", "review") || step.Title == "" || len(step.Title) > 250 || len(step.Instruction) > 4000 || !utf8.ValidString(step.Title+step.Instruction) || strings.ContainsRune(step.Title+step.Instruction, 0) || (step.ID != "" && (!validID(step.ID) || seen[step.ID])) {
			return false
		}
		if step.ID != "" {
			seen[step.ID] = true
		}
	}
	return true
}
func learningCookie(r *http.Request) bool {
	p := current(r)
	return p != nil && p.Kind == "user" && p.TokenID == "" && !p.ScopeRestricted && p.PluginID == ""
}
func knowledgePathReadTx(ctx context.Context, tx pgx.Tx, p *Principal, id string, writing bool) (knowledgePath, error) {
	if p == nil || !validID(id) || !hasIntegrationScope(p, "document:read") {
		return knowledgePath{}, pgx.ErrNoRows
	}
	lock := " FOR SHARE OF p"
	if writing {
		lock = " FOR UPDATE OF p"
	}
	return scanKnowledgePath(tx.QueryRow(ctx, "SELECT "+knowledgePathSelect+" FROM knowledge_paths p WHERE p.id=$1 AND madi_knowledge_path_allowed($2,p.id,$3) AND ($4='' OR p.workspace_id::text=$4)"+lock, id, p.ID, writing, p.WorkspaceID))
}
func knowledgePathStepsTx(ctx context.Context, tx pgx.Tx, id string) ([]knowledgePathStep, error) {
	rows, e := tx.Query(ctx, "SELECT id::text,document_id::text,ordinal,kind,title,instruction FROM knowledge_path_steps WHERE path_id=$1 AND active ORDER BY ordinal", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []knowledgePathStep{}
	for rows.Next() {
		var v knowledgePathStep
		if e = rows.Scan(&v.ID, &v.DocumentID, &v.Ordinal, &v.Kind, &v.Title, &v.Instruction); e != nil {
			return nil, e
		}
		items = append(items, v)
	}
	return items, rows.Err()
}
func (s *Server) protectKnowledgePathTx(r *http.Request, tx pgx.Tx, in knowledgePathInput) (knowledgePathInput, error) {
	textSteps := []map[string]string{}
	for _, step := range in.Steps {
		textSteps = append(textSteps, map[string]string{"title": step.Title, "instruction": step.Instruction})
	}
	result, e := s.ProtectDocumentMetadataTx(r.Context(), tx, current(r), "", in.WorkspaceID, map[string]any{"title": in.Title, "description": in.Description, "role_labels": in.RoleLabels, "steps": textSteps})
	if e != nil {
		return in, e
	}
	var value struct {
		Title       string              `json:"title"`
		Description string              `json:"description"`
		RoleLabels  []string            `json:"role_labels"`
		Steps       []knowledgePathStep `json:"steps"`
	}
	if json.Unmarshal(jsonValue(result.Value), &value) != nil {
		return in, errors.New("경로 내용을 정제할 수 없습니다")
	}
	in.Title = value.Title
	in.Description = value.Description
	in.RoleLabels = value.RoleLabels
	if len(value.Steps) != len(in.Steps) {
		return in, errors.New("정제된 단계 수를 확인하세요")
	}
	for i := range in.Steps {
		in.Steps[i].Title = value.Steps[i].Title
		in.Steps[i].Instruction = value.Steps[i].Instruction
	}
	if !validKnowledgePathInput(&in) {
		return in, errors.New("정제된 경로의 이름·단계를 다시 확인하세요")
	}
	return in, nil
}
