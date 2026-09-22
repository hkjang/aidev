package server

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed search_index.sql
var searchIndexSchema string

func (s *Server) migrateSearchIndex(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, searchIndexSchema)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec(ctx, searchOperationsSchema)
	return e
}
func (s *Server) registerSearchIndex() {
	s.registerSearchOperations()
	s.handle("GET /api/v1/search", s.universalSearch)
	s.handle("GET /api/v1/search/index-status", s.searchIndexStatus)
	s.handle("POST /api/v1/search/reindex", s.requestSearchIndex)
	s.handle("GET /api/v1/documents/{id}/citation", s.getCitation)
}

// Local, derived PostgreSQL projection only. No content is sent to an AI
// provider by this worker. Every writer (REST, CRDT, import, connector, restore)
// enters the same transactional trigger queue.
func (s *Server) StartSearchIndex(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		cursor := ""
		backfillDone := false
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !backfillDone {
					next, done, e := s.searchIndexBackfill(ctx, cursor)
					if e != nil {
						slog.Error("search index backfill failed", "error", e)
					} else {
						cursor = next
						backfillDone = done
					}
				}
				for i := 0; i < 16; i++ {
					worked, e := s.indexSearchNext(ctx)
					if e != nil {
						slog.Error("search index failed", "error", e)
						break
					}
					if !worked {
						break
					}
				}
			}
		}
	}()
}

func (s *Server) searchIndexBackfill(ctx context.Context, cursor string) (string, bool, error) {
	rows, e := s.DB.Query(ctx, `WITH batch AS (SELECT d.id FROM documents d WHERE ($1='' OR d.id>NULLIF($1,'')::uuid) ORDER BY d.id LIMIT 200), queued AS (INSERT INTO search_index_queue(document_id) SELECT d.id FROM batch b JOIN documents d ON d.id=b.id LEFT JOIN search_index_documents x ON x.document_id=d.id WHERE d.deleted_at IS NULL AND (x.document_id IS NULL OR x.document_version<>d.version OR NOT x.links_indexed OR NOT EXISTS(SELECT 1 FROM search_folded_documents z WHERE z.document_id=d.id AND z.document_version=d.version)) ON CONFLICT(document_id) DO NOTHING) SELECT id::text FROM batch ORDER BY id`, cursor)
	if e != nil {
		return cursor, false, e
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		if e = rows.Scan(&cursor); e != nil {
			return cursor, false, e
		}
		count++
	}
	return cursor, count < 200, rows.Err()
}

