package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Event is inserted in the SAME transaction as its originating mutation.
// ActorID remains the original triggering actor across automation chains.
type Event struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	WorkspaceID  string           `json:"workspace_id"`
	ResourceID   string           `json:"resource_id"`
	ResourceType string           `json:"resource_type"`
	ActorID      string           `json:"actor_id"`
	TokenID      string           `json:"-"`
	Constraints  actorConstraints `json:"-"`
	Depth        int              `json:"depth"`
	JobID        string           `json:"job_id,omitempty"`
	AutomationID string           `json:"automation_id,omitempty"`
	Before       map[string]any   `json:"before,omitempty"`
	After        map[string]any   `json:"after,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

var automationEventTypes = []string{"document.created", "document.updated", "document.deleted", "document.status_changed", "comment.created", "task.completed", "database.created", "database.updated", "database.deleted", "database.row.created", "database.row.updated", "database.row.deleted", "date.reached"}

func eventMetadata(input map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range []string{"id", "title", "status", "version", "tags", "visibility", "owner_id", "document_id", "database_id", "row_id", "done", "line", "space_id"} {
		if value, ok := input[key]; ok {
			out[key] = value
		}
	}
	return out
}
func (s *Server) enqueueEvent(ctx context.Context, tx pgx.Tx, event Event) error {
	if tx == nil {
		return errors.New("이벤트는 원본 변경과 같은 트랜잭션에서 기록해야 합니다")
	}
	if !slices.Contains(automationEventTypes, event.Type) {
		return errors.New("지원하지 않는 자동화 이벤트입니다")
	}
	if event.ID == "" {
		event.ID = newID()
	}
	if event.ResourceType == "" {
		event.ResourceType = "document"
		if strings.HasPrefix(event.Type, "database.") {
			event.ResourceType = "database"
		}
		if event.Type == "date.reached" {
			event.ResourceType = "schedule"
		}
	}
	if p, _ := ctx.Value(principalKey).(*Principal); p != nil {
		if event.ActorID == "" {
			event.ActorID = p.ID
		}
		if event.TokenID == "" {
			event.TokenID = p.TokenID
		}
		event.Constraints = constraintsFor(p)
	}
	if inherited, ok := ctx.Value(jobContextKey{}).(jobContext); ok {
		event.ActorID = inherited.ActorID
		event.TokenID = inherited.TokenID
		event.Depth = inherited.Depth + 1
		event.JobID = inherited.JobID
		event.AutomationID = inherited.AutomationID
		event.Constraints = inherited.Constraints
	}
	if event.Depth > 5 {
		return nil
	}
	if !validID(event.WorkspaceID) || !validID(event.ActorID) {
		return errors.New("이벤트 워크스페이스와 실행 사용자를 확인하세요")
	}
	event.Before = eventMetadata(event.Before)
	event.After = eventMetadata(event.After)
	var protectionError error
	event.Before, protectionError = s.protectEventMetadataTx(ctx, tx, event.Before)
	if protectionError != nil {
		return protectionError
	}
	event.After, protectionError = s.protectEventMetadataTx(ctx, tx, event.After)
	if protectionError != nil {
		return protectionError
	}
	event.CreatedAt = time.Now().UTC()
	event.Constraints.TokenBound = event.Constraints.TokenBound || event.TokenID != ""
	tag, err := tx.Exec(ctx, `INSERT INTO automation_events(id,type,workspace_id,actor_id,token_id,resource_id,resource_type,depth,job_id,automation_id,payload,actor_constraints) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(id) DO NOTHING`, event.ID, event.Type, event.WorkspaceID, event.ActorID, event.TokenID, event.ResourceID, event.ResourceType, event.Depth, event.JobID, event.AutomationID, jsonValue(event), jsonValue(event.Constraints))
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO automation_jobs(id,kind,workspace_id,owner_id,actor_id,token_id,event_id,resource_id,depth,payload,target_id,actor_constraints) VALUES($1,'event.dispatch',$2,$3,$3,NULLIF($4,'')::uuid,$5,$6,$7,'{}','dispatch',$8)`, newID(), event.WorkspaceID, event.ActorID, event.TokenID, event.ID, event.ResourceID, event.Depth, jsonValue(event.Constraints))
	return err
}
func (s *Server) automationManager(ctx context.Context, p *Principal, wid string) bool {
	if !s.canWorkspace(ctx, p, wid, true) {
		return false
	}
	var role string
	_ = s.DB.QueryRow(ctx, "SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2", wid, p.ID).Scan(&role)
	return role == "owner" || role == "admin"
}
func (s *Server) automationSourceAccess(ctx context.Context, p *Principal, e Event, write bool) bool {
	if !s.canWorkspace(ctx, p, e.WorkspaceID, write) {
		return false
	}
	if e.ResourceID == "" || e.ResourceType == "schedule" {
		return true
	}
	if e.ResourceType == "database" {
		if !jobScope(p, "database:read") {
			return false
		}
		if e.Type == "database.deleted" {
			return s.canSpace(ctx, p, e.WorkspaceID, str(e.Before, "space_id"), write)
		}
		databaseID := str(e.After, "database_id")
		if databaseID == "" {
			databaseID = str(e.Before, "database_id")
		}
		if databaseID == "" {
			databaseID = e.ResourceID
		}
		r, _ := http.NewRequestWithContext(context.WithValue(ctx, principalKey, p), "GET", "/", nil)
		return s.canDatabase(r, databaseID, write)
	}
	// Deleted documents remain in the trash, where canDocument can still evaluate ownership.
	return jobScope(p, "document:read") && validID(e.ResourceID) && s.canDocument(ctx, p, e.ResourceID, write)
}
func (s *Server) eventByID(ctx context.Context, id string) (Event, error) {
	var event Event
	var raw, constraints []byte
	err := s.DB.QueryRow(ctx, "SELECT payload,coalesce(token_id::text,''),actor_constraints FROM automation_events WHERE id=$1", id).Scan(&raw, &event.TokenID, &constraints)
	if err == nil {
		err = json.Unmarshal(raw, &event)
	}
	if err == nil {
		err = json.Unmarshal(constraints, &event.Constraints)
	}
	return event, err
}
func (s *Server) dispatchEvent(ctx context.Context, j Job) (map[string]any, error) {
	event, err := s.eventByID(ctx, j.EventID)
	if err != nil {
		return nil, err
	}
	actor, err := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if !s.automationSourceAccess(ctx, actor, event, false) {
		return nil, jobPermanent("이벤트 원본에 대한 현재 접근 권한이 없습니다")
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	queued := 0
	hooks, err := tx.Query(ctx, `SELECT id::text,owner_id::text,max_attempts,timeout_seconds FROM automation_webhooks WHERE workspace_id=$1 AND enabled AND events ? $2`, event.WorkspaceID, event.Type)
	if err != nil {
		return nil, err
	}
	type target struct {
		id, owner         string
		attempts, timeout int
	}
	targets := []target{}
	for hooks.Next() {
		var x target
		if err = hooks.Scan(&x.id, &x.owner, &x.attempts, &x.timeout); err != nil {
			hooks.Close()
			return nil, err
		}
		targets = append(targets, x)
	}
	hooks.Close()
	if hooks.Err() != nil {
		return nil, hooks.Err()
	}
	for _, hook := range targets {
		owner, e := s.workerPrincipal(ctx, hook.owner, "", j.WorkspaceID)
		if e != nil || !s.automationSourceAccess(ctx, owner, event, false) {
			continue
		}
		tag, e := tx.Exec(ctx, `INSERT INTO automation_jobs(id,kind,workspace_id,owner_id,actor_id,token_id,event_id,resource_id,depth,target_id,payload,max_attempts,timeout_seconds,actor_constraints) VALUES($1,'webhook.deliver',$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(event_id,kind,target_id) DO NOTHING`, newID(), j.WorkspaceID, hook.owner, j.ActorID, j.TokenID, j.EventID, j.ResourceID, j.Depth, hook.id, jsonValue(map[string]any{"webhook_id": hook.id}), hook.attempts, hook.timeout+5, jsonValue(j.Constraints))
		if e != nil {
			return nil, e
		}
		queued += int(tag.RowsAffected())
	}
	rules, err := tx.Query(ctx, `SELECT id::text,owner_id::text,conditions,revision FROM automation_rules WHERE workspace_id=$1 AND enabled AND trigger=$2 AND id::text<>$3`, j.WorkspaceID, event.Type, event.AutomationID)
	if err != nil {
		return nil, err
	}
	type ruleTarget struct {
		id, owner  string
		conditions map[string]any
		revision   int
	}
	rt := []ruleTarget{}
	for rules.Next() {
		var x ruleTarget
		var data []byte
		if err = rules.Scan(&x.id, &x.owner, &data, &x.revision); err != nil {
			rules.Close()
			return nil, err
		}
		_ = json.Unmarshal(data, &x.conditions)
		rt = append(rt, x)
	}
	rules.Close()
	if rules.Err() != nil {
		return nil, rules.Err()
	}
	for _, rule := range rt {
		if event.Type == "date.reached" && event.ResourceID != rule.id {
			continue
		}
		if !s.automationRuleConditions(ctx, rule.conditions, event) {
			continue
		}
		owner, e := s.workerPrincipal(ctx, rule.owner, "", j.WorkspaceID)
		if e != nil || !s.automationSourceAccess(ctx, owner, event, false) {
			continue
		}
		tag, e := tx.Exec(ctx, `INSERT INTO automation_jobs(id,kind,workspace_id,owner_id,actor_id,token_id,event_id,resource_id,automation_id,depth,target_id,payload,timeout_seconds,actor_constraints) VALUES($1,'automation.execute',$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9,$8,$10,300,$11) ON CONFLICT(event_id,kind,target_id) DO NOTHING`, newID(), j.WorkspaceID, rule.owner, j.ActorID, j.TokenID, j.EventID, j.ResourceID, rule.id, j.Depth, jsonValue(map[string]any{"automation_id": rule.id, "revision": rule.revision}), jsonValue(j.Constraints))
		if e != nil {
			return nil, e
		}
		queued += int(tag.RowsAffected())
	}
	return map[string]any{"queued": queued}, tx.Commit(ctx)
}
func automationConditions(c map[string]any, e Event) bool {
	if id := str(c, "document_id"); id != "" && id != e.ResourceID {
		return false
	}
	if status := str(c, "status"); status != "" && status != str(e.After, "status") {
		return false
	}
	if tag := str(c, "tag"); tag != "" {
		found := false
		switch list := e.After["tags"].(type) {
		case []any:
			for _, x := range list {
				if x == tag {
					found = true
				}
			}
		case []string:
			found = slices.Contains(list, tag)
		}
		if !found {
			return false
		}
	}
	return true
}

// Database conditions are evaluated against the currently readable row; values
// are not copied into the durable event or outbound webhook payload.
func (s *Server) automationRuleConditions(ctx context.Context, c map[string]any, e Event) bool {
	if !automationConditions(c, e) {
		return false
	}
	property := str(c, "property_id")
	if property == "" {
		return true
	}
	if e.ResourceType != "database" || !validID(str(e.After, "row_id")) {
		return false
	}
	var value []byte
	if s.DB.QueryRow(ctx, "SELECT values->$3 FROM database_rows WHERE database_id=$1 AND id=$2", str(e.After, "database_id"), str(e.After, "row_id"), property).Scan(&value) != nil {
		return false
	}
	var actual any
	if json.Unmarshal(value, &actual) != nil {
		return false
	}
	return string(jsonValue(actual)) == string(jsonValue(c["equals"]))
}
func (s *Server) scheduleAutomations(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scheduleDueAutomations(ctx)
		}
	}
}
func (s *Server) scheduleDueAutomations(ctx context.Context) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	var paused bool
	if tx.QueryRow(ctx, "SELECT paused FROM job_settings WHERE id=1").Scan(&paused) != nil || paused {
		return
	}
	var id, wid, owner string
	var due time.Time
	var interval int
	err = tx.QueryRow(ctx, `SELECT id::text,workspace_id::text,owner_id::text,schedule_at,interval_minutes FROM automation_rules WHERE enabled AND trigger='date.reached' AND schedule_at<=now() ORDER BY schedule_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&id, &wid, &owner, &due, &interval)
	if err == nil {
		event := Event{Type: "date.reached", WorkspaceID: wid, ActorID: owner, ResourceType: "schedule", ResourceID: id, After: map[string]any{"id": id}}
		if err = s.enqueueEvent(ctx, tx, event); err != nil {
			return
		}
		_, err = tx.Exec(ctx, `UPDATE automation_rules SET schedule_at=CASE WHEN interval_minutes=0 THEN NULL ELSE now()+make_interval(mins=>interval_minutes) END,updated_at=now() WHERE id=$1`, id)
		if err != nil {
			return
		}
	}
	_, err = tx.Exec(ctx, `UPDATE automation_jobs SET status=CASE WHEN cancel_requested THEN 'cancelled' ELSE 'failed' END,last_error=CASE WHEN cancel_requested THEN '작업 취소' ELSE '최대 시도 횟수를 초과했습니다' END,finished_at=now(),updated_at=now() WHERE (status='pending' AND cancel_requested) OR (status='running' AND lease_until<now() AND (attempts>=max_attempts OR cancel_requested))`)
	if err != nil {
		return
	}
	_, err = tx.Exec(ctx, `DELETE FROM automation_jobs WHERE id IN (SELECT j.id FROM automation_jobs j,job_settings c WHERE c.id=1 AND j.status IN ('succeeded','failed','cancelled') AND j.finished_at<now()-make_interval(days=>c.retention_days) LIMIT 100)`)
	if err == nil {
		_, err = tx.Exec(ctx, `DELETE FROM automation_events WHERE id IN (SELECT e.id FROM automation_events e,job_settings c WHERE c.id=1 AND e.created_at<now()-make_interval(days=>c.retention_days) AND NOT EXISTS(SELECT 1 FROM automation_jobs j WHERE j.event_id=e.id) LIMIT 100)`)
	}
	if err == nil {
		_ = tx.Commit(ctx)
	}
}

func validWebhookURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil && u.Fragment == "" && len(raw) <= 2000
}

type webhookInput struct {
	WorkspaceID    string   `json:"workspace_id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Secret         string   `json:"secret"`
	Events         []string `json:"events"`
	Enabled        bool     `json:"enabled"`
	MaxAttempts    int      `json:"max_attempts"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

func validateWebhook(in webhookInput, creating bool) error {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 120 || !validWebhookURL(in.URL) {
		return errors.New("이름과 HTTP(S) Webhook 주소를 확인하세요")
	}
	if len(in.Events) == 0 {
		return errors.New("이벤트를 하나 이상 선택하세요")
	}
	for _, e := range in.Events {
		if !slices.Contains(automationEventTypes, e) {
			return errors.New("지원하지 않는 이벤트입니다")
		}
	}
	if in.MaxAttempts < 1 || in.MaxAttempts > 10 || in.TimeoutSeconds < 1 || in.TimeoutSeconds > 120 {
		return errors.New("재시도는 1~10회, 제한시간은 1~120초입니다")
	}
	if creating && len(in.Secret) < 16 || in.Secret != "" && len(in.Secret) < 16 || len(in.Secret) > 1000 {
		return errors.New("서명 비밀은 16~1000자로 입력하세요")
	}
	return nil
}

const webhookJSON = `jsonb_build_object('id',h.id,'workspace_id',h.workspace_id,'owner_id',h.owner_id,'name',h.name,'url',h.url,'secret_configured',h.secret_ciphertext<>'','events',h.events,'enabled',h.enabled,'max_attempts',h.max_attempts,'timeout_seconds',h.timeout_seconds,'created_at',h.created_at,'updated_at',h.updated_at)`

func (s *Server) registerAutomation() {
	s.RegisterJobHandler("event.dispatch", s.dispatchEvent)
	s.RegisterJobHandler("webhook.deliver", s.deliverWebhook)
	s.RegisterJobHandler("automation.execute", s.executeAutomation)
	s.handle("GET /api/v1/webhooks", s.listWebhooks)
	s.handle("POST /api/v1/webhooks", s.saveWebhook)
	s.handle("PUT /api/v1/webhooks/{id}", s.saveWebhook)
	s.handle("DELETE /api/v1/webhooks/{id}", s.deleteWebhook)
	s.handle("GET /api/v1/webhooks/{id}/deliveries", s.webhookDeliveries)
	s.handle("POST /api/v1/webhooks/{id}/test", s.testWebhook)
	s.handle("GET /api/v1/automations", s.listAutomations)
	s.handle("POST /api/v1/automations", s.saveAutomation)
	s.handle("PUT /api/v1/automations/{id}", s.saveAutomation)
	s.handle("DELETE /api/v1/automations/{id}", s.deleteAutomation)
}
func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) || !s.automationManager(r.Context(), current(r), wid) {
		apiError(w, 403, "워크스페이스 관리자가 필요합니다")
		return
	}
	v, e := s.rows(r.Context(), "SELECT "+webhookJSON+" FROM automation_webhooks h WHERE workspace_id=$1 ORDER BY created_at DESC", wid)
	respond(w, v, e)
}
func (s *Server) hookWorkspace(r *http.Request) (string, error) {
	var wid string
	e := s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM automation_webhooks WHERE id=$1", r.PathValue("id")).Scan(&wid)
	return wid, e
}
func (s *Server) saveWebhook(w http.ResponseWriter, r *http.Request) {
	var in webhookInput
	if decode(r, &in) != nil {
		apiError(w, 400, "Webhook 설정을 확인하세요")
		return
	}
	creating := r.Method == "POST"
	id := r.PathValue("id")
	if creating {
		id = newID()
	} else {
		wid, e := s.hookWorkspace(r)
		if e != nil {
			apiError(w, 404, "Webhook을 찾을 수 없습니다")
			return
		}
		in.WorkspaceID = wid
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted || !s.automationManager(r.Context(), current(r), in.WorkspaceID) {
		apiError(w, 403, "사용자 세션의 워크스페이스 관리자가 필요합니다")
		return
	}
	if e := validateWebhook(in, creating); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	var secret string
	var e error
	if in.Secret != "" {
		secret, e = s.encrypt(in.Secret)
		if e != nil {
			respond(w, nil, e)
			return
		}
	}
	if creating {
		_, e = s.DB.Exec(r.Context(), `INSERT INTO automation_webhooks(id,workspace_id,owner_id,name,url,secret_ciphertext,events,enabled,max_attempts,timeout_seconds) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, in.WorkspaceID, current(r).ID, strings.TrimSpace(in.Name), in.URL, secret, jsonValue(in.Events), in.Enabled, in.MaxAttempts, in.TimeoutSeconds)
	} else {
		_, e = s.DB.Exec(r.Context(), `UPDATE automation_webhooks SET name=$2,url=$3,secret_ciphertext=CASE WHEN $4='' THEN secret_ciphertext ELSE $4 END,events=$5,enabled=$6,max_attempts=$7,timeout_seconds=$8,updated_at=now() WHERE id=$1`, id, strings.TrimSpace(in.Name), in.URL, secret, jsonValue(in.Events), in.Enabled, in.MaxAttempts, in.TimeoutSeconds)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "WEBHOOK_CONFIGURE", id, map[string]any{"enabled": in.Enabled, "events": in.Events})
	v, e := s.one(r.Context(), "SELECT "+webhookJSON+" FROM automation_webhooks h WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	wid, e := s.hookWorkspace(r)
	if e != nil || current(r).TokenID != "" || current(r).ScopeRestricted || !s.automationManager(r.Context(), current(r), wid) {
		apiError(w, 403, "Webhook 관리 권한이 없습니다")
		return
	}
	_, e = s.DB.Exec(r.Context(), "DELETE FROM automation_webhooks WHERE id=$1", r.PathValue("id"))
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) webhookDeliveries(w http.ResponseWriter, r *http.Request) {
	wid, e := s.hookWorkspace(r)
	if e != nil || !s.automationManager(r.Context(), current(r), wid) {
		apiError(w, 403, "Webhook 관리 권한이 없습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+jobSelect+" FROM automation_jobs WHERE kind='webhook.deliver' AND target_id=$1 ORDER BY created_at DESC LIMIT 100", r.PathValue("id"))
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer rows.Close()
	out := []Job{}
	for rows.Next() {
		j, e := scanJob(rows)
		if e != nil {
			respond(w, nil, e)
			return
		}
		if s.jobVisible(r, j) {
			out = append(out, j)
		}
	}
	respond(w, out, rows.Err())
}
func (s *Server) testWebhook(w http.ResponseWriter, r *http.Request) {
	wid, e := s.hookWorkspace(r)
	if e != nil || current(r).TokenID != "" || current(r).ScopeRestricted || !s.automationManager(r.Context(), current(r), wid) {
		apiError(w, 403, "Webhook 관리 권한이 없습니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	id, e := s.EnqueueJob(r.Context(), tx, "webhook.deliver", current(r).ID, wid, map[string]any{"webhook_id": r.PathValue("id"), "test": true})
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE automation_jobs SET target_id=h.id::text,max_attempts=h.max_attempts,timeout_seconds=h.timeout_seconds+5 FROM automation_webhooks h WHERE automation_jobs.id=$1 AND h.id=$2`, id, r.PathValue("id"))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]string{"job_id": id}, e)
}

type AutomationAction struct {
	Type       string         `json:"type"`
	DocumentID string         `json:"document_id,omitempty"`
	DatabaseID string         `json:"database_id,omitempty"`
	RowID      string         `json:"row_id,omitempty"`
	WebhookID  string         `json:"webhook_id,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	Title      string         `json:"title,omitempty"`
	Markdown   string         `json:"markdown,omitempty"`
	Prompt     string         `json:"prompt,omitempty"`
	Values     map[string]any `json:"values,omitempty"`
	Status     string         `json:"status,omitempty"`
}
type automationInput struct {
	WorkspaceID     string             `json:"workspace_id"`
	OwnerID         string             `json:"owner_id"`
	Name            string             `json:"name"`
	Enabled         bool               `json:"enabled"`
	Trigger         string             `json:"trigger"`
	Conditions      map[string]any     `json:"conditions"`
	Actions         []AutomationAction `json:"actions"`
	ScheduleAt      *time.Time         `json:"schedule_at"`
	IntervalMinutes int                `json:"interval_minutes"`
}

func (s *Server) listAutomations(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) || !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 권한이 없습니다")
		return
	}
	v, e := s.rows(r.Context(), `SELECT to_jsonb(a)-'token_id' FROM automation_rules a WHERE workspace_id=$1 AND (owner_id=$2 OR $3) ORDER BY created_at DESC`, wid, current(r).ID, s.automationManager(r.Context(), current(r), wid))
	respond(w, v, e)
}
func (s *Server) saveAutomation(w http.ResponseWriter, r *http.Request) {
	var in automationInput
	if decode(r, &in) != nil {
		apiError(w, 400, "자동화 설정을 확인하세요")
		return
	}
	p := current(r)
	id := r.PathValue("id")
	creating := id == ""
	if !creating {
		var owner, wid string
		if s.DB.QueryRow(r.Context(), "SELECT owner_id::text,workspace_id::text FROM automation_rules WHERE id=$1", id).Scan(&owner, &wid) != nil {
			apiError(w, 404, "자동화를 찾을 수 없습니다")
			return
		}
		in.WorkspaceID = wid
		if p.ID != owner && !s.automationManager(r.Context(), p, wid) {
			apiError(w, 403, "자동화 관리 권한이 없습니다")
			return
		}
		if in.OwnerID == "" {
			in.OwnerID = owner
		}
	}
	if p.TokenID != "" || p.ScopeRestricted || !s.canWorkspace(r.Context(), p, in.WorkspaceID, true) {
		apiError(w, 403, "자동화 설정에는 편집 권한이 있는 사용자 세션이 필요합니다")
		return
	}
	if in.OwnerID == "" {
		in.OwnerID = p.ID
	}
	if in.OwnerID != p.ID && !s.automationManager(r.Context(), p, in.WorkspaceID) {
		apiError(w, 403, "다른 실행 계정은 워크스페이스 관리자가 지정합니다")
		return
	}
	owner, e := s.workerPrincipal(r.Context(), in.OwnerID, "", in.WorkspaceID)
	if e != nil || !s.canWorkspace(r.Context(), owner, in.WorkspaceID, true) {
		apiError(w, 400, "실행 계정의 워크스페이스 편집 권한을 확인하세요")
		return
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 120 || !slices.Contains(automationEventTypes, in.Trigger) || len(in.Actions) < 1 || len(in.Actions) > 10 || in.IntervalMinutes < 0 || in.IntervalMinutes > 525600 {
		apiError(w, 400, "이름·이벤트·실행 작업을 확인하세요 (최대 10개)")
		return
	}
	if in.Trigger == "date.reached" && in.ScheduleAt == nil {
		apiError(w, 400, "날짜 트리거의 실행 시각을 입력하세요")
		return
	}
	if len(jsonValue(in)) > 128<<10 {
		apiError(w, 400, "자동화 설정은 128KB 이하여야 합니다")
		return
	}
	for key, value := range in.Conditions {
		if key == "equals" {
			switch value.(type) {
			case string, bool, float64:
				continue
			default:
				apiError(w, 400, "속성 비교값은 문자열·숫자·참/거짓이어야 합니다")
				return
			}
		}
		if !slices.Contains([]string{"document_id", "status", "tag", "property_id"}, key) {
			apiError(w, 400, "지원하는 조건은 문서·상태·태그·데이터베이스 속성입니다")
			return
		}
		if _, ok := value.(string); !ok {
			apiError(w, 400, "자동화 조건은 문자열이어야 합니다")
			return
		}
	}
	for _, action := range in.Actions {
		if !slices.Contains([]string{"notification", "create_document", "update_document", "update_property", "webhook", "ai"}, action.Type) {
			apiError(w, 400, "지원하지 않는 실행 작업입니다")
			return
		}
		for _, id := range []string{action.DocumentID, action.DatabaseID, action.RowID, action.UserID} {
			if id != "" && !validID(id) {
				apiError(w, 400, "자동화 대상 ID를 확인하세요")
				return
			}
		}
		if action.Type == "create_document" && strings.TrimSpace(action.Title) == "" || action.Type == "ai" && strings.TrimSpace(action.Prompt) == "" || action.Type == "update_property" && len(action.Values) == 0 {
			apiError(w, 400, "문서 제목·AI 지시문·속성 값 등 작업 필수값을 입력하세요")
			return
		}
		if action.Status != "" && !slices.Contains([]string{"draft", "published", "stale", "archived"}, action.Status) {
			apiError(w, 400, "검토·승인 상태는 승인 절차를 통해 변경합니다")
			return
		}
		if action.Type == "webhook" {
			var wid string
			if !s.automationManager(r.Context(), p, in.WorkspaceID) || s.DB.QueryRow(r.Context(), "SELECT workspace_id::text FROM automation_webhooks WHERE id=$1", action.WebhookID).Scan(&wid) != nil || wid != in.WorkspaceID {
				apiError(w, 403, "Webhook 작업은 같은 공간의 관리자가 설정합니다")
				return
			}
		}
	}
	if creating {
		id = newID()
		_, e = s.DB.Exec(r.Context(), `INSERT INTO automation_rules(id,workspace_id,owner_id,name,enabled,trigger,conditions,actions,schedule_at,interval_minutes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, in.WorkspaceID, in.OwnerID, strings.TrimSpace(in.Name), in.Enabled, in.Trigger, jsonValue(in.Conditions), jsonValue(in.Actions), in.ScheduleAt, in.IntervalMinutes)
	} else {
		_, e = s.DB.Exec(r.Context(), `UPDATE automation_rules SET owner_id=$2,name=$3,enabled=$4,trigger=$5,conditions=$6,actions=$7,schedule_at=$8,interval_minutes=$9,updated_at=now(),revision=revision+1 WHERE id=$1`, id, in.OwnerID, strings.TrimSpace(in.Name), in.Enabled, in.Trigger, jsonValue(in.Conditions), jsonValue(in.Actions), in.ScheduleAt, in.IntervalMinutes)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "AUTOMATION_CONFIGURE", id, map[string]any{"trigger": in.Trigger, "enabled": in.Enabled})
	v, e := s.one(r.Context(), "SELECT to_jsonb(a)-'token_id' FROM automation_rules a WHERE id=$1", id)
	respond(w, v, e)
}
func (s *Server) deleteAutomation(w http.ResponseWriter, r *http.Request) {
	var owner, wid string
	e := s.DB.QueryRow(r.Context(), "SELECT owner_id::text,workspace_id::text FROM automation_rules WHERE id=$1", r.PathValue("id")).Scan(&owner, &wid)
	if e != nil || current(r).TokenID != "" || current(r).ScopeRestricted || !s.canWorkspace(r.Context(), current(r), wid, true) || (current(r).ID != owner && !s.automationManager(r.Context(), current(r), wid)) {
		apiError(w, 403, "자동화 관리 권한이 없습니다")
		return
	}
	_, e = s.DB.Exec(r.Context(), "DELETE FROM automation_rules WHERE id=$1", r.PathValue("id"))
	respond(w, map[string]bool{"ok": true}, e)
}
