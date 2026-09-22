package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed system_status.sql
var systemStatusSchema string

func (s *Server) migrateSystemStatus(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, systemStatusSchema)
	return e
}

const systemStatusNotice = "관측값은 명시된 보고 주체가 전달한 주장입니다. madi가 서버에 접속해 배포나 정상 동작을 독립 검증한 결과가 아닙니다. 등록 기대값·Runbook 실행 결과·커넥터 수집 사실은 서로 다른 근거입니다."

type systemStatusPolicy struct {
	Revision  int64 `json:"revision"`
	Enabled   bool  `json:"enabled"`
	MaxTTL    int   `json:"max_ttl_seconds"`
	MaxAge    int   `json:"max_observation_age_seconds"`
	Retention int   `json:"retention_days"`
}
type systemStatusExpected struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	Deployment  string `json:"deployment"`
	Note        string `json:"note"`
}
type systemStatusObservation struct {
	Version    string `json:"version"`
	Deployment string `json:"deployment"`
	Health     string `json:"health"`
	Note       string `json:"note"`
}
type systemStatusCard struct {
	DocumentID      string               `json:"document_id"`
	WorkspaceID     string               `json:"workspace_id"`
	Revision        int64                `json:"revision"`
	DocumentVersion int                  `json:"document_version"`
	Epoch           int64                `json:"verification_epoch"`
	ReportRevision  int64                `json:"report_revision"`
	OwnerID         string               `json:"owner_id"`
	Reporters       []string             `json:"reporter_ids"`
	TTL             int                  `json:"ttl_seconds"`
	Expected        systemStatusExpected `json:"expected"`
	Updated         time.Time            `json:"updated_at"`
}
type systemStatusReport struct {
	ID              string                  `json:"id"`
	RequestID       string                  `json:"request_id"`
	Revision        int64                   `json:"revision"`
	CardRevision    int64                   `json:"card_revision"`
	Epoch           int64                   `json:"verification_epoch"`
	DocumentVersion int                     `json:"document_version"`
	ActorID         string                  `json:"actor_id"`
	ActorKind       string                  `json:"actor_kind"`
	Observed        time.Time               `json:"observed_at"`
	Received        time.Time               `json:"received_at"`
	Observation     systemStatusObservation `json:"observation"`
	Expected        systemStatusExpected    `json:"expected_at_report"`
}
type systemStatusFailure struct {
	Status  int
	Message string
}

