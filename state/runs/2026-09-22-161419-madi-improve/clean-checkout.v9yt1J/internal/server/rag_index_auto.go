package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

// Dispatch is database-serialized per document. The worker independently checks
// every grant again; merely changing a document does not authorize a new provider.
func (s *Server) StartRAGIndex(ctx context.Context) {
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if e := s.dispatchRAGReindex(ctx); e != nil && ctx.Err() == nil {
					slog.Error("RAG 자동 색인 예약 실패", "error_type", "database")
				}
			}
		}
	}()
}

func (s *Server) dispatchRAGReindex(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, `SELECT document_id::text FROM rag_reindex_queue WHERE enqueued_at<now()-interval '2 seconds' ORDER BY enqueued_at LIMIT 20`)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, id := range ids {
		if e = s.dispatchRAGDocument(ctx, id); e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
	}
	return nil
}

func (s *Server) dispatchRAGDocument(ctx context.Context, id string) error {
	initial, e := ragGrantTx(ctx, s.DB, id, false)
	if e != nil {
		return e
	}
	p, actorErr := s.ragCurrentActor(ctx, initial)
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var version int
	e = tx.QueryRow(ctx, `SELECT version FROM documents WHERE id=$1 AND deleted_at IS NULL FOR SHARE`, id).Scan(&version)
	if e != nil {
		return e
	}
	g, e := ragGrantTx(ctx, tx, id, true)
	if e != nil {
		return e
	}
	var queued string
	e = tx.QueryRow(ctx, `SELECT document_id::text FROM rag_reindex_queue WHERE document_id=$1 FOR UPDATE SKIP LOCKED`, id).Scan(&queued)
	if e != nil {
		return e
	}
	if g.Revision != initial.Revision {
		return nil
	}
	cfg, e := s.ragGrantSettingsTx(ctx, tx, g)
	if e != nil {
		return e
	}
	valid := g.Active && g.Auto && actorErr == nil && boolean(cfg, "rag_enabled") && ragProviderFingerprint(cfg) == g.Provider
	if valid {
		valid = ragActorTx(ctx, tx, p, id, g.WorkspaceID, true) == nil
	}
	if !valid {
		_, e = tx.Exec(ctx, `UPDATE rag_index_grants SET auto_reindex=false,updated_at=now() WHERE id=$1`, g.ID)
		if e == nil {
			_, e = tx.Exec(ctx, `DELETE FROM rag_reindex_queue WHERE document_id=$1`, id)
		}
		if e == nil {
			e = tx.Commit(ctx)
		}
		return e
	}
	if g.JobID != "" {
		var active bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM automation_jobs WHERE id=$1 AND status IN ('pending','running') AND NOT cancel_requested)`, g.JobID).Scan(&active)
		if e != nil {
			return e
		}
		if active {
			if version != g.Version {
				_, e = tx.Exec(ctx, `UPDATE automation_jobs SET cancel_requested=true,status=CASE WHEN status='pending' THEN 'cancelled' ELSE status END WHERE id=$1`, g.JobID)
			}
			if e == nil {
				e = tx.Commit(ctx)
			}
			return e
		}
	}
	// Repeated saves of the already-indexed version are idempotent.
	if version == g.Version {
		var ready bool
		e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM rag_vector_indexes WHERE grant_id=$1 AND document_version=$2 AND grant_revision=$3 AND status='ready')`, g.ID, version, g.Revision).Scan(&ready)
		if e != nil {
			return e
		}
		if ready {
			_, e = tx.Exec(ctx, `DELETE FROM rag_reindex_queue WHERE document_id=$1`, id)
			if e == nil {
				e = tx.Commit(ctx)
			}
			return e
		}
	}
	var pending int
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('madi:rag:actor:' || $1,0))`, g.ActorID); e != nil {
		return e
	}
	e = tx.QueryRow(ctx, `SELECT count(*) FROM automation_jobs WHERE actor_id=$1 AND kind='rag.index' AND status IN ('pending','running')`, g.ActorID).Scan(&pending)
	if e != nil {
		return e
	}
	if pending >= 10 {
		return nil
	}
	g.Version = version
	jobCtx := context.WithValue(ctx, jobContextKey{}, jobContext{ActorID: g.ActorID, TokenID: g.TokenID, Constraints: g.Constraints})
	jobID, e := s.EnqueueJob(jobCtx, tx, "rag.index", g.ActorID, g.WorkspaceID, map[string]any{"grant_id": g.ID, "document_id": id, "document_version": version, "grant_revision": g.Revision, "provider_fingerprint": g.Provider})
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE automation_jobs SET timeout_seconds=1800,max_attempts=3,resource_id=$2 WHERE id=$1`, jobID, id)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `UPDATE rag_index_grants SET expected_version=$2,last_job_id=$3,updated_at=now() WHERE id=$1`, g.ID, version, jobID)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM rag_vector_indexes WHERE grant_id=$1`, g.ID)
	}
	if e == nil {
		_, e = tx.Exec(ctx, `DELETE FROM rag_reindex_queue WHERE document_id=$1`, id)
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	return e
}
