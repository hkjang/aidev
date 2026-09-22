package server

import (
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) getSystemStatus(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		systemStatusError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	id := r.PathValue("id")
	wid, version, e := systemStatusDocumentTx(r, tx, id, false)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	c, e := s.systemStatusCardTx(r.Context(), tx, id, false)
	missing := errors.Is(e, pgx.ErrNoRows)
	if e != nil && !missing {
		systemStatusError(w, e)
		return
	}
	p, e := systemStatusPolicyTx(r.Context(), tx, false)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	var canWrite bool
	e = tx.QueryRow(r.Context(), `SELECT madi_document_allowed($1,$2,true)`, current(r).ID, id).Scan(&canWrite)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	out := map[string]any{"policy": p, "current_document_version": version, "card": nil, "reports": []any{}, "latest_report": nil, "summary": map[string]any{"state": "unregistered", "fresh": false, "independently_verified": false}, "can_manage": systemStatusHuman(current(r)) && canWrite, "can_report": false, "reporter_options": []any{}, "notice": systemStatusNotice}
	if !missing {
		reports, e := s.systemStatusReportsTx(r.Context(), tx, id, 50)
		if e != nil {
			systemStatusError(w, e)
			return
		}
		var latest *systemStatusReport
		if len(reports) > 0 {
			latest = &reports[0]
		}
		if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, map[string]any{"card": c.Expected, "reports": reports}); e != nil {
			apiError(w, 422, "현재 정보 보호 정책에 따라 카드·보고 내용을 표시할 수 없습니다")
			return
		}
		out["card"] = c
		out["reports"] = reports
		out["latest_report"] = latest
		out["summary"] = systemStatusFreshness(c, p, version, latest, time.Now())
		out["can_report"] = p.Enabled && canWrite && c.DocumentVersion == version && systemStatusReporter(c, current(r)) && c.TTL <= p.MaxTTL
	}
	if boolean(out, "can_manage") {
		rows, e := runbookRowsTx(r.Context(), tx, `SELECT jsonb_build_object('id',u.id,'name',left(u.name,120),'kind',u.kind) FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$1 WHERE NOT u.disabled AND u.role<>'viewer' AND m.role IN ('owner','admin','editor') AND madi_document_allowed(u.id,$2,true) ORDER BY u.name,u.id LIMIT 101`, wid, id)
		if e != nil {
			systemStatusError(w, e)
			return
		}
		out["reporter_options_truncated"] = len(rows) > 100
		if len(rows) > 100 {
			rows = rows[:100]
		}
		out["reporter_options"] = rows
	}
	// No logs, runner credentials, remote URLs or remote bodies enter this card.
	context, e := s.systemStatusContextTx(r, tx, id, wid)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	out["context"] = context
	if e = s.systemStatusActorTx(r, tx, wid, id, false); e != nil {
		apiError(w, 403, "현재 문서·세션·키 권한을 다시 확인하세요")
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		systemStatusError(w, e)
		return
	}
	jsonResponse(w, 200, out)
}

