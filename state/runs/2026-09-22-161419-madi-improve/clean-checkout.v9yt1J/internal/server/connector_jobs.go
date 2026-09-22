package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) queueConnector(w http.ResponseWriter, r *http.Request) {
	c, e := s.loadConnector(r.Context(), r.PathValue("id"))
	if e != nil || !s.connectorManage(r, c) {
		apiError(w, 404, "관리할 연결을 찾을 수 없습니다")
		return
	}
	var in struct {
		Rewind bool `json:"rewind"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "실행 옵션을 확인하세요")
		return
	}
	preview := strings.HasSuffix(r.URL.Path, "/preview")
	if !preview && !c.Enabled {
		apiError(w, 400, "연결을 활성화한 후 실행하세요")
		return
	}
	if _, e = s.connectorHTTP(r.Context(), c); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	id, e := s.enqueueConnector(r.Context(), tx, c, preview, in.Rewind)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		apiError(w, 409, e.Error())
		return
	}
	s.audit(r, "CONNECTOR_RUN", c.ID, map[string]any{"job_id": id, "preview": preview, "rewind": in.Rewind})
	respond(w, map[string]any{"job_id": id}, nil)
}

func (s *Server) enqueueConnector(ctx context.Context, tx pgx.Tx, c connectorConfig, preview, rewind bool) (string, error) {
	var revision int
	var enabled bool
	if e := tx.QueryRow(ctx, "SELECT revision,enabled FROM connector_configs WHERE id=$1 FOR UPDATE", c.ID).Scan(&revision, &enabled); e != nil {
		return "", e
	}
	if revision != c.Revision || (!preview && !enabled) {
		return "", errors.New("연결 설정이 변경되었습니다. 새로 불러오세요")
	}
	var busy bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM connector_runs r JOIN automation_jobs j ON j.id=r.job_id WHERE r.connector_id=$1 AND j.status IN ('pending','running'))`, c.ID).Scan(&busy); e != nil {
		return "", e
	}
	if busy {
		return "", errors.New("이 연결의 실행 중인 작업이 있습니다")
	}
	kind := "connector.sync"
	if preview {
		kind = "connector.preview"
	}
	id, e := s.EnqueueJob(ctx, tx, kind, c.ServiceID, c.WorkspaceID, map[string]any{"connector_id": c.ID, "revision": c.Revision})
	if e != nil {
		return "", e
	}
	cursor := c.Cursor
	if rewind || preview {
		cursor = ""
	}
	_, e = tx.Exec(ctx, `UPDATE automation_jobs SET timeout_seconds=3600,max_attempts=5 WHERE id=$1`, id)
	if e == nil {
		_, e = tx.Exec(ctx, `INSERT INTO connector_runs(job_id,connector_id,revision,cursor) VALUES($1,$2,$3,$4)`, id, c.ID, c.Revision, cursor)
	}
	return id, e
}

func (s *Server) connectorPrincipals(ctx context.Context, j Job, c connectorConfig) (*Principal, *Principal, error) {
	if c.Revision != number(j.Payload, "revision", 0) || c.ServiceID != j.OwnerID || c.WorkspaceID != j.WorkspaceID || (!c.Enabled && j.Kind != "connector.preview") {
		return nil, nil, jobPermanent("연결이 변경되거나 비활성화되었습니다")
	}
	owner, e := s.workerPrincipal(ctx, j.OwnerID, "", j.WorkspaceID)
	if e != nil {
		return nil, nil, e
	}
	actor, e := s.workerPrincipal(ctx, j.ActorID, j.TokenID, j.WorkspaceID)
	if e != nil {
		return nil, nil, e
	}
	if owner.Kind != "service" || actor.TokenID != "" || actor.ScopeRestricted || !s.automationManager(ctx, actor, c.WorkspaceID) || !s.canSpace(ctx, actor, c.WorkspaceID, c.SpaceID, true) || !s.canSpace(ctx, owner, c.WorkspaceID, c.SpaceID, true) {
		return nil, nil, jobPermanent("연결 관리자 또는 서비스 계정의 현재 공간 쓰기 권한이 없습니다")
	}
	return owner, actor, nil
}

