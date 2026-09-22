package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

func (s *Server) agentFail(ctx context.Context, id, jobID string, err error) (map[string]any, error) {
	// Persist a redacted terminal state even when the provider request context
	// was cancelled. A newer continuation job must never be overwritten.
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	tx, e := s.DB.Begin(cleanup)
	if e == nil {
		defer tx.Rollback(cleanup)
		var currentJob, status string
		e = tx.QueryRow(cleanup, `SELECT coalesce(job_id::text,''),status FROM agent_runs WHERE id=$1 FOR UPDATE`, id).Scan(&currentJob, &status)
		if e == nil && currentJob == jobID && oneOf(status, "pending", "running") {
			message := agentPublicError(err)
			_, e = tx.Exec(cleanup, `UPDATE agent_runs SET status='failed',error=$2,updated_at=now() WHERE id=$1`, id, message)
			if e == nil {
				_, e = tx.Exec(cleanup, `UPDATE agent_actions a SET status=CASE WHEN EXISTS(SELECT 1 FROM automation_effects x WHERE x.job_id=a.job_id AND x.action_index=0) THEN 'applied' ELSE 'failed' END WHERE run_id=$1 AND job_id=$2 AND status='confirmed'`, id, jobID)
			}
			if e == nil {
				e = agentEventTx(cleanup, tx, id, "retract", map[string]any{"error": message})
			}
			if e == nil {
				e = agentEventTx(cleanup, tx, id, "status", map[string]any{"status": "failed"})
			}
			if e == nil {
				e = tx.Commit(cleanup)
			}
		}
	}
	return nil, jobPermanent(agentPublicError(err))
}
func (s *Server) agentAppendEvent(ctx context.Context, run agentRun, p *Principal, kind string, value any) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	a, e := s.agentRunTx(ctx, tx, run, p)
	if e == nil {
		e = s.agentSourceTx(ctx, tx, p, a, run.ID, "", 0)
	}
	if e == nil {
		e = agentEventTx(ctx, tx, run.ID, kind, value)
	}
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func agentAvailableContext(a workspaceAgent) string {
	return "\n현재 도구용 지식 범위 식별자(접근 권한은 도구가 별도 검증): " + string(jsonValue(map[string]any{"space_ids": a.SpaceIDs, "document_ids": a.DocumentIDs, "database_ids": a.DatabaseIDs}))
}

