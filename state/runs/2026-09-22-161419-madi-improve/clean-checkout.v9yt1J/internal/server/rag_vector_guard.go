package server

import (
	"context"
	"strconv"
)

// Semantic/reranked sources carry server-only grant identities through chat SSE.
// Revocation invalidates the in-flight answer even when the reader can still see
// the source document. Pure keyword sources have no external indexing grant.
func (s *Server) validateRAGSourceGrants(ctx context.Context, sources []aiSource, cfg map[string]any) error {
	checked := map[string]int64{}
	for _, source := range sources {
		if source.RAGGrantID == "" {
			continue
		}
		if !boolean(cfg, "rag_enabled") {
			return errRAGChanged
		}
		key := source.RAGGrantID + ":" + source.ID + ":" + strconv.Itoa(source.Version)
		if revision, ok := checked[key]; ok {
			if revision != source.RAGGrantRevision {
				return errRAGChanged
			}
			continue
		}
		g, e := ragGrantTx(ctx, s.DB, source.ID, false)
		if e != nil || g.ID != source.RAGGrantID || !g.Active || g.Revision != source.RAGGrantRevision || g.Version != source.Version || g.Provider != ragProviderFingerprint(cfg) {
			return errRAGChanged
		}
		if _, e = s.ragCurrentActor(ctx, g); e != nil {
			return errRAGChanged
		}
		var current bool
		if s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM documents WHERE id=$1 AND workspace_id=$2 AND version=$3 AND deleted_at IS NULL)`, source.ID, g.WorkspaceID, source.Version).Scan(&current) != nil || !current {
			return errRAGChanged
		}
		checked[key] = g.Revision
	}
	return nil
}

// A question embedding is not sent unless at least one currently authorized
// grant can answer it. Disabled actors/revoked keys/plugins are checked before
// contacting the provider, not merely after a vector query has completed.
func (s *Server) ragQuerySource(ctx context.Context, p *Principal, wid, docID string, cfg map[string]any) (*aiSource, error) {
	rows, e := s.DB.Query(ctx, `SELECT d.id::text,d.version,g.id::text,g.revision FROM rag_vector_indexes i JOIN rag_index_grants g ON g.id=i.grant_id JOIN documents d ON d.id=i.document_id JOIN users owner ON owner.id=g.actor_id WHERE d.workspace_id=$1 AND ($2='' OR d.id::text=$2) AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false) AND i.document_version=d.version AND i.status='ready' AND g.active AND g.generation_id IS NOT DISTINCT FROM (SELECT active_id FROM rag_generation_state WHERE workspace_id=g.workspace_id) AND g.revision=i.grant_revision AND g.provider_fingerprint=$4 AND NOT owner.disabled AND madi_document_allowed(owner.id,d.id,true) ORDER BY i.updated_at DESC LIMIT 200`, wid, docID, p.ID, ragProviderFingerprint(cfg))
	if e != nil {
		return nil, e
	}
	candidates := []aiSource{}
	for rows.Next() {
		var source aiSource
		if e = rows.Scan(&source.ID, &source.Version, &source.RAGGrantID, &source.RAGGrantRevision); e != nil {
			rows.Close()
			return nil, e
		}
		candidates = append(candidates, source)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for _, source := range candidates {
		if s.validateRAGSourceGrants(ctx, []aiSource{source}, cfg) == nil {
			return &source, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, nil
}
