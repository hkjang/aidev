package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Lifecycle automation is independent of approval and opt-in per workspace.
// A disabled/removed owner cannot authorize automated edits of their documents.
func (s *Server) StartKnowledgeLifecycle(ctx context.Context) {
	go func() {
		timer := time.NewTicker(time.Hour)
		defer timer.Stop()
		for {
			run, cancel := context.WithTimeout(ctx, 5*time.Minute)
			rows, e := s.DB.Query(run, "SELECT workspace_id::text FROM workspace_settings WHERE data->'lifecycle_enabled'='true'::jsonb ORDER BY workspace_id")
			workspaces := []string{}
			if e == nil {
				for rows.Next() {
					var id string
					if e = rows.Scan(&id); e != nil {
						break
					}
					workspaces = append(workspaces, id)
				}
				if e == nil {
					e = rows.Err()
				}
				rows.Close()
			}
			if e == nil {
				for _, id := range workspaces {
					if _, e = s.runWorkspaceLifecycle(run, id); e != nil {
						break
					}
				}
			}
			cancel()
			if e != nil && ctx.Err() == nil {
				slog.Error("문서 생명주기 점검 실패", "error", e)
			}
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()
}
func (s *Server) runWorkspaceLifecycle(ctx context.Context, wid string) (int, error) {
	cfg, e := s.effectiveSettings(ctx, wid)
	if e != nil {
		return 0, e
	}
	if !boolean(cfg, "lifecycle_enabled") {
		return 0, nil
	}
	days := number(cfg, "review_period_days", 90)
	if days < 1 || days > 3650 {
		return 0, nil
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	var allowed bool
	if e = tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock_shared(726234801)").Scan(&allowed); e != nil || !allowed {
		return 0, e
	}
	if e = tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock(hashtextextended($1,41))", wid).Scan(&allowed); e != nil || !allowed {
		return 0, e
	}
	rows, e := tx.Query(ctx, `SELECT d.id::text,d.title,d.version,u.id::text,u.name,u.email,u.role FROM documents d JOIN users u ON u.id=d.owner_id LEFT JOIN knowledge_document_meta k ON k.document_id=d.id WHERE d.workspace_id=$1 AND d.deleted_at IS NULL AND d.status='published' AND coalesce(k.last_reviewed_at,d.updated_at)<now()-make_interval(days=>coalesce(k.review_period_days,$2)) AND madi_document_allowed(d.owner_id,d.id,true) AND EXISTS(SELECT 1 FROM workspace_settings ws WHERE ws.workspace_id=d.workspace_id AND ws.data->'lifecycle_enabled'='true'::jsonb) ORDER BY d.updated_at,d.id LIMIT 100 FOR UPDATE OF d SKIP LOCKED`, wid, days)
	if e != nil {
		return 0, e
	}
	type item struct {
		id, title string
		version   int
		owner     Principal
	}
	items := []item{}
	for rows.Next() {
		var d item
		e = rows.Scan(&d.id, &d.title, &d.version, &d.owner.ID, &d.owner.Name, &d.owner.Email, &d.owner.Role)
		if e != nil {
			break
		}
		d.owner.Kind = "user"
		items = append(items, d)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		return 0, e
	}
	for _, d := range items {
		_, e = tx.Exec(ctx, `INSERT INTO knowledge_document_meta(document_id,classification,last_reviewed_at) SELECT d.id,coalesce(sp.classification,'internal'),d.updated_at FROM documents d LEFT JOIN spaces sp ON sp.id=d.space_id WHERE d.id=$1 ON CONFLICT(document_id) DO UPDATE SET last_reviewed_at=coalesce(knowledge_document_meta.last_reviewed_at,excluded.last_reviewed_at)`, d.id)
		if e != nil {
			return 0, e
		}
		_, e = tx.Exec(ctx, "UPDATE documents SET status='stale',version=version+1,updated_at=now() WHERE id=$1", d.id)
		if e != nil {
			return 0, e
		}
		_, e = tx.Exec(ctx, "INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata) SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1", d.id, d.owner.ID)
		if e != nil {
			return 0, e
		}
		// Separate IDs are generated in Go; no optional PostgreSQL extension is required.
		recipients, e := tx.Query(ctx, `SELECT DISTINCT uid::text FROM (SELECT owner_id uid FROM documents WHERE id=$1 UNION SELECT reviewer_id FROM knowledge_document_meta WHERE document_id=$1 UNION SELECT maintainer_id FROM knowledge_document_meta WHERE document_id=$1) p WHERE uid IS NOT NULL AND madi_document_allowed(uid,$1,false)`, d.id)
		if e != nil {
			return 0, e
		}
		ids := []string{}
		for recipients.Next() {
			var uid string
			if e = recipients.Scan(&uid); e != nil {
				break
			}
			ids = append(ids, uid)
		}
		if e == nil {
			e = recipients.Err()
		}
		recipients.Close()
		if e != nil {
			return 0, e
		}
		for _, uid := range ids {
			_, e = tx.Exec(ctx, "INSERT INTO notifications(id,user_id,title,document_id) VALUES($1,$2,$3,$4)", newID(), uid, "문서 검토 주기가 지났습니다: "+d.title, d.id)
			if e != nil {
				return 0, e
			}
		}
		actorCtx := context.WithValue(ctx, principalKey, &d.owner)
		e = s.enqueueEvent(actorCtx, tx, Event{Type: "document.status_changed", WorkspaceID: wid, ResourceID: d.id, Before: map[string]any{"id": d.id, "status": "published", "version": d.version}, After: map[string]any{"id": d.id, "status": "stale", "version": d.version + 1, "reason": "review_period_elapsed"}})
		if e != nil {
			return 0, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return 0, e
	}
	for _, d := range items {
		r, _ := http.NewRequestWithContext(ctx, "POST", "http://madi.internal/knowledge/lifecycle", nil)
		s.audit(r, "DOCUMENT_LIFECYCLE_STALE", d.id, map[string]any{"workspace_id": wid, "owner_id": d.owner.ID})
	}
	return len(items), nil
}