// A page, its imported documents, outbox events and checkpoint commit together.
// A crash can repeat a GET, but never leave a skipped half-page or double-count it.
func (s *Server) executeConnectorJob(ctx context.Context, j Job) (map[string]any, error) {
	c, e := s.loadConnector(ctx, str(j.Payload, "connector_id"))
	if e != nil {
		return nil, jobPermanent("연결을 찾을 수 없습니다")
	}
	var cursor string
	var raw []byte
	if e = s.DB.QueryRow(ctx, "SELECT cursor,report FROM connector_runs WHERE job_id=$1 AND connector_id=$2 AND revision=$3", j.ID, c.ID, c.Revision).Scan(&cursor, &raw); e != nil {
		return nil, e
	}
	report := map[string]any{}
	if json.Unmarshal(raw, &report) != nil {
		return nil, jobPermanent("연결 체크포인트를 읽을 수 없습니다")
	}
	if boolean(report, "completed") {
		return report, nil
	}
	limit := number(c.Config, "max_items", 1000)
	seen := map[string]bool{}
	processed := 0
	for pageIndex := 0; pageIndex < 100; pageIndex++ {
		if ctx.Err() != nil {
			return report, ctx.Err()
		}
		if seen[cursor] {
			return report, jobPermanent("원격 API가 동일한 페이지 주소를 반복했습니다")
		}
		seen[cursor] = true
		c, e = s.loadConnector(ctx, c.ID)
		if e != nil {
			return report, e
		}
		owner, actor, e := s.connectorPrincipals(ctx, j, c)
		if e != nil {
			return report, e
		}
		page, e := s.fetchConnectorPage(ctx, c, cursor)
		if e != nil {
			return report, e
		}
		if len(page.Items) > 1000 || len(page.Next) > 16384 {
			return report, jobPermanent("원격 페이지 또는 체크포인트가 허용 크기를 초과했습니다")
		}
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return report, e
		}
		applyErr := func() error {
			var revision int
			var enabled bool
			if e := tx.QueryRow(ctx, "SELECT revision,enabled FROM connector_configs WHERE id=$1 FOR SHARE", c.ID).Scan(&revision, &enabled); e != nil {
				return e
			}
			if revision != c.Revision || (!enabled && j.Kind != "connector.preview") {
				return jobPermanent("실행 중 연결 설정이 변경되었습니다")
			}
			var active bool
			if e := tx.QueryRow(ctx, `SELECT status='running' AND NOT cancel_requested AND lease_id=$2::uuid AND lease_until>now() FROM automation_jobs WHERE id=$1 FOR SHARE`, j.ID, j.LeaseID).Scan(&active); e != nil {
				return e
			}
			if !active {
				return jobPermanent("연결 작업이 취소되거나 임대가 만료되었습니다")
			}
			var policy bool
			if e := tx.QueryRow(ctx, "SELECT enabled FROM connector_settings WHERE id=1 FOR SHARE").Scan(&policy); e != nil {
				return e
			}
			if !policy {
				return jobPermanent("외부 연결 정책이 비활성화되었습니다")
			}
			if j.Kind == "connector.preview" {
				samples := []map[string]any{}
				for _, item := range page.Items {
					if len(samples) < 30 {
						samples = append(samples, map[string]any{"remote_id": item.ID, "title": item.Title, "source_url": item.URL, "characters": len([]rune(item.Markdown))})
					}
				}
				report["samples"] = samples
				report["preview_count"] = len(page.Items)
				report["has_more"] = page.Next != ""
				report["acl_notice"] = "원격 권한을 복사하지 않고 지정한 대상 공간 ACL을 적용합니다"
			} else {
				for _, item := range page.Items {
					outcome, e := s.applyConnectorItem(ctx, tx, c, owner, actor, item)
					if e != nil {
						return e
					}
					report[outcome] = number(report, outcome, 0) + 1
				}
			}
			warnings := listStrings(report["warnings"])
			for _, warning := range page.Warnings {
				if len(warnings) < 50 {
					warnings = append(warnings, connectorTrim(warning, 500))
				}
			}
			report["warnings"] = warnings
			report["pages"] = number(report, "pages", 0) + 1
			processed += len(page.Items)
			done := j.Kind == "connector.preview" || page.Next == "" || processed >= limit || pageIndex == 99
			report["completed"] = done
			report["has_more"] = page.Next != ""
			report["processed"] = number(report, "processed", 0) + len(page.Items)
			if _, e := tx.Exec(ctx, "UPDATE connector_runs SET cursor=$2,report=$3,updated_at=now() WHERE job_id=$1", j.ID, page.Next, jsonValue(report)); e != nil {
				return e
			}
			if j.Kind != "connector.preview" {
				if _, e := tx.Exec(ctx, "UPDATE connector_configs SET cursor=$2,updated_at=now() WHERE id=$1", c.ID, page.Next); e != nil {
					return e
				}
			}
			return tx.Commit(ctx)
		}()
		_ = tx.Rollback(ctx)
		if applyErr != nil {
			return report, applyErr
		}
		if boolean(report, "completed") {
			return report, nil
		}
		cursor = page.Next
	}
	return report, nil
}