func (s *Server) putSystemStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision        int64                `json:"revision"`
		DocumentVersion int                  `json:"document_version"`
		OwnerID         string               `json:"owner_id"`
		Reporters       []string             `json:"reporter_ids"`
		TTL             int                  `json:"ttl_seconds"`
		Expected        systemStatusExpected `json:"expected"`
		Confirm         bool                 `json:"confirm"`
	}
	if !systemStatusHuman(current(r)) {
		apiError(w, 403, "카드 기준 변경은 일반 사용자 로그인에서만 가능합니다")
		return
	}
	if decode(r, &in) != nil || !systemStatusRevision(in.Revision, true) || in.DocumentVersion < 1 || in.DocumentVersion > 2147483647 || !validID(in.OwnerID) || len(in.Reporters) > 100 || !in.Confirm || in.TTL < 60 || in.TTL > 604800 || !systemStatusExpectedValid(in.Expected) {
		apiError(w, 400, "현재 버전·담당자·기대값·60초 이상 TTL과 열람자 공유 확인이 필요합니다")
		return
	}
	in.OwnerID = strings.ToLower(in.OwnerID)
	seen := map[string]bool{}
	for i, id := range in.Reporters {
		id = strings.ToLower(id)
		in.Reporters[i] = id
		if !validID(id) || seen[id] {
			apiError(w, 400, "보고 주체 ID는 중복 없는 실제 사용자 ID여야 합니다")
			return
		}
		seen[id] = true
	}
	sort.Strings(in.Reporters)
	if in.Reporters == nil {
		in.Reporters = []string{}
	}
	in.Expected.Name = strings.TrimSpace(in.Expected.Name)
	in.Expected.Environment = strings.TrimSpace(in.Expected.Environment)
	in.Expected.Version = strings.TrimSpace(in.Expected.Version)
	in.Expected.Deployment = strings.TrimSpace(in.Expected.Deployment)
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		systemStatusError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	id := r.PathValue("id")
	wid, version, e := systemStatusDocumentTx(r, tx, id, true)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	c, e := s.systemStatusCardTx(r.Context(), tx, id, true)
	missing := errors.Is(e, pgx.ErrNoRows)
	if e != nil && !missing {
		systemStatusError(w, e)
		return
	}
	p, e := systemStatusPolicyTx(r.Context(), tx, false)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if !p.Enabled {
		apiError(w, 409, "관리자가 운영 현황 카드를 활성화하지 않았습니다")
		return
	}
	if version != in.DocumentVersion || (!missing && c.Revision != in.Revision) || missing && in.Revision != 0 {
		apiError(w, 409, "문서 또는 카드 기준이 변경되었습니다. 현재 저장본을 다시 확인하세요")
		return
	}
	if in.TTL > p.MaxTTL {
		apiError(w, 400, "TTL이 현재 관리자 정책의 최대값을 초과합니다")
		return
	}
	ids := append([]string{in.OwnerID}, in.Reporters...)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	for _, actor := range ids {
		var kind string
		e = tx.QueryRow(r.Context(), `SELECT u.kind FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=$2 WHERE u.id=$1 AND NOT u.disabled AND u.role<>'viewer' AND m.role IN ('owner','admin','editor') AND madi_document_allowed(u.id,$3,true) FOR SHARE OF u,m`, actor, wid, id).Scan(&kind)
		if e != nil || actor == in.OwnerID && kind != "user" {
			apiError(w, 400, "담당자는 현재 작성 권한이 있는 사용자, 보고 주체는 현재 작성 권한이 있는 사용자 또는 서비스 계정이어야 합니다")
			return
		}
	}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, in.Expected); e != nil {
		apiError(w, 422, "기대값과 메모를 현재 정보 보호 정책에 맞게 수정하세요")
		return
	}
	sealed, e := s.encrypt(string(jsonValue(in.Expected)))
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if e = s.systemStatusActorTx(r, tx, wid, id, true); e != nil {
		apiError(w, 403, "현재 문서·세션 권한을 다시 확인하세요")
		return
	}
	if missing {
		_, e = tx.Exec(r.Context(), `INSERT INTO system_status_cards(document_id,workspace_id,document_version,owner_id,reporter_ids,ttl_seconds,ciphertext,updated_by,verification_epoch) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, wid, version, in.OwnerID, in.Reporters, in.TTL, sealed, current(r).ID, systemStatusNewEpoch())
	} else {
		_, e = tx.Exec(r.Context(), `UPDATE system_status_cards SET revision=revision+1,verification_epoch=verification_epoch+1,document_version=$2,owner_id=$3,reporter_ids=$4,ttl_seconds=$5,ciphertext=$6,updated_by=$7,updated_at=clock_timestamp() WHERE document_id=$1`, id, version, in.OwnerID, in.Reporters, in.TTL, sealed, current(r).ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		systemStatusError(w, e)
		return
	}
	s.audit(r, "SYSTEM_STATUS_EXPECTATION_UPDATE", id, map[string]any{"revision": in.Revision + 1, "document_version": version, "reporter_count": len(in.Reporters)})
	jsonResponse(w, 200, map[string]any{"revision": in.Revision + 1, "document_version": version, "notice": systemStatusNotice})
}

func (s *Server) reportSystemStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RequestID       string                  `json:"request_id"`
		CardRevision    int64                   `json:"card_revision"`
		ReportRevision  int64                   `json:"report_revision"`
		Epoch           int64                   `json:"verification_epoch"`
		DocumentVersion int                     `json:"document_version"`
		Observed        time.Time               `json:"observed_at"`
		Observation     systemStatusObservation `json:"observation"`
		Confirm         bool                    `json:"confirm"`
	}
	if decode(r, &in) != nil || !validID(in.RequestID) || !systemStatusRevision(in.CardRevision, false) || !systemStatusRevision(in.ReportRevision, true) || !systemStatusRevision(in.Epoch, false) || in.DocumentVersion < 1 || in.DocumentVersion > 2147483647 || !in.Confirm || in.Observed.IsZero() || !systemStatusObservationValid(in.Observation) {
		apiError(w, 400, "보고 ID·현재 기준/보고 revision·관측 시각·관측값과 명시 보고 확인이 필요합니다")
		return
	}
	in.Observation.Version = strings.TrimSpace(in.Observation.Version)
	in.Observation.Deployment = strings.TrimSpace(in.Observation.Deployment)
	hash := digest(string(jsonValue(map[string]any{"request": in, "actor_id": current(r).ID, "token_id": current(r).TokenID})))
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		systemStatusError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	id := r.PathValue("id")
	wid, version, e := systemStatusDocumentTx(r, tx, id, true)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	c, e := s.systemStatusCardTx(r.Context(), tx, id, true)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	p, e := systemStatusPolicyTx(r.Context(), tx, false)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if !p.Enabled || c.TTL > p.MaxTTL {
		apiError(w, 409, "운영 보고가 비활성화되었거나 TTL 기준을 다시 등록해야 합니다")
		return
	}
	if !systemStatusReporter(c, current(r)) {
		apiError(w, 403, "현재 카드에 지정된 보고 주체의 작성 권한이 필요합니다")
		return
	}
	if c.Revision != in.CardRevision || c.Epoch != in.Epoch || c.DocumentVersion != version || version != in.DocumentVersion {
		apiError(w, 409, "문서 또는 운영 기준이 변경되었습니다. 새 기준을 다시 확인하세요")
		return
	}
	// Resolve retries only after current resource/policy authorization. A replay
	// is not a capability and never extends observed_at/received_at/TTL.
	var previousID, previousHash string
	var previousRevision int64
	e = tx.QueryRow(r.Context(), `SELECT id::text,payload_hash,revision FROM system_status_reports WHERE document_id=$1 AND request_id=$2`, id, in.RequestID).Scan(&previousID, &previousHash, &previousRevision)
	if e == nil {
		if previousHash != hash {
			apiError(w, 409, "같은 보고 ID에 다른 내용을 재전송할 수 없습니다")
			return
		}
		if e = s.systemStatusActorTx(r, tx, wid, id, true); e != nil {
			apiError(w, 403, "현재 보고 계정·세션·키 권한을 다시 확인하세요")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			systemStatusError(w, e)
			return
		}
		jsonResponse(w, 200, map[string]any{"id": previousID, "revision": previousRevision, "replayed": true, "notice": systemStatusNotice})
		return
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		systemStatusError(w, e)
		return
	}
	if c.ReportRevision != in.ReportRevision {
		apiError(w, 409, "다른 관측 보고가 먼저 저장되었습니다. 최신 보고를 확인하세요")
		return
	}
	var now time.Time
	if e = tx.QueryRow(r.Context(), `SELECT clock_timestamp()`).Scan(&now); e != nil {
		systemStatusError(w, e)
		return
	}
	if in.Observed.After(now.Add(time.Minute)) || in.Observed.Before(now.Add(-time.Duration(p.MaxAge)*time.Second)) {
		apiError(w, 400, "관측 시각은 서버 시각보다 60초 넘게 미래일 수 없고 현재 정책의 최대 과거 범위 안이어야 합니다")
		return
	}
	var latest time.Time
	e = tx.QueryRow(r.Context(), `SELECT observed_at FROM system_status_reports WHERE document_id=$1 ORDER BY revision DESC LIMIT 1`, id).Scan(&latest)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		systemStatusError(w, e)
		return
	}
	if !latest.IsZero() && in.Observed.Before(latest) {
		apiError(w, 409, "더 오래된 관측으로 최신 보고를 교체할 수 없습니다")
		return
	}
	payload := map[string]any{"observation": in.Observation, "expected": c.Expected}
	if e = s.checkEvidenceProtection(r.Context(), tx, current(r), wid, payload); e != nil {
		apiError(w, 422, "관측값과 메모를 현재 정보 보호 정책에 맞게 수정하세요")
		return
	}
	sealed, e := s.encrypt(string(jsonValue(payload)))
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if e = s.systemStatusActorTx(r, tx, wid, id, true); e != nil {
		apiError(w, 403, "현재 보고 계정·세션·키 권한을 다시 확인하세요")
		return
	}
	reportID := newID()
	kind := "manual"
	if current(r).TokenID != "" {
		kind = "api_user"
		if current(r).Kind == "service" {
			kind = "service_account"
		}
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO system_status_reports(id,document_id,request_id,payload_hash,revision,card_revision,verification_epoch,document_version,policy_revision,actor_id,actor_kind,token_id,observed_at,ciphertext) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid,$13,$14)`, reportID, id, in.RequestID, hash, c.ReportRevision+1, c.Revision, c.Epoch, version, p.Revision, current(r).ID, kind, current(r).TokenID, in.Observed, sealed)
	if e == nil {
		_, e = tx.Exec(r.Context(), `UPDATE system_status_cards SET report_revision=report_revision+1,updated_at=clock_timestamp() WHERE document_id=$1`, id)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		systemStatusError(w, e)
		return
	}
	s.audit(r, "SYSTEM_STATUS_REPORTED", id, map[string]any{"report_id": reportID, "revision": c.ReportRevision + 1, "kind": kind})
	jsonResponse(w, 201, map[string]any{"id": reportID, "revision": c.ReportRevision + 1, "replayed": false, "notice": systemStatusNotice})
}