func (s *Server) runWorkspaceAgent(ctx context.Context, j Job) (map[string]any, error) {
	id := str(j.Payload, "run_id")
	for {
		run, a, p, cfg, e := s.agentGuard(ctx, id)
		if e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		if run.JobID != j.ID || run.OwnerID != j.ActorID || run.TokenID != j.TokenID {
			return nil, jobPermanent("Agent 작업 맥락이 일치하지 않습니다")
		}
		if run.Step >= a.MaxSteps {
			return s.agentFail(ctx, id, j.ID, jobPermanent("Agent 최대 실행 단계에 도달했습니다. 결과를 확인한 뒤 새 질문을 시작하세요"))
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		if _, e = s.agentRunTx(ctx, tx, run, p); e == nil {
			_, e = tx.Exec(ctx, `UPDATE agent_runs SET status='running' WHERE id=$1`, id)
		}
		if e == nil {
			e = agentEventTx(ctx, tx, id, "step_start", map[string]any{"step": run.Step + 1, "max_steps": a.MaxSteps})
		}
		if e == nil {
			e = tx.Commit(ctx)
		} else {
			tx.Rollback(ctx)
		}
		if e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		a = agentSortedTools(a, p)
		if len(a.Tools) == 0 {
			return s.agentFail(ctx, id, j.ID, errAgentChanged)
		}
		messages := append([]agentMessage{}, run.Messages...)
		if len(messages) > 0 {
			visible := a
			visible.DocumentIDs = []string{}
			visible.SpaceIDs = []string{}
			visible.DatabaseIDs = []string{}
			for _, id := range a.DocumentIDs {
				if agentDocumentAllowed(ctx, s.DB, p, a, id, false) {
					visible.DocumentIDs = append(visible.DocumentIDs, id)
				}
			}
			for _, id := range a.SpaceIDs {
				if s.canSpace(ctx, p, a.WorkspaceID, id, false) {
					visible.SpaceIDs = append(visible.SpaceIDs, id)
				}
			}
			for _, id := range a.DatabaseIDs {
				if hasIntegrationScope(p, "database:read") && s.canDatabase(automationRequest(ctx, p, http.MethodGet, nil), id, false) {
					visible.DatabaseIDs = append(visible.DatabaseIDs, id)
				}
			}
			messages[0].Content += agentAvailableContext(visible)
		}
		watched, stop, guardError := ragWatch(ctx, func(check context.Context) error { _, _, _, _, e := s.agentGuard(check, id); return e })
		var pending strings.Builder
		lastFlush := time.Now()
		flush := func() error {
			if pending.Len() == 0 {
				return nil
			}
			value := pending.String()
			pending.Reset()
			lastFlush = time.Now()
			for len(value) > 0 {
				end := min(8192, len(value))
				for end < len(value) && !utf8.RuneStart(value[end]) {
					end--
				}
				_, _, current, _, e := s.agentGuard(watched, id)
				if e != nil {
					return e
				}
				if e = s.agentAppendEvent(watched, run, current, "delta", map[string]any{"step": run.Step + 1, "text": value[:end]}); e != nil {
					return e
				}
				value = value[end:]
			}
			return nil
		}
		turn, providerErr := agentModelTurn(watched, cfg, a, messages, func(value string) error {
			pending.WriteString(value)
			if pending.Len() >= 1024 || time.Since(lastFlush) > 100*time.Millisecond {
				return flush()
			}
			return nil
		})
		if providerErr == nil {
			providerErr = flush()
		}
		stop()
		if *guardError != nil {
			return s.agentFail(ctx, id, j.ID, *guardError)
		}
		if providerErr != nil {
			return s.agentFail(ctx, id, j.ID, providerErr)
		}
		run, a, p, _, e = s.agentGuard(ctx, id)
		if e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		writes := 0
		for _, call := range turn.ToolCalls {
			if oneOf(call.Function.Name, "create_document", "update_document") {
				writes++
			}
		}
		if writes > 0 && (writes != 1 || len(turn.ToolCalls) != 1) {
			return s.agentFail(ctx, id, j.ID, jobPermanent("문서 변경 제안은 한 단계에 한 개씩 요청해야 합니다"))
		}
		nextMessages := append(append([]agentMessage{}, run.Messages...), turn)
		sources := []agentSource{}
		toolEvents := []map[string]any{}
		var plan *agentWritePlan
		var actionID, actionHash string
		for _, call := range turn.ToolCalls {
			_, a, p, _, e = s.agentGuard(ctx, id)
			if e != nil {
				return s.agentFail(ctx, id, j.ID, e)
			}
			if writes == 1 {
				value, ex := s.agentValidatePlan(ctx, p, a, call)
				if ex != nil {
					return s.agentFail(ctx, id, j.ID, jobPermanent(ex.Error()))
				}
				if call.Function.Name == "update_document" {
					var read bool
					if s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM agent_sources WHERE run_id=$1 AND kind='document' AND resource_id=$2 AND fingerprint=$3)`, id, value.DocumentID, fmt.Sprint(value.ExpectedVersion)).Scan(&read) != nil || !read {
						return s.agentFail(ctx, id, j.ID, jobPermanent("문서 변경 전에 해당 버전의 문서를 읽어야 합니다"))
					}
				}
				plan = &value
				actionID = newID()
				actionHash = digest(string(jsonValue(map[string]any{"run_id": id, "action_id": actionID, "tool": call.Function.Name, "arguments": value, "agent_revision": run.Revision})))
				break
			}
			result, readSources, ex := s.agentReadTool(ctx, p, a, call)
			if ex != nil {
				result = map[string]any{"error": "도구 입력이나 현재 지식 범위 권한을 확인하세요"}
				readSources = nil
			}
			raw := jsonValue(result)
			if len(raw) > 65536 {
				return s.agentFail(ctx, id, j.ID, jobPermanent("Agent 도구 출력이 64KB를 초과했습니다"))
			}
			nextMessages = append(nextMessages, agentMessage{Role: "tool", ToolCallID: call.ID, Content: string(raw)})
			sources = append(sources, readSources...)
			toolEvents = append(toolEvents, map[string]any{"step": run.Step + 1, "tool": call.Function.Name, "tool_call_id": call.ID, "result": result})
		}
		if len(jsonValue(nextMessages)) > 4<<20 {
			return s.agentFail(ctx, id, j.ID, jobPermanent("Agent 대화 컨텍스트가 4MB 한도에 도달했습니다"))
		}
		// Check the values just read before committing their event or sending to
		// the model. A failed check never leaves a stale tool result in history.
		tx, e = s.DB.Begin(ctx)
		if e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		a, e = s.agentRunTx(ctx, tx, run, p)
		if e == nil {
			for _, src := range sources {
				var previous string
				ex := tx.QueryRow(ctx, `SELECT fingerprint FROM agent_sources WHERE run_id=$1 AND source_key=$2`, id, src.Key).Scan(&previous)
				if ex == nil && previous != src.Fingerprint {
					e = errAgentChanged
					break
				}
				if ex != nil && !errors.Is(ex, pgx.ErrNoRows) {
					e = ex
					break
				}
				_, e = tx.Exec(ctx, `INSERT INTO agent_sources(run_id,source_key,kind,resource_id,snapshot,fingerprint) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(run_id,source_key) DO NOTHING`, id, src.Key, src.Kind, src.ResourceID, jsonValue(src.Snapshot), src.Fingerprint)
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			var count int
			e = tx.QueryRow(ctx, `SELECT count(*) FROM agent_sources WHERE run_id=$1`, id).Scan(&count)
			if e == nil && count > 256 {
				e = jobPermanent("Agent가 참조할 수 있는 원본은 실행당 최대 256개입니다")
			}
		}
		if e == nil {
			e = s.agentSourceTx(ctx, tx, p, a, id, "", 0)
		}
		status := "running"
		if len(turn.ToolCalls) == 0 {
			status = "succeeded"
		}
		if plan != nil {
			status = "awaiting_confirmation"
		}
		if e == nil {
			_, e = tx.Exec(ctx, `UPDATE agent_runs SET messages=$2,step=step+1,status=$3,updated_at=now() WHERE id=$1`, id, jsonValue(nextMessages), status)
		}
		if e == nil && plan != nil {
			call := turn.ToolCalls[0]
			_, e = tx.Exec(ctx, `INSERT INTO agent_actions(id,run_id,tool_call_id,tool,arguments,action_hash) VALUES($1,$2,$3,$4,$5,$6)`, actionID, id, call.ID, call.Function.Name, jsonValue(plan), actionHash)
			if e == nil {
				e = agentEventTx(ctx, tx, id, "action_required", map[string]any{"action_id": actionID, "action_hash": actionHash, "tool": call.Function.Name, "arguments": plan})
			}
		}
		if e == nil {
			for _, event := range toolEvents {
				if e = agentEventTx(ctx, tx, id, "tool_result", event); e != nil {
					break
				}
			}
		}
		if e == nil {
			e = agentEventTx(ctx, tx, id, "step_end", map[string]any{"step": run.Step + 1, "status": status})
		}
		if e == nil {
			e = agentEventTx(ctx, tx, id, "status", map[string]any{"status": status})
		}
		if e == nil {
			e = tx.Commit(ctx)
		} else {
			tx.Rollback(ctx)
		}
		if e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		if status != "running" {
			return map[string]any{"run_id": id, "status": status, "steps": run.Step + 1}, nil
		}
	}
}

func (s *Server) runAgentAction(ctx context.Context, j Job) (map[string]any, error) {
	id, actionID := str(j.Payload, "run_id"), str(j.Payload, "action_id")
	run, a, p, _, e := s.agentGuard(ctx, id)
	if e != nil {
		return s.agentFail(ctx, id, j.ID, e)
	}
	if run.JobID != j.ID || run.OwnerID != j.ActorID || run.TokenID != j.TokenID || run.WorkspaceID != j.WorkspaceID {
		return nil, jobPermanent("이미 교체된 Agent 작업입니다")
	}
	var status, tool, callID, hash string
	var raw []byte
	e = s.DB.QueryRow(ctx, `SELECT status,tool,tool_call_id,action_hash,arguments FROM agent_actions WHERE id=$1 AND run_id=$2 AND job_id=$3`, actionID, id, j.ID).Scan(&status, &tool, &callID, &hash, &raw)
	if e != nil || status != "confirmed" {
		return s.agentFail(ctx, id, j.ID, errAgentChanged)
	}
	var plan agentWritePlan
	if json.Unmarshal(raw, &plan) != nil {
		return s.agentFail(ctx, id, j.ID, errAgentChanged)
	}
	var receiptRaw []byte
	e = s.DB.QueryRow(ctx, `SELECT result FROM automation_effects WHERE job_id=$1 AND action_index=0`, j.ID).Scan(&receiptRaw)
	if errors.Is(e, pgx.ErrNoRows) {
		call := agentToolCall{ID: callID, Type: "function"}
		call.Function.Name = tool
		call.Function.Arguments = string(raw)
		if _, e = s.agentValidatePlan(ctx, p, a, call); e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		actionCtx := context.WithValue(ctx, automationEffectKey{}, automationEffectContext{JobID: j.ID, Index: 0})
		actionCtx = withMutationGuard(actionCtx, func(check context.Context, tx pgx.Tx) error {
			fresh, e := s.agentRunTx(check, tx, run, p)
			if e != nil {
				return e
			}
			if e = s.agentActorTx(check, tx, p, run, "document:write"); e != nil {
				return e
			}
			var currentHash, currentStatus string
			if tx.QueryRow(check, `SELECT action_hash,status FROM agent_actions WHERE id=$1 AND run_id=$2 AND job_id=$3 FOR UPDATE`, actionID, id, j.ID).Scan(&currentHash, &currentStatus) != nil || currentHash != hash || currentStatus != "confirmed" {
				return errAgentChanged
			}
			if e = s.agentSourceTx(check, tx, p, fresh, id, plan.DocumentID, plan.ExpectedVersion); e != nil {
				return e
			}
			if tool == "update_document" {
				if !agentDocumentAllowed(check, tx, p, fresh, plan.DocumentID, true) {
					return errAgentChanged
				}
				var title string
				var version int
				if tx.QueryRow(check, `SELECT title,version FROM documents WHERE id=$1`, plan.DocumentID).Scan(&title, &version) != nil || version != plan.ExpectedVersion+1 {
					return errAgentChanged
				}
				src := agentDocSource(plan.DocumentID, version, title)
				_, e = tx.Exec(check, `UPDATE agent_sources SET snapshot=$3,fingerprint=$4 WHERE run_id=$1 AND source_key=$2`, id, src.Key, jsonValue(src.Snapshot), src.Fingerprint)
				return e
			}
			var allowed bool
			if !slicesContains(fresh.SpaceIDs, plan.SpaceID) || tx.QueryRow(check, `SELECT madi_space_allowed($1,$2,true)`, p.ID, plan.SpaceID).Scan(&allowed) != nil || !allowed {
				return errAgentChanged
			}
			return nil
		})
		payload := map[string]any{"title": plan.Title, "markdown": plan.Markdown}
		method := http.MethodPut
		if tool == "create_document" {
			method = http.MethodPost
			payload["workspace_id"] = a.WorkspaceID
			payload["space_id"] = plan.SpaceID
			payload["visibility"] = plan.Visibility
		} else {
			payload["version"] = plan.ExpectedVersion
		}
		req := automationRequest(actionCtx, p, method, payload)
		req.SetPathValue("id", plan.DocumentID)
		rec := httptest.NewRecorder()
		if tool == "create_document" {
			s.createDocument(rec, req)
		} else {
			s.updateDocument(rec, req)
		}
		if _, e = automationResponse(rec); e != nil {
			return s.agentFail(ctx, id, j.ID, e)
		}
		e = s.DB.QueryRow(ctx, `SELECT result FROM automation_effects WHERE job_id=$1 AND action_index=0`, j.ID).Scan(&receiptRaw)
	}
	if e != nil {
		return s.agentFail(ctx, id, j.ID, e)
	}
	var result map[string]any
	if json.Unmarshal(receiptRaw, &result) != nil {
		return s.agentFail(ctx, id, j.ID, errAgentChanged)
	}
	// The canonical saved document may be masked by information-protection
	// policy. Read the actual committed title/version instead of claiming the
	// proposed plaintext was stored unchanged.
	docID := str(result, "id")
	var title, documentStatus string
	var version int
	if !agentDocumentAllowed(ctx, s.DB, p, a, docID, false) || s.DB.QueryRow(ctx, `SELECT title,version,status FROM documents WHERE id=$1 AND deleted_at IS NULL`, docID).Scan(&title, &version, &documentStatus) != nil {
		return s.agentFail(ctx, id, j.ID, errAgentChanged)
	}
	result = map[string]any{"id": docID, "title": title, "version": version, "status": documentStatus, "url": "/app/documents/" + docID, "applied": true}
	run, _, p, _, e = s.agentGuard(ctx, id)
	if e != nil {
		return s.agentFail(ctx, id, j.ID, e)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return s.agentFail(ctx, id, j.ID, e)
	}
	defer tx.Rollback(ctx)
	if _, e = s.agentRunTx(ctx, tx, run, p); e != nil {
		tx.Rollback(ctx)
		return s.agentFail(ctx, id, j.ID, e)
	}
	src := agentDocSource(docID, version, title)
	_, e = tx.Exec(ctx, `INSERT INTO agent_sources(run_id,source_key,kind,resource_id,snapshot,fingerprint) VALUES($1,$2,'document',$3,$4,$5) ON CONFLICT(run_id,source_key) DO UPDATE SET snapshot=excluded.snapshot,fingerprint=excluded.fingerprint`, id, src.Key, docID, jsonValue(src.Snapshot), src.Fingerprint)
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE agent_actions SET status='applied',result=$2 WHERE id=$1 AND status='confirmed'`, actionID, jsonValue(result))
	}
	if e == nil {
		run.Messages = append(run.Messages, agentMessage{Role: "tool", ToolCallID: callID, Content: string(jsonValue(result))})
		_, e = tx.Exec(ctx, `UPDATE agent_runs SET messages=$2 WHERE id=$1`, id, jsonValue(run.Messages))
	}
	if e == nil {
		e = agentEventTx(ctx, tx, id, "action_applied", map[string]any{"action_id": actionID, "result": result})
	}
	var next string
	if e == nil {
		next, e = s.enqueueAgentJob(ctx, tx, run, "agent.run", map[string]any{"run_id": id})
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		tx.Rollback(ctx)
		return s.agentFail(ctx, id, j.ID, e)
	}
	return map[string]any{"applied": true, "document_id": docID, "continuation_job_id": next}, nil
}

func (s *Server) registerWorkspaceAgent() {
	s.handle("GET /api/v1/workspaces/{id}/agents", s.listWorkspaceAgents)
	s.handle("POST /api/v1/workspaces/{id}/agents", s.saveWorkspaceAgent)
	s.handle("PUT /api/v1/agents/{id}", s.saveWorkspaceAgent)
	s.handle("GET /api/v1/agents/{id}/runs", s.listAgentRuns)
	s.handle("POST /api/v1/agents/{id}/runs", s.createAgentRun)
	s.handle("GET /api/v1/agent-runs/{id}", s.getAgentRun)
	s.handle("GET /api/v1/agent-runs/{id}/events", s.agentEvents)
	s.handle("POST /api/v1/agent-runs/{id}/cancel", s.cancelAgentRun)
	s.handle("POST /api/v1/agent-runs/{id}/actions/{action}/confirm", s.decideAgentAction)
	s.RegisterJobHandler("agent.run", s.runWorkspaceAgent)
	s.RegisterJobHandler("agent.action", s.runAgentAction)
}
