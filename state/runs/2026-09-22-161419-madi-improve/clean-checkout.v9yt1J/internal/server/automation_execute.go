package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/jackc/pgx/v5"
)

type automationEffectContext struct {
	JobID string
	Index int
}
type automationEffectKey struct{}

// recordAutomationEffect is a no-op for normal API requests. Existing mutation
// handlers call it before COMMIT so a process crash cannot duplicate a create.
func (s *Server) recordAutomationEffect(ctx context.Context, tx pgx.Tx, result map[string]any) error {
	if guard, ok := ctx.Value(mutationGuardKey{}).(mutationGuard); ok {
		if err := guard(ctx, tx); err != nil {
			return err
		}
	}
	if guard, ok := ctx.Value(mutationResultGuardKey{}).(mutationResultGuard); ok {
		if err := guard(ctx, tx, result); err != nil {
			return err
		}
	}
	effect, ok := ctx.Value(automationEffectKey{}).(automationEffectContext)
	if !ok {
		return nil
	}
	job, ok := ctx.Value(jobContextKey{}).(jobContext)
	if !ok {
		return errors.New("자동화 실행 임대 정보가 없습니다")
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT status='running' AND NOT cancel_requested AND lease_id=$2::uuid AND lease_until>now() FROM automation_jobs WHERE id=$1 FOR UPDATE`, effect.JobID, job.LeaseID).Scan(&active); err != nil {
		return err
	}
	if !active {
		return jobPermanent("자동화 실행 임대가 만료되었거나 취소되었습니다")
	}
	_, err := tx.Exec(ctx, `INSERT INTO automation_effects(job_id,action_index,result) VALUES($1,$2,$3)`, effect.JobID, effect.Index, jsonValue(result))
	return err
}
func automationRequest(ctx context.Context, p *Principal, method string, payload any) *http.Request {
	r, _ := http.NewRequestWithContext(context.WithValue(ctx, principalKey, p), method, "http://madi.internal/automation", bytes.NewReader(jsonValue(payload)))
	r.Header.Set("Content-Type", "application/json")
	return r
}
func automationResponse(rec *httptest.ResponseRecorder) (map[string]any, error) {
	value := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &value)
	if rec.Code >= 400 {
		message := str(value, "error")
		if message == "" {
			message = fmt.Sprintf("자동화 API HTTP %d", rec.Code)
		}
		if rec.Code < 500 && rec.Code != 409 {
			return nil, jobPermanent(message)
		}
		return nil, errors.New(message)
	}
	return value, nil
}
func automationTemplate(text string, event Event) string {
	return strings.NewReplacer("{{title}}", str(event.After, "title"), "{{event}}", event.Type, "{{document_id}}", event.ResourceID).Replace(text)
}
func (s *Server) executeAutomation(ctx context.Context, j Job) (map[string]any, error) {
	var ownerID, wid string
	var enabled bool
	var raw []byte
	var revision int
	if s.DB.QueryRow(ctx, `SELECT owner_id::text,workspace_id::text,enabled,actions,revision FROM automation_rules WHERE id=$1`, str(j.Payload, "automation_id")).Scan(&ownerID, &wid, &enabled, &raw, &revision) != nil || !enabled || wid != j.WorkspaceID || ownerID != j.OwnerID || revision != number(j.Payload, "revision", 0) {
		return nil, jobPermanent("자동화가 비활성화되었거나 실행 계정이 변경되었습니다")
	}
	owner, err := s.workerPrincipal(ctx, ownerID, "", wid)
	if err != nil {
		return nil, err
	}
	actor, err := s.workerPrincipal(ctx, j.ActorID, j.TokenID, wid)
	if err != nil {
		return nil, err
	}
	event, err := s.eventByID(ctx, j.EventID)
	if err != nil {
		return nil, err
	}
	if !s.automationSourceAccess(ctx, owner, event, false) || !s.automationSourceAccess(ctx, actor, event, false) {
		return nil, jobPermanent("자동화 원본에 대한 현재 접근 권한이 없습니다")
	}
	var conditions map[string]any
	var conditionData []byte
	if err = s.DB.QueryRow(ctx, "SELECT conditions FROM automation_rules WHERE id=$1", str(j.Payload, "automation_id")).Scan(&conditionData); err != nil {
		return nil, err
	}
	if json.Unmarshal(conditionData, &conditions) != nil || !s.automationRuleConditions(ctx, conditions, event) {
		return nil, jobPermanent("현재 원본 데이터가 자동화 조건과 일치하지 않습니다")
	}
	var actions []AutomationAction
	if json.Unmarshal(raw, &actions) != nil {
		return nil, jobPermanent("자동화 작업 설정이 잘못되었습니다")
	}
	completed := 0
	for index, action := range actions {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var unchanged bool
		if s.DB.QueryRow(ctx, "SELECT enabled AND revision=$2 AND owner_id=$3 FROM automation_rules WHERE id=$1", str(j.Payload, "automation_id"), revision, ownerID).Scan(&unchanged) != nil || !unchanged {
			return nil, jobPermanent("실행 도중 자동화 설정이 변경되었습니다")
		}
		var exists bool
		if err = s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM automation_effects WHERE job_id=$1 AND action_index=$2)", j.ID, index).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			completed++
			continue
		}
		// Re-evaluate both users and the source for every action, not only on enqueue.
		owner, err = s.workerPrincipal(ctx, ownerID, "", wid)
		if err != nil {
			return nil, err
		}
		actor, err = s.workerPrincipal(ctx, j.ActorID, j.TokenID, wid)
		if err != nil {
			return nil, err
		}
		if !s.automationSourceAccess(ctx, owner, event, false) || !s.automationSourceAccess(ctx, actor, event, false) {
			return nil, jobPermanent("실행 도중 원본 접근 권한이 회수되었습니다")
		}
		actionCtx := context.WithValue(ctx, automationEffectKey{}, automationEffectContext{j.ID, index})
		if _, err = s.executeAutomationAction(actionCtx, j, event, owner, actor, action); err != nil {
			return map[string]any{"completed_actions": completed, "failed_action": index + 1}, err
		}
		completed++
	}
	return map[string]any{"completed_actions": completed}, nil
}
func (s *Server) executeAutomationAction(ctx context.Context, j Job, event Event, owner, actor *Principal, a AutomationAction) (map[string]any, error) {
	docID := a.DocumentID
	if docID == "" && event.ResourceType == "document" {
		docID = event.ResourceID
	}
	requireDocument := func(write bool) bool {
		var wid string
		return validID(docID) && s.DB.QueryRow(ctx, "SELECT workspace_id::text FROM documents WHERE id=$1 AND deleted_at IS NULL", docID).Scan(&wid) == nil && wid == j.WorkspaceID && s.canDocument(ctx, owner, docID, write) && s.canDocument(ctx, actor, docID, write)
	}
	commitEffect := func(effect func(pgx.Tx) (map[string]any, error)) (map[string]any, error) {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback(ctx)
		result, e := effect(tx)
		if e == nil {
			e = s.recordAutomationEffect(ctx, tx, result)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		return result, e
	}
	switch a.Type {
	case "notification":
		uid := a.UserID
		if uid == "" {
			uid = actor.ID
		}
		recipient, e := s.workerPrincipal(ctx, uid, "", j.WorkspaceID)
		if e != nil || !s.automationSourceAccess(ctx, recipient, event, false) {
			return nil, jobPermanent("알림 수신자의 원본 접근 권한이 없습니다")
		}
		title := automationTemplate(a.Title, event)
		if title == "" {
			title = "자동화: " + event.Type
		}
		if len(title) > 500 {
			return nil, jobPermanent("알림 제목은 500바이트 이하여야 합니다")
		}
		return commitEffect(func(tx pgx.Tx) (map[string]any, error) {
			id := newID()
			source := ""
			if event.ResourceType == "document" {
				source = event.ResourceID
			}
			_, e := tx.Exec(ctx, "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,$3,NULLIF($4,'')::uuid)", id, uid, title, source)
			return map[string]any{"notification_id": id}, e
		})
	case "create_document":
		if !s.canWorkspace(ctx, owner, j.WorkspaceID, true) || !s.canWorkspace(ctx, actor, j.WorkspaceID, true) || !jobScope(actor, "document:write") {
			return nil, jobPermanent("자동화 문서 작성 권한이 없습니다")
		}
		// Private output prevents widening access to a private event's metadata.
		payload := map[string]any{"workspace_id": j.WorkspaceID, "title": automationTemplate(a.Title, event), "markdown": automationTemplate(a.Markdown, event), "visibility": "private"}
		rec := httptest.NewRecorder()
		s.createDocument(rec, automationRequest(ctx, actor, "POST", payload))
		return automationResponse(rec)
	case "update_document", "ai":
		if !requireDocument(true) || !jobScope(actor, "document:write") {
			return nil, jobPermanent("자동화 대상 문서 수정 권한이 없습니다")
		}
		r := automationRequest(ctx, actor, "PUT", nil)
		old, e := s.document(r, docID)
		if e != nil {
			return nil, e
		}
		payload := map[string]any{"version": old["version"]}
		if a.Type == "ai" {
			if !jobScope(actor, "ai:execute") {
				return nil, jobPermanent("API 키에 AI 실행 범위가 없습니다")
			}
			text, e := s.automationAI(ctx, actor, docID, automationTemplate(a.Prompt, event))
			if e != nil {
				return nil, e
			}
			payload["markdown"] = str(old, "markdown") + "\n\n" + text
		} else {
			if a.Title != "" {
				payload["title"] = automationTemplate(a.Title, event)
			}
			if a.Markdown != "" {
				payload["markdown"] = automationTemplate(a.Markdown, event)
			}
			if a.Status != "" {
				payload["status"] = a.Status
			}
		}
		// Recheck after potentially long AI completion before updating.
		if !requireDocument(true) {
			return nil, jobPermanent("자동화 실행 중 대상 권한이 회수되었습니다")
		}
		rec := httptest.NewRecorder()
		s.saveDocument(rec, r, docID, payload)
		return automationResponse(rec)
	case "update_property":
		databaseID := a.DatabaseID
		if databaseID == "" {
			databaseID = str(event.After, "database_id")
		}
		rowID := a.RowID
		if rowID == "" {
			rowID = str(event.After, "row_id")
		}
		request := automationRequest(ctx, actor, "PUT", map[string]any{"values": a.Values})
		other := automationRequest(ctx, owner, "GET", nil)
		var wid string
		if !validID(databaseID) || !validID(rowID) || s.DB.QueryRow(ctx, "SELECT workspace_id::text FROM databases WHERE id=$1", databaseID).Scan(&wid) != nil || wid != j.WorkspaceID || !s.canDatabase(request, databaseID, true) || !s.canDatabase(other, databaseID, true) || !jobScope(actor, "database:write") {
			return nil, jobPermanent("자동화 데이터베이스 수정 권한이 없습니다")
		}
		request.SetPathValue("id", databaseID)
		request.SetPathValue("rowId", rowID)
		rec := httptest.NewRecorder()
		s.updateRow(rec, request)
		return automationResponse(rec)
	case "webhook":
		if !s.automationManager(ctx, owner, j.WorkspaceID) {
			return nil, jobPermanent("Webhook 자동화 실행 계정의 관리 권한이 없습니다")
		}
		return commitEffect(func(tx pgx.Tx) (map[string]any, error) {
			var hookOwner string
			var attempts, timeout int
			if tx.QueryRow(ctx, `SELECT owner_id::text,max_attempts,timeout_seconds FROM automation_webhooks WHERE id=$1 AND workspace_id=$2 AND enabled`, a.WebhookID, j.WorkspaceID).Scan(&hookOwner, &attempts, &timeout) != nil {
				return nil, jobPermanent("사용 가능한 Webhook이 없습니다")
			}
			id := newID()
			_, e := tx.Exec(ctx, `INSERT INTO automation_jobs(id,kind,workspace_id,owner_id,actor_id,token_id,event_id,resource_id,automation_id,depth,target_id,payload,max_attempts,timeout_seconds,actor_constraints) VALUES($1,'webhook.deliver',$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(event_id,kind,target_id) DO NOTHING`, id, j.WorkspaceID, hookOwner, j.ActorID, j.TokenID, j.EventID, j.ResourceID, j.AutomationID, j.Depth, a.WebhookID, jsonValue(map[string]any{"webhook_id": a.WebhookID}), attempts, timeout+5, jsonValue(j.Constraints))
			return map[string]any{"webhook_id": a.WebhookID}, e
		})
	default:
		return nil, jobPermanent("지원하지 않는 자동화 실행 작업입니다")
	}
}

// AI still uses the normal streaming proxy. Its answer is appended only after a
// complete stream and optimistic-version validation; source is one ACL-checked
// document, never unrestricted workspace RAG under a service account.
func (s *Server) automationAI(ctx context.Context, p *Principal, docID, prompt string) (string, error) {
	rec := &automationStreamRecorder{ResponseRecorder: httptest.NewRecorder()}
	s.aiChat(rec, automationRequest(ctx, p, "POST", map[string]any{"document_id": docID, "prompt": prompt}))
	if rec.overflow {
		return "", jobPermanent("AI 자동화 응답은 4MB 이하여야 합니다")
	}
	if rec.Code >= 400 {
		_, e := automationResponse(rec.ResponseRecorder)
		return "", e
	}
	if rec.Body.Len() > 4<<20 {
		return "", jobPermanent("AI 자동화 결과는 4MB 이하여야 합니다")
	}
	var output strings.Builder
	done := false
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			done = true
			continue
		}
		var chunk map[string]any
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue
		}
		if str(chunk, "error") != "" {
			return "", errors.New("AI 자동화 스트림이 중단되었습니다")
		}
		output.WriteString(str(chunk, "text"))
	}
	if !done || output.Len() == 0 {
		return "", errors.New("AI 자동화 응답이 완료되지 않았습니다")
	}
	return output.String(), nil
}

type automationStreamRecorder struct {
	*httptest.ResponseRecorder
	overflow bool
}

func (w *automationStreamRecorder) Write(body []byte) (int, error) {
	if w.Body.Len()+len(body) > 4<<20 {
		w.overflow = true
		return 0, errors.New("AI 자동화 응답 크기 제한")
	}
	return w.ResponseRecorder.Write(body)
}