func (s *Server) indexSearchNext(ctx context.Context) (bool, error) {
	rows, e := s.DB.Query(ctx, "SELECT document_id::text FROM search_index_queue ORDER BY enqueued_at,document_id LIMIT 16")
	if e != nil {
		return false, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return false, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return false, e
	}
	for _, id := range ids {
		worked, e := s.indexSearchDocument(ctx, id)
		if e != nil {
			return false, e
		}
		if worked {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) indexSearchDocument(ctx context.Context, id string) (bool, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback(ctx)
	var markdown string
	var version int
	var deleted *time.Time
	// Match writer lock order: document first, then its queue row. Holding a
	// queue lock while waiting for the source row deadlocks with the trigger.
	e = tx.QueryRow(ctx, "SELECT markdown,version,deleted_at FROM documents WHERE id=$1 FOR SHARE", id).Scan(&markdown, &version, &deleted)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var queued string
	e = tx.QueryRow(ctx, "SELECT document_id::text FROM search_index_queue WHERE document_id=$1 FOR UPDATE SKIP LOCKED", id).Scan(&queued)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if deleted == nil {
		chunks, err := chunkMarkdown(markdown, 6144, 512)
		if err != nil {
			return false, err
		}
		if _, e = tx.Exec(ctx, "DELETE FROM search_chunks WHERE document_id=$1", id); e != nil {
			return false, e
		}
		if _, e = tx.Exec(ctx, "DELETE FROM search_fragments WHERE document_id=$1", id); e != nil {
			return false, e
		}
		fragments := projectSearchFragments(markdown)
		fragmentValues := make([][]any, 0, len(fragments))
		for i, f := range fragments {
			fragmentValues = append(fragmentValues, []any{id, i, version, f.Kind, f.Start, f.End, f.Line, f.Content, f.Metadata})
		}
		if len(fragmentValues) > 0 {
			_, e = tx.CopyFrom(ctx, pgx.Identifier{"search_fragments"}, []string{"document_id", "ordinal", "document_version", "kind", "start_byte", "end_byte", "start_line", "content", "metadata"}, pgx.CopyFromRows(fragmentValues))
			if e != nil {
				return false, e
			}
		}
		if e = s.writeFoldedProjectionTx(ctx, tx, id, version, markdown, fragments); e != nil {
			return false, e
		}
		values := make([][]any, 0, len(chunks))
		for _, c := range chunks {
			values = append(values, []any{id, c.Index, version, c.Hash, c.Start, c.End, c.StartLine, c.EndLine, c.Heading, c.Content})
		}
		if len(values) > 0 {
			_, e = tx.CopyFrom(ctx, pgx.Identifier{"search_chunks"}, []string{"document_id", "ordinal", "document_version", "content_hash", "start_byte", "end_byte", "start_line", "end_line", "heading", "content"}, pgx.CopyFromRows(values))
			if e != nil {
				return false, e
			}
		}
		links, sourceTags := projectGraphReferences(markdown)
		_, e = tx.Exec(ctx, `INSERT INTO search_index_documents(document_id,document_version,source_hash,chunk_count,links,source_tags,links_indexed) VALUES($1,$2,$3,$4,$5,$6,true) ON CONFLICT(document_id) DO UPDATE SET document_version=excluded.document_version,source_hash=excluded.source_hash,chunk_count=excluded.chunk_count,links=excluded.links,source_tags=excluded.source_tags,links_indexed=true,indexed_at=now()`, id, version, digest(markdown), len(chunks), jsonValue(links), jsonValue(sourceTags))
		if e != nil {
			return false, e
		}
	}
	if _, e = tx.Exec(ctx, "DELETE FROM search_index_queue WHERE document_id=$1", id); e != nil {
		return false, e
	}
	e = tx.Commit(ctx)
	return e == nil, e
}

func (s *Server) searchIndexStatus(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !s.canWorkspace(r.Context(), current(r), wid, false) {
		apiError(w, 403, "워크스페이스 접근 권한이 없습니다")
		return
	}
	v, e := s.one(r.Context(), `SELECT jsonb_build_object('documents',count(*),'indexed',count(*) FILTER(WHERE x.document_version=d.version),'pending',count(*) FILTER(WHERE x.document_version IS DISTINCT FROM d.version),'normalized',count(*) FILTER(WHERE z.document_version=d.version),'normalized_pending',count(*) FILTER(WHERE z.document_version IS DISTINCT FROM d.version),'chunks',coalesce(sum(x.chunk_count) FILTER(WHERE x.document_version=d.version),0),'scope','현재 열람 권한의 로컬 PostgreSQL 색인; 외부 임베딩 전송 없음') FROM documents d LEFT JOIN search_index_documents x ON x.document_id=d.id LEFT JOIN search_folded_documents z ON z.document_id=d.id WHERE d.workspace_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($1,d.id,false)`, current(r).ID, wid)
	respond(w, v, e)
}
func (s *Server) requestSearchIndex(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Document string `json:"document_id"`
	}
	if decode(r, &in) != nil || !s.canDocument(r.Context(), current(r), in.Document, false) {
		apiError(w, 404, "색인할 문서에 접근할 수 없습니다")
		return
	}
	_, e := s.DB.Exec(r.Context(), `INSERT INTO search_index_queue(document_id) SELECT id FROM documents WHERE id=$1 AND deleted_at IS NULL AND madi_document_allowed($2,id,false) ON CONFLICT(document_id) DO NOTHING`, in.Document, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	jsonResponse(w, 202, map[string]any{"queued": true, "document_id": in.Document})
}