func (s *Server) applyConnectorItem(ctx context.Context, tx pgx.Tx, c connectorConfig, owner, actor *Principal, item connectorItem) (string, error) {
	item, e := connectorPlainItem(item.ID, item.Title, item.Markdown, item.URL)
	if e != nil {
		return "", jobPermanent(e.Error())
	}
	md := item.Markdown
	if item.URL != "" {
		md += "\n\n---\n원본: [" + c.Kind + "](<" + strings.ReplaceAll(strings.ReplaceAll(item.URL, "<", "%3C"), ">", "%3E") + ">)\n"
	}
	if len(md) > 4<<20 {
		return "", jobPermanent("가져올 문서가 4MB를 초과했습니다")
	}
	digest := sha256.Sum256([]byte(item.Title + "\x00" + md))
	checksum := hex.EncodeToString(digest[:])
	var id, previousChecksum string
	var previousVersion int
	e = tx.QueryRow(ctx, `SELECT coalesce(document_id::text,''),checksum,document_version FROM connector_records WHERE connector_id=$1 AND remote_id=$2 FOR UPDATE`, c.ID, item.ID).Scan(&id, &previousChecksum, &previousVersion)
	creating := errors.Is(e, pgx.ErrNoRows)
	if e != nil && !creating {
		return "", e
	}
	if !creating {
		if _, e = tx.Exec(ctx, "UPDATE connector_records SET last_seen_at=now() WHERE connector_id=$1 AND remote_id=$2", c.ID, item.ID); e != nil {
			return "", e
		}
		if id == "" {
			return "conflicts", nil
		}
	}
	var version int
	var oldStatus, oldTitle string
	if !creating {
		var deleted bool
		var allowed bool
		e = tx.QueryRow(ctx, `SELECT version,title,status,deleted_at IS NOT NULL,madi_document_allowed($2,id,true) AND madi_document_allowed($3,id,true) FROM documents WHERE id=$1 FOR UPDATE`, id, owner.ID, actor.ID).Scan(&version, &oldTitle, &oldStatus, &deleted, &allowed)
		if e != nil {
			return "", e
		}
		if deleted || !allowed {
			return "conflicts", nil
		}
		if previousChecksum == checksum {
			return "unchanged", nil
		}
		if version != previousVersion && str(c.Config, "conflict_policy") != "replace_local" {
			return "conflicts", nil
		}
	}
	var allowed bool
	e = tx.QueryRow(ctx, `SELECT madi_space_allowed($1,NULLIF($3,'')::uuid,true) AND madi_space_allowed($2,NULLIF($3,'')::uuid,true)`, owner.ID, actor.ID, c.SpaceID).Scan(&allowed)
	if e != nil {
		return "", e
	}
	if !allowed {
		return "", jobPermanent("가져오는 중 대상 공간 권한이 변경되었습니다")
	}
	if creating {
		id = newID()
	}
	protected, e := s.ProtectDocumentTx(ctx, tx, owner, id, c.WorkspaceID, item.Title, md)
	if e != nil {
		return "", e
	}
	item.Title, md = protected.Title, protected.Markdown
	protectedSource, e := s.ProtectDocumentMetadataTx(ctx, tx, owner, id, c.WorkspaceID, map[string]any{"source_url": item.URL})
	if e != nil {
		return "", e
	}
	if protectedSource.Changed {
		item.URL = str(protectedSource.Value.(map[string]any), "source_url")
	}
	if creating {
		version = 1
		_, e = tx.Exec(ctx, `INSERT INTO documents(id,workspace_id,space_id,title,markdown,tags,owner_id,visibility,status) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,'workspace','draft')`, id, c.WorkspaceID, c.SpaceID, item.Title, md, jsonValue([]string{"connector", c.Kind}), owner.ID)
	} else {
		version++
		_, e = tx.Exec(ctx, `UPDATE documents SET title=$2,markdown=$3,status='draft',block_metadata='{}',version=$4,updated_at=now() WHERE id=$1`, id, item.Title, md, version)
	}
	if e != nil {
		return "", e
	}
	_, e = tx.Exec(ctx, `INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1`, id, owner.ID)
	if e != nil {
		return "", e
	}
	kind := "document.updated"
	outcome := "updated"
	if creating {
		kind = "document.created"
		outcome = "created"
	}
	eventCtx := context.WithValue(ctx, principalKey, owner)
	after := map[string]any{"id": id, "title": item.Title, "status": "draft", "version": version}
	before := map[string]any{"id": id, "title": oldTitle, "status": oldStatus, "version": version - 1}
	if e = s.enqueueEvent(eventCtx, tx, Event{Type: kind, WorkspaceID: c.WorkspaceID, ResourceID: id, Before: before, After: after}); e != nil {
		return "", e
	}
	if !creating && oldStatus != "draft" {
		if e = s.enqueueEvent(eventCtx, tx, Event{Type: "document.status_changed", WorkspaceID: c.WorkspaceID, ResourceID: id, Before: before, After: after}); e != nil {
			return "", e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO connector_records(connector_id,remote_id,document_id,source_url,checksum,document_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(connector_id,remote_id) DO UPDATE SET source_url=excluded.source_url,checksum=excluded.checksum,document_version=excluded.document_version,last_seen_at=now()`, c.ID, item.ID, id, item.URL, checksum, version)
	return outcome, e
}

func (s *Server) StartConnectors(ctx context.Context) {
	go func() {
		s.scheduleConnectors(ctx)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.scheduleConnectors(ctx)
			}
		}
	}()
}
func (s *Server) scheduleConnectors(ctx context.Context) {
	for count := 0; count < 20; count++ {
		tx, e := s.DB.Begin(ctx)
		if e != nil {
			return
		}
		var id string
		e = tx.QueryRow(ctx, `SELECT id::text FROM connector_configs WHERE enabled AND interval_minutes>0 AND next_run<=now() AND (SELECT enabled FROM connector_settings WHERE id=1) AND NOT(SELECT paused FROM job_settings WHERE id=1) ORDER BY next_run FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&id)
		if e != nil {
			_ = tx.Rollback(ctx)
			return
		}
		c, e := s.loadConnector(ctx, id)
		if e != nil {
			_ = tx.Rollback(ctx)
			return
		}
		owner, e := s.workerPrincipal(ctx, c.OwnerID, "", c.WorkspaceID)
		if e == nil {
			_, e = s.enqueueConnector(context.WithValue(ctx, principalKey, owner), tx, c, false, false)
		}
		// A busy or revoked connection advances its schedule; job history explains
		// actual attempted runs and an administrator can explicitly retry after repair.
		_, advanceErr := tx.Exec(ctx, "UPDATE connector_configs SET next_run=now()+make_interval(mins=>interval_minutes) WHERE id=$1", id)
		if advanceErr == nil {
			advanceErr = tx.Commit(ctx)
		}
		_ = tx.Rollback(ctx)
		if advanceErr != nil {
			return
		}
	}
}
