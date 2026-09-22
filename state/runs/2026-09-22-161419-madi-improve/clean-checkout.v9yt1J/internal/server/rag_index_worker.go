package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *Server) ragJobGuard(ctx context.Context, j Job) (ragIndexGrant, *Principal, map[string]any, error) {
	if ctx.Err() != nil {
		return ragIndexGrant{}, nil, nil, ctx.Err()
	}
	g, e := ragGrantByIDTx(ctx, s.DB, str(j.Payload, "grant_id"), false)
	if e != nil || !g.Active || g.ID != str(j.Payload, "grant_id") || g.DocumentID != str(j.Payload, "document_id") || g.Revision != int64(number(j.Payload, "grant_revision", 0)) || g.Version != number(j.Payload, "document_version", 0) || g.Provider != str(j.Payload, "provider_fingerprint") || g.JobID != j.ID || g.ActorID != j.ActorID || g.TokenID != j.TokenID {
		return g, nil, nil, errRAGChanged
	}
	p, e := s.ragCurrentActor(ctx, g)
	if e != nil {
		return g, nil, nil, e
	}
	cfg, e := s.ragGrantSettings(ctx, g)
	if e != nil || !boolean(cfg, "rag_enabled") || g.Provider != ragProviderFingerprint(cfg) {
		return g, nil, nil, errRAGChanged
	}
	var active bool
	e = s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents d,automation_jobs j WHERE d.id=$1 AND d.version=$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,true) AND j.id=$4 AND NOT j.cancel_requested AND j.status IN ('pending','running'))`, g.DocumentID, g.Version, p.ID, j.ID).Scan(&active)
	if e != nil || !active {
		return g, nil, nil, errRAGChanged
	}
	return g, p, cfg, nil
}

func (s *Server) ragJobTx(ctx context.Context, tx pgx.Tx, j Job, g ragIndexGrant, p *Principal) (string, error) {
	var markdown string
	var version int
	e := tx.QueryRow(ctx, `SELECT markdown,version FROM documents WHERE id=$1 AND deleted_at IS NULL FOR SHARE`, g.DocumentID).Scan(&markdown, &version)
	if e != nil || version != g.Version {
		return "", errRAGChanged
	}
	if e = ragActorTx(ctx, tx, p, g.DocumentID, g.WorkspaceID, true); e != nil {
		return "", e
	}
	current, e := ragGrantByIDTx(ctx, tx, g.ID, true)
	if e != nil || !current.Active || current.Revision != g.Revision || current.JobID != j.ID || current.Provider != g.Provider || current.GenerationID != g.GenerationID {
		return "", errRAGChanged
	}
	cfg, e := s.ragGrantSettingsTx(ctx, tx, g)
	if e != nil || !boolean(cfg, "rag_enabled") || ragProviderFingerprint(cfg) != g.Provider {
		return "", errRAGChanged
	}
	var active bool
	e = tx.QueryRow(ctx, `SELECT NOT cancel_requested AND status IN ('pending','running') FROM automation_jobs WHERE id=$1 FOR SHARE`, j.ID).Scan(&active)
	if e != nil || !active {
		return "", errRAGChanged
	}
	return markdown, nil
}

func (s *Server) runRAGIndex(ctx context.Context, j Job) (map[string]any, error) {
	fail := func(e error) (map[string]any, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errors.Is(e, errRAGChanged) {
			return nil, jobPermanent(errRAGChanged.Error())
		}
		return nil, e
	}
	g, p, cfg, e := s.ragJobGuard(ctx, j)
	if e != nil {
		return fail(e)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return fail(e)
	}
	markdown, e := s.ragJobTx(ctx, tx, j, g, p)
	if e != nil {
		tx.Rollback(ctx)
		return fail(e)
	}
	chunks, e := chunkMarkdown(markdown, 6144, 512)
	if e != nil || len(chunks) == 0 {
		tx.Rollback(ctx)
		return nil, jobPermanent("색인 가능한 문서 본문이 없습니다")
	}
	_, e = tx.Exec(ctx, `INSERT INTO rag_vector_indexes(id,grant_id,document_id,document_version,grant_revision,provider_fingerprint,total_chunks) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, j.ID, g.ID, g.DocumentID, g.Version, g.Revision, g.Provider, len(chunks))
	var completed, dimensions int
	var state string
	if e == nil {
		e = tx.QueryRow(ctx, `SELECT indexed_chunks,dimensions,status FROM rag_vector_indexes WHERE id=$1 AND grant_revision=$2 AND document_version=$3 FOR UPDATE`, j.ID, g.Revision, g.Version).Scan(&completed, &dimensions, &state)
	}
	if e == nil {
		e = tx.Commit(ctx)
	} else {
		tx.Rollback(ctx)
	}
	if e != nil {
		return fail(e)
	}
	if state == "ready" {
		return map[string]any{"indexed_chunks": completed, "dimensions": dimensions, "document_version": g.Version}, nil
	}
	if completed < 0 || completed > len(chunks) {
		return nil, jobPermanent("색인 체크포인트가 일치하지 않습니다. 새 색인을 요청하세요")
	}
	for start := completed; start < len(chunks); {
		g, p, cfg, e = s.ragJobGuard(ctx, j)
		if e != nil {
			return fail(e)
		}
		end := min(start+16, len(chunks))
		inputs := []string{}
		for _, c := range chunks[start:end] {
			inputs = append(inputs, c.Content)
		}
		watched, stop, guardErr := ragWatch(ctx, func(check context.Context) error { _, _, _, e := s.ragJobGuard(check, j); return e })
		vectors, providerErr := ragEmbeddings(watched, ragEmbeddingProvider(cfg), inputs)
		stop()
		if *guardErr != nil {
			return fail(*guardErr)
		}
		if providerErr != nil {
			return fail(providerErr)
		}
		g, p, _, e = s.ragJobGuard(ctx, j)
		if e != nil {
			return fail(e)
		}
		if dimensions != 0 && len(vectors[0]) != dimensions {
			return nil, jobPermanent("공급자의 벡터 차원이 작업 도중 변경되었습니다")
		}
		dimensions = len(vectors[0])
		if expected := number(cfg, "rag_generation_dimensions", 0); expected > 0 && dimensions != expected {
			return nil, jobPermanent("세대에 고정한 벡터 차원과 공급자 응답이 다릅니다")
		}
		tx, e = s.DB.Begin(ctx)
		if e != nil {
			return fail(e)
		}
		if _, e = s.ragJobTx(ctx, tx, j, g, p); e == nil {
			for i, c := range chunks[start:end] {
				_, e = tx.Exec(ctx, `INSERT INTO rag_vector_chunks(index_id,ordinal,content_hash,start_byte,end_byte,start_line,end_line,heading,embedding,generation_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid) ON CONFLICT(index_id,ordinal) DO UPDATE SET embedding=EXCLUDED.embedding WHERE rag_vector_chunks.content_hash=EXCLUDED.content_hash`, j.ID, c.Index, c.Hash, c.Start, c.End, c.StartLine, c.EndLine, c.Heading, vectors[i], g.GenerationID)
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			_, e = tx.Exec(ctx, `UPDATE rag_vector_indexes SET indexed_chunks=$2,dimensions=$3,updated_at=now() WHERE id=$1`, j.ID, end, dimensions)
		}
		if e == nil {
			e = tx.Commit(ctx)
		} else {
			tx.Rollback(ctx)
		}
		if e != nil {
			return fail(e)
		}
		start = end
	}
	g, p, _, e = s.ragJobGuard(ctx, j)
	if e != nil {
		return fail(e)
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		return fail(e)
	}
	defer tx.Rollback(ctx)
	if _, e = s.ragJobTx(ctx, tx, j, g, p); e != nil {
		return fail(e)
	}
	var count int
	e = tx.QueryRow(ctx, `SELECT count(*) FROM rag_vector_chunks WHERE index_id=$1 AND cardinality(embedding)=$2`, j.ID, dimensions).Scan(&count)
	if e != nil {
		return fail(e)
	}
	if count != len(chunks) {
		return nil, fmt.Errorf("색인 조각 수가 일치하지 않습니다")
	}
	_, e = tx.Exec(ctx, `UPDATE rag_vector_indexes SET status='ready',updated_at=now() WHERE id=$1`, j.ID)
	if e == nil {
		e = tx.Commit(ctx)
	}
	if e != nil {
		return fail(e)
	}
	return map[string]any{"indexed_chunks": count, "dimensions": dimensions, "document_version": g.Version, "provider_fingerprint": g.Provider}, nil
}
