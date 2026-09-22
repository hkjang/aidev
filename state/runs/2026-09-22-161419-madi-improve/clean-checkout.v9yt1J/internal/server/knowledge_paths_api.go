package server

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func learningRespondError(w http.ResponseWriter, e error) {
	if WriteProtectionError(w, e) {
		return
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && oneOf(pg.Code, "55P03", "57014") {
		apiError(w, 409, "다른 변경의 잠금 대기 또는 처리 한도를 넘었습니다. 저장 결과와 현재 버전을 다시 확인하세요")
		return
	}
	approvalRespondError(w, e)
}

func (s *Server) learningActorTx(r *http.Request, tx pgx.Tx, wid string, scopes ...string) error {
	if e := s.knowledgeActorTx(r, tx, wid, scopes...); e != nil {
		return approvalProblem(403, e.Error())
	}
	return nil
}

// This short, bounded transaction serializes the existing ACL mutation paths.
// No provider or file work is performed while these locks are held.
func (s *Server) learningTx(r *http.Request, writing bool) (pgx.Tx, error) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		return nil, e
	}
	_, e = tx.Exec(r.Context(), `SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='12s'`)
	if e == nil && writing {
		e = learningACLTx(r.Context(), tx)
	}
	if e != nil {
		tx.Rollback(r.Context())
		return nil, e
	}
	return tx, nil
}
func learningACLTx(ctx context.Context, tx pgx.Tx) error {
	if _, e := tx.Exec(ctx, `SET LOCAL lock_timeout='3s'; SET LOCAL statement_timeout='12s'`); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, `LOCK TABLE documents,spaces,space_members,document_shares IN SHARE MODE`)
	return e
}
func (s *Server) learningAdminTx(r *http.Request, tx pgx.Tx, wid string) error {
	if !learningCookie(r) {
		return approvalProblem(403, "경로 관리는 사람의 로그인 화면에서 가능합니다")
	}
	if e := s.learningActorTx(r, tx, wid, "document:read", "document:write"); e != nil {
		return approvalProblem(403, e.Error())
	}
	var role string
	if e := tx.QueryRow(r.Context(), `SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 FOR SHARE`, wid, current(r).ID).Scan(&role); e != nil || !oneOf(role, "owner", "admin") {
		return approvalProblem(403, "워크스페이스 관리자만 경로를 구성할 수 있습니다")
	}
	return nil
}
func learningDocumentsTx(ctx context.Context, tx pgx.Tx, p *Principal, wid string, steps []knowledgePathStep) (map[string]int, error) {
	ids := []string{}
	for _, step := range steps {
		ids = append(ids, step.DocumentID)
	}
	sort.Strings(ids)
	versions := map[string]int{}
	for _, id := range ids {
		if versions[id] > 0 {
			continue
		}
		var v int
		if e := tx.QueryRow(ctx, `SELECT version FROM documents WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL AND madi_document_allowed($3,id,false) FOR SHARE`, id, wid, p.ID).Scan(&v); e != nil {
			return nil, approvalProblem(404, "현재 읽을 수 있는 같은 워크스페이스 문서를 선택하세요")
		}
		versions[id] = v
	}
	return versions, nil
}
func (s *Server) listKnowledgePaths(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) {
		apiError(w, 400, "워크스페이스를 선택하세요")
		return
	}
	tx, e := s.learningTx(r, false)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	if e = s.learningActorTx(r, tx, wid, "document:read"); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	rows, e := tx.Query(r.Context(), "SELECT "+knowledgePathSelect+" FROM knowledge_paths p WHERE workspace_id=$1 AND madi_knowledge_path_allowed($2,id,false) AND ($3 OR NOT archived) ORDER BY updated_at DESC,id LIMIT 201", wid, current(r).ID, r.URL.Query().Get("archived") == "1")
	if e != nil {
		learningRespondError(w, e)
		return
	}
	out := []knowledgePath{}
	for rows.Next() {
		v, err := scanKnowledgePath(rows)
		if err != nil {
			e = err
			break
		}
		out = append(out, v)
	}
	rows.Close()
	if e == nil {
		e = rows.Err()
	}
	more := len(out) > 200
	if more {
		out = out[:200]
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		learningRespondError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"items": out, "has_more": more, "notice": "역할 이름은 안내용이며 접근 권한을 부여하지 않습니다. 현재 경로와 원문 권한을 각각 확인합니다."})
}
func (s *Server) saveKnowledgePath(w http.ResponseWriter, r *http.Request) {
	var in knowledgePathInput
	if decode(r, &in) != nil || !validKnowledgePathInput(&in) {
		apiError(w, 400, "경로 이름·공유 범위와 1~100개 읽기/실습/검토 단계를 확인하세요")
		return
	}
	if !learningCookie(r) {
		apiError(w, 403, "경로 관리는 로그인 화면에서 가능합니다")
		return
	}
	id := r.PathValue("id")
	creating := id == ""
	if creating {
		id = newID()
	} else if !validID(id) {
		apiError(w, 404, "경로가 없습니다")
		return
	}
	tx, e := s.learningTx(r, true)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = learningDocumentsTx(r.Context(), tx, current(r), in.WorkspaceID, in.Steps); e != nil {
		learningRespondError(w, e)
		return
	}
	var old knowledgePath
	if !creating {
		old, e = knowledgePathReadTx(r.Context(), tx, current(r), id, true)
		if e != nil {
			learningRespondError(w, e)
			return
		}
		if old.WorkspaceID != in.WorkspaceID || old.Revision != in.Revision {
			apiError(w, 409, "경로가 변경되었습니다. 최신 구성을 다시 확인하세요")
			return
		}
	}
	if in.Visibility == "workspace" && (creating || old.Visibility != "workspace" || old.SpaceID != in.SpaceID) && !in.ConfirmShared {
		apiError(w, 400, "경로 제목·설명·단계 안내를 대상 공간과 공유하는 데 동의하세요. 원문 권한은 별도입니다")
		return
	}
	var allowed bool
	e = tx.QueryRow(r.Context(), `SELECT $1='' OR EXISTS(SELECT 1 FROM spaces WHERE id=NULLIF($1,'')::uuid AND workspace_id=$2 AND madi_space_allowed($3,id,true))`, in.SpaceID, in.WorkspaceID, current(r).ID).Scan(&allowed)
	if e != nil || !allowed {
		apiError(w, 403, "대상 공간의 쓰기 권한을 확인하세요")
		return
	}
	in, e = s.protectKnowledgePathTx(r, tx, in)
	if e != nil {
		learningRespondError(w, e)
		return
	}
	if creating {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_paths(id,workspace_id,space_id,owner_id,title,description,role_labels,visibility,archived) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9)`, id, in.WorkspaceID, in.SpaceID, current(r).ID, in.Title, in.Description, in.RoleLabels, in.Visibility, in.Archived)
	} else {
		_, e = tx.Exec(r.Context(), `UPDATE knowledge_paths SET space_id=NULLIF($2,'')::uuid,title=$3,description=$4,role_labels=$5,visibility=$6,archived=$7,revision=revision+1,updated_at=clock_timestamp() WHERE id=$1`, id, in.SpaceID, in.Title, in.Description, in.RoleLabels, in.Visibility, in.Archived)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE knowledge_path_steps SET active=false WHERE path_id=$1`, id)
	}
	for _, step := range in.Steps {
		if e != nil {
			break
		}
		if step.ID == "" {
			step.ID = newID()
			_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_path_steps(id,path_id,document_id,ordinal,kind,title,instruction) VALUES($1,$2,$3,$4,$5,$6,$7)`, step.ID, id, step.DocumentID, step.Ordinal, step.Kind, step.Title, step.Instruction)
		} else {
			var existing string
			var kind string
			e = tx.QueryRow(r.Context(), `SELECT document_id::text,kind FROM knowledge_path_steps WHERE id=$1 AND path_id=$2 FOR UPDATE`, step.ID, id).Scan(&existing, &kind)
			if e == nil && (existing != step.DocumentID || kind != step.Kind) {
				e = approvalProblem(400, "기존 단계의 문서나 유형을 바꾸려면 새 단계로 추가하세요. 과거 기록은 보존됩니다")
			}
			if e == nil {
				_, e = tx.Exec(r.Context(), `UPDATE knowledge_path_steps SET ordinal=$2,title=$3,instruction=$4,active=true,updated_at=clock_timestamp() WHERE id=$1`, step.ID, step.Ordinal, step.Title, step.Instruction)
			}
		}
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_path_events(id,path_id,user_id,action,metadata) VALUES($1,$2,$3,'path_configured',jsonb_build_object('revision',(SELECT revision FROM knowledge_paths WHERE id=$2)))`, newID(), id, current(r).ID)
	}
	if e == nil {
		e = s.learningAdminTx(r, tx, in.WorkspaceID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			apiError(w, 400, "기존 단계 ID는 이 경로에 속해야 합니다")
		} else {
			learningRespondError(w, e)
		}
		return
	}
	jsonResponse(w, 200, map[string]any{"id": id, "revision": old.Revision + 1})
}