func (e *systemStatusFailure) Error() string    { return e.Message }
func statusFail(code int, message string) error { return &systemStatusFailure{code, message} }
func systemStatusError(w http.ResponseWriter, e error) {
	var failure *systemStatusFailure
	if errors.As(e, &failure) {
		apiError(w, failure.Status, failure.Message)
		return
	}
	if errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 404, "현재 접근 가능한 운영 카드가 없습니다")
		return
	}
	respond(w, nil, e)
}
func systemStatusRevision(value int64, zero bool) bool {
	return value >= 0 && (zero || value > 0) && value < 9007199254740991
}
func systemStatusNewEpoch() int64 {
	// Distinguish a deleted/recreated card from an old offline report without
	// retaining an unbounded tombstone table. 48 random UUID bits remain exactly
	// representable by JavaScript and leave room for monotonic local increments.
	value, _ := strconv.ParseInt(strings.ReplaceAll(newID(), "-", "")[:12], 16, 64)
	return value + 1
}
func systemStatusPolicyTx(ctx context.Context, tx pgx.Tx, write bool) (systemStatusPolicy, error) {
	var p systemStatusPolicy
	lock := " FOR SHARE"
	if write {
		lock = " FOR UPDATE"
	}
	e := tx.QueryRow(ctx, `SELECT revision,enabled,max_ttl_seconds,max_observation_age_seconds,retention_days FROM system_status_policy WHERE id=1`+lock).Scan(&p.Revision, &p.Enabled, &p.MaxTTL, &p.MaxAge, &p.Retention)
	return p, e
}
func systemStatusDocumentTx(r *http.Request, tx pgx.Tx, id string, write bool) (string, int, error) {
	if !validID(id) || current(r) == nil || !hasIntegrationScope(current(r), "document:read") || write && !hasIntegrationScope(current(r), "document:write") {
		return "", 0, pgx.ErrNoRows
	}
	var wid string
	var version int
	lock := " FOR SHARE"
	if write {
		lock = " FOR UPDATE"
	}
	e := tx.QueryRow(r.Context(), `SELECT workspace_id::text,version FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,$3) AND ($4='' OR workspace_id=NULLIF($4,'')::uuid)`+lock, id, current(r).ID, write, current(r).WorkspaceID).Scan(&wid, &version)
	return wid, version, e
}
func (s *Server) systemStatusCardTx(ctx context.Context, tx pgx.Tx, id string, write bool) (systemStatusCard, error) {
	var c systemStatusCard
	var sealed string
	lock := " FOR SHARE"
	if write {
		lock = " FOR UPDATE"
	}
	e := tx.QueryRow(ctx, `SELECT document_id::text,workspace_id::text,revision,document_version,verification_epoch,report_revision,owner_id::text,reporter_ids,ttl_seconds,ciphertext,updated_at FROM system_status_cards WHERE document_id=$1`+lock, id).Scan(&c.DocumentID, &c.WorkspaceID, &c.Revision, &c.DocumentVersion, &c.Epoch, &c.ReportRevision, &c.OwnerID, &c.Reporters, &c.TTL, &sealed, &c.Updated)
	if e != nil {
		return c, e
	}
	raw, e := s.decrypt(sealed)
	if e == nil {
		e = json.Unmarshal([]byte(raw), &c.Expected)
	}
	return c, e
}
func (s *Server) systemStatusReportsTx(ctx context.Context, tx pgx.Tx, id string, limit int) ([]systemStatusReport, error) {
	rows, e := tx.Query(ctx, `SELECT id::text,request_id::text,revision,card_revision,verification_epoch,document_version,actor_id::text,actor_kind,observed_at,received_at,ciphertext FROM system_status_reports WHERE document_id=$1 AND received_at >= clock_timestamp()-make_interval(days=>(SELECT retention_days FROM system_status_policy WHERE id=1)) ORDER BY revision DESC LIMIT $2`, id, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []systemStatusReport{}
	for rows.Next() {
		var report systemStatusReport
		var sealed string
		if e = rows.Scan(&report.ID, &report.RequestID, &report.Revision, &report.CardRevision, &report.Epoch, &report.DocumentVersion, &report.ActorID, &report.ActorKind, &report.Observed, &report.Received, &sealed); e != nil {
			return nil, e
		}
		raw, e := s.decrypt(sealed)
		if e != nil {
			return nil, e
		}
		var payload struct {
			Observation systemStatusObservation `json:"observation"`
			Expected    systemStatusExpected    `json:"expected"`
		}
		if e = json.Unmarshal([]byte(raw), &payload); e != nil {
			return nil, e
		}
		report.Observation = payload.Observation
		report.Expected = payload.Expected
		out = append(out, report)
	}
	return out, rows.Err()
}
func systemStatusExpectedValid(e systemStatusExpected) bool {
	return strings.TrimSpace(e.Name) != "" && len(e.Name) <= 200 && len(e.Environment) <= 200 && len(e.Version) <= 500 && len(e.Deployment) <= 500 && len(e.Note) <= 4000 && (strings.TrimSpace(e.Version) != "" || strings.TrimSpace(e.Deployment) != "")
}
func systemStatusObservationValid(o systemStatusObservation) bool {
	return len(o.Version) <= 500 && len(o.Deployment) <= 500 && len(o.Note) <= 4000 && oneOf(o.Health, "healthy", "degraded", "down", "unknown") && (strings.TrimSpace(o.Version) != "" || strings.TrimSpace(o.Deployment) != "")
}
func systemStatusFreshness(c systemStatusCard, p systemStatusPolicy, version int, report *systemStatusReport, now time.Time) map[string]any {
	out := map[string]any{"state": "unobserved", "fresh": false, "drift_fields": []string{}, "independently_verified": false}
	if !p.Enabled {
		out["state"] = "disabled"
		return out
	}
	if c.DocumentVersion != version {
		out["state"] = "document_changed"
		return out
	}
	if report == nil {
		return out
	}
	if report.Epoch != c.Epoch || report.CardRevision != c.Revision || report.DocumentVersion != version {
		out["state"] = "baseline_changed"
		return out
	}
	ttl := time.Duration(min(c.TTL, p.MaxTTL)) * time.Second
	expires := report.Observed.Add(ttl)
	if received := report.Received.Add(ttl); received.Before(expires) {
		expires = received
	}
	out["expires_at"] = expires
	if !now.Before(expires) || report.Observed.After(now.Add(time.Minute)) {
		out["state"] = "stale"
		return out
	}
	drift := []string{}
	if c.Expected.Version != "" && c.Expected.Version != report.Observation.Version {
		drift = append(drift, "version")
	}
	if c.Expected.Deployment != "" && c.Expected.Deployment != report.Observation.Deployment {
		drift = append(drift, "deployment")
	}
	out["fresh"] = true
	out["drift_fields"] = drift
	out["state"] = "reported_match"
	if len(drift) > 0 {
		out["state"] = "drift"
	}
	return out
}
func systemStatusHuman(p *Principal) bool {
	return p != nil && p.Kind == "user" && p.TokenID == "" && p.PluginID == "" && !p.ScopeRestricted
}
func (s *Server) systemStatusActorTx(r *http.Request, tx pgx.Tx, wid, id string, write bool) error {
	scopes := []string{"document:read"}
	if write {
		scopes = append(scopes, "document:write")
	}
	if e := s.knowledgeActorTx(r, tx, wid, scopes...); e != nil {
		return e
	}
	// Parent/space sharing can change independently of the locked document row.
	// Re-evaluate the inherited ACL in a fresh statement after all row-lock waits.
	var allowed bool
	e := tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,$3)`, current(r).ID, id, write).Scan(&allowed)
	if e != nil {
		return e
	}
	if !allowed {
		return statusFail(403, "현재 문서·상위 공간 접근 권한이 변경되었습니다")
	}
	return nil
}
func systemStatusReporter(c systemStatusCard, p *Principal) bool {
	return p != nil && slices.Contains(c.Reporters, p.ID) && hasIntegrationScope(p, "document:read") && hasIntegrationScope(p, "document:write") && p.PluginID == "" && !p.ScopeRestricted
}
