package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func runbookRedactRunner(r runbookRunner) runbookRunner {
	copy := map[string]any{}
	for k, v := range r.Config {
		copy[k] = v
	}
	copy["token_configured"] = str(copy, "token") != ""
	copy["token"] = ""
	r.Config = copy
	return r
}
func (s *Server) loadRunbookRunner(ctx context.Context, id string, revision int64) (runbookRunner, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return runbookRunner{}, e
	}
	defer tx.Rollback(ctx)
	r, e := runbookRunnerTx(ctx, tx, id, revision)
	if e == nil {
		e = tx.Commit(ctx)
	}
	return r, e
}
func (s *Server) adminRunbookSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		out, e := s.one(r.Context(), `SELECT to_jsonb(s) FROM runbook_settings s WHERE id=1`)
		respond(w, out, e)
		return
	}
	var in struct {
		Enabled  bool  `json:"enabled"`
		Revision int64 `json:"revision"`
	}
	if decode(r, &in) != nil || in.Revision < 1 {
		apiError(w, 400, "격리 실행 사용 여부와 설정 버전을 확인하세요")
		return
	}
	tag, e := s.DB.Exec(r.Context(), `UPDATE runbook_settings SET enabled=$1,revision=revision+1 WHERE id=1 AND revision=$2`, in.Enabled, in.Revision)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "다른 관리자가 실행 설정을 변경했습니다")
		return
	}
	s.audit(r, "RUNBOOK_SETTINGS_SAVE", "1", in)
	jsonResponse(w, 200, map[string]any{"enabled": in.Enabled, "revision": in.Revision + 1})
}
func (s *Server) adminRunbookRunners(w http.ResponseWriter, r *http.Request) {
	rows, e := s.DB.Query(r.Context(), `SELECT id::text FROM runbook_runners ORDER BY name`)
	if e != nil {
		respond(w, nil, e)
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []runbookRunner{}
	for _, id := range ids {
		runner, e := s.loadRunbookRunner(r.Context(), id, 0)
		if e != nil {
			respond(w, nil, e)
			return
		}
		out = append(out, runbookRedactRunner(runner))
	}
	jsonResponse(w, 200, out)
}
func (s *Server) adminSaveRunbookRunner(w http.ResponseWriter, r *http.Request) {
	var in runbookRunner
	if decode(r, &in) != nil {
		apiError(w, 400, "실행기 설정 형식을 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	newToken := str(in.Config, "token")
	if r.Method == "POST" {
		in.ID = newID()
		in.Revision = 1
	} else {
		in.ID = r.PathValue("id")
		if !validID(in.ID) || in.Revision < 1 {
			apiError(w, 400, "실행기 ID와 버전을 확인하세요")
			return
		}
		var revision int64
		var wid, kind string
		e = tx.QueryRow(r.Context(), `SELECT revision,workspace_id::text,kind FROM runbook_runners WHERE id=$1 FOR UPDATE`, in.ID).Scan(&revision, &wid, &kind)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if revision != in.Revision || wid != in.WorkspaceID || kind != in.Kind {
			apiError(w, 409, "실행기 버전 또는 고정 워크스페이스·유형이 변경되었습니다")
			return
		}
		old, e := runbookRunnerTx(r.Context(), tx, in.ID, revision)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if newToken == "" {
			if in.Config == nil {
				in.Config = map[string]any{}
			}
			in.Config["token"] = str(old.Config, "token")
		}
		in.Revision++
	}
	if e = validateRunbookRunner(&in); e != nil {
		approvalRespondError(w, e)
		return
	}
	if newToken != "" {
		in.Config["token"], e = s.encrypt(newToken)
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	delete(in.Config, "token_configured")
	var exists bool
	if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM workspaces WHERE id=$1)`, in.WorkspaceID).Scan(&exists); e != nil {
		respond(w, nil, e)
		return
	}
	if !exists {
		apiError(w, 400, "워크스페이스를 찾을 수 없습니다")
		return
	}
	for _, action := range in.Actions {
		for _, team := range action.Teams {
			if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM teams WHERE id=$1 AND workspace_id=$2)`, team, in.WorkspaceID).Scan(&exists); e != nil {
				respond(w, nil, e)
				return
			}
			if !exists {
				apiError(w, 400, "허용 팀은 같은 워크스페이스에 있어야 합니다")
				return
			}
		}
	}
	if r.Method == "POST" {
		_, e = tx.Exec(r.Context(), `INSERT INTO runbook_runners(id,workspace_id,name,kind,enabled,revision,created_by) VALUES($1,$2,$3,$4,$5,$6,$7)`, in.ID, in.WorkspaceID, in.Name, in.Kind, in.Enabled, in.Revision, current(r).ID)
	} else {
		_, e = tx.Exec(r.Context(), `UPDATE runbook_runners SET name=$2,enabled=$3,revision=$4,updated_at=now() WHERE id=$1`, in.ID, in.Name, in.Enabled, in.Revision)
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO runbook_runner_versions(runner_id,revision,config,actions,created_by) VALUES($1,$2,$3,$4,$5)`, in.ID, in.Revision, jsonValue(in.Config), jsonValue(in.Actions), current(r).ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RUNBOOK_RUNNER_SAVE", in.ID, map[string]any{"revision": in.Revision, "enabled": in.Enabled})
	jsonResponse(w, 200, runbookRedactRunner(in))
}
func (s *Server) adminTestRunbookRunner(w http.ResponseWriter, r *http.Request) {
	runner, e := s.loadRunbookRunner(r.Context(), r.PathValue("id"), 0)
	if e != nil {
		respond(w, nil, e)
		return
	}
	remote, e := s.newRunbookRemote(runner)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	defer remote.close()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	reports := []any{}
	for _, action := range runner.Actions {
		report, e := remote.preflight(ctx, action)
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		reports = append(reports, map[string]any{"action_id": action.ID, "checks": report})
	}
	s.audit(r, "RUNBOOK_RUNNER_CHECK", runner.ID, nil)
	jsonResponse(w, 200, map[string]any{"ok": true, "actions": reports})
}
func (s *Server) runbookRunners(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) || !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = runbookEnabledTx(r.Context(), tx); e != nil {
		approvalRespondError(w, e)
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT id::text FROM runbook_runners WHERE workspace_id=$1 AND enabled ORDER BY name`, wid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			break
		}
		ids = append(ids, id)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	out := []any{}
	for _, id := range ids {
		runner, e := runbookRunnerTx(r.Context(), tx, id, 0)
		if e != nil {
			respond(w, nil, e)
			return
		}
		actions := []runbookAction{}
		for _, action := range runner.Actions {
			allowed, e := runbookActionAllowedTx(r.Context(), tx, current(r).ID, wid, action)
			if e != nil {
				respond(w, nil, e)
				return
			}
			if allowed {
				actions = append(actions, action)
			}
		}
		if len(actions) > 0 {
			out = append(out, map[string]any{"id": runner.ID, "name": runner.Name, "kind": runner.Kind, "revision": runner.Revision, "actions": actions})
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, out)
}
func (s *Server) getRunbook(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	doc, e := runbookDocumentTx(r.Context(), tx, current(r), r.PathValue("id"), false)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if _, e = runbookEnabledTx(r.Context(), tx); e != nil {
		approvalRespondError(w, e)
		return
	}
	definition, e := runbookDefinitionTx(r.Context(), tx, r.PathValue("id"))
	if errors.Is(e, pgx.ErrNoRows) {
		definition = runbookDefinition{DocumentID: r.PathValue("id"), Steps: []runbookStep{}, ValidationSteps: []runbookStep{}, RollbackSteps: []runbookStep{}}
		e = nil
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	var tested any
	var tester string
	if definition.Version > 0 {
		if e = tx.QueryRow(r.Context(), `SELECT last_tested_at,COALESCE(last_tested_by::text,'') FROM runbook_documents WHERE document_id=$1`, definition.DocumentID).Scan(&tested, &tester); e != nil {
			respond(w, nil, e)
			return
		}
	}
	var canWrite bool
	if e = tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,true)`, current(r).ID, definition.DocumentID).Scan(&canWrite); e != nil {
		respond(w, nil, e)
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"definition": definition, "title": doc["title"], "owner_id": doc["owner_id"], "workspace_id": doc["workspace_id"], "can_write": canWrite && hasIntegrationScope(current(r), "document:write"), "document_version": doc["version"], "last_tested_at": tested, "last_tested_by": tester})
}
func (s *Server) saveRunbook(w http.ResponseWriter, r *http.Request) {
	var in runbookDefinition
	if decode(r, &in) != nil {
		apiError(w, 400, "운영 절차 형식을 확인하세요")
		return
	}
	in.DocumentID = r.PathValue("id")
	if e := validateRunbookDefinition(&in); e != nil {
		approvalRespondError(w, e)
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	doc, e := runbookDocumentTx(r.Context(), tx, current(r), in.DocumentID, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for _, steps := range [][]runbookStep{in.Steps, in.ValidationSteps, in.RollbackSteps} {
		for _, step := range steps {
			runner, e := runbookRunnerTx(r.Context(), tx, step.RunnerID, 0)
			if e != nil {
				respond(w, nil, e)
				return
			}
			if runner.WorkspaceID != str(doc, "workspace_id") {
				apiError(w, 403, "다른 워크스페이스 실행기를 사용할 수 없습니다")
				return
			}
			found := false
			for _, a := range runner.Actions {
				if a.ID == step.ActionID {
					found = true
					if _, _, e = runbookParameters(a, step.Parameters); e != nil {
						approvalRespondError(w, e)
						return
					}
				}
			}
			if !found {
				apiError(w, 400, "허용된 실행 작업을 선택하세요")
				return
			}
		}
	}
	if _, e = runbookEnabledTx(r.Context(), tx); e != nil {
		approvalRespondError(w, e)
		return
	}
	tag, e := tx.Exec(r.Context(), `INSERT INTO runbook_documents(document_id,purpose,prerequisites,validation,rollback,steps,validation_steps,rollback_steps) SELECT $1,$2,$3,$4,$5,$6,$7,$8 WHERE $9::bigint=0 ON CONFLICT(document_id) DO NOTHING`, in.DocumentID, in.Purpose, in.Prerequisites, in.Validation, in.Rollback, jsonValue(in.Steps), jsonValue(in.ValidationSteps), jsonValue(in.RollbackSteps), in.Version)
	if e == nil && in.Version > 0 {
		tag, e = tx.Exec(r.Context(), `UPDATE runbook_documents SET purpose=$2,prerequisites=$3,validation=$4,rollback=$5,steps=$6,validation_steps=$7,rollback_steps=$8,version=version+1,updated_at=now() WHERE document_id=$1 AND version=$9`, in.DocumentID, in.Purpose, in.Prerequisites, in.Validation, in.Rollback, jsonValue(in.Steps), jsonValue(in.ValidationSteps), jsonValue(in.RollbackSteps), in.Version)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "운영 절차 버전이 변경되었습니다. 다시 불러오세요")
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RUNBOOK_SAVE", in.DocumentID, map[string]any{"version": in.Version + 1})
	s.getRunbook(w, r)
}
func (s *Server) prepareRunbook(w http.ResponseWriter, r *http.Request) {
	if current(r).Kind != "user" || current(r).TokenID != "" || current(r).ScopeRestricted {
		apiError(w, 403, "실행 계획은 실제 사용자의 로그인 세션에서 준비하세요")
		return
	}
	if !allowRunbookPreparation(current(r).ID, time.Now()) {
		apiError(w, 429, "실행 계획 준비는 사용자당 1분에 10회 이하입니다")
		return
	}
	var in struct {
		Phase string `json:"phase"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "실행 계획 형식을 확인하세요")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	plan, e := s.buildRunbookPlanTx(ctx, tx, current(r), r.PathValue("id"), in.Phase)
	if e == nil {
		e = tx.Commit(ctx)
	} else {
		tx.Rollback(ctx)
	}
	_ = tx.Rollback(ctx)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	for i, step := range plan.Steps {
		runner, e := s.loadRunbookRunner(ctx, step.RunnerID, step.RunnerRevision)
		if e != nil {
			respond(w, nil, e)
			return
		}
		remote, e := s.newRunbookRemote(runner)
		if e != nil {
			approvalRespondError(w, e)
			return
		}
		report, e := remote.preflight(ctx, step.Action)
		remote.close()
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		plan.Steps[i].Remote = report
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(ctx)
	fresh, e := s.buildRunbookPlanTx(ctx, tx, current(r), r.PathValue("id"), in.Phase)
	if e != nil {
		approvalRespondError(w, e)
		return
	}
	if runbookLocalPlanHash(fresh) != runbookLocalPlanHash(plan) {
		apiError(w, 409, "실행기 확인 중 원본이나 설정이 변경되었습니다. 다시 준비하세요")
		return
	}
	var approval bool
	if e = tx.QueryRow(ctx, `SELECT COALESCE((data->>'approval_enabled')::boolean,false) FROM settings WHERE id=1 FOR SHARE`).Scan(&approval); e != nil {
		respond(w, nil, e)
		return
	}
	var wid string
	if e = tx.QueryRow(ctx, `SELECT workspace_id::text FROM documents WHERE id=$1`, plan.DocumentID).Scan(&wid); e != nil {
		respond(w, nil, e)
		return
	}
	id := newID()
	if _, e = tx.Exec(ctx, `INSERT INTO runbook_executions(id,workspace_id,document_id,owner_id,phase,snapshot,approval_required) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, wid, plan.DocumentID, current(r).ID, plan.Phase, jsonValue(plan), approval); e != nil {
		respond(w, nil, e)
		return
	}
	for i, step := range plan.Steps {
		if _, e = tx.Exec(ctx, `INSERT INTO runbook_execution_steps(execution_id,step_index,runner_id,runner_revision,action_id) VALUES($1,$2,$3,$4,$5)`, id, i, step.RunnerID, step.RunnerRevision, step.Action.ID); e != nil {
			respond(w, nil, e)
			return
		}
	}
	if e = tx.Commit(ctx); e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "RUNBOOK_PREPARE", id, map[string]any{"document_id": plan.DocumentID, "phase": plan.Phase})
	jsonResponse(w, 201, map[string]any{"id": id, "version": 1, "snapshot": plan, "approval_required": approval, "status": "prepared"})
}

func runbookIDVersion(r *http.Request) (string, int64, error) {
	id := r.PathValue("id")
	version, e := strconv.ParseInt(r.PathValue("version"), 10, 64)
	if !validID(id) || e != nil || version < 1 {
		return "", 0, fmt.Errorf("실행기 버전을 확인하세요")
	}
	return id, version, nil
}
func (s *Server) adminRunbookVersions(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("version") != "" {
		id, version, e := runbookIDVersion(r)
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		runner, e := s.loadRunbookRunner(r.Context(), id, version)
		if e != nil {
			respond(w, nil, e)
			return
		}
		runner.Revision = version
		jsonResponse(w, 200, runbookRedactRunner(runner))
		return
	}
	if !validID(r.PathValue("id")) {
		apiError(w, 404, "실행기를 찾을 수 없습니다")
		return
	}
	rows, e := s.rows(r.Context(), `SELECT jsonb_build_object('revision',v.revision,'created_at',v.created_at,'created_by',v.created_by,'name',u.name) FROM runbook_runner_versions v JOIN users u ON u.id=v.created_by WHERE v.runner_id=$1 ORDER BY v.revision DESC LIMIT 100`, r.PathValue("id"))
	respond(w, rows, e)
}

func runbookHuman(r *http.Request) bool {
	return current(r).Kind == "user" && current(r).TokenID == "" && !current(r).ScopeRestricted
}
func runbookPhaseName(phase string) string {
	return map[string]string{"execute": "실행", "validate": "검증", "rollback": "롤백"}[phase]
}
func runbookCleanLog(log, token string) string {
	if token != "" {
		log = strings.ReplaceAll(log, token, "[실행기 자격 증명 숨김]")
	}
	return strings.Map(func(r rune) rune {
		if r < ' ' && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, strings.ToValidUTF8(log, "�"))
}
