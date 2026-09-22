package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed jobs_schema.sql
var jobsSchema string

// Job is the durable envelope delivered to module handlers. Payload must never
// contain passwords or decrypted integration secrets. Handlers must enforce the
// operation-specific ACL in addition to the runner's current-account checks.
type Job struct {
	ID              string           `json:"id"`
	Kind            string           `json:"kind"`
	WorkspaceID     string           `json:"workspace_id"`
	OwnerID         string           `json:"owner_id"`
	ActorID         string           `json:"actor_id"`
	TokenID         string           `json:"-"`
	EventID         string           `json:"event_id,omitempty"`
	ResourceID      string           `json:"resource_id,omitempty"`
	AutomationID    string           `json:"automation_id,omitempty"`
	Status          string           `json:"status"`
	LeaseID         string           `json:"-"`
	Payload         map[string]any   `json:"-"`
	Constraints     actorConstraints `json:"-"`
	Attempts        int              `json:"attempts"`
	MaxAttempts     int              `json:"max_attempts"`
	TimeoutSeconds  int              `json:"timeout_seconds"`
	Depth           int              `json:"depth"`
	CancelRequested bool             `json:"cancel_requested"`
	RunAfter        time.Time        `json:"run_after"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}
type JobHandler func(context.Context, Job) (map[string]any, error)
type jobRuntime struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
	once     sync.Once
}
type jobContext struct {
	ActorID, TokenID, JobID, AutomationID, LeaseID string
	Depth                                          int
	Constraints                                    actorConstraints
}
type actorConstraints struct {
	TokenBound bool     `json:"token_bound"`
	PluginID   string   `json:"plugin_id,omitempty"`
	Restricted bool     `json:"scope_restricted"`
	Scopes     []string `json:"scopes"`
}

func constraintsFor(p *Principal) actorConstraints {
	return actorConstraints{TokenBound: p.TokenID != "", PluginID: p.PluginID, Restricted: p.ScopeRestricted || p.TokenID != "", Scopes: append([]string{}, p.Scopes...)}
}

type jobContextKey struct{}
type permanentJobError struct{ message string }

func (e permanentJobError) Error() string { return e.message }
func jobPermanent(message string) error   { return permanentJobError{message} }

func (s *Server) migrateJobs(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, jobsSchema)
	return err
}
func (s *Server) jobRuntime() *jobRuntime {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	if s.jobs == nil {
		s.jobs = &jobRuntime{handlers: map[string]JobHandler{}}
	}
	return s.jobs
}
func (s *Server) RegisterJobHandler(name string, handler func(context.Context, Job) (map[string]any, error)) {
	if name == "" || handler == nil {
		panic("invalid job handler")
	}
	rt := s.jobRuntime()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.handlers[name] = handler
}
func (s *Server) EnqueueJob(ctx context.Context, tx pgx.Tx, kind, ownerID, workspaceID string, payload map[string]any) (string, error) {
	if !validID(ownerID) || !validID(workspaceID) || kind == "" || len(kind) > 80 {
		return "", errors.New("작업 소유자·워크스페이스·유형을 확인하세요")
	}
	if len(jsonValue(payload)) > 1<<20 {
		return "", errors.New("작업 데이터는 1MB 이하여야 합니다")
	}
	if tx == nil {
		created, err := s.DB.Begin(ctx)
		if err != nil {
			return "", err
		}
		defer created.Rollback(ctx)
		id, err := s.EnqueueJob(ctx, created, kind, ownerID, workspaceID, payload)
		if err != nil {
			return "", err
		}
		return id, created.Commit(ctx)
	}
	actor := ownerID
	token := ""
	depth := 0
	automation := ""
	constraints := actorConstraints{}
	if p, _ := ctx.Value(principalKey).(*Principal); p != nil {
		actor = p.ID
		token = p.TokenID
		constraints = constraintsFor(p)
	}
	if v, ok := ctx.Value(jobContextKey{}).(jobContext); ok {
		actor = v.ActorID
		token = v.TokenID
		depth = v.Depth
		automation = v.AutomationID
		constraints = v.Constraints
	}
	id := newID()
	constraints.TokenBound = constraints.TokenBound || token != ""
	_, err := tx.Exec(ctx, `INSERT INTO automation_jobs(id,kind,workspace_id,owner_id,actor_id,token_id,payload,depth,automation_id,actor_constraints) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10)`, id, kind, workspaceID, ownerID, actor, token, jsonValue(payload), depth, automation, jsonValue(constraints))
	return id, err
}

const jobSelect = `id::text,kind,workspace_id::text,owner_id::text,actor_id::text,COALESCE(token_id::text,''),COALESCE(event_id::text,''),resource_id,automation_id,status,COALESCE(lease_id::text,''),payload,attempts,max_attempts,timeout_seconds,depth,cancel_requested,run_after,created_at,updated_at,actor_constraints`

func scanJob(row pgx.Row) (Job, error) {
	var j Job
	var data []byte
	var constraints []byte
	err := row.Scan(&j.ID, &j.Kind, &j.WorkspaceID, &j.OwnerID, &j.ActorID, &j.TokenID, &j.EventID, &j.ResourceID, &j.AutomationID, &j.Status, &j.LeaseID, &data, &j.Attempts, &j.MaxAttempts, &j.TimeoutSeconds, &j.Depth, &j.CancelRequested, &j.RunAfter, &j.CreatedAt, &j.UpdatedAt, &constraints)
	if err == nil {
		err = json.Unmarshal(data, &j.Payload)
	}
	if err == nil {
		err = json.Unmarshal(constraints, &j.Constraints)
	}
	return j, err
}
func (s *Server) claimJob(ctx context.Context) (Job, error) {
	lease := newID()
	return scanJob(s.DB.QueryRow(ctx, `WITH next AS (SELECT id FROM automation_jobs WHERE cancel_requested=false AND attempts<max_attempts AND ((status='pending' AND run_after<=now()) OR (status='running' AND lease_until<now())) ORDER BY run_after,created_at FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE automation_jobs SET status='running',attempts=attempts+1,lease_id=$1,lease_until=now()+make_interval(secs=>timeout_seconds+30),updated_at=now() WHERE id=(SELECT id FROM next) RETURNING `+jobSelect, lease))
}
func (s *Server) workerPrincipal(ctx context.Context, userID, tokenID, workspaceID string) (*Principal, error) {
	p := &Principal{}
	var disabled bool
	err := s.DB.QueryRow(ctx, `SELECT id::text,email,name,role,kind,disabled FROM users WHERE id=$1`, userID).Scan(&p.ID, &p.Email, &p.Name, &p.Role, &p.Kind, &disabled)
	if err != nil || disabled {
		return nil, jobPermanent("작업 실행 계정이 없거나 비활성화되었습니다")
	}
	if tokenID != "" {
		var expires time.Time
		var revoked *time.Time
		err = s.DB.QueryRow(ctx, `SELECT id::text,workspace_id::text,scopes,expires_at,revoked_at FROM api_keys WHERE id=$1 AND user_id=$2`, tokenID, userID).Scan(&p.TokenID, &p.WorkspaceID, &p.Scopes, &expires, &revoked)
		if err != nil || revoked != nil || !expires.After(time.Now()) {
			return nil, jobPermanent("작업에 사용한 API 키가 만료되었거나 폐기되었습니다")
		}
		cfg, e := s.settings(ctx)
		if e != nil {
			return nil, e
		}
		allowed := settingStrings(cfg, "allowed_key_scopes", keyScopes)
		p.Scopes = slices.DeleteFunc(p.Scopes, func(v string) bool { return !slices.Contains(allowed, v) })
		if p.WorkspaceID != workspaceID {
			return nil, jobPermanent("작업 API 키의 워크스페이스 범위가 일치하지 않습니다")
		}
	}
	if inherited, ok := ctx.Value(jobContextKey{}).(jobContext); ok && inherited.ActorID == userID {
		constraints := inherited.Constraints
		if constraints.TokenBound && inherited.TokenID == "" {
			return nil, jobPermanent("원래 API 키 연결이 사라졌습니다. 현재 키로 작업을 다시 요청하세요")
		}
		if constraints.Restricted || constraints.PluginID != "" {
			scopes := []string{}
			for _, scope := range constraints.Scopes {
				if tokenID == "" || slices.Contains(p.Scopes, scope) {
					scopes = append(scopes, scope)
				}
			}
			p.ScopeRestricted = true
			p.WorkspaceID = workspaceID
			p.PluginID = constraints.PluginID
			p.Scopes = scopes
			if p.PluginID != "" {
				r := automationRequest(ctx, p, http.MethodGet, nil)
				_, caps, e := s.pluginGrant(r, p.PluginID, workspaceID)
				if e != nil {
					return nil, jobPermanent("작업 플러그인의 현재 사용 권한이 회수되었습니다")
				}
				p.Scopes = slices.DeleteFunc(p.Scopes, func(scope string) bool { return !slices.Contains(caps, scope) })
			}
		}
	}
	if !s.canWorkspace(ctx, p, workspaceID, false) {
		return nil, jobPermanent("작업 워크스페이스 접근 권한이 없습니다")
	}
	return p, nil
}
func jobScope(p *Principal, scope string) bool {
	return (p.TokenID == "" && !p.ScopeRestricted) || slices.Contains(p.Scopes, scope)
}
func (s *Server) runJob(ctx context.Context, j Job) {
	started := time.Now()
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(j.TimeoutSeconds)*time.Second)
	defer cancel()
	runCtx = context.WithValue(runCtx, jobContextKey{}, jobContext{ActorID: j.ActorID, TokenID: j.TokenID, JobID: j.ID, AutomationID: j.AutomationID, Depth: j.Depth, LeaseID: j.LeaseID, Constraints: j.Constraints})
	monitorDone := make(chan struct{})
	var pluginPolicyCancelled atomic.Bool
	defer close(monitorDone)
	go func() {
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		lastPluginCheck := time.Time{}
		for {
			select {
			case <-monitorDone:
				return
			case <-runCtx.Done():
				return
			case <-tick.C:
				if (j.Constraints.PluginID != "" || j.Constraints.TokenBound || j.TokenID != "") && (lastPluginCheck.IsZero() || time.Since(lastPluginCheck) >= time.Second) {
					lastPluginCheck = time.Now()
					fresh, err := s.workerPrincipal(runCtx, j.ActorID, j.TokenID, j.WorkspaceID)
					if err == nil {
						for _, scope := range j.Constraints.Scopes {
							if !hasIntegrationScope(fresh, scope) {
								err = jobPermanent("작업 중 원래 API 키·플러그인 권한이 축소되었습니다")
								break
							}
						}
					}
					if err != nil {
						pluginPolicyCancelled.Store(true)
						cancel()
						return
					}
				}
				var stopped bool
				err := s.DB.QueryRow(runCtx, `SELECT cancel_requested OR status<>'running' OR lease_id<>$2::uuid FROM automation_jobs WHERE id=$1`, j.ID, j.LeaseID).Scan(&stopped)
				if err == nil && stopped {
					cancel()
					return
				}
			}
		}
	}()
	_, err := s.workerPrincipal(runCtx, j.OwnerID, "", j.WorkspaceID)
	if err == nil {
		_, err = s.workerPrincipal(runCtx, j.ActorID, j.TokenID, j.WorkspaceID)
	}
	var result map[string]any
	if err == nil {
		rt := s.jobRuntime()
		rt.mu.RLock()
		handler := rt.handlers[j.Kind]
		rt.mu.RUnlock()
		if handler == nil {
			err = jobPermanent("등록되지 않은 작업 유형입니다")
		} else {
			result, err = invokeJobHandler(handler, runCtx, j)
		}
	}
	finishCtx, finishCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer finishCancel()
	status := "succeeded"
	message := ""
	delay := time.Duration(1<<min(j.Attempts, 10)) * time.Second
	if err != nil {
		var retry interface{ RetryAfter() time.Duration }
		if errors.As(err, &retry) && retry.RetryAfter() > delay {
			delay = min(retry.RetryAfter(), time.Hour)
		}
		message = err.Error()
		if len(message) > 500 {
			message = message[:500]
		}
		status = "pending"
		var permanent permanentJobError
		if errors.As(err, &permanent) || errors.Is(err, errQueryResultTooLarge) || j.Attempts >= j.MaxAttempts {
			status = "failed"
		}
		if errors.Is(err, errQueryResultTooLarge) {
			message = errQueryResultTooLarge.Error()
			result = nil
		}
		if errors.Is(err, context.Canceled) {
			message = "작업이 취소되었거나 서비스가 종료되었습니다"
		}
	}
	if len(jsonValue(result)) > 1<<20 {
		result = map[string]any{"truncated": true}
		status = "failed"
		message = "작업 결과가 1MB 제한을 초과했습니다"
	}
	if pluginPolicyCancelled.Load() {
		status = "cancelled"
		message = "API 키·플러그인 또는 현재 실행 권한이 변경되어 작업을 취소했습니다"
		result = nil
	}
	tx, e := s.DB.Begin(finishCtx)
	if e != nil {
		return
	}
	defer tx.Rollback(finishCtx)
	e = tx.QueryRow(finishCtx, `UPDATE automation_jobs SET status=CASE WHEN cancel_requested THEN 'cancelled' ELSE $3 END,result=$4,last_error=$5,run_after=now()+$6::interval,lease_until=NULL,lease_id=NULL,updated_at=now(),finished_at=CASE WHEN cancel_requested OR $3 IN ('succeeded','failed','cancelled') THEN now() ELSE NULL END WHERE id=$1 AND lease_id=$2 AND status='running' RETURNING status`, j.ID, j.LeaseID, status, jsonValue(result), message, delay.String()).Scan(&status)
	if e != nil {
		return
	}
	_, e = tx.Exec(finishCtx, `INSERT INTO automation_job_attempts(job_id,attempt,status,error,result,duration_ms) VALUES($1,$2,$3,$4,$5,$6)`, j.ID, j.Attempts, status, message, jsonValue(result), time.Since(started).Milliseconds())
	if e == nil {
		e = tx.Commit(finishCtx)
	}
	if e != nil {
		slog.Error("job result persistence failed", "job_id", j.ID, "error", e)
	}
}
func invokeJobHandler(handler JobHandler, ctx context.Context, j Job) (result map[string]any, err error) {
	defer func() {
		if recover() != nil {
			result = nil
			err = jobPermanent("작업 처리 중 내부 오류가 발생했습니다")
		}
	}()
	return handler(ctx, j)
}

// StartJobs starts bounded workers; SQL leases recover work after process loss.
func (s *Server) StartJobs(ctx context.Context) {
	rt := s.jobRuntime()
	rt.once.Do(func() {
		for index := 0; index < 16; index++ {
			go func(index int) {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						var paused bool
						var concurrency int
						if s.DB.QueryRow(ctx, "SELECT paused,concurrency FROM job_settings WHERE id=1").Scan(&paused, &concurrency) != nil || paused || index >= concurrency {
							continue
						}
						j, e := s.claimJob(ctx)
						if e == nil {
							s.runJob(ctx, j)
						}
					}
				}
			}(index)
		}
		go s.scheduleAutomations(ctx)
	})
}
func (s *Server) registerJobs() {
	s.handle("GET /api/v1/jobs", s.listJobs)
	s.handle("GET /api/v1/jobs/{id}", s.getJob)
	s.handle("POST /api/v1/jobs/{id}/cancel", s.cancelJob)
	s.handle("POST /api/v1/jobs/{id}/retry", s.retryJob)
	s.admin("GET /api/v1/admin/jobs/settings", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.one(r.Context(), "SELECT to_jsonb(x) FROM job_settings x WHERE id=1")
		respond(w, v, e)
	})
	s.admin("PUT /api/v1/admin/jobs/settings", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Paused        bool `json:"paused"`
			Concurrency   int  `json:"concurrency"`
			RetentionDays int  `json:"retention_days"`
		}
		if decode(r, &in) != nil || in.Concurrency < 1 || in.Concurrency > 16 || in.RetentionDays < 1 || in.RetentionDays > 3650 {
			apiError(w, 400, "동시 작업은 1~16개, 이력 보존은 1~3650일로 입력하세요")
			return
		}
		_, e := s.DB.Exec(r.Context(), "UPDATE job_settings SET paused=$1,concurrency=$2,retention_days=$3 WHERE id=1", in.Paused, in.Concurrency, in.RetentionDays)
		if e != nil {
			respond(w, nil, e)
			return
		}
		s.audit(r, "JOBS_SETTINGS_UPDATE", "jobs", in)
		jsonResponse(w, 200, in)
	})
}
func (s *Server) jobVisible(r *http.Request, j Job) bool {
	p := current(r)
	if !s.canWorkspace(r.Context(), p, j.WorkspaceID, false) || !(p.ID == j.OwnerID || p.ID == j.ActorID || p.Role == "admin") {
		return false
	}
	if j.EventID != "" {
		event, e := s.eventByID(r.Context(), j.EventID)
		if e != nil || !s.automationSourceAccess(r.Context(), p, event, false) {
			return false
		}
	}
	if j.ResourceID != "" && validID(j.ResourceID) {
		var count int
		_ = s.DB.QueryRow(r.Context(), "SELECT count(*) FROM documents WHERE id=$1", j.ResourceID).Scan(&count)
		if count > 0 && !s.canDocument(r.Context(), p, j.ResourceID, false) {
			return false
		}
	}
	return true
}
func (s *Server) loadVisibleJob(w http.ResponseWriter, r *http.Request) (Job, bool) {
	j, e := scanJob(s.DB.QueryRow(r.Context(), "SELECT "+jobSelect+" FROM automation_jobs WHERE id=$1", r.PathValue("id")))
	if e != nil || !s.jobVisible(r, j) {
		apiError(w, 404, "작업을 찾을 수 없거나 권한이 없습니다")
		return j, false
	}
	return j, true
}
func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !validID(wid) || !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 권한이 없습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+jobSelect+" FROM automation_jobs WHERE workspace_id=$1 ORDER BY created_at DESC LIMIT 200", wid)
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
func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	j, ok := s.loadVisibleJob(w, r)
	if !ok {
		return
	}
	attempts, e := s.rows(r.Context(), "SELECT to_jsonb(a) FROM automation_job_attempts a WHERE job_id=$1 ORDER BY id DESC", j.ID)
	respond(w, map[string]any{"job": j, "attempts": attempts}, e)
}
func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	j, ok := s.loadVisibleJob(w, r)
	if !ok {
		return
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted {
		apiError(w, 403, "작업 관리는 사용자 세션이 필요합니다")
		return
	}
	_, e := s.DB.Exec(r.Context(), `UPDATE automation_jobs SET cancel_requested=true,finished_at=CASE WHEN status='pending' THEN now() ELSE finished_at END,status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END,updated_at=now() WHERE id=$1 AND status IN ('pending','running')`, j.ID)
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) retryJob(w http.ResponseWriter, r *http.Request) {
	j, ok := s.loadVisibleJob(w, r)
	if !ok {
		return
	}
	if current(r).TokenID != "" || current(r).ScopeRestricted {
		apiError(w, 403, "작업 관리는 사용자 세션이 필요합니다")
		return
	}
	tag, e := s.DB.Exec(r.Context(), `UPDATE automation_jobs SET status='pending',attempts=0,cancel_requested=false,last_error='',run_after=now(),finished_at=NULL,updated_at=now() WHERE id=$1 AND status IN ('failed','cancelled')`, j.ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "실패하거나 취소된 작업만 다시 실행할 수 있습니다")
		return
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