func (s *Server) deleteSystemStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision        int64 `json:"revision"`
		DocumentVersion int   `json:"document_version"`
		Confirm         bool  `json:"confirm"`
	}
	if !systemStatusHuman(current(r)) {
		apiError(w, 403, "카드 삭제는 일반 사용자 로그인에서만 가능합니다")
		return
	}
	if decode(r, &in) != nil || !systemStatusRevision(in.Revision, false) || in.DocumentVersion < 1 || !in.Confirm {
		apiError(w, 400, "현재 버전과 카드·관측 이력 삭제 확인이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		systemStatusError(w, e)
		return
	}
	defer tx.Rollback(r.Context())
	id := r.PathValue("id")
	wid, version, e := systemStatusDocumentTx(r, tx, id, true)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	c, e := s.systemStatusCardTx(r.Context(), tx, id, true)
	if e != nil {
		systemStatusError(w, e)
		return
	}
	if c.Revision != in.Revision || version != in.DocumentVersion {
		apiError(w, 409, "문서 또는 카드가 변경되었습니다. 삭제 대상을 다시 확인하세요")
		return
	}
	if e = s.systemStatusActorTx(r, tx, wid, id, true); e != nil {
		apiError(w, 403, "현재 문서·세션 권한을 다시 확인하세요")
		return
	}
	_, e = tx.Exec(r.Context(), `DELETE FROM system_status_cards WHERE document_id=$1`, id)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		systemStatusError(w, e)
		return
	}
	s.audit(r, "SYSTEM_STATUS_DELETED", id, map[string]any{"revision": c.Revision, "reports": c.ReportRevision})
	jsonResponse(w, 200, map[string]any{"deleted": true, "notice": "운영 카드와 관측 이력을 삭제했습니다. 문서 원문은 유지됩니다. 백업이 없다면 삭제된 카드·보고를 복구할 수 없습니다."})
}
