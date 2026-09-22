package server

import "fmt"

// Keep the empty-search path free of optional OR predicates and CASE sorting so
// PostgreSQL can use (workspace_id, updated_at DESC). Values remain parameters;
// both branches retain the exact same current-document ACL predicate.
func documentListQuery(actor, workspaceID, term, tag string, trash, favorite bool, limit, offset int) (string, []any) {
	args := []any{actor}
	bind := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	query := "SELECT " + docSummaryJSON + " FROM documents d WHERE " + docACL
	if workspaceID != "" {
		query += " AND d.workspace_id=" + bind(workspaceID) + "::uuid"
	}
	if trash {
		query += " AND d.deleted_at IS NOT NULL"
	} else {
		query += " AND d.deleted_at IS NULL"
	}
	order := "d.updated_at DESC"
	if term != "" {
		q := bind(term)
		query += " AND (d.search_vector@@websearch_to_tsquery('simple'," + q + ") OR d.title ILIKE '%'||" + q + "||'%' OR d.markdown ILIKE '%'||" + q + "||'%' OR d.tags::text ILIKE '%'||" + q + "||'%')"
		order = "ts_rank_cd(d.search_vector,websearch_to_tsquery('simple'," + q + ")) DESC," + order
	}
	if tag != "" {
		query += " AND d.tags ? " + bind(tag)
	}
	if favorite {
		query += " AND EXISTS(SELECT 1 FROM favorites f WHERE f.document_id=d.id AND f.user_id=$1)"
	}
	query += " ORDER BY " + order + " LIMIT " + bind(limit) + " OFFSET " + bind(offset)
	return query, args
}

// Lists are a bounded navigation/read model, never a batch raw-document API.
// Keep actual alias/tag strings intact: clipping them could invent a different
// wiki-link alias. Detail GET preserves all original metadata and Markdown.
const documentSummaryTags = `(SELECT coalesce(jsonb_agg(v),'[]'::jsonb) FROM (SELECT v FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(d.tags)='array' THEN d.tags ELSE '[]'::jsonb END) t(v) WHERE octet_length(v)<=200 AND octet_length(to_jsonb(v)::text)<=256 LIMIT 32) bounded)`
const documentSummaryAliases = `(SELECT coalesce(jsonb_agg(v),'[]'::jsonb) FROM (SELECT v FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(d.aliases)='array' THEN d.aliases ELSE '[]'::jsonb END) t(v) WHERE octet_length(v)<=200 AND octet_length(to_jsonb(v)::text)<=256 LIMIT 32) bounded)`
const docSummaryJSON = `jsonb_build_object(
 'id',d.id,'workspace_id',d.workspace_id,'parent_id',d.parent_id,'space_id',d.space_id,
 'title',left(d.title,500),'icon',left(d.icon,64),'status',left(d.status,32),'visibility',left(d.visibility,32),
 'owner_id',d.owner_id,'version',d.version,'created_at',d.created_at,'updated_at',d.updated_at,'deleted_at',d.deleted_at,
 'excerpt',left(d.markdown,160),'tags',` + documentSummaryTags + `,'aliases',` + documentSummaryAliases + `,
 'tags_truncated',CASE WHEN jsonb_typeof(d.tags)='array' THEN jsonb_array_length(d.tags)>jsonb_array_length(` + documentSummaryTags + `) ELSE true END,
 'aliases_truncated',CASE WHEN jsonb_typeof(d.aliases)='array' THEN jsonb_array_length(d.aliases)>jsonb_array_length(` + documentSummaryAliases + `) ELSE true END,
 'is_favorite',EXISTS(SELECT 1 FROM favorites f WHERE f.document_id=d.id AND f.user_id=$1),
 'can_write',d.deleted_at IS NULL AND madi_document_allowed($1,d.id,true))`
